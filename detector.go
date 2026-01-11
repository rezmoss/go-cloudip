package cloudip

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

// Detector is the main type for cloud IP detection.
// It is safe for concurrent use.
type Detector struct {
	state atomic.Pointer[detectorState]

	// Configuration
	opts *options

	// Background updater
	stopUpdate chan struct{}
	updateWg   sync.WaitGroup

	// Initialization
	initOnce sync.Once
	initErr  error
}

// NewDetector creates a new Detector with the given options.
// By default, it loads embedded data and can fetch updates from GitHub.
func NewDetector(opts ...Option) (*Detector, error) {
	d := &Detector{
		opts:       defaultOptions(),
		stopUpdate: make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(d.opts)
	}

	// Load initial data
	if err := d.loadInitialData(); err != nil {
		return nil, err
	}

	// Start background updater if configured
	if d.opts.autoUpdate > 0 && !d.opts.offline {
		d.startAutoUpdate()
	}

	return d, nil
}

// loadInitialData loads data from cache, network, or embedded data.
func (d *Detector) loadInitialData() error {
	// Try cache first if configured
	if d.opts.dataDir != "" {
		data, err := loadFromCache(d.opts.dataDir)
		if err == nil {
			state, err := loadFromBytes(data)
			if err == nil {
				d.state.Store(state)
				return nil
			}
		}
	}

	// Try fetching from network if not offline
	if !d.opts.offline {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		data, err := fetchData(ctx, d.opts.httpClient)
		if err == nil {
			state, err := loadFromBytes(data)
			if err == nil {
				d.state.Store(state)
				// Save to cache
				if d.opts.dataDir != "" {
					_ = saveToCache(d.opts.dataDir, data)
				}
				return nil
			}
		}
	}

	// Fall back to embedded data
	data, err := getEmbeddedData()
	if err != nil {
		return fmt.Errorf("no data available: %w", err)
	}

	state, err := loadFromBytes(data)
	if err != nil {
		return fmt.Errorf("failed to load embedded data: %w", err)
	}

	d.state.Store(state)
	return nil
}

// startAutoUpdate starts the background update goroutine.
func (d *Detector) startAutoUpdate() {
	d.updateWg.Add(1)
	go func() {
		defer d.updateWg.Done()

		ticker := time.NewTicker(d.opts.autoUpdate)
		defer ticker.Stop()

		for {
			select {
			case <-d.stopUpdate:
				return
			case <-ticker.C:
				_ = d.Update(context.Background())
			}
		}
	}()
}

// Update fetches the latest data and updates the detector.
// This is safe to call concurrently.
func (d *Detector) Update(ctx context.Context) error {
	if d.opts.offline {
		return fmt.Errorf("update disabled in offline mode")
	}

	data, err := fetchData(ctx, d.opts.httpClient)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	state, err := loadFromBytes(data)
	if err != nil {
		return fmt.Errorf("load failed: %w", err)
	}

	// Atomic swap - readers never block
	d.state.Store(state)

	// Update cache
	if d.opts.dataDir != "" {
		_ = saveToCache(d.opts.dataDir, data)
	}

	return nil
}

// Close stops background updates and releases resources.
func (d *Detector) Close() error {
	close(d.stopUpdate)
	d.updateWg.Wait()
	return nil
}

// Lookup returns information about the given IP address.
func (d *Detector) Lookup(ip string) LookupResult {
	state := d.state.Load()
	if state == nil {
		return LookupResult{Found: false}
	}
	entry := state.lookupString(ip)
	return entry.toLookupResult()
}

// LookupAddr returns information about the given IP address.
func (d *Detector) LookupAddr(addr netip.Addr) LookupResult {
	state := d.state.Load()
	if state == nil {
		return LookupResult{Found: false}
	}
	entry := state.lookupAddr(addr)
	return entry.toLookupResult()
}

// GetProvider returns the provider for the given IP, or empty string if not found.
func (d *Detector) GetProvider(ip string) Provider {
	return d.Lookup(ip).Provider
}

// GetProviderAddr returns the provider for the given IP address.
func (d *Detector) GetProviderAddr(addr netip.Addr) Provider {
	return d.LookupAddr(addr).Provider
}

// IsCloudProvider returns true if the IP belongs to any cloud provider.
func (d *Detector) IsCloudProvider(ip string) bool {
	return d.Lookup(ip).Found
}

// IsCloudProviderAddr returns true if the IP belongs to any cloud provider.
func (d *Detector) IsCloudProviderAddr(addr netip.Addr) bool {
	return d.LookupAddr(addr).Found
}

// IsAWS returns true if the IP belongs to AWS.
func (d *Detector) IsAWS(ip string) bool {
	return d.GetProvider(ip) == ProviderAWS
}

// IsAWSAddr returns true if the IP belongs to AWS.
func (d *Detector) IsAWSAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderAWS
}

// IsGCP returns true if the IP belongs to Google Cloud.
func (d *Detector) IsGCP(ip string) bool {
	return d.GetProvider(ip) == ProviderGCP
}

// IsGCPAddr returns true if the IP belongs to Google Cloud.
func (d *Detector) IsGCPAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderGCP
}

// IsAzure returns true if the IP belongs to Microsoft Azure.
func (d *Detector) IsAzure(ip string) bool {
	return d.GetProvider(ip) == ProviderAzure
}

// IsAzureAddr returns true if the IP belongs to Microsoft Azure.
func (d *Detector) IsAzureAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderAzure
}

// IsCloudflare returns true if the IP belongs to Cloudflare.
func (d *Detector) IsCloudflare(ip string) bool {
	return d.GetProvider(ip) == ProviderCloudflare
}

// IsCloudflareAddr returns true if the IP belongs to Cloudflare.
func (d *Detector) IsCloudflareAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderCloudflare
}

// IsDigitalOcean returns true if the IP belongs to DigitalOcean.
func (d *Detector) IsDigitalOcean(ip string) bool {
	return d.GetProvider(ip) == ProviderDigitalOcean
}

// IsDigitalOceanAddr returns true if the IP belongs to DigitalOcean.
func (d *Detector) IsDigitalOceanAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderDigitalOcean
}

// IsOracle returns true if the IP belongs to Oracle Cloud.
func (d *Detector) IsOracle(ip string) bool {
	return d.GetProvider(ip) == ProviderOracle
}

// IsOracleAddr returns true if the IP belongs to Oracle Cloud.
func (d *Detector) IsOracleAddr(addr netip.Addr) bool {
	return d.GetProviderAddr(addr) == ProviderOracle
}

// Version returns the data version string.
func (d *Detector) Version() string {
	state := d.state.Load()
	if state == nil {
		return ""
	}
	return state.version
}

// RangeCount returns the number of IP ranges loaded.
func (d *Detector) RangeCount() int {
	state := d.state.Load()
	if state == nil {
		return 0
	}
	return state.rangeCount
}

// Providers returns the list of providers in the database.
func (d *Detector) Providers() []string {
	state := d.state.Load()
	if state == nil {
		return nil
	}
	// Return a copy to prevent modification
	result := make([]string, len(state.providers))
	copy(result, state.providers)
	return result
}
