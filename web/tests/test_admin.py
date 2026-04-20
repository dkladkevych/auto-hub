def test_dashboard_requires_login(client):
    resp = client.get("/admin/", follow_redirects=True)
    assert resp.request.path == "/admin/login"


def test_dashboard(admin_client):
    resp = admin_client.get("/admin/")
    assert resp.status_code == 200
    assert b"Admin Dashboard" in resp.data


def test_create_listing(admin_client):
    resp = admin_client.post("/admin/new", data={
        "year": "2021",
        "make": "Mazda",
        "model": "3",
        "price": "18000",
        "mileage_km": "50000",
        "transmission": "Automatic",
        "drivetrain": "FWD",
        "location": "Mississauga",
        "source_url": "https://example.com",
        "description": "Great condition",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)
    assert resp.status_code == 200
    assert b"Mazda" in resp.data or b"Admin Dashboard" in resp.data

    resp = admin_client.get("/")
    assert b"Mazda" in resp.data


def test_edit_listing(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2017",
        "make": "Nissan",
        "model": "Altima",
        "price": "10000",
        "mileage_km": "60000",
        "transmission": "Automatic",
        "drivetrain": "FWD",
        "location": "Toronto",
        "source_url": "https://example.com",
        "description": "Fair",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.post("/admin/edit/1", data={
        "year": "2017",
        "make": "Nissan",
        "model": "Altima",
        "price": "9500",
        "mileage_km": "60000",
        "transmission": "Automatic",
        "drivetrain": "FWD",
        "location": "Toronto",
        "source_url": "https://example.com",
        "description": "Updated",
        "risk_level": "MEDIUM",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)
    assert resp.status_code == 200

    resp = admin_client.get("/listing/1")
    assert b"Updated" in resp.data


def test_archive_and_publish(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2015",
        "make": "BMW",
        "model": "320i",
        "price": "14000",
        "mileage_km": "80000",
        "transmission": "Automatic",
        "drivetrain": "RWD",
        "location": "Vancouver",
        "source_url": "https://example.com",
        "description": "Used",
        "condition": "Fair",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.post("/admin/archive/1", follow_redirects=True)
    assert resp.status_code == 200

    resp = admin_client.get("/listing/1")
    assert resp.status_code == 200
    assert b"archived" in resp.data.lower()

    resp = admin_client.post("/admin/unarchive/1", follow_redirects=True)
    assert resp.status_code == 200

    resp = admin_client.get("/listing/1")
    assert resp.status_code == 200


def test_delete_listing(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2016",
        "make": "Audi",
        "model": "A4",
        "price": "16000",
        "mileage_km": "45000",
        "transmission": "Automatic",
        "drivetrain": "AWD",
        "location": "Calgary",
        "source_url": "https://example.com",
        "description": "Clean",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.post("/admin/delete/1", follow_redirects=True)
    assert resp.status_code == 200

    resp = admin_client.get("/listing/1")
    assert resp.status_code == 404


def test_location_filter(admin_client):
    admin_client.post("/admin/new", data={
        "year": "2022",
        "make": "Tesla",
        "model": "Model 3",
        "price": "35000",
        "mileage_km": "30000",
        "transmission": "Automatic",
        "drivetrain": "RWD",
        "location": "Brampton, ON",
        "source_url": "https://example.com",
        "description": "Electric",
        "condition": "Good",
        "save_mode": "publish",
    }, follow_redirects=True)

    resp = admin_client.get("/?location=Brampton")
    assert resp.status_code == 200
    assert b"Tesla" in resp.data
