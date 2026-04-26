PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mailbox_email TEXT NOT NULL REFERENCES mailboxes(email) ON DELETE CASCADE,
    folder TEXT NOT NULL DEFAULT 'Inbox',
    message_uid TEXT NOT NULL UNIQUE,
    in_reply_to TEXT,
    subject TEXT NOT NULL,
    sender_name TEXT,
    sender_email TEXT NOT NULL,
    recipient TEXT NOT NULL,
    date TEXT NOT NULL,
    text_body TEXT,
    html_body TEXT,
    seen INTEGER NOT NULL DEFAULT 0,
    flagged INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_mailbox_folder ON messages(mailbox_email, folder);
CREATE INDEX IF NOT EXISTS idx_messages_uid ON messages(message_uid);
CREATE INDEX IF NOT EXISTS idx_messages_flagged ON messages(mailbox_email, flagged) WHERE flagged = 1;

CREATE TABLE IF NOT EXISTS message_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    file_size INTEGER NOT NULL DEFAULT 0,
    file_path TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
