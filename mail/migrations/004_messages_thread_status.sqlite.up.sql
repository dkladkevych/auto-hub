PRAGMA foreign_keys = ON;

ALTER TABLE messages ADD COLUMN thread_id TEXT;
ALTER TABLE messages ADD COLUMN status TEXT NOT NULL DEFAULT 'delivered';

CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
