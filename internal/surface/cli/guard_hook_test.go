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
// is the host's deny — exit 0 and one JSON object on stdout — whose reason
// carries the successor and the why, the text the host shows the person and
// hands the agent. The block is the lesson, so both must be present.
func TestGuardHookBlocksWithHostExitCode(t *testing.T) {
	dir := guardRepo(t)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", dir), "guard", "hook")

	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, "absolute path") || !strings.Contains(reason, "the delete still runs") {
		t.Errorf("the block message must carry the successor and the why; reason = %q", reason)
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
// command runs (never a block) and the warning is said out loud. It
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
		// A malformed per-repo registry is NOT a fail-open case any more: it is
		// fail-SAFE (bundled hazards stay armed). Its behaviour is pinned by
		// TestGuardHookBrokenRepoConfigKeepsBundledHazardsArmed below
		// (iss-2608261551087492).
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := guardRepo(t)
			stdout, stderr, code := runGuard(tc.stdin(t, dir), "guard", "hook")

			if code == 2 || isDeny(stdout) {
				t.Errorf("must fail OPEN: a block must never come from a non-decision; stdout = %q", stdout)
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
// a one-byte bypass of every blocker. Both must now reach the host's deny
// with the entry named. A line the tokenizer cannot split at all is blocked
// too (TestGuardHookBlocksAnUnparsableLine).
func TestGuardHookBlocksWhatBashWouldRun(t *testing.T) {
	for name, command := range map[string]string{
		"trailing backslash":            "git push --force origin main \\",
		"unterminated heredoc body":     "git push --force origin main <<EOF\n",
		"spaced arithmetic then hazard": "echo $(( x << y ))\ngit push --force origin main",
	} {
		t.Run(name, func(t *testing.T) {
			dir := guardRepo(t)
			stdout, stderr, code := runGuard(preToolUse(t, "Bash", command, dir), "guard", "hook")
			reason := mustDeny(t, stdout, stderr, code)
			if !strings.Contains(reason, "git-push-force") {
				t.Errorf("the block must name the entry; reason = %q", reason)
			}
		})
	}
}

// TestGuardHookBlocksAnUnparsableLine — review4-guard finding 2. A command
// line the tokenizer cannot split used to fail OPEN: the hook let it run
// unchecked. Where the tokenizer is right, no shell runs the line either, so a
// block costs nothing; where it is wrong — `$'\c'` read as swallowing its own
// closing quote — bash runs a line the guard never read, and the fail-open was
// a bypass of every blocker by construction. The line is blocked under a
// reserved id, with the way past: close the quote.
func TestGuardHookBlocksAnUnparsableLine(t *testing.T) {
	dir := guardRepo(t)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", `rm -rf "unterminated`, dir), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, "command-unparsable") || !strings.Contains(reason, "quote") {
		t.Errorf("the block must name the reserved id and the way past; reason = %q", reason)
	}
	if strings.Contains(reason, "UNGUARDED") {
		t.Errorf("a block is not a fail-open; reason = %q", reason)
	}
}

// TestGuardHookRunsANestedQuoteInABraceExpansion — review5-guard finding 2,
// the other side of the block above. A double-quoted `${…}` whose word carries
// double quotes of its own is valid bash, and an apostrophe in the nested
// quotes is data, so the line runs; it is not one the tokenizer cannot split.
func TestGuardHookRunsANestedQuoteInABraceExpansion(t *testing.T) {
	dir := guardRepo(t)
	for _, line := range []string{
		`echo "${MSG:-"don't"}"`,
		`printf '%s\n' "${NAME:-"O'Brien"}"`,
		`echo "${X//"'"/x}"`,
	} {
		stdout, stderr, code := runGuard(preToolUse(t, "Bash", line, dir), "guard", "hook")
		if code != 0 || isDeny(stdout) || strings.Contains(stderr, "command-unparsable") {
			t.Errorf("%s is valid bash and must run: want a silent exit 0, got %d (stdout %q stderr %q)", line, code, stdout, stderr)
		}
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
		stdout, stderr, code := runGuard(preToolUse(t, "Bash", `git commit --no-verify -m "wip"`, dir), "guard", "hook")
		reason := mustDeny(t, stdout, stderr, code)
		if !strings.Contains(reason, guard.RepoRelPath) {
			t.Errorf("the dropped repo layer must be announced by name; reason = %q", reason)
		}
		if !strings.Contains(reason, "DROPPED") {
			t.Errorf("the broken repo layer must be announced loudly as dropped; reason = %q", reason)
		}
	})

	t.Run("allowed command still announces the drop loudly", func(t *testing.T) {
		dir := guardRepo(t)
		writeBroken(t, dir)
		// An innocuous command the bundled registry allows. The drop notice must
		// still reach a human — exit 0 would discard it — so the hook exits 1
		// (loud, non-blocking), never 0, and never the blocking 2.
		stdout, stderr, code := runGuard(preToolUse(t, "Bash", "ls -la", dir), "guard", "hook")
		if code == 2 || isDeny(stdout) {
			t.Fatalf("an allowed command must not be blocked; got exit %d, stdout = %q", code, stdout)
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
	commitGuardConfig(t, dir, `{"schema_version":1,"disabled":true,"entries":{}}`)
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", dir), "guard", "hook")

	if code == 2 || isDeny(stdout) {
		t.Error("a disabled registry allows: it must never produce a block")
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
// stay armed and the blocker still blocks with the host's deny.
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

	stdout, stderr, code := runGuard(preToolUse(t, "Bash", "cd scratch && rm -rf *", inner), "guard", "hook")
	if reason := mustDeny(t, stdout, stderr, code); !strings.Contains(reason, "rm-rf-after-cd-chain") {
		t.Errorf("the bundled blocker must still fire; reason = %q", reason)
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
			stdout, stderr, code := runGuard(preToolUse(t, "Bash", command, dir), "guard", "hook")
			mustDeny(t, stdout, stderr, code)
			stdout, stderr, code = runGuard(command, "guard", "check")
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
