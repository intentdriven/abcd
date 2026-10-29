package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// stubProver answers every delete with one proof.
type stubProver struct{ proof guard.DeleteProof }

func (s stubProver) ProveDelete(guard.DeleteTarget) guard.DeleteProof { return s.proof }

// withProver makes p the prover both front doors attach, for this test.
func withProver(t *testing.T, p guard.DeleteProver) {
	t.Helper()
	old := guardDeleteProver
	guardDeleteProver = func() guard.DeleteProver { return p }
	t.Cleanup(func() { guardDeleteProver = old })
}

// allowRepo is a git repository with one commit under a fresh home, as the
// working directory.
func allowRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Commit("one")
	t.Chdir(repo.Root())
	return repo
}

var unprovable = stubProver{guard.DeleteProof{Failed: "its tip holds work origin/main does not", Settle: "Merge it first, or keep it."}}

// TestGuardHookRefusesADeleteItCannotProve: the hook denies a delete whose
// target the prover does not prove safe, naming the check that failed, what
// settles it, and the person's one-command way past it.
func TestGuardHookRefusesADeleteItCannotProve(t *testing.T) {
	repo := allowRepo(t)
	withProver(t, unprovable)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "git branch -D feat/x", repo.Root()), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	for _, want := range []string{"holds work origin/main does not", "Merge it first", "abcd guard allow --once"} {
		if !strings.Contains(reason, want) {
			t.Errorf("the deny does not say %q: %s", want, reason)
		}
	}
}

// TestGuardHookRunsADeleteThePersonAllowedOnce: the person's allow of the
// exact line lets it through the hook once, loudly, and the next run of the
// same line is denied again.
func TestGuardHookRunsADeleteThePersonAllowedOnce(t *testing.T) {
	repo := allowRepo(t)
	withProver(t, unprovable)
	const line = "git push origin --delete feat/x"
	if _, stderr, code := runGuard("", "guard", "allow", "--once", "--command", line); code != 0 {
		t.Fatalf("guard allow: exit %d (%s)", code, stderr)
	}
	_, stderr, code := runGuard(preToolUse(t, "Bash", line, repo.Root()), "guard", "hook")
	if code != 1 || !strings.Contains(stderr, "allowed this exact command once") {
		t.Fatalf("the allowed line: exit %d, stderr %q; want it to run loudly", code, stderr)
	}
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", line, repo.Root()), "guard", "hook")
	mustDeny(t, stdout, stderr, code)
}

// TestGuardHookKeepsEveryOtherBlockerPastAnAllow: an allow passes only the
// unproven delete; a line any other hazard blocks is denied, and its allow is
// left unused.
func TestGuardHookKeepsEveryOtherBlockerPastAnAllow(t *testing.T) {
	repo := allowRepo(t)
	withProver(t, unprovable)
	const line = "git branch -D feat/x && abcd library ledger --flip 3"
	if _, stderr, code := runGuard("", "guard", "allow", "--once", "--command", line); code != 0 {
		t.Fatalf("guard allow: exit %d (%s)", code, stderr)
	}
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", line, repo.Root()), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, "ledger") {
		t.Errorf("the deny is not the ledger flip's: %s", reason)
	}
}

// TestGuardAllowRefusesWithoutOnce: an allow is only ever for one use, so the
// verb refuses without --once, and without the line.
func TestGuardAllowRefusesWithoutOnce(t *testing.T) {
	allowRepo(t)
	if _, _, code := runGuard("", "guard", "allow", "--command", "git branch -D x"); code != 2 {
		t.Errorf("without --once: exit %d, want 2", code)
	}
	if _, _, code := runGuard("", "guard", "allow", "--once"); code != 2 {
		t.Errorf("without --command: exit %d, want 2", code)
	}
}

// TestGuardCheckAndHookProveARealLocalBranch: with the prover the front doors
// attach by default, a local branch merged into origin's default branch is
// let through (with the entry's warn), and one holding work it lacks is
// refused, by the check and the hook alike. No forge is asked for a local
// branch.
func TestGuardCheckAndHookProveARealLocalBranch(t *testing.T) {
	repo := allowRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, origin)
	repo.Git("remote", "add", "origin", origin)
	for _, b := range []string{"done", "open"} {
		repo.Git("switch", "-q", "-c", b)
		repo.Write(b+".txt", b+"\n")
		repo.Git("add", "-A")
		repo.Git("commit", "-q", "-m", b)
		repo.Git("switch", "-q", "main")
	}
	repo.Git("merge", "-q", "--ff-only", "done")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	old := worktreeForge
	worktreeForge = func(string, ...string) (string, error) {
		t.Fatal("a local branch's proof asked the forge")
		return "", nil
	}
	t.Cleanup(func() { worktreeForge = old })

	if _, stderr, code := runGuard("", "guard", "check", "--command", "git branch -D done"); code != 0 {
		t.Errorf("check, merged branch: exit %d (%s), want it let through", code, stderr)
	}
	if _, _, code := runGuard("", "guard", "check", "--command", "git branch -D open"); code != 1 {
		t.Errorf("check, unmerged branch: exit %d, want the block's 1", code)
	}
	if _, stderr, code := runGuard(preToolUse(t, "Bash", "git branch -D done", repo.Root()), "guard", "hook"); code != 1 || !strings.Contains(stderr, "git-branch-force-delete") {
		t.Errorf("hook, merged branch: exit %d, stderr %q; want the entry's warn", code, stderr)
	}
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "git branch -D open", repo.Root()), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "holds work origin/main does not") {
		t.Errorf("hook, unmerged branch: %s", reason)
	}
}
