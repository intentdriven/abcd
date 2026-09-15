package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// preToolUse builds the host's PreToolUse payload for a Bash tool call. The
// adapter reads exactly two things out of it — the tool name and
// tool_input.command — so the fixture spells both out rather than hiding them
// behind a helper's defaults.
func preToolUse(t *testing.T, tool, command, cwd string) string {
	t.Helper()
	payload := map[string]any{
		"session_id":      "s1",
		"cwd":             cwd,
		"hook_event_name": "PreToolUse",
		"tool_name":       tool,
		"tool_input":      map[string]any{"command": command, "description": "x"},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestGuardHookBlocksWithHostExitCode is AC 2 on the guard plane: a blocker match
// exits 2 — the host's blocking status — and puts the successor and the why on
// stderr, which is the channel the host feeds back to the agent. The block is the
// lesson, so both must be present.
func TestGuardHookBlocksWithHostExitCode(t *testing.T) {
	dir := guardRepo(t)
	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", dir), "guard", "hook")

	if code != 2 {
		t.Errorf("a blocker must map to the host's blocking exit status 2; got %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "absolute path") || !strings.Contains(stderr, "the delete still runs") {
		t.Errorf("the block message must carry the successor and the why; stderr = %q", stderr)
	}
}

// TestGuardHookAllowsSilently is the common case, and the one that must stay
// cheap and quiet: an allowed command produces no output at all.
func TestGuardHookAllowsSilently(t *testing.T) {
	dir := guardRepo(t)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "git status --porcelain", dir), "guard", "hook")

	if code != 0 {
		t.Errorf("an allow must exit 0; got %d (stderr %q)", code, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("an allow must be silent; stdout=%q stderr=%q", stdout, stderr)
	}
}

// TestGuardHookWarnAllowsAndSurfaces is AC 2's warn half on the guard plane: the
// command runs (never the blocking status 2) and the warning is said out loud. It
// exits 1, not 0, because a pre-tool-use hook that exits 0 has its stderr
// DISCARDED — the same loud-but-non-blocking status failOpen uses — so a warn is
// visible rather than silent (iss-231).
func TestGuardHookWarnAllowsAndSurfaces(t *testing.T) {
	dir := guardRepo(t)
	_, stderr, code := runGuard(preToolUse(t, "Bash", "git clean -fd", dir), "guard", "hook")

	if code != 1 {
		t.Errorf("a warn must be loud but non-blocking: want exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "git-clean") {
		t.Errorf("the warning must be surfaced and name its entry; stderr = %q", stderr)
	}
}

// TestGuardHookFailsOpenLoud is AC 1 at the adapter: every input the adapter
// cannot turn into a decision allows the command AND says so unmissably. A silent
// allow would leave a session unguarded with nobody told; a block would brick it.
//
// "Unmissable" is why the exit code is 1 and not 0. A pre-tool-use hook that
// exits 0 has its stderr discarded — the warning would exist and nobody would
// ever see it. A non-zero, non-blocking status is the one channel that both lets
// the command run and puts the warning in front of a human.
func TestGuardHookFailsOpenLoud(t *testing.T) {
	cases := []struct {
		name  string
		stdin func(t *testing.T, dir string) string
	}{
		{"unparsable JSON", func(t *testing.T, _ string) string { return "{not json" }},
		{"empty payload", func(t *testing.T, _ string) string { return "" }},
		{"non-Bash tool call", func(t *testing.T, dir string) string {
			return preToolUse(t, "Write", "cd scratch && rm -rf *", dir)
		}},
		{"Bash call with no command", func(t *testing.T, dir string) string {
			return preToolUse(t, "Bash", "", dir)
		}},
		{"command the tokenizer cannot split", func(t *testing.T, dir string) string {
			return preToolUse(t, "Bash", `rm -rf "unterminated`, dir)
		}},
		// A malformed per-repo registry is NOT a fail-open case any more: it is
		// fail-SAFE (bundled hazards stay armed). Its behaviour is pinned by
		// TestGuardHookBrokenRepoConfigKeepsBundledHazardsArmed below
		// (iss-2608261551087492).
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := guardRepo(t)
			_, stderr, code := runGuard(tc.stdin(t, dir), "guard", "hook")

			if code == 2 {
				t.Errorf("must fail OPEN: the blocking status must never come from a non-decision")
			}
			if code == 0 {
				t.Errorf("must fail LOUD: exit 0 discards the hook's stderr, so the warning would never be seen")
			}
			if !strings.Contains(stderr, "abcd guard") {
				t.Errorf("must fail LOUD: stderr must carry an abcd guard warning; stderr = %q", stderr)
			}
		})
	}
}

// TestGuardHookBlocksWhatBashWouldRun — GHSA-5wx3-2c86-fjpx. Two inputs the
// tokenizer used to refuse are inputs bash RUNS: a trailing backslash (dropped
// by bash 3.2 and zsh) and a here-document body with no delimiter line
// (recovered silently). On the hook a tokenizer error is fail-open, so each was
// a one-byte bypass of every blocker. Both must now reach the blocking status
// with the entry named, while the unterminated-quote row in
// TestGuardHookFailsOpenLoud stays a loud fail-open: no shell runs that one.
func TestGuardHookBlocksWhatBashWouldRun(t *testing.T) {
	for name, command := range map[string]string{
		"trailing backslash":            "git push --force origin main \\",
		"unterminated heredoc body":     "git push --force origin main <<EOF\n",
		"spaced arithmetic then hazard": "echo $(( x << y ))\ngit push --force origin main",
	} {
		t.Run(name, func(t *testing.T) {
			dir := guardRepo(t)
			_, stderr, code := runGuard(preToolUse(t, "Bash", command, dir), "guard", "hook")
			if code != 2 {
				t.Errorf("bash runs this line, so the hook must block it: want exit 2, got %d (stderr %q)", code, stderr)
			}
			if !strings.Contains(stderr, "git-push-force") {
				t.Errorf("the block must name the entry; stderr = %q", stderr)
			}
		})
	}
}

// TestGuardHookBrokenRepoConfigKeepsBundledHazardsArmed pins the fail-SAFE
// doctrine of iss-2608261551087492. A malformed repo .abcd/guard.json must NOT
// disable the whole guard: the repo's own overrides are dropped, but the bundled
// hazards stay armed and keep BLOCKING. The drop is announced loudly so a human
// learns their committed guard config is broken and that bundled protection is
// still active — the exact opposite of the old fail-open behaviour.
func TestGuardHookBrokenRepoConfigKeepsBundledHazardsArmed(t *testing.T) {
	writeBroken := func(t *testing.T, dir string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("bundled blocker still blocks", func(t *testing.T) {
		dir := guardRepo(t)
		writeBroken(t, dir)
		// git commit --no-verify is a bundled blocker that never depended on the
		// repo layer; a broken repo config must not defang it.
		_, stderr, code := runGuard(preToolUse(t, "Bash", `git commit --no-verify -m "wip"`, dir), "guard", "hook")
		if code != 2 {
			t.Fatalf("a bundled blocker must still exit 2 when only the repo layer is broken; got %d, stderr = %q", code, stderr)
		}
		if !strings.Contains(stderr, guard.RepoRelPath) {
			t.Errorf("the dropped repo layer must be announced by name; stderr = %q", stderr)
		}
		if !strings.Contains(stderr, "DROPPED") {
			t.Errorf("the broken repo layer must be announced loudly as dropped; stderr = %q", stderr)
		}
	})

	t.Run("allowed command still announces the drop loudly", func(t *testing.T) {
		dir := guardRepo(t)
		writeBroken(t, dir)
		// An innocuous command the bundled registry allows. The drop notice must
		// still reach a human — exit 0 would discard it — so the hook exits 1
		// (loud, non-blocking), never 0, and never the blocking 2.
		_, stderr, code := runGuard(preToolUse(t, "Bash", "ls -la", dir), "guard", "hook")
		if code == 2 {
			t.Fatalf("an allowed command must not be blocked; got exit 2, stderr = %q", stderr)
		}
		if code == 0 {
			t.Fatalf("exit 0 discards stderr, so the broken-repo notice would be lost; stderr = %q", stderr)
		}
		if !strings.Contains(stderr, "DROPPED") || !strings.Contains(stderr, "bundled hazards remain armed") {
			t.Errorf("the notice must say the repo layer dropped and the bundled hazards remain armed; stderr = %q", stderr)
		}
	})
}

// TestGuardHookAnnouncesADisabledRegistry is AC 1 applied to the escape hatch.
// A disabled registry allows everything, which makes it an unguarded session —
// and an unguarded session that says nothing is the one failure mode this whole
// feature exists to prevent. It is not a FAULT (someone chose it, and the choice
// is in a file a reviewer can see), but it must never pass for protection.
//
// It matters more than the other unguarded states, not less: those need a broken
// install, while this one needs a single file write that the guard itself allows.
func TestGuardHookAnnouncesADisabledRegistry(t *testing.T) {
	dir := guardRepo(t)
	cfg := `{"schema_version":1,"disabled":true,"entries":{}}`
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", dir), "guard", "hook")

	if code == 2 {
		t.Error("a disabled registry allows: it must never produce the blocking status")
	}
	if code == 0 {
		t.Error("exit 0 discards the hook's stderr, so a disabled guard would run in silence")
	}
	if !strings.Contains(stderr, guard.RepoRelPath) {
		t.Errorf("the warning must name the file that turned the guard off; stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("a disabled guard is an unguarded session and must say so; stderr = %q", stderr)
	}
}

// TestGuardHookIgnoresAncestorKillSwitch is GHSA-vvqc-3mv2-5p49 on the guard
// plane: a guard.json with the kill switch set, planted ABOVE the git working
// tree, must not disarm the guard for a session inside it. The bundled hazards
// stay armed and the blocker still blocks with the host's exit status.
func TestGuardHookIgnoresAncestorKillSwitch(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	planted := `{"schema_version":1,"disabled":true}`
	if err := os.WriteFile(filepath.Join(outer, ".abcd", "guard.json"), []byte(planted), 0o644); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner)

	_, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", inner), "guard", "hook")
	if code != 2 {
		t.Errorf("a guard.json planted above the working tree disarmed the guard: exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "rm-rf-after-cd-chain") {
		t.Errorf("the bundled blocker must still fire; stderr = %q", stderr)
	}
}

// TestGuardCheckAndHookAgreeOnAHereDocumentLeftOpen pins the two front doors to
// the same verdict for the same command. `guard check` trims the trailing
// newline off a candidate read from stdin, and the tokenizer used to resolve a
// pending here-document only when it crossed a newline — so `cat <<EOF` with its
// newline intact took the fail-closed heredoc-unterminated block on the hook,
// while the identical command reached `check` one byte shorter and was cleared.
// A verdict belongs to the command, not to whether its last byte is a newline.
func TestGuardCheckAndHookAgreeOnAHereDocumentLeftOpen(t *testing.T) {
	for name, command := range map[string]string{
		"with a trailing newline": "cat <<EOF\n",
		"ending at the `<<` line": "cat <<EOF",
	} {
		t.Run(name, func(t *testing.T) {
			dir := guardRepo(t)
			_, stderr, code := runGuard(preToolUse(t, "Bash", command, dir), "guard", "hook")
			if code != 2 {
				t.Errorf("hook: an unterminated here-document must block: want exit 2, got %d (stderr %q)", code, stderr)
			}
			stdout, stderr, code := runGuard(command, "guard", "check")
			if code != 1 {
				t.Errorf("check on stdin: want the blocking exit 1, got %d (stdout %q stderr %q)", code, stdout, stderr)
			}
			stdout, stderr, code = runGuard("", "guard", "check", "--command", command)
			if code != 1 {
				t.Errorf("check --command: want the blocking exit 1, got %d (stdout %q stderr %q)", code, stdout, stderr)
			}
		})
	}
}
