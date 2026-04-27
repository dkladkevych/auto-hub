package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAdminLoginPage(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAdminLoginSuccess(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/login", strings.NewReader(url.Values{
		"password": []string{"testpass"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestAdminLoginFail(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/login", strings.NewReader(url.Values{
		"password": []string{"wrong"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Wrong password") {
		t.Fatal("expected 'Wrong password' in body")
	}
}

func TestAdminLogout(t *testing.T) {
	r, _, store := setupTestApp(t)
	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/logout", nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Verify dashboard redirects to login
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect after logout, got %d", w.Code)
	}
}
