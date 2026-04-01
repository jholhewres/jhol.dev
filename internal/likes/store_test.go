package likes

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "nested", "likes")

	s, err := New(subdir)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if s == nil {
		t.Fatal("New() returned nil store")
	}
}

func TestGetReturnsZeroForUnknownSlug(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if got := s.Get("nonexistent"); got != 0 {
		t.Errorf("Get(nonexistent) = %d, want 0", got)
	}
}

func TestIncrementAndGet(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	count, err := s.Increment("my-post")
	if err != nil {
		t.Fatalf("Increment() error: %v", err)
	}
	if count != 1 {
		t.Errorf("Increment() = %d, want 1", count)
	}

	count, err = s.Increment("my-post")
	if err != nil {
		t.Fatalf("Increment() error: %v", err)
	}
	if count != 2 {
		t.Errorf("Increment() = %d, want 2", count)
	}

	if got := s.Get("my-post"); got != 2 {
		t.Errorf("Get() = %d, want 2", got)
	}
}

func TestPersistenceAcrossRestart(t *testing.T) {
	dir := t.TempDir()

	s1, _ := New(dir)
	s1.Increment("post-a")
	s1.Increment("post-a")
	s1.Increment("post-b")

	// Simulate restart: create new store from same dir
	s2, err := New(dir)
	if err != nil {
		t.Fatalf("New() error on reload: %v", err)
	}

	if got := s2.Get("post-a"); got != 2 {
		t.Errorf("post-a after reload = %d, want 2", got)
	}
	if got := s2.Get("post-b"); got != 1 {
		t.Errorf("post-b after reload = %d, want 1", got)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	s.Increment("test")

	// Verify no .tmp file left behind
	_, err := os.Stat(filepath.Join(dir, "likes.json.tmp"))
	if err == nil {
		t.Error("temp file should not exist after successful write")
	}

	// Verify main file exists and is valid
	raw, err := os.ReadFile(filepath.Join(dir, "likes.json"))
	if err != nil {
		t.Fatalf("likes.json not found: %v", err)
	}
	if len(raw) == 0 {
		t.Error("likes.json is empty")
	}
}

func TestCorruptFileRecovery(t *testing.T) {
	dir := t.TempDir()
	// Write garbage to likes.json
	os.WriteFile(filepath.Join(dir, "likes.json"), []byte("{invalid"), 0644)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("New() should not fail on corrupt file: %v", err)
	}
	if got := s.Get("anything"); got != 0 {
		t.Errorf("Get() on corrupt file = %d, want 0", got)
	}
}

func TestConcurrentIncrements(t *testing.T) {
	s, _ := New(t.TempDir())
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.Increment("concurrent-post")
		}()
	}
	wg.Wait()

	if got := s.Get("concurrent-post"); got != goroutines {
		t.Errorf("concurrent increments = %d, want %d", got, goroutines)
	}
}
