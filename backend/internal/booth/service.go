package booth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/scanner"
)

// Service combines BOOTH lookups with the asset repository.
type Service struct {
	db          *sql.DB
	client      *Client
	assets      *asset.Repository
	previewsDir string
}

// NewService creates a BOOTH import service.
func NewService(db *sql.DB, client *Client, assets *asset.Repository, previewsDir string) *Service {
	return &Service{db: db, client: client, assets: assets, previewsDir: previewsDir}
}

// Lookup fetches an item by URL or id and returns suggestions for an asset.
func (s *Service) Lookup(ctx context.Context, input string, refresh bool) (*Suggestion, error) {
	id, err := ParseItemID(input)
	if err != nil {
		return nil, err
	}
	item, err := s.client.Fetch(ctx, id, refresh)
	if err != nil {
		return nil, err
	}
	categoryIDs, err := s.categoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	avatars, err := s.avatars(ctx)
	if err != nil {
		return nil, err
	}
	suggestion := Suggest(item, categoryIDs, avatars)
	return &suggestion, nil
}

// SetPreviewFromURL downloads a BOOTH image and makes it the asset's preview.
func (s *Service) SetPreviewFromURL(ctx context.Context, assetID int64, imageURL string) (*asset.Asset, error) {
	if _, err := s.assets.GetByID(ctx, assetID); err != nil {
		return nil, err
	}
	body, err := s.client.DownloadImage(ctx, imageURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	previewPath, err := asset.SavePreviewImage(s.previewsDir, assetID, body)
	if err != nil {
		return nil, err
	}
	if err := s.assets.UpdatePreviewPath(ctx, assetID, previewPath); err != nil {
		return nil, err
	}
	return s.assets.GetByID(ctx, assetID)
}

// ApplyResult reports what happened to one asset in a bulk apply.
type ApplyResult struct {
	AssetID int64    `json:"asset_id"`
	Name    string   `json:"name"`
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

// Apply fills assets from their BOOTH links:
//   - name ← BOOTH item name
//   - author ← shop name, only if empty
//   - category ← suggestion, only if none
//   - compatible avatars ← merged with suggestions
//   - preview ← first image, only if none
//   - tags ← merged with suggestions, only when includeTags
func (s *Service) Apply(ctx context.Context, ids []int64, includeTags bool) []ApplyResult {
	results := make([]ApplyResult, 0, len(ids))
	for _, id := range ids {
		res := ApplyResult{AssetID: id}
		changed, name, err := s.applyOne(ctx, id, includeTags)
		res.Name, res.Changed = name, changed
		if err != nil {
			res.Error = err.Error()
		} else {
			res.OK = true
		}
		results = append(results, res)
	}
	return results
}

func (s *Service) applyOne(ctx context.Context, id int64, includeTags bool) ([]string, string, error) {
	a, err := s.assets.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(a.BoothURL) == "" {
		return nil, a.Name, errors.New("no BOOTH link")
	}
	sug, err := s.Lookup(ctx, a.BoothURL, false)
	if err != nil {
		return nil, a.Name, err
	}

	var changed []string
	req := asset.UpdateAssetRequest{
		Name:        a.Name,
		CategoryID:  a.CategoryID,
		Author:      a.Author,
		BoothURL:    a.BoothURL,
		LocalPath:   a.LocalPath,
		Description: a.Description,
	}
	if sug.Name != "" && sug.Name != a.Name {
		req.Name = sug.Name
		changed = append(changed, "name")
	}
	if strings.TrimSpace(a.Author) == "" && sug.Author != "" {
		req.Author = sug.Author
		changed = append(changed, "author")
	}
	if a.CategoryID == nil && sug.CategoryID != nil {
		req.CategoryID = sug.CategoryID
		changed = append(changed, "category")
	}

	compat := append([]asset.CompatAvatar{}, a.CompatibleAvatars...)
	for _, c := range sug.CompatibleAvatars {
		dup := false
		for _, existing := range compat {
			if strings.EqualFold(existing.AvatarName, c.AvatarName) ||
				(c.AvatarAssetID != nil && existing.AvatarAssetID != nil && *c.AvatarAssetID == *existing.AvatarAssetID) {
				dup = true
				break
			}
		}
		if !dup {
			compat = append(compat, c)
			changed = append(changed, "compatible:"+c.AvatarName)
		}
	}
	req.CompatibleAvatars = &compat

	if includeTags {
		tags := append([]string{}, a.Tags...)
		for _, t := range sug.Tags {
			dup := false
			for _, existing := range tags {
				if strings.EqualFold(existing, t) {
					dup = true
					break
				}
			}
			if !dup {
				tags = append(tags, t)
				changed = append(changed, "tag:"+t)
			}
		}
		req.Tags = tags
	}

	if _, err := s.assets.Update(ctx, id, req); err != nil {
		return nil, a.Name, err
	}

	if a.PreviewPath == "" && len(sug.Images) > 0 {
		if _, err := s.SetPreviewFromURL(ctx, id, sug.Images[0]); err != nil {
			return changed, req.Name, fmt.Errorf("saved, but preview failed: %w", err)
		}
		changed = append(changed, "preview")
	}
	return changed, req.Name, nil
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
func (s *Service) avatars(ctx context.Context) ([]Avatar, error) {
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
	var avatars []Avatar
	for rows.Next() {
		var a Avatar
		var boothURL string
		if err := rows.Scan(&a.ID, &a.Name, &boothURL); err != nil {
			return nil, err
		}
		a.BoothID = scanner.BoothIDFromURL(boothURL)
		avatars = append(avatars, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// Aliases also come from the avatar's BOOTH name when it is cached
	// ("キプフェル Kipfel / オリジナル3Dモデル"), so Japanese descriptions match.
	for i := range avatars {
		names := []string{avatars[i].Name}
		if avatars[i].BoothID != "" {
			var data string
			if s.db.QueryRowContext(ctx, "SELECT data FROM booth_cache WHERE item_id = ?", avatars[i].BoothID).Scan(&data) == nil {
				var item Item
				if json.Unmarshal([]byte(data), &item) == nil {
					names = append(names, item.Name)
				}
			}
		}
		avatars[i].Aliases = AvatarAliases(names...)
	}
	return avatars, nil
}
