import os
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
    os.makedirs(Config.DB_DIR, exist_ok=True)
    os.makedirs(Config.LISTINGS_DIR, exist_ok=True)

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
            mileage_km INTEGER,
            location TEXT,
            location_search TEXT,
            condition TEXT,
            notes TEXT,
            seller_status TEXT,

            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listing_images (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            listing_id INTEGER NOT NULL,
            image_url TEXT NOT NULL,
            FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS site_visits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            visited_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listing_views (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            listing_id INTEGER NOT NULL,
            viewed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE
        )
    """)

    conn.commit()
    conn.close()
