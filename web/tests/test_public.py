def test_home_page(client):
    resp = client.get("/")
    assert resp.status_code == 200
    assert b"Available Cars" in resp.data


def test_empty_home(client):
    resp = client.get("/")
    assert b"No listings yet" in resp.data


def test_search_and_filters(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2020",
        "make": "Toyota",
        "model": "Camry",
        "price": "15000",
        "mileage_km": "40000",
        "transmission": "Automatic",
        "drivetrain": "FWD",
        "location": "Toronto, ON",
        "source_url": "https://example.com",
        "description": "Nice car",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.get("/?q=toyota")
    assert resp.status_code == 200
    assert b"Toyota" in resp.data

    resp = admin_client.get("/?price_max=10000")
    assert b"No listings" in resp.data or b"Nothing found" in resp.data

    resp = admin_client.get("/?location=toronto")
    assert b"Toyota" in resp.data


def test_listing_detail(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2019",
        "make": "Honda",
        "model": "Civic",
        "price": "12000",
        "mileage_km": "35000",
        "transmission": "Manual",
        "drivetrain": "FWD",
        "location": "Ottawa",
        "source_url": "https://example.com",
        "description": "Clean",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.get("/listing/1")
    assert resp.status_code == 200
    assert b"Honda" in resp.data


def test_saved_page(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2018",
        "make": "Ford",
        "model": "Focus",
        "price": "8000",
        "mileage_km": "90000",
        "transmission": "Automatic",
        "drivetrain": "FWD",
        "location": "Montreal",
        "source_url": "https://example.com",
        "description": "Ok",
        "condition": "Fair",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.get("/saved?ids=1")
    assert resp.status_code == 200
    assert b"Ford" in resp.data
