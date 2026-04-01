package handler

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"jhol.dev/internal/content"
)

type SPA struct {
	fileFS  fs.FS
	handler http.Handler
	store   *content.Store
}

func NewSPA(fsys fs.FS, store *content.Store) *SPA {
	return &SPA{
		fileFS:  fsys,
		handler: http.FileServer(http.FS(fsys)),
		store:   store,
	}
}

var crawlerAgents = []string{
	"linkedinbot", "facebookexternalhit", "twitterbot",
	"slackbot", "telegrambot", "whatsapp", "googlebot",
	"bingbot", "yandexbot", "baiduspider", "duckduckbot",
}

func isCrawler(ua string) bool {
	lower := strings.ToLower(ua)
	for _, bot := range crawlerAgents {
		if strings.Contains(lower, bot) {
			return true
		}
	}
	return false
}

func (s *SPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to serve the exact file
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if the file exists in the embedded FS
	f, err := s.fileFS.Open(path)
	if err == nil {
		f.Close()

		// Add cache headers for static assets (Vite hashed files)
		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		s.handler.ServeHTTP(w, r)
		return
	}

	// File not found — serve index.html for SPA client-side routing
	indexFile, err := fs.ReadFile(s.fileFS, "index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	html := string(indexFile)

	// For crawlers, inject dynamic meta tags based on the route
	if isCrawler(r.UserAgent()) {
		html = s.injectSEO(html, r.URL.Path)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(html))
}

func (s *SPA) injectSEO(html, urlPath string) string {
	const baseURL = "https://jhol.dev"

	// Blog post: /blog/{slug}
	if strings.HasPrefix(urlPath, "/blog/") {
		slug := strings.TrimPrefix(urlPath, "/blog/")
		slug = strings.TrimSuffix(slug, "/")
		if slug != "" {
			// Try English first, then Portuguese
			for _, lang := range []string{"en", "pt"} {
				if postMap, ok := s.store.PostMap[lang]; ok {
					if post, ok := postMap[slug]; ok {
						return replaceMeta(html,
							post.Title+" — Jhol Hewres",
							post.Summary,
							baseURL+"/blog/"+slug,
							"article",
							baseURL+"/og-blog.png",
						)
					}
				}
			}
		}
	}

	// Default meta for other pages
	titles := map[string]string{
		"/about":      "About — Jhol Hewres",
		"/blog":       "Blog — Jhol Hewres",
		"/projects":   "Projects — Jhol Hewres",
		"/experience": "Experience — Jhol Hewres",
		"/contact":    "Contact — Jhol Hewres",
	}

	cleanPath := strings.TrimSuffix(urlPath, "/")
	if title, ok := titles[cleanPath]; ok {
		return replaceMeta(html, title, "", baseURL+cleanPath, "website")
	}

	return html
}

func replaceMeta(html, title, description, url, ogType string, image ...string) string {
	if len(image) > 0 && image[0] != "" {
		html = replaceMetaContent(html, `property="og:image"`, image[0])
		html = replaceMetaContent(html, `name="twitter:image"`, image[0])
	}
	if title != "" {
		html = replaceTag(html, "<title>", "</title>", fmt.Sprintf("<title>%s</title>", escapeHTML(title)))
		html = replaceMetaContent(html, `property="og:title"`, escapeHTML(title))
		html = replaceMetaContent(html, `name="twitter:title"`, escapeHTML(title))
	}
	if description != "" {
		html = replaceMetaContent(html, `name="description"`, escapeHTML(description))
		html = replaceMetaContent(html, `property="og:description"`, escapeHTML(description))
		html = replaceMetaContent(html, `name="twitter:description"`, escapeHTML(description))
	}
	if url != "" {
		html = replaceMetaContent(html, `property="og:url"`, url)
		html = replaceLinkHref(html, `rel="canonical"`, url)
	}
	if ogType != "" {
		html = replaceMetaContent(html, `property="og:type"`, ogType)
	}
	return html
}

func replaceTag(html, open, close, replacement string) string {
	start := strings.Index(html, open)
	if start == -1 {
		return html
	}
	end := strings.Index(html[start:], close)
	if end == -1 {
		return html
	}
	return html[:start] + replacement + html[start+end+len(close):]
}

func replaceMetaContent(html, attr, newContent string) string {
	idx := strings.Index(html, attr)
	if idx == -1 {
		return html
	}
	// Find content="..." after this attribute
	rest := html[idx:]
	cIdx := strings.Index(rest, `content="`)
	if cIdx == -1 {
		return html
	}
	contentStart := idx + cIdx + len(`content="`)
	contentEnd := strings.Index(html[contentStart:], `"`)
	if contentEnd == -1 {
		return html
	}
	return html[:contentStart] + newContent + html[contentStart+contentEnd:]
}

func replaceLinkHref(html, attr, newHref string) string {
	idx := strings.Index(html, attr)
	if idx == -1 {
		return html
	}
	rest := html[idx:]
	hIdx := strings.Index(rest, `href="`)
	if hIdx == -1 {
		return html
	}
	hrefStart := idx + hIdx + len(`href="`)
	hrefEnd := strings.Index(html[hrefStart:], `"`)
	if hrefEnd == -1 {
		return html
	}
	return html[:hrefStart] + newHref + html[hrefStart+hrefEnd:]
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
