package cloudip

import (
	"slices"
	"testing"

	"github.com/rezmoss/go-cloudip/internal/source"
)

func batchTestState(t *testing.T, provider string) *detectorState {
	t.Helper()
	state, err := buildState(&source.Database{
		Providers: []string{provider, "azure"},
		Ranges: []source.Range{
			{CIDR: "192.0.2.0/24", Provider: 0, Region: "test-region", Service: "TEST"},
			{CIDR: "2001:db8::/32", Provider: 1, Service: "V6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestLookupBatchOrderAndMisses(t *testing.T) {
	d := &Detector{}
	d.state.Store(batchTestState(t, "aws"))
	ips := []string{"192.0.2.5", "bad-ip", "2001:db8::1", "203.0.113.5", "192.0.2.5", "::ffff:192.0.2.5"}
	results := d.LookupBatch(ips)
	if len(results) != len(ips) {
		t.Fatalf("result length = %d, want %d", len(results), len(ips))
	}
	for i, ip := range ips {
		if want := d.Lookup(ip); results[i] != want {
			t.Errorf("result[%d] for %q = %+v, want %+v", i, ip, results[i], want)
		}
	}
	if !results[0].Found || results[0].Provider != ProviderAWS ||
		!results[2].Found || results[2].Provider != ProviderAzure ||
		results[1].Found || results[3].Found ||
		results[0] != results[4] || results[0] != results[5] {
		t.Fatalf("unexpected batch results: %+v", results)
	}
}

func TestLookupBatchEmptyAndUninitialized(t *testing.T) {
	d := &Detector{}
	if got := d.LookupBatch(nil); got != nil {
		t.Fatalf("nil input = %v, want nil", got)
	}
	if got := d.LookupBatch([]string{}); got == nil || len(got) != 0 {
		t.Fatalf("empty input = %v, want non-nil empty slice", got)
	}
	results := d.LookupBatch([]string{"192.0.2.5", "invalid"})
	if len(results) != 2 || results[0].Found || results[1].Found {
		t.Fatalf("uninitialized detector results = %+v", results)
	}
}

func TestLookupBatchConsistentSnapshotDuringUpdates(t *testing.T) {
	d := &Detector{}
	one := batchTestState(t, "aws")
	two := batchTestState(t, "gcp")
	d.state.Store(one)
	ips := slices.Repeat([]string{"192.0.2.5"}, 4096)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			d.state.Store(two)
			d.state.Store(one)
		}
	}()
	for i := 0; i < 20; i++ {
		results := d.LookupBatch(ips)
		provider := results[0].Provider
		if provider != ProviderAWS && provider != ProviderGCP {
			t.Fatalf("batch provider %q", provider)
		}
		for j, result := range results {
			if !result.Found || result.Provider != provider {
				t.Fatalf("batch %d mixed snapshots at item %d: first %q, got %+v", i, j, provider, result)
			}
		}
	}
	<-done
}
