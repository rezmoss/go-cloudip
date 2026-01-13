// Package cloudip provides fast cloud provider IP detection.
//
// It determines whether an IP address belongs to AWS, GCP, Azure,
// Cloudflare, DigitalOcean, or Oracle Cloud with sub-microsecond
// lookup times using a Patricia trie data structure.
//
// # Quick Start
//
// Use package-level functions for simple lookups:
//
//	if cloudip.IsAWS("52.94.76.1") {
//	    fmt.Println("This is an AWS IP")
//	}
//
//	provider := cloudip.GetProvider("34.64.0.1")
//	fmt.Println("Provider:", provider) // "gcp"
//
//	result := cloudip.Lookup("52.94.76.1")
//	if result.Found {
//	    fmt.Printf("Provider: %s, Region: %s\n", result.Provider, result.Region)
//	}
//
// # Custom Detector
//
// For more control, create a custom Detector:
//
//	detector, err := cloudip.NewDetector(
//	    cloudip.WithDataDir("./cache"),
//	    cloudip.WithAutoUpdate(24 * time.Hour),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer detector.Close()
//
//	result := detector.Lookup("52.94.76.1")
//
// # High Performance
//
// For best performance, use netip.Addr directly:
//
//	addr, _ := netip.ParseAddr("52.94.76.1")
//	result := cloudip.LookupAddr(addr)
//
// # Data Source
//
// IP ranges are sourced from official cloud provider publications
// and compiled into an embedded database that can be updated at runtime.
package cloudip
