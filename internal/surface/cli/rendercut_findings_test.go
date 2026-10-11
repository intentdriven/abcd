package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/release"
)

// The render half of iss-2610072347238476 and iss-2610090642371836. The
// findings gate refuses a working tree whose open/ issue records differ from
// HEAD. The cut once raised that reason as a refusal of its own only when no
// unfixed or deleted finding was present, so the findings line named the paths
// beside either; release.Emit now raises it as its own entry whatever else is
// present, so the refusal names the paths and the findings line counts them,
// as it counts the other two halves. Every reason is named once.

// findingsLineOf returns the `findings:` line renderCut writes for a cut.
func findingsLineOf(t *testing.T, cut release.Cut) string {
	t.Helper()
	var buf bytes.Buffer
	renderCut(&buf, "abcd launch ship", cut)
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "  findings:") {
			return line
		}
	}
	t.Fatalf("renderCut wrote no findings line:\n%s", buf.String())
	return ""
}

// TestLaunchShipNamesAnUncommittedRecordOnceBesideAnUnfixedFinding runs the
// verb over a cut refused for an unfixed finding and an uncommitted open record
// together: both reasons reach the render as refusals, the uncommitted path is
// printed exactly once, and the findings line counts it without naming it.
func TestLaunchShipNamesAnUncommittedRecordOnceBesideAnUnfixedFinding(t *testing.T) {
	const dirty = ".abcd/work/issues/open/iss-92-captured.md"
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Write(".abcd/work/issues/open/iss-90-found.md", "---\nid: \"iss-90\"\nseverity: \"major\"\n---\n\nfound.\n")
	r.Commit("ship an intent and capture a finding")
	r.Write(dirty, "---\nid: \"iss-92\"\nseverity: \"critical\"\n---\n\ncaptured, never committed.\n")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	render := string(out)
	if n := strings.Count(render, dirty); n != 1 {
		t.Errorf("the render names %s %d time(s), want once:\n%s", dirty, n, render)
	}
	if n := strings.Count(render, "refused (unfixed-finding)"); n != 2 {
		t.Errorf("the render carries %d unfixed-finding refusal(s), want two (the finding, the uncommitted record):\n%s", n, render)
	}
	var line string
	for _, l := range strings.Split(render, "\n") {
		if strings.HasPrefix(l, "  findings:") {
			line = l
		}
	}
	if !strings.Contains(line, "1 unfixed finding(s)") || !strings.Contains(line, "1 open record(s) differ from HEAD") {
		t.Errorf("findings line %q does not count both halves", line)
	}
	if strings.Contains(line, dirty) {
		t.Errorf("findings line %q names the path the refusal already names", line)
	}
}

// TestFindingsLineCountsUncommittedRecordsAlone: with no other half present the
// line still counts them, so the line reads the same whatever sits beside them.
func TestFindingsLineCountsUncommittedRecordsAlone(t *testing.T) {
	line := findingsLineOf(t, release.Cut{BaseTag: "v0.7.0", Findings: changelog.FindingGuard{
		BaseTag: "v0.7.0", Status: changelog.FindingGuardFailed,
		Uncommitted: []string{".abcd/work/issues/open/iss-3-a.md", ".abcd/work/issues/open/iss-4-b.md"},
	}})
	if !strings.Contains(line, "2 open record(s) differ from HEAD") {
		t.Errorf("findings line %q does not count the uncommitted records", line)
	}
}

// TestRenderCutSanitisesAnUncommittedPath holds brief invariant 13 where the
// path is now printed: a filename from the working tree reaches the terminal
// through the refusal's reason like any other record-derived value.
func TestRenderCutSanitisesAnUncommittedPath(t *testing.T) {
	g := changelog.FindingGuard{
		BaseTag: "v0.7.0", Status: changelog.FindingGuardFailed,
		Uncommitted: []string{".abcd/work/issues/open/iss-5-\x1b[31mred.md"},
	}
	var buf bytes.Buffer
	renderCut(&buf, "abcd launch ship", release.Cut{BaseTag: "v0.7.0", Findings: g,
		Refusals: []release.Refusal{{Kind: release.RefusalUnfixedFinding, Reason: g.UncommittedReason()}}})
	if strings.Contains(buf.String(), "\x1b") {
		t.Errorf("an escape byte survived the render: %q", buf.String())
	}
}
