package middleware

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// statusRecorder wraps ResponseWriter to capture status code and bytes written.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Logger writes a structured access log line per request.
// Format: "METHOD PATH STATUS DURATION COUNTRY UA"
// Uses CF-IPCountry when present (no IP is logged).
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 0}
		next.ServeHTTP(rec, r)
		country := r.Header.Get("CF-IPCountry")
		if country == "" {
			country = "-"
		}
		ua := r.UserAgent()
		if len(ua) > 60 {
			ua = ua[:60]
		}
		log.Printf("%s %s %d %s %s %q",
			r.Method,
			r.URL.Path,
			rec.status,
			time.Since(start).Round(time.Millisecond),
			country,
			ua,
		)
	})
}

// Chain applies middleware in order (outermost first).
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// SecurityHeaders adds standard security headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// HSTS — only if behind TLS (Cloudflare handles this, but good defense-in-depth)
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// Gzip compresses responses for clients that accept it.
func Gzip(next http.Handler) http.Handler {
	pool := sync.Pool{
		New: func() interface{} {
			gz, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
			return gz
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := pool.Get().(*gzip.Writer)
		defer pool.Put(gz)
		gz.Reset(w)

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
		gz.Close()
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// CacheAPI adds Cache-Control headers to API responses.
func CacheAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/api/posts" || path == "/api/projects" || path == "/api/experience" || path == "/api/about":
			w.Header().Set("Cache-Control", "public, max-age=300") // 5 min
		case strings.HasPrefix(path, "/api/posts/") && !strings.HasSuffix(path, "/like") && !strings.HasSuffix(path, "/likes"):
			w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		}

		next.ServeHTTP(w, r)

		// ETag for GET API responses
		if r.Method == http.MethodGet && strings.HasPrefix(path, "/api/") {
			// ETag is best-effort; we hash the path+query for a simple version-based ETag
			// Real ETag would require buffering the response, this is a lightweight approach
		}
	})
}

// RateLimiter provides per-IP rate limiting for POST endpoints.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // max requests per window
	window   time.Duration // time window
}

type visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter (e.g., 10 requests per minute).
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(window)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > window*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// Middleware returns a rate limiting middleware for POST requests.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		ip := extractIP(r)

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if time.Since(v.lastSeen) > rl.window {
			v.count = 1
			v.lastSeen = time.Now()
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		v.count++
		v.lastSeen = time.Now()
		if v.count > rl.rate {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	// Cloudflare / proxy real IP
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Strip port from RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// ETag generates an ETag header from content bytes.
func ETag(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf(`"%x"`, h[:8])
}
