package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// preToolUseIn builds a shell-tool payload whose tool input carries a per-call
// working directory, the field a host with one (opencode's bash tool names it
// `workdir`) forwards beside the command. workdir is marshalled as given, so a
// test can hand the adapter a non-string value exactly as a broken or hostile
// adapter would.
func preToolUseIn(t *testing.T, command, cwd string, workdir any) string {
	t.Helper()
	payload := map[string]any{
		"session_id":      "s1",
		"cwd":             cwd,
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]any{"command": command, "description": "x", "workdir": workdir},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// releaseBlockerCfg is a repo guard.json naming a hazard the bundled registry
// does not: `make release` in THAT repository. It is the directory-dependent
// verdict the tests need — the same command is a hazard in one directory and
// ordinary work in another.
const releaseBlockerCfg = `{"schema_version":1,"entries":{"make-release":{"pattern":{"command":"make","subcommand":"release"},"tier":"blocker","successor":"make release-dry-run","why":"cuts and publishes a release from this repository."}}}`

// workdirSession lays out a session directory holding two sub-directories: hot/,
// a repository whose own registry blocks `make release`, and cold/, a plain
// directory with no registry of its own.
func workdirSession(t *testing.T) string {
	t.Helper()
	dir := guardRepo(t)
	hot := filepath.Join(dir, "hot", ".abcd")
	if err := os.MkdirAll(hot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hot, "guard.json"), []byte(releaseBlockerCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cold"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestGuardHookResolvesTheCommandAgainstTheWorkdir is the ruled extension
// (iss-2609212142557657, option A): the guard reads the host-supplied per-call
// working directory, resolves it against the session directory, and checks the
// command against the registry of the directory it will actually run in. The
// same relative workdir value, and the same command, is a block in one place and
// an allow in another.
func TestGuardHookResolvesTheCommandAgainstTheWorkdir(t *testing.T) {
	dir := workdirSession(t)

	t.Run("hazard in the workdir's own registry blocks", func(t *testing.T) {
		_, stderr, code := runGuard(preToolUseIn(t, "make release", dir, "hot"), "guard", "hook")
		if code != 2 {
			t.Fatalf("`make release` run in hot/ must block on hot/'s registry: want exit 2, got %d (stderr %q)", code, stderr)
		}
		if !strings.Contains(stderr, "make-release") || !strings.Contains(stderr, "make release-dry-run") {
			t.Errorf("the refusal must name the workdir registry's entry and its successor; stderr = %q", stderr)
		}
	})
	t.Run("an absolute workdir resolves to the same registry", func(t *testing.T) {
		_, stderr, code := runGuard(preToolUseIn(t, "make release", dir, filepath.Join(dir, "hot")), "guard", "hook")
		if code != 2 {
			t.Errorf("an absolute workdir naming hot/ must block the same way: got %d (stderr %q)", code, stderr)
		}
	})
	t.Run("the same command in a workdir with no such entry is allowed", func(t *testing.T) {
		stdout, stderr, code := runGuard(preToolUseIn(t, "make release", dir, "cold"), "guard", "hook")
		if code != 0 || stdout != "" || stderr != "" {
			t.Errorf("`make release` in cold/ must be a silent allow; got exit %d stdout %q stderr %q", code, stdout, stderr)
		}
	})
	t.Run("the same relative workdir from a session where it names nothing is allowed", func(t *testing.T) {
		// hot/ exists only under dir; from its sibling cold/ the value "hot"
		// names a directory that does not exist, and the host fails the call.
		_, stderr, code := runGuard(preToolUseIn(t, "make release", filepath.Join(dir, "cold"), "hot"), "guard", "hook")
		if code != 0 {
			t.Errorf("a workdir naming no directory runs nothing on the probed host: want exit 0, got %d (stderr %q)", code, stderr)
		}
	})
	t.Run("without a workdir the session registry alone decides", func(t *testing.T) {
		_, stderr, code := runGuard(preToolUse(t, "Bash", "make release", dir), "guard", "hook")
		if code != 0 {
			t.Errorf("the session registry does not name `make release`: want exit 0, got %d (stderr %q)", code, stderr)
		}
	})
}

// TestGuardHookWorkdirRegistryCannotDisarmTheSession: the session's registry is
// the floor. A workdir whose own guard.json switches the guard off, or re-tiers a
// bundled blocker, can add hazards but never subtract one, because a model
// chooses the workdir and a planted file there would otherwise be a one-field
// kill switch.
func TestGuardHookWorkdirRegistryCannotDisarmTheSession(t *testing.T) {
	cases := map[string]string{
		"kill switch":        `{"schema_version":1,"disabled":true}`,
		"blocker re-tiered":  `{"schema_version":1,"entries":{"git-push-force":{"tier":"warn"}}}`,
		"malformed override": `{not json`,
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			dir := guardRepo(t)
			off := filepath.Join(dir, "off", ".abcd")
			if err := os.MkdirAll(off, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(off, "guard.json"), []byte(cfg), 0o644); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := runGuard(preToolUseIn(t, "git push --force origin main", dir, "off"), "guard", "hook")
			if code != 2 || !strings.Contains(stderr, "git-push-force") {
				t.Errorf("the session's bundled blocker must still block in a workdir whose registry says otherwise: exit %d, stderr %q", code, stderr)
			}
		})
	}
}

// TestGuardHookMissingWorkdirIsNotAFailedCd pins what the probe found on the one
// host with a per-call workdir (opencode 1.18.31): a workdir that does not exist,
// or names a file, FAILS the tool call — nothing runs, and nothing falls back to
// the session directory. So the failed-cd hazard rm-rf-after-cd-chain exists for
// does not exist for a host workdir, and the guard must not manufacture it by
// reading the workdir as `cd <workdir> &&`. The cd chain spelled in the command
// string still blocks, workdir or not.
func TestGuardHookMissingWorkdirIsNotAFailedCd(t *testing.T) {
	dir := workdirSession(t)
	if err := os.WriteFile(filepath.Join(dir, "afile"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, wd := range []string{"missing", "afile", "cold"} {
		t.Run("rm -rf * in workdir "+wd, func(t *testing.T) {
			_, stderr, code := runGuard(preToolUseIn(t, "rm -rf *", dir, wd), "guard", "hook")
			if code != 0 {
				t.Errorf("a host workdir is not a shell cd: want exit 0, got %d (stderr %q)", code, stderr)
			}
		})
	}
	t.Run("a cd chain in the command still blocks inside a workdir", func(t *testing.T) {
		_, stderr, code := runGuard(preToolUseIn(t, "cd scratch && rm -rf *", dir, "cold"), "guard", "hook")
		if code != 2 || !strings.Contains(stderr, "rm-rf-after-cd-chain") {
			t.Errorf("the command's own cd chain must still block: exit %d, stderr %q", code, stderr)
		}
	})
}

// TestGuardHookRefusesAMalformedWorkdir is the refusal half (guards prove
// themselves): the workdir is written by the model, so a value that names no
// directory the host could run in is refused with the host's blocking status and
// the reason, never resolved into a guess and never allowed to drop the whole
// payload into the fail-open path. Before the field was read, any workdir was
// ignored and the command was checked as if it ran in the session directory; the
// refusal is new behaviour, not a repair of an unguarded path.
func TestGuardHookRefusesAMalformedWorkdir(t *testing.T) {
	cases := []struct {
		name    string
		workdir any
		want    string
	}{
		{"a number", 42, "not a string"},
		{"an object", map[string]any{"path": "cold"}, "not a string"},
		{"a list", []string{"cold"}, "not a string"},
		{"a boolean", true, "not a string"},
		{"a NUL byte", "cold\x00/etc", "NUL"},
		{"a newline", "cold\nrm -rf /", "control character"},
		{"an escape sequence", "cold\x1b[2J", "control character"},
		{"over the length cap", strings.Repeat("a", 5000), "over"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := workdirSession(t)
			stdout, stderr, code := runGuard(preToolUseIn(t, "ls", dir, tc.workdir), "guard", "hook")
			if code != 2 {
				t.Fatalf("a malformed workdir must be refused with the blocking status 2; got %d (stderr %q)", code, stderr)
			}
			if !strings.Contains(stderr, "workdir") || !strings.Contains(stderr, tc.want) {
				t.Errorf("the refusal must name the field and what was wrong with it (%q); stderr = %q", tc.want, stderr)
			}
			if strings.Contains(stderr, "UNGUARDED") {
				t.Errorf("a refusal is a decision, not a fail-open; stderr = %q", stderr)
			}
			if strings.ContainsAny(stderr, "\x00\x1b") {
				t.Errorf("the refusal must not echo raw control bytes to the host; stderr = %q", stderr)
			}
			if stdout != "" {
				t.Errorf("the hook writes nothing to stdout; got %q", stdout)
			}
		})
	}
}

// TestGuardHookAdmitsOrdinaryWorkdirs is the permitted side of the same guard:
// the values a host legitimately sends are never refused.
func TestGuardHookAdmitsOrdinaryWorkdirs(t *testing.T) {
	dir := workdirSession(t)
	spaced := filepath.Join(dir, "café dir")
	if err := os.MkdirAll(spaced, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]any{
		"absent (null)":                 nil,
		"empty (the session directory)": "",
		"relative":                      "cold",
		"absolute":                      filepath.Join(dir, "cold"),
		"dot segments":                  "cold/../cold/",
		"a space and non-ASCII":         "café dir",
		"a missing directory":           "not-there-yet",
	}
	for name, wd := range cases {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := runGuard(preToolUseIn(t, "git status --porcelain", dir, wd), "guard", "hook")
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("an ordinary workdir must be a silent allow; got exit %d stdout %q stderr %q", code, stdout, stderr)
			}
		})
	}
}
