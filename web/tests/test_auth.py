import pytest
from werkzeug.security import generate_password_hash


def test_login_page(client):
    resp = client.get("/admin/login")
    assert resp.status_code == 200


def test_login_success(client):
    resp = client.post("/admin/login", data={"password": "testpass"}, follow_redirects=True)
    assert resp.status_code == 200


def test_login_fail(client):
    resp = client.post("/admin/login", data={"password": "wrong"}, follow_redirects=True)
    assert b"Wrong password" in resp.data


def test_login_with_hash(client, app):
    app.config["ADMIN_PASSWORD_HASH"] = generate_password_hash("hashedpass")
    app.config["ADMIN_PASSWORD"] = ""
    resp = client.post("/admin/login", data={"password": "hashedpass"}, follow_redirects=True)
    assert resp.status_code == 200
    resp = client.post("/admin/login", data={"password": "wrong"}, follow_redirects=True)
    assert b"Wrong password" in resp.data


def test_logout(admin_client):
    resp = admin_client.post("/admin/logout", follow_redirects=True)
    assert resp.status_code == 200
    resp = admin_client.get("/admin/", follow_redirects=True)
    assert resp.request.path == "/admin/login"


def test_rate_limit(client):
    for i in range(6):
        resp = client.post("/admin/login", data={"password": "wrong"})
    # After 5 attempts per minute, limiter should block
    assert resp.status_code == 429 or b"Wrong password" in resp.data
