package loop

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestItd50NamesTheFixLoopsVerdicts holds itd-50 to the build loop it extends
// (iss-2610262037443119): its scope section describes the loop as build's last
// stage with fix rounds bounded by pace.fix_rounds, as the 2026-09-21 re-scope
// ruled, and names the implementation-grain fix loop's verdicts in the words
// the ruthless reviewer states, never a NEEDS_WORK no validator returns.
func TestItd50NamesTheFixLoopsVerdicts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", ".abcd", "development", "intents", "planned", "itd-50-loop-toward-acceptance.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read itd-50: %v", err)
	}
	doc := string(raw)
	if m := regexp.MustCompile(`(?i)NEEDS[ _-]WORK`).FindString(doc); m != "" {
		t.Errorf("%s names a %q verdict; the fix loop's validators return SHIP or FIX FIRST, APPROVE, BLOCK or NEEDS-INPUT", path, m)
	}
	start := strings.Index(doc, "\n## What's In Scope\n")
	if start < 0 {
		t.Fatalf("%s has no \"## What's In Scope\" section", path)
	}
	scope := doc[start+1:]
	if end := strings.Index(scope[len("## What's In Scope"):], "\n## "); end >= 0 {
		scope = scope[:len("## What's In Scope")+end]
	}
	for _, w := range reviewerVerdicts[RoleRuthless] {
		if !strings.Contains(scope, "`"+w.word+"`") {
			t.Errorf("%s: the scope section does not name the fix loop's verdict `%s`", path, w.word)
		}
	}
	for _, want := range []string{"last stage", "`pace.fix_rounds`"} {
		if !strings.Contains(scope, want) {
			t.Errorf("%s: the scope section does not say %q, as the 2026-09-21 re-scope rules", path, want)
		}
	}
	for _, stale := range []string{"facilitator-elected per intent", "`record-only`** (the current behaviour, and the default)", "max-iteration budget"} {
		if strings.Contains(scope, stale) {
			t.Errorf("%s: the scope section still carries the pre-re-scope %q", path, stale)
		}
	}
}
