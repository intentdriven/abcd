package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/release"
)

// The detector iss-2610072347238476 owes. The findings gate refuses a working
// tree whose open/ issue records differ from HEAD, but the cut raises that
// reason as its own refusal only when no unfixed or deleted finding is present
// (the backstop in release.Emit). With either of those present, the refusal
// carries their half of the prose alone, and the uncommitted paths reached the
// operator only through --json: they fixed the named findings, ran again, and
// met a refusal they could have been shown the first time. The findings line
// is where the terminal render names them, so one run names every reason.

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

func TestFindingsLineNamesUncommittedRecordsBesideUnfixedAndDeletedFindings(t *testing.T) {
	const dirtyA = ".abcd/work/issues/open/iss-3-regraded.md"
	const dirtyB = ".abcd/work/issues/open/iss-4-captured.md"
	cases := []struct {
		name  string
		guard changelog.FindingGuard
	}{
		{"beside an unfixed finding", changelog.FindingGuard{
			BaseTag: "v0.7.0", Status: changelog.FindingGuardFailed,
			Unfixed:     []changelog.Finding{{ID: "iss-1", Path: ".abcd/work/issues/open/iss-1-a.md", Severity: "major"}},
			Uncommitted: []string{dirtyA, dirtyB},
		}},
		{"beside a deleted finding", changelog.FindingGuard{
			BaseTag: "v0.7.0", Status: changelog.FindingGuardFailed,
			Deleted:     []changelog.Finding{{ID: "iss-2", Path: ".abcd/work/issues/open/iss-2-b.md", Severity: "critical"}},
			Uncommitted: []string{dirtyA, dirtyB},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line := findingsLineOf(t, release.Cut{BaseTag: "v0.7.0", Findings: tc.guard})
			for _, want := range []string{"2 open record(s) differ from HEAD", dirtyA, dirtyB} {
				if !strings.Contains(line, want) {
					t.Errorf("findings line %q does not carry %q", line, want)
				}
			}
		})
	}
}

// TestFindingsLineSanitisesAnUncommittedPath holds brief invariant 13 for the
// new fragment: a path is a filename from the working tree, which reaches a
// terminal here like any other record-derived value.
func TestFindingsLineSanitisesAnUncommittedPath(t *testing.T) {
	line := findingsLineOf(t, release.Cut{BaseTag: "v0.7.0", Findings: changelog.FindingGuard{
		BaseTag: "v0.7.0", Status: changelog.FindingGuardFailed,
		Unfixed:     []changelog.Finding{{ID: "iss-1", Severity: "major"}},
		Uncommitted: []string{".abcd/work/issues/open/iss-5-\x1b[31mred.md"},
	}})
	if strings.Contains(line, "\x1b") {
		t.Errorf("an escape byte survived the findings line: %q", line)
	}
}
