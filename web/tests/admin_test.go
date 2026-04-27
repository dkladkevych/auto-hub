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

func TestDashboardRequiresLogin(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestDashboard(t *testing.T) {
	r, _, store := setupTestApp(t)
	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Admin Dashboard") {
		t.Fatal("expected 'Admin Dashboard' in body")
	}
}

func TestCreateListing(t *testing.T) {
	r, _, store := setupTestApp(t)
	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/new", strings.NewReader(url.Values{
		"year":         []string{"2010"},
		"make":         []string{"Toyota"},
		"model":        []string{"Corolla"},
		"price":        []string{"5000"},
		"mileage_km":   []string{"100000"},
		"transmission": []string{"Automatic"},
		"drivetrain":   []string{"FWD"},
		"location":     []string{"Brampton, ON"},
		"source_url":   []string{"http://example.com"},
		"description":  []string{"Test description"},
		"condition":    []string{"Good"},
		"save_mode":    []string{"publish"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Verify on home page
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "Toyota") {
		t.Fatal("expected new listing on home page")
	}
}

func TestEditListing(t *testing.T) {
	r, db, store := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Old Title", Price: 3000, Description: "Old",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2005), Make: strPtr("Honda"), Model: strPtr("Civic"),
		MileageKm: intPtr(150000), Location: strPtr("Toronto"),
		Transmission: strPtr("Manual"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Fair"),
	})

	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/edit/"+itoa(id), strings.NewReader(url.Values{
		"year":         []string{"2005"},
		"make":         []string{"Honda"},
		"model":        []string{"Civic"},
		"price":        []string{"4000"},
		"mileage_km":   []string{"150000"},
		"transmission": []string{"Manual"},
		"drivetrain":   []string{"FWD"},
		"location":     []string{"Toronto"},
		"source_url":   []string{"http://example.com"},
		"description":  []string{"Updated description"},
		"condition":    []string{"Fair"},
		"save_mode":    []string{"publish"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Verify updated
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/listing/"+itoa(id), nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "Updated description") {
		t.Fatal("expected updated description")
	}
}

func TestArchiveAndPublish(t *testing.T) {
	r, db, store := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Test", Price: 3000, Description: "Test",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2005), Make: strPtr("Honda"), Model: strPtr("Civic"),
		MileageKm: intPtr(150000), Location: strPtr("Toronto"),
		Transmission: strPtr("Manual"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Fair"),
	})

	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	// Archive
	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/archive/"+itoa(id), nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("archive: expected redirect, got %d", w.Code)
	}

	// Check listing page shows archived
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/listing/"+itoa(id), nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "archived") {
		t.Fatal("expected archived banner")
	}

	// Unarchive
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/admin/unarchive/"+itoa(id), nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("unarchive: expected redirect, got %d", w.Code)
	}
}

func TestDeleteListing(t *testing.T) {
	r, db, store := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Delete Me", Price: 3000, Description: "Test",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2005), Make: strPtr("Honda"), Model: strPtr("Civic"),
		MileageKm: intPtr(150000), Location: strPtr("Toronto"),
		Transmission: strPtr("Manual"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Fair"),
	})

	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/delete/"+itoa(id), nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: cookie})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Verify 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/listing/"+itoa(id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLocationFilter(t *testing.T) {
	r, db, store := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Brampton Car", Price: 5000, Description: "Nice",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2010), Make: strPtr("Toyota"), Model: strPtr("Corolla"),
		MileageKm: intPtr(100000), Location: strPtr("Brampton, ON"),
		Transmission: strPtr("Automatic"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Good"),
	})
	_ = id

	w := createAdminSession(t, r, store)
	cookie := extractCookie(w, "admin_session")
	_ = cookie

	w = httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?location=Brampton", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Brampton Car") {
		t.Fatal("location filter failed")
	}
}
