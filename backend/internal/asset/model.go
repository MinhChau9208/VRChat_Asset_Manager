package asset

import (
	"encoding/json"
	"time"
)

// CategoryInfo represents simplified category details embedded in an asset.
type CategoryInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Tag represents a tag record in the database.
type Tag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTagRequest represents the incoming JSON body when creating a tag.
type CreateTagRequest struct {
	Name string `json:"name"`
}

// Asset represents a full VRChat asset entity.
type Asset struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	CategoryID  *int64        `json:"category_id"`
	Category    *CategoryInfo `json:"category,omitempty"`
	Author      string        `json:"author"`
	BoothURL    string        `json:"booth_url"`
	LocalPath   string        `json:"local_path"`
	PreviewPath string        `json:"preview_path"`
	Description string        `json:"description"`
	Tags        []string      `json:"tags"`
	IsFavorite  bool          `json:"is_favorite"`
	Status      string        `json:"status"`
	// ScanInfo explains scanner suggestions for drafts (sources, candidates).
	ScanInfo        json.RawMessage `json:"scan_info,omitempty"`
	LocalFileExists *bool           `json:"local_file_exists,omitempty"`
	// Files and CompatibleAvatars are only loaded for single-asset responses.
	Files             []AssetFile    `json:"files,omitempty"`
	CompatibleAvatars []CompatAvatar `json:"compatible_avatars,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// CreateAssetRequest represents the incoming JSON body when creating an asset.
type CreateAssetRequest struct {
	Name        string   `json:"name"`
	CategoryID  *int64   `json:"category_id"`
	Author      string   `json:"author"`
	BoothURL    string   `json:"booth_url"`
	LocalPath   string   `json:"local_path"`
	PreviewPath string   `json:"preview_path"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	IsFavorite  *bool    `json:"is_favorite"`
	Status      string   `json:"status"` // "active" (default) or "draft"
	// CompatibleAvatars replaces the asset's compatibility list.
	CompatibleAvatars []CompatAvatar `json:"compatible_avatars"`
}

// UpdateAssetRequest represents the incoming JSON body when updating an asset.
// PreviewPath is a pointer so that omitting it keeps the existing preview
// (previews are normally managed through the /preview endpoints).
type UpdateAssetRequest struct {
	Name        string   `json:"name"`
	CategoryID  *int64   `json:"category_id"`
	Author      string   `json:"author"`
	BoothURL    string   `json:"booth_url"`
	LocalPath   string   `json:"local_path"`
	PreviewPath *string  `json:"preview_path"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	IsFavorite  *bool    `json:"is_favorite"`
	Status      *string  `json:"status"` // omitted keeps the current status
	// CompatibleAvatars replaces the compatibility list; omitted (null) keeps it.
	CompatibleAvatars *[]CompatAvatar `json:"compatible_avatars"`
}

// FilterParams represents query options for listing assets.
type FilterParams struct {
	Category    string
	Tag         string   // single tag filter
	Tags        []string // multiple tags filter (AND semantics)
	Search      string
	Favorite    *bool
	HasPreview  *bool
	HasBooth    *bool
	LocalStatus string // "all", "available", "missing", "not_specified"
	Sort        string // "recent", "updated", "name_asc", "name_desc"
	Status      string // "active" (default), "draft", "all"
	// CompatibleWith limits results to assets compatible with this avatar asset id.
	CompatibleWith *int64
}

// ToggleFavoriteRequest represents the incoming JSON body when setting favorite status.
type ToggleFavoriteRequest struct {
	IsFavorite *bool `json:"is_favorite"`
}

// BatchStatusRequest represents the body when requesting existence status for multiple assets.
type BatchStatusRequest struct {
	IDs []int64 `json:"ids"`
}

// BatchStatusResponse represents the result of checking local file status for multiple assets.
type BatchStatusResponse struct {
	Statuses map[string]bool `json:"statuses"`
}

// LibraryStats summarizes the whole library, independent of any active filters.
type LibraryStats struct {
	Total      int            `json:"total"`
	Favorites  int            `json:"favorites"`
	Drafts     int            `json:"drafts"`
	ByCategory map[string]int `json:"by_category"` // category id -> asset count
}

// AssetFile is one folder, archive or package on disk that belongs to an asset.
type AssetFile struct {
	ID        int64     `json:"id"`
	AssetID   int64     `json:"asset_id"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind"` // folder | archive | unitypackage | file
	Version   string    `json:"version"`
	Exists    bool      `json:"exists"`
	CreatedAt time.Time `json:"created_at"`
}

// AddFileRequest is the body for POST /api/assets/{id}/files.
type AddFileRequest struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"` // inferred from the path when empty
	Version string `json:"version"`
}

// CompatAvatar links an asset to an avatar it works with. AvatarAssetID is set
// when the avatar is in the library; AvatarName is always filled in responses.
type CompatAvatar struct {
	AvatarAssetID *int64 `json:"avatar_asset_id"`
	AvatarName    string `json:"avatar_name"`
}
