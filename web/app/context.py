from .config import Config


def inject_admin_path():
    """Делает ADMIN_PATH доступным во всех шаблонах."""
    return {"ADMIN_PATH": Config.ADMIN_PATH}