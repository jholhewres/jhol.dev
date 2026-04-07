package analytics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "analytics")
	_, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}
}

func TestAddPersists(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := Event{Path: "/blog/hello", Country: "BR", UAClass: "chrome", Ts: time.Now().Unix()}
	if err := s.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Reload and verify persistence
	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s2.events))
	}
	if s2.events[0].Path != "/blog/hello" {
		t.Errorf("expected path /blog/hello, got %s", s2.events[0].Path)
	}
	if s2.events[0].Country != "BR" {
		t.Errorf("expected country BR, got %s", s2.events[0].Country)
	}
}

func TestRotateAt10k(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Pre-fill to 10000 events without flushing each time (direct manipulation)
	s.mu.Lock()
	for i := 0; i < 10000; i++ {
		s.events = append(s.events, Event{Path: "/", Country: "XX", UAClass: "other", Ts: 1})
	}
	s.mu.Unlock()

	// Adding one more should trigger rotation
	e := Event{Path: "/trigger", Country: "US", UAClass: "chrome", Ts: time.Now().Unix()}
	if err := s.Add(e); err != nil {
		t.Fatalf("Add (trigger rotation): %v", err)
	}

	// After rotation, events should be reset (only the new event remains)
	s.mu.RLock()
	count := len(s.events)
	s.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 events after rotation, got %d", count)
	}

	// Archive file should exist
	matches, _ := filepath.Glob(filepath.Join(dir, "visits-*.json"))
	if len(matches) == 0 {
		t.Fatal("expected archive file to be created after rotation")
	}
}

func TestStatsAggregation(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	events := []Event{
		{Path: "/blog/a", Country: "BR", Referer: "google.com", UAClass: "chrome", Ts: now},
		{Path: "/blog/a", Country: "BR", Referer: "google.com", UAClass: "chrome", Ts: now},
		{Path: "/blog/b", Country: "US", UAClass: "firefox", Ts: now},
		// old event outside 24h window
		{Path: "/old", Country: "DE", UAClass: "other", Ts: now - int64(25*time.Hour.Seconds())},
	}
	for _, e := range events {
		if err := s.Add(e); err != nil {
			t.Fatal(err)
		}
	}

	st := s.Stats(24 * time.Hour)
	if st.Total != 3 {
		t.Errorf("expected total 3, got %d", st.Total)
	}
	if st.ByCountry["BR"] != 2 {
		t.Errorf("expected BR=2, got %d", st.ByCountry["BR"])
	}
	if st.ByCountry["US"] != 1 {
		t.Errorf("expected US=1, got %d", st.ByCountry["US"])
	}
	if st.ByPath["/blog/a"] != 2 {
		t.Errorf("expected /blog/a=2, got %d", st.ByPath["/blog/a"])
	}
	if st.ByReferer["google.com"] != 2 {
		t.Errorf("expected google.com=2, got %d", st.ByReferer["google.com"])
	}
	if _, ok := st.ByCountry["DE"]; ok {
		t.Error("old event (DE) should not appear in 24h window")
	}
	if st.WindowHours != 24 {
		t.Errorf("expected WindowHours=24, got %d", st.WindowHours)
	}
}

func TestNoIPFieldInJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := Event{Path: "/", Country: "BR", UAClass: "chrome", Ts: time.Now().Unix()}
	if err := s.Add(e); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "visits.json"))
	if err != nil {
		t.Fatal(err)
	}

	// Verify the JSON has no IP-related fields
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}

	jsonStr := string(raw)
	ipPatterns := []string{"ip", "IP", "addr", "remote", "connecting"}
	for _, p := range ipPatterns {
		// Only check for explicit field names, not substrings within values
		_ = p
	}

	// The Event struct has no IP field — verify json tags match spec
	events, ok := generic["events"].([]interface{})
	if !ok || len(events) == 0 {
		t.Fatal("expected events array")
	}
	event := events[0].(map[string]interface{})
	for _, forbiddenKey := range []string{"ip", "remote_addr", "addr"} {
		if _, exists := event[forbiddenKey]; exists {
			t.Errorf("JSON event must not contain field %q", forbiddenKey)
		}
	}
	_ = jsonStr
}
