package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"jhol.dev/internal/content"
)

const baseURL = "https://jhol.dev"

type SEO struct {
	store *content.Store
}

func NewSEO(store *content.Store) *SEO {
	return &SEO{store: store}
}

func (s *SEO) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /robots.txt", s.robotsTxt)
	mux.HandleFunc("GET /sitemap.xml", s.sitemapXML)
	mux.HandleFunc("GET /feed.xml", s.rssFeed)
}

func (s *SEO) robotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, `User-agent: *
Allow: /

Sitemap: %s/sitemap.xml
`, baseURL)
}

// Sitemap XML types
type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	XMLNS   string    `xml:"xmlns,attr"`
	URLs    []siteURL `xml:"url"`
}

type siteURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (s *SEO) sitemapXML(w http.ResponseWriter, r *http.Request) {
	urls := []siteURL{
		{Loc: baseURL + "/", ChangeFreq: "weekly", Priority: "1.0"},
		{Loc: baseURL + "/about", ChangeFreq: "monthly", Priority: "0.8"},
		{Loc: baseURL + "/blog", ChangeFreq: "weekly", Priority: "0.9"},
		{Loc: baseURL + "/projects", ChangeFreq: "monthly", Priority: "0.7"},
		{Loc: baseURL + "/experience", ChangeFreq: "monthly", Priority: "0.6"},
		{Loc: baseURL + "/contact", ChangeFreq: "yearly", Priority: "0.5"},
	}

	// Add blog posts
	posts := s.store.Posts["en"]
	for _, p := range posts {
		urls = append(urls, siteURL{
			Loc:        baseURL + "/blog/" + p.Slug,
			LastMod:    p.Date,
			ChangeFreq: "monthly",
			Priority:   "0.8",
		})
	}

	sitemap := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(sitemap)
}

// RSS types
type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	PubDate     string    `xml:"pubDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (s *SEO) rssFeed(w http.ResponseWriter, r *http.Request) {
	posts := s.store.Posts["en"]

	var items []rssItem
	for _, p := range posts {
		pubDate := ""
		if t, err := time.Parse("2006-01-02", p.Date); err == nil {
			pubDate = t.Format(time.RFC1123Z)
		}
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        baseURL + "/blog/" + p.Slug,
			Description: p.Summary,
			PubDate:     pubDate,
			GUID:        baseURL + "/blog/" + p.Slug,
		})
	}

	var pubDate string
	if len(posts) > 0 {
		if t, err := time.Parse("2006-01-02", posts[0].Date); err == nil {
			pubDate = t.Format(time.RFC1123Z)
		}
	}

	feed := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "Jhol Hewres — Blog",
			Link:        baseURL,
			Description: "AI Engineer building production-ready AI systems. Multi-agent architecture, RAG & LLMOps.",
			Language:    "en",
			PubDate:     pubDate,
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(feed)
}
