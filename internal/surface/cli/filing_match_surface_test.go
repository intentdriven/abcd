package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// filing_match_surface_test.go is the wiring proof for the filing-time match
// on the two unattended writers (itd-2609212137116617, iss-2609281911024185):
// `inbox promote` and `intent consistency ingest` run it through the layered
// configuration, and print what it found as the capture verb does.

const (
	fmHeld = "The capture ledger reader silently skips a record whose frontmatter carries " +
		"a duplicated key, so the finding disappears from every listing without a warning."
	fmDouble = "Capture ledger reader silently skips any record whose frontmatter carries a " +
		"duplicated key: the finding disappears from every listing, and no warning is printed."
)

// TestInboxPromoteMatchesTheLedger: a promoted report that doubles an open
// record is filed with the link written, and the verb says so.
func TestInboxPromoteMatchesTheLedger(t *testing.T) {
	repo, _ := gitRepoNoStore(t)
	t.Chdir(repo)
	held, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: fmHeld, Severity: capture.SeverityMinor, Category: "bug",
		Source: "user-observation", FoundDuring: "fixture", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatal(err)
	}
	skeleton := string(runCLI(t, "report", "--template"))
	filed := string(runCLIStdin(t, fillTemplate(t, skeleton, "Ledger reader skips duplicated keys", fmDouble), "report", "-"))
	i := strings.Index(filed, "rpt-")
	if i < 0 {
		t.Fatalf("report did not name an id:\n%s", filed)
	}
	id := filed[i : i+20]
	t.Cleanup(report.SetAbcdRootCommitForTest(gitutil.RootCommit(repo)))

	out := string(runCLI(t, "inbox", "promote", id))
	if !strings.Contains(out, "matched "+held.ID) || !strings.Contains(out, "link written") {
		t.Fatalf("promote does not report the match on %s:\n%s", held.ID, out)
	}
}

// TestIntentConsistencyIngestReportsTheMatch: every finding the pass files
// carries the filing-time match's outcome, in the text render and in --json.
func TestIntentConsistencyIngestReportsTheMatch(t *testing.T) {
	repo := consistencyCLIRepo(t)
	var em consistencyEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	fp := consistencyFindingsFile(t, repo, em)
	var res struct {
		Rows []struct {
			Match *struct {
				Threshold float64 `json:"threshold"`
			} `json:"match"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "ingest", "--findings-json", fp, "--json"), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 || res.Rows[0].Match == nil || res.Rows[0].Match.Threshold == 0 {
		t.Fatalf("ingest --json rows = %+v; want the match outcome on the filed row", res.Rows)
	}
}
