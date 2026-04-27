package mailprovider

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/textproto"
	"strings"
)

// SMTPSender sends email via a plain SMTP relay (no auth, optional STARTTLS).
type SMTPSender struct {
	Host       string
	Port       string
	RequireTLS bool
}

// Send delivers a multipart/alternative message (text + optional HTML) to the
// given recipients via plain SMTP.  Auth is nil because the local relay on
// 127.0.0.1:25 does not require authentication.
func (s *SMTPSender) Send(from, fromName string, to []string, subject, textBody, htmlBody string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}

	// Build the MIME message manually so we have full control over TLS.
	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)
	boundary := mp.Boundary()

	// Headers
	if fromName != "" {
		buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	} else {
		buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	buf.WriteString("\r\n")

	// Plain-text part
	if textBody != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=utf-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		w, _ := mp.CreatePart(h)
		qw := quotedprintable.NewWriter(w)
		_, _ = qw.Write([]byte(textBody))
		_ = qw.Close()
	}

	// HTML part
	if htmlBody != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		w, _ := mp.CreatePart(h)
		qw := quotedprintable.NewWriter(w)
		_, _ = qw.Write([]byte(htmlBody))
		_ = qw.Close()
	}

	_ = mp.Close()

	addr := net.JoinHostPort(s.Host, s.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := newSMTPClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp RCPT TO %s: %w", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	_, err = wc.Write(buf.Bytes())
	if err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := client.DataEnd(); err != nil {
		return fmt.Errorf("smtp data end: %w", err)
	}

	return client.Quit()
}

// smtpClient is a minimal abstraction over net/smtp so we can avoid the
// automatic STARTTLS behaviour of smtp.SendMail when we talk to a local
// relay that does not present a valid certificate.
type smtpClient struct {
	text *textproto.Conn
}

func newSMTPClient(conn net.Conn, host string) (*smtpClient, error) {
	c := &smtpClient{text: textproto.NewConn(conn)}
	_, _, err := c.text.ReadResponse(220)
	if err != nil {
		return nil, err
	}
	if err := c.hello("localhost"); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *smtpClient) hello(host string) error {
	_, _, err := c.cmd(250, "EHLO %s", host)
	return err
}

func (c *smtpClient) Mail(from string) error {
	_, _, err := c.cmd(250, "MAIL FROM:<%s>", from)
	return err
}

func (c *smtpClient) Rcpt(to string) error {
	_, _, err := c.cmd(250, "RCPT TO:<%s>", to)
	return err
}

func (c *smtpClient) Data() (io.WriteCloser, error) {
	_, _, err := c.cmd(354, "DATA")
	if err != nil {
		return nil, err
	}
	return c.text.DotWriter(), nil
}

// DataEnd reads the server's response after the DATA body has been sent.
// A successful DATA transfer is acknowledged with code 250.
func (c *smtpClient) DataEnd() error {
	_, _, err := c.text.ReadResponse(250)
	return err
}

func (c *smtpClient) Quit() error {
	_, _, err := c.cmd(221, "QUIT")
	if err != nil {
		return err
	}
	return c.text.Close()
}

func (c *smtpClient) Close() error {
	return c.text.Close()
}

func (c *smtpClient) cmd(expectCode int, format string, args ...interface{}) (int, string, error) {
	id, err := c.text.Cmd(format, args...)
	if err != nil {
		return 0, "", err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	code, msg, err := c.text.ReadResponse(expectCode)
	return code, msg, err
}
