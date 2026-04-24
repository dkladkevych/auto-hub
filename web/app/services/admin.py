"""
Admin panel business logic.

- CRUD operations on the inventory table
- Form validation for creating/editing listings
- Dashboard statistics (JOIN with stats table)
- Form parsing and data normalization

Connected to: db (inventory + stats), utils/images (previews), utils/location (normalization).
"""

from flask import abort

from ..db import get_db
from ..utils.images import get_preview_image, get_listing_image_urls

from ..constants import (
    CONDITION_OPTIONS,
    DRIVETRAIN_OPTIONS,
    TRANSMISSION_OPTIONS,
)


def normalize_choice(value, allowed):
    value = (value or "").strip()
    return value if value in allowed else ""


def to_int_or_none(value):
    if not value:
        return None
    try:
        return int(value)
    except ValueError:
        return None


def _build_title(year, make, model):
    parts = [p for p in (year, make, model) if p]
    return " ".join(str(p) for p in parts)


def get_dashboard_data(page: int, per_page: int = 10):
    """Dashboard stats and pagination."""
    offset = (page - 1) * per_page
    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM inventory").fetchone()["count"]
    active_count = db.execute("SELECT COUNT(*) AS count FROM inventory WHERE status = 'active'").fetchone()["count"]
    archived_count = db.execute("SELECT COUNT(*) AS count FROM inventory WHERE status = 'archived'").fetchone()["count"]
    draft_count = db.execute("SELECT COUNT(*) AS count FROM inventory WHERE status = 'draft'").fetchone()["count"]
    demo_count = db.execute("SELECT COUNT(*) AS count FROM inventory WHERE status = 'demo'").fetchone()["count"]

    site_visits_row = db.execute(
        "SELECT view_count FROM stats WHERE target_type = 'site' AND target_id = 0"
    ).fetchone()
    total_site_visits = site_visits_row["view_count"] if site_visits_row else 0

    listing_views_row = db.execute(
        "SELECT COALESCE(SUM(view_count), 0) AS count FROM stats WHERE target_type = 'listing'"
    ).fetchone()
    total_listing_views = listing_views_row["count"] if listing_views_row else 0

    total_pages = max(1, (total_count + per_page - 1) // per_page)

    rows = db.execute("""
        SELECT i.*, COALESCE(s.view_count, 0) AS view_count
        FROM inventory i
        LEFT JOIN stats s ON s.target_type = 'listing' AND s.target_id = i.id
        ORDER BY i.id DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    listings = []
    for row in rows:
        d = dict(row)
        d["preview_image"] = get_preview_image(d["id"], thumb=True)
        listings.append(d)

    db.close()

    return {
        "total_count": total_count,
        "draft_count": draft_count,
        "active_count": active_count,
        "archived_count": archived_count,
        "demo_count": demo_count,
        "total_site_visits": total_site_visits,
        "total_listing_views": total_listing_views,
        "listings": listings,
        "page": page,
        "total_pages": total_pages,
    }


def get_dashboard_listings_block(page: int, per_page: int = 10):
    """Table block only, for AJAX pagination."""
    offset = (page - 1) * per_page
    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM inventory").fetchone()["count"]
    total_pages = max(1, (total_count + per_page - 1) // per_page)

    rows = db.execute("""
        SELECT i.*, COALESCE(s.view_count, 0) AS view_count
        FROM inventory i
        LEFT JOIN stats s ON s.target_type = 'listing' AND s.target_id = i.id
        ORDER BY i.id DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    listings = []
    for row in rows:
        d = dict(row)
        d["preview_image"] = get_preview_image(d["id"], thumb=True)
        listings.append(d)

    db.close()

    return {
        "listings": listings,
        "page": page,
        "total_pages": total_pages,
    }


def parse_listing_form(form):
    """Collects form data into a single dict."""
    save_mode = form.get("save_mode", "draft").strip()

    if save_mode == "publish":
        status = "active"
    elif save_mode == "archive":
        status = "archived"
    elif save_mode == "demo":
        status = "demo"
    else:
        status = "draft"

    year = to_int_or_none(form.get("year", "").strip())
    make = form.get("make", "").strip()
    model = form.get("model", "").strip()

    data = {
        "title": _build_title(year, make, model),
        "price": form.get("price", "").strip(),
        "description": form.get("description", "").strip(),
        "source_url": form.get("source_url", "").strip(),

        "year": year,
        "make": make,
        "model": model,
        "mileage_km": to_int_or_none(form.get("mileage_km", "").strip()),
        "location": form.get("location", "").strip(),

        "condition": form.get("condition", "").strip(),
        "notes": form.get("notes", "").strip(),
        "transmission": form.get("transmission", "").strip(),
        "drivetrain": form.get("drivetrain", "").strip(),
    }

    data["status"] = status
    data["condition"] = normalize_choice(data["condition"], CONDITION_OPTIONS)
    data["transmission"] = normalize_choice(data["transmission"], TRANSMISSION_OPTIONS)
    data["drivetrain"] = normalize_choice(data["drivetrain"], DRIVETRAIN_OPTIONS)

    return data


def validate_listing_form(data):
    """Validates required fields and formats. Returns dict of field errors or None."""
    errors = {}
    is_draft = data.get("status") == "draft"

    if not data.get("title"):
        errors["year"] = "At least one of Year, Make, or Model is required"

    if is_draft:
        # Default empty price to 0 for DB compatibility
        if not data.get("price"):
            data["price"] = 0
        else:
            try:
                data["price"] = int(data["price"])
            except ValueError:
                data["price"] = 0
        return errors if errors else None

    # Publish / demo / archive — full validation
    if not data.get("price"):
        errors["price"] = "Price is required"
    else:
        try:
            data["price"] = int(data["price"])
        except ValueError:
            errors["price"] = "Price must be a number"

    if not data.get("mileage_km"):
        errors["mileage_km"] = "Mileage is required"
    if not data.get("transmission"):
        errors["transmission"] = "Transmission is required"
    if not data.get("drivetrain"):
        errors["drivetrain"] = "Drivetrain is required"
    if not data.get("location"):
        errors["location"] = "Location is required"
    if not data.get("source_url"):
        errors["source_url"] = "Source URL is required"
    if not data.get("description"):
        errors["description"] = "Description is required"
    if not data.get("condition"):
        errors["condition"] = "Condition is required"

    return errors if errors else None


def create_listing(data):
    """Creates a new listing. Returns listing_id."""
    db = get_db()

    cursor = db.execute("""
        INSERT INTO inventory (
            account_id, title, price, description, source_url, status,
            year, make, model, mileage_km, location,
            condition, notes, transmission, drivetrain, published_at
        )
        VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            CASE WHEN ? IN ('active', 'demo') THEN CURRENT_TIMESTAMP END
        )
    """, (
        0,
        data["title"], data["price"], data["description"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["mileage_km"], data["location"],
        data["condition"], data["notes"], data["transmission"], data["drivetrain"], data["status"],
    ))

    listing_id = cursor.lastrowid
    db.commit()
    db.close()
    return listing_id


def get_listing_for_edit(listing_id: int):
    """Gets a listing and its media for the edit form."""
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM inventory
        WHERE id = ?
    """, (listing_id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    db.close()
    existing_images = get_listing_image_urls(listing_id)
    return car, existing_images


def update_listing(listing_id: int, data):
    """Updates a listing (without media)."""
    db = get_db()

    db.execute("""
        UPDATE inventory
        SET
            title = ?,
            price = ?,
            description = ?,
            source_url = ?,
            status = ?,
            year = ?,
            make = ?,
            model = ?,
            mileage_km = ?,
            location = ?,
            condition = ?,
            notes = ?,
            transmission = ?,
            drivetrain = ?,
            published_at = COALESCE(published_at, CASE WHEN ? IN ('active', 'demo') THEN CURRENT_TIMESTAMP END)
        WHERE id = ?
    """, (
        data["title"], data["price"], data["description"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["mileage_km"], data["location"],
        data["condition"], data["notes"], data["transmission"], data["drivetrain"], data["status"],
        listing_id,
    ))

    db.commit()
    db.close()


def delete_listing_by_id(listing_id: int):
    db = get_db()
    db.execute("DELETE FROM inventory WHERE id = ?", (listing_id,))
    db.execute(
        "DELETE FROM view_log WHERE target_type = 'listing' AND target_id = ?",
        (listing_id,),
    )
    db.execute(
        "DELETE FROM stats WHERE target_type = 'listing' AND target_id = ?",
        (listing_id,),
    )
    db.commit()
    db.close()


def set_listing_status(listing_id: int, status: str):
    db = get_db()
    db.execute("""
        UPDATE inventory
        SET status = ?
        WHERE id = ?
    """, (status, listing_id))

    # Set published_at on first publish (active or demo)
    if status in ("active", "demo"):
        db.execute("""
            UPDATE inventory
            SET published_at = COALESCE(published_at, CURRENT_TIMESTAMP)
            WHERE id = ?
        """, (listing_id,))

    db.commit()
    db.close()
