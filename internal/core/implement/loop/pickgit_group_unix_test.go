//go:build unix

package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestAStalledHooksChildrenAreStoppedWithIt is the security review's finding
// on the deadline: SIGTERM reached git alone, so a hook it ran and the
// programs that hook started outlived the stop and went on writing into the
// lane's worktree, which the undo of a stopped sync had just reset. The
// command now runs in its own process group and the stop reaches the whole
// group: a child the hook left sleeping, which would write a file once it
// woke, is gone, and the file never appears.
func TestAStalledHooksChildrenAreStoppedWithIt(t *testing.T) {
	r := stalledCommitRepo(t)
	out := t.TempDir()
	pidFile, late := filepath.Join(out, "child.pid"), filepath.Join(out, "late.txt")
	hook := "#!/bin/sh\n" +
		"sh -c 'echo $$ > \"" + pidFile + "\"; sleep 3; echo late > \"" + late + "\"' &\n" +
		"exec sleep 60\n"
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "hooks", "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	shortCommitTimeout(t, time.Second)

	head := r.Git("rev-parse", "HEAD")
	r.Write("a.md", "picked\n")
	began := time.Now()
	_, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md")
	assertStopped(t, r, head, began, err)

	raw, rerr := os.ReadFile(pidFile)
	if rerr != nil {
		t.Fatalf("the hook's child never started: %v", rerr)
	}
	pid, perr := strconv.Atoi(strings.TrimSpace(string(raw)))
	if perr != nil {
		t.Fatal(perr)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	// Past the moment the child would have written, with room to spare.
	time.Sleep(time.Until(began.Add(5 * time.Second)))
	if _, err := os.Lstat(late); err == nil {
		t.Fatal("the hook's child outlived the stop and wrote after it")
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("the hook's child %d is still running after the stop (kill 0: %v)", pid, err)
	}
}
