package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
)

// stageRaw stages one raw transcript for the repo under test and returns the
// staged file's path.
func stageRaw(t *testing.T, repoRoot, rootSHA, session, body string) string {
	t.Helper()
	res, err := history.Stage(repoRoot, rootSHA,
		history.StageMeta{Lineage: history.CaptureMeta{SessionID: session, Kind: "native"}},
		[]byte(body))
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	return res.Staged.Path
}

// TestPromptRouterDrainsWhileTheSessionIsLive is the first limb of
// iss-2609090722466403. The drain ran from SessionStart alone, so raw text
// waited for a NEW session in the SAME repository — and sub-agent capture
// stages during a session, which means a session's own raw transcripts sat
// unredacted on disk while that session was still running. UserPromptSubmit is
// the only hook that fires while a session is live.
func TestPromptRouterDrainsWhileTheSessionIsLive(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo, rootSHA := sessionEndRepo(t)
	staged := stageRaw(t, repo, rootSHA, "sess-live", "assistant: mid-session work\n")

	stdout, stderr := runHook(t, hookInputJSON(t, "sess-live", repo, "carry on"), "hook", "prompt-router")

	left, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("the staged transcript survived a live prompt: %+v", left)
	}
	if _, err := os.Stat(staged); err == nil {
		t.Error("the raw staged file is still on disk after a live drain")
	}
	records, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("want the transcript redacted into the store, got %d records", len(records))
	}
	// A UserPromptSubmit hook's stdout is INJECTED INTO THE SESSION'S CONTEXT.
	// The drain's strings are transcript paths and capture errors, which is the
	// last text in the program that should reach a model, so every word of it
	// must be on stderr.
	if strings.Contains(stdout, "history") || strings.Contains(stdout, staged) {
		t.Errorf("the live drain put transcript-derived text on the context channel:\n%s", stdout)
	}
	if !strings.Contains(stderr, "stored 1 staged transcript") {
		t.Errorf("the live drain was silent about what it did:\n%s", stderr)
	}
}

// TestPromptRouterDrainsEvenWhenTheRulesLoaderFails: two unrelated subsystems
// share this hook and neither may switch the other off. A repo whose
// .abcd/rules.json will not parse is a rules problem; leaving its unredacted
// transcripts on disk because of it is not a consequence anyone chose.
func TestPromptRouterDrainsEvenWhenTheRulesLoaderFails(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo, rootSHA := sessionEndRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "rules.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	stageRaw(t, repo, rootSHA, "sess-rulesbroken", "assistant: still worth storing\n")

	_, stderr := runHook(t, hookInputJSON(t, "sess-rulesbroken", repo, "carry on"), "hook", "prompt-router")

	left, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("an unparseable rules.json disabled transcript redaction; %d raw transcript(s) left staged (stderr: %s)",
			len(left), stderr)
	}
}

// TestSessionStartReportsAnotherRepositorysBacklog is the third limb. The
// notice was per-repo and therefore blind to exactly the case that goes wrong:
// the repository nobody opens is the one whose staged files nothing drains, and
// standing in a different checkout could not reveal it.
func TestSessionStartReportsAnotherRepositorysBacklog(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	home := os.Getenv("HOME")
	const quietSHA = "cccccccccccccccccccccccccccccccccccccccc"
	if err := os.MkdirAll(abcdhome.Path(home, "transcripts", quietSHA, "records"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home, "history", quietSHA), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "history", quietSHA, "meta.json"),
		[]byte(`{"root_commit":"`+quietSHA+`","name":"abandoned-project"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stageRaw(t, "", quietSHA, "sess-abandoned", strings.Repeat("raw transcript text\n", 200))

	_, stderr := runHook(t, `{"session_id":"s","hook_event_name":"SessionStart","cwd":`+
		mustJSON(t, repo)+`}`, "hook", "session-start")

	if !strings.Contains(stderr, "abandoned-project") {
		t.Errorf("a session start in one repository said nothing about another repository's raw transcripts:\n%s", stderr)
	}
	if !strings.Contains(stderr, "UNREDACTED") {
		t.Errorf("the cross-repository notice does not say what the bytes are:\n%s", stderr)
	}
	if strings.Contains(stderr, "sess-abandoned") {
		t.Errorf("the cross-repository notice leaked another repository's session id into this session:\n%s", stderr)
	}
	if !strings.Contains(stderr, "--all-repos") {
		t.Errorf("the notice does not name the verb that shows the detail:\n%s", stderr)
	}
}

// mustJSON quotes a string as a JSON scalar for an inline hook payload.
func mustJSON(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestHistoryStagedAllReposSurveysTheWholeStore: the read-only front door onto
// the same fact. It resolves no root SHA, so it answers from anywhere.
func TestHistoryStagedAllReposSurveysTheWholeStore(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	home := os.Getenv("HOME")
	const quietSHA = "dddddddddddddddddddddddddddddddddddddddd"
	if err := os.MkdirAll(abcdhome.Path(home, "transcripts", quietSHA, "records"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home, "history", quietSHA), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "history", quietSHA, "meta.json"),
		[]byte(`{"root_commit":"`+quietSHA+`","name":"the-quiet-one"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stageRaw(t, "", quietSHA, "sess-quiet", "quiet\n")
	stageRaw(t, repo, rootSHA, "sess-here", "here\n")

	out := string(runCLI(t, "history", "staged", "--all-repos"))
	if !strings.Contains(out, "the-quiet-one") {
		t.Errorf("--all-repos missed a repository holding raw transcripts:\n%s", out)
	}
	if !strings.Contains(out, "2 repositor") {
		t.Errorf("--all-repos did not total the store:\n%s", out)
	}
	// The per-repo form must still answer only for this repo, or the two forms
	// are the same verb and the flag means nothing.
	here := string(runCLI(t, "history", "staged"))
	if strings.Contains(here, "the-quiet-one") {
		t.Errorf("the per-repo listing reported another repository:\n%s", here)
	}
}

// TestHistoryDiscardRefusesWithoutConfirmation: this is the only path in abcd
// that destroys a transcript nothing has stored. Core deletes what it is told
// to and never prompts; the human decision belongs to the front door, and a
// front door that did not ask would make the terminal state a shredder.
func TestHistoryDiscardRefusesWithoutConfirmation(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	staged := stageRaw(t, repo, rootSHA, "sess-discard", "raw bytes\n")
	name := filepath.Base(staged)

	out, err := runCLIErr(t, "history", "discard", name)
	if err == nil {
		t.Fatalf("history discard deleted an unredacted transcript with no confirmation:\n%s", out)
	}
	if _, statErr := os.Stat(staged); statErr != nil {
		t.Fatalf("the refused discard removed the file anyway: %v", statErr)
	}
	if !strings.Contains(string(out)+err.Error(), "--yes") {
		t.Errorf("the refusal does not say how to confirm:\n%s\n%v", out, err)
	}

	got := string(runCLI(t, "history", "discard", name, "--yes"))
	if _, statErr := os.Stat(staged); statErr == nil {
		t.Error("the confirmed discard did not remove the transcript")
	}
	if !strings.Contains(got, "unrecoverably") {
		t.Errorf("the discard did not say the deletion was irreversible:\n%s", got)
	}
}

// TestSessionStartNamesAnOrphanedPileAndDiscardByRootSHAEmptiesIt is
// iss-2609091717146700 end to end: a transcript staged under a checkout that
// was then deleted can never be drained (the drain needs that repository's own
// redaction settings), so the next session start names the pile, its size and
// the commands that remove it, and discard by --root-sha, run from a different
// repository with the checkout gone, empties it. The notice still carries no
// session id from the other repository.
func TestSessionStartNamesAnOrphanedPileAndDiscardByRootSHAEmptiesIt(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	home := os.Getenv("HOME")
	const goneSHA = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	gone := filepath.Join(t.TempDir(), "deleted-checkout")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home, "history", goneSHA), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "history", goneSHA, "meta.json"),
		[]byte(`{"root_commit":"`+goneSHA+`","name":"deleted-project"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "history", "index.json"),
		[]byte(`{"schema":1,"repos":[{"root_commit":"`+goneSHA+`","name":"deleted-project","path":`+mustJSON(t, gone)+`,"status":"active"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	first := stageRaw(t, gone, goneSHA, "sess-orphan-1", strings.Repeat("raw transcript text\n", 100))
	second := stageRaw(t, gone, goneSHA, "sess-orphan-2", "more raw text\n")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	_, stderr := runHook(t, `{"session_id":"s","hook_event_name":"SessionStart","cwd":`+
		mustJSON(t, repo)+`}`, "hook", "session-start")
	for _, want := range []string{
		"deleted-project",
		"UNREDACTED",
		"abcd history staged --root-sha " + goneSHA,
		"abcd history discard --root-sha " + goneSHA + " <file> --yes",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the orphaned pile's notice lacks %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "sess-orphan") {
		t.Errorf("the notice leaked the gone repository's session ids:\n%s", stderr)
	}

	t.Chdir(repo)
	listing := string(runCLI(t, "history", "staged", "--root-sha", goneSHA))
	for _, p := range []string{first, second} {
		name := filepath.Base(p)
		if !strings.Contains(listing, name) {
			t.Errorf("history staged --root-sha does not show the file name to discard, %s:\n%s", name, listing)
		}
		if out, err := runCLIErr(t, "history", "discard", "--root-sha", goneSHA, name); err == nil {
			t.Fatalf("discard by root SHA ran without --yes:\n%s", out)
		}
		runCLI(t, "history", "discard", "--root-sha", goneSHA, name, "--yes")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("discard --root-sha left %s in place", name)
		}
	}
	survey := string(runCLI(t, "history", "staged", "--all-repos"))
	if strings.Contains(survey, "deleted-project") {
		t.Errorf("the orphaned pile is still reported after discarding it:\n%s", survey)
	}
}
