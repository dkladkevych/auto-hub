"""
View logging with bot filtering, fingerprinting, and deduplication.

- Site visits and listing views are counted once per fingerprint per calendar day.
- Global counters in `stats` are updated only on unique views.
- Old fingerprint records are cleaned up automatically.
"""

import hashlib
import random

from flask import current_app, request

from ..db import get_db


# Common bot/crawler user-agent substrings (case-insensitive)
_BOT_PATTERNS = [
    "bot",
    "crawl",
    "spider",
    "slurp",
    "scrape",
    "googlebot",
    "bingbot",
    "yandexbot",
    "baiduspider",
    "facebookexternalhit",
    "whatsapp",
    "discordbot",
    "twitterbot",
    "applebot",
    "linkedinbot",
    "embedly",
    "quora link preview",
    "showyoubot",
    "outbrain",
    "pinterest",
    "slackbot",
    "vkshare",
    "w3c_validator",
    "curl",
    "wget",
    "python-requests",
    "scrapy",
    "headless",
    "phantomjs",
    "selenium",
    "puppeteer",
    "playwright",
]


def _is_bot() -> bool:
    """Return True if the request looks like it comes from a known bot/crawler."""
    ua = (request.headers.get("User-Agent", "") or "").lower()
    return any(pattern in ua for pattern in _BOT_PATTERNS)


def _get_fingerprint() -> str:
    """Create a daily-unique fingerprint from IP, User-Agent and secret key."""
    ip = request.headers.get("X-Forwarded-For", request.remote_addr) or "unknown"
    ua = request.headers.get("User-Agent", "") or ""
    secret = current_app.config.get("SECRET_KEY", "")
    raw = f"{ip}|{ua}|{secret}"
    return hashlib.sha256(raw.encode()).hexdigest()[:32]


def _cleanup_old_views(db):
    """Delete fingerprint records older than 24 hours."""
    db.execute("DELETE FROM view_log WHERE viewed_at < datetime('now', '-1 day')")


def _try_insert_view(db, target_type: str, target_id: int, fingerprint: str) -> bool:
    """Insert a view log record. Return True if the view is unique for today."""
    db.execute(
        """
        INSERT OR IGNORE INTO view_log (target_type, target_id, fingerprint)
        VALUES (?, ?, ?)
        """,
        (target_type, target_id, fingerprint),
    )
    changes = db.execute("SELECT changes()").fetchone()[0]
    return changes > 0


def _increment_stats(db, target_type: str, target_id: int):
    """Increment the global counter in the stats table."""
    db.execute(
        """
        INSERT INTO stats (target_type, target_id, view_count)
        VALUES (?, ?, 1)
        ON CONFLICT(target_type, target_id)
        DO UPDATE SET view_count = view_count + 1
        """,
        (target_type, target_id),
    )


def log_site_visit():
    """Log a unique site visit once per fingerprint per day."""
    if _is_bot():
        return

    db = get_db()
    try:
        if _try_insert_view(db, "site", 0, _get_fingerprint()):
            _increment_stats(db, "site", 0)

        if random.random() < 0.01:
            _cleanup_old_views(db)

        db.commit()
    finally:
        db.close()


def log_listing_view(listing_id: int):
    """Log a unique listing view once per fingerprint per day."""
    if _is_bot():
        return

    db = get_db()
    try:
        if _try_insert_view(db, "listing", listing_id, _get_fingerprint()):
            _increment_stats(db, "listing", listing_id)

        if random.random() < 0.01:
            _cleanup_old_views(db)

        db.commit()
    finally:
        db.close()
