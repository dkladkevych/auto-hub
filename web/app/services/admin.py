from flask import abort

from ..db import get_db
from ..utils.location import build_location_search

from ..constants import (
    RISK_LEVELS,
    TRANSMISSION_OPTIONS,
    DRIVETRAIN_OPTIONS,
    FUEL_TYPE_OPTIONS,
    BODY_STYLE_OPTIONS,
    TURBO_OPTIONS,
    SELLER_TYPE_OPTIONS,
    SELLER_STATUS_OPTIONS,
    TITLE_STATUS_OPTIONS,
    LIEN_STATUS_OPTIONS,
    REBUILT_STATUS_OPTIONS,
    ACCIDENT_HISTORY_OPTIONS,
    CONDITION_OPTIONS,
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
    """Собирает form data в один словарь.

    Это уменьшает дубли между create/edit.
    """
    save_mode = form.get("save_mode", "draft").strip()

    if save_mode == "publish":
        status = "active"
    elif save_mode == "archive":
        status = "archived"
    else:
        status = "draft"

    data = {
        "title": form.get("title", "").strip(),
        "price": form.get("price", "").strip(),
        "description": form.get("description", "").strip(),
        "risk_level": form.get("risk_level", "").strip(),
        "source_url": form.get("source_url", "").strip(),
        "image_urls_raw": form.get("image_urls", "").strip(),

        "year": form.get("year", "").strip(),
        "make": form.get("make", "").strip(),
        "model": form.get("model", "").strip(),
        "trim": form.get("trim", "").strip(),
        "mileage_km": form.get("mileage_km", "").strip(),
        "transmission": form.get("transmission", "").strip(),
        "drivetrain": form.get("drivetrain", "").strip(),
        "fuel_type": form.get("fuel_type", "").strip(),
        "engine": form.get("engine", "").strip(),
        "body_style": form.get("body_style", "").strip(),
        "exterior_color": form.get("exterior_color", "").strip(),
        "interior_color": form.get("interior_color", "").strip(),
        "doors": form.get("doors", "").strip(),
        "seats": form.get("seats", "").strip(),
        "vin": form.get("vin", "").strip(),
        "location": form.get("location", "").strip(),

        "contact_phone": form.get("contact_phone", "").strip(),
        "contact_email": form.get("contact_email", "").strip(),
        "contact_facebook": form.get("contact_facebook", "").strip(),

        "condition": form.get("condition", "").strip(),
        "title_status": form.get("title_status", "").strip(),
        "seller_type": form.get("seller_type", "").strip(),
        "seller_status": form.get("seller_status", "").strip(),
        "accident_history": form.get("accident_history", "").strip(),
        "lien_status": form.get("lien_status", "").strip(),
        "rebuilt_status": form.get("rebuilt_status", "").strip(),
        "horsepower": form.get("horsepower", "").strip(),
        "turbo": form.get("turbo", "").strip(),
        "mods": form.get("mods", "").strip(),
        "rust": form.get("rust", "").strip(),
        "issues": form.get("issues", "").strip(),
        "maintenance": form.get("maintenance", "").strip(),
        "tires": form.get("tires", "").strip(),
        "brakes": form.get("brakes", "").strip(),
        "suspension": form.get("suspension", "").strip(),
        "extras": form.get("extras", "").strip(),
        "notes": form.get("notes", "").strip(),
    }
    data["status"] = status
    data["risk_level"] = normalize_choice(data["risk_level"], RISK_LEVELS)
    data["transmission"] = normalize_choice(data["transmission"], TRANSMISSION_OPTIONS)
    data["drivetrain"] = normalize_choice(data["drivetrain"], DRIVETRAIN_OPTIONS)
    data["fuel_type"] = normalize_choice(data["fuel_type"], FUEL_TYPE_OPTIONS)
    data["body_style"] = normalize_choice(data["body_style"], BODY_STYLE_OPTIONS)
    data["turbo"] = normalize_choice(data["turbo"], TURBO_OPTIONS)
    data["seller_type"] = normalize_choice(data["seller_type"], SELLER_TYPE_OPTIONS)
    data["seller_status"] = normalize_choice(data["seller_status"], SELLER_STATUS_OPTIONS)
    data["title_status"] = normalize_choice(data["title_status"], TITLE_STATUS_OPTIONS)
    data["lien_status"] = normalize_choice(data["lien_status"], LIEN_STATUS_OPTIONS)
    data["rebuilt_status"] = normalize_choice(data["rebuilt_status"], REBUILT_STATUS_OPTIONS)
    data["accident_history"] = normalize_choice(data["accident_history"], ACCIDENT_HISTORY_OPTIONS)
    data["condition"] = normalize_choice(data["condition"], CONDITION_OPTIONS)

    data["location_search"] = build_location_search(data["location"])
    data["image_urls"] = [line.strip() for line in data["image_urls_raw"].splitlines() if line.strip()]

    data["year"] = to_int_or_none(data["year"])
    data["mileage_km"] = to_int_or_none(data["mileage_km"])
    data["doors"] = to_int_or_none(data["doors"])
    data["seats"] = to_int_or_none(data["seats"])

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
            year, make, model, trim, mileage_km, transmission, drivetrain,
            fuel_type, engine, body_style, exterior_color, interior_color,
            doors, seats, vin, location, location_search, contact_phone, contact_email,
            contact_facebook, condition, title_status, seller_type,
            seller_status, accident_history, lien_status,
            rebuilt_status, horsepower, turbo, mods, rust, issues,
            maintenance, tires, brakes, suspension, extras, notes
        )
        VALUES (
            ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?,
            ?, ?, ?,
            ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?
        )
    """, (
        data["title"], data["price"], data["description"], data["risk_level"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["trim"], data["mileage_km"], data["transmission"], data["drivetrain"],
        data["fuel_type"], data["engine"], data["body_style"], data["exterior_color"], data["interior_color"],
        data["doors"], data["seats"], data["vin"], data["location"], data["location_search"], data["contact_phone"], data["contact_email"],
        data["contact_facebook"], data["condition"], data["title_status"], data["seller_type"],
        data["seller_status"], data["accident_history"], data["lien_status"],
        data["rebuilt_status"], data["horsepower"], data["turbo"], data["mods"], data["rust"], data["issues"],
        data["maintenance"], data["tires"], data["brakes"], data["suspension"], data["extras"], data["notes"],
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
            trim = ?,
            mileage_km = ?,
            transmission = ?,
            drivetrain = ?,
            fuel_type = ?,
            engine = ?,
            body_style = ?,
            exterior_color = ?,
            interior_color = ?,
            doors = ?,
            seats = ?,
            vin = ?,
            location = ?,
            location_search = ?,
            condition = ?,
            title_status = ?,
            seller_type = ?,
            seller_status = ?,
            accident_history = ?,
            lien_status = ?,
            rebuilt_status = ?,
            horsepower = ?,
            turbo = ?,
            mods = ?,
            rust = ?,
            issues = ?,
            maintenance = ?,
            tires = ?,
            brakes = ?,
            suspension = ?,
            extras = ?,
            notes = ?,
            contact_phone = ?,
            contact_email = ?,
            contact_facebook = ?
        WHERE id = ?
    """, (
        data["title"], data["price"], data["description"], data["risk_level"], data["source_url"], data["status"],
        data["year"], data["make"], data["model"], data["trim"], data["mileage_km"], data["transmission"], data["drivetrain"],
        data["fuel_type"], data["engine"], data["body_style"], data["exterior_color"], data["interior_color"],
        data["doors"], data["seats"], data["vin"], data["location"], data["location_search"], data["condition"], data["title_status"],
        data["seller_type"], data["seller_status"], data["accident_history"], data["lien_status"],
        data["rebuilt_status"], data["horsepower"], data["turbo"], data["mods"], data["rust"], data["issues"],
        data["maintenance"], data["tires"], data["brakes"], data["suspension"], data["extras"], data["notes"],
        data["contact_phone"], data["contact_email"], data["contact_facebook"],
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