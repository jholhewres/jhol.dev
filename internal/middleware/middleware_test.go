package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	tests := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
		"X-XSS-Protection":      "1; mode=block",
	}

	for header, want := range tests {
		if got := rr.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestSecurityHeadersHSTS(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	// Without HTTPS - no HSTS
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS should not be set without TLS, got %q", got)
	}

	// With X-Forwarded-Proto: https
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Forwarded-Proto", "https")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if got := rr2.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("HSTS should be set when X-Forwarded-Proto is https")
	}
}

func TestGzipCompression(t *testing.T) {
	body := strings.Repeat("Hello, World! ", 100)
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))

	// With gzip accept
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip Content-Encoding")
	}

	gr, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gr.Close()

	decoded, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("reading gzip body: %v", err)
	}
	if string(decoded) != body {
		t.Errorf("decoded body length = %d, want %d", len(decoded), len(body))
	}
}

func TestGzipSkippedWithoutAcceptEncoding(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not gzip without Accept-Encoding header")
	}
	if rr.Body.String() != "hello" {
		t.Errorf("body = %q, want 'hello'", rr.Body.String())
	}
}

func TestCacheAPIHeaders(t *testing.T) {
	handler := CacheAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	tests := []struct {
		path  string
		cache string
	}{
		{"/api/posts", "public, max-age=300"},
		{"/api/projects", "public, max-age=300"},
		{"/api/experience", "public, max-age=300"},
		{"/api/about", "public, max-age=300"},
		{"/api/posts/my-slug", "public, max-age=3600"},
		{"/api/posts/my-slug/likes", ""},
		{"/api/posts/my-slug/like", ""},
		{"/other", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		got := rr.Header().Get("Cache-Control")
		if got != tt.cache {
			t.Errorf("CacheAPI(%s) = %q, want %q", tt.path, got, tt.cache)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 3 POST requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i+1, rr.Code)
		}
	}

	// 4th should be rate limited
	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: got %d, want 429", rr.Code)
	}

	// GET requests should not be limited
	reqGet := httptest.NewRequest("GET", "/api/test", nil)
	reqGet.RemoteAddr = "1.2.3.4:1234"
	rrGet := httptest.NewRecorder()
	handler.ServeHTTP(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("GET request: got %d, want 200", rrGet.Code)
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	rl := NewRateLimiter(1, time.Second)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Different IPs should have separate limits
	for _, ip := range []string{"1.1.1.1:1234", "2.2.2.2:1234", "3.3.3.3:1234"} {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("IP %s: got %d, want 200", ip, rr.Code)
		}
	}
}

func TestExtractIPFromCloudflare(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("CF-Connecting-IP", "9.8.7.6")
	req.RemoteAddr = "127.0.0.1:1234"

	if got := extractIP(req); got != "9.8.7.6" {
		t.Errorf("extractIP with CF header = %q, want 9.8.7.6", got)
	}
}

func TestExtractIPFromXFF(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:1234"

	if got := extractIP(req); got != "5.6.7.8" {
		t.Errorf("extractIP with XFF = %q, want 5.6.7.8", got)
	}
}

func TestChain(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), mw1, mw2)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("chain order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestLoggerWritesAccessLine(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	req.Header.Set("CF-IPCountry", "BR")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	line := buf.String()
	if !strings.Contains(line, "GET") {
		t.Errorf("log line missing method GET: %q", line)
	}
	if !strings.Contains(line, "/test-path") {
		t.Errorf("log line missing path /test-path: %q", line)
	}
	if !strings.Contains(line, "200") {
		t.Errorf("log line missing status 200: %q", line)
	}
	if !strings.Contains(line, "BR") {
		t.Errorf("log line missing country BR: %q", line)
	}
}

func TestLoggerCapturesStatusCode(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
	}))

	req := httptest.NewRequest(http.MethodGet, "/teapot", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	line := buf.String()
	if !strings.Contains(line, "418") {
		t.Errorf("log line missing status 418: %q", line)
	}
}
