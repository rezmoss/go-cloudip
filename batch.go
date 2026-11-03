package cloudip

// LookupBatch returns one lookup result per IP in input order.
// All results use the same immutable database snapshot, even during an update.
func (d *Detector) LookupBatch(ips []string) []LookupResult {
	if ips == nil {
		return nil
	}
	results := make([]LookupResult, len(ips))
	state := d.state.Load()
	if state == nil {
		return results
	}
	for i, ip := range ips {
		results[i] = state.idx.result(state.lookupString(ip))
	}
	return results
}

// LookupBatch looks up multiple IP strings using the default detector.
func LookupBatch(ips []string) []LookupResult {
	return mustGetDefaultDetector().LookupBatch(ips)
}
