package handler

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jhol.dev/internal/content"
)

func newTestStore() *content.Store {
	return &content.Store{
		Posts: map[string][]content.Post{
			"en": {
				{Slug: "test-post", Title: "Test Post", Date: "2026-01-15", Summary: "A test", Tags: []string{"go"}, ReadingTime: 3},
			},
		},
		PostMap: map[string]map[string]content.Post{
			"en": {
				"test-post": {Slug: "test-post", Title: "Test Post", Date: "2026-01-15", Summary: "A test", Tags: []string{"go"}, ReadingTime: 3},
			},
		},
		Projects:   map[string][]content.Project{},
		Experience: map[string][]content.Experience{},
		About:      map[string]content.AboutContent{},
	}
}

func TestRobotsTxt(t *testing.T) {
	seo := NewSEO(newTestStore())
	req := httptest.NewRequest("GET", "/robots.txt", nil)
	rr := httptest.NewRecorder()

	seo.robotsTxt(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "User-agent: *") {
		t.Error("robots.txt missing User-agent")
	}
	if !strings.Contains(body, "Sitemap:") {
		t.Error("robots.txt missing Sitemap")
	}
}

func TestSitemapXML(t *testing.T) {
	seo := NewSEO(newTestStore())
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	rr := httptest.NewRecorder()

	seo.sitemapXML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "xml") {
		t.Errorf("Content-Type = %q, want xml", ct)
	}

	// Verify valid XML
	var sitemap urlSet
	if err := xml.Unmarshal(rr.Body.Bytes(), &sitemap); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	// Should have static pages + 1 blog post
	if len(sitemap.URLs) < 7 {
		t.Errorf("sitemap has %d URLs, want at least 7", len(sitemap.URLs))
	}

	// Check blog post URL is present
	found := false
	for _, u := range sitemap.URLs {
		if strings.Contains(u.Loc, "test-post") {
			found = true
			break
		}
	}
	if !found {
		t.Error("sitemap missing blog post URL")
	}
}

func TestRSSFeed(t *testing.T) {
	seo := NewSEO(newTestStore())
	req := httptest.NewRequest("GET", "/feed.xml", nil)
	rr := httptest.NewRecorder()

	seo.rssFeed(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var feed rssRoot
	if err := xml.Unmarshal(rr.Body.Bytes(), &feed); err != nil {
		t.Fatalf("invalid RSS XML: %v", err)
	}

	if feed.Channel.Title == "" {
		t.Error("RSS feed missing channel title")
	}
	if len(feed.Channel.Items) != 1 {
		t.Errorf("RSS items = %d, want 1", len(feed.Channel.Items))
	}
	if feed.Channel.Items[0].Title != "Test Post" {
		t.Errorf("item title = %q, want 'Test Post'", feed.Channel.Items[0].Title)
	}
}
