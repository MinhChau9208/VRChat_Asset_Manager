package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"

	_ "modernc.org/sqlite"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("  VRChat Asset Manager - Milestone 1 Verification ")
	fmt.Println("==================================================")

	allPassed := true

	// 1. Check live database file
	dbPath := filepath.Join("..", "data", "app.db")
	if _, err := os.Stat(dbPath); err != nil {
		// try from workspace root
		dbPath = filepath.Join("data", "app.db")
		if _, err := os.Stat(dbPath); err != nil {
			fmt.Printf("[FAIL] 1. SQLite database file does not exist at expected path: %v\n", err)
			os.Exit(1)
		}
	}
	absPath, _ := filepath.Abs(dbPath)
	fmt.Printf("[PASS] 1. Database file exists: %s\n", absPath)

	db, err := database.Connect(dbPath)
	if err != nil {
		fmt.Printf("[FAIL] 1. Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Check table existence
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		fmt.Printf("[FAIL] 2. Querying tables failed: %v\n", err)
		allPassed = false
	} else {
		defer rows.Close()
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				tables = append(tables, name)
			}
		}

		expectedTables := map[string]bool{
			"categories":        false,
			"assets":            false,
			"tags":              false,
			"asset_tags":        false,
			"schema_migrations": false,
		}

		for _, t := range tables {
			if _, ok := expectedTables[t]; ok {
				expectedTables[t] = true
			}
		}

		missing := []string{}
		for t, found := range expectedTables {
			if !found {
				missing = append(missing, t)
			}
		}

		if len(missing) == 0 {
			fmt.Printf("[PASS] 2. All expected tables found: %s\n", strings.Join(tables, ", "))
		} else {
			fmt.Printf("[FAIL] 2. Missing tables: %s\n", strings.Join(missing, ", "))
			allPassed = false
		}
	}

	// 3. Verify Foreign Keys
	var fkStatus int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkStatus); err != nil || fkStatus != 1 {
		fmt.Printf("[FAIL] 3. PRAGMA foreign_keys is not enabled (status=%d, err=%v)\n", fkStatus, err)
		allPassed = false
	} else {
		fmt.Println("[PASS] 3a. PRAGMA foreign_keys is enabled in connection.")
	}

	// Inspect assets FK
	fkRows, err := db.Query("PRAGMA foreign_key_list(assets);")
	if err != nil {
		fmt.Printf("[FAIL] 3b. Inspecting assets foreign keys failed: %v\n", err)
		allPassed = false
	} else {
		defer fkRows.Close()
		foundCategoryFK := false
		for fkRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err == nil {
				if table == "categories" && from == "category_id" && to == "id" && onDelete == "SET NULL" {
					foundCategoryFK = true
				}
			}
		}
		if foundCategoryFK {
			fmt.Println("[PASS] 3b. assets.category_id -> categories(id) ON DELETE SET NULL configured.")
		} else {
			fmt.Println("[FAIL] 3b. assets.category_id foreign key constraint missing or misconfigured.")
			allPassed = false
		}
	}

	// Inspect asset_tags FKs
	atRows, err := db.Query("PRAGMA foreign_key_list(asset_tags);")
	if err != nil {
		fmt.Printf("[FAIL] 3c. Inspecting asset_tags foreign keys failed: %v\n", err)
		allPassed = false
	} else {
		defer atRows.Close()
		foundAssetFK := false
		foundTagFK := false
		for atRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := atRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err == nil {
				if table == "assets" && from == "asset_id" && to == "id" && onDelete == "CASCADE" {
					foundAssetFK = true
				}
				if table == "tags" && from == "tag_id" && to == "id" && onDelete == "CASCADE" {
					foundTagFK = true
				}
			}
		}
		if foundAssetFK && foundTagFK {
			fmt.Println("[PASS] 3c. asset_tags -> assets(id) & tags(id) ON DELETE CASCADE configured.")
		} else {
			fmt.Println("[FAIL] 3c. asset_tags foreign key constraints missing or misconfigured.")
			allPassed = false
		}
	}

	// 4. Verify Seeded Categories
	expectedSeeds := []string{
		"Avatar", "Hair", "Clothes", "Shoes", "Accessory",
		"Gimmick", "Texture", "Material", "Shader", "Other",
	}

	catRows, err := db.Query("SELECT id, name FROM categories ORDER BY id ASC")
	if err != nil {
		fmt.Printf("[FAIL] 4. Failed to query categories: %v\n", err)
		allPassed = false
	} else {
		defer catRows.Close()
		var foundSeeds []string
		for catRows.Next() {
			var id int
			var name string
			if err := catRows.Scan(&id, &name); err == nil {
				foundSeeds = append(foundSeeds, name)
			}
		}

		matches := true
		if len(foundSeeds) != len(expectedSeeds) {
			matches = false
		} else {
			for i, exp := range expectedSeeds {
				if foundSeeds[i] != exp {
					matches = false
					break
				}
			}
		}

		if matches {
			fmt.Printf("[PASS] 4. All 10 seeded categories verified: %s\n", strings.Join(foundSeeds, ", "))
		} else {
			fmt.Printf("[FAIL] 4. Seed categories mismatch. Found: %v, Expected: %v\n", foundSeeds, expectedSeeds)
			allPassed = false
		}
	}

	// 5. Verify Clean Migration from scratch
	tempDir, err := os.MkdirTemp("", "vram-clean-test-*")
	if err != nil {
		fmt.Printf("[FAIL] 5. Failed to create temp directory for clean test: %v\n", err)
		allPassed = false
	} else {
		defer os.RemoveAll(tempDir)
		cleanDBPath := filepath.Join(tempDir, "clean.db")
		cleanDB, err := database.Connect(cleanDBPath)
		if err != nil {
			fmt.Printf("[FAIL] 5. Connecting to clean database failed: %v\n", err)
			allPassed = false
		} else {
			defer cleanDB.Close()
			if err := cleanDB.Migrate(migrations.FS); err != nil {
				fmt.Printf("[FAIL] 5. Clean migration execution failed: %v\n", err)
				allPassed = false
			} else {
				// Verify clean db has all categories
				var cleanCount int
				_ = cleanDB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&cleanCount)
				if cleanCount == 10 {
					fmt.Println("[PASS] 5. Migrations execute cleanly and seed properly on a fresh database.")
				} else {
					fmt.Printf("[FAIL] 5. Clean database has %d categories, expected 10.\n", cleanCount)
					allPassed = false
				}
			}
		}
	}

	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// 6. Verify GET /api/health/db HTTP endpoint
	resp, err := http.Get(baseURL + "/api/health/db")
	if err != nil {
		fmt.Printf("[FAIL] 6. HTTP request to %s/api/health/db failed: %v\n", baseURL, err)
		allPassed = false
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusOK && strings.Contains(bodyStr, `"status":"ok"`) {
			fmt.Printf("[PASS] 6. GET %s/api/health/db responded %d OK with payload: %s\n", baseURL, resp.StatusCode, bodyStr)
		} else {
			fmt.Printf("[FAIL] 6. Unexpected response from %s/api/health/db (code=%d, body=%s)\n", baseURL, resp.StatusCode, bodyStr)
			allPassed = false
		}
	}

	// 7. Verify Milestone 2 Asset CRUD Lifecycle
	fmt.Println("\n--- Milestone 2 Asset CRUD Lifecycle Verification ---")
	client := &http.Client{}

	// 7.1 POST /api/assets (Create)
	createJSON := `{
		"name": "Verification Test Avatar",
		"category_id": 1,
		"author": "Verify Author",
		"booth_url": "https://booth.pm/en/items/123456",
		"local_path": "D:/Assets/TestAvatar",
		"description": "Verification test description",
		"tags": ["verification", "test"]
	}`
	postResp, err := client.Post(baseURL+"/api/assets", "application/json", strings.NewReader(createJSON))
	if err != nil || postResp.StatusCode != http.StatusCreated {
		fmt.Printf("[FAIL] 7.1 POST %s/api/assets failed: err=%v, code=%d\n", baseURL, err, postResp.StatusCode)
		allPassed = false
	} else {
		defer postResp.Body.Close()
		var created asset.Asset
		_ = json.NewDecoder(postResp.Body).Decode(&created)
		fmt.Printf("[PASS] 7.1 Created asset (ID: %d, Name: %q, Tags: %v)\n", created.ID, created.Name, created.Tags)

		// 7.2 GET /api/assets/:id (Retrieve)
		getURL := fmt.Sprintf("%s/api/assets/%d", baseURL, created.ID)
		getResp, err := client.Get(getURL)
		if err != nil || getResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 7.2 GET %s failed: err=%v, code=%d\n", getURL, err, getResp.StatusCode)
			allPassed = false
		} else {
			defer getResp.Body.Close()
			var retrieved asset.Asset
			_ = json.NewDecoder(getResp.Body).Decode(&retrieved)
			if retrieved.ID == created.ID && retrieved.Name == created.Name {
				fmt.Printf("[PASS] 7.2 Retrieved asset by ID (%d)\n", retrieved.ID)
			} else {
				fmt.Printf("[FAIL] 7.2 Retrieved asset mismatch: %+v\n", retrieved)
				allPassed = false
			}
		}

		// 7.3 GET /api/assets (Retrieve list)
		listResp, err := client.Get(baseURL + "/api/assets?search=Verification")
		if err != nil || listResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 7.3 GET /api/assets?search=Verification failed: err=%v, code=%d\n", err, listResp.StatusCode)
			allPassed = false
		} else {
			defer listResp.Body.Close()
			var list []asset.Asset
			_ = json.NewDecoder(listResp.Body).Decode(&list)
			if len(list) > 0 {
				fmt.Printf("[PASS] 7.3 Retrieved asset list containing %d matching item(s)\n", len(list))
			} else {
				fmt.Println("[FAIL] 7.3 Asset list did not contain created asset.")
				allPassed = false
			}
		}

		// 7.4 PUT /api/assets/:id (Update)
		updateJSON := `{
			"name": "Updated Verification Avatar",
			"category_id": 1,
			"author": "Updated Author",
			"booth_url": "https://booth.pm/en/items/654321",
			"tags": ["verification", "updated_tag"]
		}`
		req, _ := http.NewRequest(http.MethodPut, getURL, strings.NewReader(updateJSON))
		req.Header.Set("Content-Type", "application/json")
		putResp, err := client.Do(req)
		if err != nil || putResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 7.4 PUT %s failed: err=%v, code=%d\n", getURL, err, putResp.StatusCode)
			allPassed = false
		} else {
			defer putResp.Body.Close()
			var updated asset.Asset
			_ = json.NewDecoder(putResp.Body).Decode(&updated)
			fmt.Printf("[PASS] 7.4 Updated asset (Name: %q, Author: %q, Tags: %v)\n", updated.Name, updated.Author, updated.Tags)
		}

		// 7.5 Verify updated values
		getVerifyResp, err := client.Get(getURL)
		if err != nil || getVerifyResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 7.5 GET verification of updated asset failed: %v\n", err)
			allPassed = false
		} else {
			defer getVerifyResp.Body.Close()
			var verified asset.Asset
			_ = json.NewDecoder(getVerifyResp.Body).Decode(&verified)
			if verified.Name == "Updated Verification Avatar" && verified.Author == "Updated Author" && len(verified.Tags) == 2 {
				fmt.Println("[PASS] 7.5 Verified updated values persisted correctly.")
			} else {
				fmt.Printf("[FAIL] 7.5 Updated values mismatch: %+v\n", verified)
				allPassed = false
			}
		}

		// 7.6 DELETE /api/assets/:id (Delete)
		delReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
		delResp, err := client.Do(delReq)
		if err != nil || delResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 7.6 DELETE %s failed: err=%v, code=%d\n", getURL, err, delResp.StatusCode)
			allPassed = false
		} else {
			defer delResp.Body.Close()
			fmt.Printf("[PASS] 7.6 Deleted asset %d\n", created.ID)
		}

		// 7.7 Verify asset no longer exists
		postDelResp, err := client.Get(getURL)
		if err == nil && postDelResp.StatusCode == http.StatusNotFound {
			fmt.Println("[PASS] 7.7 Verified deleted asset returns 404 Not Found.")
		} else {
			fmt.Printf("[FAIL] 7.7 Expected 404 after deletion, got: %d, err=%v\n", postDelResp.StatusCode, err)
			allPassed = false
		}
	}

	// 8. Verify Milestone 4 Status & Folder Operations
	fmt.Println("\n--- Milestone 4 Detail & Filesystem Status Verification ---")
	{
		// 8.1 Create asset with current existing directory
		cwd, _ := os.Getwd()
		m4CreateJSON := fmt.Sprintf(`{
			"name": "M4 Filesystem Test Asset",
			"local_path": %q
		}`, cwd)
		m4PostResp, err := client.Post(baseURL+"/api/assets", "application/json", strings.NewReader(m4CreateJSON))
		if err != nil || m4PostResp.StatusCode != http.StatusCreated {
			fmt.Printf("[FAIL] 8.1 Failed to create M4 test asset: err=%v, code=%d\n", err, m4PostResp.StatusCode)
			allPassed = false
		} else {
			defer m4PostResp.Body.Close()
			var m4Asset asset.Asset
			_ = json.NewDecoder(m4PostResp.Body).Decode(&m4Asset)

			// 8.2 Verify GET /api/assets/:id/status returns exists: true for real path
			statusResp, err := client.Get(fmt.Sprintf("%s/api/assets/%d/status", baseURL, m4Asset.ID))
			if err != nil || statusResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 8.2 Failed to get status for existing path: err=%v, code=%d\n", err, statusResp.StatusCode)
				allPassed = false
			} else {
				defer statusResp.Body.Close()
				var st map[string]bool
				_ = json.NewDecoder(statusResp.Body).Decode(&st)
				if st["exists"] {
					fmt.Println("[PASS] 8.2 GET /api/assets/:id/status returned exists=true for valid path.")
				} else {
					fmt.Printf("[FAIL] 8.2 Expected exists=true, got %v\n", st["exists"])
					allPassed = false
				}
			}

			// Clean up M4 asset
			delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d", baseURL, m4Asset.ID), nil)
			_, _ = client.Do(delReq)
		}

		// 8.3 Verify missing path returns exists: false
		missingJSON := `{
			"name": "M4 Missing Path Asset",
			"local_path": "Z:/NonExistent/Directory/Path"
		}`
		missingPostResp, err := client.Post(baseURL+"/api/assets", "application/json", strings.NewReader(missingJSON))
		if err == nil && missingPostResp.StatusCode == http.StatusCreated {
			defer missingPostResp.Body.Close()
			var missingAsset asset.Asset
			_ = json.NewDecoder(missingPostResp.Body).Decode(&missingAsset)

			statusResp, err := client.Get(fmt.Sprintf("%s/api/assets/%d/status", baseURL, missingAsset.ID))
			if err == nil && statusResp.StatusCode == http.StatusOK {
				defer statusResp.Body.Close()
				var st map[string]bool
				_ = json.NewDecoder(statusResp.Body).Decode(&st)
				if !st["exists"] {
					fmt.Println("[PASS] 8.3 GET /api/assets/:id/status returned exists=false for missing path.")
				} else {
					fmt.Println("[FAIL] 8.3 Expected exists=false for missing path, got true.")
					allPassed = false
				}
			}

			// 8.4 Verify POST /api/assets/:id/open-folder on non-existent path returns 404
			openMissingResp, err := client.Post(fmt.Sprintf("%s/api/assets/%d/open-folder", baseURL, missingAsset.ID), "application/json", nil)
			if err == nil && openMissingResp.StatusCode == http.StatusNotFound {
				fmt.Println("[PASS] 8.4 POST /api/assets/:id/open-folder returned 404 for non-existent path on disk.")
			} else {
				fmt.Printf("[FAIL] 8.4 Expected 404 for missing path open-folder, got %d\n", openMissingResp.StatusCode)
				allPassed = false
			}

			// Clean up
			delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d", baseURL, missingAsset.ID), nil)
			_, _ = client.Do(delReq)
		}

		// 8.5 Verify 404 for non-existent asset ID on status endpoint
		st404Resp, err := client.Get(baseURL + "/api/assets/999999/status")
		if err == nil && st404Resp.StatusCode == http.StatusNotFound {
			fmt.Println("[PASS] 8.5 GET /api/assets/999999/status returned 404 Not Found.")
		} else {
			fmt.Printf("[FAIL] 8.5 Expected 404 for non-existent asset status, got %d\n", st404Resp.StatusCode)
			allPassed = false
		}
	}

	// 9. Verify Milestone 5 Preview Upload, Retrieval & Deletion
	fmt.Println("\n--- Milestone 5 Preview Image Infrastructure Verification ---")
	{
		// 9.1 Create test asset for preview testing
		pAssetJSON := `{
			"name": "M5 Verification Asset",
			"description": "Asset for testing preview infrastructure"
		}`
		pCreateResp, err := client.Post(baseURL+"/api/assets", "application/json", strings.NewReader(pAssetJSON))
		if err != nil || pCreateResp.StatusCode != http.StatusCreated {
			fmt.Printf("[FAIL] 9.1 Failed to create preview test asset: %v\n", err)
			allPassed = false
		} else {
			defer pCreateResp.Body.Close()
			var pAsset asset.Asset
			_ = json.NewDecoder(pCreateResp.Body).Decode(&pAsset)

			// 9.2 Verify GET /preview returns 404 when no preview exists
			pGetEmptyResp, err := client.Get(fmt.Sprintf("%s/api/assets/%d/preview", baseURL, pAsset.ID))
			if err == nil && pGetEmptyResp.StatusCode == http.StatusNotFound {
				fmt.Println("[PASS] 9.2 GET preview on asset without preview returned 404.")
			} else {
				fmt.Printf("[FAIL] 9.2 Expected 404 for missing preview, got %v\n", pGetEmptyResp)
				allPassed = false
			}

			// 9.3 Upload valid PNG preview
			pngBytes := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			part, _ := mw.CreateFormFile("file", "test.png")
			_, _ = part.Write(pngBytes)
			_ = mw.Close()

			pUploadReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/assets/%d/preview", baseURL, pAsset.ID), &buf)
			pUploadReq.Header.Set("Content-Type", mw.FormDataContentType())
			pUploadResp, err := client.Do(pUploadReq)
			if err != nil || pUploadResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 9.3 Failed to upload preview: err=%v, code=%d\n", err, pUploadResp.StatusCode)
				allPassed = false
			} else {
				defer pUploadResp.Body.Close()
				var updatedWithPreview asset.Asset
				_ = json.NewDecoder(pUploadResp.Body).Decode(&updatedWithPreview)
				if strings.HasSuffix(updatedWithPreview.PreviewPath, ".png") {
					fmt.Printf("[PASS] 9.3 Uploaded PNG preview (preview_path: %s).\n", updatedWithPreview.PreviewPath)
				} else {
					fmt.Printf("[FAIL] 9.3 Unexpected preview_path: %s\n", updatedWithPreview.PreviewPath)
					allPassed = false
				}
			}

			// 9.4 Retrieve uploaded preview via GET
			pGetResp, err := client.Get(fmt.Sprintf("%s/api/assets/%d/preview", baseURL, pAsset.ID))
			if err != nil || pGetResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 9.4 Failed to retrieve preview: %v\n", err)
				allPassed = false
			} else {
				defer pGetResp.Body.Close()
				contentType := pGetResp.Header.Get("Content-Type")
				if strings.Contains(contentType, "png") {
					fmt.Printf("[PASS] 9.4 Retrieved preview successfully (Content-Type: %s).\n", contentType)
				} else {
					fmt.Printf("[FAIL] 9.4 Expected image/png content type, got: %s\n", contentType)
					allPassed = false
				}
			}

			// 9.5 Delete preview via DELETE
			pDelReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d/preview", baseURL, pAsset.ID), nil)
			pDelResp, err := client.Do(pDelReq)
			if err != nil || pDelResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 9.5 Failed to delete preview: %v\n", err)
				allPassed = false
			} else {
				defer pDelResp.Body.Close()
				fmt.Println("[PASS] 9.5 DELETE /preview succeeded with 200 OK.")
			}

			// 9.6 Verify GET after delete returns 404
			pGetAfterDel, err := client.Get(fmt.Sprintf("%s/api/assets/%d/preview", baseURL, pAsset.ID))
			if err == nil && pGetAfterDel.StatusCode == http.StatusNotFound {
				fmt.Println("[PASS] 9.6 Verified preview returns 404 after deletion.")
			} else {
				fmt.Printf("[FAIL] 9.6 Expected 404 for deleted preview, got %d\n", pGetAfterDel.StatusCode)
				allPassed = false
			}

			// Clean up test asset
			delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d", baseURL, pAsset.ID), nil)
			_, _ = client.Do(delReq)
		}

		// --- Milestone 6 Asset Management UI & Delete Safety Verification ---
		fmt.Println("\n--- Milestone 6 Asset Management UI & Delete Safety Verification ---")

		// 10.1 Verify GET /api/tags
		tagsResp, err := client.Get(baseURL + "/api/tags")
		if err != nil || tagsResp.StatusCode != http.StatusOK {
			fmt.Printf("[FAIL] 10.1 GET /api/tags failed: %v\n", err)
			allPassed = false
		} else {
			defer tagsResp.Body.Close()
			var tags []asset.Tag
			if err := json.NewDecoder(tagsResp.Body).Decode(&tags); err == nil {
				fmt.Printf("[PASS] 10.1 GET /api/tags returned %d tag(s).\n", len(tags))
			} else {
				fmt.Printf("[FAIL] 10.1 Failed to parse tags response: %v\n", err)
				allPassed = false
			}
		}

		// 10.2 Verify POST /api/tags
		tagBody, _ := json.Marshal(asset.CreateTagRequest{Name: "M6VerifyTag"})
		createTagResp, err := client.Post(baseURL+"/api/tags", "application/json", bytes.NewReader(tagBody))
		if err != nil || createTagResp.StatusCode != http.StatusCreated {
			fmt.Printf("[FAIL] 10.2 POST /api/tags failed: %v\n", err)
			allPassed = false
		} else {
			defer createTagResp.Body.Close()
			var createdTag asset.Tag
			if err := json.NewDecoder(createTagResp.Body).Decode(&createdTag); err == nil && createdTag.Name == "M6VerifyTag" {
				fmt.Printf("[PASS] 10.2 POST /api/tags created tag: %s (ID: %d).\n", createdTag.Name, createdTag.ID)
			} else {
				fmt.Printf("[FAIL] 10.2 Unexpected tag created: %+v\n", createdTag)
				allPassed = false
			}
		}

		// 10.3 Delete Asset Safety Verification
		safeTestDir, err := os.MkdirTemp("", "vram-safety-check-*")
		if err == nil {
			defer os.RemoveAll(safeTestDir)
			safeSubdir := filepath.Join(safeTestDir, "Textures")
			_ = os.MkdirAll(safeSubdir, 0755)
			mockFbx := filepath.Join(safeTestDir, "avatar.fbx")
			_ = os.WriteFile(mockFbx, []byte("VRChat 3D Model Content"), 0644)
			mockPng := filepath.Join(safeSubdir, "texture.png")
			_ = os.WriteFile(mockPng, []byte("Texture Data"), 0644)

			safeAssetBody, _ := json.Marshal(asset.CreateAssetRequest{
				Name:      "Safety Check Asset",
				LocalPath: safeTestDir,
			})
			createSafeResp, err := client.Post(baseURL+"/api/assets", "application/json", bytes.NewReader(safeAssetBody))
			if err == nil && createSafeResp.StatusCode == http.StatusCreated {
				var safeAsset asset.Asset
				_ = json.NewDecoder(createSafeResp.Body).Decode(&safeAsset)
				createSafeResp.Body.Close()

				delSafeReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d", baseURL, safeAsset.ID), nil)
				delSafeResp, err := client.Do(delSafeReq)
				if err == nil && delSafeResp.StatusCode == http.StatusOK {
					delSafeResp.Body.Close()

					getSafeResp, _ := client.Get(fmt.Sprintf("%s/api/assets/%d", baseURL, safeAsset.ID))
					if getSafeResp.StatusCode == http.StatusNotFound {
						fmt.Println("[PASS] 10.3 Asset record deleted from database (404 on GET).")
					} else {
						fmt.Printf("[FAIL] 10.3 Asset record not deleted: %d\n", getSafeResp.StatusCode)
						allPassed = false
					}

					if _, err := os.Stat(safeTestDir); err == nil {
						fmt.Println("[PASS] 10.4 CRITICAL SAFETY: Local asset folder was NOT deleted.")
					} else {
						fmt.Println("[FAIL] 10.4 FATAL SAFETY VIOLATION: Local asset directory was deleted!")
						allPassed = false
					}

					if data, err := os.ReadFile(mockFbx); err == nil && string(data) == "VRChat 3D Model Content" {
						fmt.Println("[PASS] 10.5 CRITICAL SAFETY: Files in local_path are completely unmodified.")
					} else {
						fmt.Println("[FAIL] 10.5 FATAL SAFETY VIOLATION: Files in local_path were deleted or modified!")
						allPassed = false
					}
				}
			}
		}

		// 11. Milestone 7 Verification
		fmt.Println("--- Milestone 7 Verification ---")
		// 11.1 Create test asset for M7
		m7AssetBody, _ := json.Marshal(asset.CreateAssetRequest{
			Name:     "M7 Test Asset Alpha",
			Author:   "M7 Author Studio",
			BoothURL: "https://booth.pm/en/items/77777",
			Tags:     []string{"M7TagA", "M7TagB"},
		})
		m7Resp, err := client.Post(baseURL+"/api/assets", "application/json", bytes.NewReader(m7AssetBody))
		if err != nil || m7Resp.StatusCode != http.StatusCreated {
			fmt.Printf("[FAIL] 11.1 Failed to create M7 test asset: %v\n", err)
			allPassed = false
		} else {
			var m7Asset asset.Asset
			_ = json.NewDecoder(m7Resp.Body).Decode(&m7Asset)
			m7Resp.Body.Close()

			// 11.2 Toggle Favorite
			favResp, err := client.Post(fmt.Sprintf("%s/api/assets/%d/favorite", baseURL, m7Asset.ID), "application/json", nil)
			if err != nil || favResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 11.2 Failed to toggle favorite: %v\n", err)
				allPassed = false
			} else {
				var favAsset asset.Asset
				_ = json.NewDecoder(favResp.Body).Decode(&favAsset)
				favResp.Body.Close()
				if favAsset.IsFavorite {
					fmt.Println("[PASS] 11.2 Asset favorite toggled to true.")
				} else {
					fmt.Println("[FAIL] 11.2 Expected asset favorite to be true.")
					allPassed = false
				}
			}

			// 11.3 Filter by favorite=true
			favListResp, err := client.Get(baseURL + "/api/assets?favorite=true")
			if err != nil || favListResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 11.3 Failed to list favorites: %v\n", err)
				allPassed = false
			} else {
				var favList []asset.Asset
				_ = json.NewDecoder(favListResp.Body).Decode(&favList)
				favListResp.Body.Close()
				found := false
				for _, a := range favList {
					if a.ID == m7Asset.ID {
						found = true
						break
					}
				}
				if found {
					fmt.Println("[PASS] 11.3 Filter favorite=true returned the favorited asset.")
				} else {
					fmt.Println("[FAIL] 11.3 Filter favorite=true did not contain the favorited asset.")
					allPassed = false
				}
			}

			// 11.4 Search across tags and author
			searchResp, err := client.Get(baseURL + "/api/assets?search=M7TagA")
			if err != nil || searchResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 11.4 Failed search by tag: %v\n", err)
				allPassed = false
			} else {
				var searchList []asset.Asset
				_ = json.NewDecoder(searchResp.Body).Decode(&searchList)
				searchResp.Body.Close()
				if len(searchList) > 0 && searchList[0].ID == m7Asset.ID {
					fmt.Println("[PASS] 11.4 Search across tags successfully returned asset.")
				} else {
					fmt.Println("[FAIL] 11.4 Search across tags failed to return asset.")
					allPassed = false
				}
			}

			// 11.5 Multi-tag filter AND
			multiTagResp, err := client.Get(baseURL + "/api/assets?tags=M7TagA,M7TagB")
			if err != nil || multiTagResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 11.5 Failed multi-tag filter: %v\n", err)
				allPassed = false
			} else {
				var multiList []asset.Asset
				_ = json.NewDecoder(multiTagResp.Body).Decode(&multiList)
				multiTagResp.Body.Close()
				if len(multiList) > 0 && multiList[0].ID == m7Asset.ID {
					fmt.Println("[PASS] 11.5 Multi-tag filter (AND) successfully returned asset.")
				} else {
					fmt.Println("[FAIL] 11.5 Multi-tag filter failed.")
					allPassed = false
				}
			}

			// 11.6 Batch Status Check
			batchBody, _ := json.Marshal(asset.BatchStatusRequest{IDs: []int64{m7Asset.ID}})
			batchResp, err := client.Post(baseURL+"/api/assets/batch-status", "application/json", bytes.NewReader(batchBody))
			if err != nil || batchResp.StatusCode != http.StatusOK {
				fmt.Printf("[FAIL] 11.6 Batch status API failed: %v\n", err)
				allPassed = false
			} else {
				var bRes asset.BatchStatusResponse
				_ = json.NewDecoder(batchResp.Body).Decode(&bRes)
				batchResp.Body.Close()
				fmt.Println("[PASS] 11.6 Batch status API succeeded.")
			}

			// Clean up test asset
			delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/assets/%d", baseURL, m7Asset.ID), nil)
			_, _ = client.Do(delReq)
		}
	}

	fmt.Println("==================================================")
	if allPassed {
		fmt.Println("  ALL VERIFICATION CHECKS PASSED                  ")
	} else {
		fmt.Println("  SOME CHECKS FAILED                              ")
		os.Exit(1)
	}
	fmt.Println("==================================================")
}
