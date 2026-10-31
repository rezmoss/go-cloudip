package cloudip

import "net/netip"

// lookupString: IP string -> ref (0 = miss/invalid). Fast IPv4 path, else netip.ParseAddr.
func (s *detectorState) lookupString(ip string) uint64 {
	if x, ok := parseIPv4(ip); ok {
		return s.idx.lookup4(x)
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return 0
	}
	return s.idx.lookupAddr(addr)
}

// parseIPv4: strict dotted quad, same rules as netip.ParseAddr. false on anything else.
func parseIPv4(s string) (uint32, bool) {
	var ip uint32
	var field, digits, dots int
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			if digits == 1 && field == 0 {
				return 0, false // leading zero
			}
			field = field*10 + int(c-'0')
			digits++
			if field > 255 {
				return 0, false
			}
		case c == '.' && digits > 0 && dots < 3:
			ip = ip<<8 | uint32(field)
			field, digits = 0, 0
			dots++
		default:
			return 0, false
		}
	}
	if dots != 3 || digits == 0 {
		return 0, false
	}
	return ip<<8 | uint32(field), true
}
