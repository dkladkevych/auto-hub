import sqlite3

from .config import Config


def get_db():
    """Открывает новое соединение с SQLite.

    Сейчас приложение маленькое, поэтому простого connect на запрос достаточно.
    """
    conn = sqlite3.connect(Config.DB_PATH)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    return conn


def init_db():
    """Создает таблицы, если база пустая.

    Важно: это должно вызываться и при локальном запуске, и при gunicorn.
    """
    conn = sqlite3.connect(Config.DB_PATH)
    conn.execute("PRAGMA foreign_keys = ON")

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,

            title TEXT NOT NULL,
            price INTEGER NOT NULL,
            description TEXT NOT NULL,
            risk_level TEXT NOT NULL,
            source_url TEXT,
            status TEXT NOT NULL DEFAULT 'active',

            year INTEGER,
            make TEXT,
            model TEXT,
            trim TEXT,
            mileage_km INTEGER,
            transmission TEXT,
            drivetrain TEXT,
            fuel_type TEXT,
            engine TEXT,
            body_style TEXT,
            exterior_color TEXT,
            interior_color TEXT,
            doors INTEGER,
            seats INTEGER,
            vin TEXT,
            location TEXT,
            location_search TEXT,
            contact_phone TEXT,
            contact_email TEXT,
            contact_facebook TEXT,
            condition TEXT,
            title_status TEXT,
            seller_type TEXT,
            seller_status TEXT,
            accident_history TEXT,
            lien_status TEXT,
            rebuilt_status TEXT,
            horsepower TEXT,
            turbo TEXT,
            mods TEXT,
            rust TEXT,
            issues TEXT,
            maintenance TEXT,
            tires TEXT,
            brakes TEXT,
            suspension TEXT,
            extras TEXT,
            notes TEXT,

            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listing_images (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            listing_id INTEGER NOT NULL,
            image_url TEXT NOT NULL,
            sort_order INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE
        )
    """)

    conn.commit()
    conn.close()