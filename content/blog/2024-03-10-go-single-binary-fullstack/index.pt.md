---
title: "Apps Full-Stack em um Unico Binario Go"
date: 2024-03-10
tags: ["go", "fullstack", "deployment"]
summary: "Como embutir um frontend React dentro de um binario Go para deploys sem dependencias"
reading_time: 6
---

Uma das features mais subestimadas do Go e o pacote `embed`. Combinado com um framework frontend moderno, voce pode entregar uma aplicacao full-stack completa como um unico binario. Sem runtime Node.js, sem servidor de arquivos estaticos, sem reverse proxy — apenas um arquivo.

## A Abordagem

A ideia e simples:

1. Build do frontend React (ou qualquer SPA) com Vite
2. Use `go:embed` para empacotar os assets buildados no binario Go
3. Sirva a API e o frontend do mesmo servidor HTTP
4. Trate o roteamento SPA com fallback para `index.html`

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

## O SPA Handler

A parte complicada e lidar com roteamento client-side. Quando um usuario navega para `/about`, o servidor precisa servir `index.html` (nao retornar um 404), e deixar o React Router lidar com a rota no lado do cliente.

```go
func spaHandler(fsys fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(fsys))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/")
        _, err := fs.Stat(fsys, path)
        if err != nil {
            r.URL.Path = "/"
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

## Modo Dev

Durante o desenvolvimento, voce nao quer rebuildar o binario Go toda vez que muda uma classe CSS. A solucao e uma flag `-dev` que faz proxy das requests do frontend para o servidor dev do Vite:

```go
if devMode {
    viteURL, _ := url.Parse("http://localhost:5173")
    proxy := httputil.NewSingleHostReverseProxy(viteURL)
    mux.Handle("/", proxy)
}
```

Isso te da o melhor dos dois mundos: HMR do Vite para o frontend e compilacao rapida do Go para o backend.

## Por Que Isso Importa

- **Deploy e trivial**: `scp binary server:/usr/local/bin/` e pronto
- **Imagens Docker sao minusculas**: Estagio final so precisa do binario, sem Node.js
- **Sem dor de cabeca com CORS**: API e frontend estao na mesma origem
- **Um processo para gerenciar**: Nao precisa coordenar multiplos servicos

Este site (jhol.dev) e construido e implantado exatamente assim.
