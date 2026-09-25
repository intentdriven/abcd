package guard

import "testing"

// TestKillByPatternIsBlocked — iss-2609240646538696. A kill by name or pattern
// signals every matching process on the machine, other sessions' included: on
// 2026-09-23 `pkill -f "make preflight"` stopped two peer lanes' gates. The
// bundled registry had no entry for it, so the guard answered allow. The safe
// routes — the pid recorded at start, or the process's own group — stay
// allowed, including `pkill`'s own group and parent selectors, which carry no
// pattern at all.
func TestKillByPatternIsBlocked(t *testing.T) {
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`pkill -f 'make preflight'`, VerdictBlock, "pkill-by-pattern"},
		{`pkill make`, VerdictBlock, "pkill-by-pattern"},
		{`pkill -9 -f 'go test'`, VerdictBlock, "pkill-by-pattern"},
		{`pkill --signal TERM -f node`, VerdictBlock, "pkill-by-pattern"},
		{`sudo pkill -f node`, VerdictBlock, "pkill-by-pattern"},
		{`killall make`, VerdictBlock, "killall-by-name"},
		{`killall -9 node`, VerdictBlock, "killall-by-name"},
		{`killall -s KILL make`, VerdictBlock, "killall-by-name"},

		{`pkill -g 4242`, VerdictAllow, ""},
		{`pkill -P $$`, VerdictAllow, ""},
		{`kill 4242`, VerdictAllow, ""},
		{`kill -- -4242`, VerdictAllow, ""},
		{`pgrep -f 'make preflight'`, VerdictAllow, ""},
		{`killall -l`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d := verdictOf(t, tc.cmd)
			if d.Verdict != tc.want || d.EntryID != tc.entry {
				t.Errorf("verdict = %q via %q, want %q via %q", d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}
}

// TestMinOperandsConstraint pins the pattern field the kill entries use: an
// entry that requires N operands does not fire on a command carrying fewer,
// with the entry's value flags stepped over first.
func TestMinOperandsConstraint(t *testing.T) {
	p := Pattern{Command: "pkill", ValueFlags: []string{"-g"}, MinOperands: 1}
	for cmd, want := range map[string]bool{
		"pkill make":      true,
		"pkill -g 42":     false,
		"pkill -g 42 foo": true,
		"pkill":           false,
	} {
		segs, err := tokenize(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if got := matchSegment(p, segs[0]); got != want {
			t.Errorf("matchSegment(min_operands 1, %q) = %v, want %v", cmd, got, want)
		}
	}
	bad := Registry{SchemaVersion: 1, Entries: map[string]Entry{"x": {
		Tier: TierWarn, Why: "w", Successor: "s", Pattern: Pattern{Command: "x", MinOperands: -1},
	}}}
	if err := Validate(bad); err == nil {
		t.Error("a negative min_operands must be rejected at load")
	}
}
