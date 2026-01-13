package source

import (
	_ "embed"
	"errors"
)

// Embedded data file compiled from cloudip-db.
// This provides offline functionality and serves as a fallback
// when network is unavailable.
//
//go:embed data/cloudip.msgpack
var embeddedData []byte

// ErrNoEmbeddedData is returned when no embedded data is available.
var ErrNoEmbeddedData = errors.New("no embedded data available")

// GetEmbeddedData returns the embedded MessagePack data.
func GetEmbeddedData() ([]byte, error) {
	if len(embeddedData) == 0 {
		return nil, ErrNoEmbeddedData
	}
	return embeddedData, nil
}
