package scanner

import (
	"strings"
	"testing"
)

// TestGluedSweepWorkIsLinear pins the sweep's cost class in the manner of the
// adjacency guards: a line of underscore-joined words, with and without glued
// tokens in it, quadrupled, at most multiplies the sweep's charge by the
// package's linear bar. Re-scanning every suffix would square it. The escaped
// shapes pin the sweep over the decoded views (iss-2609290743362554).
func TestGluedSweepWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	pat, _, akia, _ := gluedTokens()
	bs := string(rune(0x5c))
	shapes := []struct {
		name  string
		build func(n int) string
	}{
		{"underscore-joined words", func(n int) string { return strings.Repeat("notes_", n) }},
		{"underscore-joined glued tokens", func(n int) string { return strings.Repeat("notes_"+pat+"_", n) }},
		{"letter-glued access keys", func(n int) string { return strings.Repeat("x"+akia, n) }},
		{"a long word run with a prefix at every step", func(n int) string { return strings.Repeat("ghp_AKIA", n) }},
		{"percent-escaped glued tokens", func(n int) string { return strings.Repeat("notes_%67"+pat[1:]+"_", n) }},
		{"JSON-escaped glued keys", func(n int) string { return strings.Repeat("x"+bs+"u0041"+akia[1:], n) }},
	}
	patterns := DefaultPatterns()
	for _, sh := range shapes {
		t.Run(sh.name, func(t *testing.T) {
			base := max(4096/max(len(sh.build(1)), 1), 1)
			small, large := sh.build(base), sh.build(4*base)
			charge := func(line string) int {
				n := 0
				scanMeter.tally = func(_ string, k int) { n += k }
				defer func() { scanMeter.tally = nil }()
				gluedFindings(line, patterns, "f")
				return n
			}
			lo, hi := charge(small), charge(large)
			if lo == 0 {
				t.Fatalf("the %d-byte shape charged nothing; it pins nothing", len(small))
			}
			growth := float64(hi) / float64(lo)
			t.Logf("%d -> %d bytes; charged %d -> %d (%.2fx, bar %.1fx)", len(small), len(large), lo, hi, growth, linearCostBar)
			if growth > linearCostBar {
				t.Errorf("quadrupling the line multiplied the glued sweep's charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}
