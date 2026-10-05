// Package update tells the UI whether a newer release exists on GitHub.
// It only notifies; nothing is downloaded or replaced.
//
// Only release builds with a version number (e.g. 1.2.0) check. Development
// builds ("dev", or a commit hash) never touch the network.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Info is the response of GET /api/version.
type Info struct {
	Version         string `json:"version"`
	Latest          string `json:"latest,omitempty"`
	URL             string `json:"url,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
}

// Checker asks GitHub for the latest release, at most once per cacheFor.
type Checker struct {
	current string
	apiURL  string
	client  *http.Client

	mu        sync.Mutex
	checkedAt time.Time
	latest    string
	url       string
}

const cacheFor = 6 * time.Hour

// NewChecker checks releases of github.com/<repo> ("owner/name").
func NewChecker(current, repo string) *Checker {
	return &Checker{
		current: current,
		apiURL:  "https://api.github.com/repos/" + repo + "/releases/latest",
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// SetAPIURL points the checker at another endpoint (tests).
func (c *Checker) SetAPIURL(url string) { c.apiURL = url }

// RegisterRoutes registers GET /api/version.
func (c *Checker) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(c.Info(r.Context()))
	})
}

// Info reports the running version and, when known, a newer release.
func (c *Checker) Info(ctx context.Context) Info {
	info := Info{Version: c.current}
	current, ok := parseVersion(c.current)
	if !ok {
		return info
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.checkedAt) > cacheFor {
		// A failed check also waits for the next period: being offline is normal.
		c.checkedAt = time.Now()
		if tag, url, err := c.fetchLatest(ctx); err == nil {
			c.latest, c.url = tag, url
		}
	}

	if latest, ok := parseVersion(c.latest); ok && newer(latest, current) {
		info.Latest = strings.TrimPrefix(c.latest, "v")
		info.URL = c.url
		info.UpdateAvailable = true
	}
	return info
}

func (c *Checker) fetchLatest(ctx context.Context) (tag, url string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "VRChat-Asset-Manager/"+c.current)

	res, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub returned %s", res.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		return "", "", err
	}
	return release.TagName, release.HTMLURL, nil
}

// parseVersion reads "1.2.3" or "v1.2" (missing parts are 0). Anything else,
// such as "dev" or "1.2.3-dirty", is not a release version.
func parseVersion(s string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func newer(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
