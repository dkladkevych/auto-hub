-- Seed default domain
INSERT OR IGNORE INTO domains (domain, is_default, is_active)
VALUES ('auto-hub.ca', 1, 1);

-- Seed admin user
-- Password: admin123
INSERT OR IGNORE INTO users (email, password_hash, full_name, role, is_active)
VALUES (
    'admin@auto-hub.ca',
    '$2a$10$uPdEUqQH66NMwLiSL8WjOO7s9ALxloiad857FrNoeRYFsyH7rMFzu',
    'System Administrator',
    'admin',
    1
);

-- Seed admin personal mailbox
INSERT OR IGNORE INTO mailboxes (email, local_part, domain, display_name, mailbox_type, is_active, can_receive, can_send, quota_mb, maildir_path, imap_login_enabled, smtp_login_enabled, mailbox_password_hash)
VALUES (
    'admin@auto-hub.ca',
    'admin',
    'auto-hub.ca',
    'System Administrator',
    'personal',
    1,
    1,
    1,
    1024,
    '/var/mail/vhosts/auto-hub.ca/admin/',
    1,
    1,
    ''
);

-- Link admin user to personal mailbox
INSERT OR IGNORE INTO mailbox_members (user_id, mailbox_id, access_role)
VALUES (
    (SELECT id FROM users WHERE email = 'admin@auto-hub.ca'),
    (SELECT id FROM mailboxes WHERE email = 'admin@auto-hub.ca'),
    'manager'
);
