package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// driftFixture lays a migrated tree carrying every kind of drift the check
// reports, plus joins that are healthy and must stay silent.
func driftFixture(t *testing.T) (repo, ir string) {
	t.Helper()
	repo, ir = ledger(t)
	const led = ".abcd/work/issues/"
	const itd = ".abcd/development/intents/"
	for rel, body := range map[string]string{
		// healthy: both halves, intent shipped, issue resolved.
		led + "resolved/iss-1-one.md": issueRecordText("iss-1", "one", "related_intents: [itd-1]\nresolution: \"done\"\n"),
		itd + "shipped/itd-1-one.md":  "---\nid: itd-1\nslug: one\nspec_id: null\nkind: standalone\nrelated_issues: [iss-1]\n---\n\n# One\n",
		// healthy: a loose relation, the issue names an intent that does not name it back.
		led + "open/iss-2-two.md": issueRecordText("iss-2", "two", "related_intents: [itd-1]\n"),
		// one-sided: the intent names iss-3, iss-3 does not name the intent.
		led + "open/iss-3-three.md":   issueRecordText("iss-3", "three", ""),
		itd + "drafts/itd-3-three.md": "---\nid: itd-3\nslug: three\nspec_id: null\nkind: null\nrelated_issues: [iss-3]\n---\n\n# Three\n",
		// dangling: the intent names an issue no ledger holds.
		itd + "drafts/itd-4-four.md": "---\nid: itd-4\nslug: four\nspec_id: null\nkind: null\nrelated_issues: [iss-99]\n---\n\n# Four\n",
		// shipped, but the issue it was promoted from is still open.
		led + "open/iss-5-five.md":    issueRecordText("iss-5", "five", "related_intents: [itd-5]\n"),
		itd + "shipped/itd-5-five.md": "---\nid: itd-5\nslug: five\nspec_id: null\nkind: standalone\nrelated_issues: [iss-5]\n---\n\n# Five\n",
		// retired: an unmigrated back-link.
		led + "open/iss-6-six.md": issueRecordText("iss-6", "six", "promoted_to: itd-3\n"),
		// dangling from the ledger end: an issue naming an intent no bucket holds.
		led + "open/iss-7-seven.md": issueRecordText("iss-7", "seven", "related_intents: [itd-77]\n"),
		// one-sided from a reading item, which carries no loose relation.
		led + "readings/rdg-9/rdi-8.md": "---\nschema_version: 1\nid: rdi-8\nrun: rdg-9\npattern: p\nrelated_intents: [itd-3]\n---\n\nbody\n",
	} {
		writeTree(t, repo, rel, body)
	}
	return repo, ir
}

// TestIssueDriftReportsEveryBrokenJoinAndNothingElse — itd-4 AC3's drift
// detection (the spc-23 shape): a corpus-wide bidirectional walk between the
// intent store and the ledger. Each broken join is one finding naming both
// records; a healthy pair and a loose issue→intent relation say nothing.
func TestIssueDriftReportsEveryBrokenJoinAndNothingElse(t *testing.T) {
	repo, ir := driftFixture(t)
	res, err := IssueDrift(IssueDriftRequest{RepoRoot: repo, IssuesRoot: ir, Now: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("IssueDrift: %v", err)
	}
	got := map[string]bool{}
	for _, f := range res.Findings {
		got[f.Kind+" "+f.Record+" "+f.Other] = true
	}
	want := []string{
		DriftOneSided + " itd-3 iss-3",
		DriftDangling + " itd-4 iss-99",
		DriftShippedUnresolved + " itd-5 iss-5",
		DriftRetiredField + " iss-6 itd-3",
		DriftDangling + " iss-7 itd-77",
		DriftOneSided + " rdi-8 itd-3",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing finding %q; got %+v", w, res.Findings)
		}
	}
	if len(res.Findings) != len(want) {
		t.Errorf("findings = %d, want %d: %+v", len(res.Findings), len(want), res.Findings)
	}
	for _, f := range res.Findings {
		if f.Message == "" || f.Path == "" {
			t.Errorf("a finding must carry a message and the record's path: %+v", f)
		}
	}

	// The receipt lands in the local tier and holds the same findings.
	if !strings.HasPrefix(filepath.ToSlash(res.ReceiptPath), ".abcd/.work.local/logs/audit/issue-drift-") {
		t.Fatalf("receipt path = %q, want it under .abcd/.work.local/logs/audit/issue-drift-<ts>/", res.ReceiptPath)
	}
	data, err := os.ReadFile(filepath.Join(repo, res.ReceiptPath))
	if err != nil {
		t.Fatalf("receipt unreadable: %v", err)
	}
	var back IssueDriftResult
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("receipt is not the result JSON: %v", err)
	}
	if len(back.Findings) != len(res.Findings) {
		t.Fatalf("receipt holds %d findings, the result %d", len(back.Findings), len(res.Findings))
	}
}

// TestIssueDriftIsSilentOnAHealthyTree — a tree whose joins all read from both
// ends reports nothing (and still leaves its receipt).
func TestIssueDriftIsSilentOnAHealthyTree(t *testing.T) {
	repo, ir := ledger(t)
	writeTree(t, repo, ".abcd/work/issues/open/iss-1-one.md", issueRecordText("iss-1", "one", "related_intents: [itd-1]\n"))
	writeTree(t, repo, ".abcd/development/intents/planned/itd-1-one.md",
		"---\nid: itd-1\nslug: one\nspec_id: null\nkind: standalone\nrelated_issues: [iss-1]\n---\n\n# One\n")
	res, err := IssueDrift(IssueDriftRequest{RepoRoot: repo, IssuesRoot: ir, Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("a healthy tree reported drift: %+v", res.Findings)
	}
	if res.Findings == nil {
		t.Fatal("findings must be an empty list, not null, so the JSON says so")
	}
}
