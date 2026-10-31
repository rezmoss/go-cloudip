# go-cloudip

[![Go Reference](https://pkg.go.dev/badge/github.com/rezmoss/go-cloudip.svg)](https://pkg.go.dev/github.com/rezmoss/go-cloudip)
[![Go Report Card](https://goreportcard.com/badge/github.com/rezmoss/go-cloudip)](https://goreportcard.com/report/github.com/rezmoss/go-cloudip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Fast cloud provider IP detection for Go. Determine if an IP address belongs to AWS, GCP, Azure, Cloudflare, DigitalOcean, or Oracle Cloud in tens of nanoseconds, with zero allocations.

## Features

- **Fast lookups** - ~50 ns IPv4 hits, ~12 ns misses, zero allocations (see [Performance](#performance))
- **Deterministic results** - Duplicate source records resolve the same way every build; `LookupAll` returns all of them
- **Multiple providers** - AWS, GCP, Azure, Cloudflare, DigitalOcean, Oracle
- **Automatic updates** - Optional background refresh from [cloudip-db](https://github.com/rezmoss/cloudip-db)
- **Offline support** - Works without network using embedded data
- **Thread-safe** - Concurrent lookups with lock-free reads
- **Zero config** - First run fetches latest data and caches it; works offline afterwards
- **Embedded fallback** - Includes compiled-in data for air-gapped environments
- **No runtime dependencies** - Standard library only

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
cloudip.LookupAll(ip string) []LookupResult // every record containing ip

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
| `WithOffline()` | Disable network; use cached data if present, else embedded |
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

// No network requests: uses the cache if present, else embedded data.
// Add cloudip.WithNoCache() to use only the embedded data.
result := detector.Lookup("52.94.76.1")
```

### Overlapping Records

Providers list some prefixes more than once. `1.178.10.0/24`, for example, appears as AWS `AMAZON`, `S3` and `EC2`, and many Azure prefixes appear under a global service tag (no region) and its regional twin. `Lookup` picks the most specific prefix. Among the records for that prefix, it prefers one with a region, then a specific service over an umbrella tag (`AMAZON`, `AzureCloud`). The choice never depends on file order. If the chosen record has no region, the region of the nearest enclosing range of the same provider is used.

`LookupAll` returns every record, unmodified, most specific prefix first:

```go
for _, r := range cloudip.LookupAll("1.178.10.7") {
    fmt.Println(r.CIDR, r.Service, r.Region)
}
// 1.178.10.0/24 EC2 eu-central-1
// 1.178.10.0/24 S3 eu-central-1
// 1.178.10.0/24 AMAZON eu-central-1
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

Measured on an Apple M1 Max with Go 1.26 and the embedded data (131,624 records, 83,206 distinct prefixes). Each benchmark cycles through 65,536 distinct addresses, so cache misses are included; looking up the same address in a loop is about 14 ns.

| Operation | Time | Allocs |
|-----------|------|--------|
| `LookupAddr`, IPv4 hit | ~50 ns | 0 |
| `LookupAddr`, IPv4 random address (mostly misses) | ~12 ns | 0 |
| `LookupAddr`, IPv4 50/50 mix | ~40 ns | 0 |
| `LookupAddr`, IPv6 hit | ~100 ns | 0 |
| `Lookup` (string), IPv4 hit | ~65 ns | 0 |
| `IsCloudProviderAddr`, IPv4 mix | ~20 ns | 0 |
| Load embedded data (gunzip + decode + build) | ~50 ms | ~4k |

Memory retained after load: ~9.5 MB.

Compared on the same machine and workload (median of 5 runs):

| Workload | go-cloudip | [bart](https://github.com/gaissmai/bart) v0.30 | previous go-cloudip (cidranger) |
|----------|-----------|------|-----------|
| IPv4 hit | 48 ns | 68 ns | 1,077 ns, 5 allocs |
| IPv4 random | 12 ns | 19 ns | 135 ns |
| IPv4 mix | 39 ns | 48 ns | 653 ns |
| IPv6 hit | 114 ns | 189 ns | 1,403 ns |

Run `make bench` to reproduce.

## Data Source

IP ranges are sourced from [cloud-provider-ip-addresses](https://github.com/rezmoss/cloud-provider-ip-addresses) and compiled daily into MessagePack format by [cloudip-db](https://github.com/rezmoss/cloudip-db). The embedded copy is refreshed weekly; at runtime the detector downloads the latest gzipped database (~750 KB).

## License

MIT License - see [LICENSE](LICENSE) file.
