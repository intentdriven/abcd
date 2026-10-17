package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/shellquote"
)

// ShimGit puts a git first on PATH, for the length of t, that answers one
// kind of command itself: a git run with every word of when among its
// arguments prints out to stdout and exits code, and every other git command
// runs the git that was first on PATH before. Shims stack: a later one falls
// through to an earlier one.
//
// It lets a test hand the code under test an answer real git never gives (a
// listing that stops partway, a line of an unexpected shape, a failure of one
// command) while every other git command behaves as git does. The fixture's
// own Repo.Git keeps the environment it was made with, so it is not shimmed
// when the Repo was made first. It cannot be used with t.Parallel.
func ShimGit(t *testing.T, when []string, out string, code int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the git shim is a POSIX shell script")
	}
	next, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	answer := filepath.Join(dir, "answer")
	if err := os.WriteFile(answer, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("#!/bin/sh\nhas() { w=$1; shift; for a; do [ \"$a\" = \"$w\" ] && return 0; done; return 1; }\nif true")
	for _, w := range when {
		b.WriteString(" && has " + shellquote.Single(w) + " \"$@\"")
	}
	b.WriteString("; then cat " + shellquote.Single(answer) + "; exit " + strconv.Itoa(code) + "; fi\n")
	b.WriteString("exec " + shellquote.Single(next) + " \"$@\"\n")
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(b.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// ShimGitNamingNoGitfileTarget puts a git first on PATH, for the length of t,
// that runs the git first on PATH before and reports a broken git file as
// Linux's git does: its "fatal: not a git repository: <path>" names the path
// as "(null)", where macOS's git names where the git file points. A reason
// that names that target only through git's stderr names it on one platform;
// under this shim a test sees what every Linux machine, CI's included, sees.
// It cannot be used with t.Parallel.
func ShimGitNamingNoGitfileTarget(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the git shim is a POSIX shell script")
	}
	next, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	body := "#!/bin/sh\n" +
		"err=$(mktemp " + shellquote.Single(filepath.Join(dir, "stderr.XXXXXX")) + ") || exit 70\n" +
		shellquote.Single(next) + " \"$@\" 2>\"$err\"; rc=$?\n" +
		"sed 's/not a git repository: .*/not a git repository: (null)/' \"$err\" >&2\n" +
		"rm -f \"$err\"\n" +
		"exit $rc\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
