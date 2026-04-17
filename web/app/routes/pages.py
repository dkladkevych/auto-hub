"""
Статические и SEO-страницы (terms, privacy, 404, robots.txt, sitemap.xml).

sitemap.xml формируется динамически из активных объявлений в inventory.
"""

from flask import Blueprint, current_app, render_template, request

from ..db import get_db

pages_bp = Blueprint("pages", __name__)


@pages_bp.route("/terms")
def terms():
    return render_template("public/terms.html")


@pages_bp.route("/privacy")
def privacy():
    return render_template("public/privacy.html")


@pages_bp.route("/robots.txt")
def robots():
    admin_path = current_app.config.get("ADMIN_PATH", "admin")
    sitemap_url = request.host_url.rstrip("/") + "/sitemap.xml"
    lines = [
        "User-agent: *",
        "Allow: /",
        f"Disallow: /{admin_path}/",
        f"Sitemap: {sitemap_url}",
        "",
    ]
    return "\n".join(lines), 200, {"Content-Type": "text/plain; charset=utf-8"}


@pages_bp.route("/sitemap.xml")
def sitemap():
    db = get_db()
    listings = db.execute(
        "SELECT id, created_at FROM inventory WHERE status IN ('active', 'demo') ORDER BY id"
    ).fetchall()
    db.close()
    return render_template(
        "public/sitemap.xml",
        listings=listings,
        host=request.host_url.rstrip("/"),
    ), 200, {"Content-Type": "application/xml; charset=utf-8"}
