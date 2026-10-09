package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vrchat-asset-manager/backend/internal/asset"
)

// ErrMergeTarget is returned when a draft cannot be merged into the chosen asset.
var ErrMergeTarget = errors.New("choose another asset to merge into")

// MergeCandidate is an existing asset a draft probably belongs to.
type MergeCandidate struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`           // booth | file_name | name
	Source string `json:"source,omitempty"` // where the BOOTH id was found
}

// libAsset is what matching needs to know about an asset already in the library.
type libAsset struct {
	id       int64
	name     string
	family   string // top-level category name, lower-case ("" if none)
	boothID  string
	nameKey  string
	fileKeys map[string]bool // matchKey of each linked folder / file name
}

// library indexes the assets that existed before this scan.
type library struct {
	assets []*libAsset
	family map[string]string // category name (lower-case) -> top-level name (lower-case)
}

func (s *Service) loadLibrary(ctx context.Context) (*library, error) {
	lib := &library{family: map[string]string{}}
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.name, COALESCE(p.name, c.name) FROM categories c LEFT JOIN categories p ON c.parent_id = p.id`)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}
	for rows.Next() {
		var name, top string
		if err := rows.Scan(&name, &top); err != nil {
			rows.Close()
			return nil, err
		}
		lib.family[strings.ToLower(name)] = strings.ToLower(top)
	}
	rows.Close()

	byID := map[int64]*libAsset{}
	rows, err = s.db.QueryContext(ctx, `
		SELECT a.id, a.name, COALESCE(c.name, ''), COALESCE(a.booth_url, '')
		FROM assets a LEFT JOIN categories c ON a.category_id = c.id`)
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}
	for rows.Next() {
		a := &libAsset{fileKeys: map[string]bool{}}
		var category, boothURL string
		if err := rows.Scan(&a.id, &a.name, &category, &boothURL); err != nil {
			rows.Close()
			return nil, err
		}
		a.family = lib.familyOf(category)
		a.boothID = boothIDFromURL(boothURL)
		a.nameKey = matchKey(a.name)
		byID[a.id] = a
		lib.assets = append(lib.assets, a)
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, "SELECT asset_id, path FROM asset_files")
	if err != nil {
		return nil, fmt.Errorf("failed to load asset files: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		if a := byID[id]; a != nil {
			if key := matchKey(filepath.Base(filepath.Clean(path))); key != "" {
				a.fileKeys[key] = true
			}
		}
	}
	return lib, rows.Err()
}

func (l *library) familyOf(category string) string {
	if category == "" {
		return ""
	}
	if top, ok := l.family[strings.ToLower(category)]; ok {
		return top
	}
	return strings.ToLower(category)
}

// fits reports whether a group may belong to an asset by category: same
// top-level category (Clothes and Shoes are both Outfit) or one is unknown,
// and never an avatar with a non-avatar.
func (l *library) fits(g *Group, a *libAsset) bool {
	gf := l.familyOf(g.Category)
	if gf == "" || a.family == "" {
		return true
	}
	return gf == a.family
}

// match finds the existing asset a new group belongs to. A sure match (the
// BOOTH id in the folder name or a .url file, or the name of a folder or file
// the asset already had, e.g. an older version that was deleted) is returned
// as target; weaker ones (same display name, several sure matches) only as
// candidates for the review screen.
func (l *library) match(g *Group, deps map[string]bool) (target *libAsset, candidates []MergeCandidate) {
	add := func(seen map[int64]bool, list []*libAsset, a *libAsset) []*libAsset {
		if seen[a.id] {
			return list
		}
		seen[a.id] = true
		return append(list, a)
	}

	strongIDs := map[string]string{} // BOOTH id -> where it was found
	if g.BoothID != "" {
		strongIDs[g.BoothID] = "folder name"
	}
	for _, link := range g.BoothLinks {
		if strings.EqualFold(filepath.Ext(link.File), ".url") && !deps[link.ID] {
			if _, ok := strongIDs[link.ID]; !ok {
				strongIDs[link.ID] = link.File
			}
		}
	}

	var sure, weak []*libAsset
	reasons := map[int64]string{}
	seenSure, seenWeak := map[int64]bool{}, map[int64]bool{}
	for _, a := range l.assets {
		if !l.fits(g, a) {
			continue
		}
		switch {
		case a.boothID != "" && strongIDs[a.boothID] != "":
			sure = add(seenSure, sure, a)
			reasons[a.id] = "booth"
		case a.fileKeys[g.Key]:
			sure = add(seenSure, sure, a)
			reasons[a.id] = "file_name"
		case a.nameKey == g.Key:
			weak = add(seenWeak, weak, a)
			reasons[a.id] = "name"
		}
	}
	if len(sure) == 1 {
		return sure[0], nil
	}
	for _, a := range append(sure, weak...) {
		c := MergeCandidate{ID: a.id, Name: a.name, Reason: reasons[a.id]}
		if c.Reason == "booth" {
			c.Source = strongIDs[a.boothID]
		}
		candidates = append(candidates, c)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Name < candidates[j].Name })
	return nil, candidates
}

// Merge moves a draft's files into an existing asset and removes the draft.
// The target keeps its own data; its empty BOOTH link and preview are filled
// from the draft, and the draft's compatible avatars are added.
func (s *Service) Merge(ctx context.Context, draftID, targetID int64) (*asset.Asset, error) {
	if draftID == targetID {
		return nil, ErrMergeTarget
	}
	if err := s.requireDrafts(ctx, []int64{draftID}); err != nil {
		return nil, err
	}
	draft, err := s.assets.GetByID(ctx, draftID)
	if err != nil {
		return nil, err
	}
	target, err := s.assets.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	steps := []struct {
		query string
		args  []any
	}{
		{"UPDATE asset_files SET asset_id = ? WHERE asset_id = ?", []any{targetID, draftID}},
		{`INSERT OR IGNORE INTO asset_compat (asset_id, avatar_asset_id, avatar_name)
		  SELECT ?, avatar_asset_id, avatar_name FROM asset_compat
		  WHERE asset_id = ? AND (avatar_asset_id IS NULL OR avatar_asset_id != ?)`, []any{targetID, draftID, targetID}},
		{`UPDATE assets SET
		    local_path = CASE WHEN COALESCE(local_path, '') = '' THEN ? ELSE local_path END,
		    booth_url = CASE WHEN COALESCE(booth_url, '') = '' THEN ? ELSE booth_url END,
		    updated_at = CURRENT_TIMESTAMP
		  WHERE id = ?`, []any{draft.LocalPath, draft.BoothURL, targetID}},
		{"DELETE FROM assets WHERE id = ? AND status = 'draft'", []any{draftID}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.args...); err != nil {
			return nil, fmt.Errorf("failed to merge draft %d: %w", draftID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if draft.PreviewPath != "" {
		src := filepath.Join(s.previewsDir, filepath.Base(draft.PreviewPath))
		if target.PreviewPath == "" {
			if previewPath, err := s.copyPreview(targetID, src); err == nil {
				_ = s.assets.UpdatePreviewPath(ctx, targetID, previewPath)
			}
		}
		_ = os.Remove(src)
	}
	return s.assets.GetByID(ctx, targetID)
}
