package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// AuditReferences is the release cut's reading of the record's prose
// (iss-2611031731385043). These are its edges: what the cut refuses on, what it
// reports for removal, and what it leaves to the rulings the record already
// made.

// proseConfigJSON arms prose_citation_resolves over proseCfg's stores, as the
// repository's own .abcd/record-lint.json does.
const proseConfigJSON = `{"rules": {"prose_citation_resolves": {"enabled": true, "severity": "blocker", "record_stores": {
  "adr": ".abcd/development/decisions/adrs", "itd": ".abcd/development/intents",
  "spc": ".abcd/development/specs", "iss": ".abcd/work/issues"}}}}`

// auditFixture is a corpus with the configuration armed and one issue whose
// body is body; it returns the root and the issue's repo-relative path.
func auditFixture(t *testing.T, body string) (string, string) {
	t.Helper()
	root := t.TempDir()
	proseCorpus(t, root)
	writeFile(t, root, DefaultRecordLintConfigPath, proseConfigJSON)
	name := "iss-2611031720012399-a-citing-record.md"
	writeProseIssue(t, root, "open", name, body)
	return root, filepath.ToSlash(filepath.Join(".abcd", "work", "issues", "open", name))
}

func audit(t *testing.T, root string) ReferenceAudit {
	t.Helper()
	a, err := AuditReferences(root)
	if err != nil {
		t.Fatalf("AuditReferences: %v", err)
	}
	return a
}

// TestAuditReferencesNamesEveryCitationOfARecordNotInTheTree: marked or not,
// an id the tree being cut does not resolve is unlanded, with its line.
func TestAuditReferencesNamesEveryCitationOfARecordNotInTheTree(t *testing.T) {
	root, rel := auditFixture(t, strings.Join([]string{
		"Per dcn-2611031615094092, unmarked.",
		"",
		"Per iss-2611039999999999 and the live iss-42. <!-- record-lint: forward-looking -->",
	}, "\n"))
	a := audit(t, root)
	want := []UnlandedReference{
		{File: rel, Line: 5, ID: "dcn-2611031615094092", Marked: false},
		{File: rel, Line: 7, ID: "iss-2611039999999999", Marked: true},
	}
	if len(a.Unlanded) != len(want) {
		t.Fatalf("Unlanded = %+v, want %+v", a.Unlanded, want)
	}
	for i := range want {
		if a.Unlanded[i] != want[i] {
			t.Errorf("Unlanded[%d] = %+v, want %+v", i, a.Unlanded[i], want[i])
		}
	}
	// A marker still waiting on one of its ids is doing its job.
	if len(a.StaleMarkers) != 0 {
		t.Errorf("a marker with an id still to land was reported stale: %+v", a.StaleMarkers)
	}
}

// TestAuditReferencesLeavesTheRecordsOwnRulingsStanding: an illustrative
// number is not a citation, and a baseline entry is the record's ruling on its
// id, drained by the baseline's own ratchet; neither refuses the cut.
func TestAuditReferencesLeavesTheRecordsOwnRulingsStanding(t *testing.T) {
	root, _ := auditFixture(t, strings.Join([]string{
		"A gate refuses adr-999 here. <!-- record-lint: illustrative -->",
		"",
		"The renderer half is fixed (iss-2608231243286557).",
	}, "\n"))
	writeFile(t, root, DefaultProseBaselinePath, `{"schema_version": 1, "ids": [
  {"id": "iss-2608231243286557", "class": "suspect", "note": "carried for a human to correct"}]}`)
	if a := audit(t, root); len(a.Unlanded) != 0 || len(a.StaleMarkers) != 0 {
		t.Errorf("the record's own rulings refused or were reported: %+v", a)
	}
}

// TestAuditReferencesReportsOnlyAMarkerWhoseRecordsAllArrived: a marker whose
// every cited id resolves is stale. A sentence ABOUT the marker, in a code
// span, is not a marker, and a marker on a line citing nothing waits on
// nothing the audit can judge.
func TestAuditReferencesReportsOnlyAMarkerWhoseRecordsAllArrived(t *testing.T) {
	root, rel := auditFixture(t, strings.Join([]string{
		"Rests on iss-42 and itd-73, both landed. <!-- record-lint: forward-looking -->",
		"",
		"Mark it `<!-- record-lint: forward-looking -->`, as iss-42 did.",
		"",
		"A line with a marker and no id. <!-- record-lint: forward-looking -->",
		"",
		"```",
		"iss-42 in a fence <!-- record-lint: forward-looking -->",
		"```",
	}, "\n"))
	a := audit(t, root)
	if len(a.Unlanded) != 0 {
		t.Errorf("Unlanded = %+v, want none", a.Unlanded)
	}
	if len(a.StaleMarkers) != 1 {
		t.Fatalf("StaleMarkers = %+v, want the one marker whose records arrived", a.StaleMarkers)
	}
	m := a.StaleMarkers[0]
	if m.File != rel || m.Line != 5 || strings.Join(m.IDs, ",") != "iss-42,itd-73" {
		t.Errorf("stale marker = %+v, want %s:5 citing iss-42,itd-73", m, rel)
	}
}

// TestAuditReferencesGatesNothingWithoutTheRule: a repository with no
// record-lint configuration, or one that does not arm the rule, gates no
// citation at the cut, as it gates none at the change.
func TestAuditReferencesGatesNothingWithoutTheRule(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-2611031720012399-x.md", "Per dcn-2611031615094092.")
	if a := audit(t, root); len(a.Unlanded) != 0 {
		t.Errorf("no configuration, yet Unlanded = %+v", a.Unlanded)
	}
	writeFile(t, root, DefaultRecordLintConfigPath, strings.Replace(proseConfigJSON, `"enabled": true`, `"enabled": false`, 1))
	if a := audit(t, root); len(a.Unlanded) != 0 {
		t.Errorf("the rule is off, yet Unlanded = %+v", a.Unlanded)
	}
}
