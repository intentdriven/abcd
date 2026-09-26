package guard

import "testing"

// TestAliasInsideABangBodyIsResolved — iss-2609020348038749, half (2). A
// `!`-prefixed alias body is handed to a shell, and the command it holds may be
// a git command declaring an alias of its own. The body's segments went through
// the execute-a-string expansion but never back through the alias pre-pass, so
// the inner alias was never resolved and its rewrite never checked: the guard
// said nothing where its own reading one level up would have blocked.
func TestAliasInsideABangBodyIsResolved(t *testing.T) {
	force := "--force"
	cases := []struct {
		name  string
		line  string
		want  Verdict
		entry string
	}{
		{
			"an alias declared inside a bang body",
			"git -c alias.p='!git -c alias.q=push q " + force + " origin main' p",
			VerdictBlock, "git-push-force",
		},
		{
			"an alias body carrying the flag, declared inside a bang body",
			`git -c alias.p='!git -c alias.q="push ` + force + `" q origin main' p`,
			VerdictBlock, "git-push-force",
		},
		{
			"a bang body inside a bang body, resolved within the depth budget",
			`git -c alias.a='!git -c alias.b="!git -c alias.c=push c ` + force + ` origin main" b' a`,
			VerdictBlock, "git-push-force",
		},
		{
			"a benign alias inside a bang body stays allowed",
			"git -c alias.p='!git -c alias.s=status s' p",
			VerdictAllow, "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v", tc.line, err)
			}
			if d.Verdict != tc.want || d.EntryID != tc.entry {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q", tc.line, d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}
}

// TestBangAliasReentryIsBounded pins the depth budget and the repeat guard the
// re-entry carries. Past the budget the guard has lost the thread, so an alias
// rewrite it would have had to follow is a fail-closed block rather than a
// silent allow; a body repeated on one line is inspected once.
func TestBangAliasReentryIsBounded(t *testing.T) {
	// Three bang bodies deep, the innermost declaring the alias that carries
	// the hazard: one level past maxBangAliasDepth.
	inner := `git -c alias.d=push d --force origin main`
	l3 := `git -c alias.c=\"!` + inner + `\" c`
	l2 := `git -c alias.b="!` + l3 + `" b`
	line := `git -c alias.a='!` + l2 + `' a`
	d, err := Defaults().Check(line)
	if err != nil {
		t.Fatalf("Check(%q): %v", line, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != gitConfigEntryID {
		t.Fatalf("Check(%q) = %q via %q, want a block via %q: a rewrite past the depth budget is fail-closed", line, d.Verdict, d.EntryID, gitConfigEntryID)
	}

	segs, err := tokenize(`git -c alias.a='!git status' a && git -c alias.a='!git status' a`)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := Defaults().expandGitAliases(segs)
	bodies := 0
	for _, s := range out {
		if len(s.tokens) == 2 && s.tokens[0] == "git" && s.tokens[1] == "status" {
			bodies++
		}
	}
	if bodies != 1 {
		t.Errorf("a bang body repeated on one line was inspected %d times, want once", bodies)
	}
}
