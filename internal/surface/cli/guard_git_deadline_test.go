package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// guardHookLimit is the timeout hooks/hooks.json gives the PreToolUse hook,
// which runs the guard. The host lets the call run unjudged when it passes.
const guardHookLimit = 240 * time.Second

// pipeIncludedRepo is a git checkout with a .abcd directory whose repository
// configuration includes a named pipe nothing writes to, so every git call in
// it waits on the read until it is killed. The pipe lives outside the checkout
// and its path is returned beside the checkout's.
func pipeIncludedRepo(t *testing.T) (repo, pipe string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	pipe = filepath.Join(realPath(t, t.TempDir()), "config-pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no named pipe here: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(repo, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("[include]\n\tpath = " + pipe + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo, pipe
}

// hookAnswer is one run of the guard hook.
type hookAnswer struct {
	stdout, stderr string
	code           int
}

// runGuardHookWithin runs the guard hook on stdin and fails the test when it
// has not answered within limit. A hook that overruns is then let finish:
// each git call waiting on the pipe is handed an empty read, so no git it
// started is left waiting once the test ends.
func runGuardHookWithin(t *testing.T, stdin, pipe string, limit time.Duration) hookAnswer {
	t.Helper()
	done := make(chan hookAnswer, 1)
	go func() {
		so, se, c := runGuard(stdin, "guard", "hook")
		done <- hookAnswer{so, se, c}
	}()
	select {
	case a := <-done:
		return a
	case <-time.After(limit):
	}
	t.Errorf("the guard hook had not answered %s after it started; it waits on git's read of the pipe", limit)
	for {
		select {
		case a := <-done:
			return a
		default:
		}
		// A reader waiting in its open is released by a writer's open and
		// close; with no reader waiting the open fails and is tried again.
		if f, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = f.Close()
		} else if !errors.Is(err, syscall.ENXIO) {
			t.Fatalf("releasing the pipe's reader: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestTheGuardHookSaysItCouldNotJudgeACommandWhenGitDoesNotAnswer: a shell
// command in a checkout whose configuration git cannot finish reading is
// answered within the hook's git budget, not at the host's timeout. It runs,
// as every call the guard cannot judge does, with exit 1 and a line on stderr
// that names git, the budget and the cause.
func TestTheGuardHookSaysItCouldNotJudgeACommandWhenGitDoesNotAnswer(t *testing.T) {
	repo, pipe := pipeIncludedRepo(t)
	setBound(t, &guardGitBudget, 500*time.Millisecond)

	a := runGuardHookWithin(t, preToolUse(t, "Bash", "git status --porcelain", repo), pipe, 10*time.Second)

	if a.code != 1 || isDeny(a.stdout) {
		t.Errorf("want the loud, non-blocking exit 1 and no deny; got exit %d, stdout %q", a.code, a.stdout)
	}
	for _, want := range []string{"NOT CHECKED", "git did not answer within 500ms", "named pipe", "This command runs UNGUARDED"} {
		if !strings.Contains(a.stderr, want) {
			t.Errorf("stderr must say %q; stderr = %q", want, a.stderr)
		}
	}
}

// TestTheGuardHookSaysItCouldNotJudgeAWorkdirWhoseGitDoesNotAnswer: the same
// holds for a call whose per-call workdir lies in such a checkout while the
// session's own checkout answers.
func TestTheGuardHookSaysItCouldNotJudgeAWorkdirWhoseGitDoesNotAnswer(t *testing.T) {
	stuck, pipe := pipeIncludedRepo(t)
	session := guardRepo(t)
	setBound(t, &guardGitBudget, 500*time.Millisecond)
	b, err := json.Marshal(map[string]any{
		"session_id": "s1", "cwd": session, "hook_event_name": "PreToolUse", "tool_name": "Bash",
		"tool_input": map[string]any{"command": "git status --porcelain", "workdir": stuck},
	})
	if err != nil {
		t.Fatal(err)
	}

	a := runGuardHookWithin(t, string(b), pipe, 10*time.Second)

	if a.code != 1 || isDeny(a.stdout) {
		t.Errorf("want the loud, non-blocking exit 1 and no deny; got exit %d, stdout %q", a.code, a.stdout)
	}
	if !strings.Contains(a.stderr, "git did not answer") || !strings.Contains(a.stderr, "the call's working directory") {
		t.Errorf("stderr must name git and the working directory; stderr = %q", a.stderr)
	}
}

// TestTheGuardHookSaysItCouldNotGateAQuestionWhenGitDoesNotAnswer: a question
// asked in such a checkout runs within the budget too, with the question
// gate's loud warning naming the cause.
func TestTheGuardHookSaysItCouldNotGateAQuestionWhenGitDoesNotAnswer(t *testing.T) {
	repo, pipe := pipeIncludedRepo(t)
	setBound(t, &guardGitBudget, 500*time.Millisecond)

	a := runGuardHookWithin(t, askPayload(t, repo, wellBuilt()), pipe, 10*time.Second)

	if a.code != 1 || isDeny(a.stdout) {
		t.Errorf("want the loud, non-blocking exit 1 and no deny; got exit %d, stdout %q", a.code, a.stdout)
	}
	for _, want := range []string{"NOT CHECKED", "git did not answer within 500ms", "The question runs UNGATED"} {
		if !strings.Contains(a.stderr, want) {
			t.Errorf("stderr must say %q; stderr = %q", want, a.stderr)
		}
	}
}

// TestTheGuardHooksGitBudgetSitsUnderItsTimeout: hooks/hooks.json gives the
// PreToolUse hook guardHookLimit, and the hook's git budget for its two root
// lookups, with the fetched facilitator.mode read a merge command adds, leaves
// most of it, so the hook gives its own answer before the host gives up on it.
func TestTheGuardHooksGitBudgetSitsUnderItsTimeout(t *testing.T) {
	data, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	var doc sessionStartHooks
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	groups := doc.Hooks["PreToolUse"]
	if len(groups) != 1 || len(groups[0].Hooks) != 1 {
		t.Fatalf("hooks.json: want one PreToolUse hook, got %d group(s)", len(groups))
	}
	got := groups[0].Hooks[0].Timeout
	if got == nil {
		t.Fatal("hooks.json gives the PreToolUse hook no timeout, so the host's default applies")
	}
	if limit := time.Duration(*got) * time.Second; limit != guardHookLimit {
		t.Errorf("hooks.json gives the PreToolUse hook %s; the test holds it to %s", limit, guardHookLimit)
	}
	if spent := 2*guardGitBudget + landing.FetchedReadTimeout; 2*spent > guardHookLimit {
		t.Errorf("the hook's bounded reads take up to %s, more than half its %s timeout", spent, guardHookLimit)
	}
}
