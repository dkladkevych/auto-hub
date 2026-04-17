"""
Декораторы доступа.

@admin_required — проверяет авторизацию администратора через сессию.
Используется на всех маршрутах админ-панели.
"""

from functools import wraps

from flask import redirect, session, url_for


def admin_required(view_func):
    """Пускает внутрь только авторизованного админа."""
    @wraps(view_func)
    def wrapper(*args, **kwargs):
        if not session.get("is_admin"):
            return redirect(url_for("admin.admin_login"))
        return view_func(*args, **kwargs)

    return wrapper