package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// lintIssueImpact runs only the issue_impact_valid record-lint blocker over the
// ledger under repoRoot and returns its findings. It is the real gate an issue
// the tool resolves must satisfy, so a resolve path that cannot stamp a valid
// impact is caught here rather than at a later hand-lint.
func lintIssueImpact(t *testing.T, repoRoot string) []lint.Finding {
	t.Helper()
	cfg := lint.Config{
		Rules: map[string]lint.RuleConfig{
			"issue_impact_valid": {Enabled: true, Severity: "blocker", IssuesDir: LedgerRelPath},
		},
	}
	fs, err := lint.Lint(cfg, repoRoot)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	return fs
}

// TestResolveProducesImpactValidRecord is the iss-117 reproduction: a resolved
// issue lands in resolved/, where issue_impact_valid REQUIRES a valid, non-null
// impact. The tool's own resolve path must therefore be able to stamp one, or the
// record it produces is rejected by the very blocker the tool ships.
func TestResolveProducesImpactValidRecord(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed the thing", Impact: "fix",
	}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if fs := lintIssueImpact(t, repo); len(fs) != 0 {
		t.Fatalf("resolved record must satisfy issue_impact_valid, got %d finding(s): %+v", len(fs), fs)
	}
}

// TestResolveRefusesInvalidImpact proves the resolve boundary fails closed on a
// missing or misspelled impact rather than minting a record the blocker rejects:
// there is no default, so an absent judgement is refused, not invented.
func TestResolveRefusesInvalidImpact(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, bad := range []string{"", "additiv", "Additive", `"fix"`} {
		if _, err := Resolve(ResolveRequest{
			Grounds:  testGrounds,
			RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "note", Impact: bad,
		}); err == nil {
			t.Fatalf("Resolve with impact %q must be refused", bad)
		}
	}
	// The refused transitions must not have moved the issue out of open/.
	if _, status, ferr := findIssue(ir, res.ID); ferr != nil || status != StateOpen {
		t.Fatalf("a refused resolve must leave the issue in open/: status=%s err=%v", status, ferr)
	}
}

// TestResolveImpactUnquoted guards the wire format: impact is a machine-read enum
// the frontmatter line-scanner compares byte-for-byte, so it must be written bare
// (impact: fix), never YAML-quoted (impact: "fix") the way a prose note is.
func TestResolveImpactUnquoted(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
	})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "done", Impact: "internal",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo, tr.Path))
	if err != nil {
		t.Fatal(err)
	}
	data := string(raw)
	if !strings.Contains(data, "\nimpact: internal\n") {
		t.Fatalf("impact must be written bare (impact: internal), got:\n%s", data)
	}
	if strings.Contains(data, `impact: "internal"`) {
		t.Fatalf("impact must not be YAML-quoted:\n%s", data)
	}
}
