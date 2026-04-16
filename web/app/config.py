import os
from dotenv import load_dotenv

load_dotenv()

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


class Config:
    """Централизованный конфиг приложения."""

    SECRET_KEY = os.getenv("SECRET_KEY", "fallback_secret")
    ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "fallback_password")
    ADMIN_PATH = os.getenv("ADMIN_PATH", "admin")

    DB_PATH = os.path.join(BASE_DIR, "db.sqlite")