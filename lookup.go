package cloudip

import (
	"net"
	"net/netip"
)

// lookup performs a lookup in the trie for the given IP address.
// Returns the matching range entry or nil if not found.
func (s *detectorState) lookup(ip net.IP) *rangeEntry {
	if s == nil || s.ranger == nil {
		return nil
	}

	entries, err := s.ranger.ContainingNetworks(ip)
	if err != nil || len(entries) == 0 {
		return nil
	}

	// Return the most specific (smallest) matching network
	// cidranger returns entries in order, but we want the most specific
	var best *rangeEntry
	var bestSize int = -1

	for _, e := range entries {
		entry, ok := e.(*rangeEntry)
		if !ok {
			continue
		}

		// Calculate network size (smaller = more specific)
		ones, _ := entry.network.Mask.Size()
		if bestSize < 0 || ones > bestSize {
			best = entry
			bestSize = ones
		}
	}

	return best
}

// lookupAddr performs a lookup using netip.Addr.
func (s *detectorState) lookupAddr(addr netip.Addr) *rangeEntry {
	return s.lookup(netIPToNetIP(addr))
}

// lookupString performs a lookup using an IP string.
func (s *detectorState) lookupString(ipStr string) *rangeEntry {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil
	}
	return s.lookupAddr(addr)
}

// toLookupResult converts a range entry to a LookupResult.
func (e *rangeEntry) toLookupResult() LookupResult {
	if e == nil {
		return LookupResult{Found: false}
	}
	return LookupResult{
		Found:    true,
		Provider: e.provider,
		Region:   e.region,
		Service:  e.service,
		CIDR:     e.cidr,
	}
}
