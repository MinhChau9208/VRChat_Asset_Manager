package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vrchat-asset-manager/backend/internal/asset"
)

const settingsKey = "scanner"

var (
	// ErrNoRoots is returned when scanning without any configured library root.
	ErrNoRoots = errors.New("no library folder configured")
	// ErrNotDraft is returned when an accept/ignore targets a non-draft asset.
	ErrNotDraft = errors.New("only draft assets can be accepted or ignored")
)

// Service applies scan plans to the database.
type Service struct {
	db          *sql.DB
	assets      *asset.Repository
	previewsDir string
}

// NewService creates a scanner service. previewsDir is where preview copies go.
func NewService(db *sql.DB, assets *asset.Repository, previewsDir string) *Service {
	return &Service{db: db, assets: assets, previewsDir: previewsDir}
}

// LoadConfig returns the saved scanner configuration, or the defaults.
func (s *Service) LoadConfig(ctx context.Context) (Config, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = ?", settingsKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("failed to load scanner config: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid scanner config: %w", err)
	}
	return cfg, nil
}

// SaveConfig validates and stores the scanner configuration.
func (s *Service) SaveConfig(ctx context.Context, cfg Config) (Config, error) {
	clean := func(in []string) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, v := range in {
			v = strings.TrimSpace(v)
			if v != "" && !seen[strings.ToLower(v)] {
				seen[strings.ToLower(v)] = true
				out = append(out, v)
			}
		}
		return out
	}
	cfg.Roots = clean(cfg.Roots)
	cfg.Ignore = clean(cfg.Ignore)
	cfg.ArchiveDirs = clean(cfg.ArchiveDirs)
	cfg.KnownDependencies = clean(cfg.KnownDependencies)
	folderMap := map[string]string{}
	for folder, category := range cfg.FolderMap {
		folder, category = strings.ToLower(strings.TrimSpace(folder)), strings.TrimSpace(category)
		if folder != "" && category != "" {
			folderMap[folder] = category
		}
	}
	cfg.FolderMap = folderMap

	data, err := json.Marshal(cfg)
	if err != nil {
		return Config{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		settingsKey, string(data))
	if err != nil {
		return Config{}, fmt.Errorf("failed to save scanner config: %w", err)
	}
	return cfg, nil
}

// AttachedFile reports a new file linked to an asset that already existed.
type AttachedFile struct {
	AssetID   int64  `json:"asset_id"`
	AssetName string `json:"asset_name"`
	Path      string `json:"path"`
}

// Result summarizes one scan.
type Result struct {
	Groups        int            `json:"groups"`
	Created       int            `json:"created"`
	Attached      []AttachedFile `json:"attached"`
	AlreadyLinked int            `json:"already_linked"`
	Ignored       int            `json:"ignored"`
	Warnings      []string       `json:"warnings"`
	DurationMs    int64          `json:"duration_ms"`
}

// BoothCandidate is a BOOTH item the scanner found for a draft.
type BoothCandidate struct {
	URL    string `json:"url"`
	Source string `json:"source"` // "folder name" or the readme / .url file name
}

// Info is stored in assets.scan_info so the review screen can explain suggestions.
type Info struct {
	ScannedAt       string           `json:"scanned_at"`
	BoothSource     string           `json:"booth_source,omitempty"`
	BoothCandidates []BoothCandidate `json:"booth_candidates,omitempty"`
	CompatReasons   []string         `json:"compat_reasons,omitempty"`
	PreviewSource   string           `json:"preview_source,omitempty"`
	CategorySource  string           `json:"category_source,omitempty"`
}

type avatarRef struct {
	id      int64
	name    string
	key     string
	boothID string
}

// Scan walks the configured roots and records what is new: unknown groups
// become drafts, new files of known assets are linked to them.
func (s *Service) Scan(ctx context.Context) (*Result, error) {
	start := time.Now()
	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if len(cfg.Roots) == 0 {
		return nil, ErrNoRoots
	}

	groups, warnings := Plan(cfg)
	result := &Result{Groups: len(groups), Attached: []AttachedFile{}, Warnings: append([]string{}, warnings...)}

	linked, err := s.linkedPaths(ctx)
	if err != nil {
		return nil, err
	}
	ignored, err := s.ignoredPaths(ctx)
	if err != nil {
		return nil, err
	}
	categories, err := s.categoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	avatars, err := s.avatars(ctx)
	if err != nil {
		return nil, err
	}
	deps := lowerSet(cfg.KnownDependencies)

	// Avatars first, so outfits scanned in the same run can reference new avatar drafts.
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].Category == "Avatar" && groups[j].Category != "Avatar"
	})

	for _, g := range groups {
		var fresh []Member
		var linkedID int64
		for _, m := range g.Members {
			p := strings.ToLower(m.Path)
			if id, ok := linked[p]; ok {
				if linkedID == 0 {
					linkedID = id
				}
				continue
			}
			if ignored[p] {
				result.Ignored++
				continue
			}
			fresh = append(fresh, m)
		}
		if len(fresh) == 0 {
			result.AlreadyLinked++
			continue
		}

		if linkedID != 0 {
			s.attach(ctx, linkedID, fresh, result)
			continue
		}

		id, err := s.createDraft(ctx, g, fresh, categories, avatars, deps, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", g.Name, err))
			continue
		}
		for _, m := range fresh {
			linked[strings.ToLower(m.Path)] = id
		}
		if g.Category == "Avatar" {
			avatars = append(avatars, avatarRef{id: id, name: g.Name, key: g.Key, boothID: g.BoothID})
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

// attach links new files (e.g. a newly downloaded version) to an existing asset.
func (s *Service) attach(ctx context.Context, assetID int64, members []Member, result *Result) {
	var name string
	_ = s.db.QueryRowContext(ctx, "SELECT name FROM assets WHERE id = ?", assetID).Scan(&name)
	for _, m := range members {
		_, err := s.assets.AddFile(ctx, assetID, asset.AddFileRequest{Path: m.Path, Kind: m.Kind, Version: m.Version})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", m.Path, err))
			continue
		}
		result.Attached = append(result.Attached, AttachedFile{AssetID: assetID, AssetName: name, Path: m.Path})
	}
}

func (s *Service) createDraft(
	ctx context.Context, g *Group, members []Member,
	categories map[string]int64, avatars []avatarRef, deps map[string]bool, result *Result,
) (int64, error) {
	info := Info{ScannedAt: time.Now().UTC().Format(time.RFC3339)}

	var categoryID *int64
	if id, ok := categories[strings.ToLower(g.Category)]; ok {
		categoryID = &id
		info.CategorySource = "folder"
	}

	// BOOTH: an id in the folder name wins; otherwise readme links, scored.
	avatarByBooth := map[string]avatarRef{}
	for _, a := range avatars {
		if a.boothID != "" {
			avatarByBooth[a.boothID] = a
		}
	}
	compat := map[int64]asset.CompatAvatar{}
	var boothURL string
	if g.BoothID != "" {
		boothURL = boothItemURL(g.BoothID)
		info.BoothSource = "folder name"
		info.BoothCandidates = append(info.BoothCandidates, BoothCandidate{URL: boothURL, Source: "folder name"})
	}
	score := map[string]int{}
	firstFile := map[string]string{}
	for _, l := range g.BoothLinks {
		if deps[l.ID] || l.ID == g.BoothID {
			continue
		}
		if a, ok := avatarByBooth[l.ID]; ok {
			if a.key != g.Key {
				compat[a.id] = asset.CompatAvatar{AvatarAssetID: &a.id, AvatarName: a.name}
				info.CompatReasons = append(info.CompatReasons, fmt.Sprintf("%s is linked in %s", a.name, l.File))
			}
			continue
		}
		score[l.ID]++
		if strings.EqualFold(filepath.Ext(l.File), ".url") {
			score[l.ID] += 2
		}
		if _, ok := firstFile[l.ID]; !ok {
			firstFile[l.ID] = l.File
		}
	}
	ids := make([]string, 0, len(score))
	for id := range score {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if score[ids[i]] != score[ids[j]] {
			return score[ids[i]] > score[ids[j]]
		}
		return ids[i] < ids[j]
	})
	for _, id := range ids {
		info.BoothCandidates = append(info.BoothCandidates, BoothCandidate{URL: boothItemURL(id), Source: firstFile[id]})
	}
	if boothURL == "" && len(ids) > 0 && (len(ids) == 1 || score[ids[0]] > score[ids[1]]) {
		boothURL = boothItemURL(ids[0])
		info.BoothSource = firstFile[ids[0]]
	}

	// Compatibility by name: "hamanosis_Small_Lady_Kipfel" mentions the Kipfel avatar.
	if g.Category != "Avatar" {
		for _, a := range avatars {
			if _, done := compat[a.id]; done || len(a.key) < 3 || a.key == g.Key {
				continue
			}
			if containsWord(g.Key, a.key) {
				compat[a.id] = asset.CompatAvatar{AvatarAssetID: &a.id, AvatarName: a.name}
				info.CompatReasons = append(info.CompatReasons, fmt.Sprintf("name mentions %s", a.name))
			}
		}
	}
	compatList := make([]asset.CompatAvatar, 0, len(compat))
	for _, c := range compat {
		compatList = append(compatList, c)
	}

	created, err := s.assets.Create(ctx, asset.CreateAssetRequest{
		Name:              g.Name,
		CategoryID:        categoryID,
		BoothURL:          boothURL,
		LocalPath:         members[0].Path,
		Status:            "draft",
		CompatibleAvatars: compatList,
	})
	if err != nil {
		return 0, err
	}

	// Create linked the primary path; record the planner's kind and version for it.
	_, _ = s.db.ExecContext(ctx, "UPDATE asset_files SET kind = ?, version = ? WHERE asset_id = ? AND path = ?",
		members[0].Kind, members[0].Version, created.ID, members[0].Path)
	for _, m := range members[1:] {
		if _, err := s.assets.AddFile(ctx, created.ID, asset.AddFileRequest{Path: m.Path, Kind: m.Kind, Version: m.Version}); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", m.Path, err))
		}
	}

	if g.PreviewPath != "" {
		if previewPath, err := s.copyPreview(created.ID, g.PreviewPath); err == nil {
			_ = s.assets.UpdatePreviewPath(ctx, created.ID, previewPath)
			info.PreviewSource = filepath.Base(g.PreviewPath)
		}
	}

	if data, err := json.Marshal(info); err == nil {
		_, _ = s.db.ExecContext(ctx, "UPDATE assets SET scan_info = ? WHERE id = ?", string(data), created.ID)
	}
	result.Created++
	return created.ID, nil
}

// copyPreview copies a local image into the previews directory (never the other way).
func (s *Service) copyPreview(assetID int64, src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return asset.SavePreviewImage(s.previewsDir, assetID, f)
}

// Accept turns drafts into regular library assets.
func (s *Service) Accept(ctx context.Context, ids []int64) (int, error) {
	if err := s.requireDrafts(ctx, ids); err != nil {
		return 0, err
	}
	accepted := 0
	for _, id := range ids {
		res, err := s.db.ExecContext(ctx,
			"UPDATE assets SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 'draft'", id)
		if err != nil {
			return accepted, fmt.Errorf("failed to accept draft %d: %w", id, err)
		}
		n, _ := res.RowsAffected()
		accepted += int(n)
	}
	return accepted, nil
}

// Ignore deletes drafts and remembers their paths so later scans skip them.
// Only the draft record and its preview copy are removed; files on disk are untouched.
func (s *Service) Ignore(ctx context.Context, ids []int64) (int, error) {
	if err := s.requireDrafts(ctx, ids); err != nil {
		return 0, err
	}
	ignored := 0
	for _, id := range ids {
		var previewPath string
		_ = s.db.QueryRowContext(ctx, "SELECT COALESCE(preview_path, '') FROM assets WHERE id = ?", id).Scan(&previewPath)

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return ignored, err
		}
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO scan_ignored (path) SELECT path FROM asset_files WHERE asset_id = ?", id)
		if err == nil {
			_, err = tx.ExecContext(ctx, "DELETE FROM assets WHERE id = ? AND status = 'draft'", id)
		}
		if err != nil {
			_ = tx.Rollback()
			return ignored, fmt.Errorf("failed to ignore draft %d: %w", id, err)
		}
		if err := tx.Commit(); err != nil {
			return ignored, err
		}
		ignored++

		// Same safety rule as asset deletion: only files inside previewsDir.
		if previewPath != "" {
			target := filepath.Join(s.previewsDir, filepath.Base(previewPath))
			if rel, err := filepath.Rel(s.previewsDir, target); err == nil && !strings.HasPrefix(rel, "..") {
				_ = os.Remove(target)
			}
		}
	}
	return ignored, nil
}

func (s *Service) requireDrafts(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		var status string
		err := s.db.QueryRowContext(ctx, "SELECT status FROM assets WHERE id = ?", id).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return asset.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "draft" {
			return ErrNotDraft
		}
	}
	return nil
}

// IgnoredPaths lists paths skipped by the scanner.
func (s *Service) IgnoredPaths(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT path FROM scan_ignored ORDER BY path")
	if err != nil {
		return nil, fmt.Errorf("failed to query ignored paths: %w", err)
	}
	defer rows.Close()
	paths := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// Unignore lets the scanner pick up a path again.
func (s *Service) Unignore(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM scan_ignored WHERE path = ?", path)
	return err
}

func (s *Service) linkedPaths(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT path, asset_id FROM asset_files")
	if err != nil {
		return nil, fmt.Errorf("failed to load linked paths: %w", err)
	}
	defer rows.Close()
	linked := map[string]int64{}
	for rows.Next() {
		var p string
		var id int64
		if err := rows.Scan(&p, &id); err != nil {
			return nil, err
		}
		linked[strings.ToLower(p)] = id
	}
	return linked, rows.Err()
}

func (s *Service) ignoredPaths(ctx context.Context) (map[string]bool, error) {
	paths, err := s.IgnoredPaths(ctx)
	if err != nil {
		return nil, err
	}
	return lowerSet(paths), nil
}

func (s *Service) categoryIDs(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM categories")
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}
	defer rows.Close()
	ids := map[string]int64{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		ids[strings.ToLower(name)] = id
	}
	return ids, rows.Err()
}

// avatars returns assets in the Avatar category (or its subcategories).
func (s *Service) avatars(ctx context.Context) ([]avatarRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name, COALESCE(a.booth_url, '')
		FROM assets a
		JOIN categories c ON a.category_id = c.id
		LEFT JOIN categories p ON c.parent_id = p.id
		WHERE c.name = 'Avatar' COLLATE NOCASE OR p.name = 'Avatar' COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("failed to load avatars: %w", err)
	}
	defer rows.Close()
	var avatars []avatarRef
	for rows.Next() {
		var a avatarRef
		var boothURL string
		if err := rows.Scan(&a.id, &a.name, &boothURL); err != nil {
			return nil, err
		}
		a.key = matchKey(a.name)
		a.boothID = boothIDFromURL(boothURL)
		avatars = append(avatars, a)
	}
	return avatars, rows.Err()
}
