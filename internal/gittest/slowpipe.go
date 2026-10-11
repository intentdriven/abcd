package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// SlowPipeGit puts a `git` first on PATH that runs the real one and, for a
// call whose arguments contain match, leaves a background process holding its
// output pipes open for a few seconds after it exits. A caller that waits on
// git with a WaitDelay (gitutil's context-bound runner) then sees
// exec.ErrWaitDelay ("WaitDelay expired before I/O complete") although git
// answered: the same error a heavily loaded machine produces when the copy of
// git's output misses the delay. It is the deterministic stand-in for that
// load, so no test has to generate CPU load to reproduce it.
//
// The first hangs matching calls hold the pipe; later ones answer cleanly. A
// negative hangs holds it on every matching call. The returned function
// reports how many matching calls git has received so far.
func SlowPipeGit(t *testing.T, match string, hangs int) (calls func() int) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	if strings.ContainsAny(match, "'*?[]\\") {
		t.Fatalf("SlowPipeGit: match %q must be a plain word", match)
	}
	dir := t.TempDir()
	count := filepath.Join(t.TempDir(), "calls")
	body := "#!/bin/sh\n" +
		"case \"$*\" in\n" +
		"*'" + match + "'*)\n" +
		"  n=$(cat '" + count + "' 2>/dev/null || echo 0); n=$((n+1)); echo $n > '" + count + "'\n" +
		"  if [ " + strconv.Itoa(hangs) + " -lt 0 ] || [ $n -le " + strconv.Itoa(hangs) + " ]; then\n" +
		"    '" + real + "' \"$@\"; rc=$?\n" +
		"    sleep 3 &\n" +
		"    exit $rc\n" +
		"  fi;;\n" +
		"esac\n" +
		"exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() int {
		raw, err := os.ReadFile(count)
		if err != nil {
			return 0
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
		return n
	}
}
