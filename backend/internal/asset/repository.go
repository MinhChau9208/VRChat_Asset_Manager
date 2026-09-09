package asset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrNotFound is returned when the requested asset does not exist.
	ErrNotFound = errors.New("asset not found")
	// ErrCategoryNotFound is returned when category_id does not refer to an existing category.
	ErrCategoryNotFound = errors.New("category not found")
)

// Repository handles database interactions for assets and their tags.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new asset repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CategoryExists checks if a category exists by its ID.
func (r *Repository) CategoryExists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)"
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

// normalizeTags trims, cleans, and deduplicates tag names.
func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range tags {
		clean := strings.TrimSpace(t)
		if clean == "" {
			continue
		}
		lower := strings.ToLower(clean)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, clean)
		}
	}
	if result == nil {
		result = []string{}
	}
	return result
}

// syncAssetTags ensures tags exist in the tags table and links them to the asset.
func (r *Repository) syncAssetTags(ctx context.Context, tx *sql.Tx, assetID int64, tags []string) ([]string, error) {
	normalized := normalizeTags(tags)

	// Remove existing tag associations for this asset
	if _, err := tx.ExecContext(ctx, "DELETE FROM asset_tags WHERE asset_id = ?", assetID); err != nil {
		return nil, fmt.Errorf("failed to clear asset tags: %w", err)
	}

	for _, tagName := range normalized {
		// Check if tag already exists case-insensitively
		var tagID int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ? COLLATE NOCASE", tagName).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.ExecContext(ctx, "INSERT INTO tags (name) VALUES (?)", tagName)
			if err != nil {
				return nil, fmt.Errorf("failed to insert tag %q: %w", tagName, err)
			}
			tagID, err = res.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("failed to get tag ID for %q: %w", tagName, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to check tag %q: %w", tagName, err)
		}

		// Link tag to asset
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_tags (asset_id, tag_id) VALUES (?, ?)", assetID, tagID)
		if err != nil {
			return nil, fmt.Errorf("failed to link tag %q to asset: %w", tagName, err)
		}
	}

	return normalized, nil
}

// Create inserts a new asset and associates its tags in a single transaction.
func (r *Repository) Create(ctx context.Context, req CreateAssetRequest) (*Asset, error) {
	if req.CategoryID != nil {
		exists, err := r.CategoryExists(ctx, *req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify category: %w", err)
		}
		if !exists {
			return nil, ErrCategoryNotFound
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	isFav := false
	if req.IsFavorite != nil {
		isFav = *req.IsFavorite
	}

	insertQuery := `
		INSERT INTO assets (name, category_id, author, booth_url, local_path, preview_path, description, is_favorite)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := tx.ExecContext(ctx, insertQuery,
		req.Name,
		req.CategoryID,
		req.Author,
		req.BoothURL,
		req.LocalPath,
		req.PreviewPath,
		req.Description,
		isFav,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert asset: %w", err)
	}

	assetID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset ID: %w", err)
	}

	if _, err := r.syncAssetTags(ctx, tx, assetID, req.Tags); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(ctx, assetID)
}

// GetByID retrieves a single asset with category and tags by ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Asset, error) {
	query := `
		SELECT 
			a.id, a.name, a.category_id, c.name,
			a.author, a.booth_url, a.local_path, a.preview_path, a.description,
			a.is_favorite,
			a.created_at, a.updated_at
		FROM assets a
		LEFT JOIN categories c ON a.category_id = c.id
		WHERE a.id = ?
	`
	var a Asset
	var catName sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.Name,
		&a.CategoryID,
		&catName,
		&a.Author,
		&a.BoothURL,
		&a.LocalPath,
		&a.PreviewPath,
		&a.Description,
		&a.IsFavorite,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query asset: %w", err)
	}

	if a.LocalPath != "" {
		_, err := os.Stat(a.LocalPath)
		exists := (err == nil)
		a.LocalFileExists = &exists
	}

	if a.CategoryID != nil && catName.Valid {
		a.Category = &CategoryInfo{
			ID:   *a.CategoryID,
			Name: catName.String,
		}
	}

	// Fetch tags
	tagQuery := `
		SELECT t.name 
		FROM tags t 
		JOIN asset_tags at ON t.id = at.tag_id 
		WHERE at.asset_id = ? 
		ORDER BY t.name ASC
	`
	rows, err := r.db.QueryContext(ctx, tagQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query asset tags: %w", err)
	}
	defer rows.Close()

	a.Tags = []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			a.Tags = append(a.Tags, tag)
		}
	}

	return &a, nil
}

// List returns a list of assets matching optional filters.
func (r *Repository) List(ctx context.Context, filters FilterParams) ([]Asset, error) {
	var conditions []string
	var args []any

	if filters.Search != "" {
		conditions = append(conditions, `(
			a.name LIKE ? OR a.author LIKE ? OR a.description LIKE ? OR EXISTS (
				SELECT 1 FROM asset_tags at_s 
				JOIN tags t_s ON at_s.tag_id = t_s.id 
				WHERE at_s.asset_id = a.id AND t_s.name LIKE ?
			)
		)`)
		searchTerm := "%" + filters.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if filters.Category != "" && strings.ToLower(filters.Category) != "all" {
		if catID, err := strconv.ParseInt(filters.Category, 10, 64); err == nil {
			conditions = append(conditions, "(a.category_id = ? OR c.name = ? COLLATE NOCASE)")
			args = append(args, catID, filters.Category)
		} else {
			conditions = append(conditions, "c.name = ? COLLATE NOCASE")
			args = append(args, filters.Category)
		}
	}

	// Combine single Tag and multiple Tags slice with AND semantics
	allTags := make([]string, 0, len(filters.Tags)+1)
	if filters.Tag != "" {
		allTags = append(allTags, filters.Tag)
	}
	for _, t := range filters.Tags {
		clean := strings.TrimSpace(t)
		if clean != "" {
			allTags = append(allTags, clean)
		}
	}
	seenTags := make(map[string]bool)
	for _, t := range allTags {
		lower := strings.ToLower(t)
		if !seenTags[lower] {
			seenTags[lower] = true
			conditions = append(conditions, `EXISTS (
				SELECT 1 FROM asset_tags at_t 
				JOIN tags t_t ON at_t.tag_id = t_t.id 
				WHERE at_t.asset_id = a.id AND t_t.name = ? COLLATE NOCASE
			)`)
			args = append(args, t)
		}
	}

	if filters.Favorite != nil {
		if *filters.Favorite {
			conditions = append(conditions, "a.is_favorite = 1")
		} else {
			conditions = append(conditions, "a.is_favorite = 0")
		}
	}

	if filters.HasPreview != nil {
		if *filters.HasPreview {
			conditions = append(conditions, "(a.preview_path IS NOT NULL AND a.preview_path != '')")
		} else {
			conditions = append(conditions, "(a.preview_path IS NULL OR a.preview_path = '')")
		}
	}

	if filters.HasBooth != nil {
		if *filters.HasBooth {
			conditions = append(conditions, "(a.booth_url IS NOT NULL AND a.booth_url != '')")
		} else {
			conditions = append(conditions, "(a.booth_url IS NULL OR a.booth_url = '')")
		}
	}

	switch strings.ToLower(filters.LocalStatus) {
	case "not_specified":
		conditions = append(conditions, "(a.local_path IS NULL OR a.local_path = '')")
	case "available", "missing":
		conditions = append(conditions, "(a.local_path IS NOT NULL AND a.local_path != '')")
	}

	orderBy := "a.created_at DESC, a.id DESC"
	switch strings.ToLower(filters.Sort) {
	case "updated", "updated_desc":
		orderBy = "a.updated_at DESC, a.id DESC"
	case "name_asc":
		orderBy = "a.name COLLATE NOCASE ASC, a.id ASC"
	case "name_desc":
		orderBy = "a.name COLLATE NOCASE DESC, a.id DESC"
	case "recent", "created_desc":
		orderBy = "a.created_at DESC, a.id DESC"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.name, a.category_id, c.name,
			a.author, a.booth_url, a.local_path, a.preview_path, a.description,
			a.is_favorite,
			a.created_at, a.updated_at
		FROM assets a
		LEFT JOIN categories c ON a.category_id = c.id
		%s
		ORDER BY %s
	`, whereClause, orderBy)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	var assetIDs []int64
	assetMap := make(map[int64]*Asset)

	for rows.Next() {
		var a Asset
		var catName sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.CategoryID,
			&catName,
			&a.Author,
			&a.BoothURL,
			&a.LocalPath,
			&a.PreviewPath,
			&a.Description,
			&a.IsFavorite,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset row: %w", err)
		}

		if a.CategoryID != nil && catName.Valid {
			a.Category = &CategoryInfo{
				ID:   *a.CategoryID,
				Name: catName.String,
			}
		}
		a.Tags = []string{}
		assets = append(assets, a)
		assetIDs = append(assetIDs, a.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(assets) == 0 {
		return []Asset{}, nil
	}

	// Map pointers to populate tags efficiently
	for i := range assets {
		assetMap[assets[i].ID] = &assets[i]
	}

	// Batch query tags for all listed assets
	placeholders := make([]string, len(assetIDs))
	tagArgs := make([]any, len(assetIDs))
	for i, id := range assetIDs {
		placeholders[i] = "?"
		tagArgs[i] = id
	}

	tagsQuery := fmt.Sprintf(`
		SELECT at.asset_id, t.name
		FROM asset_tags at
		JOIN tags t ON at.tag_id = t.id
		WHERE at.asset_id IN (%s)
		ORDER BY t.name ASC
	`, strings.Join(placeholders, ","))

	tagRows, err := r.db.QueryContext(ctx, tagsQuery, tagArgs...)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var assetID int64
			var tagName string
			if err := tagRows.Scan(&assetID, &tagName); err == nil {
				if a, ok := assetMap[assetID]; ok {
					a.Tags = append(a.Tags, tagName)
				}
			}
		}
	}

	// Compute LocalFileExists and filter by LocalStatus if needed
	var filteredAssets []Asset
	for i := range assets {
		if assets[i].LocalPath != "" {
			_, err := os.Stat(assets[i].LocalPath)
			exists := (err == nil)
			assets[i].LocalFileExists = &exists

			if strings.ToLower(filters.LocalStatus) == "available" && !exists {
				continue
			}
			if strings.ToLower(filters.LocalStatus) == "missing" && exists {
				continue
			}
		} else {
			if strings.ToLower(filters.LocalStatus) == "available" || strings.ToLower(filters.LocalStatus) == "missing" {
				continue
			}
		}
		filteredAssets = append(filteredAssets, assets[i])
	}
	if filteredAssets == nil {
		filteredAssets = []Asset{}
	}

	return filteredAssets, nil
}

// Update modifies an existing asset and updates its associated tags.
func (r *Repository) Update(ctx context.Context, id int64, req UpdateAssetRequest) (*Asset, error) {
	// Check if asset exists
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check asset existence: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	if req.CategoryID != nil {
		catExists, err := r.CategoryExists(ctx, *req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify category: %w", err)
		}
		if !catExists {
			return nil, ErrCategoryNotFound
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var isFavVal any = nil
	if req.IsFavorite != nil {
		isFavVal = *req.IsFavorite
	}

	updateQuery := `
		UPDATE assets 
		SET name = ?, category_id = ?, author = ?, booth_url = ?, local_path = ?, preview_path = ?, description = ?,
		    is_favorite = COALESCE(?, is_favorite),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, updateQuery,
		req.Name,
		req.CategoryID,
		req.Author,
		req.BoothURL,
		req.LocalPath,
		req.PreviewPath,
		req.Description,
		isFavVal,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	if req.Tags != nil {
		if _, err := r.syncAssetTags(ctx, tx, id, req.Tags); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Delete removes an asset by its ID.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM assets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdatePreviewPath updates only the preview_path and updated_at timestamp of an asset.
func (r *Repository) UpdatePreviewPath(ctx context.Context, id int64, previewPath string) error {
	res, err := r.db.ExecContext(ctx, "UPDATE assets SET preview_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", previewPath, id)
	if err != nil {
		return fmt.Errorf("failed to update preview path: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetTags returns all tags ordered by name.
func (r *Repository) GetTags(ctx context.Context) ([]Tag, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, created_at FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}
	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

// CreateTag inserts a new tag if it does not already exist, or returns the existing tag.
func (r *Repository) CreateTag(ctx context.Context, name string) (*Tag, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	// First check if a tag with this name already exists case-insensitively
	var existing Tag
	err := r.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM tags WHERE name = ? COLLATE NOCASE", cleanName).Scan(&existing.ID, &existing.Name, &existing.CreatedAt)
	if err == nil {
		return &existing, nil
	}

	res, err := r.db.ExecContext(ctx, "INSERT INTO tags (name) VALUES (?)", cleanName)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tag: %w", err)
	}

	tagID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	var tag Tag
	err = r.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM tags WHERE id = ?", tagID).Scan(&tag.ID, &tag.Name, &tag.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created tag: %w", err)
	}

	return &tag, nil
}

// ToggleFavorite sets or toggles the is_favorite flag on an asset.
func (r *Repository) ToggleFavorite(ctx context.Context, id int64, isFav *bool) (*Asset, error) {
	var query string
	var args []any
	if isFav != nil {
		query = "UPDATE assets SET is_favorite = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
		args = []any{*isFav, id}
	} else {
		query = "UPDATE assets SET is_favorite = CASE WHEN is_favorite = 1 THEN 0 ELSE 1 END, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
		args = []any{id}
	}

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update favorite status: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

// BatchStatus checks file existence for a list of asset IDs and returns a map of string(id) -> exists.
func (r *Repository) BatchStatus(ctx context.Context, ids []int64) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id, local_path FROM assets WHERE id IN (%s)", strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query asset paths: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id int64
		var localPath string
		if err := rows.Scan(&id, &localPath); err == nil {
			idStr := strconv.FormatInt(id, 10)
			if localPath == "" {
				result[idStr] = false
			} else {
				_, statErr := os.Stat(localPath)
				result[idStr] = (statErr == nil)
			}
		}
	}

	return result, nil
}

