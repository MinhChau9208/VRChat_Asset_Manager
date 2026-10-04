package database_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vrchat-asset-manager/backend/internal/category"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Connect(dbPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("Failed to connect to test db: %v", err)
	}

	if err := db.Migrate(migrations.FS); err != nil {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestMigrationsAndSeedCategories(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	categories, err := category.NewRepository(db.DB).List(context.Background())
	if err != nil {
		t.Fatalf("List categories failed: %v", err)
	}

	// Display order: each top-level category followed by its children ("> " prefix).
	expected := []string{
		"Avatar",
		"Outfit", "> Clothes", "> Shoes",
		"Hair",
		"Accessory", "> Ears & Tail",
		"Face", "> Eyes", "> Expression", "> Makeup",
		"Gimmick", "> Prop",
		"Animation",
		"Texture & Material",
		"Tool & Shader",
		"World",
		"Audio",
		"Other",
	}

	var got []string
	for _, c := range categories {
		if c.ParentID != nil {
			got = append(got, "> "+c.Name)
		} else {
			got = append(got, c.Name)
		}
	}
	if strings.Join(got, ", ") != strings.Join(expected, ", ") {
		t.Errorf("Unexpected category tree:\n got: %v\nwant: %v", got, expected)
	}
}

func TestAssetAndTagRelationships(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Insert a test asset linked to category 1 ("Avatar")
	res, err := db.Exec(`
		INSERT INTO assets (name, category_id, author, booth_url, local_path, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Test Avatar", 1, "ArtistA", "https://booth.pm/123", "D:/assets/avatar", "Test description")
	if err != nil {
		t.Fatalf("Failed to insert asset: %v", err)
	}

	assetID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get asset ID: %v", err)
	}

	// 2. Insert test tags
	resTag1, err := db.Exec("INSERT INTO tags (name) VALUES (?)", "anime")
	if err != nil {
		t.Fatalf("Failed to insert tag: %v", err)
	}
	tagID1, _ := resTag1.LastInsertId()

	resTag2, err := db.Exec("INSERT INTO tags (name) VALUES (?)", "cute")
	if err != nil {
		t.Fatalf("Failed to insert tag: %v", err)
	}
	tagID2, _ := resTag2.LastInsertId()

	// 3. Link asset to tags in asset_tags
	_, err = db.Exec("INSERT INTO asset_tags (asset_id, tag_id) VALUES (?, ?)", assetID, tagID1)
	if err != nil {
		t.Fatalf("Failed to link asset to tag1: %v", err)
	}

	_, err = db.Exec("INSERT INTO asset_tags (asset_id, tag_id) VALUES (?, ?)", assetID, tagID2)
	if err != nil {
		t.Fatalf("Failed to link asset to tag2: %v", err)
	}

	// 4. Verify foreign key enforcement: invalid category_id should fail
	_, err = db.Exec("INSERT INTO assets (name, category_id) VALUES (?, ?)", "Invalid Asset", 99999)
	if err == nil {
		t.Fatalf("Expected error when inserting asset with non-existent category_id, got nil")
	}

	// 5. Test ON DELETE CASCADE on asset_tags
	_, err = db.Exec("DELETE FROM assets WHERE id = ?", assetID)
	if err != nil {
		t.Fatalf("Failed to delete asset: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM asset_tags WHERE asset_id = ?", assetID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count asset_tags: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 asset_tags after asset deletion, got %d", count)
	}

	// Tags themselves should still exist
	err = db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tags: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 tags to remain, got %d", count)
	}
}

func TestCategoryOnDeleteSetNull(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert custom category
	resCat, err := db.Exec("INSERT INTO categories (name) VALUES (?)", "Temporary Category")
	if err != nil {
		t.Fatalf("Failed to insert category: %v", err)
	}
	catID, _ := resCat.LastInsertId()

	// Insert asset with this category
	resAsset, err := db.Exec("INSERT INTO assets (name, category_id) VALUES (?, ?)", "Asset With Temp Category", catID)
	if err != nil {
		t.Fatalf("Failed to insert asset: %v", err)
	}
	assetID, _ := resAsset.LastInsertId()

	// Delete category
	_, err = db.Exec("DELETE FROM categories WHERE id = ?", catID)
	if err != nil {
		t.Fatalf("Failed to delete category: %v", err)
	}

	// Asset category_id should now be NULL
	var categoryID *int64
	err = db.QueryRow("SELECT category_id FROM assets WHERE id = ?", assetID).Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to query asset category_id: %v", err)
	}
	if categoryID != nil {
		t.Errorf("Expected asset category_id to be NULL, got %v", *categoryID)
	}
}

func TestLiveDatabaseFile(t *testing.T) {
	dbPath := filepath.Join("..", "..", "..", "data", "app.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("data/app.db does not exist yet; skipping live file verification")
	}

	db, err := database.Connect(dbPath)
	if err != nil {
		t.Fatalf("Failed to connect to data/app.db: %v", err)
	}
	defer db.Close()

	// 1. Verify health
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping data/app.db: %v", err)
	}

	// 2. Verify categories exist (the user may have edited them)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		t.Fatalf("Failed to query categories from data/app.db: %v", err)
	}
	if count == 0 {
		t.Fatalf("Expected categories in data/app.db, found none")
	}
	t.Logf("data/app.db has %d categories", count)
}

