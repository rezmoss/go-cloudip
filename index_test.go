package cloudip

import (
	"math/rand/v2"
	"net/netip"
	"slices"
	"testing"

	"github.com/rezmoss/go-cloudip/internal/source"
)

func embeddedDB(tb testing.TB) *source.Database {
	tb.Helper()
	data, err := source.GetEmbeddedData()
	if err != nil {
		tb.Fatal(err)
	}
	db, err := source.ParseDatabase(data)
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func mustIndex(tb testing.TB, db *source.Database) *index {
	tb.Helper()
	ix, err := newIndex(db)
	if err != nil {
		tb.Fatal(err)
	}
	return ix
}

// oracle: naive LPM, map probed from longest prefix down
type oracle struct {
	rows      map[netip.Prefix][]source.Range
	providers []string
}

func newOracle(db *source.Database) *oracle {
	o := &oracle{rows: map[netip.Prefix][]source.Range{}, providers: db.Providers}
	for _, r := range db.Ranges {
		p := netip.MustParsePrefix(r.CIDR).Masked()
		o.rows[p] = append(o.rows[p], r)
	}
	return o
}

func (o *oracle) match(a netip.Addr) (netip.Prefix, []source.Range) {
	for bits := a.BitLen(); bits >= 0; bits-- {
		p, _ := a.Prefix(bits)
		if rs, ok := o.rows[p]; ok {
			return p, rs
		}
	}
	return netip.Prefix{}, nil
}

// probes: first/last addr of each prefix + neighbours (where flat tables break)
func probes(db *source.Database) []netip.Addr {
	var out []netip.Addr
	for _, r := range db.Ranges {
		p := netip.MustParsePrefix(r.CIDR).Masked()
		first := p.Addr()
		last := lastAddr(p)
		out = append(out, first, last)
		if prev := first.Prev(); prev.IsValid() {
			out = append(out, prev)
		}
		if next := last.Next(); next.IsValid() {
			out = append(out, next)
		}
	}
	return out
}

func lastAddr(p netip.Prefix) netip.Addr {
	b := p.Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 0x80 >> (i % 8)
	}
	a, _ := netip.AddrFromSlice(b)
	return a
}

func TestIndexMatchesOracle(t *testing.T) {
	db := embeddedDB(t)
	ix := mustIndex(t, db)
	o := newOracle(db)

	addrs := probes(db)
	rng := rand.New(rand.NewPCG(1, 2))
	for range 200000 {
		addrs = append(addrs, netip.AddrFrom4([4]byte{byte(rng.Uint32()), byte(rng.Uint32()), byte(rng.Uint32()), byte(rng.Uint32())}))
		var b [16]byte
		hi, lo := rng.Uint64(), rng.Uint64()
		for i := range 8 {
			b[i], b[8+i] = byte(hi>>(56-8*i)), byte(lo>>(56-8*i))
		}
		b[0], b[1] = 0x26, 0x03 // dense Azure /16
		addrs = append(addrs, netip.AddrFrom16(b))
	}

	for _, a := range addrs {
		wantPfx, wantRows := o.match(a)
		e := ix.lookupAddr(a)
		got := ix.result(e)
		if !wantPfx.IsValid() {
			if got.Found {
				t.Fatalf("%s: got %+v, want not found", a, got)
			}
			continue
		}
		if !got.Found || got.CIDR != wantPfx.String() {
			t.Fatalf("%s: got %+v, want CIDR %s", a, got, wantPfx)
		}
		// svc from best-ranked row; empty region may come from same-provider parent
		best := slices.MinFunc(wantRows, func(x, y source.Range) int {
			return x.Provider*4 + int(rowRank(x.Region, x.Service)) - y.Provider*4 - int(rowRank(y.Region, y.Service))
		})
		i := slices.IndexFunc(wantRows, func(r source.Range) bool {
			return r.Service == got.Service && r.Provider == best.Provider && (r.Region == got.Region || r.Region == "")
		})
		if i < 0 || got.Provider != Provider(db.Providers[best.Provider]) || rowRank(wantRows[i].Region, wantRows[i].Service) != rowRank(best.Region, best.Service) {
			t.Fatalf("%s: got %+v, want rank of %+v", a, got, best)
		}
		if wantRows[i].Region != got.Region {
			if !inheritedRegion(o, wantPfx, got) {
				t.Fatalf("%s: got region %q, not from %s or an enclosing range", a, got.Region, wantPfx)
			}
		}
	}
}

// inheritedRegion: got.Region == nearest same-provider ancestor's region
func inheritedRegion(o *oracle, p netip.Prefix, got LookupResult) bool {
	for bits := p.Bits() - 1; bits >= 0; bits-- {
		pp, _ := p.Addr().Prefix(bits)
		for _, r := range o.rows[pp] {
			if r.Region != "" && Provider(o.providers[r.Provider]) == got.Provider {
				best := slices.MinFunc(o.rows[pp], func(x, y source.Range) int {
					return int(rowRank(x.Region, x.Service)) - int(rowRank(y.Region, y.Service))
				})
				return best.Region == got.Region
			}
		}
	}
	return false
}

func TestDuplicatePrefixes(t *testing.T) {
	db := &source.Database{
		Providers: []string{"aws", "azure"},
		Ranges: []source.Range{
			{CIDR: "1.178.10.0/24", Provider: 0, Region: "eu-central-1", Service: "S3"},
			{CIDR: "1.178.10.0/24", Provider: 0, Region: "eu-central-1", Service: "AMAZON"},
			{CIDR: "1.178.10.0/24", Provider: 0, Region: "eu-central-1", Service: "EC2"},
			{CIDR: "1.178.10.0/24", Provider: 0, Region: "eu-central-1", Service: "EC2"},
			{CIDR: "157.56.216.0/26", Provider: 1, Service: "AZURECLOUD"},
			{CIDR: "157.56.216.0/26", Provider: 1, Service: "STORAGE"},
			{CIDR: "157.56.216.0/26", Provider: 1, Region: "northcentralus", Service: "AZURECLOUD.NORTHCENTRALUS"},
			{CIDR: "157.56.216.0/26", Provider: 1, Region: "northcentralus", Service: "STORAGE.NORTHCENTRALUS"},
			{CIDR: "157.56.0.0/16", Provider: 1, Service: "AZURECLOUD"},
			{CIDR: "157.56.0.0/16", Provider: 1, Region: "northcentralus", Service: "AZURECLOUD.NORTHCENTRALUS"},
			{CIDR: "157.56.100.0/24", Provider: 1, Service: "SQL"},
			{CIDR: "157.56.96.0/20", Provider: 0, Service: "OTHER"}, // other provider between
		},
	}
	// order must not matter
	for range 20 {
		rand.Shuffle(len(db.Ranges), func(i, j int) { db.Ranges[i], db.Ranges[j] = db.Ranges[j], db.Ranges[i] })
		s, err := buildState(db)
		if err != nil {
			t.Fatal(err)
		}
		d := &Detector{}
		d.state.Store(s)

		if got := d.Lookup("1.178.10.7"); got.Service != "EC2" || got.Region != "eu-central-1" {
			t.Fatalf("aws dup: got %+v", got)
		}
		all := d.LookupAll("1.178.10.7")
		var svcs []string
		for _, r := range all {
			svcs = append(svcs, r.Service)
		}
		if !slices.Equal(svcs, []string{"EC2", "S3", "AMAZON"}) {
			t.Fatalf("aws LookupAll services = %v", svcs)
		}

		if got := d.Lookup("157.56.216.1"); got.Service != "STORAGE.NORTHCENTRALUS" || got.Region != "northcentralus" {
			t.Fatalf("azure dup: got %+v", got)
		}
		if all := d.LookupAll("157.56.216.1"); len(all) != 6 || all[4].CIDR != "157.56.0.0/16" || all[5].Region != "" {
			t.Fatalf("azure LookupAll = %+v", all)
		}
		// global-tag prefix takes parent region; LookupAll stays raw
		if got := d.Lookup("157.56.100.1"); got.Service != "SQL" || got.Region != "northcentralus" || got.CIDR != "157.56.100.0/24" {
			t.Fatalf("region fallback: got %+v", got)
		}
		if all := d.LookupAll("157.56.100.1"); len(all) != 4 || all[0].Region != "" || all[1].Service != "OTHER" {
			t.Fatalf("LookupAll should keep raw region: %+v", all[0])
		}
		if got := d.Lookup("157.56.217.1"); got.CIDR != "157.56.0.0/16" {
			t.Fatalf("azure parent: got %+v", got)
		}
		if d.LookupAll("8.8.8.8") != nil {
			t.Fatal("LookupAll on miss should be nil")
		}
	}
}

func TestIndexFamilyBoundary(t *testing.T) {
	ix := mustIndex(t, &source.Database{
		Providers: []string{"a", "b"},
		Ranges:    []source.Range{{CIDR: "0.0.0.0/0", Provider: 0}, {CIDR: "::/0", Provider: 1}},
	})
	if got := ix.result(ix.lookupAddr(netip.MustParseAddr("1.1.1.1"))); got.CIDR != "0.0.0.0/0" || got.Provider != "a" {
		t.Errorf("v4: %+v", got)
	}
	if got := ix.result(ix.lookupAddr(netip.MustParseAddr("2001::1"))); got.CIDR != "::/0" || got.Provider != "b" {
		t.Errorf("v6: %+v", got)
	}
}

func TestIndexEdges(t *testing.T) {
	db := &source.Database{
		Providers: []string{"a", "b"},
		Ranges: []source.Range{
			{CIDR: "0.0.0.0/0", Provider: 0},
			{CIDR: "255.255.255.255/32", Provider: 1},
			{CIDR: "0.0.0.0/32", Provider: 1},
			{CIDR: "10.0.0.0/8", Provider: 1},
			{CIDR: "10.0.0.0/8", Provider: 0}, // tie: lower provider idx wins
			{CIDR: "::/0", Provider: 0},
			{CIDR: "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128", Provider: 1},
			{CIDR: "2001:db8::/64", Provider: 1},
			{CIDR: "2001:db8:0:0:8000::/65", Provider: 0},
			{CIDR: "1.2.3.4/24", Provider: 1}, // host bits set
			{CIDR: "bogus", Provider: 0},
			{CIDR: "9.9.9.0/24", Provider: 7}, // bad provider index
		},
	}
	ix := mustIndex(t, db)
	cases := map[string]string{
		"0.0.0.0":              "0.0.0.0/32",
		"0.0.0.1":              "0.0.0.0/0",
		"255.255.255.254":      "0.0.0.0/0",
		"255.255.255.255":      "255.255.255.255/32",
		"10.255.0.1":           "10.0.0.0/8",
		"1.2.3.99":             "1.2.3.0/24",
		"9.9.9.9":              "0.0.0.0/0",
		"::ffff:10.0.0.1":      "10.0.0.0/8",
		"::":                   "::/0",
		"2001:db8::1":          "2001:db8::/64",
		"2001:db8::8000:0:0:1": "2001:db8:0:0:8000::/65",
		"2001:db8:0:1::":       "::/0",
		"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff": "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128",
	}
	for ip, want := range cases {
		got := ix.result(ix.lookupAddr(netip.MustParseAddr(ip)))
		if got.CIDR != want {
			t.Errorf("%s: got %q, want %q", ip, got.CIDR, want)
		}
	}
	if got := ix.result(ix.lookupAddr(netip.MustParseAddr("10.1.1.1"))); got.Provider != "a" {
		t.Errorf("cross-provider tie: got %q", got.Provider)
	}
}

func TestParseIPv4MatchesNetip(t *testing.T) {
	for _, s := range []string{
		"", ".", "1.2.3", "1.2.3.4", "1.2.3.4.", ".1.2.3", "1..2.3", "0.0.0.0", "255.255.255.255",
		"256.1.1.1", "01.2.3.4", "1.2.3.04", "1.2.3.0", "1.2.3.4 ", "1.2.3.-4", "1.2.3.4%x",
		"1.2.3.1000", "00.0.0.0", "::1", "::ffff:1.2.3.4", "1.2.3.4/8",
	} {
		checkParseIPv4(t, s)
	}
}

func FuzzParseIPv4(f *testing.F) {
	f.Add("1.2.3.4")
	f.Add("01.2.3.4")
	f.Fuzz(checkParseIPv4)
}

func checkParseIPv4(t *testing.T, s string) {
	got, ok := parseIPv4(s)
	a, err := netip.ParseAddr(s)
	want := err == nil && a.Is4()
	if ok != want {
		t.Fatalf("parseIPv4(%q) ok=%v, netip ok=%v", s, ok, want)
	}
	if ok {
		b := a.As4()
		if got != uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]) {
			t.Fatalf("parseIPv4(%q) = %x, netip %s", s, got, a)
		}
	}
}
