// Package booth fetches item metadata from BOOTH (booth.pm) on user request
// and turns it into suggestions for an asset.
package booth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"vrchat-asset-manager/backend/internal/scanner"
)

var (
	// ErrInvalidURL is returned when no BOOTH item id can be found in the input.
	ErrInvalidURL = errors.New("not a BOOTH item URL or id")
	// ErrItemNotFound is returned when BOOTH has no such item (deleted or private).
	ErrItemNotFound = errors.New("BOOTH item not found (it may have been removed)")
	// ErrImageHost is returned for image URLs outside BOOTH's image hosts.
	ErrImageHost = errors.New("only BOOTH images (pximg.net) can be downloaded")
)

const (
	cacheTTL    = 7 * 24 * time.Hour
	minInterval = time.Second // at most one request per second to BOOTH
	userAgent   = "Mozilla/5.0 (VRChat Asset Manager; personal library)"
)

var bareIDRe = regexp.MustCompile(`^\d{3,10}$`)

// Item is the subset of BOOTH's item JSON the app uses.
type Item struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	IsAdult     bool   `json:"is_adult"`
	Price       string `json:"price"`
	Category    struct {
		Name   string `json:"name"`
		Parent struct {
			Name string `json:"name"`
		} `json:"parent"`
	} `json:"category"`
	Shop struct {
		Name      string `json:"name"`
		Subdomain string `json:"subdomain"`
		URL       string `json:"url"`
	} `json:"shop"`
	Images []struct {
		Original string `json:"original"`
		Resized  string `json:"resized"`
	} `json:"images"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
	Variations []struct {
		Name string `json:"name"`
	} `json:"variations"`
}

// Client fetches BOOTH items, caching them in the database.
type Client struct {
	db      *sql.DB
	http    *http.Client
	baseURL string // https://booth.pm, overridable in tests

	// allowImageHost decides which hosts images may be downloaded from.
	allowImageHost func(host string) bool

	mu   sync.Mutex
	last time.Time
}

// NewClient creates a BOOTH client using db for caching.
func NewClient(db *sql.DB) *Client {
	return &Client{
		db:      db,
		http:    &http.Client{Timeout: 20 * time.Second},
		baseURL: "https://booth.pm",
		allowImageHost: func(host string) bool {
			return host == "pximg.net" || strings.HasSuffix(host, ".pximg.net")
		},
	}
}

// ParseItemID accepts a BOOTH item URL (any shop subdomain or language) or a bare id.
func ParseItemID(input string) (string, error) {
	input = strings.TrimSpace(input)
	if bareIDRe.MatchString(input) {
		return input, nil
	}
	if id := scanner.BoothIDFromURL(input); id != "" {
		return id, nil
	}
	return "", ErrInvalidURL
}

// wait enforces the minimum interval between requests to BOOTH.
func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d := minInterval - time.Since(c.last); d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	c.last = time.Now()
	return nil
}

// Fetch returns item metadata, from the cache when fresh unless refresh is set.
func (c *Client) Fetch(ctx context.Context, id string, refresh bool) (*Item, error) {
	if !refresh {
		var data string
		var fetchedAt time.Time
		err := c.db.QueryRowContext(ctx, "SELECT data, fetched_at FROM booth_cache WHERE item_id = ?", id).Scan(&data, &fetchedAt)
		if err == nil && time.Since(fetchedAt) < cacheTTL {
			var item Item
			if json.Unmarshal([]byte(data), &item) == nil {
				return &item, nil
			}
		}
	}

	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ja/items/"+id+".json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach BOOTH: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrItemNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BOOTH returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read BOOTH response: %w", err)
	}
	var item Item
	if err := json.Unmarshal(data, &item); err != nil || item.ID == 0 {
		return nil, errors.New("unexpected response from BOOTH (adult items may need an age confirmation)")
	}

	_, _ = c.db.ExecContext(ctx, `
		INSERT INTO booth_cache (item_id, data, fetched_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(item_id) DO UPDATE SET data = excluded.data, fetched_at = CURRENT_TIMESTAMP`,
		id, string(data))
	return &item, nil
}

// DownloadImage fetches a BOOTH image. The caller stores it.
func (c *Client) DownloadImage(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || !c.allowImageHost(u.Hostname()) {
		return nil, ErrImageHost
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not download image: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("image download returned HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}
