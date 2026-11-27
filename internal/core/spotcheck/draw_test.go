package spotcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func keysN(n int) []string {
	ks := make([]string, n)
	for i := range ks {
		ks[i] = fmt.Sprintf("%040x", i+1)
	}
	return ks
}

// TestDrawIsDeterministic: the same randomness and population give the same
// list, across two runs and a shuffled population; another randomness or
// another cycle gives another.
func TestDrawIsDeterministic(t *testing.T) {
	r, _ := hex.DecodeString(vectorRandomness)
	keys := keysN(50)
	a := drawKeys(r, "20261125T010203Z-0123456789ab", GroupDrainLanded, keys, 10)
	shuffled := slices.Clone(keys)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	b := drawKeys(r, "20261125T010203Z-0123456789ab", GroupDrainLanded, append(shuffled, keys[3]), 10)
	if len(a) != 10 || !slices.Equal(a, b) {
		t.Fatalf("two draws over one population differ:\n%v\n%v", a, b)
	}
	if slices.Equal(a, drawKeys(r, "20261125T010203Z-0123456789ac", GroupDrainLanded, keys, 10)) {
		t.Fatal("another cycle drew the same list")
	}
	if slices.Equal(a, drawKeys(r, "20261125T010203Z-0123456789ab", GroupAutomaticMerges, keys, 10)) {
		t.Fatal("another group drew the same list")
	}
	if got := drawKeys(r, "c", GroupDrainLanded, keys[:3], 10); len(got) != 3 {
		t.Fatalf("a group smaller than the size drew %d; want all 3", len(got))
	}
}

// TestEveryItemCanBeDrawn: over many randomness values, every key of a
// 50-item group is drawn at least once.
func TestEveryItemCanBeDrawn(t *testing.T) {
	keys := keysN(50)
	seen := map[string]bool{}
	for i := range 400 {
		sum := sha256.Sum256([]byte(fmt.Sprint(i)))
		for _, k := range drawKeys(sum[:], "c", GroupDrainLanded, keys, 10) {
			seen[k] = true
		}
	}
	for _, k := range keys {
		if !seen[k] {
			t.Fatalf("key %s was never drawn in 400 draws", k)
		}
	}
}

// TestLargePopulationStoresRangeAndHash: a population of up to 1,000 keys is
// kept whole and sorted; above it, by its first and last key and the hash of
// the sorted keys.
func TestLargePopulationStoresRangeAndHash(t *testing.T) {
	small := population([]string{"b", "a"})
	if small.Count != 2 || !slices.Equal(small.Keys, []string{"a", "b"}) || small.SHA256 != "" {
		t.Fatalf("population = %+v", small)
	}
	keys := keysN(maxListedKeys + 1)
	big := population(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n") + "\n"))
	if big.Count != maxListedKeys+1 || big.Keys != nil || big.First != keys[0] || big.Last != keys[maxListedKeys] || big.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("population of 1,001 = count %d, %d keys, %s..%s, %s", big.Count, len(big.Keys), big.First, big.Last, big.SHA256)
	}
	if whole := population(keys[:maxListedKeys]); len(whole.Keys) != maxListedKeys {
		t.Fatalf("a population of 1,000 kept %d keys whole", len(whole.Keys))
	}
}

// TestACycleListingEveryGroupWholeFitsItsBound: four groups of the most keys
// a population lists whole, each key as long as a record's, encode within a
// cycle's bound, so a busy window never makes a draw that cannot be written.
func TestACycleListingEveryGroupWholeFitsItsBound(t *testing.T) {
	keys := make([]string, maxListedKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("20261113T070000Z-iss-%016d", i)
	}
	c := Cycle{SchemaVersion: SchemaVersion, ID: cycleID(at0, sha('a')), Sample: Sample{Counted: []string{}}}
	for _, g := range groupOrder {
		c.Groups = append(c.Groups, GroupDraw{Group: g, Population: population(keys), Drawn: keys[:maxPerGroup]})
	}
	if _, err := encodeFile(CyclesDir, c.ID+".json", c); err != nil {
		t.Fatal(err)
	}
}
