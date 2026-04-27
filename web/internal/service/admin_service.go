package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"auto-hub/web/internal/models"
	"auto-hub/web/internal/repo"
)

// AdminService handles admin panel business logic.
type AdminService struct {
	listingRepo *repo.ListingRepo
	statsRepo   *repo.StatsRepo
}

// NewAdminService creates a new AdminService.
func NewAdminService(listingRepo *repo.ListingRepo, statsRepo *repo.StatsRepo) *AdminService {
	return &AdminService{listingRepo: listingRepo, statsRepo: statsRepo}
}

// DashboardData holds data for the admin dashboard.
type DashboardData struct {
	TotalCount        int
	DraftCount        int
	ActiveCount       int
	ArchivedCount     int
	DemoCount         int
	TotalSiteVisits   int
	TotalListingViews int
	Listings          []*EnrichedListing
	Page              int
	TotalPages        int
	Views             map[int]int
}

// Dashboard returns stats and paginated listings.
func (s *AdminService) Dashboard(page, perPage int) (*DashboardData, error) {
	total, draft, active, archived, demo, err := s.listingRepo.CountByStatus()
	if err != nil {
		return nil, err
	}
	siteVisits, _ := s.statsRepo.GetSiteVisits()
	listingViews, _ := s.statsRepo.GetTotalListingViews()

	listings, views, err := s.listingRepo.GetDashboardListings(page, perPage)
	if err != nil {
		return nil, err
	}

	totalPages := max(1, (total+perPage-1)/perPage)
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	var enriched []*EnrichedListing
	for _, l := range listings {
		enriched = append(enriched, enrich(l))
	}

	return &DashboardData{
		TotalCount:        total,
		DraftCount:        draft,
		ActiveCount:       active,
		ArchivedCount:     archived,
		DemoCount:         demo,
		TotalSiteVisits:   siteVisits,
		TotalListingViews: listingViews,
		Listings:          enriched,
		Page:              page,
		TotalPages:        totalPages,
		Views:             views,
	}, nil
}

// DashboardListingsBlock returns just listings for AJAX pagination.
func (s *AdminService) DashboardListingsBlock(page, perPage int) ([]*EnrichedListing, map[int]int, int, int, error) {
	listings, views, err := s.listingRepo.GetDashboardListings(page, perPage)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	total, _, _, _, _, _ := s.listingRepo.CountByStatus()
	totalPages := max(1, (total+perPage-1)/perPage)
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	var enriched []*EnrichedListing
	for _, l := range listings {
		enriched = append(enriched, enrich(l))
	}
	return enriched, views, page, totalPages, nil
}

// ListingFormData holds parsed form data for a listing.
type ListingFormData struct {
	Title        string
	Price        int
	Description  string
	SourceURL    string
	Status       string
	Year         *int
	Make         *string
	Model        *string
	MileageKm    *int
	Location     *string
	Condition    *string
	Notes        *string
	Transmission *string
	Drivetrain   *string
}

// ParseListingForm extracts and normalizes form values.
func ParseListingForm(form map[string][]string) *ListingFormData {
	get := func(key string) string {
		if v, ok := form[key]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	getInt := func(key string) *int {
		v := get(key)
		if v == "" {
			return nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil
		}
		return &n
	}
	getPtr := func(key string) *string {
		v := get(key)
		if v == "" {
			return nil
		}
		return &v
	}

	d := &ListingFormData{
		Title:       get("title"),
		Description: get("description"),
		SourceURL:   get("source_url"),
		Status:      get("save_mode"),
		Year:        getInt("year"),
		Make:        getPtr("make"),
		Model:       getPtr("model"),
		MileageKm:   getInt("mileage_km"),
		Location:    getPtr("location"),
		Condition:   getPtr("condition"),
		Notes:       getPtr("notes"),
		Transmission: getPtr("transmission"),
		Drivetrain:  getPtr("drivetrain"),
	}

	switch d.Status {
	case "publish":
		d.Status = "active"
	case "":
		d.Status = "draft"
	}
	if d.Make != nil && d.Model != nil && d.Year != nil {
		d.Title = fmt.Sprintf("%d %s %s", *d.Year, *d.Make, *d.Model)
	}
	if d.Title == "" {
		d.Title = "Untitled"
	}

	priceStr := get("price")
	if priceStr == "" {
		if d.Status == "draft" {
			d.Price = 0
		}
	} else {
		d.Price, _ = strconv.Atoi(priceStr)
	}

	// Normalize choices
	if d.Condition != nil {
		*d.Condition = normalizeChoice(*d.Condition, []string{"Good", "Fair", "Poor"})
	}
	if d.Transmission != nil {
		*d.Transmission = normalizeChoice(*d.Transmission, []string{"Automatic", "Manual", "CVT", "Unknown"})
	}
	if d.Drivetrain != nil {
		*d.Drivetrain = normalizeChoice(*d.Drivetrain, []string{"FWD", "RWD", "AWD", "4WD", "Unknown"})
	}

	return d
}

func normalizeChoice(val string, options []string) string {
	for _, o := range options {
		if strings.EqualFold(val, o) {
			return o
		}
	}
	return options[len(options)-1]
}

// ValidateListingForm validates form data.
func ValidateListingForm(d *ListingFormData) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(d.Title) == "" {
		errors["title"] = "Title is required"
	}
	if d.Status == "draft" {
		return errors
	}
	if d.Price <= 0 {
		errors["price"] = "Price is required"
	}
	if d.MileageKm == nil || *d.MileageKm < 0 {
		errors["mileage_km"] = "Mileage is required"
	}
	if d.Transmission == nil || *d.Transmission == "" {
		errors["transmission"] = "Transmission is required"
	}
	if d.Drivetrain == nil || *d.Drivetrain == "" {
		errors["drivetrain"] = "Drivetrain is required"
	}
	if d.Location == nil || *d.Location == "" {
		errors["location"] = "Location is required"
	}
	if d.SourceURL == "" {
		errors["source_url"] = "Source URL is required"
	}
	if strings.TrimSpace(d.Description) == "" {
		errors["description"] = "Description is required"
	}
	if d.Condition == nil || *d.Condition == "" {
		errors["condition"] = "Condition is required"
	}
	return errors
}

// CreateListing creates a new listing from form data.
func (s *AdminService) CreateListing(d *ListingFormData) (int, error) {
	l := &models.Listing{
		AccountID:    0,
		Title:        d.Title,
		Price:        d.Price,
		Description:  d.Description,
		SourceURL:    &d.SourceURL,
		Status:       d.Status,
		Year:         d.Year,
		Make:         d.Make,
		Model:        d.Model,
		MileageKm:    d.MileageKm,
		Location:     d.Location,
		Condition:    d.Condition,
		Notes:        d.Notes,
		Transmission: d.Transmission,
		Drivetrain:   d.Drivetrain,
	}
	if d.Status == "active" || d.Status == "demo" {
		now := time.Now().UTC()
		l.PublishedAt = &now
	}
	return s.listingRepo.Create(l)
}

// GetListingForEdit fetches a listing and its image URLs.
func (s *AdminService) GetListingForEdit(id int) (*models.Listing, []string, error) {
	l, err := s.listingRepo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	images := GetMediaURLs(id, false)
	return l, images, nil
}

// UpdateListing updates an existing listing.
func (s *AdminService) UpdateListing(id int, d *ListingFormData) error {
	l, err := s.listingRepo.GetByID(id)
	if err != nil {
		return err
	}
	l.Title = d.Title
	l.Price = d.Price
	l.Description = d.Description
	l.SourceURL = &d.SourceURL
	l.Status = d.Status
	l.Year = d.Year
	l.Make = d.Make
	l.Model = d.Model
	l.MileageKm = d.MileageKm
	l.Location = d.Location
	l.Condition = d.Condition
	l.Notes = d.Notes
	l.Transmission = d.Transmission
	l.Drivetrain = d.Drivetrain
	if (d.Status == "active" || d.Status == "demo") && l.PublishedAt == nil {
		now := time.Now().UTC()
		l.PublishedAt = &now
	}
	return s.listingRepo.Update(l)
}

// DeleteListing removes a listing.
func (s *AdminService) DeleteListing(id int) error {
	return s.listingRepo.Delete(id)
}

// SetListingStatus changes a listing's status.
func (s *AdminService) SetListingStatus(id int, status string) error {
	return s.listingRepo.SetStatus(id, status)
}
