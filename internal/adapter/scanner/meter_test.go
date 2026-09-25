package scanner

import (
	"sort"
	"strings"
	"testing"
)

// meterFixture is one adversarial line shape for TestScanLineWorkIsLinear: a
// unit repeated to fill the line, scanned under an identity.
type meterFixture struct {
	name string
	id   Identity
	unit string
}

// Reserved identity shapes. meterGenericID's login is on the generic-account
// floor, so the per-match position checks run for every occurrence;
// meterNamedID's is not, so every occurrence is a finding and the suppressions
// run instead. meterRootID is the single-segment home.
var (
	meterGenericID = Identity{HomePath: "/home/dev", HomeUser: "dev"} // abcd-audit:allow
	meterNamedID   = Identity{
		GitUserName:       "Zed Q Eight",
		GitUserEmail:      "zq8@example.test",
		GitRemoteUsername: "zq8handle",
		HomePath:          "/Users/zq8home", // abcd-audit:allow
		HomeUser:          "zq8home",
	}
	meterRootID = Identity{HomePath: "/root", HomeUser: "root"}
)

// meterFixtures are the shapes every stage of a line's scan is held linear on.
// Each packs one stage's per-match work as densely as a line allows: a match
// every few bytes, each one asking its position check, suppression or context
// helper about the line around it.
var meterFixtures = []meterFixture{
	// Secret patterns, their Skip callbacks and the adjacency probes.
	{"github_tokens", Identity{}, "ghp_" + strings.Repeat("a", 36) + " "},
	{"public_ipv4", Identity{}, "8.8.8.8 "},
	{"reserved_ipv4", Identity{}, "10.1.2.3 "},
	{"ipv4_version_run", Identity{}, "10.1."},
	{"public_ipv6", Identity{}, "2600:1f18:aaaa:bbbb:cccc:dddd:eeee:ffff "},
	{"public_mac", Identity{}, "3c:22:fb:01:23:45 "},
	{"lan_hosts", Identity{}, "a.local "},
	{"lan_host_selectors", Identity{}, "x = cfg.local "},
	{"device_hosts", Identity{}, "the alice-laptop "},
	{"percent_encoded", meterNamedID, "%2Fhome%2Fzq8home "},
	// Identity matchers.
	{"named_login_words", meterNamedID, "zq8home "},
	{"other_homes", Identity{}, "/home/bob/x "},                  // abcd-audit:allow
	{"shared_home_traversal", Identity{}, "/Users/Shared/../x "}, // abcd-audit:allow
}

// TestScanLineWorkIsLinear is the cost-class guard for the whole of a line's
// scan (iss-2609240203462704). probeWork holds the adjacency probes; this
// holds every stage scanMeter charges — each pattern's pass, the Skip and
// SkipAt callbacks, the identity matchers over the raw and the decoded line
// with their per-match position checks, and the percent-decode passes. It
// quadruples each shape and fails when any stage's charge, or the total, grows
// past linearCostBar: a helper that reads the line around every match turns a
// line dense in matches into quadratic work, which the growth shows and a
// fixed ceiling at one size does not.
func TestScanLineWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	for _, fx := range meterFixtures {
		t.Run(fx.name, func(t *testing.T) {
			base := 4096 / len(fx.unit)
			small := strings.Repeat(fx.unit, base)
			large := strings.Repeat(fx.unit, 4*base)
			lo, hi := meterScan(small, fx.id), meterScan(large, fx.id)
			var loTotal, hiTotal int
			stages := map[string]bool{}
			for s, n := range lo {
				loTotal += n
				stages[s] = true
			}
			for s, n := range hi {
				hiTotal += n
				stages[s] = true
			}
			if loTotal == 0 {
				t.Fatalf("the %d-byte shape charged no stage; it pins nothing", len(small))
			}
			names := make([]string, 0, len(stages))
			for s := range stages {
				names = append(names, s)
			}
			sort.Strings(names)
			for _, s := range names {
				// A stage with nothing to do at the small size has no ratio to
				// take; it is held to the line's own length instead.
				allowed := linearCostBar * float64(lo[s])
				if lo[s] == 0 {
					allowed = float64(len(large))
				}
				if float64(hi[s]) > allowed {
					t.Errorf("stage %s: quadrupling the line (%d -> %d bytes) took its charge %d -> %d (%.1fx), want at most %.1fx: a per-match step reads the line around every match",
						s, len(small), len(large), lo[s], hi[s], float64(hi[s])/float64(max(lo[s], 1)), linearCostBar)
				}
			}
			growth := float64(hiTotal) / float64(loTotal)
			t.Logf("%d -> %d bytes; charged %d -> %d (%.2fx, bar %.1fx)", len(small), len(large), loTotal, hiTotal, growth, linearCostBar)
			if growth > linearCostBar {
				t.Errorf("quadrupling the line multiplied the whole scan's charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}

// meterScan runs ScanText over line under a tallying meter and returns the
// charge per stage.
func meterScan(line string, id Identity) map[string]int {
	got := map[string]int{}
	scanMeter.tally = func(stage string, n int) { got[stage] += n }
	defer func() { scanMeter.tally = nil }()
	ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	return got
}

// TestScanMeterIsInertByDefault pins the production half of the seam: nothing
// is installed unless a test installs it, so a scan charges no one.
func TestScanMeterIsInertByDefault(t *testing.T) {
	if scanMeter.tally != nil {
		t.Fatal("scanMeter carries a tally outside a test that installed one")
	}
	scanMeter.charge(stagePattern, 1) // must not panic with nothing installed
}
