// Package addr derives SCDO shard from a 20-byte address.
package addr

// DeriveShard computes shard 1..4 from a 20-byte address.
// When the node patch is active, byte[0] may not be a valid shard byte;
// we fall back to sum(bytes)%4+1 for deterministic routing.
func DeriveShard(addr []byte) uint {
	if len(addr) == 0 {
		return 1
	}
	b := addr[0]
	if b >= 1 && b <= 4 {
		return uint(b)
	}
	var sum byte
	for _, x := range addr {
		sum += x
	}
	return uint(sum%4) + 1
}
