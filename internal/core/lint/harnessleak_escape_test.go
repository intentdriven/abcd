package lint

import (
	"path/filepath"
	"testing"
)

// harness_leak reads each committed line in the spellings the scanner reads
// (iss-2609261658553101): in a JSON transcript or export quoted into the
// record, a session URL written straight after a \n escape defeats the
// pattern's leading word boundary, and a footer after one sits on a line of
// its own once the string is read. Both are the leak the plain spelling is.
func TestHarnessLeakReadsEscapedSpellings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "session.md"),
		"# Run\n\n"+`{"body":"done\n`+synthSessionURL(t, 41)+`"}`+"\n")
	writeFile(t, root, filepath.Join("docs", "footer.md"),
		"# Body\n\n"+`{"body":"Shipped.\n\n`+harnessFooter+`"}`+"\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"session.md", "footer.md"} {
		if !hasFinding(fs, filepath.Join("docs", rel), ruleHarnessLeak, 3) {
			t.Errorf("no harness_leak finding on docs/%s:3; got %+v", rel, fs)
		}
	}
}

// A footer quoted mid-sentence inside an escaped string is still prose about
// the ban, and an escape that hides nothing adds no finding.
func TestHarnessLeakEscapedSpellingsSpareProse(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("docs", "policy.md"),
		"# Policy\n\n"+`{"body":"We refuse the \"`+harnessFooter+`\" footer.\nThanks."}`+"\n")

	fs, err := Lint(harnessLeakCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleHarnessLeak); n != 0 {
		t.Fatalf("expected prose about the ban to be spared, got %d: %+v", n, fs)
	}
}
