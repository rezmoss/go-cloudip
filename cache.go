package cloudip

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheDataFile     = "cloudip.msgpack"
	cacheMetadataFile = "metadata.json"
	cacheBackupFile   = "cloudip.msgpack.bak"
)

// cacheMetadata stores information about cached data.
type cacheMetadata struct {
	Version    string    `json:"version"`
	FetchedAt  time.Time `json:"fetched_at"`
	Size       int64     `json:"size"`
	DataSource string    `json:"data_source"` // "network" or "embedded"
}

// defaultCacheDir returns the default cache directory.
func defaultCacheDir() string {
	// Try XDG_CACHE_HOME first
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "go-cloudip")
	}

	// Fall back to ~/.cache
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "go-cloudip")
}

// loadFromCache loads data from the cache directory.
func loadFromCache(dir string) ([]byte, error) {
	dataPath := filepath.Join(dir, cacheDataFile)
	return os.ReadFile(dataPath)
}

// saveToCache saves data to the cache directory.
func saveToCache(dir string, data []byte) error {
	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	dataPath := filepath.Join(dir, cacheDataFile)
	backupPath := filepath.Join(dir, cacheBackupFile)
	metaPath := filepath.Join(dir, cacheMetadataFile)

	// Backup existing file
	if _, err := os.Stat(dataPath); err == nil {
		_ = os.Rename(dataPath, backupPath)
	}

	// Write new data
	if err := os.WriteFile(dataPath, data, 0644); err != nil {
		// Try to restore backup
		if _, err := os.Stat(backupPath); err == nil {
			_ = os.Rename(backupPath, dataPath)
		}
		return err
	}

	// Write metadata
	meta := cacheMetadata{
		FetchedAt:  time.Now(),
		Size:       int64(len(data)),
		DataSource: "network",
	}

	// Try to get version from data
	if state, err := loadFromBytes(data); err == nil {
		meta.Version = state.version
	}

	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, metaJSON, 0644)

	return nil
}

// getCacheMetadata reads the cache metadata.
func getCacheMetadata(dir string) (*cacheMetadata, error) {
	metaPath := filepath.Join(dir, cacheMetadataFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta cacheMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// isCacheStale returns true if the cache is older than the given duration.
func isCacheStale(dir string, maxAge time.Duration) bool {
	meta, err := getCacheMetadata(dir)
	if err != nil {
		return true
	}
	return time.Since(meta.FetchedAt) > maxAge
}

// clearCache removes all cached data.
func clearCache(dir string) error {
	dataPath := filepath.Join(dir, cacheDataFile)
	backupPath := filepath.Join(dir, cacheBackupFile)
	metaPath := filepath.Join(dir, cacheMetadataFile)

	_ = os.Remove(dataPath)
	_ = os.Remove(backupPath)
	_ = os.Remove(metaPath)

	return nil
}
