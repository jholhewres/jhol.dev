---
title: "Apps Full-Stack em um Único Binário Go"
date: 2024-03-10
tags: ["go", "fullstack", "deployment"]
summary: "Como embutir um frontend React dentro de um binário Go para deploys sem dependências"
reading_time: 6
---

Uma das features mais subestimadas do Go é o pacote `embed`. Combinado com um framework frontend moderno, você pode entregar uma aplicação full-stack completa como um único binário. Sem runtime Node.js, sem servidor de arquivos estáticos, sem reverse proxy — apenas um arquivo.

## A Abordagem

A ideia é simples:

1. Build do frontend React (ou qualquer SPA) com Vite
2. Use `go:embed` para empacotar os assets buildados no binário Go
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

A parte complicada é lidar com roteamento client-side. Quando um usuário navega para `/about`, o servidor precisa servir `index.html` (não retornar um 404), e deixar o React Router lidar com a rota no lado do cliente.

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

Durante o desenvolvimento, você não quer rebuildar o binário Go toda vez que muda uma classe CSS. A solução é uma flag `-dev` que faz proxy das requests do frontend para o servidor dev do Vite:

```go
if devMode {
    viteURL, _ := url.Parse("http://localhost:5173")
    proxy := httputil.NewSingleHostReverseProxy(viteURL)
    mux.Handle("/", proxy)
}
```

Isso te dá o melhor dos dois mundos: HMR do Vite para o frontend e compilação rápida do Go para o backend.

## Por Que Isso Importa

- **Deploy é trivial**: `scp binary server:/usr/local/bin/` e pronto
- **Imagens Docker são minúsculas**: Estágio final só precisa do binário, sem Node.js
- **Sem dor de cabeça com CORS**: API e frontend estão na mesma origem
- **Um processo para gerenciar**: Não precisa coordenar múltiplos serviços

Este site (jhol.dev) é construído e implantado exatamente assim.
