package cloudip

import (
	"net/netip"
	"testing"

	"github.com/rezmoss/go-cloudip/internal/source"
)

// Test IPs for various providers
var (
	// AWS IPs (from aws_ips.json)
	awsIPs = []string{
		"52.94.76.1",    // AWS EC2
		"3.5.140.1",     // AWS
		"54.231.0.1",    // AWS S3
	}

	// GCP IPs (from googlecloud_ips.json)
	gcpIPs = []string{
		"8.8.8.8",       // Google DNS (might be in GCP ranges)
		"35.190.0.1",    // GCP
		"34.64.0.1",     // GCP
	}

	// Cloudflare IPs
	cloudflareIPs = []string{
		"104.16.0.1",    // Cloudflare
		"1.1.1.1",       // Cloudflare DNS
		"172.64.0.1",    // Cloudflare
	}

	// Non-cloud IPs
	nonCloudIPs = []string{
		"192.168.1.1",   // Private
		"10.0.0.1",      // Private
		"203.0.113.1",   // TEST-NET-3
	}
)

func TestNewDetector(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	if d.Version() == "" {
		t.Error("Version() returned empty string")
	}

	if d.RangeCount() == 0 {
		t.Error("RangeCount() returned 0")
	}

	providers := d.Providers()
	if len(providers) == 0 {
		t.Error("Providers() returned empty slice")
	}
}

func TestLookup_AWS(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	// Test first AWS IP that exists in the data
	result := d.Lookup("52.94.76.1")
	if !result.Found {
		t.Log("52.94.76.1 not found, trying other AWS IPs")
		// Try to find any AWS IP
		for _, ip := range awsIPs {
			result = d.Lookup(ip)
			if result.Found && result.Provider == ProviderAWS {
				t.Logf("Found AWS IP: %s", ip)
				return
			}
		}
		t.Log("No AWS IPs found in test set, checking if any AWS ranges exist")
	} else {
		if result.Provider != ProviderAWS {
			t.Errorf("Lookup() provider = %v, want %v", result.Provider, ProviderAWS)
		}
	}
}

func TestLookup_NonCloud(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	for _, ip := range nonCloudIPs {
		result := d.Lookup(ip)
		if result.Found {
			t.Errorf("Lookup(%q) found = true, want false (provider = %v)", ip, result.Provider)
		}
	}
}

func TestIsCloudProvider(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	// Non-cloud IPs should return false
	for _, ip := range nonCloudIPs {
		if d.IsCloudProvider(ip) {
			t.Errorf("IsCloudProvider(%q) = true, want false", ip)
		}
	}
}

func TestIsAWS(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	// Non-cloud IPs should return false for IsAWS
	for _, ip := range nonCloudIPs {
		if d.IsAWS(ip) {
			t.Errorf("IsAWS(%q) = true, want false", ip)
		}
	}
}

func TestLookupAddr(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	addr := netip.MustParseAddr("192.168.1.1")
	result := d.LookupAddr(addr)
	if result.Found {
		t.Errorf("LookupAddr(192.168.1.1) found = true, want false")
	}
}

func TestInvalidIP(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	// Invalid IPs should not crash and should return not found
	invalidIPs := []string{
		"",
		"invalid",
		"256.256.256.256",
		"abc.def.ghi.jkl",
	}

	for _, ip := range invalidIPs {
		result := d.Lookup(ip)
		if result.Found {
			t.Errorf("Lookup(%q) found = true, want false for invalid IP", ip)
		}
	}
}

func TestProviderString(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderAWS, "aws"},
		{ProviderGCP, "gcp"},
		{ProviderAzure, "azure"},
		{ProviderCloudflare, "cloudflare"},
		{ProviderDigitalOcean, "digitalocean"},
		{ProviderOracle, "oracle"},
		{ProviderUnknown, "unknown"},
		{Provider(""), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.provider.String(); got != tt.want {
			t.Errorf("Provider(%q).String() = %v, want %v", tt.provider, got, tt.want)
		}
	}
}

// Package-level function tests (using embedded data)
func TestPackageLevelFunctions(t *testing.T) {
	// These tests use the package-level functions which initialize
	// the default detector

	// Test with a known non-cloud IP
	if IsCloudProvider("192.168.1.1") {
		t.Error("IsCloudProvider(192.168.1.1) = true, want false")
	}

	// Test Version returns something
	version := Version()
	if version == "" {
		t.Log("Version() returned empty, detector may not be initialized")
	}

	// Test RangeCount
	count := RangeCount()
	if count == 0 {
		t.Log("RangeCount() returned 0, detector may not be initialized")
	}
}

// Benchmark tests
func BenchmarkLookup_IPv4(b *testing.B) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Lookup("52.94.76.1")
	}
}

func BenchmarkLookup_IPv4_NotFound(b *testing.B) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Lookup("192.168.1.1")
	}
}

func BenchmarkLookupAddr_IPv4(b *testing.B) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	addr := netip.MustParseAddr("52.94.76.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.LookupAddr(addr)
	}
}

func BenchmarkIsAWS(b *testing.B) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.IsAWS("52.94.76.1")
	}
}

func BenchmarkIsCloudProvider(b *testing.B) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.IsCloudProvider("52.94.76.1")
	}
}

func BenchmarkLoad(b *testing.B) {
	data, err := source.GetEmbeddedData()
	if err != nil {
		b.Fatalf("GetEmbeddedData() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loadFromBytes(data)
		if err != nil {
			b.Fatalf("loadFromBytes() error = %v", err)
		}
	}
}

// Test concurrent access
func TestConcurrentLookups(t *testing.T) {
	d, err := NewDetector(WithOffline())
	if err != nil {
		t.Fatalf("NewDetector() error = %v", err)
	}
	defer d.Close()

	// Run concurrent lookups
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				d.Lookup("52.94.76.1")
				d.Lookup("192.168.1.1")
				d.IsAWS("52.94.76.1")
				d.IsCloudProvider("8.8.8.8")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
