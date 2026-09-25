package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// Resolving or declining an issue moves it out of open/, and every link that
// named it there follows it — from an ADR, from a sibling issue still in open/,
// and the moved issue's own bare link to that sibling (iss-2609250846525896).
func TestTransitionRepointsLinksToTheMovedIssue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		folder string
		move   func(repo, ir, id string) (TransitionResult, error)
	}{
		{"resolve", "resolved", func(repo, ir, id string) (TransitionResult, error) {
			return Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "fixed", Impact: "fix"})
		}},
		{"wontfix", "wontfix", func(repo, ir, id string) (TransitionResult, error) {
			return Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: id, Reason: "declined because the cost exceeds the benefit here"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			mk := func(slug string) string {
				res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
					Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: slug})
				if err != nil {
					t.Fatal(err)
				}
				return filepath.Base(res.Path)
			}
			a, b := mk("alpha"), mk("beta")
			appendTo := func(rel, line string) {
				f, err := os.OpenFile(filepath.Join(repo, rel), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if _, err := f.WriteString(line); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(filepath.Join(repo, ".abcd/development/decisions/adrs"), 0o755); err != nil {
				t.Fatal(err)
			}
			adr := ".abcd/development/decisions/adrs/0001-x.md"
			appendTo(adr, "# x\n\nFound in [alpha](../../../work/issues/open/"+a+").\n")
			appendTo(LedgerRelPath+"/open/"+a, "\nSee [beta]("+b+").\n")
			appendTo(LedgerRelPath+"/open/"+b, "\nSee [alpha]("+a+").\n")

			id := strings.SplitN(a, "-", 3)[0] + "-" + strings.SplitN(a, "-", 3)[1]
			res, err := tc.move(repo, ir, id)
			if err != nil {
				t.Fatal(err)
			}

			cfg := lint.Config{
				Roots: []string{".abcd/development", ".abcd/work"},
				Rules: map[string]lint.RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}},
			}
			findings, err := lint.Lint(cfg, repo)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range findings {
				if f.RuleID == "links_resolve" {
					t.Errorf("links_resolve after %s: %s:%d %s", tc.name, f.File, f.Line, f.Message)
				}
			}
			if len(res.Relinked) != 3 || res.RelinkError != "" {
				t.Errorf("the transition must report its three rewrites: %+v %q", res.Relinked, res.RelinkError)
			}
			body, err := os.ReadFile(filepath.Join(repo, adr))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "(../../../work/issues/"+tc.folder+"/"+a+")") {
				t.Errorf("ADR link not repointed:\n%s", body)
			}
		})
	}
}
