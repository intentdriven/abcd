// Package jsonstrict holds the strict-JSON checks encoding/json does not make:
// the one place a repeated object key is refused rather than read last-wins,
// shared by every reader whose input is a trust boundary (the rules overlay,
// the release-gate receipts).
package jsonstrict

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// NoDuplicateKeys walks the JSON token stream and refuses any object that
// carries a repeated key at any nesting level. It runs before the unmarshal precisely because encoding/json
// would otherwise collapse the duplicate silently. The stdlib decoder enforces a
// max nesting depth, so no separate depth guard is needed.
func NoDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		// A malformed or empty document is left for the unmarshal to report.
		return nil
	}
	return checkValue(dec, tok)
}

// checkValue recursively verifies the value whose opening token is tok. For an
// object it tracks the keys seen at that level; for an array it descends into each
// element. Scalars terminate. Any read error is swallowed as nil so the richer
// json.Unmarshal error remains the one the caller surfaces.
func checkValue(dec *json.Decoder, tok json.Token) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil
			}
			key, ok := kt.(string)
			if !ok {
				return nil
			}
			if seen[key] {
				return fmt.Errorf("duplicate key %q (last-wins is silent — refusing)", key)
			}
			seen[key] = true
			vt, err := dec.Token()
			if err != nil {
				return nil
			}
			if err := checkValue(dec, vt); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // closing '}'
			return nil
		}
	case '[':
		for dec.More() {
			vt, err := dec.Token()
			if err != nil {
				return nil
			}
			if err := checkValue(dec, vt); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // closing ']'
			return nil
		}
	}
	return nil
}
