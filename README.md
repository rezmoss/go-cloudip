# go-cloudip

Fast cloud provider IP detection for Go. Determine if an IP address belongs to AWS, GCP, Azure, Cloudflare, DigitalOcean, or Oracle Cloud with sub-microsecond lookup times.

## Features

- **Fast lookups** - Sub-microsecond performance using Patricia trie
- **Multiple providers** - AWS, GCP, Azure, Cloudflare, DigitalOcean, Oracle
- **Automatic updates** - Optional background refresh from [cloudip-db](https://github.com/rezmoss/cloudip-db)
- **Offline support** - Works without network using embedded data
- **Thread-safe** - Concurrent lookups with lock-free reads
- **Zero dependencies** at runtime (data is embedded)

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
| IPv4 Lookup | < 200 ns |
| IPv6 Lookup | < 300 ns |
| String parse + Lookup | < 400 ns |
| Data load (from cache) | < 20 ms |

Memory usage: ~10-15 MB for all providers loaded.

## Data Source

IP ranges are sourced from [cloud-provider-ip-addresses](https://github.com/rezmoss/cloud-provider-ip-addresses) and compiled daily into MessagePack format by [cloudip-db](https://github.com/rezmoss/cloudip-db).

## License

MIT License - see [LICENSE](LICENSE) file.
