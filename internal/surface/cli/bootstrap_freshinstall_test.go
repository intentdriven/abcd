package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
)

// The Cut A §4 manual gate — a fresh plugin install on a machine with no Go
// toolchain — FAILED on its third and fourth assertions and blocked the release
// (iss-253). No bootstrap line ever appeared at any of three session starts, the
// plugin root held a full source checkout and no binary, and every
// UserPromptSubmit and PreToolUse hook errored "No such file or directory" for
// the whole evening: the exact failure family the gate checks for zero of, found
// by a human spending an evening on it. The checks below are that evening turned
// into a gate — the §4 assertions, run on every build, against the script that
// actually ships.

// bootstrapTerminalLines returns every line that opens a terminal message — a
// success notice or a refusal. The contract is exactly one, and it must be the
// FIRST line of the run's stderr: the transcript renders only that one line
// (iss-208), which is why the script prints nothing ahead of it, loud as a
// "provisioning…" announcement would be. Anything printed first would take the
// line from the success — whose `ahoy install` instruction sits there for
// exactly this reason (iss-207) — and from a refusal's cause, which is the
// silence itd-154 exists to end.
func bootstrapTerminalLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "abcd bootstrap: ") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// bootstrapGoFreePath is the PATH the fresh-install check runs the script under:
// the base system and nothing else. The §4 promise is that a machine with NO Go
// toolchain provisions itself, so the check has to be run somewhere `go` cannot
// be found — inheriting the developer's PATH would let a Go build satisfy an
// assertion no user's machine can.
const bootstrapGoFreePath = "/usr/bin:/bin:/usr/sbin:/sbin"

// runBootstrapNoGo runs a fixture-pointed copy of the shipped script with the
// base-system PATH above, returning its merged output and exit code.
func runBootstrapNoGo(t *testing.T, root string, fx *bootstrapFixture) (string, int) {
	t.Helper()
	bootstrapRequires(t)
	if p, err := exec.LookPath("go"); err == nil && strings.HasPrefix(p, "/usr/") {
		t.Skipf("a Go toolchain sits on the base-system PATH (%s), so this host cannot stand in for a no-Go machine", p)
	}
	script := bootstrapFixtureScript(t, fx.base)
	cmd := exec.Command(script)
	cmd.Env = dedupEnvKeepLast(append([]string{
		"PATH=" + bootstrapGoFreePath,
		"HOME=" + t.TempDir(),
		"CLAUDE_PLUGIN_ROOT=" + root,
	}, fx.env()...))
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatalf("running the bootstrap: %v (output %s)", err, out)
		}
	}
	return string(out), code
}

// buildRealAbcd builds the committed cmd/abcd and returns the artefact's bytes,
// so the fresh-install check serves the binary a real release serves rather than
// a stand-in that could not answer anything. The build is the ONE thing that
// needs a Go toolchain, and it happens on the test host, never on the simulated
// no-Go machine the script then runs against.
func buildRealAbcd(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no Go toolchain on the test host to build the release artefact from")
	}
	out := filepath.Join(t.TempDir(), "abcd-artefact")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/abcd")
	cmd.Dir = bootstrapRepoFile(t, ".")
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building cmd/abcd for the fresh-install check: %v (%s)", err, msg)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading the built artefact: %v", err)
	}
	return body
}

// TestBootstrapFreshInstallSelfCheck is the §4 checklist as a gate. A fresh
// plugin root, a release that serves the real binary, and a PATH with no Go: the
// three assertions the manual gate made are made here, in order.
func TestBootstrapFreshInstallSelfCheck(t *testing.T) {
	body := buildRealAbcd(t)
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrapNoGo(t, root, fx)
	if code != 0 {
		t.Fatalf("a fresh install must succeed on a machine with no Go, got exit %d (output %q)", code, out)
	}

	// §4 assertion 3a: one bootstrap success line, and exactly one, leading.
	terminal := bootstrapTerminalLines(out)
	if len(terminal) != 1 {
		t.Fatalf("a run must have exactly one last word, got %d: %q", len(terminal), terminal)
	}
	if !strings.HasPrefix(terminal[0], "abcd bootstrap: installed") {
		t.Errorf("the one terminal line must be the success; got %q", terminal[0])
	}
	if got := firstLine(out); got != terminal[0] {
		t.Errorf("the success must be the FIRST line of stderr, the only one the transcript renders; first line = %q", got)
	}

	// §4 assertion 3b: a PROVISIONED binary at the plugin root — a regular,
	// executable file holding the verified release bytes, not a source checkout.
	binary := filepath.Join(root, "abcd")
	fi, err := os.Lstat(binary)
	if err != nil {
		t.Fatalf("the plugin root must hold a provisioned binary: %v", err)
	}
	if !fi.Mode().IsRegular() {
		t.Fatalf("the provisioned path must be a regular file, got mode %v", fi.Mode())
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("the provisioned binary must be executable, got %v", fi.Mode().Perm())
	}
	got, err := os.ReadFile(binary)
	if err != nil || string(got) != string(body) {
		t.Fatalf("the provisioned binary must be the verified release artefact (%v)", err)
	}

	// §4 assertion 4: the provisioned binary answers, with no Go anywhere, and
	// the answer IS the version report — the identity line first, then the
	// vintage and staleness fields. Naming abcd is not enough on its own: an
	// unknown-command error names it too. The gate is about provisioning, so it
	// asserts what the binary does and never how long the spawn took. A duration
	// is a property of the host, not of the code — on the macOS leg a freshly
	// written executable pays first-run code-signature validation against a cold
	// page cache — and the five-second budget that once stood here, widened for
	// exactly that wobble, failed at 5.77s on a loaded runner
	// (iss-2609012020479351, the fourth site of the class).
	cmd := exec.Command(binary, "version")
	cmd.Env = []string{"PATH=" + bootstrapGoFreePath, "HOME=" + t.TempDir()}
	answer, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the provisioned binary must answer on a machine with no Go: %v (output %s)", err, answer)
	}
	if !strings.Contains(string(answer), "abcd") {
		t.Errorf("the provisioned binary answered without naming itself: %q", answer)
	}
	name := core.NewVersion().Name
	identity := firstLine(string(answer))
	if fields := strings.Fields(identity); len(fields) != 2 || fields[0] != name {
		t.Errorf("the provisioned binary must answer with its identity line %q first, got %q (output %q)", name+" <version>", identity, answer)
	}
	for _, field := range []string{"vintage:", "staleness:"} {
		if !strings.Contains(string(answer), field) {
			t.Errorf("the provisioned binary's answer must be the version report, which carries %q; output %q", field, answer)
		}
	}
}

// TestBootstrapFailsLoudlyAndLeavesNothingHalfInstalled is AC4: a failure is
// loud, says why in the one line a reader gets, and leaves nothing
// half-installed for the hooks to trip over. A partial binary would be worse
// than none — the fast path would take it next session and every hook would fail
// on a corrupt executable instead of on an absent one.
func TestBootstrapFailsLoudlyAndLeavesNothingHalfInstalled(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("payload")
	// A manifest listing a DIFFERENT hash: the download succeeds and the
	// verification refuses — a failure at the far end of the provisioning run.
	fx := bootstrapServer(t, body, bootstrapManifest([]byte("something else entirely")))

	out, code := runBootstrapNoGo(t, root, fx)
	if code == 0 {
		t.Fatalf("a checksum mismatch must fail loudly, got exit 0 (output %q)", out)
	}
	terminal := bootstrapTerminalLines(out)
	if len(terminal) != 1 {
		t.Fatalf("a failed run must have exactly one last word, got %d: %q", len(terminal), terminal)
	}
	if got := firstLine(out); got != terminal[0] {
		t.Errorf("the refusal's cause must be the FIRST line of stderr, the only one the transcript renders; first line = %q", got)
	}
	if !strings.Contains(out, "does not match its SHA-256 checksum") {
		t.Errorf("the refusal must name the checksum mismatch; output %q", out)
	}
	if !strings.Contains(out, "shell commands run UNGUARDED") {
		t.Errorf("the refusal must say what is degraded; output %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); !os.IsNotExist(err) {
		t.Errorf("a refused install must leave NO binary in the plugin root (%v) — a half-installed one is what leaves the hooks limping", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".bootstrap.tmp.") {
			t.Errorf("a refused install left its temp directory %q behind, holding an unverified artefact", e.Name())
		}
	}
}

// TestBootstrapConvertsASilentDeathIntoARefusal is the §4 failure mode itself:
// the run announced provisioning and then simply stopped, leaving no success, no
// refusal, and hooks that error "No such file or directory" on every prompt with
// nothing anywhere saying why. A curl shim signals the script mid-run, which is
// as close to that as a test can get deterministically; the EXIT trap must turn
// it into the same loud refusal every other failing path produces.
func TestBootstrapConvertsASilentDeathIntoARefusal(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("payload")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	// The shim passes the release-tag resolve through to the real curl (the run
	// has to reach the download stage to be killed in it) and kills its parent —
	// the script — on the artefact fetch.
	shimDir := t.TempDir()
	real, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("curl unavailable")
	}
	shim := "#!/bin/sh\nfor a in \"$@\"; do\n\tcase \"$a\" in\n\t*/releases/latest) exec " + real + " \"$@\" ;;\n\tesac\ndone\nkill -TERM \"$PPID\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(shimDir, "curl"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrapShimmed(t, root, fx, shimDir)
	if code == 0 {
		t.Fatalf("a run killed mid-provision must not report success, got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "shell commands run UNGUARDED") {
		t.Fatalf("a death mid-provision must still produce the refusal, not silence; output %q", out)
	}
	if got := firstLine(out); !strings.Contains(got, "provisioning the abcd binary") {
		t.Errorf("the refusal must name what the run was doing when it died, in the one line the transcript renders; first line = %q", got)
	}
	if terminal := bootstrapTerminalLines(out); len(terminal) != 1 {
		t.Errorf("even a death gets exactly one last word, got %d: %q", len(terminal), terminal)
	}
	if _, err := os.Stat(filepath.Join(root, ".bootstrap.lock")); !os.IsNotExist(err) {
		t.Error("the lock must be released even when the run dies, or the next session cannot provision either")
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); !os.IsNotExist(err) {
		t.Error("a run that died mid-provision must leave no binary behind")
	}
	if n := atomic.LoadInt32(fx.artefactHits); n != 0 {
		t.Errorf("the shim must have intercepted the artefact fetch, got %d real download(s)", n)
	}
}

// runBootstrapShimmed runs the fixture-pointed script with shimDir FIRST on an
// otherwise base-system PATH, so a shimmed command wins over the real one.
func runBootstrapShimmed(t *testing.T, root string, fx *bootstrapFixture, shimDir string) (string, int) {
	t.Helper()
	bootstrapRequires(t)
	cmd := exec.Command(bootstrapFixtureScript(t, fx.base))
	cmd.Env = dedupEnvKeepLast(append([]string{
		"PATH=" + shimDir + ":" + bootstrapGoFreePath,
		"HOME=" + t.TempDir(),
		"CLAUDE_PLUGIN_ROOT=" + root,
	}, fx.env()...))
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatalf("running the bootstrap: %v (output %s)", err, out)
		}
	}
	return string(out), code
}
