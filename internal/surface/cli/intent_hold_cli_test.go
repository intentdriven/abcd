package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// intent_hold_cli_test.go — iss-2609200830076665. The hold pair at the CLI:
// hold writes the key and reports it (JSON carries `redacted` like the other
// write verbs), plan refuses the held record on exit 2 naming the lift, the
// dispatcher leads with the hold, unhold removes the line, and each refusal
// exits 2 with nothing written.
func TestIntentHoldUnholdAtTheCLI(t *testing.T) {
	repo := intentTestRepo(t)
	rel := cliDrafts + "/itd-10-alpha.md"
	writeRepoFile(t, repo, rel, cliDraftWithAC("itd-10", "alpha"))
	before, _ := os.ReadFile(filepath.Join(repo, rel))

	// A missing reason is refused at the door.
	if _, err := runCLIErr(t, "intent", "hold", "itd-10"); err == nil || !strings.Contains(err.Error(), "--reason is required") {
		t.Fatalf("hold without --reason must refuse: %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(repo, rel)); string(after) != string(before) {
		t.Fatal("a refused hold wrote the record")
	}

	out := runCLI(t, "intent", "hold", "itd-10", "--reason", "awaiting the reading rethink", "--json")
	var held struct {
		IntentID string `json:"intent_id"`
		Bucket   string `json:"bucket"`
		Reason   string `json:"reason"`
		Redacted int    `json:"redacted"`
	}
	if err := json.Unmarshal(out, &held); err != nil {
		t.Fatalf("hold --json: %v\n%s", err, out)
	}
	if held.IntentID != "itd-10" || held.Bucket != "drafts" || held.Reason != "awaiting the reading rethink" || held.Redacted != 0 {
		t.Fatalf("hold payload = %+v", held)
	}

	// The record loads through the read-only status board and the dispatcher
	// leads with the hold.
	if out := string(runCLI(t, "intent", "--json")); !strings.Contains(out, `"drafts": 1`) {
		t.Fatalf("a held record must still load: %s", out)
	}
	desc := string(runCLI(t, "itd-10"))
	if !strings.Contains(desc, "held: awaiting the reading rethink") || !strings.Contains(desc, "abcd intent unhold itd-10") {
		t.Fatalf("dispatcher must report the hold and the lift:\n%s", desc)
	}

	// plan refuses, exit 2, naming the reason and the lift; nothing moves.
	if _, err := runCLIErr(t, "intent", "plan", "itd-10"); err == nil || !strings.Contains(err.Error(), "awaiting the reading rethink") || !strings.Contains(err.Error(), "intent unhold itd-10") {
		t.Fatalf("plan must refuse a held draft naming the remedy: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, rel)); statErr != nil {
		t.Fatal("the held draft moved")
	}

	// A second hold is refused naming the standing reason.
	if _, err := runCLIErr(t, "intent", "hold", "itd-10", "--reason", "another"); err == nil || !strings.Contains(err.Error(), "awaiting the reading rethink") {
		t.Fatalf("re-hold must refuse naming the standing reason: %v", err)
	}

	// unhold removes the line and reports what stood; the record is byte-identical
	// to before the hold, and plans.
	text := string(runCLI(t, "intent", "unhold", "itd-10"))
	if !strings.Contains(text, "abcd intent unhold") || !strings.Contains(text, "awaiting the reading rethink") {
		t.Fatalf("unhold render: %s", text)
	}
	if after, _ := os.ReadFile(filepath.Join(repo, rel)); string(after) != string(before) {
		t.Fatalf("unhold must restore the record byte for byte:\n%s", after)
	}
	if _, err := runCLIErr(t, "intent", "unhold", "itd-10"); err == nil || !strings.Contains(err.Error(), "not held") {
		t.Fatalf("unhold on a record not held must refuse: %v", err)
	}
	if out := string(runCLI(t, "intent", "plan", "itd-10")); !strings.Contains(out, "drafts -> planned") {
		t.Fatalf("the lifted record plans: %s", out)
	}
}

// TestIntentHoldRenderMasksPathTail: the hold renders interpolate the path
// tail and the operator's reason, both masked like the sibling renders
// (iss-259).
func TestIntentHoldRenderMasksPathTail(t *testing.T) {
	repo := intentTestRepo(t)
	name := "itd-10-alpha" + poisonTail() + ".md"
	writeRepoFile(t, repo, cliDrafts+"/"+name, cliDraftWithAC("itd-10", "alpha"))
	out := string(runCLI(t, "intent", "hold", "itd-10", "--reason", "why"))
	assertNoAttackRunes(t, "intent hold", out)
	out = string(runCLI(t, "intent", "unhold", "itd-10"))
	assertNoAttackRunes(t, "intent unhold", out)
}
