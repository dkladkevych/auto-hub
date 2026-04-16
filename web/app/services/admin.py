from flask import abort

from ..db import get_db
from ..utils.location import build_location_search

from ..constants import (
    RISK_LEVELS,
    CONDITION_OPTIONS,
    SELLER_STATUS_OPTIONS,
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
    """Статы и пагинация админки."""
    offset = (page - 1) * per_page
    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM listings").fetchone()["count"]
    active_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE status = 'active'").fetchone()["count"]
    archived_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE status = 'archived'").fetchone()["count"]
    draft_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE status = 'draft'").fetchone()["count"]

    total_pages = max(1, (total_count + per_page - 1) // per_page)

    listings = db.execute("""
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
        ORDER BY l.created_at DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    db.close()

    return {
        "total_count": total_count,
        "draft_count": draft_count,
        "active_count": active_count,
        "archived_count": archived_count,
        "listings": listings,
        "page": page,
        "total_pages": total_pages,
    }


def get_dashboard_listings_block(page: int, per_page: int = 10):
    """Только блок таблицы для AJAX-пагинации."""
    offset = (page - 1) * per_page
    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM listings").fetchone()["count"]
    total_pages = max(1, (total_count + per_page - 1) // per_page)

    listings = db.execute("""
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
        ORDER BY l.created_at DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    db.close()

    return {
        "listings": listings,
        "page": page,
        "total_pages": total_pages,
    }


def parse_listing_form(form):
    """Собирает form data в один словарь."""
    save_mode = form.get("save_mode", "draft").strip()

    if save_mode == "publish":
        status = "active"
    elif save_mode == "archive":
        status = "archived"
    else:
        status = "draft"

    year = to_int_or_none(form.get("year", "").strip())
    make = form.get("make", "").strip()
    model = form.get("model", "").strip()

    data = {
        "title": _build_title(year, make, model),
        "price": form.get("price", "").strip(),
        "description": form.get("description", "").strip(),
        "risk_level": form.get("risk_level", "").strip(),
        "source_url": form.get("source_url", "").strip(),
        "image_urls_raw": form.get("image_urls", "").strip(),

        "year": year,
        "make": make,
        "model": model,
        "mileage_km": to_int_or_none(form.get("mileage_km", "").strip()),
        "location": form.get("location", "").strip(),

        "condition": form.get("condition", "").strip(),
        "notes": form.get("notes", "").strip(),
        "seller_status": form.get("seller_status", "").strip(),
    }

    data["status"] = status
    data["risk_level"] = normalize_choice(data["risk_level"], RISK_LEVELS)
    data["condition"] = normalize_choice(data["condition"], CONDITION_OPTIONS)
    data["seller_status"] = normalize_choice(data["seller_status"], SELLER_STATUS_OPTIONS)

    data["location_search"] = build_location_search(data["location"])
    data["image_urls"] = [line.strip() for line in data["image_urls_raw"].splitlines() if line.strip()]

    return data


def validate_listing_form(data):
    """Валидация обязательных полей и форматов."""
    if not data["title"] or not data["price"] or not data["description"] or not data["risk_level"]:
        return "Please fill in all required fields."

    try:
        data["price"] = int(data["price"])
    except ValueError:
        return "Price must be a number."

    if data["risk_level"] not in RISK_LEVELS:
        return "Risk level must be LOW, MEDIUM, or HIGH."

    return None


def create_listing(data):
    """Создает новое объявление и связанные изображения."""
    db = get_db()

    cursor = db.execute("""
        INSERT INTO listings (
            title, price, description, risk_level, source_url, status,
            year, make, model, mileage_km, location, location_search,
            condition, notes, seller_status
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, (
        data["title"], data["price"], data["description"], data["risk_level"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["mileage_km"], data["location"], data["location_search"],
        data["condition"], data["notes"], data["seller_status"],
    ))

    listing_id = cursor.lastrowid

    for index, image_url in enumerate(data["image_urls"]):
        db.execute("""
            INSERT INTO listing_images (listing_id, image_url, sort_order)
            VALUES (?, ?, ?)
        """, (listing_id, image_url, index))

    db.commit()
    db.close()


def get_listing_for_edit(listing_id: int):
    """Получает объявление и его изображения для edit-формы."""
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM listings
        WHERE id = ?
    """, (listing_id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    existing_images = db.execute("""
        SELECT *
        FROM listing_images
        WHERE listing_id = ?
        ORDER BY sort_order ASC, id ASC
    """, (listing_id,)).fetchall()

    db.close()
    return car, existing_images


def update_listing(listing_id: int, data):
    """Обновляет объявление и полностью пересоздает image list."""
    db = get_db()

    db.execute("""
        UPDATE listings
        SET
            title = ?,
            price = ?,
            description = ?,
            risk_level = ?,
            source_url = ?,
            status = ?,
            year = ?,
            make = ?,
            model = ?,
            mileage_km = ?,
            location = ?,
            location_search = ?,
            condition = ?,
            notes = ?,
            seller_status = ?
        WHERE id = ?
    """, (
        data["title"], data["price"], data["description"], data["risk_level"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["mileage_km"], data["location"], data["location_search"],
        data["condition"], data["notes"], data["seller_status"],
        listing_id,
    ))

    db.execute("DELETE FROM listing_images WHERE listing_id = ?", (listing_id,))

    for index, image_url in enumerate(data["image_urls"]):
        db.execute("""
            INSERT INTO listing_images (listing_id, image_url, sort_order)
            VALUES (?, ?, ?)
        """, (listing_id, image_url, index))

    db.commit()
    db.close()


def delete_listing_by_id(listing_id: int):
    db = get_db()
    db.execute("DELETE FROM listings WHERE id = ?", (listing_id,))
    db.commit()
    db.close()


def set_listing_status(listing_id: int, status: str):
    db = get_db()
    db.execute("""
        UPDATE listings
        SET status = ?
        WHERE id = ?
    """, (status, listing_id))
    db.commit()
    db.close()
