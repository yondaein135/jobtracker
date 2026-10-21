package http

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// SPAHandler serves files from dir, falling back to index.html for client-side routing.
// It uses Go 1.24+ os.OpenRoot to ensure sandboxed filesystem access preventing path traversal.
func SPAHandler(dir string) http.Handler {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not built: "+dir+" missing", http.StatusNotFound)
		})
	}

	fileServer := http.FileServer(http.FS(root.FS()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clean := path.Clean("/" + r.URL.Path)
		relPath := strings.TrimPrefix(clean, "/")
		if relPath == "" {
			relPath = "index.html"
		}

		if info, err := root.Stat(relPath); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				// Vite emits content-hashed filenames under /assets.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		indexFile, err := root.Open("index.html")
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.Error(w, "frontend not built: index.html missing", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to open index.html", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()

		info, err := indexFile.Stat()
		if err != nil {
			http.Error(w, "failed to stat index.html", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", info.ModTime(), indexFile)
	})
}
