# Auto-Hub

Auto-Hub is a manual-first system for finding better used cars under $15k.

Instead of browsing through hundreds of low-quality or risky listings, Auto-Hub filters the market and highlights only the options that are actually worth checking.

Each listing includes:
- basic car info
- risk level
- short human notes

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
3. Assigned a risk score
4. Added to the system with notes

Users receive a clean list of options instead of random links.

---

## Features (v0.1)

- manual listing input
- risk scoring (LOW / MEDIUM / HIGH)
- short notes per listing
- public listings page with search & filters
- listing detail pages with image gallery & lightbox
- admin panel with AJAX pagination
- archive & draft system
- image upload (drag & drop, reorder, delete)
- simple analytics (site visits & listing views)

---

## Tech Stack

- **Backend:** Python 3 + Flask
- **Frontend:** Server-rendered Jinja2 templates, vanilla JS
- **Database:** SQLite
- **Styling:** plain CSS (no frameworks)
- **Server:** gunicorn (production)

---

## Project Structure

```
auto-hub/
├── docs/                       # Business docs (MVP flow, scripts, validation)
│   ├── deal_validation.md
│   ├── mvp_flow.md
│   ├── notes.md
│   └── scripts.md
└── web/                        # Flask application
    ├── app/
    │   ├── __init__.py         # App factory, blueprints registration
    │   ├── config.py           # Centralized config from env vars
    │   ├── context.py          # Template context processors
    │   ├── db.py               # SQLite init & connection helper
    │   ├── decorators.py       # @admin_required
    │   ├── constants.py        # App constants (risk levels, etc.)
    │   ├── routes/
    │   │   ├── public.py       # Public pages (home, listing detail)
    │   │   ├── pages.py        # Static pages (terms, privacy, 404)
    │   │   └── admin.py        # Admin routes (CRUD, login, logout)
    │   ├── services/
    │   │   ├── listings.py     # Public listing logic & filters
    │   │   └── admin.py        # Admin logic (CRUD, stats, validation)
    │   └── utils/
    │       ├── images.py       # Image upload, sync, delete, preview
    │       ├── location.py     # Location normalization & aliases
    │       ├── stats.py        # Visit / view counters (UPSERT)
    │       └── vin.py          # VIN masking utility
    ├── data/
    │   ├── db/
    │   │   └── db.sqlite       # SQLite database
    │   └── listings/{id}/      # Listing image folders
    ├── static/
    │   ├── css/
    │   │   ├── base.css        # Design system (vars, buttons, forms)
    │   │   ├── site.css        # Public site layout
    │   │   └── admin.css       # Admin panel styles
    │   ├── js/
    │   │   ├── filters.js      # Advanced filters panel toggle
    │   │   ├── gallery.js      # Listing gallery & lightbox
    │   │   ├── admin_dashboard.js  # AJAX pagination
    │   │   └── admin_images.js     # Drag & drop image uploader
    │   └── images/
    ├── templates/
    │   ├── public/             # Public templates
    │   └── admin/              # Admin templates
    ├── .env                    # Environment variables
    ├── requirements.txt
    └── run.py                  # Dev entry point
```

---

## Development

### Setup

```bash
cd web
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### Run (development)

```bash
python run.py
```

### Run (production)

```bash
gunicorn -w 2 -b 0.0.0.0:8000 run:app
```

---

## Environment Variables

Create `web/.env`:

```env
SECRET_KEY=your_secret_key
ADMIN_PASSWORD=your_admin_password
ADMIN_PATH=custom_admin_url_path
```

| Variable | Default | Description |
|----------|---------|-------------|
| `SECRET_KEY` | `fallback_secret` | Flask session encryption |
| `ADMIN_PASSWORD` | `fallback_password` | Admin panel password |
| `ADMIN_PATH` | `admin` | Hidden URL prefix for admin panel |

---

## Database Schema

### `inventory`
Main table for listings. Includes `account_id` for future multi-account support.

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

Risk levels and notes are subjective and based on visible information.

Always perform your own checks and consider a professional inspection.

---

## Contact

If you're looking for a car:
send a message and get a few filtered options.
