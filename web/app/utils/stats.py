"""
Логирование просмотров через UPSERT в таблицу stats.

- site visits: счётчик посещений главной страницы
- listing views: счётчик просмотров конкретного объявления

Использует SQLite ON CONFLICT для атомарного обновления счётчиков.
"""

from ..db import get_db


def log_site_visit():
    """Увеличивает счётчик посещений сайта."""
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
    """Увеличивает счётчик просмотров конкретного объявления."""
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
