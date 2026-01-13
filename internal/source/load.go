package source

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// ParseDatabase parses MessagePack data into a Database struct.
// It automatically detects and handles gzip compression.
func ParseDatabase(data []byte) (*Database, error) {
	// Try to decompress if gzipped
	decompressed, err := Decompress(data)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	// Parse MessagePack
	var db Database
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

	return &db, nil
}

// Decompress decompresses data if it's gzip compressed.
// Returns the original data if not compressed.
func Decompress(data []byte) ([]byte, error) {
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
