"""
Public routes (no authentication required).

- Home page with filters, search, and pagination
- Listing detail page with gallery
- Saved listings page

Connected to services: listings (data), stats (view logging).
"""

from flask import Blueprint, render_template, request

from ..services.listings import get_home_listings, get_listing_page_data, get_saved_listings
from ..utils.stats import log_listing_view, log_site_visit

public_bp = Blueprint("public", __name__)


@public_bp.route("/")
def home():
    log_site_visit()
    page = request.args.get("page", 1, type=int)
    listings, has_any_filter, page, total_pages = get_home_listings(request.args, page=page)

    return render_template(
        "public/home.html",
        listings=listings,
        filters=request.args,
        has_any_filter=has_any_filter,
        page=page,
        total_pages=total_pages,
    )


@public_bp.route("/listing/<int:id>")
def listing(id):
    log_listing_view(id)
    car, images, thumbs = get_listing_page_data(id)

    return render_template(
        "public/listing.html",
        car=car,
        images=images,
        thumbs=thumbs,
    )


@public_bp.route("/saved")
def saved():
    ids_param = request.args.get("ids", "").strip()
    id_list = []
    if ids_param:
        for part in ids_param.split(","):
            try:
                id_list.append(int(part.strip()))
            except ValueError:
                continue

    listings = get_saved_listings(id_list)
    return render_template("public/saved.html", listings=listings)
