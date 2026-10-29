package tools

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// decodeSnappy decodes one snappy block (the format GitHub's attestation
// store serves a bundle in when its API does not inline it: a ".json.sn"
// blob). It is the block format only, no framing, and it refuses an output
// longer than limit before allocating it. Hand-written rather than a new
// dependency: the format is a length and a run of literal and back-reference
// elements, and every byte it produces is then verified by Sigstore, so a
// decoding fault can only make a check fail, never pass.
func decodeSnappy(src []byte, limit int) ([]byte, error) {
	n, k := binary.Uvarint(src)
	if k <= 0 {
		return nil, errors.New("snappy: no length header")
	}
	if n > uint64(limit) {
		return nil, fmt.Errorf("snappy: %d bytes exceeds the %d-byte bound", n, limit)
	}
	dst := make([]byte, 0, int(n))
	s := src[k:]
	for len(s) > 0 {
		tag := s[0]
		switch tag & 3 {
		case 0: // literal
			length := int(tag >> 2)
			s = s[1:]
			if length >= 60 {
				w := length - 59
				if len(s) < w {
					return nil, errors.New("snappy: truncated literal length")
				}
				length = 0
				for i := 0; i < w; i++ {
					length |= int(s[i]) << (8 * i)
				}
				s = s[w:]
			}
			length++
			if length <= 0 || length > len(s) || len(dst)+length > int(n) {
				return nil, errors.New("snappy: literal overruns its input or its declared length")
			}
			dst = append(dst, s[:length]...)
			s = s[length:]
		default: // copy
			var length, offset int
			switch tag & 3 {
			case 1:
				if len(s) < 2 {
					return nil, errors.New("snappy: truncated copy")
				}
				length = 4 + int(tag>>2)&7
				offset = int(tag>>5)<<8 | int(s[1])
				s = s[2:]
			case 2:
				if len(s) < 3 {
					return nil, errors.New("snappy: truncated copy")
				}
				length = 1 + int(tag>>2)
				offset = int(binary.LittleEndian.Uint16(s[1:3]))
				s = s[3:]
			case 3:
				if len(s) < 5 {
					return nil, errors.New("snappy: truncated copy")
				}
				length = 1 + int(tag>>2)
				offset = int(binary.LittleEndian.Uint32(s[1:5]))
				s = s[5:]
			}
			if offset <= 0 || offset > len(dst) || len(dst)+length > int(n) {
				return nil, errors.New("snappy: copy reaches outside the output")
			}
			start := len(dst) - offset
			for i := 0; i < length; i++ { // overlapping copies repeat, byte by byte
				dst = append(dst, dst[start+i])
			}
		}
	}
	if len(dst) != int(n) {
		return nil, fmt.Errorf("snappy: decoded %d bytes, header declared %d", len(dst), n)
	}
	return dst, nil
}
