package asset

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MaxPreviewSize is the largest preview image accepted from any source.
const MaxPreviewSize = 10 << 20

// ErrUnsupportedImage is returned for images that are not JPEG, PNG or WebP.
var ErrUnsupportedImage = errors.New("unsupported image type: only JPEG, PNG and WebP are allowed")

// SavePreviewImage stores an image as the preview of an asset in previewsDir
// (as "<id>.<ext>", replacing an older preview with another extension) and
// returns the preview_path to store in the database.
func SavePreviewImage(previewsDir string, assetID int64, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxPreviewSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}
	if len(data) > MaxPreviewSize {
		return "", errors.New("image exceeds the 10MB limit")
	}

	var ext string
	switch http.DetectContentType(data) {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		if len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
			ext = ".webp"
		} else {
			return "", ErrUnsupportedImage
		}
	}

	if err := os.MkdirAll(previewsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create previews directory: %w", err)
	}
	name := fmt.Sprintf("%d%s", assetID, ext)
	if err := os.WriteFile(filepath.Join(previewsDir, name), data, 0644); err != nil {
		return "", fmt.Errorf("failed to save preview: %w", err)
	}

	// Remove a previous preview of this asset saved with another extension.
	for _, other := range []string{".jpg", ".png", ".webp"} {
		if other != ext {
			_ = os.Remove(filepath.Join(previewsDir, fmt.Sprintf("%d%s", assetID, other)))
		}
	}
	return "data/previews/" + strings.TrimPrefix(name, "/"), nil
}
