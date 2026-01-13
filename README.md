# go-cloudip

[![Go Reference](https://pkg.go.dev/badge/github.com/rezmoss/go-cloudip.svg)](https://pkg.go.dev/github.com/rezmoss/go-cloudip)
[![Go Report Card](https://goreportcard.com/badge/github.com/rezmoss/go-cloudip)](https://goreportcard.com/report/github.com/rezmoss/go-cloudip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Fast cloud provider IP detection for Go. Determine if an IP address belongs to AWS, GCP, Azure, Cloudflare, DigitalOcean, or Oracle Cloud with sub-microsecond lookup times.

## Features

- **Fast lookups** - Sub-microsecond performance using Patricia trie
- **Multiple providers** - AWS, GCP, Azure, Cloudflare, DigitalOcean, Oracle
- **Automatic updates** - Optional background refresh from [cloudip-db](https://github.com/rezmoss/cloudip-db)
- **Offline support** - Works without network using embedded data
- **Thread-safe** - Concurrent lookups with lock-free reads
- **Zero config** - First run fetches latest data and caches it; works offline afterwards
- **Embedded fallback** - Includes compiled-in data for air-gapped environments

## Installation

```bash
go get github.com/rezmoss/go-cloudip
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/rezmoss/go-cloudip"
)

func main() {
    // Simple provider detection
    if cloudip.IsAWS("52.94.76.1") {
        fmt.Println("This is an AWS IP")
    }

    // Get provider name
    provider := cloudip.GetProvider("34.64.0.1")
    fmt.Println("Provider:", provider) // "gcp"

    // Check if any cloud provider
    if cloudip.IsCloudProvider("104.16.0.1") {
        fmt.Println("This is a cloud IP")
    }

    // Detailed lookup with region and service info
    result := cloudip.Lookup("52.94.76.1")
    if result.Found {
        fmt.Printf("Provider: %s, Region: %s, Service: %s\n",
            result.Provider, result.Region, result.Service)
    }
}
```

## How Data Loading Works

On initialization, the detector loads IP range data with this priority:

1. **Network** - Fetches latest data from [cloudip-db](https://github.com/rezmoss/cloudip-db) and caches locally
2. **Cache** - Uses previously cached data if network unavailable
3. **Embedded** - Falls back to compiled-in data (for air-gapped environments)

This means:
- **First run**: Automatically downloads latest IP ranges and caches them (~/.cache/go-cloudip)
- **Subsequent runs**: Still tries network first, but has cache as reliable fallback
- **Offline/air-gapped**: Use `WithOffline()` to skip network entirely

## API Reference

### Package-Level Functions

These use a default global detector that initializes automatically:

```go
// Provider detection
cloudip.IsAWS(ip string) bool
cloudip.IsGCP(ip string) bool
cloudip.IsAzure(ip string) bool
cloudip.IsCloudflare(ip string) bool
cloudip.IsDigitalOcean(ip string) bool
cloudip.IsOracle(ip string) bool
cloudip.IsCloudProvider(ip string) bool

// Get provider
cloudip.GetProvider(ip string) Provider

// Detailed lookup
cloudip.Lookup(ip string) LookupResult

// Metadata
cloudip.Version() string
cloudip.RangeCount() int
cloudip.Providers() []string

// Updates
cloudip.Update(ctx context.Context) error
cloudip.CheckUpdate(ctx context.Context) (bool, *VersionInfo, error)
```

### Custom Detector

For more control, create a custom detector:

```go
detector, err := cloudip.NewDetector(
    cloudip.WithDataDir("./cache"),
    cloudip.WithAutoUpdate(24 * time.Hour),
)
if err != nil {
    log.Fatal(err)
}
defer detector.Close()

result := detector.Lookup("52.94.76.1")
```

### Configuration Options

| Option | Description |
|--------|-------------|
| `WithDataDir(dir)` | Set cache directory (default: `~/.cache/go-cloudip`) |
| `WithAutoUpdate(interval)` | Enable background updates (minimum: 1 hour) |
| `WithOffline()` | Disable network, use embedded data only |
| `WithHTTPClient(client)` | Custom HTTP client for requests |
| `WithDataURL(url)` | Custom data URL (for mirrors) |
| `WithVersionURL(url)` | Custom version URL |
| `WithNoCache()` | Disable file caching |

### Types

```go
// Provider represents a cloud provider
type Provider string

const (
    ProviderAWS         Provider = "aws"
    ProviderGCP         Provider = "gcp"
    ProviderAzure       Provider = "azure"
    ProviderCloudflare  Provider = "cloudflare"
    ProviderDigitalOcean Provider = "digitalocean"
    ProviderOracle      Provider = "oracle"
    ProviderUnknown     Provider = ""
)

// LookupResult contains detailed information about an IP
type LookupResult struct {
    Found    bool     // Whether IP was found
    Provider Provider // Cloud provider
    Region   string   // Geographic region (e.g., "us-east-1")
    Service  string   // Service name (e.g., "EC2", "S3")
    CIDR     string   // Matched IP range
}
```

## Usage Examples

### Offline Mode

For air-gapped environments:

```go
detector, _ := cloudip.NewDetector(cloudip.WithOffline())
defer detector.Close()

// Uses only embedded data, no network requests
result := detector.Lookup("52.94.76.1")
```

### With Auto-Update

Keep data fresh automatically:

```go
detector, _ := cloudip.NewDetector(
    cloudip.WithDataDir("/var/cache/cloudip"),
    cloudip.WithAutoUpdate(24 * time.Hour),
)
defer detector.Close()

// Data refreshes in background every 24 hours
```

### Using netip.Addr (Faster)

For high-performance scenarios:

```go
import "net/netip"

addr, _ := netip.ParseAddr("52.94.76.1")
result := cloudip.LookupAddr(addr)
isAWS := cloudip.IsAWSAddr(addr)
```

### Check for Updates

```go
ctx := context.Background()
hasUpdate, info, err := cloudip.CheckUpdate(ctx)
if hasUpdate {
    fmt.Printf("New version: %s (%d ranges)\n", info.Version, info.Ranges)
    cloudip.Update(ctx) // Apply update
}
```

## Performance

| Operation | Time |
|-----------|------|
| IPv4 Lookup | ~200-300 ns |
| IPv6 Lookup | ~200-300 ns |
| LookupAddr (netip.Addr) | ~200-250 ns |
| Data load + parse | ~50 ms |

All lookups are **sub-microsecond**. Memory usage: ~10-15 MB for all providers loaded.

## Data Source

IP ranges are sourced from [cloud-provider-ip-addresses](https://github.com/rezmoss/cloud-provider-ip-addresses) and compiled daily into MessagePack format by [cloudip-db](https://github.com/rezmoss/cloudip-db).

## License

MIT License - see [LICENSE](LICENSE) file.
