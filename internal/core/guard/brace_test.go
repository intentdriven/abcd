package guard

import (
	"strings"
	"testing"
)

// TestUnquotedBraceGroupIsRefused is the repro for iss-2608221457227161. Bash
// expands `{--force,}` to byte-identical `--force` argv, but the guard's
// tokenizer read the literal token `{--force,}`, no blocker matched it, and a
// Tier-1 hazard was a silent allow — the same mutate-the-flag-token shape the
// redirection fix closed. The guard first refused every unquoted group; it now
// expands the group as bash does (iss-2608282026038930), so each shape here
// blocks because the argv bash builds from it is a hazard an entry names.
// Shapes bash expands to something harmless — `{{--force,--dry-run}}` keeps its
// outer braces, `\${--force,}` yields `$--force` — are pinned against bash in
// TestBraceExpansionMatchesBash instead.
func TestUnquotedBraceGroupIsRefused(t *testing.T) {
	for _, cmd := range []string{
		// The reported shape: a single-element group with an empty alternative.
		`git push {--force,} origin main`,
		// The same trick spelled the other way round, and on the long flag.
		`git push {,--force} origin main`,
		`git push {--force-with-lease,} origin main`,
		// A real two-alternative group, and a nested one whose comma is not at
		// the group's own top level.
		`git push {--force,--dry-run} origin main`,
		// A range, the other expansion form.
		`cd scratch && rm -rf dir{1..9}`,
		// The group hidden one execute-a-string layer down, where the outer
		// quotes exempt it but the payload's own tokenize does not.
		`sh -c 'git push {--force,} origin main'`,
		// An alternative holding a command substitution. `(`, `)` and a
		// backtick do NOT end a bash word when they open one, and bash
		// brace-expands straight through: `printf '[%s]' {--force,$(true)}`
		// prints `[--force]`. A scan that treated them as word terminators read
		// "no group here" exactly where bash reads one, leaving the reported
		// bypass open under a nine-character variant of itself.
		"git push {--force,$(true)} origin main",
		"git push {--force,`echo x`} origin main",
		"git push {--force,<(true)} origin main",
		"git push {--force,$(echo a)$(echo b)} origin main",
		// A `}` inside the FIRST alternative. bash does not take the first
		// closing brace as the group's end — it keeps looking for a separator,
		// so `{msg},--no-verify}` expands to `msg}` and `--no-verify` (checked
		// against bash 5.3). Stopping at that inner brace read the whole word as
		// inert text and reopened the reported bypass four characters wider,
		// this time in a form that RUNS cleanly: `-m` swallows the junk
		// `msg}` as the message and `--no-verify` skips the hooks.
		"git commit -m {msg},--no-verify}",
		"git push {x},--force} origin main",
		"cd /tmp/x && rm {y},-rf} *",
		`git push {""},--force} origin main`,
		"git push {a}},--force} origin main",
		"git push {{}},--force} origin main",
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock {
				t.Errorf("verdict = %q, want %q: an unquoted brace group expands to argv the guard cannot check", d.Verdict, VerdictBlock)
			}
		})
	}
}

// TestBraceRefusalIsFailClosed pins the refusal POSTURE, not just the verdict.
// Returning ErrUnparsableCommand would have been the obvious route and is the
// wrong one: the `guard check` verb maps a tokenize error to a blocking exit,
// but the pre-tool-use hook maps it to fail-OPEN — so the bypass would have
// survived on the surface that matters. Only a real VerdictBlock is fail-closed
// on both front doors. The refusal is reached by a group past the expansion cap.
func TestBraceRefusalIsFailClosed(t *testing.T) {
	cmd := `git push origin main ` + strings.Repeat(`{a,b}`, 13)
	d, err := Defaults().Check(cmd)
	if err != nil {
		t.Fatalf("Check(%q) returned an error: %v — a tokenize error fails OPEN on the hook", cmd, err)
	}
	if d.Verdict != VerdictBlock {
		t.Fatalf("verdict = %q, want %q", d.Verdict, VerdictBlock)
	}
	if d.Tier != TierBlocker {
		t.Errorf("tier = %q, want %q", d.Tier, TierBlocker)
	}
	if d.EntryID != braceEntryID {
		t.Errorf("EntryID = %q, want the reserved brace id %q", d.EntryID, braceEntryID)
	}
	if _, isEntry := Defaults().Entries[braceEntryID]; isEntry {
		t.Errorf("the reserved id %q must never be a registry entry", braceEntryID)
	}
	if d.Reason == "" || d.Why == "" || d.Successor == "" || d.Message == "" {
		t.Errorf("a synthetic verdict must carry Reason/Why/Successor/Message for the report, got %+v", d)
	}
	if !contains(d.Matches, braceEntryID) {
		t.Errorf("Matches = %v, must list the reserved brace id", d.Matches)
	}
}

// TestBraceHandlingLeavesEveryOtherShapeAlone is the false-positive half. A
// brace sequence that bash does NOT expand — quoted, parameter expansion, a
// group with no alternative, a reserved-word group command — must keep exactly
// the verdict it had before braces were handled at all.
func TestBraceHandlingLeavesEveryOtherShapeAlone(t *testing.T) {
	cases := []struct {
		cmd  string
		want Verdict
	}{
		// Quoted: the bytes never reach the tokenizer's structural branches, so
		// bash hands the child the literal token and so does the guard.
		{`git push '{--force,}' origin main`, VerdictAllow},
		{`git push "{--force,}" origin main`, VerdictAllow},
		{`echo '{a,b}'`, VerdictAllow},
		// Parameter expansion, not a brace group.
		{`echo ${HOME}`, VerdictAllow},
		{`echo ${x:-a,b}`, VerdictAllow},
		{`git commit -m ${MSG}`, VerdictAllow},
		// A brace with no alternative is a literal in bash too.
		{`echo {a}`, VerdictAllow},
		{`echo {`, VerdictAllow},
		// A command substitution with no alternative beside it is still no
		// group: the substitution is skipped for STRUCTURE, not treated as one.
		{`echo {$(true)}`, VerdictAllow},
		{"echo {`true`}", VerdictAllow},
		// A closing brace with no separator anywhere after it is a literal in
		// bash too, however many of them there are — looking past the first one
		// must not turn these into groups.
		{`echo {a}b`, VerdictAllow},
		{`echo {a},x`, VerdictAllow},
		{`echo {a} b,c}`, VerdictAllow},
		{`echo {}`, VerdictAllow},
		{`awk {print}`, VerdictAllow},
		// Ordinary commands, unaffected.
		{`git status`, VerdictAllow},
		{`ls -la`, VerdictAllow},
		// A reserved-word group command still puts its inner command in command
		// position, so the blocker inside it still fires — the brace handling
		// must not turn that into a synthetic refusal instead.
		{`{ git push --force origin main; }`, VerdictBlock},
	}
	for _, c := range cases {
		t.Run(c.cmd, func(t *testing.T) {
			d := verdictOf(t, c.cmd)
			if d.Verdict != c.want {
				t.Errorf("verdict = %q, want %q (%+v)", d.Verdict, c.want, d)
			}
			if c.want != VerdictBlock && d.EntryID == braceEntryID {
				t.Errorf("a shape bash does not brace-expand was refused as one: %+v", d)
			}
			if c.cmd == `{ git push --force origin main; }` && d.EntryID == braceEntryID {
				t.Errorf("a group command was refused as a brace expansion: %+v", d)
			}
		})
	}
}

// TestBraceGroupIsRecordedOnTheSegment pins the tokenizer half directly: the
// refusal flag rides on the segment whose group passed the cap, so Check folds
// one signal however many segments the line has.
func TestBraceGroupIsRecordedOnTheSegment(t *testing.T) {
	segs, err := tokenize(`echo hi && git push origin ` + strings.Repeat(`{a,b}`, 13))
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if len(segs) != 2 {
		t.Fatalf("got %d segments, want 2: %v", len(segs), render(segs))
	}
	if segs[0].braceGroup {
		t.Errorf("the brace-free segment was marked: %v", segs[0].tokens)
	}
	if !segs[1].braceGroup {
		t.Errorf("the segment carrying the brace group was not marked: %v", segs[1].tokens)
	}
}

// TestBraceScanStaysLinear is the cost guard on the forward brace scan. The scan
// looks ahead from every structural `{`, so a word made of nothing but `{` bytes
// re-scans the same tail once per byte — quadratic, and measured at over six
// minutes on a 1 MiB command, which is inside the guard's own stdin cap. That is
// a hang on the PreToolUse path, reachable by any command the agent is asked to
// run. A shared budget bounds the total look-ahead per tokenize call, and
// exhausting it is fail-closed. The guard is a count of the bytes scanned, not a
// wall-clock ceiling (iss-2609240046582859).
func TestBraceScanStaysLinear(t *testing.T) {
	build := func(n int) string { return "echo " + strings.Repeat("{", n) }
	assertWorkGrowth(t, build, 1<<18, "the brace look-ahead's shared braceScanBudget")
	segs, err := tokenize(build(1 << 20))
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	// Exhausting the budget means the scan can no longer tell a group from a
	// literal, and a guard that cannot tell refuses the segment.
	if len(segs) == 0 || !segs[0].braceGroup {
		t.Errorf("a brace word past the scan budget must be refused, not waved through: %d segments", len(segs))
	}
}

// TestReservedBraceIDCannotBeClaimed pins the invariant the brace id's own
// comment states: a repo registry may not dress an ordinary entry up as the
// guard's own voice. Validate refuses the reserved ids, and the brace id is one.
func TestReservedBraceIDCannotBeClaimed(t *testing.T) {
	r := Registry{
		SchemaVersion: 1,
		Entries: map[string]Entry{
			braceEntryID: {
				ID: braceEntryID, Tier: TierWarn,
				Why: "not the guard's voice", Successor: "something",
				Pattern: Pattern{Command: "ls"},
			},
		},
	}
	if err := Validate(r); err == nil {
		t.Errorf("Validate accepted a registry entry claiming the reserved id %q", braceEntryID)
	}
}
