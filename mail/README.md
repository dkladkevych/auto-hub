# Auto-Hub Mail Control Panel

A lightweight web control panel for managing mail users, domains, shared mailboxes and webmail preview. Built with Go 1.22+, Gin and SQLite (serverless, zero-config).

## Features

- **Operator / Admin / User** role-based access control
- **Domain management** with a configurable default domain
- **User & mailbox management** — creating a user always provisions a matching personal mailbox
- **Shared & system mailboxes** with member access control (manager / user / read-only)
- **Webmail UI** with folder browsing, message reading, compose and trash lifecycle
- **Audit logging** for every mutating action
- **Session-based authentication** with SHA-256 hashed session tokens stored in SQLite
- **SMTP outbound** for external recipients via local relay (Postfix on 127.0.0.1:25)
- **In-memory rate limiting** on login, operator login and compose endpoints
- **Mock mail provider** for instant UI testing (swap for IMAP/SMTP later without touching handlers)

## Tech Stack

| Layer        | Technology                        |
|--------------|-----------------------------------|
| Language     | Go 1.22+                          |
| Router / UI  | Gin + server-side HTML templates  |
| Database     | SQLite (modernc.org/sqlite)       |
| Migrations   | Plain SQL files, auto-run on boot |
| Passwords    | bcrypt (golang.org/x/crypto)      |
| CSS          | Vanilla CSS, no build step        |

## Project Structure

```
mail/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── config/                 # Env & .env configuration
│   ├── db/                     # SQLite connection, schema & migration helpers
│   ├── handler/                # HTTP handlers (Gin)
│   ├── mailprovider/           # MailProvider interface + in-memory mock
│   ├── middleware/             # Auth & RBAC middleware
│   ├── models/                 # Domain structs
│   ├── repo/                   # Thin SQL repositories
│   ├── service/                # Business logic
│   └── utils/                  # bcrypt & operator HMAC helpers
├── migrations/
│   ├── sqlite_schema.sql       # Base schema (tables, indexes, FKs)
│   └── 002_add_created_by.up.sql
├── seeds/
│   └── seed.sql                # Demo data (default domain + admin user)
├── templates/                  # Go html/template views
├── static/css/                 # Single stylesheet
├── .env.example                # Example environment file
└── README.md
```

## Quick Start

### 1. Prerequisites

- Go 1.22 or newer
- (Optional) `godotenv` is vendored — no extra tools needed

### 2. Configure

```bash
cp .env.example .env
```

Edit `.env` to taste. At minimum you should change:

```dotenv
SESSION_SECRET=replace-with-a-long-random-string
OPERATOR_PASSWORD_HASH=$2a$10$...   # bcrypt hash of your operator password
```

> You can generate a bcrypt hash quickly with:
> ```bash
> go run golang.org/x/crypto/bcrypt/cmd/bcrypt@latest
> ```

### 3. Run (with seed for first boot)

```bash
cd mail
go run cmd/server/main.go
```

On a fresh machine you will want the seed to create the default domain and admin account:

```bash
RUN_SEED=true go run cmd/server/main.go
```

Default seeded credentials:

| Account | Email | Password |
|---------|-------|----------|
| Admin | `admin@auto-hub.ca` | `admin123` |

### 4. Open

http://localhost:8080

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_DRIVER` | `sqlite` | Database driver: `sqlite` or `postgres` |
| `DATABASE_PATH` | `data/mail.db` | SQLite file path |
| `DATABASE_URL` | *(empty)* | PostgreSQL connection string |
| `SERVER_PORT` | `8080` | HTTP port |
| `SESSION_SECRET` | `change-me-in-production` | Secret for cookie session tokens |
| `SESSION_MAX_AGE_MINUTES` | `1440` | Session lifetime (24 h) |
| `RUN_MIGRATIONS` | `true` | Auto-run SQL migrations on start |
| `RUN_SEED` | `false` | Run seed file on start |
| `OPERATOR_PASSWORD` | *(empty)* | Plain-text operator password (dev only) |
| `OPERATOR_PASSWORD_HASH` | *(empty)* | bcrypt hash of operator password (production) |
| `OPERATOR_SESSION_SECRET` | `operator-change-me` | HMAC secret for operator cookies |
| `OPERATOR_LOGIN_PATH` | `/operator/login` | URL path for operator login (hide from public) |
| `OPERATOR_LOGOUT_PATH` | `{LOGIN_PATH}/logout` | URL path for operator logout |
| `SMTP_ENABLED` | `false` | Enable external SMTP delivery via local relay |
| `SMTP_HOST` | `127.0.0.1` | SMTP relay host |
| `SMTP_PORT` | `25` | SMTP relay port |
| `SMTP_REQUIRE_TLS` | `false` | Require TLS for SMTP |
| `IMAP_HOST` | `127.0.0.1` | IMAP server host |
| `IMAP_PORT` | `143` | IMAP port (143 or 993) |
| `IMAP_USE_SSL` | `false` | `true` = immediate TLS, `false` = STARTTLS |
| `IMAP_SKIP_TLS_VERIFY` | `true` | Skip TLS cert verification (localhost) |
| `IMAP_MASTER_PASSWORD` | *(empty)* | Master password for IMAP login |
| `RATE_LIMIT_ENABLED` | `true` | Enable in-memory rate limiting |
| `LOGIN_RATE_LIMIT_MAX` | `5` | Max login attempts per IP |
| `LOGIN_RATE_LIMIT_WINDOW_MINUTES` | `10` | Login rate-limit window |
| `OPERATOR_RATE_LIMIT_MAX` | `3` | Max operator login attempts per IP |
| `OPERATOR_RATE_LIMIT_WINDOW_MINUTES` | `15` | Operator login rate-limit window |
| `SEND_RATE_LIMIT_MAX` | `20` | Max outgoing sends per user per window |
| `SEND_RATE_LIMIT_WINDOW_MINUTES` | `10` | Send rate-limit window |
| `DRAFT_RATE_LIMIT_MAX` | `60` | Max draft saves per user per window |
| `DRAFT_RATE_LIMIT_WINDOW_MINUTES` | `10` | Draft rate-limit window |

## Deployment Notes

### Build

```bash
cd mail
go build -o server cmd/server/main.go
```

### Run as a service (systemd example)

```ini
[Unit]
Description=Auto-Hub Mail Panel
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/mail
ExecStart=/opt/mail/server
Restart=on-failure
Environment="SERVER_PORT=8080"
Environment="SESSION_SECRET=super-secret-change-me"
Environment="OPERATOR_PASSWORD_HASH=$2a$10$..."

[Install]
WantedBy=multi-user.target
```

### Reverse proxy (Nginx)

```nginx
server {
    listen 80;
    server_name mail.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

> For production always place the app behind HTTPS so that session cookies are transmitted securely.

### Security Checklist

- [ ] Change `SESSION_SECRET` and `OPERATOR_SESSION_SECRET`
- [ ] Set `OPERATOR_PASSWORD_HASH` instead of plain `OPERATOR_PASSWORD`
- [ ] Disable `RUN_SEED` in production (`RUN_SEED=false`)
- [ ] Run behind HTTPS
- [ ] Restrict file permissions on the SQLite database (`chmod 600 mail.db`)
- [ ] Back up `mail.db` regularly
- [ ] Change `OPERATOR_LOGIN_PATH` to something non-obvious
- [ ] Review rate-limit settings for your traffic volume

## Switching to a Real Mail Backend

The `mailprovider.MailProvider` interface abstracts all mail storage / transport. The current default implementation (`DevDBMailProvider`) stores messages in SQLite and can already relay external mail via a local SMTP server (e.g. Postfix on `127.0.0.1:25`).

### IMAP Provider

Set `MAIL_PROVIDER=imap` and configure the IMAP variables in `.env`. The IMAP provider authenticates with a **master password** so that the app never needs to store plaintext mailbox passwords.

#### Dovecot master user setup (one-time)

```conf
# /etc/dovecot/conf.d/10-auth.conf
auth_master_user_separator = *

# /etc/dovecot/conf.d/auth-master.conf.ext
passdb {
  driver = passwd-file
  args = /etc/dovecot/master-users
}
```

Create `/etc/dovecot/master-users`:
```
master:{PLAIN}your-imap-master-password
```

Then any mailbox can be opened with login `user@auto-hub.ca*master` and the master password. The app automatically appends `*master` internally when `IMAP_MASTER_PASSWORD` is set.

> **Note:** `IMAP_SKIP_TLS_VERIFY=true` is useful for local Dovecot with a self-signed certificate.

### Custom IMAP/SMTP stack

To connect to a remote IMAP/SMTP stack instead of the built-in providers:

1. Implement `MailProvider` with your own IMAP/SMTP clients.
2. Swap the provider in `cmd/server/main.go`:
   ```go
   realProvider := mailprovider.NewIMAPMailProvider(cfg, mailboxRepo, smtpSender)
   webmailService := service.NewWebmailService(realProvider, mailboxRepo, memberRepo)
   ```
3. Handlers, templates and services remain unchanged.

## Roadmap / TODO

- [ ] Background goroutine for expired session cleanup
- [x] SMTP outbound for external recipients (dev_db provider)
- [ ] IMAP provider implementation
- [ ] Mailbox quota usage display (Dovecot dict or IMAP QUOTA)
- [ ] 2FA for operator login

## License

Internal use only — Auto-Hub team.
