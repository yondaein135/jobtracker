// Command server runs the job tracker: JSON API under /api/* plus the compiled
// React SPA from STATIC_DIR.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"job-tracker/backend/internal/api"
	"job-tracker/backend/internal/db"
)

func main() {
	port := env("PORT", "8080")
	dbPath := env("DB_PATH", "./data/tracker.db")
	staticDir := env("STATIC_DIR", "./static")
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ",")

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := db.Migrate(ctx, conn); err != nil {
		cancel()
		log.Fatalf("migrate: %v", err)
	}

	seedDummy := env("SEED_DUMMY_DATA", "0")
	if seedDummy == "1" || seedDummy == "true" || strings.Contains(dbPath, "dummy") {
		if err := db.SeedDummyData(ctx, conn); err != nil {
			log.Printf("seed dummy data warning: %v", err)
		} else {
			log.Printf("dummy data checked/seeded at %s", dbPath)
		}
	}
	cancel()
	log.Printf("database ready at %s (WAL, foreign_keys=ON)", dbPath)

	apiHandler := api.NewRouter(conn, origins)

	root := http.NewServeMux()
	root.Handle("/api/", apiHandler)
	root.Handle("/", spaHandler(staticDir))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(root),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("listening on http://localhost:%s (static: %s)", port, staticDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// spaHandler serves files from dir, falling back to index.html for unknown
// paths so client-side routing works.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clean := path.Clean("/" + r.URL.Path)
		full := filepath.Join(dir, filepath.FromSlash(clean))

		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				// Vite emits content-hashed filenames under /assets.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fs.ServeHTTP(w, r)
			return
		}

		if _, err := os.Stat(index); err != nil {
			http.Error(w, "frontend not built: "+index+" missing", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
