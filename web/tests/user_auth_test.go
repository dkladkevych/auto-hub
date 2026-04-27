package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"auto-hub/web/internal/repo"
	"auto-hub/web/internal/service"
)

func TestRegisterPage(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/register", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Create Account") {
		t.Fatal("expected 'Create Account' in body")
	}
}

func TestRegisterAndVerifyFlow(t *testing.T) {
	r, _, _ := setupTestApp(t)

	// Register
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(url.Values{
		"email":            []string{"test@example.com"},
		"full_name":        []string{"Test User"},
		"password":         []string{"secret123"},
		"confirm_password": []string{"secret123"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Verify with wrong code
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/verify", strings.NewReader(url.Values{
		"email": []string{"test@example.com"},
		"code":  []string{"000000"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Invalid or expired") {
		t.Fatal("expected error for wrong code")
	}
}

func TestLoginUnverified(t *testing.T) {
	r, db, _ := setupTestApp(t)
	authService := service.NewAuthService(repo.NewUserRepo(db))
	_, _, _ = authService.RegisterUser("unverified@example.com", "secret123", "Unverified")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(url.Values{
		"email":    []string{"unverified@example.com"},
		"password": []string{"secret123"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Invalid email or password") {
		t.Fatal("expected login failure for unverified user")
	}
}

func TestLoginAndProfile(t *testing.T) {
	r, db, _ := setupTestApp(t)
	authService := service.NewAuthService(repo.NewUserRepo(db))
	_, code, _ := authService.RegisterUser("verified@example.com", "secret123", "Verified")
	_, _ = authService.VerifyUser("verified@example.com", code)

	// Login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(url.Values{
		"email":    []string{"verified@example.com"},
		"password": []string{"secret123"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
	cookie := extractCookie(w, "user_session")

	// Profile
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/profile", nil)
	req.AddCookie(&http.Cookie{Name: "user_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "verified@example.com") {
		t.Fatal("expected profile page with email")
	}

	// Logout
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "user_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestProfileRequiresLogin(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/profile", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	r, db, _ := setupTestApp(t)
	authService := service.NewAuthService(repo.NewUserRepo(db))
	_, _, _ = authService.RegisterUser("dup@example.com", "secret123", "Dup")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(url.Values{
		"email":            []string{"dup@example.com"},
		"full_name":        []string{"Dup"},
		"password":         []string{"secret123"},
		"confirm_password": []string{"secret123"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "already exists") {
		t.Fatal("expected duplicate email error")
	}
}
