package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"jhol.dev/internal/analytics"
	"jhol.dev/internal/content"
	"jhol.dev/internal/handler"
	"jhol.dev/internal/middleware"
	"jhol.dev/internal/views"
)

type Config struct {
	Port        int
	ContentDir  string
	DevMode     bool
	FrontendFS  fs.FS
	DataDir     string
	AdminToken  string
}

func Run(cfg Config) error {
	store, err := content.Load(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}

	mux := http.NewServeMux()

	// Views store
	viewsStore, err := views.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("views store: %w", err)
	}

	// Register API routes
	api := handler.NewAPI(store, cfg.DataDir, viewsStore)
	api.RegisterRoutes(mux)

	// SEO routes (sitemap, RSS, robots.txt)
	seo := handler.NewSEO(store)
	seo.RegisterRoutes(mux)

	// Analytics store + admin endpoint
	analyticsStore, err := analytics.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("analytics store: %w", err)
	}
	admin := handler.NewAdmin(analyticsStore, cfg.AdminToken)
	admin.RegisterRoutes(mux)

	// Frontend serving
	if cfg.DevMode {
		// In dev mode, proxy non-API requests to Vite dev server
		viteURL, _ := url.Parse("http://localhost:5173")
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		mux.Handle("/", proxy)
		log.Println("Dev mode: proxying frontend to Vite at localhost:5173")
	} else {
		// In production, serve embedded frontend
		spa := handler.NewSPA(cfg.FrontendFS, store)
		mux.Handle("/", spa)
	}

	// Rate limiter: 10 POST requests per minute per IP
	rateLimiter := middleware.NewRateLimiter(10, time.Minute)

	// Apply middleware chain
	handler := middleware.Chain(mux,
		middleware.Logger,
		middleware.SecurityHeaders,
		middleware.PageView(analyticsStore),
		middleware.Gzip,
		middleware.CacheAPI,
		rateLimiter.Middleware,
	)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s (content: %s)", addr, cfg.ContentDir)
	return http.ListenAndServe(addr, handler)
}
