package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Every binary-invoking hook self-provisions, because SessionStart is a single
// point of failure the field has already seen fail (iss-253, iss-254): a session
// where the SessionStart chain never fired — or ran without CLAUDE_PLUGIN_ROOT
// and took its silent exit — left the plugin root without a binary for hours,
// while every other hook failed as a raw "/bin/sh: .../abcd: No such file or
// directory" on every prompt and every tool call: noisy on each, actionable on
// none. So each hook that calls the binary now (1) exits quietly when no plugin
// root is set, (2) attempts one rate-limited, silent bootstrap salvage when the
// binary is absent, and (3) on continued absence says what is degraded and how
// to fix it in one plain line, with a non-blocking exit. The rate limit is a
// stamp file, so a machine where provisioning cannot succeed pays the download
// timeout at most once per window, not on every hook firing. SessionEnd is the
// exception to (2): it never salvages, because the harness cancels a slow hook
// at session exit and a mid-download cancellation loses the transcript capture
// (iss-2608210934566223) — see TestSessionEndNeverBootstraps.

// binaryHook is one binary-invoking hook event under test: the event name, an
// optional matcher, and the verb the stub binary must record when the hook runs.
type binaryHook struct {
	event string
	verb  string
	// neverBootstraps marks a hook that must NOT attempt the salvage download.
	// Both such hooks fire where the harness will cancel a slow hook rather
	// than wait — SessionEnd at session exit, SubagentStop at a sub-agent's —
	// and a blocking download there loses the transcript the hook exists to
	// capture (iss-2608210934566223).
	neverBootstraps bool
}

// binaryHooks enumerates every hook that invokes the binary outside the
// SessionStart chain (which has its own tests and remains the loud, primary
// provisioner).
var binaryHooks = []binaryHook{
	{event: "UserPromptSubmit", verb: "prompt-router"},
	{event: "PreToolUse", verb: "guard"},
	{event: "PreCompact", verb: "prompt-router-reset"},
	{event: "SessionEnd", verb: "session-end", neverBootstraps: true},
	{event: "SubagentStop", verb: "subagent-stop", neverBootstraps: true},
}

// hookCommand returns the single command string for an event, failing on any
// other shape — the one-entry-one-command rule holds for every event, for the
// same parallel-execution reason SessionStart pins it.
func hookCommand(t *testing.T, event string) string {
	t.Helper()
	data, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatalf("reading the committed hooks manifest: %v", err)
	}
	var doc sessionStartHooks
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}
	entries := doc.Hooks[event]
	if len(entries) != 1 || len(entries[0].Hooks) != 1 {
		t.Fatalf("%s must declare exactly one entry group holding one command", event)
	}
	return entries[0].Hooks[0].Command
}

// hookRun executes an event's shipped command under `sh -c` against a plugin
// root, returning stdout, stderr, and the exit code separately. PATH is
// controlled — system utilities plus one caller-supplied directory — because
// the hooks' last resolution rung is a PATH lookup and the developer machine
// running these tests may itself carry a real `abcd` there.
func hookRun(t *testing.T, event, root, pathDir string) (string, string, int) {
	t.Helper()
	return hookRunIn(t, event, root, pathDir, "")
}

// hookRunIn is hookRun with an explicit working directory. The PATH rung's
// containment check is relative to the directory the shim runs in — a hostile
// clone puts its own `abcd` inside the checkout the session is working on — so
// a test that plants a binary "inside the working tree" has to choose that
// tree. An empty dir inherits the test process's own working directory, which
// is what every pre-existing caller wants.
func hookRunIn(t *testing.T, event, root, pathDir, dir string) (string, string, int) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	pathEnv := "/usr/bin:/bin"
	if pathDir != "" {
		pathEnv = pathDir + ":" + pathEnv
	}
	cmd := exec.Command("sh", "-c", hookCommand(t, event))
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"session_id":"s1"}`)
	cmd.Env = []string{
		"PATH=" + pathEnv,
		"HOME=" + t.TempDir(),
		"CLAUDE_PLUGIN_ROOT=" + root,
		"ABCD_CALLS=" + filepath.Join(root, "calls.log"),
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		e, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running the %s command: %v (stderr %s)", event, err, stderr.String())
		}
		code = e.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}

// provisioningBootstrap is a stub bootstrap.sh that records its invocation and
// installs a stub binary — the success case of a salvage.
const provisioningBootstrap = `#!/bin/sh
printf 'bootstrap\n' >> "$CLAUDE_PLUGIN_ROOT/boot.log"
cat > "$CLAUDE_PLUGIN_ROOT/abcd" <<'EOF'
#!/bin/sh
cat >/dev/null
printf '%s %s\n' "$1" "$2" >> "$ABCD_CALLS"
exit 0
EOF
chmod +x "$CLAUDE_PLUGIN_ROOT/abcd"
exit 0
`

// failingBootstrap records its invocation and provisions nothing.
const failingBootstrap = `#!/bin/sh
printf 'bootstrap\n' >> "$CLAUDE_PLUGIN_ROOT/boot.log"
exit 1
`

// hookRoot builds a plugin root with the given bootstrap stub and, optionally,
// a pre-existing stub binary.
func hookRoot(t *testing.T, bootstrap string, withBinary bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "bootstrap.sh"), []byte(bootstrap), 0o755); err != nil {
		t.Fatal(err)
	}
	if withBinary {
		bin := "#!/bin/sh\ncat >/dev/null\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit 0\n"
		if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(bin), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestBinaryHooksProvisionWhenTheBinaryIsAbsent is the iss-253 field failure
// inverted: an empty plugin root plus a working bootstrap must yield a running
// hook, whichever hook fires first. SessionEnd is the one exception: the
// session is exiting, the harness cancels a slow hook rather than wait, and a
// blocking download there loses the transcript it exists to capture
// (iss-2608210934566223) — TestSessionEndNeverBootstraps pins the inverse.
func TestBinaryHooksProvisionWhenTheBinaryIsAbsent(t *testing.T) {
	for _, h := range binaryHooks {
		if h.neverBootstraps {
			continue
		}
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, provisioningBootstrap, false)
			_, stderr, _ := hookRun(t, h.event, root, "")
			if !strings.Contains(callLog(t, filepath.Join(root, "calls.log")), h.verb) {
				t.Fatalf("%s did not run the provisioned binary with verb %q; stderr: %s", h.event, h.verb, stderr)
			}
			if strings.Contains(stderr, "No such file") {
				t.Fatalf("%s leaked a raw exec failure: %s", h.event, stderr)
			}
		})
	}
}

// TestBinaryHooksSayWhatIsDegradedWhenUnprovisionable pins the failure UX: one
// plain actionable line naming the install remedy, a non-blocking exit, and no
// raw shell exec error.
func TestBinaryHooksSayWhatIsDegradedWhenUnprovisionable(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			_, stderr, code := hookRun(t, h.event, root, "")
			if code == 0 || code == 127 {
				t.Fatalf("%s exit = %d; want a non-zero, non-exec-failure exit", h.event, code)
			}
			if code == 2 && h.event == "UserPromptSubmit" {
				t.Fatalf("UserPromptSubmit must not exit 2 (it would block the prompt)")
			}
			if strings.Contains(stderr, "No such file") {
				t.Fatalf("%s leaked a raw exec failure: %s", h.event, stderr)
			}
			line := firstLine(stderr)
			if !strings.Contains(line, "abcd") || !strings.Contains(stderr, "#install") {
				t.Fatalf("%s stderr is not one actionable abcd line naming the install remedy: %q", h.event, stderr)
			}
		})
	}
}

// TestBinaryHooksRateLimitTheSalvage: a recent attempt stamp suppresses the
// bootstrap call entirely, so an unprovisionable machine pays the download
// timeout once per window — not once per prompt and per tool call.
func TestBinaryHooksRateLimitTheSalvage(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			stamp := filepath.Join(root, ".bootstrap.attempt")
			if err := os.WriteFile(stamp, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			if err := os.Chtimes(stamp, now, now); err != nil {
				t.Fatal(err)
			}
			hookRun(t, h.event, root, "")
			if callLog(t, filepath.Join(root, "boot.log")) != "" {
				t.Fatalf("%s invoked the bootstrap despite a fresh attempt stamp", h.event)
			}
		})
	}
}

// TestBinaryHooksSteadyStateBypassesTheSalvage: with the binary present, the
// bootstrap is never consulted and the binary's own exit code passes through.
func TestBinaryHooksSteadyStateBypassesTheSalvage(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, true)
			_, stderr, code := hookRun(t, h.event, root, "")
			if code != 0 {
				t.Fatalf("%s exit = %d with a healthy binary; stderr: %s", h.event, code, stderr)
			}
			if callLog(t, filepath.Join(root, "boot.log")) != "" {
				t.Fatalf("%s consulted the bootstrap despite a present binary", h.event)
			}
			if !strings.Contains(callLog(t, filepath.Join(root, "calls.log")), h.verb) {
				t.Fatalf("%s did not run the binary with verb %q", h.event, h.verb)
			}
		})
	}
}

// TestBinaryHooksExitQuietlyWithoutAPluginRoot: no CLAUDE_PLUGIN_ROOT means not
// a plugin session; every hook stands down silently instead of failing on an
// unexpanded path.
func TestBinaryHooksExitQuietlyWithoutAPluginRoot(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", hookCommand(t, h.event))
			cmd.Stdin = strings.NewReader("{}")
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir()}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("%s without a plugin root must exit 0; got %v (stderr %s)", h.event, err, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("%s without a plugin root must be silent; stderr: %s", h.event, stderr.String())
			}
		})
	}
}

// TestGuardHookKeepsItsExitCodeFence: the guard's 0/1/2 contract survives the
// salvage wrapper — a blocking verdict still blocks, and an unexpected exit is
// still converted to the loud UNGUARDED warning.
func TestGuardHookKeepsItsExitCodeFence(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	for _, tc := range []struct{ binExit, want int }{{0, 0}, {1, 1}, {2, 2}, {7, 1}} {
		root := hookRoot(t, failingBootstrap, false)
		bin := "#!/bin/sh\ncat >/dev/null\nexit " + string(rune('0'+tc.binExit)) + "\n"
		if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(bin), 0o755); err != nil {
			t.Fatal(err)
		}
		_, stderr, code := hookRun(t, "PreToolUse", root, "")
		if code != tc.want {
			t.Fatalf("guard exit %d passed through as %d, want %d (stderr %s)", tc.binExit, code, tc.want, stderr)
		}
		if tc.binExit == 7 && !strings.Contains(stderr, "UNGUARDED") {
			t.Fatalf("an unexpected guard exit must warn UNGUARDED; stderr: %s", stderr)
		}
	}
}

// TestSessionEndNeverBootstraps: SessionEnd is the transcript-capture hook, and
// it fires exactly when the session is going away — the harness cancels a
// still-running SessionEnd hook rather than wait for it. A blocking bootstrap
// download there is therefore a race the capture loses: after a plugin update
// lands a fresh binary-less cache dir, update-then-quit exits through
// SessionEnd, the download is cancelled mid-flight, and the session's
// transcript is silently lost (iss-2608210934566223, field-hit 2026-08-21).
// So SessionEnd must never invoke bootstrap.sh and must perform no network
// work: plugin-root binary first, PATH binary second, else one plain line and
// a non-blocking failure exit.
func TestSessionEndNeverBootstraps(t *testing.T) {
	command := hookCommand(t, "SessionEnd")
	if strings.Contains(command, "bootstrap.sh") {
		t.Fatalf("the SessionEnd command references bootstrap.sh — session end must never download the binary: %q", command)
	}
	if !strings.Contains(command, "hook session-end") {
		t.Fatalf("the SessionEnd command no longer invokes `hook session-end`: %q", command)
	}
	// Behavioural pin, not only a spelling pin: even a working bootstrap must
	// not be consulted when the binary is absent at session end.
	root := hookRoot(t, provisioningBootstrap, false)
	_, stderr, code := hookRun(t, "SessionEnd", root, "")
	if callLog(t, filepath.Join(root, "boot.log")) != "" {
		t.Fatalf("SessionEnd invoked the bootstrap; a session-end download races the harness's hook cancellation and loses the transcript")
	}
	if code == 0 || code == 127 {
		t.Fatalf("SessionEnd exit = %d without a binary; want a non-zero, non-exec-failure exit", code)
	}
	if !strings.Contains(stderr, "transcript was not captured") || !strings.Contains(stderr, "#install") {
		t.Fatalf("SessionEnd stderr must keep the one-line transcript-not-captured remedy: %q", stderr)
	}
}

// TestBinaryHooksFallBackToAPathBinary: the hooks carry the command surface's
// resolution ladder — plugin root first, PATH second — so a machine where the
// one-line install has run is rescued even when the plugin root cannot be
// provisioned at all.
func TestBinaryHooksFallBackToAPathBinary(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			stub := "#!/bin/sh\ncat >/dev/null\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit 0\n"
			if err := os.WriteFile(filepath.Join(pathDir, "abcd"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := hookRun(t, h.event, root, pathDir)
			if code != 0 {
				t.Fatalf("%s exit = %d with an abcd on PATH; stderr: %s", h.event, code, stderr)
			}
			if !strings.Contains(callLog(t, filepath.Join(root, "calls.log")), h.verb) {
				t.Fatalf("%s did not run the PATH binary with verb %q; stderr: %s", h.event, h.verb, stderr)
			}
		})
	}
}

// The PATH rung is the shims' last resort, and until GHSA-gx3m-3224-qqcv's
// design fork is settled it stays open to any `abcd` the operator's PATH
// resolves. Two shapes need no decision to refuse, because the documented
// install never produces them: a binary the working tree itself controls (a
// `.` or in-checkout PATH entry, so a hostile clone becomes the guard and the
// rules loader for the session reading it) and a binary in a world-writable
// directory (any local user's to replace). Both degrade to the shim's existing
// loud line, plus one line saying which binary was ignored and why. The full
// owned-only rung — refusing every PATH binary that `~/.abcd/path-entry` does
// not vouch for — is the parent record's open decision and is NOT attempted
// here; the documented rescue through an ordinary directory such as
// ~/.local/bin keeps working, which TestBinaryHooksFallBackToAPathBinary pins.

// pathStub writes the recording stub binary into dir.
func pathStub(t *testing.T, dir string) {
	t.Helper()
	stub := "#!/bin/sh\ncat >/dev/null\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "abcd"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
}

// pathRefusalReasons is every reason the PATH rung can print. A test names the
// one (or, where the shell decides, the ones) it expects; the helper then proves
// no OTHER reason fired, which is what turns "it refused" into "it refused for
// the reason this test is about".
var pathRefusalReasons = []string{
	"it lives inside the working tree",
	"it did not resolve to an absolute path",
	"its directory could not be resolved",
	"its directory is world-writable",
}

// assertPathBinaryRefused: the stub never ran, the shim still failed loudly with
// its own remedy line, and one line names the ignored PATH binary AND the reason
// it was ignored.
//
// Both are asserted, and the reason is asserted exclusively — the fired reason
// must be in wantReasons and no other known reason may appear. The generic
// prefix alone cannot tell the refusals apart: a stub reached through a relative
// PATH element is also inside the working tree, so a test that checked only
// "ignoring the abcd found on PATH" passes on whichever branch happens to fire
// and keeps passing after the branch it names stops existing. The binary is
// asserted because that is what the operator needs in order to go and look at
// it, and docs/how-to/install.md promises the line names it.
func assertPathBinaryRefused(t *testing.T, h binaryHook, root, stderr string, code int, wantBinaries, wantReasons []string) {
	t.Helper()
	if log := callLog(t, filepath.Join(root, "calls.log")); log != "" {
		t.Fatalf("%s executed the untrusted PATH binary: %q", h.event, log)
	}
	if code == 0 || code == 127 {
		t.Fatalf("%s exit = %d after refusing the PATH binary; want a non-zero, non-exec-failure exit; stderr: %s", h.event, code, stderr)
	}
	if !strings.Contains(stderr, "ignoring the abcd found on PATH") {
		t.Fatalf("%s did not say that it ignored the PATH binary or why; stderr: %s", h.event, stderr)
	}
	if !containsAny(stderr, wantBinaries) {
		t.Fatalf("%s did not name the ignored binary (want one of %q); stderr: %s", h.event, wantBinaries, stderr)
	}
	if !containsAny(stderr, wantReasons) {
		t.Fatalf("%s refused for the wrong reason (want one of %q); stderr: %s", h.event, wantReasons, stderr)
	}
	for _, r := range pathRefusalReasons {
		if slices.Contains(wantReasons, r) {
			continue
		}
		if strings.Contains(stderr, r) {
			t.Fatalf("%s refused with an unexpected reason %q; stderr: %s", h.event, r, stderr)
		}
	}
	if !strings.Contains(stderr, "#install") {
		t.Fatalf("%s dropped its own degraded line; stderr: %s", h.event, stderr)
	}
	if h.event == "PreToolUse" && !strings.Contains(stderr, "UNGUARDED") {
		t.Fatalf("PreToolUse must still say UNGUARDED when it refuses the PATH binary; stderr: %s", stderr)
	}
}

// containsAny reports whether s contains any of the candidates.
func containsAny(s string, candidates []string) bool {
	for _, c := range candidates {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}

// TestBinaryHooksRefuseAPathBinaryInsideTheWorkingTree: a PATH entry under the
// checkout (a vendored bin directory, say) hands the session's guard and rules
// loader to the repository being worked on.
func TestBinaryHooksRefuseAPathBinaryInsideTheWorkingTree(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			work := t.TempDir()
			pathDir := filepath.Join(work, "vendor", "bin")
			if err := os.MkdirAll(pathDir, 0o755); err != nil {
				t.Fatal(err)
			}
			pathStub(t, pathDir)
			_, stderr, code := hookRunIn(t, h.event, root, pathDir, work)
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{filepath.Join(pathDir, "abcd")},
				[]string{"it lives inside the working tree"})
		})
	}
}

// TestBinaryHooksRefuseARelativePathEntry: a `.` (or empty) PATH element
// resolves `abcd` against whatever directory the hook happens to run in.
//
// Which of two reasons fires is the SHELL's choice, not the shim's, and both are
// the same refusal for the same cause. dash — /bin/sh on the Linux CI leg —
// yields `command -v abcd` = "./abcd" verbatim, so the rung refuses it for not
// being absolute. bash in sh mode — /bin/sh on macOS — resolves a relative PATH
// element against $PWD before answering, so the rung gets an absolute path whose
// directory IS the working directory and refuses it as inside the working tree.
// Both are named here; a world-writable or unresolvable-directory refusal would
// mean the rung reached the wrong branch, and the helper fails on either.
func TestBinaryHooksRefuseARelativePathEntry(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			work := t.TempDir()
			pathStub(t, work)
			_, stderr, code := hookRunIn(t, h.event, root, ".", work)
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{"./abcd", filepath.Join(work, "abcd")},
				[]string{"it did not resolve to an absolute path", "it lives inside the working tree"})
		})
	}
}

// TestBinaryHooksRefuseAWorldWritablePathBinary: any local user can drop a
// binary into a world-writable PATH directory, so what is found there is not
// the operator's choice in the sense the fallback assumes.
func TestBinaryHooksRefuseAWorldWritablePathBinary(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			pathStub(t, pathDir)
			if err := os.Chmod(pathDir, 0o777); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := hookRunIn(t, h.event, root, pathDir, t.TempDir())
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{filepath.Join(pathDir, "abcd")},
				[]string{"its directory is world-writable"})
		})
	}
}

// TestSubagentStopNeverBootstraps is TestSessionEndNeverBootstraps' argument at
// the other exit. SubagentStop fires when a sub-agent is going away, and the
// harness cancels a still-running hook there the same way it does at session
// end — so a blocking bootstrap download is a race the sub-agent's transcript
// loses, and it would stall the parent session while it lost it. Plugin-root
// binary first, PATH binary second, else one plain line and a non-blocking exit.
func TestSubagentStopNeverBootstraps(t *testing.T) {
	command := hookCommand(t, "SubagentStop")
	if strings.Contains(command, "bootstrap.sh") {
		t.Fatalf("the SubagentStop command references bootstrap.sh — a sub-agent's exit must never download the binary: %q", command)
	}
	if !strings.Contains(command, "hook subagent-stop") {
		t.Fatalf("the SubagentStop command no longer invokes `hook subagent-stop`: %q", command)
	}
	root := hookRoot(t, provisioningBootstrap, false)
	_, stderr, code := hookRun(t, "SubagentStop", root, "")
	if callLog(t, filepath.Join(root, "boot.log")) != "" {
		t.Fatal("SubagentStop invoked the bootstrap; a download there races the harness's hook cancellation and stalls the session")
	}
	if code == 2 {
		t.Fatal("SubagentStop exited 2 without a binary — that is the host's BLOCKING status and would stop the sub-agent from finishing")
	}
	if code == 0 || code == 127 {
		t.Fatalf("SubagentStop exit = %d without a binary; want a non-zero, non-exec-failure exit", code)
	}
	if !strings.Contains(stderr, "transcript was not captured") || !strings.Contains(stderr, "#install") {
		t.Fatalf("SubagentStop stderr must keep the one-line transcript-not-captured remedy: %q", stderr)
	}
}
