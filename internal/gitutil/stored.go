package gitutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// asStored makes an isolated command read history as stored: no
// refs/replace/* ref stands one object in for another
// (GIT_NO_REPLACE_OBJECTS), and no .git/info/grafts file gives a commit
// parents it does not store (NoGraftFile), which git honours even with
// replace refs off.
func asStored(cmd *exec.Cmd) *exec.Cmd {
	cmd.Env = append(cmd.Env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+NoGraftFile)
	return cmd
}

// RunCappedAsStored is RunCapped reading history as stored (asStored): for a
// read whose answer is a commit's parents or what a branch holds, which a
// replace ref or a graft could otherwise change.
func RunCappedAsStored(root string, maxBytes int, args ...string) (string, error) {
	out, overflowed, err := runBoundedCmd(asStored(isolatedGit(root, args...)), maxBytes)
	if err != nil {
		return "", err
	}
	if overflowed {
		return "", fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return strings.TrimRight(string(out), " \t\r\n"), nil
}

// RunCappedExitOneAsStored is RunCappedExitOne reading history as stored
// (asStored).
func RunCappedExitOneAsStored(root string, maxBytes int, args ...string) ([]byte, bool, error) {
	return runCappedExitOneCmd(asStored(isolatedGit(root, args...)), maxBytes, args)
}
