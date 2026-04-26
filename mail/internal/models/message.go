// Package models (message.go) defines the envelope and body structures
// used by the MailProvider abstraction for folder listing and message
// retrieval.
package models

import "time"

// Folder represents an IMAP-style mailbox folder (Inbox, Sent, Drafts,
// Trash, etc.) together with basic message counters.
type Folder struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Unseen int    `json:"unseen"`
}

// Attachment holds metadata for a file attached to an email message.
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	FilePath    string `json:"-"` // local path on disk, not exposed in JSON
}

// Message represents a single email inside a folder.  It is intentionally
// kept close to the IMAP envelope so that swapping the mock provider for
// a real IMAP backend requires minimal mapping code.
type Message struct {
	ID            string       `json:"id"`
	DBID          int64        `json:"-"`
	MailboxEmail  string       `json:"-"` // the mailbox this message is stored in
	Folder        string       `json:"folder"`
	Subject       string       `json:"subject"`
	From          string       `json:"from"`
	To            string       `json:"to"`
	Cc            string       `json:"cc"`
	Date          time.Time    `json:"date"`
	Snippet       string       `json:"snippet"`
	TextBody      string       `json:"text_body"`
	HTMLBody      string       `json:"html_body"`
	Seen          bool         `json:"seen"`
	Flagged       bool         `json:"flagged"`
	Answered      bool         `json:"answered"`
	Draft         bool         `json:"draft"`
	DeletedAt     *time.Time   `json:"deleted_at,omitempty"`
	TrashDaysLeft int          `json:"-"`
	Attachments   []Attachment `json:"attachments"`
	InReplyTo     string       `json:"in_reply_to,omitempty"`
	ThreadID      string       `json:"thread_id,omitempty"`
	Status        string       `json:"status,omitempty"`
}

// OutgoingMessage is the payload accepted by SendMessage and SaveDraft.
// It does not carry a folder or internal ID because those are assigned by
// the provider when the message is persisted.
type OutgoingMessage struct {
	To          string       `json:"to"`
	Cc          string       `json:"cc"`
	Subject     string       `json:"subject"`
	TextBody    string       `json:"text_body"`
	HTMLBody    string       `json:"html_body"`
	Attachments []Attachment `json:"attachments"`
	InReplyTo   string       `json:"in_reply_to,omitempty"`
}
