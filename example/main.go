// Package main demonstrates the usage of the go-cloudip library.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rezmoss/go-cloudip"
)

func main() {
	fmt.Println("=== go-cloudip Example ===\n")

	// ----------------------------------------
	// 1. Simple usage with package-level functions
	// ----------------------------------------
	fmt.Println("1. Simple Provider Detection")
	fmt.Println("-----------------------------")

	testIPs := []string{
		"52.94.76.1",   // AWS
		"34.64.0.1",    // GCP
		"104.16.0.1",   // Cloudflare
		"1.1.1.1",      // Cloudflare DNS
		"192.168.1.1",  // Private (not cloud)
		"8.8.8.8",      // Google DNS
	}

	for _, ip := range testIPs {
		provider := cloudip.GetProvider(ip)
		isCloud := cloudip.IsCloudProvider(ip)
		fmt.Printf("  %s -> Provider: %-12s IsCloud: %v\n", ip, provider, isCloud)
	}

	// ----------------------------------------
	// 2. Provider-specific checks
	// ----------------------------------------
	fmt.Println("\n2. Provider-Specific Checks")
	fmt.Println("---------------------------")

	awsIP := "52.94.76.1"
	fmt.Printf("  %s:\n", awsIP)
	fmt.Printf("    IsAWS:        %v\n", cloudip.IsAWS(awsIP))
	fmt.Printf("    IsGCP:        %v\n", cloudip.IsGCP(awsIP))
	fmt.Printf("    IsCloudflare: %v\n", cloudip.IsCloudflare(awsIP))

	// ----------------------------------------
	// 3. Detailed lookup with full info
	// ----------------------------------------
	fmt.Println("\n3. Detailed Lookup")
	fmt.Println("------------------")

	for _, ip := range testIPs[:3] {
		result := cloudip.Lookup(ip)
		if result.Found {
			fmt.Printf("  %s:\n", ip)
			fmt.Printf("    Provider: %s\n", result.Provider)
			fmt.Printf("    Region:   %s\n", result.Region)
			fmt.Printf("    Service:  %s\n", result.Service)
			fmt.Printf("    CIDR:     %s\n", result.CIDR)
		} else {
			fmt.Printf("  %s: Not found in cloud IP ranges\n", ip)
		}
	}

	// ----------------------------------------
	// 4. Data version and metadata
	// ----------------------------------------
	fmt.Println("\n4. Data Information")
	fmt.Println("-------------------")
	fmt.Printf("  Version:     %s\n", cloudip.Version())
	fmt.Printf("  Range Count: %d\n", cloudip.RangeCount())
	fmt.Printf("  Providers:   %v\n", cloudip.Providers())

	// ----------------------------------------
	// 5. Custom detector with options
	// ----------------------------------------
	fmt.Println("\n5. Custom Detector")
	fmt.Println("------------------")

	detector, err := cloudip.NewDetector(
		cloudip.WithDataDir("./cloudip-cache"),
		cloudip.WithAutoUpdate(24*time.Hour),
	)
	if err != nil {
		fmt.Printf("  Error creating detector: %v\n", err)
	} else {
		defer detector.Close()

		result := detector.Lookup("34.64.0.1")
		fmt.Printf("  Custom detector lookup for 34.64.0.1:\n")
		fmt.Printf("    Found:    %v\n", result.Found)
		fmt.Printf("    Provider: %s\n", result.Provider)
	}

	// ----------------------------------------
	// 6. Check for updates
	// ----------------------------------------
	fmt.Println("\n6. Update Check")
	fmt.Println("---------------")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasUpdate, info, err := cloudip.CheckUpdate(ctx)
	if err != nil {
		fmt.Printf("  Error checking update: %v\n", err)
	} else if hasUpdate && info != nil {
		fmt.Printf("  New version available: %s\n", info.Version)
		fmt.Printf("  Ranges: %d\n", info.Ranges)
	} else {
		fmt.Println("  Data is up to date")
	}

	fmt.Println("\n=== Example Complete ===")
}
