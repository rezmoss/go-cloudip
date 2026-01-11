package cloudip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/yl2chen/cidranger"
)

// loadFromBytes loads the database from raw bytes (MessagePack format).
// It automatically detects and handles gzip compression.
func loadFromBytes(data []byte) (*detectorState, error) {
	// Try to decompress if gzipped
	decompressed, err := maybeDecompress(data)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	// Parse MessagePack
	var db database
	if err := msgpack.Unmarshal(decompressed, &db); err != nil {
		return nil, fmt.Errorf("msgpack unmarshal failed: %w", err)
	}

	// Validate database
	if len(db.Providers) == 0 {
		return nil, fmt.Errorf("database has no providers")
	}
	if len(db.Ranges) == 0 {
		return nil, fmt.Errorf("database has no ranges")
	}

	// Build the trie
	state, err := buildState(&db)
	if err != nil {
		return nil, fmt.Errorf("failed to build state: %w", err)
	}

	return state, nil
}

// maybeDecompress decompresses data if it's gzip compressed.
func maybeDecompress(data []byte) ([]byte, error) {
	// Check for gzip magic bytes
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("gzip reader creation failed: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip decompression failed: %w", err)
		}
		return decompressed, nil
	}
	return data, nil
}

// buildState constructs the detector state from a parsed database.
func buildState(db *database) (*detectorState, error) {
	ranger := cidranger.NewPCTrieRanger()

	// Map provider indices to Provider constants
	providerMap := make([]Provider, len(db.Providers))
	for i, name := range db.Providers {
		providerMap[i] = Provider(name)
	}

	// Add all ranges to the trie
	for _, r := range db.Ranges {
		network, err := networkFromCIDR(r.CIDR)
		if err != nil {
			// Skip invalid CIDRs but continue loading
			continue
		}

		// Get provider from index
		var provider Provider
		if r.Provider >= 0 && r.Provider < len(providerMap) {
			provider = providerMap[r.Provider]
		}

		entry := &rangeEntry{
			network:  network,
			provider: provider,
			region:   r.Region,
			service:  r.Service,
			cidr:     r.CIDR,
		}

		if err := ranger.Insert(entry); err != nil {
			// Skip entries that fail to insert
			continue
		}
	}

	return &detectorState{
		ranger:     ranger,
		version:    db.Version,
		buildTime:  db.BuildTime,
		providers:  db.Providers,
		rangeCount: len(db.Ranges),
	}, nil
}
