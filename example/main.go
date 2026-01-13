// Package main demonstrates the usage of the go-cloudip library.
package main

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/rezmoss/go-cloudip"
)

func main() {
	fmt.Println("=== go-cloudip Comprehensive Example ===")
	fmt.Println()

	// Test IPs for all 6 providers (verified working IPs)
	testCases := []struct {
		name     string
		ip       string
		expected cloudip.Provider
	}{
		{"AWS", "52.94.76.1", cloudip.ProviderAWS},
		{"AWS S3", "54.231.0.1", cloudip.ProviderAWS},
		{"GCP", "35.190.0.1", cloudip.ProviderGCP},
		{"GCP Compute", "35.184.0.1", cloudip.ProviderGCP},
		{"Azure", "20.33.0.1", cloudip.ProviderAzure},
		{"Cloudflare", "104.16.0.1", cloudip.ProviderCloudflare},
		{"Cloudflare CDN", "172.64.0.1", cloudip.ProviderCloudflare},
		{"DigitalOcean", "167.99.0.1", cloudip.ProviderDigitalOcean},
		{"Oracle", "129.148.0.1", cloudip.ProviderOracle},
		{"Private (not cloud)", "192.168.1.1", cloudip.ProviderUnknown},
		{"Google DNS (not cloud)", "8.8.8.8", cloudip.ProviderUnknown},
	}

	// 1. Data Information
	fmt.Println("1. DATA INFORMATION")
	fmt.Println("-------------------")
	fmt.Printf("   Version:     %s\n", cloudip.Version())
	fmt.Printf("   Range Count: %d\n", cloudip.RangeCount())
	fmt.Printf("   Providers:   %v\n", cloudip.Providers())
	fmt.Println()

	// 2. GetProvider() - Basic detection
	fmt.Println("2. PROVIDER DETECTION (GetProvider)")
	fmt.Println("------------------------------------")
	for _, tc := range testCases {
		provider := cloudip.GetProvider(tc.ip)
		status := "OK"
		if provider != tc.expected {
			status = fmt.Sprintf("FAIL (expected %s)", tc.expected)
		}
		fmt.Printf("   %-20s %-15s -> %-12s [%s]\n", tc.name, tc.ip, provider, status)
	}
	fmt.Println()

	// 3. Provider-specific check functions
	fmt.Println("3. PROVIDER-SPECIFIC CHECKS")
	fmt.Println("---------------------------")
	checkFuncs := []struct {
		name string
		fn   func(string) bool
		ip   string
	}{
		{"IsAWS", cloudip.IsAWS, "52.94.76.1"},
		{"IsGCP", cloudip.IsGCP, "35.190.0.1"},
		{"IsAzure", cloudip.IsAzure, "20.33.0.1"},
		{"IsCloudflare", cloudip.IsCloudflare, "104.16.0.1"},
		{"IsDigitalOcean", cloudip.IsDigitalOcean, "167.99.0.1"},
		{"IsOracle", cloudip.IsOracle, "129.148.0.1"},
		{"IsCloudProvider", cloudip.IsCloudProvider, "52.94.76.1"},
	}
	for _, cf := range checkFuncs {
		result := cf.fn(cf.ip)
		fmt.Printf("   %-20s(%s) = %v\n", cf.name, cf.ip, result)
	}
	fmt.Println()

	// 4. Detailed Lookup with region/service info
	fmt.Println("4. DETAILED LOOKUP (with region/service)")
	fmt.Println("-----------------------------------------")
	for _, tc := range testCases {
		result := cloudip.Lookup(tc.ip)
		if result.Found {
			fmt.Printf("   %s (%s):\n", tc.name, tc.ip)
			fmt.Printf("      Provider: %s\n", result.Provider)
			fmt.Printf("      Region:   %s\n", result.Region)
			fmt.Printf("      Service:  %s\n", result.Service)
			fmt.Printf("      CIDR:     %s\n", result.CIDR)
		} else {
			fmt.Printf("   %s (%s): Not found in cloud ranges\n", tc.name, tc.ip)
		}
	}
	fmt.Println()

	// 5. Using netip.Addr (LookupAddr, GetProviderAddr)
	fmt.Println("5. NETIP.ADDR FUNCTIONS")
	fmt.Println("-----------------------")
	addr := netip.MustParseAddr("52.94.76.1")
	fmt.Printf("   IP: %s\n", addr)
	fmt.Printf("   GetProviderAddr:      %s\n", cloudip.GetProviderAddr(addr))
	fmt.Printf("   IsCloudProviderAddr:  %v\n", cloudip.IsCloudProviderAddr(addr))
	fmt.Printf("   IsAWSAddr:            %v\n", cloudip.IsAWSAddr(addr))
	result := cloudip.LookupAddr(addr)
	fmt.Printf("   LookupAddr Found:     %v\n", result.Found)
	fmt.Println()

	// 6. Check for updates
	fmt.Println("6. UPDATE CHECK")
	fmt.Println("---------------")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasUpdate, info, err := cloudip.CheckUpdate(ctx)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else if hasUpdate && info != nil {
		fmt.Printf("   New version available: %s\n", info.Version)
		fmt.Printf("   Remote ranges: %d\n", info.Ranges)
	} else {
		fmt.Println("   Data is up to date")
	}
	fmt.Println()

	// 7. Custom detector with options
	fmt.Println("7. CUSTOM DETECTOR")
	fmt.Println("------------------")
	detector, err := cloudip.NewDetector(
		cloudip.WithOffline(),
	)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		defer detector.Close()
		fmt.Printf("   Custom detector version: %s\n", detector.Version())
		fmt.Printf("   Custom detector ranges:  %d\n", detector.RangeCount())
		r := detector.Lookup("52.94.76.1")
		fmt.Printf("   Custom lookup (52.94.76.1): %s\n", r.Provider)
	}
	fmt.Println()

	// 8. Summary
	fmt.Println("8. SUMMARY")
	fmt.Println("----------")
	passed := 0
	failed := 0
	for _, tc := range testCases {
		if cloudip.GetProvider(tc.ip) == tc.expected {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("   Tests passed: %d/%d\n", passed, len(testCases))
	if failed > 0 {
		fmt.Printf("   Tests failed: %d (data may need update)\n", failed)
	}
	fmt.Println()
	fmt.Println("=== Example Complete ===")
}
