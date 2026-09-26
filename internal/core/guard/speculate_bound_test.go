package guard

import (
	"strings"
	"testing"
)

// stdinCapBytes mirrors the 1 MiB cap both front doors put on a candidate
// (maxHookStdinBytes in cli.go, maxGuardStdinBytes in guard.go). It is the
// largest command line that can reach Check, so it is the size the bounds have
// to hold at — anything smaller measures a case an author would not choose.
const stdinCapBytes = 1 << 20

// The DoS gate, and it is load-bearing rather than an optimisation: the guard
// runs inside a PreToolUse hook, so a slow guard is an outage on every command in
// the session.
//
// The history is why this file is a table and not one case. An earlier attempt at
// iss-200 made the matcher re-tokenize and restart per entry, went quadratic, and
// was BLOCKED in review; expandPayloads' doc comment forbids that shape by name.
// Speculation reintroduced the same danger THREE times from three directions, and
// each was found by measuring rather than by reasoning:
//
//  1. 64 starts still each paid commandOf over the whole line — 14.2s;
//  2. the window capped the token COUNT, and one token can carry a megabyte of
//     payload that every start re-tokenized — 23.6s;
//  3. both bounds were per SEGMENT, and the segment count is unbounded — 8.4s.
//
// So every shape that produced one of those numbers is pinned here. A bound that
// is only tested on the shape it was written for is how the second and third
// arrived: adr-42 decision 3 says in as many words that "the benchmark must
// exercise the payload path, or it pins only the cheaper half", and the first
// version of this test pinned the cheaper half.
var boundCases = []struct {
	name string
	// build returns a candidate of roughly size bytes.
	build func(size int) string
	// why names the bound this shape defends, so a failure says which one broke.
	why string
}{
	{
		name: "repeated-wrapper",
		// Every start pays commandOf, which walks the leading wrapper chain.
		build: func(size int) string { return "myrunner " + strings.Repeat("env ", size/4) },
		why:   "maxSpeculativeStarts (a start cap), with commandOf O(N) per start",
	},
	{
		name: "payload-behind-wrappers",
		// 64 starts all resolve to the same `sh`, and each re-tokenizes the payload.
		// A token count says nothing about a token's size.
		build: func(size int) string {
			return "myrunner " + strings.Repeat("env ", 64) + "sh -c '" + strings.Repeat("a; ", size/3) + "'"
		},
		why: "maxSpeculativeExpansionBytes (a payload-bytes budget)",
	},
	{
		name: "many-segments",
		// Each segment is small and cheap; there are simply a great many of them,
		// and a per-segment bound gives each one a full allowance.
		build: func(size int) string {
			seg := "myrunner " + strings.Repeat("env ", 64) + "sh -c 'a; " + strings.Repeat("b; ", 80) + "' ; "
			return strings.Repeat(seg, size/len(seg))
		},
		why: "maxSpeculativeStartsPerCheck (a whole-line start budget)",
	},
	{
		name: "nested-payload",
		build: func(size int) string {
			return "myrunner sh -c \"sh -c '" + strings.Repeat("a; ", size/3) + "'\""
		},
		why: "the payload-bytes budget, at maxPayloadDepth",
	},
}

// TestSpeculationIsBoundedAtTheStdinCap holds every adversarial shape to linear
// work up to the size an author can actually submit: each shape is built at a
// quarter of the stdin cap and at the cap, and the guard's counted work may grow
// no faster than the input (assertWorkGrowth). It asserted a three-second
// wall-clock ceiling before, which a loaded gate run could trip with nothing
// wrong (iss-2609240046582859); every regression it defends against grew the
// work 16x or more per 4x of input, which the count sees on any machine.
func TestSpeculationIsBoundedAtTheStdinCap(t *testing.T) {
	for _, c := range boundCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if len(c.build(stdinCapBytes)) < stdinCapBytes/2 {
				t.Fatalf("the shape must be built near the %d-byte cap to say anything", stdinCapBytes)
			}
			_, d := assertWorkGrowth(t, c.build, stdinCapBytes/4, "the bound this shape defends is "+c.why)
			// It warns rather than allowing: the line is past every bound, so the
			// guard says it stopped looking. A silent allow here would be the defect,
			// not the cost.
			if d.Verdict == VerdictAllow {
				t.Errorf("verdict = %q: a candidate too long to inspect was waved through", d.Verdict)
			}
		})
	}
}

// BenchmarkCheck records what the two tiers cost on the shapes that matter, so a
// future change to matching can be compared rather than guessed at. The
// adversarial shapes are included at size for the same reason they are in the
// test above: an ordinary command line cannot show a bound working.
func BenchmarkCheck(b *testing.B) {
	cases := []struct{ name, cmd string }{
		{"ordinary", "git status --short"},
		{"tier1-block", "git push --force origin main"},
		{"tier2-warn", "myrunner git push --force origin main"},
		{"tier2-miss", "myrunner alpha beta gamma delta"},
		{"execstring", `su -c "gh repo delete owner/repo"`},
	}
	for _, c := range boundCases {
		cases = append(cases, struct{ name, cmd string }{"worst-" + c.name, c.build(stdinCapBytes)})
	}
	for _, c := range cases {
		r := Defaults()
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := r.Check(c.cmd); err != nil {
					b.Fatalf("Check: %v", err)
				}
			}
		})
	}
}
