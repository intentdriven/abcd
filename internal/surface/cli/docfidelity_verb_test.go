package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
)

const holdDraft = `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [
	{"doc": "brief", "chapter": "01-all.md", "sentence": "Alpha prints YAML.", "replacement": "Alpha prints JSON.",
	 "evidence": "alpha.go prints JSON", "disposition": "confirmed"}]}`

func TestDocsFidelityUnarmedJudgesNothing(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	out := string(runCLI(t, "docs", "fidelity"))
	if !strings.Contains(out, "not armed") {
		t.Fatalf("unarmed output: %s", out)
	}
}

// ac-7: the per-task pass reports and blocks nothing.
func TestDocsFidelityReportModeStatesFindingsAndExitsZero(t *testing.T) {
	armedCloseRepo(t, "abcd spec close")
	out := string(runCLI(t, "docs", "fidelity", "--report"))
	if !strings.Contains(out, "`abcd spec close`") || !strings.Contains(out, docfidelity.RunReviewFirst) {
		t.Fatalf("report mode did not state the findings:\n%s", out)
	}
	if _, err := runCLIErr(t, "docs", "fidelity"); err == nil {
		t.Fatal("gate mode allowed an undocumented surface")
	}
}

func TestDocsFidelityRecordSavesTheReviewTheGateFinds(t *testing.T) {
	armedCloseRepo(t)
	if _, err := runCLIErr(t, "docs", "fidelity"); err == nil {
		t.Fatal("the gate passed with no review saved")
	}
	out := string(runCLIStdin(t, `{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`,
		"docs", "fidelity", "record", "--verdict-json", "-"))
	if !strings.Contains(out, docfidelity.ReceiptsDir) {
		t.Fatalf("record did not say where the review was saved:\n%s", out)
	}
	runCLI(t, "docs", "fidelity")
	runCLI(t, "spec", "close", "spc-1")
}

// ac-5 end to end: the drafted edit is applied and flagged, and the shipped
// move then proceeds.
func TestDocsFidelityApplyAppliesTheDraftAndTheCloseProceeds(t *testing.T) {
	repo := armedCloseRepo(t)
	chapter := filepath.Join(repo, docfidelity.ChaptersDir, "01-all.md")
	data, _ := os.ReadFile(chapter)
	if err := os.WriteFile(chapter, append(data, []byte("\nAlpha prints YAML.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	runCLIStdin(t, holdDraft, "docs", "fidelity", "record", "--verdict-json", "-")
	if _, err := runCLIErr(t, "spec", "close", "spc-1"); err == nil || !strings.Contains(err.Error(), "--apply") {
		t.Fatalf("the close did not refuse pointing at the drafted edit: %v", err)
	}
	out := string(runCLI(t, "docs", "fidelity", "--apply"))
	if !strings.Contains(out, "Alpha prints JSON.") || !strings.Contains(out, docfidelity.FlagsPath) {
		t.Fatalf("apply did not list the applied edit and its flag:\n%s", out)
	}
	data, _ = os.ReadFile(chapter)
	if strings.Contains(string(data), "Alpha prints YAML.") {
		t.Fatal("the chapter still carries the false sentence")
	}
	runCLI(t, "spec", "close", "spc-1")
}

// ac-9: the autonomous flag applies and lists every edit, hands the routine
// the reviewer's request, and still fails closed with no review.
func TestDocsFidelityAutonomous(t *testing.T) {
	armedCloseRepo(t)
	out, err := runCLIErr(t, "docs", "fidelity", "--autonomous", "--json")
	if err == nil {
		t.Fatal("the autonomous run passed with no review saved")
	}
	var got struct {
		Request *struct {
			Commit string `json:"commit"`
			Record string `json:"record_with"`
		} `json:"request"`
	}
	if jerr := json.Unmarshal(out, &got); jerr != nil || got.Request == nil || got.Request.Commit == "" ||
		!strings.Contains(got.Request.Record, "docs fidelity record") {
		t.Fatalf("no reviewer request in the autonomous output (%v):\n%s", jerr, out)
	}
}

func TestDocsFidelityReportAndApplyAreExclusive(t *testing.T) {
	armedCloseRepo(t)
	if _, err := runCLIErr(t, "docs", "fidelity", "--report", "--apply"); err == nil {
		t.Fatal("report mode accepted --apply")
	}
}
