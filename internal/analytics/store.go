package analytics

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one page view. NO IP FIELD — never persist IP.
type Event struct {
	Path    string `json:"p"`
	Country string `json:"c"`            // CF-IPCountry, "XX" if missing
	City    string `json:"ci,omitempty"` // CF-IPCity, "" if missing
	Referer string `json:"r,omitempty"`  // host only, no path
	UAClass string `json:"u"`            // chrome/firefox/safari/edge/other
	Ts      int64  `json:"t"`            // unix seconds
}

type fileFormat struct {
	Events    []Event `json:"events"`
	RotatedAt int64   `json:"rotated_at"`
}

// Store manages page view events using a JSON file.
type Store struct {
	mu        sync.RWMutex
	dir       string
	path      string
	events    []Event
	rotatedAt int64
}

// Stats aggregates event data over a time window.
type Stats struct {
	Total       int            `json:"total"`
	ByCountry   map[string]int `json:"by_country"`
	ByPath      map[string]int `json:"by_path"`
	ByDay       map[string]int `json:"by_day"`    // YYYY-MM-DD
	ByReferer   map[string]int `json:"by_referer"`
	WindowHours int            `json:"window_hours"`
}

// New creates or loads an analytics store from the given directory.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		dir:  dataDir,
		path: filepath.Join(dataDir, "visits.json"),
	}

	raw, err := os.ReadFile(s.path)
	if err == nil {
		var ff fileFormat
		if err := json.Unmarshal(raw, &ff); err != nil {
			log.Printf("warning: invalid visits.json, starting fresh: %v", err)
			s.events = nil
		} else {
			s.events = ff.Events
			s.rotatedAt = ff.RotatedAt
		}
	}

	if s.events == nil {
		s.events = []Event{}
	}

	return s, nil
}

// Add appends an event and flushes atomically.
func (s *Store) Add(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, e)

	if err := s.flush(); err != nil {
		// rollback
		s.events = s.events[:len(s.events)-1]
		return err
	}

	// Rotate if over 10k events
	if len(s.events) > 10000 {
		if err := s.rotate(); err != nil {
			log.Printf("analytics: rotation failed: %v", err)
		}
	}

	return nil
}

// flush writes current state to disk atomically (must hold mu.Lock).
func (s *Store) flush() error {
	ff := fileFormat{
		Events:    s.events,
		RotatedAt: s.rotatedAt,
	}

	raw, err := json.Marshal(ff)
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// rotate moves current events to a dated archive file and resets (must hold mu.Lock).
func (s *Store) rotate() error {
	now := time.Now()
	archiveName := fmt.Sprintf("visits-%s.json", now.Format("200601"))
	archivePath := filepath.Join(s.dir, archiveName)

	ff := fileFormat{
		Events:    s.events,
		RotatedAt: now.Unix(),
	}
	raw, err := json.Marshal(ff)
	if err != nil {
		return err
	}

	tmp := archivePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, archivePath); err != nil {
		return err
	}

	s.events = []Event{}
	s.rotatedAt = now.Unix()
	return s.flush()
}

// Stats returns aggregated analytics for events within the given window.
func (s *Store) Stats(window time.Duration) Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-window).Unix()

	st := Stats{
		ByCountry:   make(map[string]int),
		ByPath:      make(map[string]int),
		ByDay:       make(map[string]int),
		ByReferer:   make(map[string]int),
		WindowHours: int(window.Hours()),
	}

	for _, e := range s.events {
		if e.Ts < cutoff {
			continue
		}
		st.Total++
		st.ByCountry[e.Country]++
		st.ByPath[e.Path]++
		day := time.Unix(e.Ts, 0).UTC().Format("2006-01-02")
		st.ByDay[day]++
		if e.Referer != "" {
			st.ByReferer[e.Referer]++
		}
	}

	return st
}
