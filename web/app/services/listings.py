"""
Public listing catalog business logic.

- Filtering and search by price, year, mileage, risk, location
- Building the home page list with pagination
- Detail page data (with gallery)

Connected to: db (inventory), utils/images (previews + thumbnails), utils/location (location normalization).
"""

from flask import abort

from ..db import get_db
from datetime import datetime, timezone

from ..utils.images import get_listing_image_urls, get_listing_media_urls, get_preview_image, listing_has_video
from ..utils.location import build_location_search


def _to_int(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _time_since(dt_str):
    """Human-readable time since published (e.g. '5 Hours', '2 Days'). Uses UTC."""
    if not dt_str:
        return ""
    try:
        dt = datetime.fromisoformat(str(dt_str).replace("Z", "+00:00"))
    except (ValueError, AttributeError):
        return ""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    now = datetime.now(timezone.utc)
    delta = now - dt

    if delta.days >= 365:
        years = delta.days // 365
        return f"{years} Year{'s' if years > 1 else ''}"
    if delta.days >= 30:
        months = delta.days // 30
        return f"{months} Month{'s' if months > 1 else ''}"
    if delta.days > 0:
        return f"{delta.days} Day{'s' if delta.days > 1 else ''}"
    hours = delta.seconds // 3600
    if hours > 0:
        return f"{hours} Hour{'s' if hours > 1 else ''}"
    minutes = delta.seconds // 60
    if minutes > 0:
        return f"{minutes} Minute{'s' if minutes > 1 else ''}"
    return "Just now"


def get_home_listings(filters, page: int = 1, per_page: int = 12):
    """Builds public listings with SQL-level filtering and pagination."""

    q = filters.get("q", "").strip()
    price_min = _to_int(filters.get("price_min", "").strip())
    price_max = _to_int(filters.get("price_max", "").strip())
    year_min = _to_int(filters.get("year_min", "").strip())
    year_max = _to_int(filters.get("year_max", "").strip())
    mileage_min = _to_int(filters.get("mileage_min", "").strip())
    mileage_max = _to_int(filters.get("mileage_max", "").strip())
    transmission = filters.get("transmission", "").strip()
    drivetrain = filters.get("drivetrain", "").strip()
    location = filters.get("location", "").strip()

    has_any_filter = any([
        q,
        location,
        price_min is not None,
        price_max is not None,
        year_min is not None,
        year_max is not None,
        mileage_min is not None,
        mileage_max is not None,
        transmission,
        drivetrain,
    ])

    where_clauses = ["status IN ('active', 'demo')"]
    params = []

    if q:
        where_clauses.append("""
            (
                title LIKE ?
                OR make LIKE ?
                OR model LIKE ?
                OR description LIKE ?
                OR location LIKE ?
            )
        """)
        like = f"%{q}%"
        params += [like, like, like, like, like]

    if price_min is not None:
        where_clauses.append("price >= ?")
        params.append(price_min)

    if price_max is not None:
        where_clauses.append("price <= ?")
        params.append(price_max)

    if year_min is not None:
        where_clauses.append("year >= ?")
        params.append(year_min)

    if year_max is not None:
        where_clauses.append("year <= ?")
        params.append(year_max)

    if mileage_min is not None:
        where_clauses.append("mileage_km >= ?")
        params.append(mileage_min)

    if mileage_max is not None:
        where_clauses.append("mileage_km <= ?")
        params.append(mileage_max)

    if transmission:
        where_clauses.append("transmission = ?")
        params.append(transmission)

    if drivetrain:
        where_clauses.append("drivetrain = ?")
        params.append(drivetrain)

    if location:
        search_tokens = build_location_search(location).split()
        if search_tokens:
            loc_clauses = []
            for token in search_tokens:
                loc_clauses.append("LOWER(location) LIKE ?")
                params.append(f"%{token}%")
            loc_sql = "(" + " OR ".join(loc_clauses) + ")"
            where_clauses.append(loc_sql)

    where_sql = " AND ".join(where_clauses)

    count_query = f"SELECT COUNT(*) AS count FROM inventory WHERE {where_sql}"
    data_query = f"SELECT * FROM inventory WHERE {where_sql} ORDER BY published_at DESC LIMIT ? OFFSET ?"

    db = get_db()
    total = db.execute(count_query, params).fetchone()["count"]

    total_pages = max(1, (total + per_page - 1) // per_page)
    page = max(1, min(page, total_pages))
    offset = (page - 1) * per_page

    rows = db.execute(data_query, params + [per_page, offset]).fetchall()

    listings = []
    for row in rows:
        d = dict(row)
        d["preview_image"] = get_preview_image(d["id"], thumb=True)
        d["has_video"] = listing_has_video(d["id"])
        d["time_since"] = _time_since(d.get("published_at"))
        listings.append(d)

    db.close()

    return listings, has_any_filter, page, total_pages


def get_saved_listings(id_list: list[int]):
    """Returns listings by a list of IDs (for the Saved page)."""
    if not id_list:
        return []

    db = get_db()
    placeholders = ",".join("?" * len(id_list))
    rows = db.execute(
        f"SELECT * FROM inventory WHERE id IN ({placeholders}) AND status IN ('active', 'demo')",
        id_list,
    ).fetchall()
    db.close()

    listings = []
    for row in rows:
        d = dict(row)
        d["preview_image"] = get_preview_image(d["id"], thumb=True)
        d["has_video"] = listing_has_video(d["id"])
        d["time_since"] = _time_since(d.get("published_at"))
        listings.append(d)

    return listings


def get_listing_page_data(listing_id: int):
    """Returns a listing and its media."""
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM inventory
        WHERE id = ?
    """, (listing_id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    if car["status"] == "draft":
        db.close()
        abort(404)

    db.close()

    car = dict(car)
    car["has_video"] = listing_has_video(listing_id)

    image_urls = get_listing_image_urls(listing_id)
    car["has_media"] = bool(image_urls)
    if not image_urls:
        image_urls = ["/static/images/empty.png"]
    images = [{"image_url": url} for url in image_urls]

    thumb_urls = get_listing_media_urls(listing_id, thumb=True)
    if not thumb_urls:
        thumb_urls = ["/static/images/empty.png"]
    thumbs = [{"image_url": url} for url in thumb_urls]

    return car, images, thumbs
