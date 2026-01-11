# go-cloudip Example

This example demonstrates the core features of the go-cloudip library.

## Running the Example

```bash
cd example
go run main.go
```

## What This Example Shows

1. **Simple Provider Detection** - Using `GetProvider()` and `IsCloudProvider()` to identify cloud IPs
2. **Provider-Specific Checks** - Using `IsAWS()`, `IsGCP()`, `IsCloudflare()`, etc.
3. **Detailed Lookup** - Using `Lookup()` to get full information (region, service, CIDR)
4. **Data Metadata** - Viewing version, range count, and supported providers
5. **Custom Detector** - Creating a detector with custom cache directory and auto-update
6. **Update Check** - Checking if newer data is available

## Sample Output

```
=== go-cloudip Example ===

1. Simple Provider Detection
-----------------------------
  52.94.76.1 -> Provider: aws          IsCloud: true
  34.64.0.1 -> Provider: gcp          IsCloud: true
  104.16.0.1 -> Provider: cloudflare   IsCloud: true
  1.1.1.1 -> Provider: cloudflare   IsCloud: true
  192.168.1.1 -> Provider: unknown      IsCloud: false
  8.8.8.8 -> Provider: gcp          IsCloud: true

2. Provider-Specific Checks
---------------------------
  52.94.76.1:
    IsAWS:        true
    IsGCP:        false
    IsCloudflare: false

...
```

## More Examples

For advanced usage patterns, see the [main package documentation](https://pkg.go.dev/github.com/rezmoss/go-cloudip).
