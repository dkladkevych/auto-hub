"""
Admin panel routes (require authentication).

CRUD operations for listings: create, edit, delete,
archive, publish. Media management (drag & drop).

Connected to services: admin (business logic), utils/images (media handling).
"""

from flask import Blueprint, current_app, jsonify, redirect, render_template, request, session, url_for
from werkzeug.security import check_password_hash

from ..decorators import admin_required
from ..extensions import limiter
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
from ..utils.images import (
    delete_listing_images,
    sync_listing_images,
    validate_images,
)

admin_bp = Blueprint("admin", __name__)


@admin_bp.route("/login", methods=["GET", "POST"])
@limiter.limit("5 per minute")
def admin_login():
    error = None

    if request.method == "POST":
        password = request.form.get("password", "").strip()

        stored_hash = current_app.config.get("ADMIN_PASSWORD_HASH", "")
        stored_plain = current_app.config.get("ADMIN_PASSWORD", "")

        valid = False
        if stored_hash:
            valid = check_password_hash(stored_hash, password)
        elif stored_plain:
            valid = password == stored_plain

        if valid:
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
        errors = validate_listing_form(data)
        form_data = dict(request.form)

        wants_json = request.headers.get("X-Requested-With") == "XMLHttpRequest"

        if errors:
            if wants_json:
                return jsonify({"errors": errors})
            return render_template("admin/new.html", errors=errors, form_data=form_data)

        uploaded_files = request.files.getlist("images")
        img_error = validate_images(uploaded_files)
        if img_error:
            if wants_json:
                return jsonify({"error": img_error})
            return render_template("admin/new.html", error=img_error, form_data=form_data)

        listing_id = create_listing(data)

        try:
            sync_listing_images(listing_id, [], uploaded_files)
        except ValueError as e:
            delete_listing_by_id(listing_id)
            delete_listing_images(listing_id)
            if wants_json:
                return jsonify({"error": str(e)})
            return render_template("admin/new.html", error=str(e), form_data=form_data)

        return redirect(url_for("admin.admin_dashboard"))

    return render_template("admin/new.html", error=None, form_data={})


@admin_bp.route("/delete/<int:id>", methods=["POST"])
@admin_required
def delete_listing(id):
    delete_listing_images(id)
    delete_listing_by_id(id)
    return redirect(url_for("admin.admin_dashboard"))


@admin_bp.route("/edit/<int:id>", methods=["GET", "POST"])
@admin_required
def admin_edit(id):
    car, existing_images = get_listing_for_edit(id)

    if request.method == "POST":
        data = parse_listing_form(request.form)
        errors = validate_listing_form(data)
        form_data = dict(request.form)

        wants_json = request.headers.get("X-Requested-With") == "XMLHttpRequest"

        if errors:
            if wants_json:
                return jsonify({"errors": errors})
            return render_template(
                "admin/edit.html",
                car=car,
                existing_images=existing_images,
                errors=errors,
                form_data=form_data,
            )

        uploaded_files = request.files.getlist("images")
        keep_images = request.form.getlist("keep_images")

        img_error = validate_images(uploaded_files)
        if img_error:
            if wants_json:
                return jsonify({"error": img_error})
            return render_template(
                "admin/edit.html",
                car=car,
                existing_images=existing_images,
                error=img_error,
                form_data=form_data,
            )

        update_listing(id, data)

        has_any = keep_images or (uploaded_files and any(f and f.filename for f in uploaded_files))

        if has_any:
            try:
                sync_listing_images(id, keep_images, uploaded_files)
            except ValueError as e:
                if wants_json:
                    return jsonify({"error": str(e)})
                return render_template(
                    "admin/edit.html",
                    car=car,
                    existing_images=existing_images,
                    error=str(e),
                    form_data=form_data,
                )
        else:
            delete_listing_images(id)

        return redirect(url_for("admin.admin_dashboard"))

    return render_template(
        "admin/edit.html",
        car=car,
        existing_images=existing_images,
        error=None,
        form_data={},
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
