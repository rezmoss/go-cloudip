package cloudip

import "github.com/rezmoss/go-cloudip/internal/source"

// loadFromBytes loads the database from raw bytes and builds the detector state.
func loadFromBytes(data []byte) (*detectorState, error) {
	db, err := source.ParseDatabase(data)
	if err != nil {
		return nil, err
	}

	return buildState(db)
}

// buildState constructs the detector state from a parsed database.
func buildState(db *source.Database) (*detectorState, error) {
	idx, err := newIndex(db)
	if err != nil {
		return nil, err
	}
	return &detectorState{
		idx:        idx,
		version:    db.Version,
		buildTime:  db.BuildTime,
		providers:  db.Providers,
		rangeCount: len(db.Ranges),
	}, nil
}
