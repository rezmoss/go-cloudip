package cloudip

import (
	"net/http"
	"time"
)

// options holds configuration for the Detector.
type options struct {
	// dataDir is the directory for caching data.
	dataDir string

	// autoUpdate is the interval for automatic updates.
	// Zero means no automatic updates.
	autoUpdate time.Duration

	// offline disables network access.
	offline bool

	// httpClient is the HTTP client to use for requests.
	httpClient *http.Client

	// dataURL overrides the default data URL.
	dataURL string

	// versionURL overrides the default version URL.
	versionURL string
}

// defaultOptions returns options with default values.
func defaultOptions() *options {
	return &options{
		dataDir:    defaultCacheDir(),
		autoUpdate: 0,
		offline:    false,
		httpClient: nil,
		dataURL:    defaultDataURL,
		versionURL: defaultVersionURL,
	}
}

// Option is a functional option for configuring the Detector.
type Option func(*options)

// WithDataDir sets the directory for caching data.
// Set to empty string to disable caching.
func WithDataDir(dir string) Option {
	return func(o *options) {
		o.dataDir = dir
	}
}

// WithAutoUpdate enables automatic background updates at the given interval.
// Minimum interval is 1 hour. Use zero to disable (default).
func WithAutoUpdate(interval time.Duration) Option {
	return func(o *options) {
		if interval > 0 && interval < time.Hour {
			interval = time.Hour
		}
		o.autoUpdate = interval
	}
}

// WithOffline disables all network access.
// The detector will only use embedded data.
func WithOffline() Option {
	return func(o *options) {
		o.offline = true
	}
}

// WithHTTPClient sets a custom HTTP client for network requests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}

// WithDataURL overrides the default data URL.
// Use this if you're hosting your own cloudip-db mirror.
func WithDataURL(url string) Option {
	return func(o *options) {
		o.dataURL = url
	}
}

// WithVersionURL overrides the default version URL.
func WithVersionURL(url string) Option {
	return func(o *options) {
		o.versionURL = url
	}
}

// WithNoCache disables the file cache.
func WithNoCache() Option {
	return func(o *options) {
		o.dataDir = ""
	}
}
