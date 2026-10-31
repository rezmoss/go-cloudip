package cloudip

import (
	"math/rand/v2"
	"net/netip"
	"testing"

	"github.com/rezmoss/go-cloudip/internal/source"
)

// many distinct addrs so cache misses count, not one hot addr
const benchN = 1 << 16

func benchDetector(b *testing.B) *Detector {
	b.Helper()
	d, err := NewDetector(WithOffline())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = d.Close() })
	return d
}

// hitAddrs: random addrs inside random ranges
func hitAddrs(db *source.Database, v6 bool) []netip.Addr {
	rng := rand.New(rand.NewPCG(3, 4))
	var out []netip.Addr
	for len(out) < benchN {
		p := netip.MustParsePrefix(db.Ranges[rng.IntN(len(db.Ranges))].CIDR).Masked()
		if p.Addr().Is6() != v6 {
			continue
		}
		b := p.Addr().AsSlice()
		for i := p.Bits(); i < len(b)*8; i++ {
			if rng.IntN(2) == 1 {
				b[i/8] |= 0x80 >> (i % 8)
			}
		}
		a, _ := netip.AddrFromSlice(b)
		out = append(out, a)
	}
	return out
}

func randomAddrs4() []netip.Addr {
	rng := rand.New(rand.NewPCG(5, 6))
	out := make([]netip.Addr, benchN)
	for i := range out {
		v := rng.Uint32()
		out[i] = netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
	}
	return out
}

func mixAddrs(db *source.Database) []netip.Addr {
	hits, misses := hitAddrs(db, false), randomAddrs4()
	out := make([]netip.Addr, benchN)
	for i := range out {
		if i%2 == 0 {
			out[i] = hits[i]
		} else {
			out[i] = misses[i]
		}
	}
	rand.New(rand.NewPCG(7, 8)).Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

var sinkResult LookupResult

func benchAddrs(b *testing.B, addrs []netip.Addr) {
	d := benchDetector(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		sinkResult = d.LookupAddr(addrs[i&(benchN-1)])
	}
}

func benchStrings(b *testing.B, addrs []netip.Addr) {
	d := benchDetector(b)
	strs := make([]string, len(addrs))
	for i, a := range addrs {
		strs[i] = a.String()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		sinkResult = d.Lookup(strs[i&(benchN-1)])
	}
}

func BenchmarkLookupAddr_IPv4Hit(b *testing.B)    { benchAddrs(b, hitAddrs(embeddedDB(b), false)) }
func BenchmarkLookupAddr_IPv4Random(b *testing.B) { benchAddrs(b, randomAddrs4()) }
func BenchmarkLookupAddr_IPv4Mix(b *testing.B)    { benchAddrs(b, mixAddrs(embeddedDB(b))) }
func BenchmarkLookupAddr_IPv6Hit(b *testing.B)    { benchAddrs(b, hitAddrs(embeddedDB(b), true)) }
func BenchmarkLookup_IPv4Hit(b *testing.B)        { benchStrings(b, hitAddrs(embeddedDB(b), false)) }
func BenchmarkLookup_IPv4Mix(b *testing.B)        { benchStrings(b, mixAddrs(embeddedDB(b))) }
func BenchmarkLookup_IPv6Hit(b *testing.B)        { benchStrings(b, hitAddrs(embeddedDB(b), true)) }

func BenchmarkIsCloudProviderAddr_IPv4Mix(b *testing.B) {
	d := benchDetector(b)
	addrs := mixAddrs(embeddedDB(b))
	var n int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if d.IsCloudProviderAddr(addrs[i&(benchN-1)]) {
			n++
		}
	}
	_ = n
}

func BenchmarkParse(b *testing.B) {
	data, err := source.GetEmbeddedData()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := source.ParseDatabase(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildIndex(b *testing.B) {
	db := embeddedDB(b)
	b.ReportAllocs()
	for b.Loop() {
		mustIndex(b, db)
	}
}
