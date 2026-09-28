package recordid

import "testing"

// TestFilenameGrammarIsCompiledOncePerFamily: the resolver derives an id for
// every record file in the checkout, and peers does the same for every worktree,
// so the filename grammar is read once per file. It is a pure function of a
// fixed family prefix, so reading it must not compile it: a regexp compiled per
// call made every scan pay one compilation per record (iss-2609261943168303).
//
// Held as counts, not a stopwatch. FilenameNumRe hands back the one compiled
// pattern its family has, and deriving an id costs the match's allocations
// alone; a compilation per call costs dozens.
func TestFilenameGrammarIsCompiledOncePerFamily(t *testing.T) {
	for _, prefix := range []string{"iss", "itd", "spc"} {
		if FilenameNumRe(prefix) != FilenameNumRe(prefix) {
			t.Errorf("FilenameNumRe(%q) compiles a new pattern on every call", prefix)
		}
	}
	name := "iss-2609261943168303-recordid-grammar.md"
	if got := fileID("iss", name); got != "iss-2609261943168303" {
		t.Fatalf("fileID(%q) = %q, want iss-2609261943168303", name, got)
	}
	const bar = 8
	if allocs := testing.AllocsPerRun(100, func() { fileID("iss", name) }); allocs > bar {
		t.Errorf("deriving one record id allocates %.0f times, want at most %d: the grammar is being compiled per file", allocs, bar)
	}
}
