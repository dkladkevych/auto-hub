package service

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"auto-hub/web/internal/models"
	"auto-hub/web/internal/repo"
	"database/sql"
)

// ListingService handles public listing catalog logic.
type ListingService struct {
	repo *repo.ListingRepo
}

// NewListingService creates a new ListingService.
func NewListingService(repo *repo.ListingRepo) *ListingService {
	return &ListingService{repo: repo}
}

// EnrichedListing adds computed fields for templates.
type EnrichedListing struct {
	*models.Listing
	PreviewImage string
	HasVideo     bool
	HasMedia     bool
	TimeSince    string
}

// TimeSince returns human-readable elapsed time.
func TimeSince(t *time.Time) string {
	if t == nil {
		return ""
	}
	delta := time.Since(*t)
	if delta < 0 {
		delta = -delta
	}
	days := int(delta.Hours() / 24)
	if days >= 365 {
		y := days / 365
		return fmt.Sprintf("%d Year%s", y, plural(y))
	}
	if days >= 30 {
		m := days / 30
		return fmt.Sprintf("%d Month%s", m, plural(m))
	}
	if days > 0 {
		return fmt.Sprintf("%d Day%s", days, plural(days))
	}
	hours := int(delta.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d Hour%s", hours, plural(hours))
	}
	mins := int(delta.Minutes())
	if mins > 0 {
		return fmt.Sprintf("%d Minute%s", mins, plural(mins))
	}
	return "Just now"
}

func plural(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

func listingDir(id int) string {
	return fmt.Sprintf("data/listings/%d", id)
}

// GetPreviewImage returns the first image URL for a listing.
func GetPreviewImage(id int, thumb bool) string {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "/static/images/empty.png"
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "thumb_") {
			if thumb {
				files = append(files, name)
			}
			continue
		}
		if thumb {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return "/static/images/empty.png"
	}
	return fmt.Sprintf("/data/listings/%d/%s", id, files[0])
}

// ListingHasVideo checks if a listing directory contains an mp4.
func ListingHasVideo(id int) bool {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mp4") {
			return true
		}
	}
	return false
}

// GetMediaURLs returns image/video URLs for a listing.
func GetMediaURLs(id int, thumb bool) []string {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "thumb_") {
			if thumb {
				files = append(files, fmt.Sprintf("/data/listings/%d/%s", id, name))
			}
			continue
		}
		if thumb {
			continue
		}
		files = append(files, fmt.Sprintf("/data/listings/%d/%s", id, name))
	}
	sort.Strings(files)
	return files
}

func enrich(l *models.Listing) *EnrichedListing {
	images := GetMediaURLs(l.ID, false)
	hasMedia := len(images) > 0 && images[0] != "/static/images/empty.png"
	return &EnrichedListing{
		Listing:      l,
		PreviewImage: GetPreviewImage(l.ID, true),
		HasVideo:     ListingHasVideo(l.ID),
		HasMedia:     hasMedia,
		TimeSince:    TimeSince(l.PublishedAt),
	}
}

// HomeListings returns filtered, paginated listings for the home page.
func (s *ListingService) HomeListings(filters repo.FilterParams, page, perPage int) ([]*EnrichedListing, bool, int, int, error) {
	hasFilter := filters.Q != "" || filters.Location != "" ||
		filters.PriceMin != nil || filters.PriceMax != nil ||
		filters.YearMin != nil || filters.YearMax != nil ||
		filters.MileageMin != nil || filters.MileageMax != nil ||
		filters.Transmission != "" || filters.Drivetrain != ""

	total, err := s.repo.CountFiltered(filters)
	if err != nil {
		return nil, false, 0, 0, err
	}

	totalPages := max(1, (total+perPage-1)/perPage)
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	raw, err := s.repo.GetFiltered(filters, page, perPage)
	if err != nil {
		return nil, false, 0, 0, err
	}

	var enriched []*EnrichedListing
	for _, l := range raw {
		enriched = append(enriched, enrich(l))
	}
	return enriched, hasFilter, page, totalPages, nil
}

// SavedListings returns listings by IDs.
func (s *ListingService) SavedListings(ids []int) ([]*EnrichedListing, error) {
	raw, err := s.repo.GetByIDs(ids)
	if err != nil {
		return nil, err
	}
	var enriched []*EnrichedListing
	for _, l := range raw {
		enriched = append(enriched, enrich(l))
	}
	return enriched, nil
}

// ListingPageData returns a listing with its media.
func (s *ListingService) ListingPageData(id int) (*EnrichedListing, []string, []string, error) {
	l, err := s.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil, fmt.Errorf("not found")
		}
		return nil, nil, nil, err
	}
	if l.Status == "draft" {
		return nil, nil, nil, fmt.Errorf("not found")
	}

	images := GetMediaURLs(id, false)
	thumbs := GetMediaURLs(id, true)
	if len(images) == 0 {
		images = []string{"/static/images/empty.png"}
	}
	if len(thumbs) == 0 {
		thumbs = []string{"/static/images/empty.png"}
	}
	return enrich(l), images, thumbs, nil
}
