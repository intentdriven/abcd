package scanner

import (
	"strings"
	"testing"
)

// escapeSeparators spells every '/' of p with sep, so a fixture names a home
// path in an escaped spelling without the committed file carrying one.
func escapeSeparators(p, sep string) string { return strings.ReplaceAll(p, "/", sep) }

// uSolidus and uSolidusUpper are the JSON \u escape of '/', assembled so no
// editor or tool reading the source can fold the six bytes back into a '/'.
var (
	uSolidus      = `\` + "u002f"
	uSolidusUpper = `\` + "u002F"
)

// TestSweepCallerHomeReadsEscapedSpellings pins the literal-home backstop to
// the spellings the detector reads through its decoded views
// (iss-2609261659041553). The backstop exists for the case where both
// detector passes miss the home, and against an escaped home it read nothing:
// the JSON solidus escape, its / form in either case, a second JSON layer,
// the percent-encoded separator, and a control escape standing before a
// single-segment home all passed it verbatim. Each is collapsed to "~" exactly
// as the literal home is, and only the home's own bytes are rewritten.
func TestSweepCallerHomeReadsEscapedSpellings(t *testing.T) {
	const home = "/Users/maya" // a registry persona home, abcd-lint:allow
	cases := []struct{ name, home, in, want string }{
		{"solidus escape", home, escapeSeparators(home+"/x", `\/`), `~\/x`},
		{"unicode escape", home, escapeSeparators(home+"/x", uSolidus), "~" + uSolidus + "x"},
		{"upper-case unicode escape", home, escapeSeparators(home+"/x", uSolidusUpper), "~" + uSolidusUpper + "x"},
		{"second JSON layer", home, escapeSeparators(home+"/x", `\\\/`), `~\\\/x`},
		{"percent-encoded separator", home, escapeSeparators(home+"/x", `%2F`), `~%2Fx`},
		{"lower-case percent", home, escapeSeparators(home+"/x", `%2f`), `~%2fx`},
		{"mixed spellings", home, `\/Users/maya\/x`, `~\/x`},
		{"inside a JSON string", home, `{"cwd":"` + escapeSeparators(home, `\/`) + `"}`, `{"cwd":"~"}`},
		{"control escape before a single-segment home", "/root", `a\n/root/x`, `a\n~/x`},
		// The anchor holds in the decoded view exactly as on the literal line.
		{"a longer name after an escaped home", home, escapeSeparators(home+"field", `\/`), escapeSeparators(home+"field", `\/`)},
		{"a single-segment home under another root", "/root", `x\/root\/y`, `x\/root\/y`},
		{"an escape with no home in it", home, `line one\nline two`, `line one\nline two`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SweepCallerHome(c.in, c.home); got != c.want {
				t.Errorf("SweepCallerHome(%q, home=%q) = %q, want %q", c.in, c.home, got, c.want)
			}
		})
	}
}

// TestSweepCallerHomeKeepsEveryOtherLineIntact pins the sweep's reach across a
// multi-line text: an escaped home on one line is swept, and the lines around
// it, which carry escapes of their own, are left byte-for-byte as written.
func TestSweepCallerHomeKeepsEveryOtherLineIntact(t *testing.T) {
	const home = "/Users/maya" // abcd-lint:allow
	in := "first\\tline\n" + escapeSeparators(home+"/notes", `\/`) + "\nlast %41 line\n"
	want := "first\\tline\n" + `~\/notes` + "\nlast %41 line\n"
	if got := SweepCallerHome(in, home); got != want {
		t.Errorf("SweepCallerHome = %q, want %q", got, want)
	}
}

// TestSurvivingCallerHomeReadsEscapedSpellings pins the second backstop to
// the same views: the caller's user segment behind another root is rewritten
// in its escaped spellings too, and an escaped home that survived is reported.
func TestSurvivingCallerHomeReadsEscapedSpellings(t *testing.T) {
	const home = "/home/maya" // abcd-lint:allow
	for _, sep := range []string{`\/`, uSolidus, `%2F`} {
		in := escapeSeparators("/Users/maya/notes", sep) // abcd-lint:allow
		out, _ := SurvivingCallerHome(in, home)
		if strings.Contains(out, "maya") {
			t.Errorf("SurvivingCallerHome(%q) kept the caller's name: %q", in, out)
		}
	}
	if _, resid := SurvivingCallerHome(escapeSeparators("/opt/root/x", `\/`), "/opt/root"); len(resid) == 0 {
		t.Error("an escaped caller home that survived was not reported")
	}
}

// TestSweepCallerHomeWorkIsLinear holds the sweep's decoded views to the cost
// class the line scan is held to: quadrupling a text dense in escaped homes,
// escape runs and percent triples at most multiplies the charged work by
// linearCostBar.
func TestSweepCallerHomeWorkIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	shapes := []struct{ name, home, unit string }{
		{"escaped homes", "/Users/zq8home", `\/Users\/zq8home\/x `}, // abcd-lint:allow
		{"escaped single-segment homes", "/root", `\n\/root`},
		{"percent homes", "/root", `%2Froot`},
		{"escape runs", "/root", `\\\\\\\"\\\\n`},
		{"escaped homes across lines", "/root", "\\/root\n"},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			charge := func(text string) int {
				total := 0
				scanMeter.tally = func(_ string, n int) { total += n }
				defer func() { scanMeter.tally = nil }()
				SweepCallerHome(text, s.home)
				return total
			}
			base := max(4096/len(s.unit), 1)
			lo, hi := charge(strings.Repeat(s.unit, base)), charge(strings.Repeat(s.unit, 4*base))
			if lo == 0 {
				t.Fatal("the shape charged nothing; it pins nothing")
			}
			if growth := float64(hi) / float64(lo); growth > linearCostBar {
				t.Errorf("quadrupling the text multiplied the sweep's charge by %.2fx, want at most %.1fx", growth, linearCostBar)
			}
		})
	}
}
