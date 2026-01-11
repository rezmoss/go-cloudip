package cloudip

import (
	_ "embed"
)

// Embedded data file compiled from cloudip-db.
// This provides offline functionality and serves as a fallback
// when network is unavailable.
//
//go:embed data/cloudip.msgpack
var embeddedData []byte

// getEmbeddedData returns the embedded MessagePack data.
func getEmbeddedData() ([]byte, error) {
	if len(embeddedData) == 0 {
		return nil, ErrNoEmbeddedData
	}
	return embeddedData, nil
}

// ErrNoEmbeddedData is returned when no embedded data is available.
var ErrNoEmbeddedData = &EmbeddedDataError{msg: "no embedded data available"}

// EmbeddedDataError represents an error with embedded data.
type EmbeddedDataError struct {
	msg string
}

func (e *EmbeddedDataError) Error() string {
	return e.msg
}
