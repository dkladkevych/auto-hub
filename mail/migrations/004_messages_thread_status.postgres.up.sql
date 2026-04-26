ALTER TABLE messages ADD COLUMN IF NOT EXISTS thread_id TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'delivered';

CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
