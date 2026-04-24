-- SQLite migration: add created_by_user_id columns
ALTER TABLE users ADD COLUMN created_by_user_id INTEGER;
ALTER TABLE mailboxes ADD COLUMN created_by_user_id INTEGER;
