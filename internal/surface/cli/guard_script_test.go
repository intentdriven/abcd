package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The guard reads a script the command runs from the directory it runs in
// (adr-2610091150447054 decision 4): `abcd guard check` from the process's
// own directory, the hook from the host's workdir when it names an existing
// one and from the session directory otherwise.

const scriptHazard = "git push --force origin main\n"

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGuardCheckReadsAScriptFromTheWorkingDirectory(t *testing.T) {
	dir := guardRepo(t)
	writeFile(t, filepath.Join(dir, "s.sh"), scriptHazard)
	writeFile(t, filepath.Join(dir, "ok.sh"), "echo hi\n")

	stdout, _, code := runGuard("bash s.sh", "guard", "check")
	if code != 1 || !strings.Contains(stdout, "script-runs-hazard") || !strings.Contains(stdout, "git-push-force") {
		t.Fatalf("bash s.sh: exit %d\n%s", code, stdout)
	}
	if _, _, code := runGuard("bash ok.sh", "guard", "check"); code != 0 {
		t.Fatalf("bash ok.sh: exit %d, want an allow", code)
	}
	stdout, _, code = runGuard("bash absent.sh", "guard", "check")
	if code != 0 || !strings.Contains(stdout, "note:") || !strings.Contains(stdout, "absent.sh") {
		t.Fatalf("bash absent.sh: exit %d, want an allow with a note\n%s", code, stdout)
	}
}

func TestGuardHookReadsAScriptFromTheWorkdirElseTheSession(t *testing.T) {
	dir := workdirSession(t)
	writeFile(t, filepath.Join(dir, "cold", "s.sh"), scriptHazard)
	writeFile(t, filepath.Join(dir, "s.sh"), "echo hi\n")

	t.Run("the workdir's script blocks", func(t *testing.T) {
		stdout, stderr, code := runGuard(preToolUseIn(t, "bash s.sh", dir, "cold"), "guard", "hook")
		reason := mustDeny(t, stdout, stderr, code)
		if !strings.Contains(reason, "s.sh") || !strings.Contains(reason, "git-push-force") {
			t.Errorf("reason = %q", reason)
		}
	})
	t.Run("without a workdir the session directory's script is read", func(t *testing.T) {
		stdout, stderr, code := runGuard(preToolUseIn(t, "bash s.sh", dir, nil), "guard", "hook")
		if code != 0 || stdout != "" {
			t.Errorf("the session's s.sh is clean: exit %d stdout %q stderr %q", code, stdout, stderr)
		}
		stdout, stderr, code = runGuard(preToolUseIn(t, "bash cold/s.sh", dir, nil), "guard", "hook")
		mustDeny(t, stdout, stderr, code)
	})
	t.Run("the workdir registry reads the same directory", func(t *testing.T) {
		writeFile(t, filepath.Join(dir, "hot", "r.sh"), "make release\n")
		stdout, stderr, code := runGuard(preToolUseIn(t, "bash r.sh", dir, "hot"), "guard", "hook")
		reason := mustDeny(t, stdout, stderr, code)
		if !strings.Contains(reason, "make-release") {
			t.Errorf("the workdir registry's entry must be read in its script; reason = %q", reason)
		}
	})
}
