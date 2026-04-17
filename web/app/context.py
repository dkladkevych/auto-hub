"""
Context processor — делает ADMIN_PATH доступным во всех Jinja-шаблонах.

Используется для динамической генерации URL админ-панели
(скрытый путь задаётся через переменную окружения).
"""

from .config import Config


def inject_admin_path():
    """Делает ADMIN_PATH доступным во всех шаблонах."""
    return {"ADMIN_PATH": Config.ADMIN_PATH}