package cloudip

import (
	"github.com/rezmoss/go-cloudip/internal/source"
	"github.com/yl2chen/cidranger"
)

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
