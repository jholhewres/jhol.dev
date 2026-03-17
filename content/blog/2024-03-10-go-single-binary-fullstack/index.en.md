---
title: "Full-Stack Apps in a Single Go Binary"
date: 2024-03-10
tags: ["go", "fullstack", "deployment"]
summary: "How to embed a React frontend inside a Go binary for zero-dependency deployments"
reading_time: 6
---

One of Go's most underrated features is the `embed` package. Combined with a modern frontend framework, you can ship a complete full-stack application as a single binary. No Node.js runtime, no static file server, no reverse proxy — just one file.

## The Approach

The idea is simple:

1. Build your React (or any SPA) frontend with Vite
2. Use `go:embed` to bundle the built assets into your Go binary
3. Serve the API and frontend from the same HTTP server
4. Handle SPA routing with a fallback to `index.html`

```go
//go:embed all:dist
var distFS embed.FS

func main() {
    frontendFS, _ := fs.Sub(distFS, "dist")

    mux := http.NewServeMux()
    mux.HandleFunc("/api/", apiHandler)
    mux.Handle("/", spaHandler(frontendFS))

    http.ListenAndServe(":8080", mux)
}
```

## The SPA Handler

The tricky part is handling client-side routing. When a user navigates to `/about`, the server needs to serve `index.html` (not return a 404), and let React Router handle the route on the client side.

```go
func spaHandler(fsys fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(fsys))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/")
        _, err := fs.Stat(fsys, path)
        if err != nil {
            // File doesn't exist, serve index.html
            r.URL.Path = "/"
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

## Dev Mode

During development, you don't want to rebuild the Go binary every time you change a CSS class. The solution is a `-dev` flag that proxies frontend requests to Vite's dev server:

```go
if devMode {
    viteURL, _ := url.Parse("http://localhost:5173")
    proxy := httputil.NewSingleHostReverseProxy(viteURL)
    mux.Handle("/", proxy)
}
```

This gives you the best of both worlds: Vite's HMR for the frontend and Go's fast compilation for the backend.

## Why This Matters

- **Deployment is trivial**: `scp binary server:/usr/local/bin/` and you're done
- **Docker images are tiny**: Final stage only needs the binary, no Node.js
- **No CORS headaches**: API and frontend are on the same origin
- **One process to manage**: No need to coordinate multiple services

This is exactly how this website (jhol.dev) is built and deployed.
