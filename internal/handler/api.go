package handler

import (
	"encoding/json"
	"net/http"

	"jhol.dev/internal/content"
)

type API struct {
	store *content.Store
}

func NewAPI(store *content.Store) *API {
	return &API{store: store}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/posts", a.listPosts)
	mux.HandleFunc("GET /api/posts/{slug}", a.getPost)
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

	// Return posts without full content
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

	// For now, just log and return success
	// In production, this would send an email or store to a file
	writeJSON(w, map[string]string{"status": "ok"})
}
