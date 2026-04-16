from flask import Flask, render_template, send_from_directory

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
        return send_from_directory(Config.DATA_DIR, filename)

    @app.errorhandler(404)
    def not_found(_error):
        return render_template("public/404.html"), 404

    return app
