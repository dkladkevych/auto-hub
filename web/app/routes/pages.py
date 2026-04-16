from flask import Blueprint, render_template

pages_bp = Blueprint("pages", __name__)


@pages_bp.route("/terms")
def terms():
    return render_template("public/terms.html")


@pages_bp.route("/privacy")
def privacy():
    return render_template("public/privacy.html")