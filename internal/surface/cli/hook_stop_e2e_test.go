package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// hook_stop_e2e_test.go — the away mode's stop check end to end, through the
// real owed reader (spc-2610180914469489 steps 4 and 5 joined at the seam):
// real git in a temporary checkout with a bare origin, and a stub forge client
// first on PATH, so nothing reaches a real forge.

// ownedUnpushedCheckout is a managed checkout with a bare origin whose default
// branch is fetched, and an own branch, feat/x, checked out in this copy and
// back (so its HEAD reflog holds it), pushed once and then given a commit of
// this checkout's author that is not pushed. A stub gh answers the away
// listing with no open pull request and logs every call to the returned file.
func ownedUnpushedCheckout(t *testing.T) (root, ghLog string) {
	t.Helper()
	root = awayCheckout(t)
	gitAt(t, root, "config", "--local", "user.email", "alice@example.invalid")
	gitAt(t, root, "config", "--local", "commit.gpgsign", "false")
	origin := filepath.Join(t.TempDir(), "origin.git")
	initBare(t, origin)
	gitAt(t, root, "remote", "add", "origin", origin)
	main := gitAt(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	gitAt(t, root, "push", "-q", "origin", main)
	gitAt(t, root, "remote", "set-head", "origin", main)
	gitAt(t, root, "checkout", "-q", "-b", "feat/x")
	gitAt(t, root, "commit", "-q", "--allow-empty", "-m", "the first half")
	gitAt(t, root, "push", "-q", "-u", "origin", "feat/x")
	gitAt(t, root, "commit", "-q", "--allow-empty", "-m", "the second half, not pushed")
	gitAt(t, root, "checkout", "-q", main)

	bin := t.TempDir()
	ghLog = filepath.Join(t.TempDir(), "gh-calls.log")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> '" + ghLog + "'\n" +
		"if [ \"$1 $2\" = 'pr list' ]; then echo '[]'; exit 0; fi\n" +
		"echo \"gh stub: unexpected call: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root, ghLog
}

func ghCalls(t *testing.T, log string) string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

// TestHookStopEndToEndOwnUnpushedBranch: while the away record holds, an own
// branch with commits not pushed blocks the stop, naming the branch and the
// push sentence that defers to the person's standing instruction, and the
// forge is asked for the open pull requests. With no away record the same
// checkout's stop is allowed, the owed branch named to the person, and the
// forge is never asked.
func TestHookStopEndToEndOwnUnpushedBranch(t *testing.T) {
	t.Run("away blocks", func(t *testing.T) {
		root, ghLog := ownedUnpushedCheckout(t)
		setAway(t)
		out, so, se := runStop(t, stopHookPayload(t, "e2e-away", root, false, "Done for now."))
		if out.Decision != "block" {
			t.Fatalf("away with an own unpushed branch: want a block, got %s\nstderr: %s", so, se)
		}
		for _, want := range []string{
			"feat/x",
			"push if your standing instruction allows",
			"queue it with `abcd mode queue",
		} {
			if !strings.Contains(out.Reason, want) {
				t.Errorf("the block's reason lacks %q:\n%s", want, out.Reason)
			}
		}
		if calls := ghCalls(t, ghLog); !strings.Contains(calls, "pr list") {
			t.Errorf("away, the forge was not asked for the open pull requests; calls: %q", calls)
		}
	})
	t.Run("present allows", func(t *testing.T) {
		root, ghLog := ownedUnpushedCheckout(t)
		out, so, se := runStop(t, stopHookPayload(t, "e2e-present", root, false, "Done for now."))
		if out.Decision == "block" {
			t.Fatalf("with no away record the stop is allowed, got %s\nstderr: %s", so, se)
		}
		if want := "abcd: stopped with work owed (1 item); next: branch feat/x has commits not pushed."; out.SystemMessage != want {
			t.Errorf("present, the owed branch is named to the person once:\n got %s\nwant %q", so, want)
		}
		if calls := ghCalls(t, ghLog); calls != "" {
			t.Errorf("present, the check reads disk only, but the forge was asked: %q", calls)
		}
	})
}

// hostIn is the table's reading of pid as a session's host, or false when the
// table does not hold it.
func hostIn(tab sessionname.Table, pid int) (sessionname.Host, bool) {
	for _, p := range tab.Procs {
		if p.PID == pid {
			return sessionname.Host{PID: pid, StartUnix: tab.ReadAt.Add(-p.Age).Unix(), Slack: tab.Took}, true
		}
	}
	return sessionname.Host{}, false
}

// TestHookStopEndToEndTwoSessionsOneCopy (iss-2611020719483054,
// iss-2610311406481214, iss-2611070830056851): two live sessions on one
// account started in one copy; one cuts a branch through the guard and leaves
// a commit on it unpushed. Away, with a wake scheduled, the session that cut
// it is told, its stop blocked naming the branch; the other is released, the
// branch named to it as the holder's. Once the holder's session has ended,
// the branch is owed to the session left. The real owed reader, the real
// guard recorder, the real session markers and the machine's process table.
func TestHookStopEndToEndTwoSessionsOneCopy(t *testing.T) {
	root := awayCheckout(t)
	gitAt(t, root, "config", "--local", "user.email", "alice@example.invalid")
	gitAt(t, root, "config", "--local", "commit.gpgsign", "false")
	origin := filepath.Join(t.TempDir(), "origin.git")
	initBare(t, origin)
	gitAt(t, root, "remote", "add", "origin", origin)
	main := gitAt(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	gitAt(t, root, "push", "-q", "origin", main)
	gitAt(t, root, "remote", "set-head", "origin", main)
	bin := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1 $2\" = 'pr list' ]; then echo '[]'; exit 0; fi\necho \"gh stub: unexpected call: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	const holder, other = "sess-holder", "sess-other"
	tab, err := sessionname.Live().ReadTable()
	if err != nil {
		t.Skipf("the process table cannot be read here: %v", err)
	}
	holderHost, ok1 := hostIn(tab, os.Getpid())
	otherHost, ok2 := hostIn(tab, os.Getppid())
	if !ok1 || !ok2 {
		t.Skip("this process or its parent is not in the process table")
	}
	home := os.Getenv("HOME")
	if err := occupancy.Note(home, holder, root, holderHost, tab); err != nil {
		t.Fatal(err)
	}
	if err := occupancy.Note(home, other, root, otherHost, tab); err != nil {
		t.Fatal(err)
	}

	held := filepath.Join(t.TempDir(), "held")
	cut := "git worktree add -b feat/held " + held
	payload, err := json.Marshal(map[string]any{
		"session_id": holder, "cwd": root, "hook_event_name": "PreToolUse", "tool_name": "Bash",
		"tool_input": map[string]any{"command": cut, "description": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stdout, stderr, code := runGuard(string(payload), "guard", "hook"); strings.Contains(stdout, `"deny"`) || code == 2 {
		t.Fatalf("%q was refused: %q %q", cut, stdout, stderr)
	}
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/held", held)
	// A commit that changes a file: one that changes nothing has already
	// landed wherever its branch is merged, and is owed nowhere.
	if err := os.WriteFile(filepath.Join(held, "held.txt"), []byte("the holder's work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, held, "add", "held.txt")
	gitAt(t, held, "commit", "-q", "-m", "the holder's work, not pushed")
	setAway(t)

	wake := "*/10 * * * *"
	out, so, se := runStop(t, stopHookPayload(t, holder, root, false, "Done for now.", wake))
	if out.Decision != "block" || !strings.Contains(out.Reason, "feat/held") {
		t.Fatalf("the session that cut feat/held is not told of it: %s\nstderr: %s", so, se)
	}
	out, so, se = runStop(t, stopHookPayload(t, other, root, false, "Done for now.", wake))
	if out.Decision == "block" {
		t.Fatalf("the other session is held for the holder's branch: %s\nstderr: %s", so, se)
	}

	if err := occupancy.Forget(home, holder); err != nil {
		t.Fatal(err)
	}
	out, so, se = runStop(t, stopHookPayload(t, other, root, false, "Done for now.", wake))
	if out.Decision != "block" || !strings.Contains(out.Reason, "feat/held") {
		t.Fatalf("once the holder has ended, its branch is not owed to the session left: %s\nstderr: %s", so, se)
	}
}
