// Package cloudip provides fast cloud provider IP detection.
//
// It can determine if an IP address belongs to major cloud providers
// (AWS, GCP, Azure, Cloudflare, DigitalOcean, Oracle) with sub-microsecond
// lookup times using a Patricia trie data structure.
package cloudip

import (
	"net"
	"net/netip"

	"github.com/yl2chen/cidranger"
)

// Provider represents a cloud provider.
type Provider string

// Cloud provider constants.
const (
	ProviderUnknown     Provider = ""
	ProviderAWS         Provider = "aws"
	ProviderGCP         Provider = "gcp"
	ProviderAzure       Provider = "azure"
	ProviderCloudflare  Provider = "cloudflare"
	ProviderDigitalOcean Provider = "digitalocean"
	ProviderOracle      Provider = "oracle"
)

// String returns the provider name.
func (p Provider) String() string {
	if p == ProviderUnknown {
		return "unknown"
	}
	return string(p)
}

// LookupResult contains information about an IP address lookup.
type LookupResult struct {
	// Found indicates whether the IP was found in any cloud provider range.
	Found bool

	// Provider is the cloud provider that owns this IP range.
	Provider Provider

	// Region is the geographic region (e.g., "us-east-1", "europe-west1").
	// May be empty if not available.
	Region string

	// Service is the cloud service (e.g., "EC2", "S3", "CLOUDFRONT").
	// May be empty if not available.
	Service string

	// CIDR is the IP range that matched.
	CIDR string
}

// rangeEntry implements cidranger.RangerEntry for storing IP range metadata.
type rangeEntry struct {
	network  net.IPNet
	provider Provider
	region   string
	service  string
	cidr     string
}

// Network returns the IP network for this entry.
func (e *rangeEntry) Network() net.IPNet {
	return e.network
}

// networkFromCIDR converts a CIDR string to net.IPNet.
func networkFromCIDR(cidr string) (net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return net.IPNet{}, err
	}
	return *ipNet, nil
}

// netIPToNetIP converts netip.Addr to net.IP.
func netIPToNetIP(addr netip.Addr) net.IP {
	return addr.AsSlice()
}

// database represents the MessagePack database structure.
type database struct {
	Version   string   `msgpack:"version"`
	BuildTime int64    `msgpack:"build_time"`
	Providers []string `msgpack:"providers"`
	Ranges    []dbRange `msgpack:"ranges"`
}

// dbRange represents a single IP range from the database.
type dbRange struct {
	CIDR     string `msgpack:"cidr"`
	Provider int    `msgpack:"p"`
	Region   string `msgpack:"r,omitempty"`
	Service  string `msgpack:"s,omitempty"`
}

// detectorState holds the immutable state for lookups.
// This is swapped atomically for lock-free reads.
type detectorState struct {
	ranger    cidranger.Ranger
	version   string
	buildTime int64
	providers []string
	rangeCount int
}
