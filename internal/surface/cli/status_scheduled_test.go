package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// `abcd status`, the prompt hook and a red latest scheduled run
// (iss-2611050035192370), through the forge adapter against a stand-in gh, in a
// repository that names its own workflow.

const (
	scheduledWorkflowName = "nightly.yaml"
	scheduledRunURL       = "https://github.com/example-org/example-app/actions/runs/101"
)

// spawnsRefreshes records each background refresh the hook asks for, and runs
// none, for the rest of the test.
func spawnsRefreshes(t *testing.T) *refreshLog {
	t.Helper()
	l := &refreshLog{}
	old := spawnScheduledRefresh
	spawnScheduledRefresh = func(dir string) error {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.dirs = append(l.dirs, dir)
		return nil
	}
	t.Cleanup(func() { spawnScheduledRefresh = old })
	return l
}

type refreshLog struct {
	mu   sync.Mutex
	dirs []string
}

func (l *refreshLog) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.dirs)
}

// scheduledJSON is the part of --json this names.
type scheduledJSON struct {
	Failure *struct {
		Workflow    string `json:"workflow"`
		Branch      string `json:"branch"`
		Conclusion  string `json:"conclusion"`
		URL         string `json:"url"`
		CompletedAt string `json:"completed_at"`
		AgeSeconds  int64  `json:"age_seconds"`
		RunsSince   int    `json:"runs_since"`
		Text        string `json:"text"`
	} `json:"scheduled_failure"`
	Doubt *struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	} `json:"scheduled_doubt"`
	Unread string `json:"scheduled_unread"`
}

// standInForge puts a gh on PATH that logs each call and answers with what
// answer holds, or fails with its text when failing, and returns the log's path. It
// answers everything else with an empty list, as a forge with nothing to say.
func standInForge(t *testing.T, answer string, failing bool) (log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	body := filepath.Join(dir, "answer")
	if err := os.WriteFile(body, []byte(answer), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>" + shellquote.Single(log) + "\ncase \"$*\" in\n*actions/workflows*) cat " + shellquote.Single(body)
	if failing {
		script += " >&2; exit 1"
	}
	script += " ;;\n*) echo '[]' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// runListing is the forge's answer of one run that ended at the given time.
func runListing(conclusion string, updated time.Time) string {
	return `{"total_count":1,"workflow_runs":[{"id":101,"head_branch":"main","event":"schedule","status":"completed","conclusion":"` +
		conclusion + `","html_url":"` + scheduledRunURL + `","updated_at":"` + updated.UTC().Format(time.RFC3339) + `"}]}`
}

// askGitHubFrom sends the adapter to a checkout whose origin names GitHub: the
// world's origin is a local bare repository, which is how the fetch reads it.
func askGitHubFrom(t *testing.T) {
	t.Helper()
	gh := t.TempDir()
	gitInitAt(t, gh)
	gitCmd(t, gh, "remote", "add", "origin", "https://github.com/example-org/example-app.git")
	real := statusScheduledRun
	statusScheduledRun = func(ctx context.Context, _, workflow, branch string) ([]claim.ScheduledRun, error) {
		return real(ctx, gh, workflow, branch)
	}
	t.Cleanup(func() { statusScheduledRun = real })
}

// nameScheduledWorkflow lands the config naming the workflow on the default
// branch, the tree the alert reads the key from.
func nameScheduledWorkflow(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"status":{"scheduled_workflow":"` + scheduledWorkflowName + `"}}`
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, root, "add", "--", ".abcd/config.json")
	gitCmd(t, root, "commit", "-q", "-m", "name the scheduled workflow")
	gitCmd(t, root, "push", "-q", "origin", "HEAD:refs/heads/main")
}

// ageKeptAnswer dates every answer kept under home d earlier than it was.
func ageKeptAnswer(t *testing.T, home string, d time.Duration) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(home, ".abcd.noindex", "scheduled-runs", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no answer kept under %s: %v", home, err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var kept map[string]any
		if err := json.Unmarshal(b, &kept); err != nil {
			t.Fatal(err)
		}
		at, ok := kept["checked_at"].(string)
		if !ok {
			continue
		}
		when, err := time.Parse(time.RFC3339, at)
		if err != nil {
			t.Fatal(err)
		}
		kept["checked_at"] = when.Add(-d).UTC().Format(time.RFC3339)
		out, err := json.Marshal(kept)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// readCalls is the stand-in forge's log.
func readCalls(t *testing.T, log string) string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

// hangOriginKeepingItsName is hangOrigin by a rewrite of origin's URL: the name
// the checkout has configured for origin keys the store, so it stays, and any
// network call to it hangs and leaves the file it returns.
func hangOriginKeepingItsName(t *testing.T, root, origin string) (touched string) {
	t.Helper()
	touched = filepath.Join(t.TempDir(), "origin-reached")
	if strings.ContainsAny(touched, " %") {
		t.Fatalf("the marker path %q needs escaping in an ext:: URL", touched)
	}
	gitCmd(t, root, "config", "protocol.ext.allow", "always")
	gitCmd(t, root, "config", "url.ext::sh -c touch% "+touched+";% sleep% 30.insteadOf", origin)
	return touched
}

// TestStatusNamesAFailedLatestScheduledRun: the verb asks the forge for the
// workflow the repository names, on its default branch, and says it in one
// line with the URL and the age; the exit stays the fetch's.
func TestStatusNamesAFailedLatestScheduledRun(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	log := standInForge(t, runListing("failure", time.Now().Add(-3*time.Hour)), false)

	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	want := "the forge's latest scheduled " + scheduledWorkflowName + " run on the default branch failed 3 hours ago: " + scheduledRunURL
	if !strings.Contains(stdout, "  "+want+"\n") {
		t.Fatalf("status:\n%s\nwant the line %q", stdout, want)
	}
	calls := readCalls(t, log)
	for _, part := range []string{"repos/example-org/example-app/actions/workflows/" + scheduledWorkflowName + "/runs?", "branch=main", "event=schedule", "status=completed"} {
		if !strings.Contains(calls, part) {
			t.Errorf("the forge was asked:\n%s\nwant it to carry %q", calls, part)
		}
	}

	code, stdout, stderr = reserveRun(t, pat, "status", "--json")
	if code != 0 {
		t.Fatalf("status --json: exit %d, %s", code, stderr)
	}
	var got scheduledJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("--json: %v\n%s", err, stdout)
	}
	if got.Failure == nil {
		t.Fatalf("--json names no failure:\n%s", stdout)
	}
	f := got.Failure
	if f.Workflow != scheduledWorkflowName || f.Branch != "main" || f.Conclusion != "failure" || f.URL != scheduledRunURL ||
		f.CompletedAt == "" || f.AgeSeconds < 3*3600 || f.Text != want {
		t.Fatalf("--json's failure = %+v", *f)
	}
	if got.Unread != "" || got.Doubt != nil {
		t.Fatalf("--json: scheduled_unread %q, scheduled_doubt %+v", got.Unread, got.Doubt)
	}
}

// TestStatusIsSilentOnAPassingLatestRun: asked and answered, with nothing
// said, in either form. The stand-in's log is the proof it was asked.
func TestStatusIsSilentOnAPassingLatestRun(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	log := standInForge(t, runListing("success", time.Now().Add(-time.Hour)), false)

	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	if !strings.Contains(readCalls(t, log), "actions/workflows") {
		t.Fatal("the forge was not asked")
	}
	if strings.Contains(stdout, scheduledWorkflowName) || strings.Contains(stdout, scheduledRunURL) {
		t.Fatalf("a passing run is spoken of:\n%s", stdout)
	}
	_, stdout, _ = reserveRun(t, pat, "status", "--json")
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"scheduled_failure", "scheduled_doubt", "scheduled_unread"} {
		if _, ok := got[k]; ok {
			t.Errorf("--json carries %s on a passing run: %v", k, got[k])
		}
	}
}

// TestStatusSaysANoVerdictNightly: a latest run that reached no verdict over a
// green older than a few days is one line, not a failure.
func TestStatusSaysANoVerdictNightly(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	standInForge(t, runListing("cancelled", time.Now().Add(-time.Hour)), false)
	code, stdout, stderr := reserveRun(t, pat, "status", "--json")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	var got scheduledJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	if got.Failure != nil || got.Doubt == nil || got.Doubt.Kind != claim.ScheduledEndedWithoutVerdict {
		t.Fatalf("--json: failure %+v, doubt %+v; want no failure and the run's own line", got.Failure, got.Doubt)
	}
	_, text, _ := reserveRun(t, pat, "status")
	if !strings.Contains(text, "  "+got.Doubt.Text+"\n") {
		t.Fatalf("status:\n%s\nwant the line %q", text, got.Doubt.Text)
	}
}

// TestStatusAsksNothingWhereNoWorkflowIsNamed: a repository that names none
// costs no call to the forge and shows nothing. The same world, with the key,
// asks: the control that the absence is the key's.
func TestStatusAsksNothingWhereNoWorkflowIsNamed(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	askGitHubFrom(t)
	log := standInForge(t, runListing("failure", time.Now()), false)

	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	if calls := readCalls(t, log); strings.Contains(calls, "actions/workflows") {
		t.Fatalf("the forge was asked with no workflow named:\n%s", calls)
	}
	if strings.Contains(stdout, "scheduled") {
		t.Fatalf("status speaks of a scheduled run:\n%s", stdout)
	}
	nameScheduledWorkflow(t, pat)
	if code, stdout, _ = reserveRun(t, pat, "status"); code != 0 || !strings.Contains(stdout, scheduledRunURL) {
		t.Fatalf("with the key named: exit %d\n%s", code, stdout)
	}
}

// TestStatusWithNoForgeSaysUnknownAndNothingAsAFailure: a gh that fails, and an
// origin that is not on GitHub at all, leave the exit as the fetch made it, no
// failed run named, one line in the verb saying why the run was not read, and a
// block that says the state is unknown rather than read like a green.
func TestStatusWithNoForgeSaysUnknownAndNothingAsAFailure(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T){
		"gh fails": func(t *testing.T) {
			askGitHubFrom(t)
			standInForge(t, "HTTP 401: Bad credentials", true)
		},
		"origin off the forge": func(t *testing.T) {
			standInForge(t, runListing("failure", time.Now()), false)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ownHookBudgets(t, 10*time.Second)
			t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
			t.Cleanup(claim.WaitPromptReadsForTest)
			spawnsRefreshes(t)
			w := newReserveWorld(t)
			pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
			nameScheduledWorkflow(t, pat)
			setup(t)
			code, stdout, stderr := reserveRun(t, pat, "status")
			if code != 0 {
				t.Fatalf("status: exit %d, %s", code, stderr)
			}
			if strings.Contains(stdout, " failed ") || strings.Contains(stdout, scheduledRunURL) {
				t.Fatalf("an unread answer reads as a failure:\n%s", stdout)
			}
			if !strings.Contains(stdout, "the latest scheduled "+scheduledWorkflowName+" run was not read") {
				t.Fatalf("status:\n%s\nwant a line saying the run was not read", stdout)
			}
			_, stdout, _ = reserveRun(t, pat, "status", "--json")
			var got scheduledJSON
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatal(err)
			}
			if got.Failure != nil || got.Unread == "" || got.Doubt == nil || got.Doubt.Kind != claim.ScheduledUnknown {
				t.Fatalf("--json = %+v; want no failure, the reason and an unknown doubt", got)
			}
			out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello"), "hook", "prompt-router")
			if !strings.Contains(out, "the state of scheduled "+scheduledWorkflowName+" is unknown") || strings.Contains(out, " failed ") {
				t.Fatalf("the block:\n%s\nwant it to say the state is unknown", out)
			}
		})
	}
}

// TestAFailedFetchAsksTheForgeNothing: the verb asks only when it fetched, so
// a machine with no network pays for one failure, not two; its exit and line
// are the fetch's as before.
func TestAFailedFetchAsksTheForgeNothing(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	log := standInForge(t, runListing("failure", time.Now()), false)
	// The positive control: the same world asks while its fetch succeeds.
	if code, stdout, _ := reserveRun(t, pat, "status"); code != 0 || !strings.Contains(stdout, scheduledRunURL) {
		t.Fatalf("with a fetch that succeeds: exit %d\n%s", code, stdout)
	}
	before := readCalls(t, log)
	gitCmd(t, pat, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	if code, _, _ := reserveRun(t, pat, "status"); code != 1 {
		t.Fatalf("a failed fetch exits %d; want 1", code)
	}
	if after := readCalls(t, log); strings.Count(after, "actions/workflows") != strings.Count(before, "actions/workflows") {
		t.Fatalf("the forge was asked after a failed fetch:\n%s", after)
	}
}

// TestThePromptBlockReadsTheFailureFromDiskAlone: the hook names the failure
// the verb kept, once per session, without reaching for origin or for gh, and a
// new session is told again.
func TestThePromptBlockReadsTheFailureFromDiskAlone(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	t.Cleanup(claim.WaitPromptReadsForTest)
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	log := standInForge(t, runListing("failure", time.Now().Add(-3*time.Hour)), false)
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	calls := readCalls(t, log)
	touched := hangOriginKeepingItsName(t, pat, w.origin)

	out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello there"), "hook", "prompt-router")
	want := "- the forge's latest scheduled " + scheduledWorkflowName + " run on the default branch failed 3 hours ago: " + scheduledRunURL + "\n"
	if !strings.Contains(out, want) {
		t.Fatalf("the hook's text:\n%s\nwant it to carry %q", out, want)
	}
	again, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello again"), "hook", "prompt-router")
	if strings.Contains(again, scheduledRunURL) {
		t.Fatalf("the session's second prompt names it again:\n%s", again)
	}
	next, _ := runHook(t, hookInputJSON(t, "s-another", pat, "hello from a new session"), "hook", "prompt-router")
	if !strings.Contains(next, want) {
		t.Fatalf("a new session's first prompt:\n%s\nwant it to carry %q", next, want)
	}
	requireOriginUntouched(t, touched)
	if after := readCalls(t, log); after != calls {
		t.Fatalf("the prompt hook asked the forge:\n%s", strings.TrimPrefix(after, calls))
	}
}

// TestThePromptHookStartsARefreshWhenOneIsDue: a session that never runs `abcd
// status` still gets the answer: its prompt, with none kept, says the state is
// unknown and starts one refresh in the background, asking the forge nothing
// itself, and the next prompt inside the cooldown starts no other.
func TestThePromptHookStartsARefreshWhenOneIsDue(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	t.Cleanup(claim.WaitPromptReadsForTest)
	nameScheduledWorkflow(t, pat)
	log := standInForge(t, runListing("failure", time.Now()), false)
	spawned := spawnsRefreshes(t)
	t.Setenv("HOME", w.home)

	out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello there"), "hook", "prompt-router")
	if !strings.Contains(out, "is unknown") {
		t.Fatalf("the hook's text:\n%s\nwant it to say the state is unknown", out)
	}
	if spawned.count() != 1 {
		t.Fatalf("the prompt started %d refreshes; want 1", spawned.count())
	}
	if calls := readCalls(t, log); strings.Contains(calls, "actions/workflows") {
		t.Fatalf("the prompt hook asked the forge itself:\n%s", calls)
	}
	runHook(t, hookInputJSON(t, "s-sched", pat, "hello again"), "hook", "prompt-router")
	if spawned.count() != 1 {
		t.Fatalf("after a second prompt, %d refreshes; want still 1", spawned.count())
	}
}

// TestTheRefreshVerbTellsASessionThatNeverRanStatus: what the prompt starts in
// the background keeps the answer, and a new worktree of the same repository
// that has run nothing is told of the red at its first prompt.
func TestTheRefreshVerbTellsASessionThatNeverRanStatus(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	t.Cleanup(claim.WaitPromptReadsForTest)
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	log := standInForge(t, runListing("failure", time.Now().Add(-3*time.Hour)), false)
	spawnsRefreshes(t)

	code, _, stderr := reserveRun(t, pat, "hook", "scheduled-refresh")
	if code != 0 {
		t.Fatalf("hook scheduled-refresh: exit %d, %s", code, stderr)
	}
	if !strings.Contains(readCalls(t, log), "actions/workflows") {
		t.Fatal("the refresh did not ask the forge")
	}
	lane := filepath.Join(t.TempDir(), "lane")
	gitCmd(t, pat, "worktree", "add", "-q", "-b", "lane", lane)
	if h, ok := cloneHomes[pat]; ok {
		cloneHomes[lane] = h
	}
	if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, _ := runHook(t, hookInputJSON(t, "s-lane", lane, "hello"), "hook", "prompt-router")
	want := "the forge's latest scheduled " + scheduledWorkflowName + " run on the default branch failed 3 hours ago: " + scheduledRunURL
	if !strings.Contains(out, want) {
		t.Fatalf("a fresh worktree's first prompt:\n%s\nwant it to carry %q", out, want)
	}
}

// TestTheRefreshVerbFailsOpen: a forge that fails is said on stderr and the
// hook verb still exits 0: nothing waits on it.
func TestTheRefreshVerbFailsOpen(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	standInForge(t, "HTTP 401: Bad credentials", true)
	code, _, stderr := reserveRun(t, pat, "hook", "scheduled-refresh")
	if code != 0 || !strings.Contains(stderr, "was not read") {
		t.Fatalf("hook scheduled-refresh: exit %d, stderr %q; want 0 and the reason", code, stderr)
	}
}

// TestAStaleRedStaysInTheVerbsTextWhenTheAskFails: the forge cannot be asked and
// the kept red is old: the text must still carry that red, its URL and why this
// ask failed, since a person reads this view to find out.
func TestAStaleRedStaysInTheVerbsTextWhenTheAskFails(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	standInForge(t, runListing("failure", time.Now().Add(-3*time.Hour)), false)
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	ageKeptAnswer(t, os.Getenv("HOME"), 72*time.Hour)
	standInForge(t, "HTTP 401: Bad credentials", true)
	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	for _, want := range []string{scheduledRunURL, "it had failed", "run was not read"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("status:\n%s\nwant it to carry %q", stdout, want)
		}
	}
}

// TestAFailedRefreshReachesThePrompt: a kept green, a refresh in the background that
// fails, whose stderr nobody reads, and the prompt says the state is unknown,
// naming the last successful ask, rather than read as healthy.
func TestAFailedRefreshReachesThePrompt(t *testing.T) {
	ownHookBudgets(t, 10*time.Second)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	t.Cleanup(claim.WaitPromptReadsForTest)
	nameScheduledWorkflow(t, pat)
	askGitHubFrom(t)
	spawnsRefreshes(t)
	standInForge(t, runListing("success", time.Now().Add(-time.Hour)), false)
	if code, _, stderr := reserveRun(t, pat, "hook", "scheduled-refresh"); code != 0 {
		t.Fatalf("the first refresh: exit %d, %s", code, stderr)
	}
	ageKeptAnswer(t, os.Getenv("HOME"), 2*time.Hour)
	standInForge(t, "HTTP 401: Bad credentials", true)
	if code, _, stderr := reserveRun(t, pat, "hook", "scheduled-refresh"); code != 0 {
		t.Fatalf("the failing refresh: exit %d, %s", code, stderr)
	}
	out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello"), "hook", "prompt-router")
	for _, want := range []string{"the state of scheduled " + scheduledWorkflowName + " is unknown", "the last successful ask was 2 hours ago"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the prompt:\n%s\nwant it to carry %q", out, want)
		}
	}
	if strings.Contains(out, "Bad credentials") {
		t.Fatalf("the prompt carries the forge's words:\n%s", out)
	}
}

// TestThePromptDoesNotWalkHistoryAgainForTheStore: the store is keyed on the root
// commit the prompt already looked up once: what the hook keeps sits under that
// answer, so no second full-history walk is made for it.
func TestThePromptDoesNotWalkHistoryAgainForTheStore(t *testing.T) {
	const handed = "abcdef0123456789abcdef0123456789abcdef01"
	ownHookBudgets(t, 10*time.Second)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	t.Cleanup(claim.WaitPromptReadsForTest)
	nameScheduledWorkflow(t, pat)
	spawned := spawnsRefreshes(t)
	t.Setenv("HOME", w.home)
	prev := routerRootCommit
	t.Cleanup(func() { routerRootCommit = prev })
	routerRootCommit = func(context.Context, string) (string, error) { return handed, nil }
	runHook(t, hookInputJSON(t, "s-sched", pat, "hello"), "hook", "prompt-router")
	if spawned.count() != 1 {
		t.Fatalf("the prompt started %d refreshes; want 1", spawned.count())
	}
	stamps, err := filepath.Glob(filepath.Join(w.home, ".abcd.noindex", "scheduled-runs", handed, "*.attempt"))
	if err != nil || len(stamps) != 1 {
		t.Fatalf("the stamp is not under the root commit the prompt looked up: %v, %v", stamps, err)
	}
}
