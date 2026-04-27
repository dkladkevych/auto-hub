package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"auto-hub/web/internal/repo"
	"auto-hub/web/internal/service"
	"auto-hub/web/internal/utils"
	"github.com/gin-gonic/gin"
)

// PublicHandler handles public routes.
type PublicHandler struct {
	listingService *service.ListingService
	statsService   *service.StatsService
}

// NewPublicHandler creates a new PublicHandler.
func NewPublicHandler(listingService *service.ListingService, statsService *service.StatsService) *PublicHandler {
	return &PublicHandler{listingService: listingService, statsService: statsService}
}

func parseFilters(c *gin.Context) repo.FilterParams {
	return repo.FilterParams{
		Q:            strings.TrimSpace(c.Query("q")),
		PriceMin:     utilsToInt(c.Query("price_min")),
		PriceMax:     utilsToInt(c.Query("price_max")),
		YearMin:      utilsToInt(c.Query("year_min")),
		YearMax:      utilsToInt(c.Query("year_max")),
		MileageMin:   utilsToInt(c.Query("mileage_min")),
		MileageMax:   utilsToInt(c.Query("mileage_max")),
		Transmission: strings.TrimSpace(c.Query("transmission")),
		Drivetrain:   strings.TrimSpace(c.Query("drivetrain")),
		Location:     strings.TrimSpace(c.Query("location")),
	}
}

func utilsToInt(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

func buildQueryString(f repo.FilterParams) string {
	var parts []string
	if f.Q != "" {
		parts = append(parts, "q="+f.Q)
	}
	if f.Location != "" {
		parts = append(parts, "location="+f.Location)
	}
	if f.PriceMin != nil {
		parts = append(parts, fmt.Sprintf("price_min=%d", *f.PriceMin))
	}
	if f.PriceMax != nil {
		parts = append(parts, fmt.Sprintf("price_max=%d", *f.PriceMax))
	}
	if f.YearMin != nil {
		parts = append(parts, fmt.Sprintf("year_min=%d", *f.YearMin))
	}
	if f.YearMax != nil {
		parts = append(parts, fmt.Sprintf("year_max=%d", *f.YearMax))
	}
	if f.MileageMin != nil {
		parts = append(parts, fmt.Sprintf("mileage_min=%d", *f.MileageMin))
	}
	if f.MileageMax != nil {
		parts = append(parts, fmt.Sprintf("mileage_max=%d", *f.MileageMax))
	}
	if f.Transmission != "" {
		parts = append(parts, "transmission="+f.Transmission)
	}
	if f.Drivetrain != "" {
		parts = append(parts, "drivetrain="+f.Drivetrain)
	}
	return strings.Join(parts, "&")
}

func BaseData(c *gin.Context) gin.H {
	currentUser, _ := c.Get("current_user")
	data := gin.H{
		"CSRFToken":   c.GetString("csrf_token"),
		"CurrentUser": currentUser,
		"AdminPath":   c.GetString("admin_path"),
		"RequestURL":  c.GetString("request_url"),
	}
	if flashes, ok := c.Get("flashes"); ok {
		data["Flashes"] = flashes
	}
	return data
}

// Home handles the home page.
func (h *PublicHandler) Home(c *gin.Context) {
	h.statsService.LogSiteVisit(c.Request)
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	filters := parseFilters(c)
	listings, hasFilter, curPage, totalPages, err := h.listingService.HomeListings(filters, page, 12)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	data := BaseData(c)
	data["Listings"] = listings
	data["Filters"] = filters
	data["HasAnyFilter"] = hasFilter
	data["Page"] = curPage
	data["TotalPages"] = totalPages
	data["QueryString"] = buildQueryString(filters)
	data["PageRange"] = pageRange(totalPages)
	c.HTML(http.StatusOK, "public_home", data)
}

// Listing handles the listing detail page.
func (h *PublicHandler) Listing(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	car, images, thumbs, err := h.listingService.ListingPageData(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "public_404", BaseData(c))
		return
	}
	if car.Status == "active" || car.Status == "demo" {
		h.statsService.LogListingView(c.Request, id)
	}

	var imageDicts []gin.H
	var imageURLs []string
	for _, u := range images {
		imageDicts = append(imageDicts, gin.H{"ImageURL": u})
		imageURLs = append(imageURLs, u)
	}
	var thumbDicts []gin.H
	for _, u := range thumbs {
		thumbDicts = append(thumbDicts, gin.H{"ImageURL": u})
	}

	firstURL := "/static/images/empty.png"
	firstIsVideo := false
	if len(images) > 0 {
		firstURL = images[0]
		firstIsVideo = strings.HasSuffix(strings.ToLower(firstURL), ".mp4")
	}

	data := BaseData(c)
	data["Car"] = car
	data["Images"] = imageDicts
	data["Thumbs"] = thumbDicts
	data["ImageURLs"] = imageURLs
	data["FirstImageURL"] = firstURL
	data["FirstIsVideo"] = firstIsVideo
	data["HasMedia"] = len(images) > 0 && images[0] != "/static/images/empty.png"
	data["DescriptionHTML"] = template.HTML(utils.RenderMarkdown(car.Description))
	data["ImagesJSON"] = template.JS(imageURLsJSON(imageURLs))
	c.HTML(http.StatusOK, "public_listing", data)
}

func imageURLsJSON(urls []string) string {
	b, _ := json.Marshal(urls)
	return string(b)
}

// Saved handles the saved listings page.
func (h *PublicHandler) Saved(c *gin.Context) {
	idsParam := strings.TrimSpace(c.Query("ids"))
	var ids []int
	if idsParam != "" {
		for _, part := range strings.Split(idsParam, ",") {
			part = strings.TrimSpace(part)
			if id, err := strconv.Atoi(part); err == nil {
				ids = append(ids, id)
			}
		}
	}
	listings, err := h.listingService.SavedListings(ids)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	data := BaseData(c)
	data["Listings"] = listings
	c.HTML(http.StatusOK, "public_saved", data)
}

func imageDicts(urls []string) []gin.H {
	var out []gin.H
	for _, u := range urls {
		out = append(out, gin.H{"image_url": u})
	}
	return out
}
