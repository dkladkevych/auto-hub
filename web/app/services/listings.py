from flask import abort

from ..db import get_db
from ..utils.images import get_listing_image_urls, get_preview_image
from ..utils.location import build_location_search


def _to_int(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def get_home_listings(filters):
    """Формирует список публичных объявлений с фильтрами."""

    q = filters.get("q", "").strip()
    price_min = _to_int(filters.get("price_min", "").strip())
    price_max = _to_int(filters.get("price_max", "").strip())
    year_min = _to_int(filters.get("year_min", "").strip())
    year_max = _to_int(filters.get("year_max", "").strip())
    mileage_min = _to_int(filters.get("mileage_min", "").strip())
    mileage_max = _to_int(filters.get("mileage_max", "").strip())
    risk_level = filters.get("risk_level", "").strip()
    location = filters.get("location", "").strip()
    include_unknown = filters.get("include_unknown") == "on"

    has_any_filter = any([
        q,
        location,
        price_min is not None,
        price_max is not None,
        year_min is not None,
        year_max is not None,
        mileage_min is not None,
        mileage_max is not None,
        risk_level,
    ])

    query = """
        SELECT *
        FROM listings
        WHERE status = 'active'
    """

    params = []

    if q:
        query += """
            AND (
                title LIKE ?
                OR make LIKE ?
                OR model LIKE ?
                OR description LIKE ?
                OR location LIKE ?
            )
        """
        like = f"%{q}%"
        params += [like, like, like, like, like]

    if location:
        normalized_location = build_location_search(location)

        if include_unknown:
            query += """
                AND (
                    location_search LIKE ?
                    OR location LIKE ?
                    OR location IS NULL
                    OR location = ''
                    OR location_search IS NULL
                    OR location_search = ''
                )
            """
        else:
            query += """
                AND (
                    location_search LIKE ?
                    OR location LIKE ?
                )
            """

        params.append(f"%{normalized_location}%")
        params.append(f"%{location}%")

    if price_min is not None:
        query += " AND price >= ?"
        params.append(price_min)

    if price_max is not None:
        query += " AND price <= ?"
        params.append(price_max)

    if year_min is not None:
        query += " AND year >= ?"
        params.append(year_min)

    if year_max is not None:
        query += " AND year <= ?"
        params.append(year_max)

    if mileage_min is not None:
        query += " AND mileage_km >= ?"
        params.append(mileage_min)

    if mileage_max is not None:
        query += " AND mileage_km <= ?"
        params.append(mileage_max)

    if risk_level:
        query += " AND risk_level = ?"
        params.append(risk_level)

    query += " ORDER BY created_at DESC"

    db = get_db()
    rows = db.execute(query, params).fetchall()
    db.close()

    listings = []
    for row in rows:
        d = dict(row)
        d["preview_image"] = get_preview_image(d["id"])
        listings.append(d)

    return listings, has_any_filter


def get_listing_page_data(listing_id: int):
    """Возвращает объявление и его картинки."""
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM listings
        WHERE id = ?
    """, (listing_id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    if car["status"] == "draft":
        db.close()
        abort(404)

    db.close()

    image_urls = get_listing_image_urls(listing_id)
    images = [{"image_url": url} for url in image_urls]

    return car, images
