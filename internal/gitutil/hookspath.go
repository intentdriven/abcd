package gitutil

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// maxHooksPathBytes bounds the core.hooksPath read: a handful of paths.
const maxHooksPathBytes = 64 << 10

// HooksPaths is every core.hooksPath value git reads for the repository at
// root, from each scope it reads (system, global, the repository's, the
// worktree's), in that order and path-expanded (a leading ~ is the home), as
// git writes them: a relative value is relative to the working tree's root.
//
// It is read as the person's own git reads it, so it runs under ScrubbedEnv
// (global and system config in force) and never under the isolated
// environment, whose -c core.hooksPath=/dev/null would answer for itself.
// Reading configuration runs no hook and starts no fsmonitor. None set is an
// empty answer; a git that cannot answer is an error.
func HooksPaths(root string) ([]string, error) {
	// -z ends each value with a NUL rather than a newline: a value may hold a
	// newline, and is one path all the same.
	cmd := exec.Command("git", "-C", root, "config", "-z", "--type=path", "--get-all", "core.hooksPath")
	cmd.Env = ScrubbedEnv()
	w := &capWriter{remaining: maxHooksPathBytes}
	e := &capWriter{remaining: 4096}
	cmd.Stdout, cmd.Stderr = w, e
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 && len(w.buf) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("git config core.hooksPath: %w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	if w.overflowed {
		return nil, fmt.Errorf("git config core.hooksPath: output exceeded the %d-byte cap", maxHooksPathBytes)
	}
	var out []string
	for _, v := range strings.Split(string(w.buf), "\x00") {
		if v != "" {
			out = append(out, v)
		}
	}
	return out, nil
}
