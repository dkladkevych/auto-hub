# Auto-Hub

Auto-Hub is a manual-first system for finding better used cars under $15k.

Instead of browsing through hundreds of low-quality listings, Auto-Hub filters the market and highlights only the options that are actually worth checking.

Each listing includes:
- basic car info (year, make, model, mileage, transmission, drivetrain)
- price & location
- short human notes
- image & video gallery

The goal is simple:
save time, reduce risk, and make the car search process cleaner.

---

## How it works

Auto-Hub is not a marketplace.

It is:
- a filter
- a curated list
- a decision-support tool

Listings are:
1. Found manually (Facebook Marketplace, groups)
2. Reviewed using a checklist
3. Added to the system with notes
4. Published with media (images + videos)

Users receive a clean list of options instead of random links.

---

## Features

- manual listing input (admin panel)
- per-field validation (draft vs publish modes)
- public listings page with search & filters (price, year, mileage, transmission, drivetrain, location)
- listing detail pages with image/video gallery & lightbox
- video thumbnail generation (ffmpeg)
- favorites (saved listings via localStorage)
- admin panel with AJAX pagination
- archive & draft system
- image upload (drag & drop, reorder, delete)
- simple analytics (site visits & listing views)
- sitemap generation
- responsive design

---

## Tech Stack

- **Backend:** Go 1.26 + Gin
- **Frontend:** Server-rendered Go templates, vanilla JS
- **Database:** SQLite (modernc.org/sqlite)
- **Styling:** plain CSS (no frameworks)
- **Tests:** `go test`

---

## Project Structure

```
auto-hub/
├── web/                        # Go web application
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── config/             # Env configuration
│   │   ├── db/                 # SQLite init & schema
│   │   ├── handler/            # HTTP handlers (Gin)
│   │   ├── middleware/         # Auth, CSRF, rate limiting, flash messages
│   │   ├── models/             # Domain structs
│   │   ├── repo/               # Thin SQL repositories
│   │   ├── service/            # Business logic
│   │   └── utils/              # Images, markdown, stats helpers
│   ├── data/
│   │   ├── db/
│   │   │   └── db.sqlite       # SQLite database (ignored by git)
│   │   └── listings/{id}/      # Listing media folders
│   ├── static/
│   │   ├── css/
│   │   ├── js/
│   │   └── images/
│   ├── templates/
│   │   ├── public/             # Public templates
│   │   └── admin/              # Admin templates
│   ├── tests/                  # Go test suite
│   ├── .env                    # Environment variables (ignored by git)
│   └── go.mod
├── mail/                       # Mail control panel (separate Go service)
└── README.md
```

---

## Development

### Prerequisites

- Go 1.26 or newer

### Run (development)

```bash
cd web
go run cmd/server/main.go
```

The server starts on `:8000` by default.

### Build

```bash
cd web
go build -o main cmd/server/main.go
```

### Run tests

```bash
cd web
go test ./tests
```

---

## Environment Variables

Create `web/.env`:

```env
SECRET_KEY=your_secret_key
ADMIN_PASSWORD_HASH=your_hashed_password
ADMIN_PATH=custom_admin_url_path
```

Generate bcrypt hash (if you want to use `ADMIN_PASSWORD_HASH`):

```bash
go run golang.org/x/crypto/bcrypt/cmd/bcrypt@latest
```

| Variable | Default | Description |
|----------|---------|-------------|
| `SECRET_KEY` | `fallback_secret` | Session cookie encryption |
| `ADMIN_PASSWORD_HASH` | — | Admin panel bcrypt hash (preferred) |
| `ADMIN_PASSWORD` | `fallback_password` | Plaintext admin password (fallback) |
| `ADMIN_PATH` | `admin` | Hidden URL prefix for admin panel |
| `DB_PATH` | `data/db/db.sqlite` | SQLite database file path |
| `DATA_DIR` | `data` | Data directory root |
| `LISTINGS_DIR` | `data/listings` | Listing media storage |
| `MAIL_SERVICE_URL` | — | URL of the mail service for sending emails |
| `INTERNAL_API_TOKEN` | — | Token for internal API calls |

---

## Database Schema

### `inventory`
Main table for listings. Includes `account_id` for future multi-account support.

Key columns:
- `status` — `draft` | `active` | `demo` | `archived`
- `published_at` — UTC datetime set on first publish
- `transmission`, `drivetrain` — vehicle specs
- `condition` — `Good` | `Fair` | `Poor`

### `stats`
Counters table. No time-series, just aggregated counts.
- `('site', 0)` — total site visits
- `('listing', N)` — views for listing N

---

## Philosophy

- manual > automation
- trust > scale
- notes = value
- simple > complex

Auto-Hub is not built by code.

It is built by:
- filtering
- judgment
- conversations

---

## Status

Early stage (v0.1)

Currently focused on:
- validating demand
- improving workflow
- testing real interactions with buyers

---

## Future (optional)

- garage network
- inspection coordination
- better filtering tools
- automation (only after validation)

---

## Disclaimer

Auto-Hub does not guarantee vehicle quality.

Notes and listings are subjective and based on visible information.

Always perform your own checks and consider a professional inspection.

---

## Contact

If you're looking for a car:
send a message and get a few filtered options.
