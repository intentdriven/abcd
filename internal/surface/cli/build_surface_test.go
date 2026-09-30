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
	for _, k := range []string{"stage", "reason", "remedy"} {
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
			Stage    string `json:"stage"`
		} `json:"lane"`
		Pending []struct {
			Number int `json:"number"`
		} `json:"pending"`
		Next string `json:"next"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Resumed || res.Lane.ID != "lane-1" || res.Lane.SpecStep != 1 || res.Lane.Stage != "worktree" || len(res.Pending) != 1 {
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
	if ref["stage"] != "check" || ref["check"] != "hold" || !strings.Contains(ref["reason"].(string), "awaiting the pacing ruling") {
		t.Fatalf("refusal = %v", ref)
	}
	if checks, _ := ref["checks"].([]any); len(checks) != 8 {
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

// stepJSON is `implement step` / `implement receipt` under --json.
type stepJSON struct {
	PerformedStage string `json:"performed_stage"`
	Stage          string `json:"stage"`
	Awaiting       *struct {
		Role    string `json:"role"`
		Brief   string `json:"brief"`
		Receipt string `json:"receipt"`
	} `json:"awaiting"`
}

func mustStep(t *testing.T, args ...string) stepJSON {
	t.Helper()
	var res stepJSON
	if err := json.Unmarshal([]byte(mustImplement(t, args...)), &res); err != nil {
		t.Fatal(err)
	}
	return res
}

// TestTheHostDrivesALaneThroughTheCLI plays the host through the front door:
// `implement step` makes the lane's worktree, renders its brief and hands the
// lane to an implementer, naming the brief and the receipt; a receipt short of
// the lane is refused naming what is missing; a receipt that verifies advances
// the lane to its validators.
func TestTheHostDrivesALaneThroughTheCLI(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	var start struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &start); err != nil {
		t.Fatal(err)
	}
	if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != "worktree" {
		t.Fatalf("first step = %+v", res)
	}
	if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != "brief" {
		t.Fatalf("second step = %+v", res)
	}
	await := mustStep(t, "implement", "step", "--json")
	if await.Awaiting == nil || await.Awaiting.Role != "implementer" || !strings.HasSuffix(await.Awaiting.Receipt, "/lane-1/receipt.json") {
		t.Fatalf("the implement step names the agent, the brief and the receipt: %+v", await)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(await.Awaiting.Brief)))
	if err != nil || !strings.Contains(string(brief), "- Run make check.") {
		t.Fatalf("the brief carries the conventions: %v", err)
	}

	st, err := os.ReadFile(filepath.Join(repo.Root(), ".abcd", ".work.local", "run", start.RunID, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Lanes []struct {
			Worktree, Branch string
		} `json:"lanes"`
	}
	if err := json.Unmarshal(st, &state); err != nil {
		t.Fatal(err)
	}
	wt := state.Lanes[0].Worktree
	if !strings.HasPrefix(wt, filepath.Join(os.Getenv("HOME"), ".abcd", "worktrees")) {
		t.Fatalf("the worktree is in the store: %q", wt)
	}
	if err := os.WriteFile(filepath.Join(wt, "built.txt"), []byte("built\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", wt, "add", "built.txt")
	repo.Git("-C", wt, "commit", "-q", "-m", "build it")
	sha := strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD"))

	laneDir := filepath.Dir(filepath.Join(repo.Root(), filepath.FromSlash(await.Awaiting.Receipt)))
	receipt := `{"schema_version":1,"run_id":"` + start.RunID + `","lane":"lane-1","branch":"` + state.Lanes[0].Branch +
		`","commits":["` + sha + `"],"definition_of_done":{"command":"make check","exit_code":0,"output":"dod.log"},"report":"report.md"}`
	if err := os.WriteFile(filepath.Join(laneDir, "receipt.json"), []byte(receipt), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := refusalDocs(t, 2, "implement", "receipt", await.Awaiting.Receipt, "--json")
	if ref["stage"] != "receipt" || !strings.Contains(ref["reason"].(string), "dod.log does not exist") ||
		!strings.Contains(ref["reason"].(string), "report.md does not exist") {
		t.Fatalf("a short receipt is refused naming what is missing: %v", ref)
	}
	for name, body := range map[string]string{"dod.log": "ok\n", "report.md": "built it\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if res := mustStep(t, "implement", "receipt", await.Awaiting.Receipt, "--json"); res.PerformedStage != "implement" || res.Stage != "validate" {
		t.Fatalf("a verified receipt advances the lane to its validators: %+v", res)
	}

	// The validate stage (piece 8): a fresh ruthless reviewer is handed the lane,
	// and its return's verdict is the one the loop records.
	review := mustStep(t, "implement", "step", "--json")
	if review.Awaiting == nil || review.Awaiting.Role != "ruthless-reviewer" || review.PerformedStage != "" {
		t.Fatalf("the validate stage hands the lane to a fresh ruthless reviewer: %+v", review)
	}
	ret := filepath.Join(repo.Root(), filepath.FromSlash(review.Awaiting.Receipt))
	if err := os.WriteFile(ret, []byte("### Verdict\n\n- **SHIP** — nothing survived.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := mustStep(t, "implement", "receipt", review.Awaiting.Receipt, "--json"); res.PerformedStage != "" || res.Stage != "validate" {
		t.Fatalf("a validator's return leaves the lane at validate: %+v", res)
	}
	if status := mustImplement(t, "implement", "status"); !strings.Contains(status, "ruthless-reviewer SHIP") {
		t.Fatalf("the status names the verdict the loop recorded:\n%s", status)
	}
}

// TestImplementStepSaysStageForTheLanesAndStepForTheSpecs: the text form of
// `implement step` and `implement status` names the lane's stage as a stage and
// keeps "step" for the spec's (BU1, iss-2609291313276243).
func TestImplementStepSaysStageForTheLanesAndStepForTheSpecs(t *testing.T) {
	buildRepo(t)
	mustImplement(t, "build", "itd-10", "--json")
	out := mustImplement(t, "implement", "step")
	if !strings.Contains(out, "completed lane-1's worktree stage") || strings.Contains(out, "worktree step") {
		t.Fatalf("the step names the stage it completed:\n%s", out)
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "spec step 1") || !strings.Contains(status, "next stage: brief") {
		t.Fatalf("the status names the spec's step and the lane's next stage:\n%s", status)
	}
	help := mustImplement(t, "implement", "status", "--help")
	if !strings.Contains(help, "spec step and next stage") {
		t.Fatalf("the status help names the lane's next stage as a stage:\n%s", help)
	}
}

// TestImplementReceiptNothingAwaitsIsRefused: a receipt no lane awaits is
// refused, and the state is unchanged.
func TestImplementReceiptNothingAwaitsIsRefused(t *testing.T) {
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
	ref := refusalDocs(t, 2, "implement", "receipt", "receipt.json", "--json")
	if ref["stage"] != "receipt" || !strings.Contains(ref["reason"].(string), "awaits a receipt") {
		t.Fatalf("a receipt nothing awaits is refused: %v", ref)
	}
	// The remedy names the lane's stage as a stage (BU1, iss-2609291313276243).
	if remedy := ref["remedy"].(string); !strings.Contains(remedy, "when a stage hands work to an agent") {
		t.Fatalf("the remedy says a stage hands work to an agent: %q", remedy)
	}
	if after, _ := os.ReadFile(statePath); !bytes.Equal(before, after) {
		t.Fatal("a refused receipt must leave the state unchanged")
	}
}

// TestImplementStepWithoutARunIsRefused: no run in progress, nothing to step.
func TestImplementStepWithoutARunIsRefused(t *testing.T) {
	repo := buildRepo(t)
	ref := refusalDocs(t, 2, "implement", "step", "--json")
	if ref["stage"] != "state" || !strings.Contains(ref["remedy"].(string), "abcd build") {
		t.Fatalf("refusal = %v", ref)
	}
	if out := mustImplement(t, "implement", "status"); !strings.Contains(out, "no run in this checkout") {
		t.Fatalf("status of no run says so:\n%s", out)
	}
	runDirAbsent(t, repo.Root())
}

// TestImplementStatusNamesALaneWorktreeOutsideHomeByItsDirectoryName: status
// --json named a lane's worktree through the home redaction alone, so one
// outside HOME was printed as an absolute local path (iss-2609281329007423).
// It is named by its directory name.
func TestImplementStatusNamesALaneWorktreeOutsideHomeByItsDirectoryName(t *testing.T) {
	repo := buildRepo(t)
	var res struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "itd-10", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(res.State))
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "lane-wt")
	state["lanes"].([]any)[0].(map[string]any)["worktree"] = outside
	raw, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	out := mustImplement(t, "implement", "status", "--json")
	var st struct {
		Runs []struct {
			Lanes []struct {
				Worktree string `json:"worktree"`
			} `json:"lanes"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(out), &st); err != nil || len(st.Runs) != 1 || len(st.Runs[0].Lanes) != 1 {
		t.Fatalf("status --json = %v: %s", err, out)
	}
	if got := st.Runs[0].Lanes[0].Worktree; got != "lane-wt" {
		t.Errorf("the lane worktree is shown as %q, want its directory name lane-wt", got)
	}
	if strings.Contains(out, filepath.Dir(outside)) {
		t.Errorf("status --json prints the absolute worktree path:\n%s", out)
	}
}

// TestBuildForASessionClaimsTheIntent: `build --session` claims the intent in
// the shared run state for a joined session and says so; an unjoined session
// is refused at exit 2 with nothing written; a build without it says the run
// holds no claim (iss-2609252050506863).
func TestBuildForASessionClaimsTheIntent(t *testing.T) {
	repo := buildRepo(t)
	ref := refusalDocs(t, 2, "build", "itd-10", "--session", "ghost", "--json")
	if ref["stage"] != "claim" {
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

// TestBuildWithoutASessionSaysItHoldsNoClaimInJSON: the JSON says what the text
// says — the claim key is present and null, never absent (iss-2609252050506863).
func TestBuildWithoutASessionSaysItHoldsNoClaimInJSON(t *testing.T) {
	buildRepo(t)
	var doc map[string]json.RawMessage
	out := mustImplement(t, "build", "itd-10", "--json")
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("build --json is not an object: %v\n%s", err, out)
	}
	claim, ok := doc["claim"]
	if !ok {
		t.Fatalf("build --json without --session omits the claim key:\n%s", out)
	}
	if string(claim) != "null" {
		t.Fatalf("build --json without --session: claim = %s; want null", claim)
	}
}
