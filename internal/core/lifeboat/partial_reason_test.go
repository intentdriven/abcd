package lifeboat

import (
	"os"
	"strings"
	"testing"
)

// partialReasonFixtures are the repositories the reason contract is asserted
// over: this repository (every tier, a real record) and each tier's fixture,
// between them reaching most of the partial branches the adapters hold.
func partialReasonFixtures(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"repo":          repoRoot(t),
		"native":        nativeTierFixture(t),
		"conventions":   convTierFixture(t),
		"markers":       convMarkerFixture(t),
		"git":           gitTierFixture(t),
		"git-revert":    gitFixtureWithRevert(t),
		"git-deletions": gitFixtureDeletionOnly(t),
	}
}

// TestProbeEveryPartialCarriesAReason is the reason contract at the report: a
// section rated partial says what was found and what would ground it, so a
// reader can act on it rather than guess (iss-2610040758387938). A blank
// already carries its question; a partial carried nothing.
func TestProbeEveryPartialCarriesAReason(t *testing.T) {
	sawPartial := false
	for name, dir := range partialReasonFixtures(t) {
		cov, err := Probe(dir)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, s := range cov.Sections {
			if s.Status != StatusPartial {
				if s.Reason != "" {
					t.Errorf("%s: %s section %s carries a reason %q; only a partial does", name, s.Status, s.Name, s.Reason)
				}
				continue
			}
			sawPartial = true
			if strings.TrimSpace(s.Reason) == "" {
				t.Errorf("%s: partial section %s carries no reason", name, s.Name)
			}
		}
		if out := cov.Render(); cov.Summary.Partial > 0 && !strings.Contains(out, "why partial: ") {
			t.Errorf("%s: rendered report states no reason for its %d partial section(s)", name, cov.Summary.Partial)
		}
	}
	if !sawPartial {
		t.Fatal("no fixture produced a partial section; the contract was not exercised")
	}
}

// TestEverySourcePartialCarriesAReason asserts the same contract below the
// report, adapter by adapter, so a partial the best-of reduction would hide
// behind a richer tier's grounded result is still held to it.
func TestEverySourcePartialCarriesAReason(t *testing.T) {
	for name, dir := range partialReasonFixtures(t) {
		ctx, err := newSourceContext(dir)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, s := range allSources() {
			ev := s.Probe(ctx)
			if ev.Status == StatusPartial && strings.TrimSpace(ev.Reason) == "" {
				t.Errorf("%s: %s (%s) returned a partial with no reason", name, s.Section(), s.Tier())
			}
		}
		ctx.Close()
	}
}

// TestPartialIsBuiltOnlyThroughTheConstructor is the source guard that makes
// the contract hold for branches no fixture reaches: every partial in this
// package is built by partial(), which requires a reason, never by an Evidence
// literal that can leave it out.
func TestPartialIsBuiltOnlyThroughTheConstructor(t *testing.T) {
	// Built at run time so this file is not itself a match.
	needle := "Status:" + " "
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		data, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			idx := strings.Index(line, needle)
			if idx < 0 {
				continue
			}
			if i > 0 && strings.HasPrefix(lines[i-1], "func partial(") {
				continue // the constructor itself
			}
			if strings.HasPrefix(strings.TrimSpace(line[idx+len(needle):]), "StatusPartial") {
				t.Errorf("%s:%d builds a partial Evidence literal — use partial(confidence, sources, reason)", n, i+1)
			}
		}
	}
}

// TestPackSectionFileStatesWhyPartial carries the reason contract into the
// pack: each partial section's brief file in the lifeboat says why it is
// partial, as coverage.json and the rendered report do, and a grounded one
// says nothing of the kind (iss-2610072347247487).
func TestPackSectionFileStatesWhyPartial(t *testing.T) {
	sawPartial := false
	for name, dir := range partialReasonFixtures(t) {
		cov, err := Probe(dir)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		lb, err := Plan(dir)
		if err != nil {
			t.Fatalf("%s: plan: %v", name, err)
		}
		for _, s := range cov.Sections {
			leaf, ok := briefLeaf(s.Name)
			if !ok || s.Status == StatusBlank {
				continue
			}
			md := string(planFile(t, lb, "brief/"+leaf).Content)
			if s.Status != StatusPartial {
				if strings.Contains(md, "Why partial:") {
					t.Errorf("%s: grounded %s's brief file states a partial reason:\n%s", name, s.Name, md)
				}
				continue
			}
			sawPartial = true
			if want := "Why partial: " + mdInline(s.Reason); !strings.Contains(md, want) {
				t.Errorf("%s: partial %s's brief file does not state its reason %q:\n%s", name, s.Name, s.Reason, md)
			}
		}
	}
	if !sawPartial {
		t.Fatal("no fixture packed a partial section; the contract was not exercised")
	}
}
