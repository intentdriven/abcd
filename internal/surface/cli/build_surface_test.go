package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	buildPlanned = ".abcd/development/intents/planned/itd-10-alpha.md"
	buildSpec    = ".abcd/development/specs/open/spc-1-alpha.md"
)

// buildIntent is a planned intent every pre-start check passes.
func buildIntent() string {
	return "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		"## Open Questions\n\n_None open._\n" + cliGroundsSection
}

// buildRepo stands up a committed repository holding a READY intent and its
// written spec, with the local tier a run lives in, under a temporary HOME, and
// changes into it.
func buildRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(buildPlanned, buildIntent())
	repo.Write(buildSpec, "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n\n"+
		"## Steps\n\n1. The state file\n2. The verb\n")
	repo.Commit("init")
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo.Root())
	return repo
}

// jsonStream decodes every JSON document on a stream.
func jsonStream(t *testing.T, out string) []map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var docs []map[string]any
	for {
		var d map[string]any
		if err := dec.Decode(&d); errors.Is(err, io.EOF) {
			return docs
		} else if err != nil {
			t.Fatalf("stdout is not a stream of JSON documents (%v): %q", err, out)
		}
		docs = append(docs, d)
	}
}

// refusalDocs asserts a --json refusal: the refusal document with its step,
// reason and remedy as fields, then the error envelope, and nothing on stderr.
func refusalDocs(t *testing.T, want int, args ...string) map[string]any {
	t.Helper()
	code, out, errOut := implementCLI(t, args...)
	if code != want {
		t.Fatalf("abcd %s exited %d, want %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, want, out, errOut)
	}
	if strings.TrimSpace(errOut) != "" {
		t.Fatalf("a --json refusal wrote prose to stderr: %q", errOut)
	}
	docs := jsonStream(t, out)
	if len(docs) != 2 || docs[1]["abcd"] != "error" {
		t.Fatalf("want the refusal document then the envelope, got %d document(s): %s", len(docs), out)
	}
	ref, ok := docs[0]["refusal"].(map[string]any)
	if !ok {
		t.Fatalf("the first document carries no refusal: %s", out)
	}
	for _, k := range []string{"step", "reason", "remedy"} {
		if s, _ := ref[k].(string); s == "" {
			t.Fatalf("the refusal names its %s as a field: %s", k, out)
		}
	}
	return ref
}

func runDirAbsent(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(root, ".abcd", ".work.local", "run")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refusal must write no state (%v)", err)
	}
}

// TestBuildStartsARunThatStatusRendersAndABuildAgainResumes: `abcd build`
// creates the state file with one lane, `implement status` renders it, and a
// second build names the same run.
func TestBuildStartsARunThatStatusRendersAndABuildAgainResumes(t *testing.T) {
	repo := buildRepo(t)
	out := mustImplement(t, "build", "itd-10", "--json")
	var res struct {
		RunID   string `json:"run_id"`
		State   string `json:"state"`
		Resumed bool   `json:"resumed"`
		Lane    struct {
			ID       string `json:"id"`
			SpecStep int    `json:"spec_step"`
			Step     string `json:"step"`
		} `json:"lane"`
		Pending []struct {
			Number int `json:"number"`
		} `json:"pending"`
		Next string `json:"next"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Resumed || res.Lane.ID != "lane-1" || res.Lane.SpecStep != 1 || res.Lane.Step != "worktree" || len(res.Pending) != 1 {
		t.Fatalf("build result = %+v", res)
	}
	if !strings.Contains(res.Next, "abcd implement step") {
		t.Fatalf("the next move names the step verb: %q", res.Next)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(res.State))); err != nil {
		t.Fatalf("the state file exists: %v", err)
	}

	text := mustImplement(t, "implement", "status")
	for _, want := range []string{res.RunID, "itd-10", "lane-1", "The state file", "pending", "The verb"} {
		if !strings.Contains(text, want) {
			t.Fatalf("status must render %q:\n%s", want, text)
		}
	}
	var st struct {
		Runs []struct {
			RunID string `json:"run_id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "status", "--json")), &st); err != nil || len(st.Runs) != 1 {
		t.Fatalf("status --json lists the one run: %+v %v", st, err)
	}

	again := mustImplement(t, "build", "itd-10")
	if !strings.Contains(again, "resumed run "+res.RunID) {
		t.Fatalf("a second build resumes the run:\n%s", again)
	}
}

// TestBuildRefusalNamesStepReasonAndRemedy is criterion 13 at the surface: a
// refused build names the step, the check, the reason and the remedy in text
// and in --json, exits 2, and writes nothing.
func TestBuildRefusalNamesStepReasonAndRemedy(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(buildPlanned, strings.Replace(buildIntent(), "kind: standalone\n", "kind: standalone\nheld: \"awaiting the pacing ruling\"\n", 1))
	repo.Commit("hold")

	ref := refusalDocs(t, 2, "build", "itd-10", "--json")
	if ref["step"] != "check" || ref["check"] != "hold" || !strings.Contains(ref["reason"].(string), "awaiting the pacing ruling") {
		t.Fatalf("refusal = %v", ref)
	}
	if checks, _ := ref["checks"].([]any); len(checks) != 7 {
		t.Fatalf("the refusal carries every check's row, got %d", len(checks))
	}
	code, _, errOut := implementCLI(t, "build", "itd-10")
	if code != 2 || !strings.Contains(errOut, "refused at check (hold)") || !strings.Contains(errOut, "remedy: settle the hold") {
		t.Fatalf("text refusal: exit %d\n%s", code, errOut)
	}
	runDirAbsent(t, repo.Root())
}

// TestBuildRefusesAPeerHoldingTheIntentAtExit3 is criterion 2 at the surface:
// contention, so exit 3, naming the peer.
func TestBuildRefusesAPeerHoldingTheIntentAtExit3(t *testing.T) {
	repo := buildRepo(t)
	repo.Git("checkout", "-q", "-b", "lane-alpha")
	repo.Remove(buildPlanned)
	repo.Write(".abcd/development/intents/shipped/itd-10-alpha.md", buildIntent())
	repo.Commit("deliver alpha")
	repo.Git("checkout", "-q", "main")

	ref := refusalDocs(t, 3, "build", "itd-10", "--json")
	if ref["check"] != "peers" || !strings.Contains(ref["reason"].(string), "lane-alpha") {
		t.Fatalf("refusal = %v", ref)
	}
	runDirAbsent(t, repo.Root())
}

// TestImplementStepRefusesAStepThisBuildDoesNotCarry: the production sequence
// carries no step body yet, so the first step is refused naming the piece that
// delivers it, and the state is unchanged.
func TestImplementStepRefusesAStepThisBuildDoesNotCarry(t *testing.T) {
	repo := buildRepo(t)
	var res struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(res.State))
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	ref := refusalDocs(t, 2, "implement", "step", "--json")
	if ref["step"] != "worktree" || ref["lane"] != "lane-1" || !strings.Contains(ref["reason"].(string), "piece 6") {
		t.Fatalf("refusal = %v", ref)
	}
	after, _ := os.ReadFile(statePath)
	if !bytes.Equal(before, after) {
		t.Fatal("a refused step must leave the state unchanged")
	}

	ref = refusalDocs(t, 2, "implement", "receipt", "receipt.json", "--json")
	if ref["step"] != "receipt" || !strings.Contains(ref["reason"].(string), "awaits a receipt") {
		t.Fatalf("a receipt nothing awaits is refused: %v", ref)
	}
}

// TestImplementStepWithoutARunIsRefused: no run in progress, nothing to step.
func TestImplementStepWithoutARunIsRefused(t *testing.T) {
	repo := buildRepo(t)
	ref := refusalDocs(t, 2, "implement", "step", "--json")
	if ref["step"] != "state" || !strings.Contains(ref["remedy"].(string), "abcd build") {
		t.Fatalf("refusal = %v", ref)
	}
	if out := mustImplement(t, "implement", "status"); !strings.Contains(out, "no run in this checkout") {
		t.Fatalf("status of no run says so:\n%s", out)
	}
	runDirAbsent(t, repo.Root())
}

// TestBuildForASessionClaimsTheIntent: `build --session` claims the intent in
// the shared run state for a joined session and says so; an unjoined session
// is refused at exit 2 with nothing written; a build without it says the run
// holds no claim (iss-2609252050506863).
func TestBuildForASessionClaimsTheIntent(t *testing.T) {
	repo := buildRepo(t)
	ref := refusalDocs(t, 2, "build", "itd-10", "--session", "ghost", "--json")
	if ref["step"] != "claim" {
		t.Fatalf("an unjoined session's build = %v; want the claim step refused", ref)
	}
	runDirAbsent(t, repo.Root())
	mustImplement(t, "implement", "join", "--session", "host-a", "--role", "first", "--json")
	out := mustImplement(t, "build", "itd-10", "--session", "host-a")
	if !strings.Contains(out, "claim:   itd-10 for session host-a") {
		t.Fatalf("build --session does not report its claim:\n%s", out)
	}
	if out := mustImplement(t, "implement", "--json"); !strings.Contains(out, `"record": "itd-10"`) {
		t.Fatalf("the shared run holds no claim on itd-10:\n%s", out)
	}
}

// TestBuildWithoutASessionSaysItHoldsNoClaim: the invisibility of a run started
// without --session is named, not silent.
func TestBuildWithoutASessionSaysItHoldsNoClaim(t *testing.T) {
	buildRepo(t)
	if out := mustImplement(t, "build", "itd-10"); !strings.Contains(out, "claim:   none (no --session)") {
		t.Fatalf("build without --session is silent about its claim:\n%s", out)
	}
}
