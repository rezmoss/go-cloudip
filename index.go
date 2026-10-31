package cloudip

import (
	"cmp"
	"errors"
	"math/bits"
	"net/netip"
	"slices"
	"strings"

	"github.com/rezmoss/go-cloudip/internal/source"
)

// index: immutable LPM table. Prefixes flattened to sorted non-overlapping
// segs, each -> most specific prefix. Lookup = dir probe + branchless bsearch.
// ref = provider<<48 | meta<<24 | entry; 0 = no match.
type index struct {
	// dir4 by top 16 bits, sub4 by next 8 for dense /16s.
	// entry: dirLeaf|ref, dirSub|off into sub4, or lo<<32|hi span of starts4.
	dir4    []uint64
	sub4    []uint64
	starts4 []uint32
	refs4   []uint64

	// top6: every top6Stride-th start, small cache-hot first pass
	starts6 []u128
	refs6   []uint64
	top6    []u128

	providers []Provider
	metas     []meta
	cidrs     []string // by entry; 0 = sentinel
	tree      []node   // by entry; LookupAll only
	rows      []uint32 // meta ids per entry, best first
}

type meta struct {
	provider Provider
	region   string
	service  string
}

// node: parent entry + source rows
type node struct {
	parent uint32
	rowLo  uint32
	rowHi  uint32
}

const (
	dirLeaf = 1 << 63
	dirSub  = 1 << 62
	// segs per /16 above which it gets a /24 table
	subMinSpan = 8
	top6Stride = 16

	refBits = 24
	refMask = 1<<refBits - 1
)

func (ix *index) lookup4(x uint32) uint64 {
	d := ix.dir4[x>>16]
	if d&dirSub != 0 {
		d = ix.sub4[uint32(d)|x>>8&0xff]
	}
	if d&dirLeaf != 0 {
		return d &^ dirLeaf
	}
	s := ix.starts4
	base, n := int(d>>32), int(uint32(d))-int(d>>32)+1
	for n > 1 {
		half := n >> 1
		base += half & geMask32(x, s[base+half])
		n -= half
	}
	return ix.refs4[base]
}

func (ix *index) lookup6(x u128) uint64 {
	b := search6(ix.top6, 0, len(ix.top6), x) * top6Stride
	// count starts <= x in block; independent loads beat bisecting
	blk := ix.starts6[b+1 : min(b+top6Stride, len(ix.starts6))]
	i := b
	for _, s := range blk {
		i -= x.geMask(s)
	}
	return ix.refs6[i]
}

// search6: last i in [base, base+n) with s[i] <= x; needs s[base] <= x.
func search6(s []u128, base, n int, x u128) int {
	for n > 1 {
		half := n >> 1
		base += half & x.geMask(s[base+half])
		n -= half
	}
	return base
}

// geMask32: -1 if x >= y else 0. Branchless; branches mispredict on random input.
func geMask32(x, y uint32) int {
	return int((uint64(x)-uint64(y))>>63) - 1
}

func (ix *index) lookupAddr(a netip.Addr) uint64 {
	if a.Is4() {
		b := a.As4()
		return ix.lookup4(uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]))
	}
	if !a.Is6() {
		return 0
	}
	x := addrU128(a)
	if x.hi == 0 && x.lo>>32 == 0xffff {
		return ix.lookup4(uint32(x.lo)) // IPv4-mapped
	}
	return ix.lookup6(x)
}

func (ix *index) result(ref uint64) LookupResult {
	if ref == 0 {
		return LookupResult{}
	}
	m := &ix.metas[ref>>refBits&refMask]
	return LookupResult{Found: true, Provider: m.provider, Region: m.region, Service: m.service, CIDR: ix.cidrs[ref&refMask]}
}

func (ix *index) provider(ref uint64) Provider {
	return ix.providers[ref>>(2*refBits)]
}

// all: every row of every prefix containing ref's addr, most specific first.
func (ix *index) all(ref uint64) []LookupResult {
	var out []LookupResult
	for e := uint32(ref & refMask); e != 0; e = ix.tree[e].parent {
		nd := ix.tree[e]
		for _, r := range ix.rows[nd.rowLo:nd.rowHi] {
			m := &ix.metas[r]
			out = append(out, LookupResult{Found: true, Provider: m.provider, Region: m.region, Service: m.service, CIDR: ix.cidrs[e]})
		}
	}
	return out
}

// buildRow: small sort key per source line; metadata via src keeps sort cheap.
type buildRow struct {
	first u128
	tail  uint32 // bits<<24 | provider<<8 | rank (tie-breakers)
	src   int32  // db.Ranges idx
}

func (r *buildRow) bits() int { return int(r.tail >> 24) }
func (r *buildRow) prov() int { return int(r.tail >> 8 & 0xffff) }

// rowRank for same-prefix rows, lower wins: has region, then not umbrella
// tag (AMAZON, AzureCloud).
func rowRank(region, service string) int8 {
	var rank int8
	if region == "" {
		rank += 2
	}
	if service == "AMAZON" || service == "AZURECLOUD" || strings.HasPrefix(service, "AZURECLOUD.") {
		rank++
	}
	return rank
}

func (a *buildRow) samePrefix(b *buildRow) bool {
	return a.first == b.first && a.tail>>24 == b.tail>>24
}

func newIndex(db *source.Database) (*index, error) {
	if len(db.Providers) >= 1<<(64-2*refBits-2) {
		return nil, errors.New("too many providers")
	}
	ix := &index{providers: make([]Provider, len(db.Providers)+1)}
	for i, p := range db.Providers {
		ix.providers[i+1] = Provider(p)
	}

	// v4/v6 sorted apart, then concatenated
	rows := make([]buildRow, 0, len(db.Ranges))
	var rows6 []buildRow
	for i, r := range db.Ranges {
		p, err := netip.ParsePrefix(r.CIDR)
		if err != nil || r.Provider < 0 || r.Provider >= len(db.Providers) {
			continue
		}
		p = p.Masked()
		row := buildRow{tail: uint32(p.Bits())<<24 | uint32(r.Provider)<<8 | uint32(rowRank(r.Region, r.Service)), src: int32(i)}
		if a := p.Addr(); a.Is4() {
			b := a.As4()
			row.first = u128{0, uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])}
			rows = append(rows, row)
		} else {
			row.first = addrU128(a)
			rows6 = append(rows6, row)
		}
	}
	// addr, len (parents first), then provider, rank, name: independent of file order
	compare := func(a, b buildRow) int {
		if a.first.hi != b.first.hi {
			return cmp.Compare(a.first.hi, b.first.hi)
		}
		if a.first.lo != b.first.lo {
			return cmp.Compare(a.first.lo, b.first.lo)
		}
		if a.tail != b.tail {
			return cmp.Compare(a.tail, b.tail)
		}
		ra, rb := &db.Ranges[a.src], &db.Ranges[b.src]
		return cmp.Or(strings.Compare(ra.Service, rb.Service), strings.Compare(ra.Region, rb.Region))
	}
	slices.SortFunc(rows, compare)
	slices.SortFunc(rows6, compare)
	rows4 := len(rows)
	rows = append(rows, rows6...)

	metaOf := func(r *buildRow) meta {
		src := &db.Ranges[r.src]
		return meta{provider: ix.providers[r.prov()+1], region: src.Region, service: src.Service}
	}

	ix.cidrs = make([]string, 1, len(rows)+1)
	ix.tree = make([]node, 1, len(rows)+1)
	ix.rows = make([]uint32, 0, len(rows))
	type entryInfo struct {
		first, last u128
		ref         uint64
		v6          bool
	}
	ents := make([]entryInfo, 1, len(rows)+1)
	metaIDs := make(map[meta]uint32)
	internMeta := func(m meta) uint32 {
		id, ok := metaIDs[m]
		if !ok {
			id = uint32(len(ix.metas))
			metaIDs[m] = id
			ix.metas = append(ix.metas, m)
		}
		return id
	}
	cidrBuf := make([]byte, 0, len(rows)*20)
	cidrEnds := make([]int, 0, len(rows))
	var prev meta
	for i := range rows {
		r := &rows[i]
		m := metaOf(r)
		newPfx := i == 0 || i == rows4 || !r.samePrefix(&rows[i-1])
		if !newPfx && m == prev {
			continue // exact dup
		}
		prev = m
		id := internMeta(m)
		if newPfx {
			e := len(ents)
			v6 := i >= rows4
			bitLen := 32
			if v6 {
				bitLen = 128
			}
			ents = append(ents, entryInfo{
				first: r.first,
				last:  r.first.withHostBits(bitLen - r.bits()),
				ref:   uint64(r.prov()+1)<<(2*refBits) | uint64(id)<<refBits | uint64(e),
				v6:    v6,
			})
			ix.tree = append(ix.tree, node{rowLo: uint32(len(ix.rows))})
			pfx := netip.PrefixFrom(u128Addr(r.first, v6), r.bits())
			cidrBuf = pfx.AppendTo(cidrBuf)
			cidrEnds = append(cidrEnds, len(cidrBuf))
		}
		ix.rows = append(ix.rows, id)
		ix.tree[len(ix.tree)-1].rowHi = uint32(len(ix.rows))
	}
	if len(ents) > refMask || len(ix.metas) > refMask {
		return nil, errors.New("too many ranges")
	}
	all := string(cidrBuf)
	start := 0
	for _, end := range cidrEnds {
		ix.cidrs = append(ix.cidrs, all[start:end])
		start = end
	}

	// inheritRegion: empty region <- nearest same-provider ancestor (Azure
	// global tags lack region; regional tag covers parent). Parents swept first.
	inheritRegion := func(ref uint64, parent uint32) uint64 {
		m := ix.metas[ref>>refBits&refMask]
		if m.region != "" {
			return ref
		}
		for p := parent; p != 0; p = ix.tree[p].parent {
			pm := ix.metas[ents[p].ref>>refBits&refMask]
			if pm.provider == m.provider && pm.region != "" {
				m.region = pm.region
				return ref&^(refMask<<refBits) | uint64(internMeta(m))<<refBits
			}
		}
		return ref
	}

	// sweep: ents[lo:hi] of one family -> segs covering family; sets parents
	sweep := func(lo, hi int, maxAddr u128) (starts []u128, refs []uint64) {
		starts = make([]u128, 1, 2*(hi-lo)+1)
		refs = make([]uint64, 1, 2*(hi-lo)+1)
		emit := func(at u128, ref uint64) {
			n := len(starts)
			switch {
			case starts[n-1] == at:
				refs[n-1] = ref
				if n > 1 && refs[n-2] == ref {
					starts, refs = starts[:n-1], refs[:n-1]
				}
			case refs[n-1] != ref:
				starts, refs = append(starts, at), append(refs, ref)
			}
		}
		stack := make([]int, 0, 64)
		pop := func() {
			t := ents[stack[len(stack)-1]]
			stack = stack[:len(stack)-1]
			if t.last == maxAddr {
				return
			}
			var ref uint64
			if len(stack) > 0 {
				ref = ents[stack[len(stack)-1]].ref
			}
			emit(t.last.inc(), ref)
		}
		for e := lo; e < hi; e++ {
			for len(stack) > 0 && ents[stack[len(stack)-1]].last.less(ents[e].first) {
				pop()
			}
			if len(stack) > 0 {
				ix.tree[e].parent = uint32(stack[len(stack)-1])
				ents[e].ref = inheritRegion(ents[e].ref, ix.tree[e].parent)
			}
			emit(ents[e].first, ents[e].ref)
			stack = append(stack, e)
		}
		for len(stack) > 0 {
			pop()
		}
		return starts, refs
	}

	split := len(ents)
	for e := 1; e < len(ents); e++ {
		if ents[e].v6 {
			split = e
			break
		}
	}
	starts, refs := sweep(1, split, u128{0, 1<<32 - 1})
	ix.starts4 = make([]uint32, len(starts))
	for i, s := range starts {
		ix.starts4[i] = uint32(s.lo)
	}
	ix.refs4 = refs
	ix.buildDir4()

	ix.starts6, ix.refs6 = sweep(split, len(ents), u128{^uint64(0), ^uint64(0)})
	for i := 0; i < len(ix.starts6); i += top6Stride {
		ix.top6 = append(ix.top6, ix.starts6[i])
	}
	// drop build headroom, index is long-lived
	ix.refs4 = slices.Clone(ix.refs4)
	ix.starts6, ix.refs6 = slices.Clone(ix.starts6), slices.Clone(ix.refs6)
	ix.cidrs, ix.tree = slices.Clone(ix.cidrs), slices.Clone(ix.tree)
	return ix, nil
}

func (ix *index) buildDir4() {
	ix.dir4 = make([]uint64, 1<<16)
	s := ix.starts4
	j := 0 // seg holding block's first addr
	// block: dir entry for [first, last]
	block := func(first, last uint32) (span int, d uint64) {
		for j+1 < len(s) && s[j+1] <= first {
			j++
		}
		k := j
		for k+1 < len(s) && s[k+1] <= last {
			k++
		}
		if j == k {
			return 1, dirLeaf | ix.refs4[j]
		}
		return k - j + 1, uint64(j)<<32 | uint64(k)
	}
	var spans [1 << 16]int32
	subs := 0
	for b := range uint32(1 << 16) {
		lo := j
		span, d := block(b<<16, b<<16|0xffff)
		ix.dir4[b] = d
		if span > subMinSpan {
			spans[b] = int32(lo) + 1
			subs++
		}
	}
	ix.sub4 = make([]uint64, 0, subs*256)
	for b := range uint32(1 << 16) {
		if spans[b] == 0 {
			continue
		}
		j = int(spans[b] - 1)
		ix.dir4[b] = dirSub | uint64(len(ix.sub4))
		for c := range uint32(256) {
			_, sd := block(b<<16|c<<8, b<<16|c<<8|0xff)
			ix.sub4 = append(ix.sub4, sd)
		}
	}
}

type u128 struct{ hi, lo uint64 }

func (a u128) less(b u128) bool { return a.hi < b.hi || a.hi == b.hi && a.lo < b.lo }

// geMask: -1 if a >= b else 0, branchless
func (a u128) geMask(b u128) int {
	_, borrow := bits.Sub64(a.lo, b.lo, 0)
	_, borrow = bits.Sub64(a.hi, b.hi, borrow)
	return int(borrow) - 1
}

func (a u128) inc() u128 {
	lo, carry := bits.Add64(a.lo, 1, 0)
	return u128{a.hi + carry, lo}
}

// withHostBits: a with low n bits set
func (a u128) withHostBits(n int) u128 {
	switch {
	case n == 0:
		return a
	case n < 64:
		return u128{a.hi, a.lo | (1<<n - 1)}
	case n == 128:
		return u128{^uint64(0), ^uint64(0)}
	default:
		return u128{a.hi | (1<<(n-64) - 1), ^uint64(0)}
	}
}

func addrU128(a netip.Addr) u128 {
	b := a.As16()
	return u128{beUint64(b[:8]), beUint64(b[8:])}
}

func u128Addr(x u128, v6 bool) netip.Addr {
	if !v6 {
		return netip.AddrFrom4([4]byte{byte(x.lo >> 24), byte(x.lo >> 16), byte(x.lo >> 8), byte(x.lo)})
	}
	var b [16]byte
	for i := range 8 {
		b[i], b[8+i] = byte(x.hi>>(56-8*i)), byte(x.lo>>(56-8*i))
	}
	return netip.AddrFrom16(b)
}

func beUint64(b []byte) uint64 {
	_ = b[7]
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}
