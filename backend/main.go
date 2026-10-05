package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/booth"
	"vrchat-asset-manager/backend/internal/category"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/internal/desktop"
	"vrchat-asset-manager/backend/internal/scanner"
	"vrchat-asset-manager/backend/internal/update"
	"vrchat-asset-manager/backend/internal/web"
	"vrchat-asset-manager/backend/migrations"
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

// releasePort is the default port of the release build. It is fixed so the
// browser keeps per-origin settings (theme, card size) between runs.
const releasePort = "47380"

// githubRepo publishes the releases the update notice compares against.
const githubRepo = "MinhChau9208/VRChat_Asset_Manager"

// HealthResponse represents the health check response payload.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
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
	if err := json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Version: version}); err != nil {
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

// resolveDataDir determines the directory holding app.db, previews/ and
// backups/. DATA_DIR wins; a release build keeps its data next to the
// executable; in development it is the repository's data/ folder.
func resolveDataDir() string {
	if envPath := os.Getenv("DATA_DIR"); envPath != "" {
		return envPath
	}

	if web.Enabled() {
		if exe, err := os.Executable(); err == nil {
			return filepath.Join(filepath.Dir(exe), "data")
		}
	}

	// If running from inside backend/
	if _, err := os.Stat("../data"); err == nil {
		return filepath.Join("..", "data")
	}

	// If running from root directory
	if _, err := os.Stat("data"); err == nil {
		return "data"
	}

	return filepath.Join("..", "data")
}

// resolveDBPath determines the SQLite database file path.
func resolveDBPath(dataDir string) string {
	if envPath := os.Getenv("DB_PATH"); envPath != "" {
		return envPath
	}
	return filepath.Join(dataDir, "app.db")
}

// resolvePreviewsDir determines the directory for storing preview images.
func resolvePreviewsDir(dataDir string) string {
	if envPath := os.Getenv("PREVIEWS_DIR"); envPath != "" {
		return envPath
	}
	return filepath.Join(dataDir, "previews")
}

// hasAssetsTable reports whether the database already holds application data
// (a brand-new database needs no backup).
func hasAssetsTable(db *database.DB) bool {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'assets'").Scan(&n)
	return err == nil && n > 0
}

// logPath is the release build's log file, once it is open.
var logPath string

// openLog sends log output to <dataDir>/logs/app.log: the release exe has no
// console window. A log over 1 MB is kept as app.log.1 and a new one started.
func openLog(dataDir string) error {
	dir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "app.log")
	if info, err := os.Stat(path); err == nil && info.Size() > 1<<20 {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	log.SetOutput(f)
	logPath = path
	return nil
}

// fatalf logs and exits. The release exe has no console, so it also shows
// the message in a dialog.
func fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	if web.Enabled() {
		text := strings.TrimSpace(msg)
		if logPath != "" {
			text += "\n\nDetails: " + logPath
		}
		desktop.ShowError("VRChat Asset Manager", text)
	}
	os.Exit(1)
}

// alreadyRunning reports whether another copy of this app answers at url.
func alreadyRunning(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url + "/health")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var health HealthResponse
	return res.StatusCode == http.StatusOK && json.NewDecoder(res.Body).Decode(&health) == nil && health.Status == "ok"
}

// openBrowser shows the app in the default browser unless NO_BROWSER is set.
func openBrowser(url string) {
	if os.Getenv("NO_BROWSER") != "" {
		return
	}
	if err := desktop.OpenBrowser(url); err != nil {
		log.Printf("Could not open the browser (%v); open %s yourself.\n", err, url)
	}
}

func main() {
	release := web.Enabled()

	dataDir := resolveDataDir()
	if release {
		if err := openLog(dataDir); err != nil {
			fatalf("Cannot write to the data folder %s: %v\n", dataDir, err)
		}
	}

	// Local only: the API opens folders and dialogs on this machine.
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		if release {
			port = releasePort
		}
	}
	appURL := "http://" + net.JoinHostPort(host, port)

	// Bind before touching the database, so a second copy never opens it.
	listener, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		if release && alreadyRunning(appURL) {
			log.Printf("VRChat Asset Manager is already running at %s, opening it.\n", appURL)
			openBrowser(appURL)
			return
		}
		fatalf("Cannot listen on %s: %v\n\nAnother program may be using port %s. Close it, or set PORT to use a different one.\n", appURL, err, port)
	}

	dbPath := resolveDBPath(dataDir)
	log.Printf("Connecting to SQLite database at: %s\n", dbPath)

	db, err := database.Connect(dbPath)
	if err != nil {
		fatalf("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	// Back up an existing database before applying new migrations to it.
	if pending, err := db.PendingMigrations(migrations.FS); err != nil {
		fatalf("Failed to check pending migrations: %v\n", err)
	} else if len(pending) > 0 && hasAssetsTable(db) {
		backupPath := filepath.Join(filepath.Dir(dbPath), "backups",
			fmt.Sprintf("app-%s-before-%s.db", time.Now().Format("20060102-150405"), strings.TrimSuffix(pending[0], ".up.sql")))
		if err := db.Backup(backupPath); err != nil {
			fatalf("Refusing to migrate without a backup: %v\n", err)
		}
		log.Printf("Backed up database to %s before applying %d migration(s)\n", backupPath, len(pending))
	}

	if err := db.Migrate(migrations.FS); err != nil {
		fatalf("Failed to run database migrations: %v\n", err)
	}
	log.Println("Database migrations applied successfully")

	previewsDir := resolvePreviewsDir(dataDir)
	if err := os.MkdirAll(previewsDir, 0755); err != nil {
		log.Printf("Warning: failed to create previews dir %s: %v", previewsDir, err)
	}
	log.Printf("Storing preview images at: %s\n", previewsDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/health/db", dbHealthHandler(db))

	// Category routes
	category.NewHandler(category.NewRepository(db.DB)).RegisterRoutes(mux)

	// Asset domain routes
	assetRepo := asset.NewRepository(db.DB)
	assetHandler := asset.NewHandler(assetRepo)
	assetHandler.SetPreviewsDir(previewsDir)
	assetHandler.RegisterRoutes(mux)

	// Filesystem scanner routes
	scanner.NewHandler(scanner.NewService(db.DB, assetRepo, previewsDir)).RegisterRoutes(mux)

	// BOOTH metadata import routes (network access only on user action)
	booth.NewHandler(booth.NewService(db.DB, booth.NewClient(db.DB), assetRepo, previewsDir)).RegisterRoutes(mux)

	// "New version available" notice (tagged release builds only)
	update.NewChecker(version, githubRepo).RegisterRoutes(mux)

	// Release build: the same server also serves the UI.
	if release {
		mux.Handle("/", web.NewHandler(web.FS()))
	}

	server := &http.Server{Handler: corsMiddleware(mux)}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fatalf("Server failed: %v\n", err)
		}
	}()

	// Ctrl+C (development) or Windows ending the session stops the app.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if release {
		absData, _ := filepath.Abs(dataDir)
		log.Printf("VRChat Asset Manager %s running at %s, data in %s\n", version, appURL, absData)
		openBrowser(appURL)

		// The tray icon is the app's only window; Quit in its menu ends main.
		go func() {
			<-ctx.Done()
			desktop.QuitTray()
		}()
		desktop.RunTray(desktop.TrayOptions{
			Tooltip:    "VRChat Asset Manager",
			Version:    version,
			OnOpen:     func() { openBrowser(appURL) },
			OnOpenData: func() { _ = desktop.OpenFolder(absData) },
		})
	} else {
		log.Printf("VRChat Asset Manager backend listening on %s\n", appURL)
		<-ctx.Done()
	}

	// Finish requests in flight; the deferred db.Close then runs.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Println("Stopped")
}
