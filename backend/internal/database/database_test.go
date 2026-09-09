package database_test

import (
	"os"
	"path/filepath"
	"testing"

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

	categories, err := db.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories failed: %v", err)
	}

	expectedCategories := []string{
		"Avatar",
		"Hair",
		"Clothes",
		"Shoes",
		"Accessory",
		"Gimmick",
		"Texture",
		"Material",
		"Shader",
		"Other",
	}

	if len(categories) != len(expectedCategories) {
		t.Fatalf("Expected %d categories, got %d", len(expectedCategories), len(categories))
	}

	for i, expected := range expectedCategories {
		if categories[i].Name != expected {
			t.Errorf("Expected category[%d] to be %q, got %q", i, expected, categories[i].Name)
		}
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

	// 2. Verify all 10 seed categories
	categories, err := db.GetCategories()
	if err != nil {
		t.Fatalf("Failed to query categories from data/app.db: %v", err)
	}

	if len(categories) != 10 {
		t.Fatalf("Expected 10 categories in data/app.db, found %d", len(categories))
	}

	t.Logf("Successfully verified %d seed categories in data/app.db:", len(categories))
	for _, c := range categories {
		t.Logf(" - [%d] %s (created_at: %s)", c.ID, c.Name, c.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

