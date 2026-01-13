package source

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

// DefaultCacheDir returns the default cache directory.
func DefaultCacheDir() string {
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

// LoadFromCache loads data from the cache directory.
func LoadFromCache(dir string) ([]byte, error) {
	dataPath := filepath.Join(dir, cacheDataFile)
	return os.ReadFile(dataPath)
}

// SaveToCache saves data to the cache directory.
// The version parameter is optional and used for metadata.
func SaveToCache(dir string, data []byte, version string) error {
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
	meta := CacheMetadata{
		Version:    version,
		FetchedAt:  time.Now().Unix(),
		Size:       int64(len(data)),
		DataSource: "network",
	}

	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, metaJSON, 0644)

	return nil
}

// GetCacheMetadata reads the cache metadata.
func GetCacheMetadata(dir string) (*CacheMetadata, error) {
	metaPath := filepath.Join(dir, cacheMetadataFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta CacheMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// IsCacheStale returns true if the cache is older than the given duration.
func IsCacheStale(dir string, maxAge time.Duration) bool {
	meta, err := GetCacheMetadata(dir)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(meta.FetchedAt, 0)) > maxAge
}

// ClearCache removes all cached data.
func ClearCache(dir string) error {
	dataPath := filepath.Join(dir, cacheDataFile)
	backupPath := filepath.Join(dir, cacheBackupFile)
	metaPath := filepath.Join(dir, cacheMetadataFile)

	_ = os.Remove(dataPath)
	_ = os.Remove(backupPath)
	_ = os.Remove(metaPath)

	return nil
}
