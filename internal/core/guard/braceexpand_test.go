package guard

import (
	"strings"
	"testing"
)

// TestBraceExpansionMatchesBash pins the expander against bash 5.3's own
// answers (`printf '[%s]' WORD`, recorded when the expander landed): the argv
// the guard checks is the argv the shell builds, including the shapes where
// bash does something a reader would not guess — a `}` inside the first
// alternative, a nested group that keeps its outer braces, a sequence that is
// not one, zero padding that counts the sign.
func TestBraceExpansionMatchesBash(t *testing.T) {
	cases := []struct {
		word string
		want []string
	}{
		{`{a,b}`, []string{"a", "b"}},
		{`{--force,}`, []string{"--force"}},
		{`{,--force}`, []string{"--force"}},
		{`{{--force,--dry-run}}`, []string{"{--force}", "{--dry-run}"}},
		{`\${--force,}`, []string{"$--force", "$"}},
		{`{msg},--no-verify}`, []string{"msg}", "--no-verify"}},
		{`{""},--force}`, []string{"}", "--force"}},
		{`{a}},--force}`, []string{"a}}", "--force"}},
		{`{{}},--force}`, []string{"{}}", "--force"}},
		{`{},a}`, []string{"{},a}"}},
		{`{a..e}`, []string{"a", "b", "c", "d", "e"}},
		{`{1..10..3}`, []string{"1", "4", "7", "10"}},
		{`{01..3}`, []string{"01", "02", "03"}},
		{`{-02..2}`, []string{"-02", "-01", "000", "001", "002"}},
		{`{a..1}`, []string{"{a..1}"}},
		{`{a..b..c}`, []string{"{a..b..c}"}},
		{`{1..3,x}`, []string{"1..3", "x"}},
		{`{'1'..3}`, []string{"{1..3}"}},
		{`x{,.bak}`, []string{"x", "x.bak"}},
		{`foo/{a,b}/{c,d}`, []string{"foo/a/c", "foo/a/d", "foo/b/c", "foo/b/d"}},
		{`{a,b}{c,d}{e,f}`, []string{"ace", "acf", "ade", "adf", "bce", "bcf", "bde", "bdf"}},
		{`{5..1}`, []string{"5", "4", "3", "2", "1"}},
		{`{1..5..-2}`, []string{"1", "3", "5"}},
		{`{a..e..2}`, []string{"a", "c", "e"}},
		{`${x:-a,b}`, []string{"${x:-a,b}"}},
		{`"$"{a,b}`, []string{"$a", "$b"}},
		{`{a,b}\}`, []string{"a}", "b}"}},
		{`{a,\,b}`, []string{"a", ",b"}},
		{`{a..}`, []string{"{a..}"}},
		{`{..a}`, []string{"{..a}"}},
		{`{a,}{b,}`, []string{"ab", "a", "b"}},
		{`a{b}c`, []string{"a{b}c"}},
		{`{a,{b,c}}d`, []string{"ad", "bd", "cd"}},
		{`'{a,b}'`, []string{"{a,b}"}},
		{`{a,b}$(true)c`, []string{"ac", "bc"}},
	}
	for _, tc := range cases {
		t.Run(tc.word, func(t *testing.T) {
			segs, err := tokenize("echo " + tc.word)
			if err != nil {
				t.Fatalf("tokenize: %v", err)
			}
			var got []string
			for _, s := range segs {
				if len(s.tokens) > 0 && s.tokens[0] == "echo" {
					// A substitution's output is marked where it goes
					// (unknown.go); `$(true)` prints nothing, as bash ran it.
					for _, tok := range s.tokens[1:] {
						got = append(got, knownText(tok))
					}
					if s.braceGroup {
						t.Errorf("the word was refused rather than expanded: %+v", s)
					}
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Errorf("echo %s expanded to %q, bash gives %q", tc.word, got, tc.want)
			}
		})
	}
}

// TestEverydayBraceCommandsAllow is the detector the record names: brace
// shorthand nobody meant as a hazard is expanded and allowed, where the guard
// used to refuse every unquoted group it met.
func TestEverydayBraceCommandsAllow(t *testing.T) {
	for _, cmd := range []string{
		`mkdir -p foo/{a,b}`,
		`cp x{,.bak}`,
		`rm -rf dir{1..9}`,
		`touch file{01..10}.txt`,
		`mv src/{old,new}.go`,
		`ls {cmd,internal}/*.go`,
		`git add internal/{core,surface}/guard`,
		`echo {a..e}`,
		`mkdir -p .abcd/.work.local/{logs,scratch}`,
		`x={a,b} git status`,
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("verdict = %q (%s), want allow: an everyday brace group is expanded, not refused", d.Verdict, d.EntryID)
			}
		})
	}
}

// TestBraceExpandedHazardsBlockByTheirEntry is the other half: a group that
// expands to a hazard now blocks under the ENTRY that names the hazard, because
// the guard reads the argv bash builds rather than refusing the word it could
// not read.
func TestBraceExpandedHazardsBlockByTheirEntry(t *testing.T) {
	cases := []struct {
		cmd, entry string
	}{
		{`git push {--force,} origin main`, "git-push-force"},
		{`git push {,--force} origin main`, "git-push-force"},
		{`git push {--force,--dry-run} origin main`, "git-push-force"},
		{`cd /tmp/x && rm {y},-rf} *`, "rm-rf-after-cd-chain"},
		{`cd scratch && rm -r{f,} *`, "rm-rf-after-cd-chain"},
		{"git push {--force,$(true)} origin main", "git-push-force"},
		{`sh -c 'git push {--force,} origin main'`, "git-push-force"},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d := verdictOf(t, tc.cmd)
			if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
				t.Errorf("verdict = %q under %q, want block under %q", d.Verdict, d.EntryID, tc.entry)
			}
		})
	}
}

// TestBraceExpansionCapRefuses pins the one place the expander refuses: a group
// that multiplies past the cap is left unexpanded and the command is blocked
// under the reserved brace id, fail-closed on both front doors.
func TestBraceExpansionCapRefuses(t *testing.T) {
	for _, cmd := range []string{
		`echo {1..100000}`,
		`echo ` + strings.Repeat(`{a,b}`, 13),
		`git push origin main ` + strings.Repeat(`{a,b}`, 13),
	} {
		t.Run(cmd[:16], func(t *testing.T) {
			d, err := Defaults().Check(cmd)
			if err != nil {
				t.Fatalf("Check returned an error, which the hook fails open on: %v", err)
			}
			if d.Verdict != VerdictBlock || d.EntryID != braceEntryID {
				t.Errorf("verdict = %q under %q, want block under %q", d.Verdict, d.EntryID, braceEntryID)
			}
		})
	}
}

// TestBraceExpansionWorkIsBounded asserts the expander's cost as a count of
// work, not a wall-clock ceiling: whatever the input, the scan steps, words and
// bytes one tokenize call spends stay inside the declared caps.
func TestBraceExpansionWorkIsBounded(t *testing.T) {
	for _, word := range []string{
		strings.Repeat("{a,", 2000) + strings.Repeat("}", 2000),
		strings.Repeat("{", 5000) + "a,b" + strings.Repeat("}", 5000),
		strings.Repeat("{}", 20000),
		"{" + strings.Repeat("a,", 50000) + "}",
	} {
		lim := newBraceLimits()
		w := bword{b: []byte(word), m: make([]byte, len(word))}
		for i := range w.m {
			w.m[i] = wordStruct
		}
		_, _ = expandBraces(w, &lim)
		if lim.work < -1 || lim.bytes < -len(word)-1 {
			t.Errorf("expansion overspent its budget: work %d, bytes %d", lim.work, lim.bytes)
		}
	}
}
