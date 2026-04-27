package service

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"auto-hub/web/internal/config"
	"auto-hub/web/internal/repo"
)

var botSubstrings = []string{
	"bot", "crawl", "spider", "slurp", "baiduspider", "yandex", "scrapy",
}

// StatsService handles view logging.
type StatsService struct {
	repo   *repo.StatsRepo
	config *config.Config
}

// NewStatsService creates a new StatsService.
func NewStatsService(repo *repo.StatsRepo, cfg *config.Config) *StatsService {
	return &StatsService{repo: repo, config: cfg}
}

func isBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, s := range botSubstrings {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func fingerprint(r *http.Request, secret string) string {
	ip := r.RemoteAddr
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		ip = strings.Split(xf, ",")[0]
	}
	ua := r.UserAgent()
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", strings.TrimSpace(ip), ua, secret)))
	return fmt.Sprintf("%x", h)
}

// LogSiteVisit logs a site visit.
func (s *StatsService) LogSiteVisit(r *http.Request) {
	if isBot(r.UserAgent()) {
		return
	}
	_ = s.repo.LogView("site", 0, fingerprint(r, s.config.SecretKey))
}

// LogListingView logs a listing view.
func (s *StatsService) LogListingView(r *http.Request, listingID int) {
	if isBot(r.UserAgent()) {
		return
	}
	_ = s.repo.LogView("listing", listingID, fingerprint(r, s.config.SecretKey))
}
