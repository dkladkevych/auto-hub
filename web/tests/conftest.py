import pytest

from app import create_app
from app.config import Config


class TestConfig(Config):
    TESTING = True
    SECRET_KEY = "test-secret-key"
    ADMIN_PASSWORD_HASH = ""
    ADMIN_PASSWORD = "testpass"
    ADMIN_PATH = "admin"
    WTF_CSRF_ENABLED = False


@pytest.fixture
def app(tmp_path):
    db_path = tmp_path / "test.db"
    TestConfig.DB_PATH = str(db_path)
    app = create_app(TestConfig)
    with app.app_context():
        yield app


@pytest.fixture
def client(app):
    return app.test_client()


@pytest.fixture
def admin_client(client):
    client.post("/admin/login", data={"password": "testpass"}, follow_redirects=True)
    return client
