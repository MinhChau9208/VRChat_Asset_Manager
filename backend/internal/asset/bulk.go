package asset

import (
	"context"
	"errors"
	"fmt"
)

// BulkUpdateRequest applies the same change to several assets. Only the
// fields that are set are changed; tags and avatars are added, never removed.
type BulkUpdateRequest struct {
	AssetIDs             []int64        `json:"asset_ids"`
	SetCategory          bool           `json:"set_category"`
	CategoryID           *int64         `json:"category_id"` // nil with set_category clears it
	AddTags              []string       `json:"add_tags"`
	AddCompatibleAvatars []CompatAvatar `json:"add_compatible_avatars"`
	IsFavorite           *bool          `json:"is_favorite"`
}

// BulkUpdate applies req to every asset in one transaction and returns how many were changed.
func (r *Repository) BulkUpdate(ctx context.Context, req BulkUpdateRequest) (int, error) {
	if req.SetCategory && req.CategoryID != nil {
		exists, err := r.CategoryExists(ctx, *req.CategoryID)
		if err != nil {
			return 0, fmt.Errorf("failed to verify category: %w", err)
		}
		if !exists {
			return 0, ErrCategoryNotFound
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range req.AssetIDs {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = ?)", id).Scan(&exists); err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("asset %d: %w", id, ErrNotFound)
		}

		if req.SetCategory {
			if _, err := tx.ExecContext(ctx, "UPDATE assets SET category_id = ? WHERE id = ?", req.CategoryID, id); err != nil {
				return 0, fmt.Errorf("failed to set category: %w", err)
			}
		}
		if req.IsFavorite != nil {
			if _, err := tx.ExecContext(ctx, "UPDATE assets SET is_favorite = ? WHERE id = ?", *req.IsFavorite, id); err != nil {
				return 0, fmt.Errorf("failed to set favorite: %w", err)
			}
		}
		if len(req.AddTags) > 0 {
			if err := addAssetTags(ctx, tx, id, req.AddTags); err != nil {
				return 0, err
			}
		}
		if len(req.AddCompatibleAvatars) > 0 {
			if err := addCompat(ctx, tx, id, req.AddCompatibleAvatars); err != nil {
				return 0, err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE assets SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", id); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit: %w", err)
	}
	return len(req.AssetIDs), nil
}

var errEmptyBulk = errors.New("asset_ids is required")
