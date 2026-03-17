# Jhol Hewres — Personal Website

Source code for my personal website, built with **Go** and **React** — a single binary that serves everything.

## About

I'm Jhol Hewres, an AI Engineer building production-ready AI systems. This website hosts my blog, projects, experience, and contact info with full PT/EN internationalization.

## Architecture

Go monolith + React SPA. The Go server embeds the built React frontend via `go:embed`, serves the API, and handles SPA routing — all from a single binary with zero runtime dependencies.

## Project Structure

```text
├── cmd/server/          # Entry point, embeds frontend
├── internal/
│   ├── server/          # HTTP server setup + routes
│   ├── handler/         # API handlers + SPA fallback
│   └── content/         # Markdown/YAML loader + renderer
├── content/
│   ├── blog/            # Blog posts (EN/PT markdown with frontmatter)
│   ├── about.{en,pt}.md # About page content
│   ├── projects.{en,pt}.yaml
│   └── experience.{en,pt}.yaml
├── web/
│   ├── src/             # React + TypeScript + Tailwind CSS
│   └── public/          # Static assets
├── Makefile
└── Dockerfile
```

## Commands

| Command      | Action                                             |
| :----------- | :------------------------------------------------- |
| `make dev`   | Start dev servers (Vite HMR + Go API)              |
| `make build` | Build production binary to `bin/`                  |
| `make serve` | Build and run production server on `:8123`         |
| `make clean` | Remove build artifacts and dependencies            |

## Stack

- **Backend:** Go 1.24+, `net/http`, goldmark, yaml.v3
- **Frontend:** React 19, TypeScript, Tailwind CSS 4, React Router 7
- **Build:** Vite 6, `go:embed`
- **Deploy:** Single binary, PM2, Docker

## License

- **Blog Posts & Content:** [CC BY 4.0](http://creativecommons.org/licenses/by/4.0/)
- **Code:** [MIT License](LICENSE)
