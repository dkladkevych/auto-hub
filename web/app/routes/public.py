from flask import Blueprint, render_template, request

from ..services.listings import get_home_listings, get_listing_page_data

public_bp = Blueprint("public", __name__)


@public_bp.route("/")
def home():
    listings, has_any_filter = get_home_listings(request.args)

    return render_template(
        "public/home.html",
        listings=listings,
        filters=request.args,
        has_any_filter=has_any_filter,
    )


@public_bp.route("/listing/<int:id>")
def listing(id):
    car, images, masked_vin = get_listing_page_data(id)

    return render_template(
        "public/listing.html",
        car=car,
        images=images,
        masked_vin=masked_vin,
    )