package likes

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Store manages post likes using a JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	data map[string]int // slug -> count
}

// New creates or loads a likes store from the given directory.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		path: filepath.Join(dataDir, "likes.json"),
		data: make(map[string]int),
	}

	raw, err := os.ReadFile(s.path)
	if err == nil {
		if err := json.Unmarshal(raw, &s.data); err != nil {
			log.Printf("warning: invalid likes.json, starting fresh: %v", err)
			s.data = make(map[string]int)
		}
	}

	return s, nil
}

// Get returns the like count for a slug.
func (s *Store) Get(slug string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[slug]
}

// Increment adds one like and returns the new count.
func (s *Store) Increment(slug string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[slug]++
	count := s.data[slug]

	raw, err := json.Marshal(s.data)
	if err != nil {
		s.data[slug]-- // rollback
		return 0, err
	}

	// Atomic write: temp file + rename to avoid corruption on crash
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		s.data[slug]-- // rollback
		return 0, err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		s.data[slug]-- // rollback
		return 0, err
	}

	return count, nil
}
