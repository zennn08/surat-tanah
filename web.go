package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// Embed hasil build Svelte. `all:` menyertakan file berawalan '_' bila ada.
//
//go:embed all:frontend/dist
var distFS embed.FS

// spaHandler melayani SPA dari embed.FS: file yang ada dilayani apa adanya,
// path lain fallback ke index.html (aman untuk refresh/deep-link).
func spaHandler() http.Handler {
	sub, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("frontend/dist/index.html tidak ditemukan — jalankan `yarn build` dulu: " + err.Error())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			serveIndex(w, indexBytes)
			return
		}
		if _, err := fs.Stat(sub, p); err != nil {
			// file tidak ada → SPA fallback
			serveIndex(w, indexBytes)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}
