package tests

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"auto-hub/web/internal/config"
	"auto-hub/web/internal/db"
	"auto-hub/web/internal/handler"
	"auto-hub/web/internal/middleware"
	"auto-hub/web/internal/repo"
	"auto-hub/web/internal/service"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"titleCase": func(s string) string { return strings.ToTitle(strings.ReplaceAll(s, "_", " ")) },
		"truncate": func(s string, n int) string {
			if len(s) <= n { return s }
			return s[:n] + "..."
		},
		"formatDate": func(t interface{}, layout string) string {
			return ""
		},
		"toJSON": func(v interface{}) template.JS { return template.JS("[]") },
		"hasSuffix": func(s, suffix string) bool { return strings.HasSuffix(s, suffix) },
	}
}

func createTestRenderer() multitemplate.Renderer {
	funcs := templateFuncs()
	r := multitemplate.NewRenderer()
	r.AddFromFilesFuncs("public_home", funcs, "../templates/public/base.html", "../templates/public/home.html")
	r.AddFromFilesFuncs("public_listing", funcs, "../templates/public/base.html", "../templates/public/listing.html")
	r.AddFromFilesFuncs("public_saved", funcs, "../templates/public/base.html", "../templates/public/saved.html")
	r.AddFromFilesFuncs("public_terms", funcs, "../templates/public/base.html", "../templates/public/terms.html")
	r.AddFromFilesFuncs("public_privacy", funcs, "../templates/public/base.html", "../templates/public/privacy.html")
	r.AddFromFilesFuncs("public_404", funcs, "../templates/public/base.html", "../templates/public/404.html")
	r.AddFromFilesFuncs("auth_login", funcs, "../templates/public/base.html", "../templates/auth/login.html")
	r.AddFromFilesFuncs("auth_register", funcs, "../templates/public/base.html", "../templates/auth/register.html")
	r.AddFromFilesFuncs("auth_verify", funcs, "../templates/public/base.html", "../templates/auth/verify.html")
	r.AddFromFilesFuncs("auth_profile", funcs, "../templates/public/base.html", "../templates/auth/profile.html")
	r.AddFromFilesFuncs("admin_login", funcs, "../templates/admin/base.html", "../templates/admin/login.html")
	r.AddFromFilesFuncs("admin_dashboard", funcs, "../templates/admin/base.html", "../templates/admin/dashboard.html", "../templates/admin/partials/listings_table.html")
	r.AddFromFilesFuncs("admin_new", funcs, "../templates/admin/base.html", "../templates/admin/new.html")
	r.AddFromFilesFuncs("admin_edit", funcs, "../templates/admin/base.html", "../templates/admin/edit.html")
	r.AddFromFilesFuncs("admin_listings_table", funcs, "../templates/admin/partials/listings_table.html")
	r.AddFromFilesFuncs("sitemap", funcs, "../templates/public/sitemap.xml")
	return r
}

func setupTestApp(t *testing.T) (*gin.Engine, *sql.DB, *sessions.CookieStore) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SecretKey:        "test-secret-key",
		AdminPassword:    "testpass",
		AdminPath:        "admin",
		MailServiceURL:   "http://127.0.0.1:8085",
		InternalAPIToken: "test-token",
		DBPath:           t.TempDir() + "/db.sqlite",
		DataDir:          t.TempDir(),
		ListingsDir:      t.TempDir() + "/listings",
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitSchema(database); err != nil {
		t.Fatal(err)
	}

	listingRepo := repo.NewListingRepo(database)
	userRepo := repo.NewUserRepo(database)
	statsRepo := repo.NewStatsRepo(database)

	listingService := service.NewListingService(listingRepo)
	adminService := service.NewAdminService(listingRepo, statsRepo)
	authService := service.NewAuthService(userRepo)
	statsService := service.NewStatsService(statsRepo, cfg)
	emailService := service.NewEmailService(cfg)

	store := sessions.NewCookieStore([]byte(cfg.SecretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	publicHandler := handler.NewPublicHandler(listingService, statsService)
	pagesHandler := handler.NewPagesHandler(cfg, listingService)
	authHandler := handler.NewAuthHandler(authService, emailService, store)
	adminHandler := handler.NewAdminHandler(cfg, adminService, listingService, store)

	r := gin.New()
	r.Use(gin.Recovery())
	r.HTMLRender = createTestRenderer()

	// Flash middleware
	r.Use(middleware.FlashMiddleware(store))
	r.Use(middleware.InjectUser(store, func(id int) interface{} {
		u, _ := authService.GetUserByID(id)
		return u
	}))
	r.Use(middleware.AdminPathMiddleware(cfg))
	r.Use(func(c *gin.Context) {
		c.Set("request_url", c.Request.URL.String())
		c.Next()
	})

	// No CSRF in tests
	r.Static("/static", "../static")

	// Data files
	r.GET("/data/*filepath", func(c *gin.Context) {
		c.FileFromFS("data/"+c.Param("filepath"), http.Dir("."))
	})

	// Public routes
	r.GET("/", publicHandler.Home)
	r.GET("/listing/:id", publicHandler.Listing)
	r.GET("/saved", publicHandler.Saved)
	r.GET("/terms", pagesHandler.Terms)
	r.GET("/privacy", pagesHandler.Privacy)
	r.GET("/robots.txt", pagesHandler.Robots)
	r.GET("/sitemap.xml", pagesHandler.Sitemap)

	// Auth routes
	r.GET("/register", authHandler.RegisterPage)
	r.POST("/register", authHandler.Register)
	r.GET("/verify", authHandler.VerifyPage)
	r.POST("/verify", authHandler.Verify)
	r.GET("/login", authHandler.LoginPage)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)
	r.GET("/profile", middleware.LoginRequired(store), authHandler.Profile)

	// Admin routes
	adminPath := cfg.AdminPath
	admin := r.Group("/" + adminPath)
	{
		admin.GET("/login", adminHandler.LoginPage)
		admin.POST("/login", adminHandler.Login)
		admin.POST("/logout", middleware.AdminRequired(cfg, store), adminHandler.Logout)
		admin.GET("/", middleware.AdminRequired(cfg, store), adminHandler.Dashboard)
		admin.GET("/listings-block", middleware.AdminRequired(cfg, store), adminHandler.ListingsBlock)
		admin.GET("/new", middleware.AdminRequired(cfg, store), adminHandler.NewPage)
		admin.POST("/new", middleware.AdminRequired(cfg, store), adminHandler.Create)
		admin.GET("/edit/:id", middleware.AdminRequired(cfg, store), adminHandler.EditPage)
		admin.POST("/edit/:id", middleware.AdminRequired(cfg, store), adminHandler.Update)
		admin.POST("/delete/:id", middleware.AdminRequired(cfg, store), adminHandler.Delete)
		admin.POST("/archive/:id", middleware.AdminRequired(cfg, store), adminHandler.Archive)
		admin.POST("/unarchive/:id", middleware.AdminRequired(cfg, store), adminHandler.Unarchive)
		admin.POST("/publish/:id", middleware.AdminRequired(cfg, store), adminHandler.Publish)
	}

	// NoRoute
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "404")
	})

	return r, database, store
}

func createAdminSession(t *testing.T, r *gin.Engine, store *sessions.CookieStore) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/login", strings.NewReader(url.Values{
		"password": []string{"testpass"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("admin login failed: %d", w.Code)
	}
	return w
}

func extractCookie(w *httptest.ResponseRecorder, name string) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
