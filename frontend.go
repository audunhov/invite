package main

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

func serveFrontend(mux *http.ServeMux) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't serve frontend for API, Swagger, or OpenAPI spec
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/swagger/") || path == "/openapi.json" {
			// This shouldn't happen if mux is configured correctly, but just in case
			http.NotFound(w, r)
			return
		}

		// Try to serve the file from the filesystem
		f, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// If the file doesn't exist, serve index.html (for SPA routing)
		index, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer index.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, index)
	})
}
