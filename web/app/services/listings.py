"""
Бизнес-логика публичного каталога объявлений.

- Фильтрация и поиск по цене, году, пробегу, риску, локации
- Формирование списка для главной страницы с пагинацией
- Данные для детальной страницы (с галереей)

Связан с: db (inventory), utils/images (превью + thumbnails), utils/location (нормализация локаций).
"""

from flask import abort

from ..db import get_db
from ..utils.images import get_listing_image_urls, get_preview_image
from ..utils.location import build_location_search


def _to_int(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _location_matches(car_location: str, search_query: str, include_unknown: bool) -> bool:
    """Проверяет, совпадает ли локация объявления с поисковым запросом."""
    if not search_query or not search_query.strip():
        return True

    car_location = (car_location or "").strip()
    if not car_location:
        return include_unknown

    search_tokens = set(build_location_search(search_query).split())
    car_tokens = set(build_location_search(car_location).split())

    return bool(search_tokens & car_tokens)


def get_home_listings(filters, page: int = 1, per_page: int = 12):
    """Формирует список публичных объявлений с фильтрами и пагинацией."""

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
        FROM inventory
        WHERE status IN ('active', 'demo')
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

    # Фильтрация по локации на Python + формирование словарей
    listings = []
    for row in rows:
        d = dict(row)
        if location and not _location_matches(d.get("location"), location, include_unknown):
            continue
        d["preview_image"] = get_preview_image(d["id"], thumb=True)
        listings.append(d)

    total = len(listings)
    total_pages = max(1, (total + per_page - 1) // per_page)
    page = max(1, min(page, total_pages))
    offset = (page - 1) * per_page

    paginated = listings[offset:offset + per_page]

    return paginated, has_any_filter, page, total_pages


def get_saved_listings(id_list: list[int]):
    """Возвращает объявления по списку ID (для страницы Saved)."""
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
        listings.append(d)

    return listings


def get_listing_page_data(listing_id: int):
    """Возвращает объявление и его картинки."""
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

    image_urls = get_listing_image_urls(listing_id)
    if not image_urls:
        image_urls = ["/static/images/empty.png"]
    images = [{"image_url": url} for url in image_urls]

    return car, images
