package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/recall"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
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
// share the loader's one per-turn budget. With the rules taking most of it, the
// router injects the rules block and a ## MEMORY block in one output, the note
// that fits what the rules left comes back in full, the one that does not comes
// back as one line, and the whole text stays within the budget.
func TestRulesAndNotesShareOneTurnBudget(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	const prompt = "run the tests"
	room := rules.MemoryFloorBytes // what the rules are sized to leave of the budget
	// A fresh window, whose bracket sends notes in full.
	setFill(t, home, "budget-session", 10)

	body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": map[string]any{
		"BULK": map[string]any{"recall": []string{"test"}, "rules": []string{strings.Repeat("bulk words ", (rules.InjectionBudgetBytes-room-200)/11)}}}})
	if err != nil {
		t.Fatal(err)
	}
	writeRepoRules(t, repo, string(body))

	bodyOf := func(word string) string { return strings.TrimSpace(strings.Repeat(word+" words ", 2000/(len(word)+7))) }
	notes := map[string]string{"mem-2610111200000001": "alpha", "mem-2610111200000002": "bravo", "mem-2610111200000003": "charlie"}
	var list [][3]string
	for id, word := range notes {
		list = append(list, [3]string{id, "A lesson on " + word, bodyOf(word)})
	}
	writeProjectNotes(t, repo, list...)

	env, errlog := recallRouterJSON(t, "budget-session", repo, prompt)
	if !slices.Contains(env.Injected, "BULK") || !strings.Contains(env.Text, "bulk words") {
		t.Fatalf("the rules block is missing: injected %v\nstderr:\n%s", env.Injected, errlog)
	}
	if !strings.Contains(env.Text, "## MEMORY") || env.Recalled == nil || len(*env.Recalled) != len(notes) {
		t.Fatalf("recalled %v, want every note, in a ## MEMORY block:\n%s", env.Recalled, errlog)
	}
	full := 0
	for _, word := range notes {
		if !strings.Contains(env.Text, "A lesson on "+word) {
			t.Fatalf("the note on %s has no line:\n%s", word, errlog)
		}
		if strings.Contains(env.Text, word+" words") {
			full++
		}
	}
	if full != 1 {
		t.Fatalf("%d notes came back in full, want the one that fits", full)
	}
	if len(env.Text) > rules.InjectionBudgetBytes {
		t.Fatalf("the turn injected %d bytes, over the %d budget", len(env.Text), rules.InjectionBudgetBytes)
	}
	if !strings.Contains(errlog, "as one line") {
		t.Fatalf("stderr does not name the notes sent as one line:\n%s", errlog)
	}
}

// hookContext is what the host would read of the hook's plain output: the
// additional context where it wrote an object, otherwise the whole of it.
func hookContext(t *testing.T, out string) string {
	t.Helper()
	if !strings.HasPrefix(out, "{") {
		return out
	}
	var o struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &o); err != nil {
		t.Fatalf("not one object: %v\n%.300s", err, out)
	}
	return o.HookSpecificOutput.AdditionalContext
}

// TestTheHooksWholeOutputStaysUnderTheHostCap: rules, notes and away together,
// on the worst prompt a repository can write, come in under what the host keeps,
// each domain as a whole form. Status has no origin to read here; its block is
// bounded by its ten validated lines and is reserved as the away block is.
func TestTheHooksWholeOutputStaysUnderTheHostCap(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	setFill(t, home, "capped", 10)

	domains := map[string]any{}
	for i := range 30 {
		var list []string
		for j := range 8 {
			list = append(list, strings.Repeat(fmt.Sprintf("d%d rule %d ", i, j), 40))
		}
		domains[fmt.Sprintf("D%02d", i)] = map[string]any{"recall": []string{"test"}, "rules": list}
	}
	body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": domains})
	if err != nil {
		t.Fatal(err)
	}
	writeRepoRules(t, repo, string(body))
	writeProjectNotes(t, repo, [3]string{"mem-2610111200000003", "A lesson", strings.TrimSpace(strings.Repeat("note words ", 180))})
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := autonomy.SetAway(repo, "", false, autonomy.Options{Home: home}); err != nil {
		t.Fatal(err)
	}

	out, errlog := runHook(t, hookInputJSON(t, "capped", repo, "run the tests"), "hook", "prompt-router")
	ctx := hookContext(t, out)
	if len(ctx) > rules.HostOutputCap {
		t.Fatalf("the hook wrote %d bytes, over the host's cap of %d\nstderr:\n%s", len(ctx), rules.HostOutputCap, errlog)
	}
	if !strings.Contains(ctx, "## AWAY") {
		t.Fatalf("the away block is missing:\n%.400s\nstderr:\n%s", ctx, errlog)
	}
	if !strings.Contains(ctx, "A lesson") {
		t.Fatalf("the note has no room:\n%s\nstderr:\n%s", ctx, errlog)
	}
	if !strings.Contains(ctx, "abcd rules D00") {
		t.Fatalf("a shortened domain names no command:\n%.600s", ctx)
	}
}

// TestAFloorNoNoteUsesGoesBackToTheRules: a domain that fits the turn is sent
// whole when no note comes back or a short one does, and shortened when a note
// takes the room.
func TestAFloorNoNoteUsesGoesBackToTheRules(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	setFill(t, home, "floor", 10)
	setFill(t, home, "floor-short-note", 10)
	setFill(t, home, "floor-long-note", 10)
	var list []string
	for j := range 8 {
		list = append(list, strings.Repeat(fmt.Sprintf("rule %d ", j), 150))
	}
	body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": map[string]any{
		"LONG": map[string]any{"recall": []string{"zebra"}, "rules": list}}})
	if err != nil {
		t.Fatal(err)
	}
	writeRepoRules(t, repo, string(body))
	env, errlog := recallRouterJSON(t, "floor", repo, "a zebra")
	if !strings.Contains(env.Text, "rule 7 ") {
		t.Fatalf("a domain that fits the turn did not come back whole:\n%.400s\nstderr:\n%s", env.Text, errlog)
	}
	writeProjectNotes(t, repo, [3]string{"mem-2610111200000004", "A zebra lesson", "Zebra words."})
	short, errlog := recallRouterJSON(t, "floor-short-note", repo, "test a zebra")
	if !strings.Contains(short.Text, "rule 7 ") || !strings.Contains(short.Text, "Zebra words.") {
		t.Fatalf("a short note did not hand back the floor it left unused:\n%.400s\nstderr:\n%s", short.Text, errlog)
	}
	writeProjectNotes(t, repo, [3]string{"mem-2610111200000005", "A zebra lesson", "Zebra words. " + strings.Repeat("more words ", 170)})
	long, errlog := recallRouterJSON(t, "floor-long-note", repo, "test a zebra")
	if strings.Contains(long.Text, "rule 7 ") || !strings.Contains(long.Text, "Zebra words.") {
		t.Fatalf("with a long note the floor was not kept for it:\n%.400s\nstderr:\n%s", long.Text, errlog)
	}
}

// TestGivingTheFloorBackStaysWithinWhatTheBlocksLeft: with the blocks read first
// taking nearly all of the turn, a turn with no note hands the rules what those
// blocks left and no more.
func TestGivingTheFloorBackStaysWithinWhatTheBlocksLeft(t *testing.T) {
	rs := rules.RuleSet{SchemaVersion: 1, Domains: map[string]rules.Domain{}}
	for i := range 20 {
		rs.Domains[fmt.Sprintf("D%02d", i)] = rules.Domain{Recall: []string{"zebra"}, Rules: []string{strings.Repeat("words ", 60), strings.Repeat("more ", 60)}}
	}
	const reserved = 7_400
	res, _ := injectWithRecall(rs, "zebra", rules.SessionState{}, 0, rules.BracketUnknown, reserved, func(map[string]string, int) recall.Result { return recall.Result{} })
	if room := rules.InjectionBudgetBytes - reserved; len(res.Text) > room {
		t.Fatalf("the rules took %d bytes of the %d the blocks left", len(res.Text), room)
	}
}

// TestRulesPromptReportsAShortenedDomainTruthfully: the diagnostic says what
// the hook would send, never in full for a domain that goes short.
func TestRulesPromptReportsAShortenedDomainTruthfully(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	var list []string
	for j := range 8 {
		list = append(list, strings.Repeat(fmt.Sprintf("rule %d ", j), 200))
	}
	body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": map[string]any{
		"LONG": map[string]any{"recall": []string{"zebra"}, "rules": list}}})
	if err != nil {
		t.Fatal(err)
	}
	writeRepoRules(t, repo, string(body))
	jout, stderr, err := runCLISplit(t, "rules", "--prompt", "zebra", "--json")
	if err != nil {
		t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
	}
	var env struct {
		Domains []struct {
			Name string `json:"name"`
			Form string `json:"form"`
		} `json:"domains"`
	}
	if err := json.Unmarshal([]byte(jout), &env); err != nil {
		t.Fatal(err)
	}
	for _, d := range env.Domains {
		if d.Name == "LONG" {
			if d.Form != promptFormShrunk {
				t.Fatalf("form = %q, want %q", d.Form, promptFormShrunk)
			}
			return
		}
	}
	t.Fatalf("LONG did not match: %+v", env.Domains)
}

// TestRulesPromptTakesTheAwayBlockOffItsBudget: `rules --prompt` reads the away
// block the hook reads first, so a domain that fits a turn whole on its own is
// reported shortened once the away block takes its share, as the hook sends it.
func TestRulesPromptTakesTheAwayBlockOffItsBudget(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	var list []string
	for j := range 8 {
		list = append(list, strings.Repeat(fmt.Sprintf("rule %d ", j), 140))
	}
	body, err := json.Marshal(map[string]any{"schema_version": 1, "domains": map[string]any{
		"LONG": map[string]any{"recall": []string{"zebra"}, "rules": list}}})
	if err != nil {
		t.Fatal(err)
	}
	writeRepoRules(t, repo, string(body))
	form := func() string {
		t.Helper()
		jout, stderr, err := runCLISplit(t, "rules", "--prompt", "zebra", "--json")
		if err != nil {
			t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
		}
		var env struct {
			Domains []struct {
				Name string `json:"name"`
				Form string `json:"form"`
			} `json:"domains"`
		}
		if err := json.Unmarshal([]byte(jout), &env); err != nil {
			t.Fatal(err)
		}
		for _, d := range env.Domains {
			if d.Name == "LONG" {
				return d.Form
			}
		}
		t.Fatalf("LONG did not match: %+v", env.Domains)
		return ""
	}
	if got := form(); got != promptFormFull {
		t.Fatalf("with no away block, form = %q, want %q", got, promptFormFull)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := autonomy.SetAway(repo, "", false, autonomy.Options{Home: home}); err != nil {
		t.Fatal(err)
	}
	if got := form(); got != promptFormShrunk {
		t.Fatalf("with the away block, form = %q, want %q", got, promptFormShrunk)
	}
}

// TestRulesPromptNamesAnAwayReadGitDidNotAnswer: in a managed checkout, a
// root-commit lookup git did not answer leaves the away record unread, as in the
// hook, and `rules --prompt` says so on stderr, once, rather than report a fuller
// form without a word. An answered lookup says nothing, and on a drive's stage,
// where the hook reads no away record, nothing is looked up or said.
func TestRulesPromptNamesAnAwayReadGitDidNotAnswer(t *testing.T) {
	const line = "the away record was not read, so nothing is reserved for it"
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := routerRootCommit
	t.Cleanup(func() { routerRootCommit = prev })
	stderrOf := func() string {
		t.Helper()
		_, stderr, err := runCLISplit(t, "rules", "--prompt", "zebra")
		if err != nil {
			t.Fatalf("rules --prompt: %v\n%s", err, stderr)
		}
		return stderr
	}
	if got := strings.Count(stderrOf(), line); got != 0 {
		t.Fatalf("an answered lookup named an unread away record %d time(s)", got)
	}
	asked := 0
	routerRootCommit = func(context.Context, string) (string, error) {
		asked++
		return "", fmt.Errorf("git rev-list -n 1 --max-parents=0 HEAD: %w", gitutil.ErrGitTimedOut)
	}
	if got := strings.Count(stderrOf(), line); got != 1 {
		t.Fatalf("a lookup git did not answer was named %d time(s), want once", got)
	}
	t.Setenv(runner.StageWritesEnv, "1")
	asked = 0
	if stderr := stderrOf(); strings.Contains(stderr, line) || asked != 0 {
		t.Fatalf("on a drive's stage: looked up %d time(s), stderr:\n%s", asked, stderr)
	}
}

// TestRulesPromptRecallsNothingOnADrivesStage: the hook recalls no notes or
// pages on a drive's stage, so `rules --prompt` there shows none either; off
// the stage the same note is shown.
func TestRulesPromptRecallsNothingOnADrivesStage(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	writeProjectNotes(t, repo, [3]string{"mem-2610111200000009", "A lesson", "note words"})
	notes := func() int {
		t.Helper()
		jout, stderr, err := runCLISplit(t, "rules", "--prompt", "test", "--json")
		if err != nil {
			t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
		}
		var env struct {
			Notes []json.RawMessage `json:"notes"`
		}
		if err := json.Unmarshal([]byte(jout), &env); err != nil {
			t.Fatal(err)
		}
		return len(env.Notes)
	}
	if got := notes(); got != 1 {
		t.Fatalf("off a stage, notes shown = %d, want 1", got)
	}
	t.Setenv(runner.StageWritesEnv, "1")
	if got := notes(); got != 0 {
		t.Fatalf("on a drive's stage, notes shown = %d, want 0: the hook recalls none there", got)
	}
}

// TestRulesPromptCapsThePromptLikeTheHook (F5): `rules --prompt` scans the
// same leading bytes of a prompt the hook does, so a recall word that lies
// only past the cap matches no domain in the diagnostic, as it injects none
// in the hook.
func TestRulesPromptCapsThePromptLikeTheHook(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	writeRepoRules(t, repo, widgetsRules)
	prompt := strings.Repeat("x ", 64<<10) + "a widget"
	if len(rules.CapPrompt(prompt)) >= len(prompt)-len("a widget") {
		t.Fatalf("the prompt does not run past the cap")
	}

	jout, stderr, err := runCLISplit(t, "rules", "--prompt", prompt, "--json")
	if err != nil {
		t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
	}
	var env struct {
		Domains []struct {
			Name string `json:"name"`
		} `json:"domains"`
	}
	if err := json.Unmarshal([]byte(jout), &env); err != nil {
		t.Fatalf("rules --prompt --json is not one document: %v\n%s", err, jout)
	}
	for _, d := range env.Domains {
		if d.Name == "WIDGETS" {
			t.Fatalf("a recall word past the prompt cap matched WIDGETS: %+v", env.Domains)
		}
	}
	// The hook, for the same prompt, injects nothing for it either.
	hook, _ := recallRouterJSON(t, "cap-session", repo, prompt)
	if slices.Contains(hook.Injected, "WIDGETS") {
		t.Fatalf("the hook matched past the cap: %v", hook.Injected)
	}
	// And within the cap the word does match, so the test is not vacuous.
	jout, _, _ = runCLISplit(t, "rules", "--prompt", "a widget", "--json")
	if !strings.Contains(jout, `"WIDGETS"`) {
		t.Fatalf("a widget prompt within the cap did not match WIDGETS:\n%s", jout)
	}
}
