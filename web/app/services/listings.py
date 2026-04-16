from flask import abort

from ..db import get_db
from ..utils.location import build_location_search
from ..utils.vin import mask_vin


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
    transmission = filters.get("transmission", "").strip()
    drivetrain = filters.get("drivetrain", "").strip()
    risk_level = filters.get("risk_level", "").strip()
    location = filters.get("location", "").strip()
    include_unknown = filters.get("include_unknown") == "on"
    fuel_type = filters.get("fuel_type", "").strip()
    body_style = filters.get("body_style", "").strip()
    condition = filters.get("condition", "").strip()

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
        risk_level,
        fuel_type,
        body_style,
        condition,
    ])

    query = """
        SELECT
            l.*,
            (
                SELECT li.image_url
                FROM listing_images li
                WHERE li.listing_id = l.id
                ORDER BY li.sort_order ASC, li.id ASC
                LIMIT 1
            ) AS preview_image
        FROM listings l
        WHERE l.status = 'active'
    """

    params = []

    if q:
        query += """
            AND (
                l.title LIKE ?
                OR l.make LIKE ?
                OR l.model LIKE ?
                OR l.trim LIKE ?
                OR l.description LIKE ?
                OR l.location LIKE ?
            )
        """
        like = f"%{q}%"
        params += [like, like, like, like, like, like]

    if location:
        normalized_location = build_location_search(location)

        if include_unknown:
            query += """
                AND (
                    l.location_search LIKE ?
                    OR l.location LIKE ?
                    OR l.location IS NULL
                    OR l.location = ''
                    OR l.location_search IS NULL
                    OR l.location_search = ''
                )
            """
        else:
            query += """
                AND (
                    l.location_search LIKE ?
                    OR l.location LIKE ?
                )
            """

        params.append(f"%{normalized_location}%")
        params.append(f"%{location}%")

    if price_min is not None:
        query += " AND l.price >= ?"
        params.append(price_min)

    if price_max is not None:
        query += " AND l.price <= ?"
        params.append(price_max)

    if year_min is not None:
        query += " AND l.year >= ?"
        params.append(year_min)

    if year_max is not None:
        query += " AND l.year <= ?"
        params.append(year_max)

    if mileage_min is not None:
        query += " AND l.mileage_km >= ?"
        params.append(mileage_min)

    if mileage_max is not None:
        query += " AND l.mileage_km <= ?"
        params.append(mileage_max)

    if transmission:
        if include_unknown:
            query += " AND (l.transmission = ? OR l.transmission IS NULL OR l.transmission = '')"
        else:
            query += " AND l.transmission = ?"
        params.append(transmission)

    if drivetrain:
        if include_unknown:
            query += " AND (l.drivetrain = ? OR l.drivetrain IS NULL OR l.drivetrain = '')"
        else:
            query += " AND l.drivetrain = ?"
        params.append(drivetrain)

    if risk_level:
        query += " AND l.risk_level = ?"
        params.append(risk_level)

    if fuel_type:
        if include_unknown:
            query += " AND (l.fuel_type = ? OR l.fuel_type IS NULL OR l.fuel_type = '')"
        else:
            query += " AND l.fuel_type = ?"
        params.append(fuel_type)

    if body_style:
        if include_unknown:
            query += " AND (l.body_style = ? OR l.body_style IS NULL OR l.body_style = '')"
        else:
            query += " AND l.body_style = ?"
        params.append(body_style)

    if condition:
        if include_unknown:
            query += " AND (l.condition = ? OR l.condition IS NULL OR l.condition = '')"
        else:
            query += " AND l.condition = ?"
        params.append(condition)

    query += " ORDER BY l.created_at DESC"

    db = get_db()
    listings = db.execute(query, params).fetchall()
    db.close()

    return listings, has_any_filter


def get_listing_page_data(listing_id: int):
    """Возвращает объявление, его картинки и замаскированный VIN."""
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
        
    images = db.execute("""
        SELECT *
        FROM listing_images
        WHERE listing_id = ?
        ORDER BY sort_order ASC, id ASC
    """, (listing_id,)).fetchall()

    db.close()

    return car, images, mask_vin(car["vin"])