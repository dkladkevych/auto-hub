import os
import sqlite3

from flask import current_app

from .config import Config


def get_db():
    """Open a new SQLite connection per request.

    The app is small, so a simple connect per request is sufficient.
    """
    conn = sqlite3.connect(current_app.config["DB_PATH"])
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    return conn


def init_db(db_path=None):
    """Create tables if the database is empty.

    Must be called both for local development and gunicorn.
    """
    if db_path is None:
        db_path = Config.DB_PATH
    if db_path != ":memory:":
        db_dir = os.path.dirname(db_path)
        if db_dir:
            os.makedirs(db_dir, exist_ok=True)
    os.makedirs(Config.LISTINGS_DIR, exist_ok=True)

    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA foreign_keys = ON")

    conn.execute("""
        CREATE TABLE IF NOT EXISTS inventory (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            account_id INTEGER NOT NULL DEFAULT 0,

            title TEXT NOT NULL,
            price INTEGER NOT NULL,
            description TEXT NOT NULL,
            source_url TEXT,
            status TEXT NOT NULL DEFAULT 'active',

            year INTEGER,
            make TEXT,
            model TEXT,
            mileage_km INTEGER,
            location TEXT,
            condition TEXT,
            notes TEXT,
            transmission TEXT,
            drivetrain TEXT,
            published_at DATETIME
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS stats (
            target_type TEXT NOT NULL CHECK(target_type IN ('site', 'listing')),
            target_id INTEGER NOT NULL DEFAULT 0,
            view_count INTEGER DEFAULT 0,
            PRIMARY KEY (target_type, target_id)
        )
    """)

    # Migration: add transmission and drivetrain if missing
    for col in ("transmission", "drivetrain"):
        try:
            conn.execute(f"ALTER TABLE inventory ADD COLUMN {col} TEXT")
        except sqlite3.OperationalError:
            pass

    # Migration: add published_at if missing
    try:
        conn.execute("ALTER TABLE inventory ADD COLUMN published_at DATETIME")
    except sqlite3.OperationalError:
        pass

    conn.commit()
    conn.close()
