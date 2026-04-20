"""
Access decorators.

@admin_required — checks admin authorization via session.
Used on all admin panel routes.
"""

from functools import wraps

from flask import redirect, session, url_for


def admin_required(view_func):
    """Allows only an authenticated admin inside."""
    @wraps(view_func)
    def wrapper(*args, **kwargs):
        if not session.get("is_admin"):
            return redirect(url_for("admin.admin_login"))
        return view_func(*args, **kwargs)

    return wrapper