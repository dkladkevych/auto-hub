package handler

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"auto-hub/web/internal/config"
	"auto-hub/web/internal/models"
	"auto-hub/web/internal/service"
	"auto-hub/web/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// AdminHandler handles admin panel routes.
type AdminHandler struct {
	cfg            *config.Config
	adminService   *service.AdminService
	listingService *service.ListingService
	store          *sessions.CookieStore
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(cfg *config.Config, adminService *service.AdminService, listingService *service.ListingService, store *sessions.CookieStore) *AdminHandler {
	return &AdminHandler{cfg: cfg, adminService: adminService, listingService: listingService, store: store}
}

func adminBaseData(c *gin.Context) gin.H {
	return gin.H{
		"CSRFToken": c.GetString("csrf_token"),
		"AdminPath": c.GetString("admin_path"),
	}
}

func pageRange(totalPages int) []int {
	var r []int
	for i := 1; i <= totalPages; i++ {
		r = append(r, i)
	}
	return r
}

// LoginPage handles GET /{admin_path}/login.
func (h *AdminHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login", adminBaseData(c))
}

// Login handles POST /{admin_path}/login.
func (h *AdminHandler) Login(c *gin.Context) {
	password := strings.TrimSpace(c.PostForm("password"))
	valid := false

	if h.cfg.AdminPasswordHash != "" {
		valid = password == h.cfg.AdminPassword
	} else if h.cfg.AdminPassword != "" {
		valid = password == h.cfg.AdminPassword
	}

	if valid {
		session, _ := h.store.Get(c.Request, adminSessionName)
		session.Values["authenticated"] = true
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
		return
	}
	data := adminBaseData(c)
	data["Error"] = "Wrong password"
	c.HTML(http.StatusOK, "admin_login", data)
}

// Logout handles POST /{admin_path}/logout.
func (h *AdminHandler) Logout(c *gin.Context) {
	session, _ := h.store.Get(c.Request, adminSessionName)
	delete(session.Values, "authenticated")
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/")
}

// Dashboard handles GET /{admin_path}/.
func (h *AdminHandler) Dashboard(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	data, err := h.adminService.Dashboard(page, 10)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	resp := adminBaseData(c)
	resp["TotalCount"] = data.TotalCount
	resp["DraftCount"] = data.DraftCount
	resp["ActiveCount"] = data.ActiveCount
	resp["ArchivedCount"] = data.ArchivedCount
	resp["DemoCount"] = data.DemoCount
	resp["TotalSiteVisits"] = data.TotalSiteVisits
	resp["TotalListingViews"] = data.TotalListingViews
	resp["Listings"] = data.Listings
	resp["Page"] = data.Page
	resp["TotalPages"] = data.TotalPages
	resp["Views"] = data.Views
	resp["PageRange"] = pageRange(data.TotalPages)
	c.HTML(http.StatusOK, "admin_dashboard", resp)
}

// ListingsBlock handles GET /{admin_path}/listings-block.
func (h *AdminHandler) ListingsBlock(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	listings, views, page, totalPages, err := h.adminService.DashboardListingsBlock(page, 10)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	resp := adminBaseData(c)
	resp["Listings"] = listings
	resp["Page"] = page
	resp["TotalPages"] = totalPages
	resp["Views"] = views
	resp["PageRange"] = pageRange(totalPages)
	c.HTML(http.StatusOK, "admin_listings_table", resp)
}

// NewPage handles GET /{admin_path}/new.
func (h *AdminHandler) NewPage(c *gin.Context) {
	resp := adminBaseData(c)
	resp["FormValues"] = map[string]string{}
	c.HTML(http.StatusOK, "admin_new", resp)
}

// Create handles POST /{admin_path}/new.
func (h *AdminHandler) Create(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "bad form")
		return
	}
	data := service.ParseListingForm(c.Request.PostForm)
	errors := service.ValidateListingForm(data)
	wantsJSON := c.GetHeader("X-Requested-With") == "XMLHttpRequest"

	if len(errors) > 0 {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
			return
		}
		resp := adminBaseData(c)
		resp["Errors"] = errors
		resp["FormValues"] = formValues(c.Request.PostForm)
		c.HTML(http.StatusOK, "admin_new", resp)
		return
	}

	form, _ := c.MultipartForm()
	var files []*multipart.FileHeader
	if form != nil {
		files = form.File["images"]
	}
	if imgErr := utils.ValidateImages(files); imgErr != nil {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": imgErr.Error()})
			return
		}
		resp := adminBaseData(c)
		resp["Error"] = imgErr.Error()
		resp["FormValues"] = formValues(c.Request.PostForm)
		c.HTML(http.StatusOK, "admin_new", resp)
		return
	}

	id, err := h.adminService.CreateListing(data)
	if err != nil {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := adminBaseData(c)
		resp["Error"] = err.Error()
		resp["FormValues"] = formValues(c.Request.PostForm)
		c.HTML(http.StatusOK, "admin_new", resp)
		return
	}

	if len(files) > 0 {
		if err := utils.SyncListingImages(id, nil, files); err != nil {
			h.adminService.DeleteListing(id)
			utils.DeleteListingImages(id)
			if wantsJSON {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := adminBaseData(c)
			resp["Error"] = err.Error()
			resp["FormValues"] = formValues(c.Request.PostForm)
			c.HTML(http.StatusOK, "admin_new", resp)
			return
		}
	}

	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

// EditPage handles GET /{admin_path}/edit/:id.
func (h *AdminHandler) EditPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	car, images, err := h.adminService.GetListingForEdit(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "public_404", BaseData(c))
		return
	}
	resp := adminBaseData(c)
	resp["Car"] = car
	resp["FormValues"] = carToFormValues(car)
	resp["ExistingImagesJSON"] = imagesJSON(images)
	c.HTML(http.StatusOK, "admin_edit", resp)
}

// Update handles POST /{admin_path}/edit/:id.
func (h *AdminHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	car, existingImages, err := h.adminService.GetListingForEdit(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "public_404", BaseData(c))
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "bad form")
		return
	}
	data := service.ParseListingForm(c.Request.PostForm)
	errors := service.ValidateListingForm(data)
	wantsJSON := c.GetHeader("X-Requested-With") == "XMLHttpRequest"

	if len(errors) > 0 {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
			return
		}
		resp := adminBaseData(c)
		resp["Car"] = car
		resp["Errors"] = errors
		resp["FormValues"] = formValues(c.Request.PostForm)
		resp["ExistingImagesJSON"] = imagesJSON(existingImages)
		c.HTML(http.StatusOK, "admin_edit", resp)
		return
	}

	form, _ := c.MultipartForm()
	var files []*multipart.FileHeader
	if form != nil {
		files = form.File["images"]
	}
	if imgErr := utils.ValidateImages(files); imgErr != nil {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": imgErr.Error()})
			return
		}
		resp := adminBaseData(c)
		resp["Car"] = car
		resp["Error"] = imgErr.Error()
		resp["FormValues"] = formValues(c.Request.PostForm)
		resp["ExistingImagesJSON"] = imagesJSON(existingImages)
		c.HTML(http.StatusOK, "admin_edit", resp)
		return
	}

	if err := h.adminService.UpdateListing(id, data); err != nil {
		if wantsJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := adminBaseData(c)
		resp["Car"] = car
		resp["Error"] = err.Error()
		resp["FormValues"] = formValues(c.Request.PostForm)
		resp["ExistingImagesJSON"] = imagesJSON(existingImages)
		c.HTML(http.StatusOK, "admin_edit", resp)
		return
	}

	keep := c.PostFormArray("keep_images")
	hasAny := len(keep) > 0 || len(files) > 0
	if hasAny {
		if err := utils.SyncListingImages(id, keep, files); err != nil {
			if wantsJSON {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := adminBaseData(c)
			resp["Car"] = car
			resp["Error"] = err.Error()
			resp["FormValues"] = formValues(c.Request.PostForm)
			resp["ExistingImagesJSON"] = imagesJSON(existingImages)
			c.HTML(http.StatusOK, "admin_edit", resp)
			return
		}
	} else {
		utils.DeleteListingImages(id)
	}

	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

// Delete handles POST /{admin_path}/delete/:id.
func (h *AdminHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	utils.DeleteListingImages(id)
	h.adminService.DeleteListing(id)
	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

// Archive handles POST /{admin_path}/archive/:id.
func (h *AdminHandler) Archive(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.adminService.SetListingStatus(id, "archived")
	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

// Unarchive handles POST /{admin_path}/unarchive/:id.
func (h *AdminHandler) Unarchive(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.adminService.SetListingStatus(id, "active")
	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

// Publish handles POST /{admin_path}/publish/:id.
func (h *AdminHandler) Publish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.adminService.SetListingStatus(id, "active")
	c.Redirect(http.StatusFound, "/"+h.cfg.AdminPath)
}

const adminSessionName = "admin_session"

func formValues(form map[string][]string) map[string]string {
	out := make(map[string]string)
	for k, v := range form {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func carToFormValues(car *models.Listing) map[string]string {
	out := make(map[string]string)
	if car.Year != nil {
		out["year"] = strconv.Itoa(*car.Year)
	}
	if car.Make != nil {
		out["make"] = *car.Make
	}
	if car.Model != nil {
		out["model"] = *car.Model
	}
	out["price"] = strconv.Itoa(car.Price)
	if car.MileageKm != nil {
		out["mileage_km"] = strconv.Itoa(*car.MileageKm)
	}
	if car.Transmission != nil {
		out["transmission"] = *car.Transmission
	}
	if car.Drivetrain != nil {
		out["drivetrain"] = *car.Drivetrain
	}
	if car.Location != nil {
		out["location"] = *car.Location
	}
	if car.SourceURL != nil {
		out["source_url"] = *car.SourceURL
	}
	out["description"] = car.Description
	if car.Condition != nil {
		out["condition"] = *car.Condition
	}
	if car.Notes != nil {
		out["notes"] = *car.Notes
	}
	return out
}

func imagesJSON(images []string) string {
	b, _ := json.Marshal(images)
	return string(b)
}
