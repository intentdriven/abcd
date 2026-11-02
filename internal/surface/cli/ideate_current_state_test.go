package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ideate"
)

// TestIdeateRenderStatesTheCurrentState proves the human view of a recorded
// verdict says which current-state mode the adversarial leg ran under
// (iss-2610310815034547), with a line of its own for each mode.
func TestIdeateRenderStatesTheCurrentState(t *testing.T) {
	scope := func(c ideate.CurrentState) string {
		out := renderIdeateResult(ideate.Result{Slug: "x", Verdict: ideate.VerdictSurvives, Graduates: true, CurrentState: c})
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "scope:") {
				return line
			}
		}
		t.Fatalf("no scope line for %s:\n%s", c, out)
		return ""
	}
	ignore, consider := scope(ideate.CurrentStateIgnore), scope(ideate.CurrentStateConsider)
	if !strings.Contains(ignore, "ignored") || !strings.Contains(ignore, "ambition") {
		t.Errorf("the ignore scope line must say the current state was ignored and the idea judged as an ambition: %q", ignore)
	}
	if !strings.Contains(consider, "considered") {
		t.Errorf("the consider scope line must say the current state was considered: %q", consider)
	}
}

// TestIdeatePageCarriesBothCurrentStateModes proves the command page offers
// the product thinker the choice, names both modes, makes ignore the default,
// words the rule for the product any managed repository builds, and shows the
// field in its payload example (iss-2610310815034547).
func TestIdeatePageCarriesBothCurrentStateModes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "ideate.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	head, _, _ := strings.Cut(page, "\n---\n\n")
	if !strings.Contains(head, "--current-state") {
		t.Errorf("the page's argument hint does not offer --current-state:\n%s", head)
	}
	for _, want := range []string{"--current-state ignore", "--current-state consider", `"current_state": "ignore"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not carry %q", want)
		}
	}
	start := strings.Index(page, "### Leg 3")
	end := strings.Index(page, "## Record the verdict")
	if start < 0 || end < start {
		t.Fatal("the page has no leg 3 section before its record section")
	}
	leg3 := page[start:end]
	for _, want := range []string{"ignore", "consider", "default", "the product the repository builds"} {
		if !strings.Contains(leg3, want) {
			t.Errorf("leg 3 does not carry %q:\n%s", want, leg3)
		}
	}
	if !strings.Contains(leg3, "`ignore` is the default") || !strings.Contains(leg3, "**`ignore`** (the default)") {
		t.Errorf("leg 3 does not make ignore the default:\n%s", leg3)
	}
	if strings.Contains(leg3, "**`consider`** (the default)") {
		t.Errorf("leg 3 marks consider as the default:\n%s", leg3)
	}
	// The gaps a reviewer lists under ignore have a home in the payload, so
	// the page has to name it: a list with nowhere to go is lost or filed as
	// the kill attempt the same rule forbids.
	if !strings.Contains(leg3, "`implies_building`") {
		t.Errorf("leg 3 does not say where the list of what the idea implies building goes:\n%s", leg3)
	}
	// The rule holds for any managed repository's product, so the section
	// never names abcd's own product as the thing being judged.
	for _, own := range []string{"abcd's brief", "abcd's product", "abcd ships", "abcd's intents"} {
		if strings.Contains(leg3, own) {
			t.Errorf("leg 3 words the rule for abcd's own product (%q)", own)
		}
	}
}
