package cloudip

import (
	"context"
	"net/netip"
	"sync"
)

var (
	// Global default detector
	defaultDetector     *Detector
	defaultDetectorOnce sync.Once
	defaultDetectorErr  error
)

// getDefaultDetector returns the global default detector, initializing it if needed.
func getDefaultDetector() (*Detector, error) {
	defaultDetectorOnce.Do(func() {
		defaultDetector, defaultDetectorErr = NewDetector()
	})
	return defaultDetector, defaultDetectorErr
}

// mustGetDefaultDetector returns the default detector or panics if initialization failed.
func mustGetDefaultDetector() *Detector {
	d, err := getDefaultDetector()
	if err != nil {
		// Return a non-nil detector that returns empty results
		// rather than panicking
		return &Detector{}
	}
	return d
}

// Lookup returns information about the given IP address using the default detector.
func Lookup(ip string) LookupResult {
	return mustGetDefaultDetector().Lookup(ip)
}

// LookupAddr returns information about the given IP address using the default detector.
func LookupAddr(addr netip.Addr) LookupResult {
	return mustGetDefaultDetector().LookupAddr(addr)
}

// GetProvider returns the provider for the given IP, or empty string if not found.
func GetProvider(ip string) Provider {
	return mustGetDefaultDetector().GetProvider(ip)
}

// GetProviderAddr returns the provider for the given IP address.
func GetProviderAddr(addr netip.Addr) Provider {
	return mustGetDefaultDetector().GetProviderAddr(addr)
}

// IsCloudProvider returns true if the IP belongs to any cloud provider.
func IsCloudProvider(ip string) bool {
	return mustGetDefaultDetector().IsCloudProvider(ip)
}

// IsCloudProviderAddr returns true if the IP belongs to any cloud provider.
func IsCloudProviderAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsCloudProviderAddr(addr)
}

// IsAWS returns true if the IP belongs to AWS.
func IsAWS(ip string) bool {
	return mustGetDefaultDetector().IsAWS(ip)
}

// IsAWSAddr returns true if the IP belongs to AWS.
func IsAWSAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsAWSAddr(addr)
}

// IsGCP returns true if the IP belongs to Google Cloud.
func IsGCP(ip string) bool {
	return mustGetDefaultDetector().IsGCP(ip)
}

// IsGCPAddr returns true if the IP belongs to Google Cloud.
func IsGCPAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsGCPAddr(addr)
}

// IsAzure returns true if the IP belongs to Microsoft Azure.
func IsAzure(ip string) bool {
	return mustGetDefaultDetector().IsAzure(ip)
}

// IsAzureAddr returns true if the IP belongs to Microsoft Azure.
func IsAzureAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsAzureAddr(addr)
}

// IsCloudflare returns true if the IP belongs to Cloudflare.
func IsCloudflare(ip string) bool {
	return mustGetDefaultDetector().IsCloudflare(ip)
}

// IsCloudflareAddr returns true if the IP belongs to Cloudflare.
func IsCloudflareAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsCloudflareAddr(addr)
}

// IsDigitalOcean returns true if the IP belongs to DigitalOcean.
func IsDigitalOcean(ip string) bool {
	return mustGetDefaultDetector().IsDigitalOcean(ip)
}

// IsDigitalOceanAddr returns true if the IP belongs to DigitalOcean.
func IsDigitalOceanAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsDigitalOceanAddr(addr)
}

// IsOracle returns true if the IP belongs to Oracle Cloud.
func IsOracle(ip string) bool {
	return mustGetDefaultDetector().IsOracle(ip)
}

// IsOracleAddr returns true if the IP belongs to Oracle Cloud.
func IsOracleAddr(addr netip.Addr) bool {
	return mustGetDefaultDetector().IsOracleAddr(addr)
}

// Version returns the data version string from the default detector.
func Version() string {
	return mustGetDefaultDetector().Version()
}

// RangeCount returns the number of IP ranges loaded in the default detector.
func RangeCount() int {
	return mustGetDefaultDetector().RangeCount()
}

// Providers returns the list of providers from the default detector.
func Providers() []string {
	return mustGetDefaultDetector().Providers()
}

// Update fetches the latest data using the default detector.
func Update(ctx context.Context) error {
	return mustGetDefaultDetector().Update(ctx)
}

// CheckUpdate checks if a new version is available using the default detector.
func CheckUpdate(ctx context.Context) (bool, *VersionInfo, error) {
	return mustGetDefaultDetector().CheckUpdate(ctx)
}
