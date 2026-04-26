package mailprovider

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// IMAPMailProvider is a production-ready MailProvider backed by a real
// IMAP server (Dovecot/Postfix).  Authentication uses a master password
// so that the application does not need to store plaintext mailbox
// passwords.
type IMAPMailProvider struct {
	cfg         *config.Config
	mailboxRepo *repo.MailboxRepo
	smtpSender  *SMTPSender
}

// NewIMAPMailProvider creates an IMAP-backed provider.
func NewIMAPMailProvider(cfg *config.Config, mailboxRepo *repo.MailboxRepo, smtpSender *SMTPSender) *IMAPMailProvider {
	return &IMAPMailProvider{cfg: cfg, mailboxRepo: mailboxRepo, smtpSender: smtpSender}
}

// connect opens a TLS or STARTTLS IMAP connection and logs in.
// When IMAP_MASTER_PASSWORD is set, Dovecot master-user authentication is used
// (mailbox*master / master-password).  Otherwise it falls back to the mailbox
// password hash (only works if the hash is stored as plaintext).
func (p *IMAPMailProvider) connect(ctx context.Context, mailboxEmail string) (*client.Client, error) {
	addr := net.JoinHostPort(p.cfg.IMAPHost, p.cfg.IMAPPort)
	tlsConfig := &tls.Config{InsecureSkipVerify: p.cfg.IMAPSkipTLSVerify}

	var c *client.Client
	var err error
	if p.cfg.IMAPUseSSL {
		c, err = client.DialTLS(addr, tlsConfig)
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("imap dial %s: %w", addr, err)
	}

	if !p.cfg.IMAPUseSSL {
		if err := c.StartTLS(tlsConfig); err != nil {
			_ = c.Logout()
			return nil, fmt.Errorf("imap starttls: %w", err)
		}
	}

	var username, password string
	if p.cfg.IMAPMasterPassword != "" {
		username = mailboxEmail + "*master"
		password = p.cfg.IMAPMasterPassword
	} else {
		username = mailboxEmail
		if p.mailboxRepo != nil {
			if m, _ := p.mailboxRepo.GetByEmail(ctx, mailboxEmail); m != nil {
				password = m.MailboxPasswordHash
			}
		}
	}

	log.Printf("IMAP login user=%s", username)
	if err := c.Login(username, password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	return c, nil
}

func toIMAPFolder(name string) string {
	switch name {
	case "Inbox":
		return "INBOX"
	case "Sent":
		return "Sent"
	case "Drafts":
		return "Drafts"
	case "Trash":
		return "Trash"
	default:
		return name
	}
}

func formatIMAPAddress(a *imap.Address) string {
	if a == nil {
		return ""
	}
	addr := a.MailboxName + "@" + a.HostName
	if a.PersonalName != "" {
		addr = a.PersonalName + " <" + addr + ">"
	}
	return addr
}

func formatIMAPAddresses(addrs []*imap.Address) string {
	var parts []string
	for _, a := range addrs {
		if s := formatIMAPAddress(a); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

func parseRFC822Body(body []byte) (text, html string) {
	mr, err := mail.CreateReader(bytes.NewReader(body))
	if err != nil {
		return string(body), ""
	}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			if ct == "text/plain" {
				b, _ := io.ReadAll(p.Body)
				text = string(b)
			} else if ct == "text/html" {
				b, _ := io.ReadAll(p.Body)
				html = string(b)
			}
		}
	}
	return text, html
}

func hasFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

func (p *IMAPMailProvider) parseIMAPMessage(msg *imap.Message) (*models.Message, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}
	m := &models.Message{
		ID:      strconv.FormatUint(uint64(msg.Uid), 10),
		Seen:    hasFlag(msg.Flags, imap.SeenFlag),
		Flagged: hasFlag(msg.Flags, imap.FlaggedFlag),
	}
	if msg.Envelope != nil {
		env := msg.Envelope
		m.Subject = env.Subject
		m.From = formatIMAPAddresses(env.From)
		m.To = formatIMAPAddresses(env.To)
		m.Cc = formatIMAPAddresses(env.Cc)
		if !env.Date.IsZero() {
			m.Date = env.Date
		}
	}
	if r := msg.GetBody(&imap.BodySectionName{}); r != nil {
		body, _ := io.ReadAll(r)
		m.TextBody, m.HTMLBody = parseRFC822Body(body)
	}
	return m, nil
}

// ListFolders returns Inbox, Sent, Drafts, Trash and Starred with live
// counters obtained via IMAP STATUS and SEARCH.
func (p *IMAPMailProvider) ListFolders(ctx context.Context, mailboxEmail string) ([]models.Folder, error) {
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	folders := []models.Folder{
		{Name: "Inbox"},
		{Name: "Sent"},
		{Name: "Drafts"},
		{Name: "Trash"},
	}
	for i := range folders {
		fb := toIMAPFolder(folders[i].Name)
		status, err := c.Status(fb, []imap.StatusItem{imap.StatusMessages, imap.StatusUnseen})
		if err == nil && status != nil {
			folders[i].Count = int(status.Messages)
			folders[i].Unseen = int(status.Unseen)
		}
	}

	flaggedCount := 0
	if _, err := c.Select("INBOX", true); err == nil {
		criteria := imap.NewSearchCriteria()
		criteria.WithFlags = []string{imap.FlaggedFlag}
		if uids, err := c.UidSearch(criteria); err == nil {
			flaggedCount = len(uids)
		}
	}
	folders = append(folders, models.Folder{Name: "Starred", Count: flaggedCount, Unseen: 0})
	return folders, nil
}

// ListMessages returns a paginated slice of messages from the given folder.
func (p *IMAPMailProvider) ListMessages(ctx context.Context, mailboxEmail string, folder string, limit, offset int) ([]models.Message, error) {
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	var uids []uint32
	if folder == "Starred" {
		if _, err := c.Select("INBOX", true); err != nil {
			return nil, err
		}
		criteria := imap.NewSearchCriteria()
		criteria.WithFlags = []string{imap.FlaggedFlag}
		uids, err = c.UidSearch(criteria)
		if err != nil {
			return nil, err
		}
	} else {
		fb := toIMAPFolder(folder)
		if _, err := c.Select(fb, true); err != nil {
			return nil, err
		}
		criteria := imap.NewSearchCriteria()
		uids, err = c.UidSearch(criteria)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(uids, func(i, j int) bool { return uids[i] > uids[j] })

	if offset >= len(uids) {
		return []models.Message{}, nil
	}
	end := offset + limit
	if end > len(uids) {
		end = len(uids)
	}
	pageUids := uids[offset:end]

	seqset := new(imap.SeqSet)
	seqset.AddNum(pageUids...)

	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, items, messages)
	}()

	var out []models.Message
	for msg := range messages {
		m, err := p.parseIMAPMessage(msg)
		if err != nil {
			continue
		}
		m.MailboxEmail = mailboxEmail
		m.Folder = folder
		out = append(out, *m)
	}
	if err := <-done; err != nil {
		return nil, err
	}
	return out, nil
}

// CountMessages returns the number of messages in a folder.
func (p *IMAPMailProvider) CountMessages(ctx context.Context, mailboxEmail string, folder string) (int, error) {
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return 0, err
	}
	defer c.Logout()

	if folder == "Starred" {
		if _, err := c.Select("INBOX", true); err != nil {
			return 0, err
		}
		criteria := imap.NewSearchCriteria()
		criteria.WithFlags = []string{imap.FlaggedFlag}
		uids, err := c.UidSearch(criteria)
		if err != nil {
			return 0, err
		}
		return len(uids), nil
	}

	fb := toIMAPFolder(folder)
	status, err := c.Status(fb, []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return 0, err
	}
	return int(status.Messages), nil
}

// GetMessage fetches a single message by its UID and parses the full RFC822 body.
func (p *IMAPMailProvider) GetMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) (*models.Message, error) {
	uid, err := strconv.ParseUint(messageID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid message id")
	}
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	fb := toIMAPFolder(folder)
	if _, err := c.Select(fb, true); err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(uid))

	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822}
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, items, messages)
	}()

	var msg *imap.Message
	for m := range messages {
		msg = m
	}
	if err := <-done; err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	res, err := p.parseIMAPMessage(msg)
	if err != nil {
		return nil, err
	}
	res.MailboxEmail = mailboxEmail
	res.Folder = folder
	return res, nil
}

// SendMessage relays the message via SMTP and appends a copy to the Sent folder.
func (p *IMAPMailProvider) SendMessage(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	if p.smtpSender == nil {
		return errors.New("SMTP sender is not configured")
	}

	var fromName string
	if p.mailboxRepo != nil {
		if mb, _ := p.mailboxRepo.GetByEmail(ctx, mailboxEmail); mb != nil {
			fromName = mb.DisplayName
		}
	}

	var toList []string
	for _, field := range []string{msg.To, msg.Cc} {
		for _, raw := range strings.Split(field, ",") {
			if addr := strings.TrimSpace(raw); addr != "" {
				toList = append(toList, addr)
			}
		}
	}

	if err := p.smtpSender.Send(mailboxEmail, fromName, toList, msg.Subject, msg.TextBody, msg.HTMLBody); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	// Append copy to Sent
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	rfc822 := buildSimpleRFC822(mailboxEmail, fromName, msg)
	return c.Append("Sent", []string{imap.SeenFlag}, time.Now(), bytes.NewReader(rfc822))
}

// buildSimpleRFC822 builds a minimal plaintext RFC822 message for APPEND to Sent.
func buildSimpleRFC822(from, fromName string, msg *models.OutgoingMessage) []byte {
	var buf bytes.Buffer
	if fromName != "" {
		fmt.Fprintf(&buf, "From: %s <%s>\r\n", fromName, from)
	} else {
		fmt.Fprintf(&buf, "From: %s\r\n", from)
	}
	fmt.Fprintf(&buf, "To: %s\r\n", msg.To)
	if msg.Cc != "" {
		fmt.Fprintf(&buf, "Cc: %s\r\n", msg.Cc)
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n")
	fmt.Fprintf(&buf, "\r\n%s", msg.TextBody)
	return buf.Bytes()
}

// MarkSeen toggles the \Seen flag via UID STORE.
func (p *IMAPMailProvider) MarkSeen(ctx context.Context, mailboxEmail string, folder string, messageID string, seen bool) error {
	uid, err := strconv.ParseUint(messageID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid message id")
	}
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	fb := toIMAPFolder(folder)
	if _, err := c.Select(fb, false); err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(uid))

	var item imap.StoreItem = imap.AddFlags
	if !seen {
		item = imap.RemoveFlags
	}
	return c.UidStore(seqset, item, []interface{}{imap.SeenFlag}, nil)
}

// SetFlagged toggles the \Flagged flag via UID STORE.
func (p *IMAPMailProvider) SetFlagged(ctx context.Context, mailboxEmail string, folder string, messageID string, flagged bool) error {
	uid, err := strconv.ParseUint(messageID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid message id")
	}
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	fb := toIMAPFolder(folder)
	if _, err := c.Select(fb, false); err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(uid))

	var item imap.StoreItem = imap.AddFlags
	if !flagged {
		item = imap.RemoveFlags
	}
	return c.UidStore(seqset, item, []interface{}{imap.FlaggedFlag}, nil)
}

// DeleteMessage moves the message to Trash.  It first tries UID MOVE (RFC 6851)
// and falls back to COPY + STORE \Deleted + EXPUNGE.
func (p *IMAPMailProvider) DeleteMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) error {
	uid, err := strconv.ParseUint(messageID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid message id")
	}
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	fb := toIMAPFolder(folder)
	if _, err := c.Select(fb, false); err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(uid))

	if err := c.UidMove(seqset, "Trash"); err == nil {
		return nil
	}

	if err := c.UidCopy(seqset, "Trash"); err != nil {
		return err
	}
	if err := c.UidStore(seqset, imap.AddFlags, []interface{}{imap.DeletedFlag}, nil); err != nil {
		return err
	}
	return c.Expunge(nil)
}

// SaveDraft appends a plaintext draft to the Drafts folder.
func (p *IMAPMailProvider) SaveDraft(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	rfc822 := buildSimpleRFC822(mailboxEmail, "", msg)
	return c.Append("Drafts", []string{imap.DraftFlag}, time.Now(), bytes.NewReader(rfc822))
}

// EmptyTrash selects the Trash folder and permanently removes all messages.
func (p *IMAPMailProvider) EmptyTrash(ctx context.Context, mailboxEmail string) error {
	c, err := p.connect(ctx, mailboxEmail)
	if err != nil {
		return err
	}
	defer c.Logout()

	if _, err := c.Select("Trash", false); err != nil {
		return err
	}

	criteria := imap.NewSearchCriteria()
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return err
	}
	if len(uids) == 0 {
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	if err := c.UidStore(seqset, imap.AddFlags, []interface{}{imap.DeletedFlag}, nil); err != nil {
		return err
	}
	return c.Expunge(nil)
}
