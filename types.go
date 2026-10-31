package cloudip

// Provider represents a cloud provider.
type Provider string

// Cloud provider constants.
const (
	ProviderUnknown      Provider = ""
	ProviderAWS          Provider = "aws"
	ProviderGCP          Provider = "gcp"
	ProviderAzure        Provider = "azure"
	ProviderCloudflare   Provider = "cloudflare"
	ProviderDigitalOcean Provider = "digitalocean"
	ProviderOracle       Provider = "oracle"
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

// detectorState holds the immutable state for lookups.
// This is swapped atomically for lock-free reads.
type detectorState struct {
	idx        *index
	version    string
	buildTime  int64
	providers  []string
	rangeCount int
}
