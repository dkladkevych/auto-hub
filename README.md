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

- **Backend:** Python 3 + Flask
- **Frontend:** Server-rendered Jinja2 templates, vanilla JS
- **Database:** SQLite
- **Styling:** plain CSS (no frameworks)
- **Server:** gunicorn (production)
- **Tests:** pytest

---

## Project Structure

```
auto-hub/
├── web/                        # Flask application
│   ├── app/
│   │   ├── __init__.py         # App factory, blueprints registration
│   │   ├── config.py           # Centralized config from env vars
│   │   ├── constants.py        # App constants (dropdown options)
│   │   ├── context.py          # Template context processors
│   │   ├── db.py               # SQLite init & connection helper
│   │   ├── decorators.py       # @admin_required
│   │   ├── extensions.py       # Flask extensions (Compress, Limiter, CSRF)
│   │   ├── routes/
│   │   │   ├── public.py       # Public pages (home, listing detail, saved, sitemap)
│   │   │   ├── pages.py        # Static pages (terms, privacy, 404)
│   │   │   └── admin.py        # Admin routes (CRUD, login, logout)
│   │   ├── services/
│   │   │   ├── listings.py     # Public listing logic & filters
│   │   │   └── admin.py        # Admin logic (CRUD, stats, validation)
│   │   └── utils/
│   │       ├── images.py       # Image/video upload, sync, delete, preview, thumbnails
│   │       ├── location.py     # Location normalization & aliases
│   │       ├── stats.py        # Visit / view counters (UPSERT)
│   │       └── vin.py          # VIN masking utility (unused)
│   ├── data/
│   │   ├── db/
│   │   │   └── db.sqlite       # SQLite database (ignored by git)
│   │   └── listings/{id}/      # Listing media folders
│   ├── static/
│   │   ├── css/
│   │   │   ├── base.css        # Design system (vars, buttons, forms)
│   │   │   ├── site.css        # Public site layout
│   │   │   ├── admin.css       # Admin panel styles
│   │   │   └── easymde.min.css # Markdown editor styles
│   │   ├── js/
│   │   │   ├── filters.js      # Advanced filters panel toggle
│   │   │   ├── favorites.js    # Save/unsave listings
│   │   │   ├── gallery.js      # Listing gallery & lightbox
│   │   │   ├── admin_dashboard.js  # AJAX pagination
│   │   │   ├── admin_images.js     # Drag & drop image uploader
│   │   │   └── easymde.min.js      # Markdown editor
│   │   └── images/
│   ├── templates/
│   │   ├── public/             # Public templates
│   │   └── admin/              # Admin templates
│   ├── tests/                  # pytest test suite
│   ├── .env                    # Environment variables (ignored by git)
│   ├── requirements.txt
│   └── run.py                  # Dev entry point
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

### Run tests

```bash
pytest
```

---

## Environment Variables

Create `web/.env`:

```env
SECRET_KEY=your_secret_key
ADMIN_PASSWORD_HASH=your_hashed_password
ADMIN_PATH=custom_admin_url_path
```

Generate hash:

```python
from werkzeug.security import generate_password_hash
print(generate_password_hash("your_password"))
```

| Variable | Default | Description |
|----------|---------|-------------|
| `SECRET_KEY` | `fallback_secret` | Flask session encryption |
| `ADMIN_PASSWORD_HASH` | — | Admin panel password hash (preferred) |
| `ADMIN_PASSWORD` | `fallback_password` | Plaintext admin password (fallback) |
| `ADMIN_PATH` | `admin` | Hidden URL prefix for admin panel |

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
