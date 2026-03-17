package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"jhol.dev/internal/content"
	"jhol.dev/internal/handler"
)

type Config struct {
	Port       int
	ContentDir string
	DevMode    bool
	FrontendFS fs.FS
}

func Run(cfg Config) error {
	store, err := content.Load(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}

	mux := http.NewServeMux()

	// Register API routes
	api := handler.NewAPI(store)
	api.RegisterRoutes(mux)

	// Frontend serving
	if cfg.DevMode {
		// In dev mode, proxy non-API requests to Vite dev server
		viteURL, _ := url.Parse("http://localhost:5173")
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		mux.Handle("/", proxy)
		log.Println("Dev mode: proxying frontend to Vite at localhost:5173")
	} else {
		// In production, serve embedded frontend
		spa := handler.NewSPA(cfg.FrontendFS)
		mux.Handle("/", spa)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s (content: %s)", addr, cfg.ContentDir)
	return http.ListenAndServe(addr, mux)
}
