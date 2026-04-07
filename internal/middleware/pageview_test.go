package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jhol.dev/internal/analytics"
)

// nopHandler is a minimal http.Handler that does nothing.
var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func newStore(t *testing.T) *analytics.Store {
	t.Helper()
	s, err := analytics.New(t.TempDir())
	if err != nil {
		t.Fatalf("analytics.New: %v", err)
	}
	return s
}

func TestPageViewSkipsAPI(t *testing.T) {
	store := newStore(t)
	h := PageView(store)(nopHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	h.ServeHTTP(httptest.NewRecorder(), req)

	// Give goroutine time to run (it won't, because we return early)
	time.Sleep(10 * time.Millisecond)

	st := store.Stats(24 * time.Hour)
	if st.Total != 0 {
		t.Errorf("expected 0 events for /api/ path, got %d", st.Total)
	}
}

func TestPageViewSkipsCrawler(t *testing.T) {
	store := newStore(t)
	h := PageView(store)(nopHandler)

	req := httptest.NewRequest(http.MethodGet, "/blog/foo", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.google.com/bot.html)")
	h.ServeHTTP(httptest.NewRecorder(), req)

	time.Sleep(10 * time.Millisecond)

	st := store.Stats(24 * time.Hour)
	if st.Total != 0 {
		t.Errorf("expected 0 events for crawler UA, got %d", st.Total)
	}
}

func TestPageViewRecordsCountry(t *testing.T) {
	store := newStore(t)
	h := PageView(store)(nopHandler)

	req := httptest.NewRequest(http.MethodGet, "/blog/foo", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	req.Header.Set("CF-IPCountry", "BR")
	h.ServeHTTP(httptest.NewRecorder(), req)

	// Wait for async Add goroutine
	time.Sleep(50 * time.Millisecond)

	st := store.Stats(24 * time.Hour)
	if st.Total != 1 {
		t.Fatalf("expected 1 event, got %d", st.Total)
	}
	if st.ByCountry["BR"] != 1 {
		t.Errorf("expected country BR=1, got %v", st.ByCountry)
	}
}

func TestPageViewDoesNotPersistIP(t *testing.T) {
	dir := t.TempDir()
	store, err := analytics.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := PageView(store)(nopHandler)

	// Use a recognizable fake IP in RemoteAddr
	req := httptest.NewRequest(http.MethodGet, "/blog/foo", nil)
	req.RemoteAddr = "192.168.99.1:1234"
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	req.Header.Set("CF-IPCountry", "BR")
	h.ServeHTTP(httptest.NewRecorder(), req)

	// Wait for async Add goroutine
	time.Sleep(50 * time.Millisecond)

	// Read the raw JSON file and verify the IP does not appear
	raw, err := os.ReadFile(filepath.Join(dir, "visits.json"))
	if err != nil {
		t.Fatalf("could not read visits.json: %v", err)
	}

	if strings.Contains(string(raw), "192.168.99.1") {
		t.Error("SECURITY: visits.json must not contain the client IP address")
	}

	// Also verify via struct: no IP field should be present in any event object
	var ff struct {
		Events []map[string]interface{} `json:"events"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		t.Fatal(err)
	}
	if len(ff.Events) == 0 {
		t.Fatal("expected at least one event in visits.json")
	}
	for _, ev := range ff.Events {
		for _, forbidden := range []string{"ip", "remote_addr", "addr", "connecting_ip"} {
			if _, ok := ev[forbidden]; ok {
				t.Errorf("event JSON must not have field %q", forbidden)
			}
		}
	}
}
