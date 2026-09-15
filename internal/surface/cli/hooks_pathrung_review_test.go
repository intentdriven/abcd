package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinaryHooksNeverRunAnLsFromThePathTheyAudit pins the shim's one rule
// about the PATH it is judging: nothing on that PATH may run before the
// verdict. A repository that ships a `bin/ls` and adds the directory to PATH
// would otherwise execute it on every prompt, whether or not its abcd is
// accepted.
func TestBinaryHooksNeverRunAnLsFromThePathTheyAudit(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			home := t.TempDir()
			pathStub(t, pathDir)
			// The stub is this machine's recorded abcd, so the rung reaches its
			// accept branch — which is where an audited-PATH `ls` would run.
			writeHookPathEntry(t, home, filepath.Join(pathDir, "abcd"))
			marker := filepath.Join(pathDir, "ls-ran")
			fakeLs := "#!/bin/sh\n: > \"" + marker + "\"\nexec /bin/ls \"$@\"\n"
			if err := os.WriteFile(filepath.Join(pathDir, "ls"), []byte(fakeLs), 0o755); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := hookRunHome(t, h.event, root, pathDir, t.TempDir(), home)
			if _, err := os.Stat(marker); err == nil {
				t.Fatalf("the %s shim ran the ls found on the PATH it was auditing (stderr %q, code %d)", h.event, stderr, code)
			}
			calls, _ := os.ReadFile(filepath.Join(root, "calls.log"))
			if !strings.Contains(string(calls), h.verb) {
				t.Fatalf("the accepted PATH abcd did not run %s: calls %q, stderr %q, code %d", h.verb, calls, stderr, code)
			}
		})
	}
}

// TestBinaryHooksSanitiseANewlineInAnIgnoredPath pins that the path the shim
// prints about an ignored binary cannot forge a second stderr line: a
// directory name carrying a newline is flattened before it is printed, the
// way termsafe.Sanitize masks a newline on every other report line.
func TestBinaryHooksSanitiseANewlineInAnIgnoredPath(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := filepath.Join(t.TempDir(), "ev\nil-forged line")
			if err := os.MkdirAll(pathDir, 0o777); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(pathDir, 0o777); err != nil {
				t.Fatal(err)
			}
			pathStub(t, pathDir)
			_, stderr, _ := hookRunIn(t, h.event, root, pathDir, t.TempDir())
			ignoring := 0
			for _, line := range strings.Split(stderr, "\n") {
				if strings.HasPrefix(line, "abcd: ignoring") {
					ignoring++
				}
				if strings.HasPrefix(line, "il-forged line") {
					t.Fatalf("a newline in the ignored path forged a stderr line: %q", stderr)
				}
			}
			if ignoring != 1 {
				t.Fatalf("want exactly one ignoring line, got %d in %q", ignoring, stderr)
			}
		})
	}
}
