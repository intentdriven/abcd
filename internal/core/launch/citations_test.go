package launch

import (
	"strings"
	"testing"
)

func runCitationGate(t *testing.T, pre *CitationPreflight) (GateSummary, []string) {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, repo, ArtefactRelPath, `{"kind": "plugin"}`)
	writeFile(t, repo, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, repo, "commands/x.md", "# doc\n")
	report, err := DryRun(DryRunRequest{RepoRoot: repo, Version: "v0.1.0", Citations: pre})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	for _, g := range report.Gates {
		if g.Name == "citation-baseline" {
			return g, report.WouldRefuseOn
		}
	}
	t.Fatalf("no citation-baseline gate in %+v", report.Gates)
	return GateSummary{}, nil
}

// TestCitationGateIsNotImplementedWithoutAPreflight keeps the gate honest in a
// repo that has not armed the citation rule: it is reported as not run, never as
// a silent pass.
func TestCitationGateIsNotImplementedWithoutAPreflight(t *testing.T) {
	gate, refusals := runCitationGate(t, nil)
	if gate.Status != "not_implemented" {
		t.Fatalf("status = %q, want not_implemented", gate.Status)
	}
	for _, r := range refusals {
		if strings.Contains(r, "citation") {
			t.Fatalf("an unarmed citation gate produced a refusal: %q", r)
		}
	}
}

// TestCitationGateNagsWithoutBlocking is spc-17's nag: entries approaching the
// blocking threshold are named in the gate detail and refuse nothing, so a
// maintainer gets notice while a release still cuts.
func TestCitationGateNagsWithoutBlocking(t *testing.T) {
	gate, refusals := runCitationGate(t, &CitationPreflight{
		Present: true, Cited: 12, Recorded: 12, Approaching: 2, ApproachingURLs: []string{"https://a.example.org/x", "https://b.example.org/y"},
	})
	if gate.Status != "ran" {
		t.Fatalf("status = %q, want ran", gate.Status)
	}
	if !strings.Contains(gate.Detail, "2 approaching") {
		t.Errorf("detail = %q, want the approaching count", gate.Detail)
	}
	for _, r := range refusals {
		if strings.Contains(r, "citation") {
			t.Fatalf("a nag blocked the release: %q", r)
		}
	}
}

// TestCitationGateBlocksOnOverdue is AC 3's release-gate half: past the blocking
// threshold, the release stops, and the refusal NAMES the citations so the
// operator knows what to refresh.
func TestCitationGateBlocksOnOverdue(t *testing.T) {
	gate, refusals := runCitationGate(t, &CitationPreflight{
		Present: true, Cited: 12, Recorded: 12, Overdue: 1, OverdueURLs: []string{"https://a.example.org/x"},
	})
	if gate.Status != "ran" {
		t.Fatalf("status = %q, want ran", gate.Status)
	}
	var named bool
	for _, r := range refusals {
		if strings.Contains(r, "https://a.example.org/x") {
			named = true
		}
	}
	if !named {
		t.Fatalf("refusals = %v, want the overdue citation named", refusals)
	}
}

// TestCitationGateBlocksOnBrokenAndUnreceipted covers the other two ways the
// record fails a release: a citation recorded broken, and one with no receipt.
func TestCitationGateBlocksOnBrokenAndUnreceipted(t *testing.T) {
	_, refusals := runCitationGate(t, &CitationPreflight{Present: true, Cited: 3, Recorded: 1, Broken: 1, Missing: 2})
	joined := strings.Join(refusals, "\n")
	if !strings.Contains(joined, "broken") || !strings.Contains(joined, "receipt") {
		t.Fatalf("refusals = %v, want both the broken and the unreceipted citations to refuse", refusals)
	}
}

// TestCitationGateRefusesAnUnmeasurableBaseline keeps a broken config from
// reading as "no gate here". A repo whose docs-lint config plainly arms the rule
// but fails to parse must not have its release wave through with the gate
// reported as unarmed — that is a false statement about a real requirement.
func TestCitationGateRefusesAnUnmeasurableBaseline(t *testing.T) {
	gate, refusals := runCitationGate(t, &CitationPreflight{Unreadable: "docs-lint.json: invalid character '}'"})
	if gate.Status != "ran" {
		t.Fatalf("status = %q, want ran", gate.Status)
	}
	if !strings.Contains(gate.Detail, "could not be measured") {
		t.Errorf("detail = %q, want it to say the baseline could not be measured", gate.Detail)
	}
	var refused bool
	for _, r := range refusals {
		if strings.Contains(r, "citation") {
			refused = true
		}
	}
	if !refused {
		t.Fatalf("refusals = %v, want an unmeasurable baseline to refuse", refusals)
	}
}

// TestCitationGatePassesOnAHealthyBaseline pins that a current record refuses
// nothing at all.
func TestCitationGatePassesOnAHealthyBaseline(t *testing.T) {
	gate, refusals := runCitationGate(t, &CitationPreflight{Cited: 12, Recorded: 12, Present: true})
	if !strings.Contains(gate.Detail, "12") {
		t.Errorf("detail = %q, want the cited count", gate.Detail)
	}
	for _, r := range refusals {
		if strings.Contains(r, "citation") {
			t.Fatalf("a healthy baseline refused: %q", r)
		}
	}
}
