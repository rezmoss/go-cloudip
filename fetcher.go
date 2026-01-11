package cloudip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Default data URL (GitHub raw content)
	defaultDataURL    = "https://github.com/rezmoss/cloudip-db/raw/main/data/cloudip.msgpack"
	defaultVersionURL = "https://github.com/rezmoss/cloudip-db/raw/main/data/version.json"

	// Maximum download size (10 MB)
	maxDownloadSize = 10 * 1024 * 1024

	// Default timeout
	defaultTimeout = 30 * time.Second
)

// VersionInfo contains metadata about the data version.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime int64  `json:"build_time"`
	SHA256    string `json:"sha256"`
	Ranges    int    `json:"ranges"`
	Size      int64  `json:"size"`
	SizeGzip  int64  `json:"size_gzip"`
}

// fetchData downloads the MessagePack data from GitHub.
func fetchData(ctx context.Context, client *http.Client) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultDataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// Set headers for efficient download
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "go-cloudip/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit download size
	reader := io.LimitReader(resp.Body, maxDownloadSize)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	return data, nil
}

// fetchVersion downloads the version information.
func fetchVersion(ctx context.Context, client *http.Client) (*VersionInfo, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultVersionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-cloudip/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &info, nil
}

// CheckUpdate checks if a new version is available.
// Returns true if the remote version is newer than the local version.
func (d *Detector) CheckUpdate(ctx context.Context) (bool, *VersionInfo, error) {
	if d.opts.offline {
		return false, nil, fmt.Errorf("update check disabled in offline mode")
	}

	info, err := fetchVersion(ctx, d.opts.httpClient)
	if err != nil {
		return false, nil, err
	}

	state := d.state.Load()
	if state == nil {
		return true, info, nil
	}

	// Compare versions (newer if remote version string is greater)
	if info.Version > state.version {
		return true, info, nil
	}

	return false, info, nil
}
