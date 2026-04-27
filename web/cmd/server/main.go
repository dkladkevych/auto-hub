package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

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
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"formatDate": func(t *time.Time, layout string) string {
			if t == nil {
				return ""
			}
			return t.Format(layout)
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"toJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"titleCase": func(s string) string {
			return strings.ToTitle(strings.ReplaceAll(s, "_", " "))
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
	}
}

func createRenderer() multitemplate.Renderer {
	funcs := templateFuncs()
	r := multitemplate.NewRenderer()

	// Public pages
	r.AddFromFilesFuncs("public_home", funcs, "templates/public/base.html", "templates/public/home.html")
	r.AddFromFilesFuncs("public_listing", funcs, "templates/public/base.html", "templates/public/listing.html")
	r.AddFromFilesFuncs("public_saved", funcs, "templates/public/base.html", "templates/public/saved.html")
	r.AddFromFilesFuncs("public_terms", funcs, "templates/public/base.html", "templates/public/terms.html")
	r.AddFromFilesFuncs("public_privacy", funcs, "templates/public/base.html", "templates/public/privacy.html")
	r.AddFromFilesFuncs("public_404", funcs, "templates/public/base.html", "templates/public/404.html")

	// Auth pages
	r.AddFromFilesFuncs("auth_login", funcs, "templates/public/base.html", "templates/auth/login.html")
	r.AddFromFilesFuncs("auth_register", funcs, "templates/public/base.html", "templates/auth/register.html")
	r.AddFromFilesFuncs("auth_verify", funcs, "templates/public/base.html", "templates/auth/verify.html")
	r.AddFromFilesFuncs("auth_profile", funcs, "templates/public/base.html", "templates/auth/profile.html")

	// Admin pages
	r.AddFromFilesFuncs("admin_login", funcs, "templates/admin/base.html", "templates/admin/login.html")
	r.AddFromFilesFuncs("admin_dashboard", funcs, "templates/admin/base.html", "templates/admin/dashboard.html", "templates/admin/partials/listings_table.html")
	r.AddFromFilesFuncs("admin_new", funcs, "templates/admin/base.html", "templates/admin/new.html")
	r.AddFromFilesFuncs("admin_edit", funcs, "templates/admin/base.html", "templates/admin/edit.html")
	r.AddFromFilesFuncs("admin_listings_table", funcs, "templates/admin/partials/listings_table.html")

	// Sitemap
	r.AddFromFilesFuncs("sitemap", funcs, "templates/public/sitemap.xml")

	return r
}

func main() {
	cfg := config.Load()

	database, err := db.New(cfg)
	if err != nil {
		panic(err)
	}
	defer database.Close()
	if err := db.InitSchema(database); err != nil {
		panic(err)
	}

	// Repositories
	listingRepo := repo.NewListingRepo(database)
	userRepo := repo.NewUserRepo(database)
	statsRepo := repo.NewStatsRepo(database)

	// Services
	listingService := service.NewListingService(listingRepo)
	adminService := service.NewAdminService(listingRepo, statsRepo)
	authService := service.NewAuthService(userRepo)
	statsService := service.NewStatsService(statsRepo, cfg)
	emailService := service.NewEmailService(cfg)

	// Session store
	store := sessions.NewCookieStore([]byte(cfg.SecretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	// Handlers
	publicHandler := handler.NewPublicHandler(listingService, statsService)
	pagesHandler := handler.NewPagesHandler(cfg, listingService)
	authHandler := handler.NewAuthHandler(authService, emailService, store)
	adminHandler := handler.NewAdminHandler(cfg, adminService, listingService, store)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Template renderer
	r.HTMLRender = createRenderer()

	// Static files
	r.Static("/static", "./static")

	// Middleware
	r.Use(middleware.CSRFMiddleware(store))
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

	// Data files
	r.GET("/data/*filepath", func(c *gin.Context) {
		c.FileFromFS("data/"+c.Param("filepath"), http.Dir("."))
	})

	// Rate limiters
	loginRateLimiter := middleware.NewRateLimiter(10.0/60.0, 10)
	adminRateLimiter := middleware.NewRateLimiter(5.0/60.0, 5)
	registerRateLimiter := middleware.NewRateLimiter(5.0/60.0, 5)

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
	r.POST("/register", middleware.RateLimit(registerRateLimiter), authHandler.Register)
	r.GET("/verify", authHandler.VerifyPage)
	r.POST("/verify", authHandler.Verify)
	r.GET("/login", authHandler.LoginPage)
	r.POST("/login", middleware.RateLimit(loginRateLimiter), authHandler.Login)
	r.POST("/logout", authHandler.Logout)
	r.GET("/profile", middleware.LoginRequired(store), authHandler.Profile)

	// Admin routes
	adminPath := cfg.AdminPath
	admin := r.Group("/" + adminPath)
	{
		admin.GET("/login", adminHandler.LoginPage)
		admin.POST("/login", middleware.RateLimit(adminRateLimiter), adminHandler.Login)
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

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "public_404", handler.BaseData(c))
	})

	r.Run(":8000")
}
