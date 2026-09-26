// Package jsonstrict holds the strict-JSON checks encoding/json does not make:
// the one place a repeated object key is refused rather than read last-wins,
// shared by every reader whose input is a trust boundary (the rules overlay,
// the release-gate receipts, the layered configuration files, the reading
// presets).
package jsonstrict

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// DuplicateKeyError names one object key a document repeats. Key is the later
// spelling and First the earlier one it collides with: equal for an exact
// repeat, different for a case twin encoding/json binds to the same field. Path
// is the chain of enclosing object keys from the root, "[]" standing for an
// array element. Callers that phrase their own refusal read the fields; Error is
// the general form.
type DuplicateKeyError struct {
	Path  []string
	Key   string
	First string
}

func (e *DuplicateKeyError) Error() string {
	where := ""
	if len(e.Path) > 0 {
		where = fmt.Sprintf(" under %q", strings.Join(e.Path, "."))
	}
	if e.Key == e.First {
		return fmt.Sprintf("duplicate key %q%s (last-wins is silent — refusing)", e.Key, where)
	}
	return fmt.Sprintf("duplicate key %q%s: it is %q spelt another way, and encoding/json binds "+
		"the two to one field case-insensitively (last-wins is silent — refusing)", e.Key, where, e.First)
}

// NoDuplicateKeys walks the JSON token stream and refuses any object that
// carries a repeated key at any nesting level, answering a *DuplicateKeyError.
// It runs before the unmarshal precisely because encoding/json would otherwise
// collapse the duplicate silently.
//
// "Repeated" is judged the way encoding/json matches a key to a struct field:
// after unescaping, and case-insensitively under Unicode simple folding (its
// foldName, which agrees with strings.EqualFold), so "VerificationResult",
// "VERIFICATIONRESULT" and "\u0056erificationResult" all repeat
// "verificationResult". The checker does not know the target type, so it folds
// in every object, map-shaped ones included: a map encoding/json would keep two
// entries in ("PII" and "pii") is refused too, because one name in two spellings
// is illegible to the reader the refusal protects.
//
// A malformed document answers nil and is left for the unmarshal to report with
// its position. The stdlib decoder enforces a max nesting depth, so no separate
// depth guard is needed.
func NoDuplicateKeys(data []byte) error {
	if !json.Valid(data) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil
	}
	return checkValue(dec, tok, nil)
}

// checkValue recursively verifies the value whose opening token is tok, at path.
// For an object it tracks the folded keys seen at that level; for an array it
// descends into each element. Scalars terminate. Any read error is swallowed as
// nil so the richer json.Unmarshal error remains the one the caller surfaces.
func checkValue(dec *json.Decoder, tok json.Token, path []string) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar
	}
	switch delim {
	case '{':
		seen := map[string]string{} // folded key -> the spelling seen first
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil
			}
			key, ok := kt.(string)
			if !ok {
				return nil
			}
			folded := fold(key)
			if first, dup := seen[folded]; dup {
				return &DuplicateKeyError{Path: append([]string(nil), path...), Key: key, First: first}
			}
			seen[folded] = key
			vt, err := dec.Token()
			if err != nil {
				return nil
			}
			if err := checkValue(dec, vt, append(path, key)); err != nil {
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
			if err := checkValue(dec, vt, append(path, "[]")); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // closing ']'
			return nil
		}
	}
	return nil
}

// fold maps a key to the form encoding/json compares field names in: each rune
// replaced by the smallest rune of its Unicode simple-fold orbit, which is the
// decoder's own foldRune. Two keys fold equal exactly when strings.EqualFold
// holds between them.
func fold(s string) string {
	return strings.Map(foldRune, s)
}

// foldRune returns the smallest rune in r's simple-fold orbit.
func foldRune(r rune) rune {
	for {
		r2 := unicode.SimpleFold(r)
		if r2 <= r {
			return r2
		}
		r = r2
	}
}
