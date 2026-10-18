package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	gitAt(t, root, "init", "-q", "--bare", origin)
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
		if !strings.Contains(out.SystemMessage, "feat/x") {
			t.Errorf("present, the owed branch is named to the person: got %s", so)
		}
		if calls := ghCalls(t, ghLog); calls != "" {
			t.Errorf("present, the check reads disk only, but the forge was asked: %q", calls)
		}
	})
}
