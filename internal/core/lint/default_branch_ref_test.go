package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpstreamRefusalDraftNamesTheDefaultBranchNotOriginMain pins draft
// itd-2609091416304128 to the brief's definition of the default branch
// (iss-2610262037219105). 34-build defines it as origin's HEAD as last
// fetched, else the first of main, master, trunk and develop, and the capture
// package resolves it that way (detectDefaultBranch). The draft read the
// ledger at the literal refs/remotes/origin/main, so a repository whose
// default branch is master or trunk would have been judged as having no
// upstream ref. The draft's in-scope items, scope conditions and acceptance
// criteria must name the default branch's remote-tracking ref instead, and say
// how it is resolved. The prose cross-reference quoting the source issue ("on
// the fetched origin/main") and the SOTA's reflog example are quotation and
// illustration, not the draft's own definition, and are outside the check.
func TestUpstreamRefusalDraftNamesTheDefaultBranchNotOriginMain(t *testing.T) {
	rel := ".abcd/development/intents/drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the.md"
	raw, err := os.ReadFile(filepath.Join(repoRootFromPackage, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, h := range []string{"## What's In Scope", "## Scope Conditions", "## Acceptance Criteria"} {
		body := sectionBody(text, h)
		if body == "" {
			t.Fatalf("%s: section %q is gone; this test pins its wording", rel, h)
		}
		if strings.Contains(body, "origin/main") {
			t.Errorf("%s: %s names origin/main as the default branch; name the default branch's remote-tracking ref, as the capture package resolves it", rel, h)
		}
	}
	scope := sectionBody(text, "## What's In Scope")
	for _, want := range []string{"origin's `HEAD`", "`main`, `master`, `trunk` and `develop`"} {
		if !strings.Contains(scope, want) {
			t.Errorf("%s: the in-scope items must say how the default branch is resolved; %q is missing", rel, want)
		}
	}
}

// sectionBody returns the text under the level-two heading h, up to the next
// level-two heading, or "" when h is absent.
func sectionBody(text, h string) string {
	_, rest, ok := strings.Cut(text, "\n"+h+"\n")
	if !ok {
		return ""
	}
	if i := strings.Index(rest, "\n## "); i >= 0 {
		rest = rest[:i]
	}
	return rest
}
