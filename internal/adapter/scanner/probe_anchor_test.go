package scanner

import (
	"strings"
	"testing"
)

// probe_anchor_test.go — iss-2609240203342061: the \A anchor adjacencyProbe
// adds was untested. scanAllPatterns discards a probe hit that does not start
// at offset 0, so an unanchored probe returned no wrong finding and every
// count-based guard kept passing — the count tallies the bytes a probe is
// handed, which the anchor does not change — while each attempt searched its
// whole window for a leftmost match instead of failing at byte 0. The anchor
// is pinned where it lives: a probe built for any bundled pattern must not
// find that pattern's token anywhere but at the start of the string.
func TestAdjacencyProbeMatchesOnlyAtOffsetZero(t *testing.T) {
	tokens := map[string]string{
		"github_pat":          "ghp_" + strings.Repeat("a1", 20),
		"aws_access_key":      "AKIA" + strings.Repeat("Q7", 8),
		"net_ipv4":            "10.1.2.3",
		"net_device_hostname": "bobs-macbook",
	}
	for _, p := range DefaultPatterns() {
		tok, ok := tokens[p.Name]
		if !ok {
			continue
		}
		probe := adjacencyProbe(p.Re)
		if loc := probe.FindStringIndex(tok); loc == nil || loc[0] != 0 {
			t.Fatalf("%s: the probe does not match its own token at offset 0: %v", p.Name, loc)
		}
		if loc := probe.FindStringIndex("filler text " + tok); loc != nil {
			t.Errorf("%s: the probe found a token at offset %d — it is not anchored to the start", p.Name, loc[0])
		}
		delete(tokens, p.Name)
	}
	if len(tokens) != 0 {
		t.Fatalf("patterns not found in the bundled set: %v", tokens)
	}
}
