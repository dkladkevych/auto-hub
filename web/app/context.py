"""
Context processor — makes ADMIN_PATH available in all Jinja templates.

Used for dynamic admin panel URL generation
(the hidden path is set via an environment variable).
"""

from .config import Config


def inject_admin_path():
    """Makes ADMIN_PATH available in all templates."""
    return {"ADMIN_PATH": Config.ADMIN_PATH}