package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// `abcd status` and a failed latest scheduled run (iss-2611050035192370),
// through the forge adapter against a stand-in gh, in a repository that names
// its own workflow.

const (
	scheduledWorkflowName = "nightly.yaml"
	scheduledRunURL       = "https://github.com/example-org/example-app/actions/runs/101"
)

// scheduledJSON is the part of --json this names.
type scheduledJSON struct {
	Failure *struct {
		Workflow    string `json:"workflow"`
		Branch      string `json:"branch"`
		Conclusion  string `json:"conclusion"`
		URL         string `json:"url"`
		CompletedAt string `json:"completed_at"`
		AgeSeconds  int64  `json:"age_seconds"`
		Text        string `json:"text"`
	} `json:"scheduled_failure"`
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
	statusScheduledRun = func(ctx context.Context, _, workflow, branch string) (*claim.ScheduledRun, error) {
		return real(ctx, gh, workflow, branch)
	}
	t.Cleanup(func() { statusScheduledRun = real })
}

// nameScheduledWorkflow writes the checkout's config naming the workflow.
func nameScheduledWorkflow(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"status":{"scheduled_workflow":"` + scheduledWorkflowName + `"}}`
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
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
	want := "the forge's latest scheduled " + scheduledWorkflowName + " run on main failed 3 hours ago: " + scheduledRunURL
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
	if got.Unread != "" {
		t.Fatalf("--json's scheduled_unread = %q", got.Unread)
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
	for _, k := range []string{"scheduled_failure", "scheduled_unread"} {
		if _, ok := got[k]; ok {
			t.Errorf("--json carries %s on a passing run: %v", k, got[k])
		}
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

// TestStatusWithNoForgeSaysItCouldNotReadAndNothingAsAFailure: a gh that fails,
// and an origin that is not on GitHub at all, leave the exit as the fetch made
// it, no failed run named, and one line saying the run was not read, in the
// verb alone: the prompt's block stays silent.
func TestStatusWithNoForgeSaysItCouldNotReadAndNothingAsAFailure(t *testing.T) {
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
			if got.Failure != nil || got.Unread == "" {
				t.Fatalf("--json = %+v; want no failure and the reason", got)
			}
			out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello"), "hook", "prompt-router")
			if strings.Contains(out, scheduledWorkflowName) {
				t.Fatalf("the block speaks of an unread answer:\n%s", out)
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
// the verb kept, once, without reaching for origin or for gh.
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
	touched := hangOrigin(t, pat)

	out, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello there"), "hook", "prompt-router")
	want := "- the forge's latest scheduled " + scheduledWorkflowName + " run on main failed 3 hours ago: " + scheduledRunURL + "\n"
	if !strings.Contains(out, want) {
		t.Fatalf("the hook's text:\n%s\nwant it to carry %q", out, want)
	}
	again, _ := runHook(t, hookInputJSON(t, "s-sched", pat, "hello again"), "hook", "prompt-router")
	if strings.Contains(again, scheduledRunURL) {
		t.Fatalf("the second prompt names it again:\n%s", again)
	}
	requireOriginUntouched(t, touched)
	if after := readCalls(t, log); after != calls {
		t.Fatalf("the prompt hook asked the forge:\n%s", strings.TrimPrefix(after, calls))
	}
}
