"""
Flask app factory for Auto-Hub.

Registers blueprints (public, admin, pages),
initializes the database, attaches context processors,
enables gzip compression and static file caching.
"""

from flask import Flask, render_template, request, send_from_directory
from flask_compress import Compress
import bleach
import markdown as md

from .config import Config
from .context import inject_admin_path
from .db import init_db
from .extensions import csrf, limiter
from .routes.admin import admin_bp
from .routes.pages import pages_bp
from .routes.public import public_bp


def create_app(config_object=None) -> Flask:
    app = Flask(
        __name__,
        template_folder="../templates",
        static_folder="../static",
    )
    if config_object:
        app.config.from_object(config_object)
    else:
        app.config.from_object(Config)

    # Gzip/Brotli response compression
    Compress(app)

    csrf.init_app(app)
    limiter.init_app(app)

    # Markdown rendering filter for Jinja2
    _ALLOWED_TAGS = list(bleach.ALLOWED_TAGS) + ["p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "ul", "ol", "li", "a"]
    _ALLOWED_ATTRS = {"a": ["href", "title"], "img": ["src", "alt"]}

    @app.template_filter("markdown")
    def markdown_filter(text):
        html = md.markdown(text or "", extensions=["nl2br"])
        return bleach.clean(html, tags=_ALLOWED_TAGS, attributes=_ALLOWED_ATTRS, strip=True)

    init_db(app.config["DB_PATH"])
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
        response.cache_control.max_age = 604800  # 7 days
        return response

    @app.errorhandler(404)
    def not_found(_error):
        return render_template("public/404.html"), 404

    @app.after_request
    def add_cache_headers(response):
        # Cache static files and images for one week
        if request.path.startswith("/static/") or request.path.startswith("/data/"):
            response.headers["Cache-Control"] = "public, max-age=604800, immutable"
        return response

    return app
