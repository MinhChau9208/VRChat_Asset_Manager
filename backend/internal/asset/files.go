package asset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrPathTaken is returned when a file path is already linked to an asset.
	ErrPathTaken = errors.New("this path is already linked to an asset")
	// ErrFileNotFound is returned when the asset file does not exist.
	ErrFileNotFound = errors.New("asset file not found")
	// ErrAvatarNotFound is returned when a compatibility entry references a missing asset.
	ErrAvatarNotFound = errors.New("compatible avatar asset not found")
)

// sqlExecer is satisfied by both *sql.DB and *sql.Tx.
type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// inferKind guesses the file kind from its extension, falling back to the disk.
// Folder names often end in a version ("Kipfel_1.2.0"), so a numeric
// "extension" on a missing path is treated as a folder.
func inferKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip", ".7z", ".rar":
		return "archive"
	case ".unitypackage":
		return "unitypackage"
	}
	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			return "folder"
		}
		return "file"
	}
	if strings.Trim(ext, ".0123456789") == "" {
		return "folder"
	}
	return "file"
}

// syncPrimaryFile keeps asset_files in step with assets.local_path. Changing the
// path renames the matching file row; a path already linked elsewhere is left alone.
func syncPrimaryFile(ctx context.Context, q sqlExecer, assetID int64, oldPath, newPath string) error {
	oldPath, newPath = strings.TrimSpace(oldPath), strings.TrimSpace(newPath)
	if newPath == "" || fileRowExists(ctx, q, assetID, newPath) {
		return nil
	}

	if oldPath != "" && oldPath != newPath && fileRowExists(ctx, q, assetID, oldPath) {
		_, err := q.ExecContext(ctx,
			"UPDATE OR IGNORE asset_files SET path = ?, kind = ? WHERE asset_id = ? AND path = ?",
			newPath, inferKind(newPath), assetID, oldPath)
		if err != nil {
			return fmt.Errorf("failed to update primary file: %w", err)
		}
		return nil
	}

	_, err := q.ExecContext(ctx,
		"INSERT OR IGNORE INTO asset_files (asset_id, path, kind) VALUES (?, ?, ?)",
		assetID, newPath, inferKind(newPath))
	if err != nil {
		return fmt.Errorf("failed to link primary file: %w", err)
	}
	return nil
}

func fileRowExists(ctx context.Context, q sqlExecer, assetID int64, path string) bool {
	var exists bool
	_ = q.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM asset_files WHERE asset_id = ? AND path = ?)", assetID, path,
	).Scan(&exists)
	return exists
}

// ListFiles returns the files of an asset with their on-disk status.
func (r *Repository) ListFiles(ctx context.Context, assetID int64) ([]AssetFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, asset_id, path, kind, version, created_at
		FROM asset_files WHERE asset_id = ?
		ORDER BY version DESC, kind, path`, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query asset files: %w", err)
	}
	defer rows.Close()

	files := []AssetFile{}
	for rows.Next() {
		var f AssetFile
		if err := rows.Scan(&f.ID, &f.AssetID, &f.Path, &f.Kind, &f.Version, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan asset file: %w", err)
		}
		_, statErr := os.Stat(f.Path)
		f.Exists = statErr == nil
		files = append(files, f)
	}
	return files, rows.Err()
}

// AddFile links another file or folder to an asset. If the asset has no
// local_path yet, the new file becomes its primary location.
func (r *Repository) AddFile(ctx context.Context, assetID int64, req AddFileRequest) (*AssetFile, error) {
	var localPath string
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(local_path, '') FROM assets WHERE id = ?", assetID).Scan(&localPath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load asset: %w", err)
	}

	path := strings.TrimSpace(req.Path)
	kind := req.Kind
	if kind == "" {
		kind = inferKind(path)
	}

	var taken bool
	if err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM asset_files WHERE path = ?)", path).Scan(&taken); err != nil {
		return nil, fmt.Errorf("failed to check path: %w", err)
	}
	if taken {
		return nil, ErrPathTaken
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO asset_files (asset_id, path, kind, version) VALUES (?, ?, ?, ?)",
		assetID, path, kind, strings.TrimSpace(req.Version))
	if err != nil {
		return nil, fmt.Errorf("failed to add asset file: %w", err)
	}
	fileID, _ := res.LastInsertId()

	if strings.TrimSpace(localPath) == "" {
		if _, err := tx.ExecContext(ctx, "UPDATE assets SET local_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", path, assetID); err != nil {
			return nil, fmt.Errorf("failed to set primary path: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	files, err := r.ListFiles(ctx, assetID)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.ID == fileID {
			return &f, nil
		}
	}
	return nil, ErrFileNotFound
}

// DeleteFile unlinks a file from an asset (the file on disk is never touched).
// Removing the primary file promotes the next remaining file, if any.
func (r *Repository) DeleteFile(ctx context.Context, assetID, fileID int64) error {
	var path, localPath string
	err := r.db.QueryRowContext(ctx, `
		SELECT f.path, COALESCE(a.local_path, '')
		FROM asset_files f JOIN assets a ON a.id = f.asset_id
		WHERE f.id = ? AND f.asset_id = ?`, fileID, assetID).Scan(&path, &localPath)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFileNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to load asset file: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM asset_files WHERE id = ?", fileID); err != nil {
		return fmt.Errorf("failed to delete asset file: %w", err)
	}

	if strings.TrimSpace(localPath) == path {
		var next string
		err := tx.QueryRowContext(ctx,
			"SELECT path FROM asset_files WHERE asset_id = ? ORDER BY version DESC, id LIMIT 1", assetID,
		).Scan(&next)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to pick next primary file: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE assets SET local_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", next, assetID); err != nil {
			return fmt.Errorf("failed to update primary path: %w", err)
		}
	}
	return tx.Commit()
}

// syncCompat replaces an asset's compatibility list. Entries pointing to a
// library avatar take that asset's name; duplicates (by name) are dropped.
func syncCompat(ctx context.Context, q sqlExecer, assetID int64, avatars []CompatAvatar) error {
	if _, err := q.ExecContext(ctx, "DELETE FROM asset_compat WHERE asset_id = ?", assetID); err != nil {
		return fmt.Errorf("failed to clear compatibility: %w", err)
	}

	for _, c := range avatars {
		name := strings.TrimSpace(c.AvatarName)
		if c.AvatarAssetID != nil {
			if *c.AvatarAssetID == assetID {
				continue // an avatar is trivially compatible with itself
			}
			var avatarName string
			err := q.QueryRowContext(ctx, "SELECT name FROM assets WHERE id = ?", *c.AvatarAssetID).Scan(&avatarName)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAvatarNotFound
			}
			if err != nil {
				return fmt.Errorf("failed to load avatar: %w", err)
			}
			if name == "" {
				name = avatarName
			}
		}
		if name == "" {
			continue
		}
		_, err := q.ExecContext(ctx,
			"INSERT OR IGNORE INTO asset_compat (asset_id, avatar_asset_id, avatar_name) VALUES (?, ?, ?)",
			assetID, c.AvatarAssetID, name)
		if err != nil {
			return fmt.Errorf("failed to save compatibility: %w", err)
		}
	}
	return nil
}

// ListCompat returns the avatars an asset is compatible with.
func (r *Repository) ListCompat(ctx context.Context, assetID int64) ([]CompatAvatar, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT avatar_asset_id, avatar_name FROM asset_compat
		WHERE asset_id = ? ORDER BY avatar_name COLLATE NOCASE`, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query compatibility: %w", err)
	}
	defer rows.Close()

	avatars := []CompatAvatar{}
	for rows.Next() {
		var c CompatAvatar
		if err := rows.Scan(&c.AvatarAssetID, &c.AvatarName); err != nil {
			return nil, fmt.Errorf("failed to scan compatibility: %w", err)
		}
		avatars = append(avatars, c)
	}
	return avatars, rows.Err()
}
