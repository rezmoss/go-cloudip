package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultDataURL is the default data URL (GitHub raw content).
	DefaultDataURL = "https://github.com/rezmoss/cloudip-db/raw/main/data/cloudip.msgpack"

	// DefaultVersionURL is the default version URL.
	DefaultVersionURL = "https://github.com/rezmoss/cloudip-db/raw/main/data/version.json"

	// MaxDownloadSize limits downloads to 10 MB.
	MaxDownloadSize = 10 * 1024 * 1024

	// DefaultTimeout for HTTP requests.
	DefaultTimeout = 30 * time.Second
)

// FetchData downloads the MessagePack data from the given URL.
func FetchData(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	if url == "" {
		url = DefaultDataURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	reader := io.LimitReader(resp.Body, MaxDownloadSize)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	return data, nil
}

// FetchVersion downloads the version information from the given URL.
func FetchVersion(ctx context.Context, client *http.Client, url string) (*VersionInfo, error) {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	if url == "" {
		url = DefaultVersionURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
