package spotcheck

// draw.go is the draw itself (spc-2610301925429250 approach 4): within each
// group, every key is given a keyed hash of the beacon's randomness, and the
// smallest are drawn. The same randomness and population give the same list,
// every key has a chance, and none can be steered, since the randomness does
// not exist until the window has closed.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// drawKeys is the size keys of a group drawn by randomness for the window
// seed names (drawSeed): each key, sorted and once, is given
// HMAC-SHA256(randomness, seed NUL group NUL key), and the keys of the size
// smallest values are answered in that order (DrawRule).
func drawKeys(randomness []byte, seed, group string, keys []string, size int) []string {
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	type scored struct {
		mac []byte
		key string
	}
	all := make([]scored, 0, len(sorted))
	for _, k := range sorted {
		m := hmac.New(sha256.New, randomness)
		m.Write([]byte(seed + "\x00" + group + "\x00" + k))
		all = append(all, scored{mac: m.Sum(nil), key: k})
	}
	slices.SortFunc(all, func(a, b scored) int {
		if c := strings.Compare(string(a.mac), string(b.mac)); c != 0 {
			return c
		}
		return strings.Compare(a.key, b.key)
	})
	out := make([]string, 0, min(size, len(all)))
	for i := 0; i < len(all) && i < size; i++ {
		out = append(out, all[i].key)
	}
	return out
}

// population is a group's keys as a cycle keeps them: whole up to
// maxListedKeys, and above it by the first and last key and the SHA-256 of
// the sorted keys, one per line. Only a landing group reaches the second
// form: a record group never takes more than maxListedKeys (windowRecords).
func population(keys []string) Population {
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	p := Population{Count: len(sorted)}
	if len(sorted) <= maxListedKeys {
		p.Keys = sorted
		if p.Keys == nil {
			p.Keys = []string{}
		}
		return p
	}
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\n") + "\n"))
	p.First, p.Last, p.SHA256 = sorted[0], sorted[len(sorted)-1], hex.EncodeToString(sum[:])
	return p
}
