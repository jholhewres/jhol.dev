package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"

	"jhol.dev/internal/server"
)

//go:embed all:dist
var distFS embed.FS

func main() {
	port := flag.Int("port", 8123, "server port")
	contentDir := flag.String("content", "content", "content directory path")
	dataDir := flag.String("data", "data", "data directory for likes, etc.")
	dev := flag.Bool("dev", false, "enable dev mode (proxy to Vite)")
	flag.Parse()

	var frontendFS fs.FS
	if !*dev {
		var err error
		frontendFS, err = fs.Sub(distFS, "dist")
		if err != nil {
			log.Fatal("failed to get dist sub-filesystem:", err)
		}
	}

	if err := server.Run(server.Config{
		Port:       *port,
		ContentDir: *contentDir,
		DataDir:    *dataDir,
		DevMode:    *dev,
		FrontendFS: frontendFS,
	}); err != nil {
		log.Fatal(err)
	}
}
