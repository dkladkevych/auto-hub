from flask import Blueprint, current_app, redirect, render_template, request, session, url_for

from ..decorators import admin_required
from ..services.admin import (
    create_listing,
    delete_listing_by_id,
    get_dashboard_data,
    get_dashboard_listings_block,
    get_listing_for_edit,
    parse_listing_form,
    set_listing_status,
    update_listing,
    validate_listing_form,
)

admin_bp = Blueprint("admin", __name__)


@admin_bp.route("/login", methods=["GET", "POST"])
def admin_login():
    error = None

    if request.method == "POST":
        password = request.form.get("password", "").strip()

        if password == current_app.config["ADMIN_PASSWORD"]:
            session["is_admin"] = True
            return redirect(url_for("admin.admin_dashboard"))

        error = "Wrong password"

    return render_template("admin/login.html", error=error)


@admin_bp.route("/logout", methods=["POST"])
@admin_required
def admin_logout():
    session.clear()
    return redirect(url_for("public.home"))


@admin_bp.route("/")
@admin_required
def admin_dashboard():
    page = request.args.get("page", 1, type=int)
    data = get_dashboard_data(page)
    return render_template("admin/dashboard.html", **data)


@admin_bp.route("/new", methods=["GET", "POST"])
@admin_required
def admin_new():
    if request.method == "POST":
        data = parse_listing_form(request.form)
        error = validate_listing_form(data)

        if error:
            return render_template("admin/new.html", error=error)

        create_listing(data)
        return redirect(url_for("admin.admin_dashboard"))

    return render_template("admin/new.html", error=None)


@admin_bp.route("/delete/<int:id>", methods=["POST"])
@admin_required
def delete_listing(id):
    delete_listing_by_id(id)
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/edit/<int:id>", methods=["GET", "POST"])
@admin_required
def admin_edit(id):
    car, existing_images = get_listing_for_edit(id)
    image_urls_text = "\n".join([img["image_url"] for img in existing_images])

    if request.method == "POST":
        data = parse_listing_form(request.form)
        error = validate_listing_form(data)

        if error:
            return render_template(
                "admin/edit.html",
                car=car,
                existing_images=existing_images,
                image_urls_text=image_urls_text,
                error=error,
            )

        update_listing(id, data)
        return redirect(url_for("admin.admin_dashboard"))

    return render_template(
        "admin/edit.html",
        car=car,
        existing_images=existing_images,
        image_urls_text=image_urls_text,
        error=None,
    )


@admin_bp.route("/listings-block")
@admin_required
def admin_listings_block():
    page = request.args.get("page", 1, type=int)
    data = get_dashboard_listings_block(page)
    return render_template("admin/partials/listings_table.html", **data)


@admin_bp.route("/archive/<int:id>", methods=["POST"])
@admin_required
def archive_listing(id):
    set_listing_status(id, "archived")
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/unarchive/<int:id>", methods=["POST"])
@admin_required
def unarchive_listing(id):
    set_listing_status(id, "active")
    return redirect(url_for("admin.admin_dashboard"))

@admin_bp.route("/publish/<int:id>", methods=["POST"])
@admin_required
def publish_listing(id):
    set_listing_status(id, "active")
    return redirect(url_for("admin.admin_dashboard"))