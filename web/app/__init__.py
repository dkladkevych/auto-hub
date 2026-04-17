"""
App factory для Flask-приложения Auto-Hub.

Регистрирует blueprints (public, admin, pages),
инициализирует БД, подключает context processors,
включает gzip-сжатие и кэширование статики.
"""

from flask import Flask, render_template, request, send_from_directory
from flask_compress import Compress

from .config import Config
from .context import inject_admin_path
from .db import init_db
from .routes.admin import admin_bp
from .routes.pages import pages_bp
from .routes.public import public_bp


def create_app() -> Flask:
    app = Flask(
        __name__,
        template_folder="../templates",
        static_folder="../static",
    )
    app.config.from_object(Config)

    # Gzip/Brotli сжатие ответов
    Compress(app)

    init_db()
    app.context_processor(inject_admin_path)

    app.register_blueprint(public_bp)
    app.register_blueprint(
        admin_bp,
        url_prefix=f"/{app.config['ADMIN_PATH']}",
    )
    app.register_blueprint(pages_bp)

    @app.route("/data/<path:filename>")
    def data_file(filename):
        response = send_from_directory(Config.DATA_DIR, filename)
        response.cache_control.max_age = 604800  # 7 дней
        return response

    @app.errorhandler(404)
    def not_found(_error):
        return render_template("public/404.html"), 404

    @app.after_request
    def add_cache_headers(response):
        # Кэшируем статику и изображения на неделю
        if request.path.startswith("/static/") or request.path.startswith("/data/"):
            response.headers["Cache-Control"] = "public, max-age=604800, immutable"
        return response

    return app
