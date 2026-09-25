package scanner

import (
	"sort"
	"strings"
	"testing"
)

// meterFixture is one adversarial line shape for TestScanLineWorkIsLinear,
// scanned under an identity: build(n) is the line at size n, and the guard
// compares n with 4n.
type meterFixture struct {
	name  string
	id    Identity
	build func(n int) string
}

// rep builds the commonest shape: one unit repeated n times.
func rep(unit string) func(int) string {
	return func(n int) string { return strings.Repeat(unit, n) }
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
	{"github_tokens", Identity{}, rep("ghp_" + strings.Repeat("a", 36) + " ")},
	{"public_ipv4", Identity{}, rep("8.8.8.8 ")},
	{"reserved_ipv4", Identity{}, rep("10.1.2.3 ")},
	{"ipv4_version_run", Identity{}, rep("10.1.")},
	{"public_ipv6", Identity{}, rep("2600:1f18:aaaa:bbbb:cccc:dddd:eeee:ffff ")},
	{"public_mac", Identity{}, rep("3c:22:fb:01:23:45 ")},
	{"lan_hosts", Identity{}, rep("a.local ")},
	{"lan_host_selectors", Identity{}, rep("x = cfg.local ")},
	{"device_hosts", Identity{}, rep("the alice-laptop ")},
	{"fingerprint_colon_run", Identity{}, rep("ab:")},
	{"footers", Identity{}, rep("generated with [x](y) ")},
	{"footer_openings_without_links", Identity{}, rep("generated with [x ")},
	{"footers_after_prose", Identity{}, func(n int) string { return strings.Repeat("prose ", n) + "generated with [x](y)" }},
	{"percent_encoded", meterNamedID, rep("%2Fhome%2Fzq8home ")},
	// Identity matchers.
	{"named_login_words", meterNamedID, rep("zq8home ")},
	{"named_login_dotted_run", meterNamedID, rep("zq8home.")},
	{"named_login_in_urls", meterNamedID, rep("https://h.example/x zq8home ")},
	{"own_home_paths", meterNamedID, rep("/Users/zq8home/x ")},                    // abcd-audit:allow
	{"own_home_url_roots", meterNamedID, rep("https://h.example/Users/zq8home ")}, // abcd-audit:allow
	{"own_email_and_login", meterNamedID, rep("zq8@example.test zq8home ")},
	{"github_handles", meterNamedID, rep("zq8handle https://h.example/x ")},
	{"root_home_url_paths", meterRootID, rep("https://h.example/root ")},
	{"root_home_behind_a_long_authority", meterRootID, func(n int) string {
		return "https://" + strings.Repeat("h", n) + ".example" + strings.Repeat("/root", n)
	}},
	{"root_home_behind_a_long_scp_host", meterRootID, func(n int) string {
		return "git@" + strings.Repeat("h", n) + ".example:" + strings.Repeat("/root", n)
	}},
	{"other_homes", Identity{}, rep("/home/bob/x ")},                  // abcd-audit:allow
	{"shared_home_traversal", Identity{}, rep("/Users/Shared/../x ")}, // abcd-audit:allow
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
			// Size the base so the small line is about 4 KB whatever the shape.
			base := max(4096/max(len(fx.build(1)), 1), 1)
			small, large := fx.build(base), fx.build(4*base)
			if len(large) < 3*len(small) {
				t.Fatalf("the shape does not scale with its parameter: %d bytes at base, %d at four times it", len(small), len(large))
			}
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

// TestBoundedContextHelpersKeepTheFinding pins the direction each bound fails
// in. A helper that stops reading at its bound cannot see the context that
// would have spared a match, so the match stands: the bound costs an
// over-report on an input no real text carries, never a leak.
func TestBoundedContextHelpersKeepTheFinding(t *testing.T) {
	// A login inside a dotted run longer than any identifier is reported, as a
	// bare mention would be.
	run := strings.Repeat("a.", maxDottedIdentifier) + "zq8home" + strings.Repeat(".b", 3)
	if f := ScanText(run, meterNamedID, DefaultPatterns(), DefaultIdentitySeverities(), "f"); !hasKind(f, kindLocalUser) {
		t.Errorf("a login inside an over-long dotted run was spared as a namespace component: %+v", f)
	}
	// A footer whose link text runs past the bound is reported even though its
	// target is a reserved documentation host.
	footer := "Generated with [" + strings.Repeat("x", maxFooterLinkText+1) + "](https://example.com)"
	if f := ScanText(footer, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "f"); !hasKind(f, kindHarnessFooter) {
		t.Errorf("a footer with an over-long link text was spared: %+v", f)
	}
	// Within the bound the exemption still holds.
	short := "Generated with [x](https://example.com)"
	if f := ScanText(short, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "f"); hasKind(f, kindHarnessFooter) {
		t.Errorf("a footer linking a reserved documentation host was reported: %+v", f)
	}
}
