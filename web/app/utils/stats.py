from ..db import get_db


def log_site_visit():
    """Логирует посещение главной страницы."""
    db = get_db()
    db.execute("INSERT INTO site_visits DEFAULT VALUES")
    db.commit()
    db.close()


def log_listing_view(listing_id: int):
    """Логирует просмотр конкретного объявления."""
    db = get_db()
    db.execute("INSERT INTO listing_views (listing_id) VALUES (?)", (listing_id,))
    db.commit()
    db.close()
