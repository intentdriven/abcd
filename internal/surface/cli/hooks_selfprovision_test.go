package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	return hookRunHome(t, event, root, pathDir, dir, "")
}

// hookRunHome is hookRunIn with an explicit HOME. The PATH rung's ownership
// check reads `$HOME/.abcd/path-entry`, so a test that vouches for a planted
// binary has to control the home the shim reads. An empty home gets a fresh
// temporary one, which carries no record — the shape every caller that predates
// the ownership rung wants.
func hookRunHome(t *testing.T, event, root, pathDir, dir, home string) (string, string, int) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	pathEnv := "/usr/bin:/bin"
	if pathDir != "" {
		pathEnv = pathDir + ":" + pathEnv
	}
	if home == "" {
		home = sandboxHome(t)
	}
	cmd := exec.Command("sh", "-c", hookCommand(t, event))
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"session_id":"s1"}`)
	cmd.Env = []string{
		"PATH=" + pathEnv,
		"HOME=" + home,
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
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + sandboxHome(t)}
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

// briefChapter locates a committed design-record chapter from this test file's
// own on-disk position, the same way hooksManifest locates the manifest — so
// the assertions below read the chapter that actually ships in the checkout.
func briefChapter(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to locate the test source file")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the committed chapter %s: %v", rel, err)
	}
	return string(data)
}

// bootstrapPassage returns the paragraph of a chapter that documents the
// self-provisioning shims: the run of consecutive non-blank lines carrying the
// `.bootstrap.attempt` throttle the salvage is keyed on. Scoping the assertions
// to that paragraph is what keeps them about the claim under test rather than
// about every sentence in a long chapter.
func bootstrapPassage(t *testing.T, rel, body string) string {
	t.Helper()
	for _, para := range strings.Split(body, "\n\n") {
		if strings.Contains(para, ".bootstrap.attempt") {
			return para
		}
	}
	t.Fatalf("%s no longer documents the .bootstrap.attempt throttle at all", rel)
	return ""
}

// passageLine returns the first line of a passage containing needle, so a
// failure quotes the sentence at fault rather than the whole chapter paragraph.
func passageLine(passage, needle string) string {
	for _, line := range strings.Split(passage, "\n") {
		if strings.Contains(line, needle) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// TestTheBriefNamesSessionEndAsTheBootstrapException: the design record must
// describe the salvage set the manifest actually wires, and SessionEnd is not
// in it. The false universal has already been shipped once and corrected once —
// the README carried it until iss-2608211432384091 — and it survived in the
// brief, where 01-ahoy.md and 05-internals/03-configuration.md both called the
// self-provisioning shims "the four non-SessionStart" ones. That set names
// SessionEnd, which deliberately never downloads (iss-2608210934566223,
// TestSessionEndNeverBootstraps above), so a reader following the brief would
// reintroduce the field failure that lost session 8db3dbd6's transcript. The
// salvage set is derived from the shipped manifest here rather than spelled
// out, so a shim that gains or loses its bootstrap rung fails this test until
// the chapter says so too.
func TestTheBriefNamesSessionEndAsTheBootstrapException(t *testing.T) {
	var salvages []string
	for _, h := range binaryHooks {
		if strings.Contains(hookCommand(t, h.event), "bootstrap.sh") {
			salvages = append(salvages, h.event)
		}
	}
	if slices.Contains(salvages, "SessionEnd") {
		t.Fatal("SessionEnd grew a bootstrap rung; see TestSessionEndNeverBootstraps")
	}
	if len(salvages) == 0 {
		t.Fatal("no binary-invoking hook self-provisions any more; the chapters below describe a salvage that no longer exists")
	}
	for _, rel := range []string{
		".abcd/development/brief/04-surfaces/01-ahoy.md",
		".abcd/development/brief/05-internals/03-configuration.md",
	} {
		t.Run(rel, func(t *testing.T) {
			passage := bootstrapPassage(t, rel, briefChapter(t, rel))
			if strings.Contains(passage, "non-SessionStart") {
				t.Fatalf("%s describes the self-provisioning shims as the non-SessionStart ones; that set includes SessionEnd, which never bootstraps: %q", rel, passageLine(passage, "non-SessionStart"))
			}
			for _, event := range salvages {
				if !strings.Contains(passage, event) {
					t.Fatalf("%s does not name %s, which the shipped manifest does provision through bootstrap.sh", rel, event)
				}
			}
			if !strings.Contains(passage, "SessionEnd") {
				t.Fatalf("%s does not name SessionEnd as the exception, so nothing in the chapter says the transcript hook must not download", rel)
			}
		})
	}
}

// The PATH rung is the shims' last resort, and it is OWNED-ONLY
// (GHSA-gx3m-3224-qqcv, CWE-426): a hook takes an `abcd` from PATH only when
// `~/.abcd/path-entry` records that exact path as the binary this machine
// installed. Before the ownership rule the rung ran whatever `command -v abcd`
// resolved, so a hijack directory early on PATH became the session's rules
// loader and — through PreToolUse, which passes the guard's 0/1/2 verdict
// straight through — an approver of every shell command it was asked about.
// Three shapes are refused before ownership is even consulted, because the
// documented install never produces them: a binary the working tree itself
// controls, a relative resolution, and a binary in a world-writable directory.
// Every refusal degrades to the shim's existing loud line, plus one line saying
// which binary was ignored and why; for PreToolUse that is UNGUARDED and exit
// 1, never a silent 0. SessionStart has no PATH rung at all and still fails
// closed. The install one-liners write the record, which is what keeps the
// documented rescue through ~/.local/bin working.

// writeHookPathEntry vouches for target as this machine's installed abcd, in
// the shape ahoy's writePathEntry produces (path + binary_sha256, plugin_root
// optional). The digest is never read by the shim — adr-46 keeps hashing off
// the hook fast path — but a record without one is not a record the binary
// itself would accept, so the fixture carries a well-formed one.
func writeHookPathEntry(t *testing.T, home, target string) {
	t.Helper()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "path=" + target + "\nbinary_sha256=" + strings.Repeat("a", 64) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "path-entry"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBinaryHooksRunAnOwnedPathBinary: the documented rescue still works. With
// the plugin root unprovisionable and `~/.abcd/path-entry` naming the abcd that
// PATH resolves, every binary-invoking hook runs it.
func TestBinaryHooksRunAnOwnedPathBinary(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			home := t.TempDir()
			pathStub(t, pathDir)
			writeHookPathEntry(t, home, filepath.Join(pathDir, "abcd"))
			_, stderr, code := hookRunHome(t, h.event, root, pathDir, t.TempDir(), home)
			if code != 0 {
				t.Fatalf("%s exit = %d with an owned abcd on PATH; stderr: %s", h.event, code, stderr)
			}
			if !strings.Contains(callLog(t, filepath.Join(root, "calls.log")), h.verb) {
				t.Fatalf("%s did not run the owned PATH binary with verb %q; stderr: %s", h.event, h.verb, stderr)
			}
			if strings.Contains(stderr, "ignoring the abcd found on PATH") {
				t.Fatalf("%s refused the abcd its own path-entry vouches for; stderr: %s", h.event, stderr)
			}
		})
	}
}

// TestBinaryHooksRefuseAnUnrecordedPathBinary is the advisory itself: a hijack
// directory first on PATH, a plausible `abcd` inside it, no plugin-root binary,
// and no ownership record. The planted binary must never run — and PreToolUse
// must report UNGUARDED and exit 1 rather than pass the planted exit 0 through
// as an approval.
func TestBinaryHooksRefuseAnUnrecordedPathBinary(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			pathStub(t, pathDir)
			_, stderr, code := hookRunIn(t, h.event, root, pathDir, t.TempDir())
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{filepath.Join(pathDir, "abcd")},
				[]string{pathRefusalUnowned})
		})
	}
}

// TestBinaryHooksRefuseAPathBinaryTheRecordDoesNotName: a record exists, but it
// vouches for a different file — the shape a hijack directory inserted ahead of
// the real ~/.local/bin install produces. Ownership is the recorded path, not
// the presence of a record.
func TestBinaryHooksRefuseAPathBinaryTheRecordDoesNotName(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			hijack := t.TempDir()
			home := t.TempDir()
			pathStub(t, hijack)
			writeHookPathEntry(t, home, filepath.Join(home, ".local", "bin", "abcd"))
			_, stderr, code := hookRunHome(t, h.event, root, hijack, t.TempDir(), home)
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{filepath.Join(hijack, "abcd")},
				[]string{pathRefusalUnowned})
		})
	}
}

// TestSessionStartHasNoPathRungAndFailsClosed: SessionStart is the loud primary
// provisioner and resolves the plugin root only. The ownership rung changes
// nothing there — not even an abcd that path-entry vouches for is run, because
// the event has no PATH rung to reach it.
func TestSessionStartHasNoPathRungAndFailsClosed(t *testing.T) {
	command := hookCommand(t, "SessionStart")
	if strings.Contains(command, "command -v abcd") {
		t.Fatalf("SessionStart grew a PATH rung: %q", command)
	}
	root := hookRoot(t, failingBootstrap, false)
	pathDir := t.TempDir()
	home := t.TempDir()
	pathStub(t, pathDir)
	writeHookPathEntry(t, home, filepath.Join(pathDir, "abcd"))
	_, stderr, code := hookRunHome(t, "SessionStart", root, pathDir, t.TempDir(), home)
	if log := callLog(t, filepath.Join(root, "calls.log")); log != "" {
		t.Fatalf("SessionStart ran a PATH binary: %q", log)
	}
	if code == 0 || code == 127 {
		t.Fatalf("SessionStart exit = %d without a plugin-root binary; want a non-zero, non-exec-failure exit; stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "not installed") || !strings.Contains(stderr, "#install") {
		t.Fatalf("SessionStart dropped its fail-closed line: %q", stderr)
	}
}

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
	pathRefusalUnowned,
	pathRefusalUnownedRecord,
}

// pathRefusalUnownedRecord is the refusal when the RECORD ITSELF is not this
// user's word — group- or other-writable, foreign-owned, or not a regular file.
// It is separate from pathRefusalUnowned because the two say different things to
// an operator: "you never recorded this binary" versus "you recorded it, but the
// file saying so is one another local uid can rewrite" (iss-2609091927085132).
const pathRefusalUnownedRecord = "its ~/.abcd/path-entry record is not owned by you or is writable by others"

// pathRefusalUnowned is the ownership refusal — the rung's last gate and the
// one GHSA-gx3m-3224-qqcv turns on. It is spelled once here and asserted
// against the shipped manifest's own wording.
const pathRefusalUnowned = "~/.abcd/path-entry does not record it as the abcd installed here"

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
	// The guard's exit code is the whole advisory: the harness reads 0 as
	// "approved". A refused PATH binary must leave the fence at 1, and never
	// at the 0 the planted binary itself exited with.
	if h.event == "PreToolUse" && code != 1 {
		t.Fatalf("PreToolUse exit = %d after refusing the PATH binary; want exactly 1; stderr: %s", code, stderr)
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

// TestBinaryHooksRefuseAPathBinaryVouchedForByAnUnownedRecord is the first
// acceptance criterion of iss-2609091927085132. The shim carefully establishes
// that the candidate binary resolves absolutely, sits outside the working tree
// and lives in a directory that is not world-writable — and then read the file
// that NAMES that binary with no check on the file at all.
//
// A group- or other-writable path-entry is not covered by the accepted same-uid
// residual (iss-2609012039107700). It lets a DIFFERENT local uid name a binary of
// their choosing in a directory they own at mode 755: every check the shim makes
// about the binary passes, and the hook then executes it on every prompt, tool
// call and compaction. That is the case Perm()&0o022 exists to refuse, and the
// two sibling declaration files have always refused it.
func TestBinaryHooksRefuseAPathBinaryVouchedForByAnUnownedRecord(t *testing.T) {
	for _, mode := range []os.FileMode{0o664, 0o646, 0o666} {
		t.Run(mode.String(), func(t *testing.T) {
			for _, h := range binaryHooks {
				t.Run(h.event, func(t *testing.T) {
					root := hookRoot(t, failingBootstrap, false)
					pathDir := t.TempDir()
					home := t.TempDir()
					pathStub(t, pathDir)
					// The record names the binary correctly. The ONLY defect is
					// the mode of the record itself.
					writeHookPathEntry(t, home, filepath.Join(pathDir, "abcd"))
					if err := os.Chmod(filepath.Join(home, ".abcd", "path-entry"), mode); err != nil {
						t.Fatal(err)
					}
					_, stderr, code := hookRunHome(t, h.event, root, pathDir, t.TempDir(), home)
					assertPathBinaryRefused(t, h, root, stderr, code,
						[]string{filepath.Join(pathDir, "abcd")},
						[]string{pathRefusalUnownedRecord})
				})
			}
		})
	}
}

// TestBinaryHooksRefuseAPathBinaryVouchedForByASymlinkedRecord: `[ -f "$e" ]`
// FOLLOWS a symlink, so a record that is a link to a file some other uid owns
// passed the shim's only test of it. The guard judges the link as itself.
func TestBinaryHooksRefuseAPathBinaryVouchedForByASymlinkedRecord(t *testing.T) {
	for _, h := range binaryHooks {
		t.Run(h.event, func(t *testing.T) {
			root := hookRoot(t, failingBootstrap, false)
			pathDir := t.TempDir()
			home := t.TempDir()
			pathStub(t, pathDir)
			// A well-formed record, reached through a symlink at the declared
			// location — the shape whose target's owner the shim never saw.
			elsewhere := t.TempDir()
			writeHookPathEntry(t, elsewhere, filepath.Join(pathDir, "abcd"))
			if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(elsewhere, ".abcd", "path-entry"),
				filepath.Join(home, ".abcd", "path-entry")); err != nil {
				t.Fatal(err)
			}
			_, stderr, code := hookRunHome(t, h.event, root, pathDir, t.TempDir(), home)
			assertPathBinaryRefused(t, h, root, stderr, code,
				[]string{filepath.Join(pathDir, "abcd")},
				[]string{pathRefusalUnownedRecord})
		})
	}
}

// TestBinaryHooksStillRunAnOwnedPathBinaryWithATightRecord is the third
// acceptance criterion at the shim: the guard must refuse the unowned shapes and
// NOTHING else. A 0600 record — tighter than the 0644 the install writes — still
// vouches, so the documented rescue through ~/.local/bin keeps working.
func TestBinaryHooksStillRunAnOwnedPathBinaryWithATightRecord(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640, 0o644} {
		t.Run(mode.String(), func(t *testing.T) {
			for _, h := range binaryHooks {
				t.Run(h.event, func(t *testing.T) {
					root := hookRoot(t, failingBootstrap, false)
					pathDir := t.TempDir()
					home := t.TempDir()
					pathStub(t, pathDir)
					writeHookPathEntry(t, home, filepath.Join(pathDir, "abcd"))
					if err := os.Chmod(filepath.Join(home, ".abcd", "path-entry"), mode); err != nil {
						t.Fatal(err)
					}
					_, stderr, code := hookRunHome(t, h.event, root, pathDir, t.TempDir(), home)
					if code != 0 {
						t.Fatalf("%s exit = %d with a %v record it owns; stderr: %s", h.event, code, mode, stderr)
					}
					if !strings.Contains(callLog(t, filepath.Join(root, "calls.log")), h.verb) {
						t.Fatalf("%s did not run the owned PATH binary with verb %q; stderr: %s", h.event, h.verb, stderr)
					}
					if strings.Contains(stderr, "ignoring the abcd found on PATH") {
						t.Fatalf("%s refused the abcd its own %v path-entry vouches for; stderr: %s", h.event, mode, stderr)
					}
				})
			}
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
