"""
View logging via UPSERT into the stats table.

- site visits: counter for home page visits
- listing views: counter for individual listing views

Uses SQLite ON CONFLICT for atomic counter updates.
"""

from ..db import get_db


def log_site_visit():
    """Increments the site visit counter."""
    db = get_db()
    db.execute(
        """
        INSERT INTO stats (target_type, target_id, view_count)
        VALUES ('site', 0, 1)
        ON CONFLICT(target_type, target_id)
        DO UPDATE SET view_count = view_count + 1
        """
    )
    db.commit()
    db.close()


def log_listing_view(listing_id: int):
    """Increments the view counter for a specific listing."""
    db = get_db()
    db.execute(
        """
        INSERT INTO stats (target_type, target_id, view_count)
        VALUES ('listing', ?, 1)
        ON CONFLICT(target_type, target_id)
        DO UPDATE SET view_count = view_count + 1
        """,
        (listing_id,),
    )
    db.commit()
    db.close()
