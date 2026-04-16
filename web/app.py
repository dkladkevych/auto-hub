import os
import sqlite3
from functools import wraps
from flask import Flask, render_template, request, redirect, session, url_for, abort
import re
from dotenv import load_dotenv

load_dotenv()

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DB = os.path.join(BASE_DIR, "db.sqlite")

app = Flask(__name__)

app.secret_key = os.getenv("SECRET_KEY", "fallback_secret")

ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "fallback_password")
ADMIN_PATH = os.getenv("ADMIN_PATH", "admin")

def get_db():
    conn = sqlite3.connect(DB)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    return conn


def init_db():
    conn = sqlite3.connect(DB)
    conn.execute("PRAGMA foreign_keys = ON")

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,

            title TEXT NOT NULL,
            price INTEGER NOT NULL,
            description TEXT NOT NULL,
            risk_level TEXT NOT NULL,
            source_url TEXT,
            status TEXT NOT NULL DEFAULT 'active',

            year INTEGER,
            make TEXT,
            model TEXT,
            trim TEXT,
            mileage_km INTEGER,
            transmission TEXT,
            drivetrain TEXT,
            fuel_type TEXT,
            engine TEXT,
            body_style TEXT,
            exterior_color TEXT,
            interior_color TEXT,
            doors INTEGER,
            seats INTEGER,
            vin TEXT,
            location TEXT,
            location_search TEXT,
            contact_phone TEXT,
            contact_email TEXT,
            contact_facebook TEXT,
            condition TEXT,
            title_status TEXT,
            seller_type TEXT,
            seller_status TEXT,
            accident_history TEXT,
            lien_status TEXT,
            rebuilt_status TEXT,
            horsepower TEXT,
            turbo TEXT,
            mods TEXT,
            rust TEXT,
            issues TEXT,
            maintenance TEXT,
            tires TEXT,
            brakes TEXT,
            suspension TEXT,
            extras TEXT,
            notes TEXT,

            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE IF NOT EXISTS listing_images (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            listing_id INTEGER NOT NULL,
            image_url TEXT NOT NULL,
            sort_order INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE
        )
    """)

    conn.commit()
    conn.close()


def admin_required(view_func):
    @wraps(view_func)
    def wrapper(*args, **kwargs):
        if not session.get("is_admin"):
            return redirect(url_for("admin_login"))
        return view_func(*args, **kwargs)
    return wrapper


def mask_vin(vin):
    if not vin:
        return None

    vin = vin.strip()

    if len(vin) <= 6:
        return vin

    return vin[:3] + "*" * (len(vin) - 7) + vin[-4:]

def build_location_search(location):
    if not location:
        return ""

    raw = location.strip().lower()

    province_aliases = {
        "on": "ontario",
        "ontario": "on",
        "bc": "british columbia",
        "british columbia": "bc",
        "ab": "alberta",
        "alberta": "ab",
        "qc": "quebec",
        "quebec": "qc",
        "mb": "manitoba",
        "manitoba": "mb",
        "sk": "saskatchewan",
        "saskatchewan": "sk",
        "ns": "nova scotia",
        "nova scotia": "ns",
        "nb": "new brunswick",
        "new brunswick": "nb",
        "pe": "prince edward island",
        "prince edward island": "pe",
        "nl": "newfoundland labrador",
        "newfoundland": "nl",
        "newfoundland labrador": "nl"
    }

    clean = re.sub(r"[^a-z0-9\s,]", " ", raw)
    parts = re.split(r"[\s,]+", clean)
    parts = [p for p in parts if p]

    expanded = set(parts)

    joined = " ".join(parts)
    if joined in province_aliases:
        expanded.add(province_aliases[joined])

    for part in parts:
        if part in province_aliases:
            expanded.add(province_aliases[part])

    if "mississauga" in expanded or "toronto" in expanded or "brampton" in expanded or "vaughan" in expanded:
        expanded.add("gta")
        expanded.add("greater toronto area")

    return " ".join(sorted(expanded))

@app.route("/")
def home():
    q = request.args.get("q", "").strip()
    price_min = request.args.get("price_min", "").strip()
    price_max = request.args.get("price_max", "").strip()
    year_min = request.args.get("year_min", "").strip()
    year_max = request.args.get("year_max", "").strip()
    mileage_min = request.args.get("mileage_min", "").strip()
    mileage_max = request.args.get("mileage_max", "").strip()
    transmission = request.args.get("transmission", "").strip()
    drivetrain = request.args.get("drivetrain", "").strip()
    risk_level = request.args.get("risk_level", "").strip()
    location = request.args.get("location", "").strip()
    include_unknown = request.args.get("include_unknown") == "on"

    def to_int(v):
        try:
            return int(v)
        except:
            return None

    price_min = to_int(price_min)
    price_max = to_int(price_max)
    year_min = to_int(year_min)
    year_max = to_int(year_max)
    mileage_min = to_int(mileage_min)
    mileage_max = to_int(mileage_max)

    db = get_db()

    query = """
        SELECT
            l.*,
            (
                SELECT li.image_url
                FROM listing_images li
                WHERE li.listing_id = l.id
                ORDER BY li.sort_order ASC, li.id ASC
                LIMIT 1
            ) AS preview_image
        FROM listings l
        WHERE l.status = 'active'
    """

    params = []

    has_any_filter = any([
        q,
        location,
        price_min is not None,
        price_max is not None,
        year_min is not None,
        year_max is not None,
        mileage_min is not None,
        mileage_max is not None,
        transmission,
        drivetrain,
        risk_level
    ])

    # --- SEARCH ---
    if q:
        query += """
            AND (
                l.title LIKE ?
                OR l.make LIKE ?
                OR l.model LIKE ?
                OR l.trim LIKE ?
                OR l.description LIKE ?
                OR l.location LIKE ?
            )
        """
        like = f"%{q}%"
        params += [like, like, like, like, like, like]

    # --- LOCATION ---
    if location:
        normalized_location = build_location_search(location)

        if include_unknown:
            query += """
                AND (
                    l.location_search LIKE ?
                    OR l.location LIKE ?
                    OR l.location IS NULL
                    OR l.location = ''
                    OR l.location_search IS NULL
                    OR l.location_search = ''
                )
            """
        else:
            query += """
                AND (
                    l.location_search LIKE ?
                    OR l.location LIKE ?
                )
            """

        params.append(f"%{normalized_location}%")
        params.append(f"%{location}%")

    # --- PRICE ---
    if price_min is not None:
        query += " AND l.price >= ?"
        params.append(price_min)

    if price_max is not None:
        query += " AND l.price <= ?"
        params.append(price_max)

    # --- YEAR ---
    if year_min is not None:
        query += " AND l.year >= ?"
        params.append(year_min)

    if year_max is not None:
        query += " AND l.year <= ?"
        params.append(year_max)

    # --- MILEAGE ---
    if mileage_min is not None:
        query += " AND l.mileage_km >= ?"
        params.append(mileage_min)

    if mileage_max is not None:
        query += " AND l.mileage_km <= ?"
        params.append(mileage_max)

    # --- TRANSMISSION ---
    if transmission:
        if include_unknown:
            query += " AND (l.transmission = ? OR l.transmission IS NULL OR l.transmission = '')"
        else:
            query += " AND l.transmission = ?"
        params.append(transmission)

    # --- DRIVETRAIN ---
    if drivetrain:
        if include_unknown:
            query += " AND (l.drivetrain = ? OR l.drivetrain IS NULL OR l.drivetrain = '')"
        else:
            query += " AND l.drivetrain = ?"
        params.append(drivetrain)

    # --- RISK ---
    if risk_level:
        query += " AND l.risk_level = ?"
        params.append(risk_level)

    query += " ORDER BY l.created_at DESC"

    listings = db.execute(query, params).fetchall()
    db.close()

    return render_template(
        "home.html",
        listings=listings,
        filters=request.args,
        has_any_filter=has_any_filter
    )

@app.route("/listing/<int:id>")
def listing(id):
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM listings
        WHERE id = ?
    """, (id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    images = db.execute("""
        SELECT *
        FROM listing_images
        WHERE listing_id = ?
        ORDER BY sort_order ASC, id ASC
    """, (id,)).fetchall()

    db.close()

    masked_vin = mask_vin(car["vin"])

    return render_template("listing.html", car=car, images=images, masked_vin=masked_vin)


@app.route(f"/{ADMIN_PATH}/login", methods=["GET", "POST"])
def admin_login():
    error = None

    if request.method == "POST":
        password = request.form.get("password", "").strip()

        if password == ADMIN_PASSWORD:
            session["is_admin"] = True
            return redirect(url_for("admin_new"))
        else:
            error = "Wrong password"

    return render_template("admin_login.html", error=error)


@app.route(f"/{ADMIN_PATH}/logout", methods=["POST"])
@admin_required
def admin_logout():
    session.clear()
    return redirect(url_for("home"))

@app.route(f"/{ADMIN_PATH}")
@admin_required
def admin_dashboard():
    page = request.args.get("page", 1, type=int)
    per_page = 10
    offset = (page - 1) * per_page

    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM listings").fetchone()["count"]
    active_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE status = 'active'").fetchone()["count"]
    archived_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE status = 'archived'").fetchone()["count"]
    high_risk_count = db.execute("SELECT COUNT(*) AS count FROM listings WHERE risk_level = 'HIGH'").fetchone()["count"]

    total_pages = max(1, (total_count + per_page - 1) // per_page)

    listings = db.execute("""
        SELECT
            l.*,
            (
                SELECT li.image_url
                FROM listing_images li
                WHERE li.listing_id = l.id
                ORDER BY li.sort_order ASC, li.id ASC
                LIMIT 1
            ) AS preview_image
        FROM listings l
        ORDER BY l.created_at DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    db.close()

    return render_template(
        "admin_dashboard.html",
        total_count=total_count,
        active_count=active_count,
        archived_count=archived_count,
        high_risk_count=high_risk_count,
        listings=listings,
        page=page,
        total_pages=total_pages
    )

@app.route(f"/{ADMIN_PATH}/new", methods=["GET", "POST"])
@admin_required
def admin_new():
    if request.method == "POST":
        title = request.form.get("title", "").strip()
        price = request.form.get("price", "").strip()
        description = request.form.get("description", "").strip()
        risk_level = request.form.get("risk_level", "").strip()
        source_url = request.form.get("source_url", "").strip()
        image_urls_raw = request.form.get("image_urls", "").strip()

        year = request.form.get("year", "").strip()
        make = request.form.get("make", "").strip()
        model = request.form.get("model", "").strip()
        trim = request.form.get("trim", "").strip()
        mileage_km = request.form.get("mileage_km", "").strip()
        transmission = request.form.get("transmission", "").strip()
        drivetrain = request.form.get("drivetrain", "").strip()
        fuel_type = request.form.get("fuel_type", "").strip()
        engine = request.form.get("engine", "").strip()
        body_style = request.form.get("body_style", "").strip()
        exterior_color = request.form.get("exterior_color", "").strip()
        interior_color = request.form.get("interior_color", "").strip()
        doors = request.form.get("doors", "").strip()
        seats = request.form.get("seats", "").strip()
        vin = request.form.get("vin", "").strip()
        location = request.form.get("location", "").strip()
        location_search = build_location_search(location)

        contact_phone = request.form.get("contact_phone", "").strip()
        contact_email = request.form.get("contact_email", "").strip()
        contact_facebook = request.form.get("contact_facebook", "").strip()

        condition = request.form.get("condition", "").strip()
        title_status = request.form.get("title_status", "").strip()
        seller_type = request.form.get("seller_type", "").strip()
        seller_status = request.form.get("seller_status", "").strip()
        accident_history = request.form.get("accident_history", "").strip()
        lien_status = request.form.get("lien_status", "").strip()
        rebuilt_status = request.form.get("rebuilt_status", "").strip()
        horsepower = request.form.get("horsepower", "").strip()
        turbo = request.form.get("turbo", "").strip()
        mods = request.form.get("mods", "").strip()
        rust = request.form.get("rust", "").strip()
        issues = request.form.get("issues", "").strip()
        maintenance = request.form.get("maintenance", "").strip()
        tires = request.form.get("tires", "").strip()
        brakes = request.form.get("brakes", "").strip()
        suspension = request.form.get("suspension", "").strip()
        extras = request.form.get("extras", "").strip()
        notes = request.form.get("notes", "").strip()

        if not title or not price or not description or not risk_level:
            return render_template("admin_new.html", error="Please fill in all required fields.")

        try:
            price = int(price)
        except ValueError:
            return render_template("admin_new.html", error="Price must be a number.")

        if risk_level not in ["LOW", "MEDIUM", "HIGH"]:
            return render_template("admin_new.html", error="Risk level must be LOW, MEDIUM, or HIGH.")

        def to_int_or_none(value):
            if not value:
                return None
            try:
                return int(value)
            except ValueError:
                return None

        year = to_int_or_none(year)
        mileage_km = to_int_or_none(mileage_km)
        doors = to_int_or_none(doors)
        seats = to_int_or_none(seats)

        image_urls = [line.strip() for line in image_urls_raw.splitlines() if line.strip()]

        db = get_db()

        cursor = db.execute("""
            INSERT INTO listings (
                title, price, description, risk_level, source_url, status,
                year, make, model, trim, mileage_km, transmission, drivetrain,
                fuel_type, engine, body_style, exterior_color, interior_color,
                doors, seats, vin, location, location_search, contact_phone, contact_email,
                contact_facebook, condition, title_status, seller_type,
                seller_status, accident_history, lien_status,
                rebuilt_status, horsepower, turbo, mods, rust, issues,
                maintenance, tires, brakes, suspension, extras, notes
            )
            VALUES (
                ?, ?, ?, ?, ?, 'active',
                ?, ?, ?, ?, ?, ?, ?,
                ?, ?, ?, ?, ?,
                ?, ?, ?, ?, ?, ?, ?,
                ?, ?, ?, ?,
                ?, ?, ?,
                ?, ?, ?, ?, ?, ?,
                ?, ?, ?, ?, ?, ?
            )
        """, (
            title, price, description, risk_level, source_url,
            year, make, model, trim, mileage_km, transmission, drivetrain,
            fuel_type, engine, body_style, exterior_color, interior_color,
            doors, seats, vin, location, location_search, contact_phone, contact_email,
            contact_facebook, condition, title_status, seller_type,
            seller_status, accident_history, lien_status,
            rebuilt_status, horsepower, turbo, mods, rust, issues,
            maintenance, tires, brakes, suspension, extras, notes
        ))

        listing_id = cursor.lastrowid

        for index, image_url in enumerate(image_urls):
            db.execute("""
                INSERT INTO listing_images (listing_id, image_url, sort_order)
                VALUES (?, ?, ?)
            """, (listing_id, image_url, index))

        db.commit()
        db.close()

        return redirect(url_for("admin_dashboard"))

    return render_template("admin_new.html", error=None)

@app.route(f"/{ADMIN_PATH}/delete/<int:id>", methods=["POST"])
@admin_required
def delete_listing(id):
    db = get_db()
    db.execute("DELETE FROM listings WHERE id = ?", (id,))
    db.commit()
    db.close()

    return redirect(url_for("admin_dashboard"))

@app.route(f"/{ADMIN_PATH}/edit/<int:id>", methods=["GET", "POST"])
@admin_required
def admin_edit(id):
    db = get_db()

    car = db.execute("""
        SELECT *
        FROM listings
        WHERE id = ?
    """, (id,)).fetchone()

    if not car:
        db.close()
        abort(404)

    existing_images = db.execute("""
        SELECT *
        FROM listing_images
        WHERE listing_id = ?
        ORDER BY sort_order ASC, id ASC
    """, (id,)).fetchall()

    if request.method == "POST":
        title = request.form.get("title", "").strip()
        price = request.form.get("price", "").strip()
        description = request.form.get("description", "").strip()
        risk_level = request.form.get("risk_level", "").strip()
        source_url = request.form.get("source_url", "").strip()
        image_urls_raw = request.form.get("image_urls", "").strip()

        year = request.form.get("year", "").strip()
        make = request.form.get("make", "").strip()
        model = request.form.get("model", "").strip()
        trim = request.form.get("trim", "").strip()
        mileage_km = request.form.get("mileage_km", "").strip()
        transmission = request.form.get("transmission", "").strip()
        drivetrain = request.form.get("drivetrain", "").strip()
        fuel_type = request.form.get("fuel_type", "").strip()
        engine = request.form.get("engine", "").strip()
        body_style = request.form.get("body_style", "").strip()
        exterior_color = request.form.get("exterior_color", "").strip()
        interior_color = request.form.get("interior_color", "").strip()
        doors = request.form.get("doors", "").strip()
        seats = request.form.get("seats", "").strip()
        vin = request.form.get("vin", "").strip()
        location = request.form.get("location", "").strip()
        location_search = build_location_search(location)
        condition = request.form.get("condition", "").strip()
        title_status = request.form.get("title_status", "").strip()
        seller_type = request.form.get("seller_type", "").strip()
        seller_status = request.form.get("seller_status", "").strip()
        accident_history = request.form.get("accident_history", "").strip()
        lien_status = request.form.get("lien_status", "").strip()
        rebuilt_status = request.form.get("rebuilt_status", "").strip()
        horsepower = request.form.get("horsepower", "").strip()
        turbo = request.form.get("turbo", "").strip()
        mods = request.form.get("mods", "").strip()
        rust = request.form.get("rust", "").strip()
        issues = request.form.get("issues", "").strip()
        maintenance = request.form.get("maintenance", "").strip()
        tires = request.form.get("tires", "").strip()
        brakes = request.form.get("brakes", "").strip()
        suspension = request.form.get("suspension", "").strip()
        extras = request.form.get("extras", "").strip()
        notes = request.form.get("notes", "").strip()

        contact_phone = request.form.get("contact_phone", "").strip()
        contact_email = request.form.get("contact_email", "").strip()
        contact_facebook = request.form.get("contact_facebook", "").strip()

        if not title or not price or not description or not risk_level:
            db.close()
            return render_template(
                "admin_edit.html",
                car=car,
                existing_images=existing_images,
                image_urls_text="\n".join([img["image_url"] for img in existing_images]),
                error="Please fill in all required fields."
            )

        try:
            price = int(price)
        except ValueError:
            db.close()
            return render_template(
                "admin_edit.html",
                car=car,
                existing_images=existing_images,
                image_urls_text="\n".join([img["image_url"] for img in existing_images]),
                error="Price must be a number."
            )

        def to_int_or_none(value):
            if not value:
                return None
            try:
                return int(value)
            except ValueError:
                return None

        year = to_int_or_none(year)
        mileage_km = to_int_or_none(mileage_km)
        doors = to_int_or_none(doors)
        seats = to_int_or_none(seats)

        image_urls = [line.strip() for line in image_urls_raw.splitlines() if line.strip()]

        db.execute("""
            UPDATE listings
            SET
                title = ?,
                price = ?,
                description = ?,
                risk_level = ?,
                source_url = ?,
                year = ?,
                make = ?,
                model = ?,
                trim = ?,
                mileage_km = ?,
                transmission = ?,
                drivetrain = ?,
                fuel_type = ?,
                engine = ?,
                body_style = ?,
                exterior_color = ?,
                interior_color = ?,
                doors = ?,
                seats = ?,
                vin = ?,
                location = ?,
                location_search = ?,
                condition = ?,
                title_status = ?,
                seller_type = ?,
                seller_status = ?,
                accident_history = ?,
                lien_status = ?,
                rebuilt_status = ?,
                horsepower = ?,
                turbo = ?,
                mods = ?,
                rust = ?,
                issues = ?,
                maintenance = ?,
                tires = ?,
                brakes = ?,
                suspension = ?,
                extras = ?,
                notes = ?,
                contact_phone = ?,
                contact_email = ?,
                contact_facebook = ?
            WHERE id = ?
        """, (
            title, price, description, risk_level, source_url,
            year, make, model, trim, mileage_km, transmission, drivetrain,
            fuel_type, engine, body_style, exterior_color, interior_color,
            doors, seats, vin, location, location_search, condition, title_status,
            seller_type, seller_status, accident_history, lien_status,
            rebuilt_status, horsepower, turbo, mods, rust, issues,
            maintenance, tires, brakes, suspension, extras, notes,
            contact_phone, contact_email, contact_facebook,
            id
        ))

        db.execute("DELETE FROM listing_images WHERE listing_id = ?", (id,))

        for index, image_url in enumerate(image_urls):
            db.execute("""
                INSERT INTO listing_images (listing_id, image_url, sort_order)
                VALUES (?, ?, ?)
            """, (id, image_url, index))

        db.commit()
        db.close()

        return redirect(url_for("admin_dashboard"))

    db.close()

    return render_template(
        "admin_edit.html",
        car=car,
        existing_images=existing_images,
        image_urls_text="\n".join([img["image_url"] for img in existing_images]),
        error=None
    )

@app.route(f"/{ADMIN_PATH}/listings-block")
@admin_required
def admin_dashboardings_block():
    page = request.args.get("page", 1, type=int)
    per_page = 10
    offset = (page - 1) * per_page

    db = get_db()

    total_count = db.execute("SELECT COUNT(*) AS count FROM listings").fetchone()["count"]
    total_pages = max(1, (total_count + per_page - 1) // per_page)

    listings = db.execute("""
        SELECT
            l.*,
            (
                SELECT li.image_url
                FROM listing_images li
                WHERE li.listing_id = l.id
                ORDER BY li.sort_order ASC, li.id ASC
                LIMIT 1
            ) AS preview_image
        FROM listings l
        ORDER BY l.created_at DESC
        LIMIT ? OFFSET ?
    """, (per_page, offset)).fetchall()

    db.close()

    return render_template(
        "partials/admin_listings_table.html",
        listings=listings,
        page=page,
        total_pages=total_pages
    )

@app.route(f"/{ADMIN_PATH}/archive/<int:id>", methods=["POST"])
@admin_required
def archive_listing(id):
    db = get_db()
    db.execute("""
        UPDATE listings
        SET status = 'archived'
        WHERE id = ?
    """, (id,))
    db.commit()
    db.close()

    return redirect(url_for("admin_dashboard"))


@app.route(f"/{ADMIN_PATH}/unarchive/<int:id>", methods=["POST"])
@admin_required
def unarchive_listing(id):
    db = get_db()
    db.execute("""
        UPDATE listings
        SET status = 'active'
        WHERE id = ?
    """, (id,))
    db.commit()
    db.close()

    return redirect(url_for("admin_dashboard"))

@app.route("/terms")
def terms():
    return render_template("terms.html")


@app.route("/privacy")
def privacy():
    return render_template("privacy.html")

@app.context_processor
def inject_admin_path():
    return dict(ADMIN_PATH=ADMIN_PATH)

@app.errorhandler(404)
def not_found(e):
    return render_template("404.html"), 404

init_db()

if __name__ == "__main__":
    app.run(debug=True)