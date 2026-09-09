package asset

import "time"

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
	ID              int64         `json:"id"`
	Name            string        `json:"name"`
	CategoryID      *int64        `json:"category_id"`
	Category        *CategoryInfo `json:"category,omitempty"`
	Author          string        `json:"author"`
	BoothURL        string        `json:"booth_url"`
	LocalPath       string        `json:"local_path"`
	PreviewPath     string        `json:"preview_path"`
	Description     string        `json:"description"`
	Tags            []string      `json:"tags"`
	IsFavorite      bool          `json:"is_favorite"`
	LocalFileExists *bool         `json:"local_file_exists,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
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
}

// UpdateAssetRequest represents the incoming JSON body when updating an asset.
type UpdateAssetRequest struct {
	Name        string   `json:"name"`
	CategoryID  *int64   `json:"category_id"`
	Author      string   `json:"author"`
	BoothURL    string   `json:"booth_url"`
	LocalPath   string   `json:"local_path"`
	PreviewPath string   `json:"preview_path"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	IsFavorite  *bool    `json:"is_favorite"`
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
