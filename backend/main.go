package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

// HealthResponse represents the health check response payload.
type HealthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// corsMiddleware enforces CORS headers for local development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow local development frontend
		if origin == "http://localhost:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler handles GET /health requests.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(HealthResponse{Status: "ok"}); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// dbHealthHandler handles GET /api/health/db requests.
func dbHealthHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(HealthResponse{
				Status: "error",
				Error:  "database connection failed: " + err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}
}

// resolveDBPath determines the default SQLite database file path.
func resolveDBPath() string {
	if envPath := os.Getenv("DB_PATH"); envPath != "" {
		return envPath
	}

	// If running from inside backend/
	if _, err := os.Stat("../data"); err == nil {
		return filepath.Join("..", "data", "app.db")
	}

	// If running from root directory
	if _, err := os.Stat("data"); err == nil {
		return filepath.Join("data", "app.db")
	}

	return filepath.Join("..", "data", "app.db")
}

// resolvePreviewsDir determines the default directory for storing preview images.
func resolvePreviewsDir() string {
	if envPath := os.Getenv("PREVIEWS_DIR"); envPath != "" {
		return envPath
	}

	// If running from inside backend/
	if _, err := os.Stat("../data"); err == nil {
		return filepath.Join("..", "data", "previews")
	}

	// If running from root directory
	if _, err := os.Stat("data"); err == nil {
		return filepath.Join("data", "previews")
	}

	return filepath.Join("..", "data", "previews")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := resolveDBPath()
	log.Printf("Connecting to SQLite database at: %s\n", dbPath)

	db, err := database.Connect(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	if err := db.Migrate(migrations.FS); err != nil {
		log.Fatalf("Failed to run database migrations: %v\n", err)
	}
	log.Println("Database migrations applied successfully")

	previewsDir := resolvePreviewsDir()
	if err := os.MkdirAll(previewsDir, 0755); err != nil {
		log.Printf("Warning: failed to create previews dir %s: %v", previewsDir, err)
	}
	log.Printf("Storing preview images at: %s\n", previewsDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/health/db", dbHealthHandler(db))

	// Categories route
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
		categories, err := db.GetCategories()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load categories"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(categories)
	})

	// Asset domain routes
	assetRepo := asset.NewRepository(db.DB)
	assetHandler := asset.NewHandler(assetRepo)
	assetHandler.SetPreviewsDir(previewsDir)
	assetHandler.RegisterRoutes(mux)

	handler := corsMiddleware(mux)

	addr := ":" + port
	log.Printf("VRChat Asset Manager backend listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}
