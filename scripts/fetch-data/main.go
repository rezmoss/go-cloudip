// Command fetch-data refreshes internal/source/data from cloudip-db
// (verifies SHA-256, checks it parses). Run: go run ./scripts/fetch-data
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rezmoss/go-cloudip/internal/source"
)

const outputDir = "internal/source/data"

func main() {
	if err := run(); err != nil {
		log.Fatalf("fetch-data: %v", err)
	}
}

func run() error {
	ctx := context.Background()
	client := &http.Client{Timeout: source.DefaultTimeout}

	versionData, err := source.FetchVersionRaw(ctx, client, source.DefaultVersionURL)
	if err != nil {
		return fmt.Errorf("fetch version.json: %w", err)
	}
	var info source.VersionInfo
	if err := json.Unmarshal(versionData, &info); err != nil {
		return fmt.Errorf("parse version.json: %w", err)
	}

	rawMsgpack, err := source.FetchData(ctx, client, source.DefaultDataURL)
	if err != nil {
		return fmt.Errorf("fetch cloudip.msgpack: %w", err)
	}

	if info.SHA256 != "" {
		sum := sha256.Sum256(rawMsgpack)
		got := hex.EncodeToString(sum[:])
		if got != info.SHA256 {
			return fmt.Errorf("sha256 mismatch: version.json=%s downloaded=%s", info.SHA256, got)
		}
	}

	db, err := source.ParseDatabase(rawMsgpack)
	if err != nil {
		return fmt.Errorf("downloaded data does not parse: %w", err)
	}

	// gzip before embed to shrink module; Decompress inflates on load
	compressed, err := gzipBytes(rawMsgpack)
	if err != nil {
		return fmt.Errorf("gzip data: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outputDir, err)
	}

	dataPath := filepath.Join(outputDir, "cloudip.msgpack")
	if err := os.WriteFile(dataPath, compressed, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dataPath, err)
	}

	versionPath := filepath.Join(outputDir, "version.json")
	if err := os.WriteFile(versionPath, versionData, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", versionPath, err)
	}

	log.Printf("synced cloudip-db %s: %d ranges, %d providers, %d bytes (%d gzipped)",
		db.Version, len(db.Ranges), len(db.Providers), len(rawMsgpack), len(compressed))
	return nil
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gw.Write(data); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
