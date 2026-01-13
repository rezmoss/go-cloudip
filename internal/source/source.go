// Package source provides data loading functionality for cloudip.
// It handles fetching, caching, and parsing of IP range data.
package source

// Database represents the parsed MessagePack database structure.
type Database struct {
	Version   string  `msgpack:"version"`
	BuildTime int64   `msgpack:"build_time"`
	Providers []string `msgpack:"providers"`
	Ranges    []Range `msgpack:"ranges"`
}

// Range represents a single IP range from the database.
type Range struct {
	CIDR     string `msgpack:"cidr"`
	Provider int    `msgpack:"p"`
	Region   string `msgpack:"r,omitempty"`
	Service  string `msgpack:"s,omitempty"`
}

// VersionInfo contains metadata about the data version.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime int64  `json:"build_time"`
	SHA256    string `json:"sha256"`
	Ranges    int    `json:"ranges"`
	Size      int64  `json:"size"`
	SizeGzip  int64  `json:"size_gzip"`
}

// CacheMetadata stores information about cached data.
type CacheMetadata struct {
	Version    string `json:"version"`
	FetchedAt  int64  `json:"fetched_at"`
	Size       int64  `json:"size"`
	DataSource string `json:"data_source"` // "network" or "embedded"
}
