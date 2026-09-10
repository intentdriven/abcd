package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The SessionStart wiring these tests pin exists because the harness runs every
// hook matching an event IN PARALLEL (iss-204). The previous manifest listed the
// bootstrap and the two binary-backed commands as three sibling entries of one
// event and relied on list order; spc-21 asserted that ordering was preserved,
// which is false. Both gated entries lost the race against a ~10.7 MB download,
// printed "the plugin binary is not installed", and genuinely did not run — on
// every fresh install and every plugin update, since an update lands in a fresh
// cache directory with no binary.
//
// So sequencing is owned by ONE command string: bootstrap first, then the two
// binary calls in the same shell. These tests assert that shape and the
// behaviour that only the collapsed form can have.

// sessionStartHooks is the decoded shape of the events these tests read. Only
// what is asserted is modelled; anything else in the manifest is ignored.
type sessionStartHooks struct {
	Hooks map[string][]struct {
		Hooks []struct {
			Type    string `json:"type"`
			Timeout int    `json:"timeout"`
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// hooksManifest locates the committed hooks/hooks.json from this test file's own
// on-disk position, the same way bootstrapScript locates the shipped script — so
// the assertions below derive from the manifest that actually ships.
func hooksManifest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to locate the test source file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "hooks", "hooks.json"))
}

// sessionStartCommand returns the single SessionStart command string, failing if
// the event carries anything other than exactly one entry holding exactly one
// command. The count IS the fix: a second sibling entry is a hook the harness
// would run in parallel with the bootstrap again.
func sessionStartCommand(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatalf("reading the committed hooks manifest: %v", err)
	}
	var doc sessionStartHooks
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}
	entries := doc.Hooks["SessionStart"]
	if len(entries) != 1 {
		t.Fatalf("SessionStart must declare exactly one entry group (the harness runs matching hooks in parallel), found %d", len(entries))
	}
	if len(entries[0].Hooks) != 1 {
		t.Fatalf("the SessionStart entry group must hold exactly one command (siblings race each other), found %d", len(entries[0].Hooks))
	}
	return entries[0].Hooks[0].Command
}

// sessionStartRun executes the shipped SessionStart command under `sh -c` with a
// constructed environment and stdin, returning stdout, stderr and the exit code
// SEPARATELY — the split is the point. A SessionStart hook's stdout becomes model
// context while only a non-zero exit puts its stderr in front of the human, and
// the transcript shows only the FIRST line of that stderr (iss-208).
//
// stdin is never left nil: the harness delivers the hook payload there, and a
// command that reads it must be exercised with it.
func sessionStartRun(t *testing.T, root, stdin string, extraEnv ...string) (string, string, int) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	cmd := exec.Command("sh", "-c", sessionStartCommand(t))
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append([]string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + sandboxHome(t),
		"CLAUDE_PLUGIN_ROOT=" + root,
	}, extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		e, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running the SessionStart command: %v (stderr %s)", err, stderr.String())
		}
		code = e.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}

// sessionStartRoot builds a plugin root holding a stub hooks/bootstrap.sh, and
// (when body is non-empty) a stub abcd that appends every invocation to a call
// log. The stubs stand in for the real script and binary so the WIRING is what
// is under test; hooks/bootstrap.sh's own behaviour is covered by bootstrap_test.
func sessionStartRoot(t *testing.T, bootstrap, binary string) (root, calls string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls = filepath.Join(root, "calls.log")
	if err := os.WriteFile(filepath.Join(root, "hooks", "bootstrap.sh"), []byte(bootstrap), 0o755); err != nil {
		t.Fatal(err)
	}
	if binary != "" {
		if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(binary), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, calls
}

// stubBinary stands in for the real binary, and it models the three things
// about it the chained command has to respect:
//
//   - every hook verb CONSUMES stdin whole (readHookInput is io.ReadAll over a
//     LimitReader), so two calls sharing one stdin leave the second reading EOF;
//   - `hook prompt-router-reset` writes an UNCONDITIONAL success diagnostic to
//     stderr, so whichever call runs first owns the line the transcript renders;
//   - `hook session-start` exits 0 even when it has a notice, writing the text to
//     stderr and a constant to stdout. It used to exit 2, on the belief that a
//     non-zero SessionStart puts stderr in front of the human; the harness instead
//     renders an opaque error banner and drops the text (iss-2608241115201044).
//     This stub models the current binary, so the chain's own exits are what these
//     tests exercise.
const stubBinary = `#!/bin/sh
in=$(cat)
printf '%s %s stdin=[%s]\n' "$1" "$2" "$in" >> "$ABCD_CALLS"
printf 'stdout from %s\n' "$2"
if [ "$2" = "prompt-router-reset" ]; then
	printf 'abcd rules: reset session ("SessionStart")\n' >&2
	exit 0
fi
printf 'abcd: a session-start notice\n' >&2
printf 'abcd: 1 session-start notice(s) on this hook stderr\n'
exit 0
`

// sessionStartPayload is a harness SessionStart payload of the shape
// readHookInput unmarshals. Every chained call must receive it in full.
const sessionStartPayload = `{"session_id":"s1","hook_event_name":"SessionStart","source":"startup","cwd":"/tmp"}`

// callLog returns the recorded invocations, or "" when nothing ran.
func callLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(data)
}

// firstLine is what the transcript renders of a hook's stderr.
func firstLine(s string) string {
	s = strings.TrimLeft(s, "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestSessionStartIsOneChainedCommand pins the collapse itself: one entry, one
// command, naming the bootstrap and both binary calls, and parsing as POSIX sh.
func TestSessionStartIsOneChainedCommand(t *testing.T) {
	command := sessionStartCommand(t)
	for _, want := range []string{"hooks/bootstrap.sh", "hook prompt-router-reset", "hook session-start"} {
		if !strings.Contains(command, want) {
			t.Errorf("the single SessionStart command must run %q; command = %q", want, command)
		}
	}
	if strings.Index(command, "hooks/bootstrap.sh") > strings.Index(command, "hook prompt-router-reset") {
		t.Error("the bootstrap must be invoked before the binary-backed calls in the chained command")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	script := filepath.Join(t.TempDir(), "session-start.sh")
	if err := os.WriteFile(script, []byte(command+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("sh", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("the SessionStart command is not valid POSIX sh: %v (%s)", err, out)
	}
}

// TestSessionStartLeadsWithTheBootstrapSuccess is iss-208's fix and iss-204's
// together: on a fresh install the bootstrap's success is the FIRST stderr line
// (the only one the transcript renders), and both binary-backed calls still run
// — against the binary the bootstrap has just installed.
func TestSessionStartLeadsWithTheBootstrapSuccess(t *testing.T) {
	root, calls := sessionStartRoot(t, `#!/bin/sh
cp "$CLAUDE_PLUGIN_ROOT/staged-abcd" "$CLAUDE_PLUGIN_ROOT/abcd"
chmod 0755 "$CLAUDE_PLUGIN_ROOT/abcd"
printf 'abcd bootstrap: installed the checksum-verified abcd binary\n' >&2
exit 0
`, "")
	// Staged beside the root rather than written by the stub: the binary the
	// bootstrap installs is what the chained calls must then find.
	if err := os.WriteFile(filepath.Join(root, "staged-abcd"), []byte(stubBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := sessionStartRun(t, root, sessionStartPayload, "ABCD_CALLS="+calls)

	if got := firstLine(stderr); !strings.HasPrefix(got, "abcd bootstrap: installed") {
		t.Errorf("the first stderr line must be the bootstrap's success (it is the only line the transcript shows); got %q\nfull stderr:\n%s", got, stderr)
	}
	// Exit 0 throughout now. bootstrap.sh's notice() announces a SUCCESSFUL
	// install, which is a reported condition and not a fault, so it no longer
	// raises an error banner that drops its own text (iss-2608251011427187). The
	// stub above models that, because a fixture pinning a contract the real
	// script has abandoned proves nothing.
	//
	// refuse() keeps its non-zero exit, and that split is the point: a genuine
	// fault should raise a banner, a successful install should not. What this
	// test pins either way is the TEXT and its ORDER, which the first-line
	// assertion above carries.
	if code != 0 {
		t.Errorf("a successful bootstrap plus a notice is not a fault; code = %d", code)
	}
	if strings.Contains(stderr, "the plugin binary is not installed") {
		t.Errorf("a successful bootstrap must not be followed by a missing-binary complaint; stderr:\n%s", stderr)
	}
	log := callLog(t, calls)
	for _, want := range []string{"hook prompt-router-reset", "hook session-start"} {
		if !strings.Contains(log, want) {
			t.Errorf("%q must run after the bootstrap in the same shell; call log = %q", want, log)
		}
	}
	if !strings.Contains(stdout, "stdout from prompt-router-reset") || !strings.Contains(stdout, "stdout from session-start") {
		t.Errorf("the binary calls' stdout must pass through untouched (it is the model-context channel); stdout = %q", stdout)
	}
}

// TestSessionStartSteadyStateRunsBothCalls is the boring session: the binary is
// already there, the bootstrap takes its fast path and says nothing, and both
// calls run.
//
// The first-line assertion is the one with teeth. `hook prompt-router-reset`
// ends by writing an UNCONDITIONAL "abcd rules: reset session" diagnostic to
// stderr (cli.go), so if it runs first it owns the only line the transcript
// renders and `hook session-start`'s actionable notice — "transcripts will not
// be captured", the version-skew line — is never seen. The chain therefore runs
// session-start FIRST; the exit precedence is computed from the saved codes and
// does not depend on the order.
func TestSessionStartSteadyStateRunsBothCalls(t *testing.T) {
	root, calls := sessionStartRoot(t, "#!/bin/sh\nexit 0\n", stubBinary)
	_, stderr, code := sessionStartRun(t, root, sessionStartPayload, "ABCD_CALLS="+calls)

	log := callLog(t, calls)
	if !strings.Contains(log, "hook prompt-router-reset") || !strings.Contains(log, "hook session-start") {
		t.Errorf("both binary calls must run in the steady state; call log = %q", log)
	}
	if got := firstLine(stderr); got != "abcd: a session-start notice" {
		t.Errorf("session-start's notice must be the line the transcript renders, not the reset's success diagnostic; first line = %q\nfull stderr:\n%s", got, stderr)
	}
	// A notice is not a failure, so nothing propagates a non-zero code here. The
	// first-line assertion above is what carries the coverage: the notice TEXT
	// must be the line the transcript renders.
	if code != 0 {
		t.Errorf("a notice must not surface as a hook failure; code = %d", code)
	}
}

// TestSessionStartFeedsThePayloadToEveryCall is the defect a shared stdin
// creates. Every hook verb reads its payload with io.ReadAll over the whole of
// stdin (readHookInput), so two calls chained on ONE stdin leave the second
// reading EOF: json.Unmarshal fails and `hook session-start` takes its silent
// return-nil path, permanently disabling both of its notices in every session.
// The wrapper therefore reads the payload once and feeds each call a copy.
func TestSessionStartFeedsThePayloadToEveryCall(t *testing.T) {
	root, calls := sessionStartRoot(t, "#!/bin/sh\nexit 0\n", stubBinary)
	sessionStartRun(t, root, sessionStartPayload, "ABCD_CALLS="+calls)

	log := callLog(t, calls)
	for _, verb := range []string{"prompt-router-reset", "session-start"} {
		want := "hook " + verb + " stdin=[" + sessionStartPayload + "]"
		if !strings.Contains(log, want) {
			t.Errorf("`hook %s` must receive the whole payload, not what a previous call left; call log = %q", verb, log)
		}
	}
}

// TestSessionStartReportsAMissingBinaryOnce is the genuinely-no-binary window:
// the honest failure is unchanged, but it is now said ONCE rather than by two
// sibling hooks that raced the download.
func TestSessionStartReportsAMissingBinaryOnce(t *testing.T) {
	root, calls := sessionStartRoot(t, "#!/bin/sh\nexit 0\n", "")
	_, stderr, code := sessionStartRun(t, root, sessionStartPayload, "ABCD_CALLS="+calls)

	if n := strings.Count(stderr, "the plugin binary is not installed"); n != 1 {
		t.Errorf("the missing binary must be reported exactly once, got %d; stderr:\n%s", n, stderr)
	}
	if code == 0 {
		t.Error("a missing binary must exit non-zero so the message reaches the human")
	}
	if log := callLog(t, calls); log != "" {
		t.Errorf("no binary call can have run with no binary; call log = %q", log)
	}
}

// TestSessionStartKeepsTheBootstrapRefusalFirst: when the bootstrap refuses it
// has already said what is missing and the three ways out. That message leads,
// its exit code propagates, and the wrapper adds no second complaint on top of
// it — the first line of a refusal is the one the transcript shows.
func TestSessionStartKeepsTheBootstrapRefusalFirst(t *testing.T) {
	root, calls := sessionStartRoot(t, `#!/bin/sh
printf 'abcd bootstrap: the latest release tag could not be resolved\n\nThe abcd binary is not installed in the plugin root.\n' >&2
exit 1
`, "")
	_, stderr, code := sessionStartRun(t, root, sessionStartPayload, "ABCD_CALLS="+calls)

	if got := firstLine(stderr); got != "abcd bootstrap: the latest release tag could not be resolved" {
		t.Errorf("the bootstrap's refusal must lead; first line = %q", got)
	}
	if code != 1 {
		t.Errorf("the bootstrap's refusal exit must propagate; code = %d", code)
	}
	if strings.Contains(stderr, "the plugin binary is not installed, so") {
		t.Errorf("a refusal already names the missing binary; the wrapper must not repeat it. stderr:\n%s", stderr)
	}
}

// TestSessionStartWithoutAPluginRootDoesNothing: the manifest is only ever run
// by the harness, but a command that dereferences an unset CLAUDE_PLUGIN_ROOT
// would reach for /hooks/bootstrap.sh and report a raw shell error.
func TestSessionStartWithoutAPluginRootDoesNothing(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	cmd := exec.Command("sh", "-c", sessionStartCommand(t))
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + sandboxHome(t)}
	cmd.Stdin = strings.NewReader(sessionStartPayload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Errorf("an unset CLAUDE_PLUGIN_ROOT must exit 0 and change nothing: %v (stderr %s)", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("an unset CLAUDE_PLUGIN_ROOT must say nothing; stderr = %q", stderr.String())
	}
}
