package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImplementCheckNamesTheStage pins ruling CM1 (2026-09-29): `implement
// check` names what a session is about to take a stage, like the loop's lane
// stages (BU1, iss-2609291313276243), in its JSON field, its text, its
// refusals, the run log's refusal line and its help.
func TestImplementCheckNamesTheStage(t *testing.T) {
	_, runDir := implementRepo(t)
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	mustImplement(t, "implement", "join", "--session", "beta", "--role", "second", "--json")

	out := mustImplement(t, "implement", "check", "review", "--session", "beta", "--json")
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("check --json is not JSON: %v\n%s", err, out)
	}
	if v["stage"] != "review" {
		t.Errorf(`check --json: want "stage": "review", got %v in %s`, v["stage"], out)
	}
	if _, old := v["step"]; old {
		t.Errorf(`check --json still carries the v0.11 "step" field: %s`, out)
	}
	if text := mustImplement(t, "implement", "check", "review", "--session", "beta"); !strings.Contains(text, "may take the review stage") {
		t.Errorf("check text does not name the stage: %q", text)
	}
	if msg := refusalEnvelope(t, 2, "implement", "check", "ship", "--session", "alpha", "--json"); !strings.Contains(msg, "unknown stage") {
		t.Errorf("an unknown operand is not refused as an unknown stage: %q", msg)
	}
	if msg := refusalEnvelope(t, 2, "implement", "check", "release", "--session", "beta", "--json"); !strings.Contains(msg, "release stage") {
		t.Errorf("the release refusal does not name the release stage: %q", msg)
	}
	logs, _ := filepath.Glob(filepath.Join(runDir, "*.jsonl"))
	var refusalLine map[string]any
	for _, l := range logs {
		b, err := os.ReadFile(l)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			var e map[string]any
			if json.Unmarshal([]byte(line), &e) == nil && e["event"] == "refusal" {
				refusalLine = e
			}
		}
	}
	if refusalLine == nil {
		t.Fatalf("no refusal line in the run log under %s", runDir)
	}
	if refusalLine["stage"] != "release" {
		t.Errorf(`the logged refusal: want "stage": "release", got %v`, refusalLine)
	}
	if _, old := refusalLine["step"]; old {
		t.Errorf(`the logged refusal still carries "step": %v`, refusalLine)
	}
	help := mustImplement(t, "implement", "check", "--help")
	if !strings.Contains(help, "may take a stage") || !strings.Contains(help, "file the stage touches") {
		t.Errorf("check --help does not say stage for what the session takes:\n%s", help)
	}
	if strings.Contains(help, " step") {
		t.Errorf("check --help still says step:\n%s", help)
	}
}
