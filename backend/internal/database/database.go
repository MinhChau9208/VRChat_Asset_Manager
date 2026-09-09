package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps sql.DB with helper methods for the application.
type DB struct {
	*sql.DB
}

// Connect opens an SQLite database at the specified path and configures connection pragmas.
func Connect(dbPath string) (*DB, error) {
	// If path is not in-memory, ensure parent directory exists
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file::memory:") {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create database directory: %w", err)
			}
		}
	}

	// SQLite connection string with pragmas
	// _pragma=foreign_keys(1) ensures foreign keys are enforced
	// _pragma=busy_timeout(5000) prevents database locked errors on concurrent accesses
	// _pragma=journal_mode(WAL) enables Write-Ahead Logging for better concurrency
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite works best with a single write connection pool in Go
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &DB{DB: conn}, nil
}

// Migrate applies all unapplied .sql migrations from the provided filesystem in lexical order.
func (db *DB) Migrate(migrationFS fs.FS) error {
	initQuery := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.Exec(initQuery); err != nil {
		return fmt.Errorf("failed to initialize schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, fileName := range migrationFiles {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", fileName).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", fileName, err)
		}

		if exists {
			continue
		}

		content, err := fs.ReadFile(migrationFS, fileName)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", fileName); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", fileName, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", fileName, err)
		}

		log.Printf("Applied database migration: %s\n", fileName)
	}

	return nil
}

// Category represents a category record.
type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// GetCategories returns all categories currently in the database.
func (db *DB) GetCategories() ([]Category, error) {
	rows, err := db.Query("SELECT id, name, created_at FROM categories ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}
