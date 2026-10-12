package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// writeProjectNotes writes the project's .abcd/memory.json with notes, each
// an id, a title and a body recalled on "test".
func writeProjectNotes(t *testing.T, repo string, notes ...[3]string) {
	t.Helper()
	var list []map[string]any
	for _, n := range notes {
		list = append(list, map[string]any{"id": n[0], "title": n[1], "body": n[2], "recall": []string{"test"},
			"origin": "agent", "written_by": "Example:model-1", "written_at": "2026-10-11T12:00:00Z", "session": nil})
	}
	data, err := json.Marshal(map[string]any{"schema_version": 1, "notes": list})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "memory.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRulesAndNotesShareOneTurnBudget (F4): the rules and the memory recall
// share the loader's one per-turn budget. With the rules taking most of it,
// the router injects the rules block and a ## MEMORY block in one output,
// the note that fits what the rules left comes back, the note that does not
// is omitted — named on stderr and left unrecorded, so the next turn brings
// it back — and the whole text stays within the budget.
func TestRulesAndNotesShareOneTurnBudget(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	const prompt = "run the tests"
	const room = 1500 // what the rules are sized to leave of the budget

	// Size one domain's rule so the rules render to the budget less room.
	rulesText := func(size int) string {
		body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": map[string]any{
			"BULK": map[string]any{"recall": []string{"test"}, "rules": []string{strings.Repeat("bulk words ", size/11)}}}})
		if err != nil {
			t.Fatal(err)
		}
		writeRepoRules(t, repo, string(body))
		rs, err := rules.Load(repo)
		if err != nil {
			t.Fatal(err)
		}
		return rules.Inject(rs, prompt, rules.SessionState{}, 0).Text
	}
	size := rules.InjectionBudgetBytes - room
	size += rules.InjectionBudgetBytes - room - len(rulesText(size))
	rendered := rulesText(size)
	left := rules.InjectionBudgetBytes - len(rendered)
	if left < room-20 || left > room+20 {
		t.Fatalf("the rules left %d bytes, want about %d", left, room)
	}

	const small, big = "mem-2610111200000001", "mem-2610111200000002"
	writeProjectNotes(t, repo,
		[3]string{small, "A short lesson", "Short words."},
		[3]string{big, "A long lesson", strings.TrimSpace(strings.Repeat("long words ", room/11))})

	env, errlog := recallRouterJSON(t, "budget-session", repo, prompt)
	if !slices.Contains(env.Injected, "BULK") || !strings.Contains(env.Text, "## BULK") {
		t.Fatalf("the rules block is missing: injected %v\nstderr:\n%s", env.Injected, errlog)
	}
	if !strings.Contains(env.Text, "## MEMORY") || env.Recalled == nil || !slices.Equal(*env.Recalled, []string{small}) {
		t.Fatalf("recalled %v, want only %s, in a ## MEMORY block:\n%s", env.Recalled, small, errlog)
	}
	if strings.Contains(env.Text, "A long lesson") {
		t.Fatalf("the note past the remaining budget was injected")
	}
	if len(env.Text) > rules.InjectionBudgetBytes {
		t.Fatalf("the turn injected %d bytes, over the %d budget", len(env.Text), rules.InjectionBudgetBytes)
	}
	if !strings.Contains(errlog, "did not fit this turn's budget") || !strings.Contains(errlog, big) {
		t.Fatalf("stderr does not name the omitted note:\n%s", errlog)
	}

	// Next turn the rules are already sent, so the omitted note, never
	// recorded, now fits and comes back.
	next, _ := recallRouterJSON(t, "budget-session", repo, prompt)
	if next.Recalled == nil || !slices.Equal(*next.Recalled, []string{big}) {
		t.Fatalf("the omitted note was recorded on the turn it was left out: recalled %v", next.Recalled)
	}
}
