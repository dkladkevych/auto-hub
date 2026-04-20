"""
Centralized application configuration.

Reads variables from .env (SECRET_KEY, ADMIN_PASSWORD, ADMIN_PATH)
and defines data folder paths (DB, images).
"""

import os
from dotenv import load_dotenv

load_dotenv()

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


class Config:
    """Centralized app configuration."""

    SECRET_KEY = os.getenv("SECRET_KEY", "fallback_secret")
    ADMIN_PASSWORD_HASH = os.getenv("ADMIN_PASSWORD_HASH", "")
    ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "fallback_password")
    ADMIN_PATH = os.getenv("ADMIN_PATH", "admin")

    DATA_DIR = os.path.join(BASE_DIR, "data")
    DB_DIR = os.path.join(DATA_DIR, "db")
    LISTINGS_DIR = os.path.join(DATA_DIR, "listings")
    DB_PATH = os.path.join(DB_DIR, "db.sqlite")
