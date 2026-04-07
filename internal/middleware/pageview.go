package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"jhol.dev/internal/analytics"
	"jhol.dev/internal/uaclass"
)

var skipPrefixes = []string{
	"/api/",
	"/assets/",
	"/sitemap.xml",
	"/rss.xml",
	"/robots.txt",
	"/favicon.ico",
	"/og-blog.png",
	"/avatar.jpg",
}

// PageView records page view events into the analytics store.
// It never reads CF-Connecting-IP or RemoteAddr — no IP is ever stored.
func PageView(store *analytics.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			path := r.URL.Path

			// Skip excluded prefixes
			for _, prefix := range skipPrefixes {
				if strings.HasPrefix(path, prefix) || path == prefix {
					return
				}
			}

			// Skip crawlers
			if uaclass.IsCrawler(r.UserAgent()) {
				return
			}

			country := r.Header.Get("CF-IPCountry")
			if country == "" {
				country = "XX"
			}

			city := r.Header.Get("CF-IPCity") // "" if missing — optional

			referer := parseRefererHost(r.Referer(), r.Host)

			e := analytics.Event{
				Path:    path,
				Country: country,
				City:    city,
				Referer: referer,
				UAClass: uaclass.ClassifyUA(r.UserAgent()),
				Ts:      time.Now().Unix(),
			}

			// Non-blocking: never delay the response
			go func() {
				if err := store.Add(e); err != nil {
					// Best-effort: log is not available here, silently drop
					_ = err
				}
			}()
		})
	}
}

// parseRefererHost returns the hostname of the referer URL,
// or "" if it is the same host as the request or unparseable.
func parseRefererHost(referer, requestHost string) string {
	if referer == "" {
		return ""
	}
	u, err := url.Parse(referer)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	// Strip port from requestHost for comparison
	reqHost := requestHost
	if idx := strings.LastIndex(reqHost, ":"); idx != -1 {
		reqHost = reqHost[:idx]
	}
	if host == "" || host == reqHost {
		return ""
	}
	return host
}
