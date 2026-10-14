package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// laterHiddenRunes are the runes termsafe.Sanitize came to mask after the
// stored-text refusals were written (iss-2610121602237516): a flag emoji's tag
// sequence, the combining grapheme joiner, the Mongolian vowel separator and
// an interlinear annotation mark. A run an earlier abcd wrote may carry any of
// them in text a person or a tool legitimately wrote.
var laterHiddenRunes = []string{"\U0001F3F4\U000E0067\U000E0062\U000E0073\U000E0063\U000E0074\U000E007F", "͏", "᠎", "￹"}

// TestARunStateMasksTheRunesItsRefusalsNeverRefused: a hand-back, a receipt's
// model and a resolution's text carrying a rune termsafe masks only since the
// refusals were written is read, not refused, and comes back masked, the
// prose keeping its line breaks; the refusals' own set is still refused.
func TestARunStateMasksTheRunesItsRefusalsNeverRefused(t *testing.T) {
	repo, runID, l, dir := issueAwaiting(t)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, runID, l, dir, sha)
	rc.Model = "Claude:claude-opus-5-5"
	rc.Resolves = []Resolution{{Issue: eligibleIssue, Commit: sha, Note: "guarded the map\nand the slice", Impact: "fix", Grounds: "pursued: the guard holds"}}
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	stored := stateBytes(t, repo.Root(), runID)
	defer func() {
		if err := os.WriteFile(statePath, stored, 0o600); err != nil {
			t.Fatal(err)
		}
	}()
	forge := func(t *testing.T, bad string, edit func(lane map[string]any)) {
		t.Helper()
		var doc map[string]any
		if err := json.Unmarshal(stored, &doc); err != nil {
			t.Fatal(err)
		}
		edit(doc["lanes"].([]any)[0].(map[string]any))
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(statePath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range laterHiddenRunes {
		forge(t, bad, func(lane map[string]any) {
			lane["receipts"].([]any)[0].(map[string]any)["model"] = "claude" + bad
			res := lane["resolves"].([]any)[0].(map[string]any)
			res["note"] = "guarded the map" + bad + "\nand the slice"
			res["grounds"] = "pursued: " + bad
			hb := map[string]any{}
			for k, v := range capHandBack {
				hb[k] = v
			}
			hb["verdicts"] = "ruthless-reviewer FIX FIRST " + bad
			hb["findings"] = []any{"lane-1/ruthless-2.md" + bad}
			lane["hand_back"], lane["stage"] = hb, string(StageHandedBack)
		})
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatalf("%+q: a run whose stored text carries the rune is read, not refused: %v", bad, err)
		}
		if _, err := ReadRecord(repo.Root(), runID); err != nil {
			t.Fatalf("%+q: the record reads: %v", bad, err)
		}
		b, err := json.Marshal(st)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), bad) {
			t.Errorf("%+q: the rune reads back raw: %s", bad, b)
		}
		ln := st.Lanes[0]
		if ln.Receipts[0].Model != "claude"+termsafe.Sanitize(bad) || ln.Resolves[0].Note != "guarded the map"+termsafe.Sanitize(bad)+"\nand the slice" ||
			!strings.HasPrefix(ln.HandBack.Verdicts, "ruthless-reviewer FIX FIRST ") || strings.Contains(ln.HandBack.Findings[0], bad) {
			t.Errorf("%+q: each field reads back masked, the note keeping its line break: %+v %+v", bad, ln.Resolves, ln.HandBack)
		}
	}
	for _, bad := range []string{"‮", "\u009b", "​"} {
		forge(t, bad, func(lane map[string]any) { lane["receipts"].([]any)[0].(map[string]any)["model"] = "claude" + bad })
		r := mustRefusal(t, func() error { _, err := ReadState(repo.Root(), runID); return err }())
		if !strings.Contains(r.Reason, "carries a receipt's model or resolution the loop never writes") {
			t.Fatalf("%s: the refusals' own set is still refused: %+v", fmt.Sprintf("%+q", bad), r)
		}
	}
}
