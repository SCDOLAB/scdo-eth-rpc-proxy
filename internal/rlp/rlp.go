// Package rlp provides a minimal RLP decoder for parsing the "to" field
// from a raw signed transaction, without full decode.
package rlp

import "fmt"

// ParseToAddress extracts the "to" address (bytes 0..19 of the 20-byte field)
// from a legacy RLP-encoded transaction: [nonce, gasPrice, gas, to, value, data, v, r, s].
// Returns nil for contract creation (to is empty).
func ParseToAddress(tx []byte) ([]byte, error) {
	// RLP list header
	_, offset, err := parseListHeader(tx, 0)
	if err != nil {
		return nil, err
	}
	// Skip nonce (field 0)
	_, offset, err = skipItem(tx, offset)
	if err != nil {
		return nil, err
	}
	// Skip gasPrice (field 1)
	_, offset, err = skipItem(tx, offset)
	if err != nil {
		return nil, err
	}
	// Skip gas (field 2)
	_, offset, err = skipItem(tx, offset)
	if err != nil {
		return nil, err
	}
	// to (field 3)
	toData, toOffset, err := readItem(tx, offset)
	if err != nil {
		return nil, err
	}
	_ = toOffset
	if len(toData) == 0 {
		return nil, nil // contract creation
	}
	if len(toData) != 20 {
		return nil, fmt.Errorf("unexpected to length %d", len(toData))
	}
	return toData, nil
}

func parseListHeader(b []byte, pos int) (total int, next int, err error) {
	if pos >= len(b) {
		return 0, 0, fmt.Errorf("eof")
	}
	b0 := b[pos]
	switch {
	case b0 <= 0x7f:
		return 1, pos + 1, nil
	case b0 <= 0xb7:
		strLen := int(b0 - 0x80)
		return strLen + 1, pos + 1 + strLen, nil
	case b0 <= 0xbf:
		lenOfLen := int(b0 - 0xb7)
		if pos+1+lenOfLen > len(b) {
			return 0, 0, fmt.Errorf("eof in list len")
		}
		listLen := 0
		for i := 0; i < lenOfLen; i++ {
			listLen = listLen<<8 | int(b[pos+1+i])
		}
		return listLen + 1 + lenOfLen, pos + 1 + lenOfLen + listLen, nil
	case b0 <= 0xf7:
		listLen := int(b0 - 0xc0)
		return listLen + 1, pos + 1 + listLen, nil
	default:
		lenOfLen := int(b0 - 0xf7)
		if pos+1+lenOfLen > len(b) {
			return 0, 0, fmt.Errorf("eof in list len")
		}
		listLen := 0
		for i := 0; i < lenOfLen; i++ {
			listLen = listLen<<8 | int(b[pos+1+i])
		}
		return listLen + 1 + lenOfLen, pos + 1 + lenOfLen + listLen, nil
	}
}

func skipItem(b []byte, pos int) (int, int, error) {
	data, next, err := readItem(b, pos)
	return len(data), next, err
}

func readItem(b []byte, pos int) ([]byte, int, error) {
	if pos >= len(b) {
		return nil, 0, fmt.Errorf("eof")
	}
	b0 := b[pos]
	switch {
	case b0 <= 0x7f:
		return []byte{b0}, pos + 1, nil
	case b0 <= 0xb7:
		strLen := int(b0 - 0x80)
		end := pos + 1 + strLen
		if end > len(b) {
			return nil, 0, fmt.Errorf("eof in string")
		}
		return b[pos+1 : end], end, nil
	case b0 <= 0xbf:
		lenOfLen := int(b0 - 0xb7)
		endLen := pos + 1 + lenOfLen
		if endLen > len(b) {
			return nil, 0, fmt.Errorf("eof in long string len")
		}
		strLen := 0
		for i := 0; i < lenOfLen; i++ {
			strLen = strLen<<8 | int(b[pos+1+i])
		}
		end := endLen + strLen
		if end > len(b) {
			return nil, 0, fmt.Errorf("eof in long string")
		}
		return b[endLen:end], end, nil
	default:
		return nil, 0, fmt.Errorf("unexpected list item at %d", pos)
	}
}
