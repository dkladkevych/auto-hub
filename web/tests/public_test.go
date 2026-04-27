package tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"auto-hub/web/internal/repo"
	"auto-hub/web/internal/service"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
func itoa(v int) string       { return strconv.Itoa(v) }

func TestHomePage(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Available Cars") {
		t.Fatal("expected 'Available Cars' in body")
	}
}

func TestEmptyHome(t *testing.T) {
	r, _, _ := setupTestApp(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "No listings yet") {
		t.Fatal("expected 'No listings yet' in body")
	}
}

func TestSearchAndFilters(t *testing.T) {
	r, db, _ := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	_, _ = adminService.CreateListing(&service.ListingFormData{
		Title: "Toyota Camry", Price: 5000, Description: "Nice car",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2010), Make: strPtr("Toyota"), Model: strPtr("Camry"),
		MileageKm: intPtr(100000), Location: strPtr("Toronto"),
		Transmission: strPtr("Automatic"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Good"),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?q=toyota", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Toyota") {
		t.Fatal("search failed")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/?price_max=10000", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Camry") {
		t.Fatal("price filter failed")
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/?location=toronto", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Camry") {
		t.Fatal("location filter failed")
	}
}

func TestListingDetail(t *testing.T) {
	r, db, _ := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Honda Civic", Price: 6000, Description: "Clean",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2012), Make: strPtr("Honda"), Model: strPtr("Civic"),
		MileageKm: intPtr(80000), Location: strPtr("Vancouver"),
		Transmission: strPtr("Manual"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Good"),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/listing/"+itoa(id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Honda") {
		t.Fatal("expected 'Honda' in body")
	}
}

func TestSavedPage(t *testing.T) {
	r, db, _ := setupTestApp(t)
	adminService := service.NewAdminService(repo.NewListingRepo(db), repo.NewStatsRepo(db))
	id, _ := adminService.CreateListing(&service.ListingFormData{
		Title: "Ford Focus", Price: 4000, Description: "Reliable",
		SourceURL: "http://example.com", Status: "active",
		Year: intPtr(2015), Make: strPtr("Ford"), Model: strPtr("Focus"),
		MileageKm: intPtr(70000), Location: strPtr("Montreal"),
		Transmission: strPtr("Automatic"), Drivetrain: strPtr("FWD"),
		Condition: strPtr("Fair"),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/saved?ids="+itoa(id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Ford") {
		t.Fatal("expected 'Ford' in body")
	}
}
