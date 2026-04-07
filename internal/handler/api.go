package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"jhol.dev/internal/content"
	"jhol.dev/internal/likes"
	"jhol.dev/internal/views"
)

type API struct {
	store      *content.Store
	likesStore *likes.Store
	viewsStore *views.Store
}

func NewAPI(store *content.Store, dataDir string, viewsStore *views.Store) *API {
	ls, err := likes.New(dataDir)
	if err != nil {
		log.Printf("warning: likes store at %s failed: %v, falling back to /tmp", dataDir, err)
		ls, err = likes.New("/tmp/jhol-dev-likes")
		if err != nil {
			log.Printf("warning: likes store fallback also failed: %v, likes disabled", err)
		}
	}
	return &API{store: store, likesStore: ls, viewsStore: viewsStore}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/posts", a.listPosts)
	mux.HandleFunc("GET /api/posts/{slug}", a.getPost)
	mux.HandleFunc("GET /api/posts/{slug}/likes", a.getLikes)
	mux.HandleFunc("POST /api/posts/{slug}/like", a.addLike)
	mux.HandleFunc("GET /api/posts/{slug}/views", a.getViews)
	mux.HandleFunc("POST /api/posts/{slug}/view", a.addView)
	mux.HandleFunc("GET /api/projects", a.listProjects)
	mux.HandleFunc("GET /api/experience", a.listExperience)
	mux.HandleFunc("GET /api/about", a.getAbout)
	mux.HandleFunc("POST /api/contact", a.submitContact)
}

func lang(r *http.Request) string {
	l := r.URL.Query().Get("lang")
	if l == "pt" {
		return "pt"
	}
	return "en"
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (a *API) listPosts(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	posts := a.store.Posts[l]
	if posts == nil {
		posts = a.store.Posts["en"]
	}

	type postSummary struct {
		Slug        string   `json:"slug"`
		Title       string   `json:"title"`
		Date        string   `json:"date"`
		Tags        []string `json:"tags"`
		Summary     string   `json:"summary"`
		ReadingTime int      `json:"reading_time"`
	}

	summaries := make([]postSummary, len(posts))
	for i, p := range posts {
		summaries[i] = postSummary{
			Slug:        p.Slug,
			Title:       p.Title,
			Date:        p.Date,
			Tags:        p.Tags,
			Summary:     p.Summary,
			ReadingTime: p.ReadingTime,
		}
	}

	writeJSON(w, summaries)
}

func (a *API) getPost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	l := lang(r)

	// Avoid matching /likes and /like sub-routes
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}

	postMap := a.store.PostMap[l]
	if postMap == nil {
		postMap = a.store.PostMap["en"]
	}

	post, ok := postMap[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, post)
}

func (a *API) getLikes(w http.ResponseWriter, r *http.Request) {
	if a.likesStore == nil {
		writeJSON(w, map[string]int{"likes": 0})
		return
	}
	slug := r.PathValue("slug")
	count := a.likesStore.Get(slug)
	writeJSON(w, map[string]int{"likes": count})
}

func (a *API) addLike(w http.ResponseWriter, r *http.Request) {
	if a.likesStore == nil {
		http.Error(w, "Likes not available", http.StatusServiceUnavailable)
		return
	}
	slug := r.PathValue("slug")

	// Verify post exists
	found := false
	for _, lang := range []string{"en", "pt"} {
		if pm, ok := a.store.PostMap[lang]; ok {
			if _, ok := pm[slug]; ok {
				found = true
				break
			}
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}

	count, err := a.likesStore.Increment(slug)
	if err != nil {
		http.Error(w, "Failed to save like", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int{"likes": count})
}

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	projects := a.store.Projects[l]
	if projects == nil {
		projects = a.store.Projects["en"]
	}
	writeJSON(w, projects)
}

func (a *API) listExperience(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	exp := a.store.Experience[l]
	if exp == nil {
		exp = a.store.Experience["en"]
	}
	writeJSON(w, exp)
}

func (a *API) getAbout(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	about, ok := a.store.About[l]
	if !ok {
		about = a.store.About["en"]
	}
	writeJSON(w, about)
}

func (a *API) getViews(w http.ResponseWriter, r *http.Request) {
	if a.viewsStore == nil {
		writeJSON(w, map[string]int{"views": 0})
		return
	}
	slug := r.PathValue("slug")
	count := a.viewsStore.Get(slug)
	writeJSON(w, map[string]int{"views": count})
}

func (a *API) addView(w http.ResponseWriter, r *http.Request) {
	if a.viewsStore == nil {
		http.Error(w, "Views not available", http.StatusServiceUnavailable)
		return
	}
	slug := r.PathValue("slug")

	// Verify post exists
	found := false
	for _, lang := range []string{"en", "pt"} {
		if pm, ok := a.store.PostMap[lang]; ok {
			if _, ok := pm[slug]; ok {
				found = true
				break
			}
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}

	count, err := a.viewsStore.Increment(slug)
	if err != nil {
		http.Error(w, "Failed to save view", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int{"views": count})
}

func (a *API) submitContact(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Name == "" || body.Email == "" || body.Message == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}
