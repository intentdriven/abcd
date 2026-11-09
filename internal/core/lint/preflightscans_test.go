package lint_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The two hosted scans preflight runs before the push (iss-2610261151381604,
// iss-2610261151481013): govulncheck over the module and gitleaks over the
// commits the branch would land, the scans that failed 12 of 32 failed CI runs
// between 2026-10-18 and 2026-10-26. Both are preflight prerequisites, so a
// finding stops the run before its build and test stages, and both
// take their versions from .github/workflows/ci.yml, so the local scan is CI's.

func TestPreflightRunsTheHostedScans(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	declared := preflightPrereqs(t, root)
	for _, gate := range []string{"vulncheck", "secrets-check"} {
		if !slices.Contains(declared, gate) {
			t.Errorf("preflight does not run %s (it declares: %s), so a finding CI's scan makes is found only after the push",
				gate, strings.Join(declared, " "))
		}
	}
}

// TestPreflightScansTakeTheirVersionsFromCI: the script names the version of
// each tool that ci.yml pins, read from ci.yml itself, so there is no second
// pin to fall behind.
func TestPreflightScansTakeTheirVersionsFromCI(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ci := readRepoFile(t, root, ".github/workflows/ci.yml")
	vuln := regexp.MustCompile(`golang\.org/x/vuln/cmd/govulncheck@(v[0-9]+\.[0-9]+\.[0-9]+)`).FindStringSubmatch(ci)
	leaks := regexp.MustCompile(`(?m)^\s*GITLEAKS_VERSION:\s*([0-9][0-9.]*)\s*$`).FindStringSubmatch(ci)
	if vuln == nil || leaks == nil {
		t.Fatal("ci.yml pins no govulncheck or no gitleaks version; the scans have nothing to match")
	}
	cmd := exec.Command("bash", filepath.Join(root, "scripts", "preflight-scans.sh"), "pins")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("preflight-scans.sh pins: %v\n%s", err, out)
	}
	if want := "govulncheck " + vuln[1] + "\ngitleaks " + leaks[1] + "\n"; string(out) != want {
		t.Fatalf("preflight-scans.sh pins printed\n%s\nwant\n%s", out, want)
	}
}

// scanRepo is a repository with a ci.yml pinning both tools, a default branch
// on origin, and a branch one commit ahead of it; stubs is a directory put
// first on the script's PATH, and tmp the script's TMPDIR.
type scanRepo struct {
	t     *testing.T
	dir   string
	stubs string
	tmp   string
}

func newScanRepo(t *testing.T, changeGoMod bool) *scanRepo {
	t.Helper()
	r := &scanRepo{t: t, dir: t.TempDir(), stubs: t.TempDir(), tmp: t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.name", "Alice Example")
	r.git("config", "user.email", "alice@example.com")
	r.git("config", "commit.gpgsign", "false")
	r.write(".github/workflows/ci.yml", "jobs:\n  gitleaks:\n    env:\n      GITLEAKS_VERSION: 8.24.3\n"+
		"  govulncheck:\n    steps:\n      - run: go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...\n")
	r.write("go.mod", "module example.com/m\n\ngo 1.26\n")
	r.git("add", ".")
	r.git("commit", "-q", "-m", "the base")
	r.git("update-ref", "refs/remotes/origin/main", "HEAD")
	r.git("checkout", "-q", "-b", "feat/x")
	if changeGoMod {
		r.write("go.mod", "module example.com/m\n\ngo 1.26\n\nrequire example.com/dep v1.2.3\n")
	} else {
		r.write("feature.txt", "the change\n")
	}
	r.git("add", ".")
	r.git("commit", "-q", "-m", "the change")
	return r
}

func (r *scanRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.dir}, args...)...)
	cmd.Env = gittest.Env(r.t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *scanRepo) write(rel, body string) {
	r.t.Helper()
	p := filepath.Join(r.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// stub writes an executable named name that records its arguments and runs
// body.
func (r *scanRepo) stub(name, body string) {
	r.t.Helper()
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"" + filepath.Join(r.stubs, name+".args") + "\"\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(r.stubs, name), []byte(script), 0o755); err != nil {
		r.t.Fatal(err)
	}
}

func (r *scanRepo) args(name string) string {
	b, _ := os.ReadFile(filepath.Join(r.stubs, name+".args"))
	return string(b)
}

// run runs one mode of the script in the repository, with the stubs first on
// PATH and nothing else of the caller's but the system tools.
func (r *scanRepo) run(mode string) (int, string) {
	r.t.Helper()
	return r.runHome(mode, r.t.TempDir())
}

// runHome is run with HOME set to home.
func (r *scanRepo) runHome(mode, home string) (int, string) {
	r.t.Helper()
	cmd := r.command(mode)
	for i, e := range cmd.Env {
		if strings.HasPrefix(e, "HOME=") {
			cmd.Env[i] = "HOME=" + home
		}
	}
	out, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), string(out)
	} else if err != nil {
		r.t.Fatalf("preflight-scans.sh %s: %v\n%s", mode, err, out)
	}
	return 0, string(out)
}

// command is the script's command for one mode, as run runs it.
func (r *scanRepo) command(mode string) *exec.Cmd {
	r.t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		r.t.Fatal(err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		r.t.Skip("git unavailable")
	}
	// git gets a directory of its own rather than its install directory on
	// PATH: a package manager's bin (/opt/homebrew/bin) also holds gitleaks,
	// which would turn the no-gitleaks case into a real scan.
	gitDir := r.t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "git"), []byte("#!/bin/sh\nexec '"+gitPath+"' \"$@\"\n"), 0o755); err != nil {
		r.t.Fatal(err)
	}
	cmd := exec.Command("bash", filepath.Join(root, "scripts", "preflight-scans.sh"), mode)
	cmd.Dir = r.dir
	cmd.Env = []string{
		"PATH=" + r.stubs + ":" + gitDir + ":/usr/bin:/bin",
		"HOME=" + r.t.TempDir(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"TMPDIR=" + r.tmp,
	}
	return cmd
}

// noCopyLeft fails the test when the script left a copy of a tree in its
// TMPDIR.
func (r *scanRepo) noCopyLeft() {
	r.t.Helper()
	left, err := os.ReadDir(r.tmp)
	if err != nil {
		r.t.Fatal(err)
	}
	for _, e := range left {
		r.t.Errorf("the scan left %s behind in its TMPDIR", e.Name())
	}
}

// govulncheckFinding is a finding as govulncheck prints one: exit 3, and a
// "Fixed in:" line per fix.
const govulncheckFinding = `echo 'Vulnerability #1: GO-2026-0001'
echo '  Standard library'
echo '    Found in: net/http@go1.26.9'
echo '    Fixed in: net/http@go1.26.10'
exit 3`

func TestVulncheckRunsCIsVersionAndPassesAClearModule(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", "echo 'No vulnerabilities found.'; exit 0")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a clear module: exit %d\n%s", code, out)
	}
	if got := r.args("go"); got != "run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...\n" {
		t.Fatalf("the scan ran %q, want CI's command", got)
	}
}

// TestVulncheckWarnsOfAFindingTheForkPointCarries: a branch that changes
// neither go.mod nor go.sum, whose fork point's tree reports the same
// vulnerability, did not bring the finding in, and govulncheck is not a
// required check, so the scan warns, says to rebase first if the default
// branch has moved (it may hold the fix), names each fix, and does not stop a
// preflight no branch cut from there could pass.
func TestVulncheckWarnsOfAFindingTheForkPointCarries(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", govulncheckFinding)
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a finding the branch did not bring in: exit %d, want 0\n%s", code, out)
	}
	for _, want := range []string{"WARNING", "origin/main", "neither go.mod nor go.sum", "fork point", "rebase first", "net/http@go1.26.10"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, out)
		}
	}
	if n := strings.Count(r.args("go"), "govulncheck@v1.7.0 ./..."); n != 2 {
		t.Errorf("the scan ran %d time(s), want twice, the head and the fork point:\n%s", n, r.args("go"))
	}
	r.noCopyLeft()
}

// TestVulncheckStopsAFindingTheBranchsOwnCodeBringsIn: govulncheck judges
// reachability, so a branch that moves no dependency can still newly reach a
// vulnerable symbol in a module the default branch already requires
// (iss-2610262259078996). The fork point's tree is scanned the same way, and
// an id it does not report stops the preflight, named; an id it does report is
// not blamed on the branch.
func TestVulncheckStopsAFindingTheBranchsOwnCodeBringsIn(t *testing.T) {
	r := newScanRepo(t, false)
	// The stub runs where the scan runs: the branch's own tree holds
	// feature.txt, the fork point's does not.
	r.stub("go", `echo 'Vulnerability #1: GO-2026-0001'
if [ -f feature.txt ]; then echo 'Vulnerability #2: GO-2026-0002'; echo '    Fixed in: example.com/dep@v1.2.4'; fi
exit 3`)
	code, out := r.run("vulncheck")
	if code != 1 {
		t.Fatalf("a finding the branch's code brings in: exit %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "WARNING") {
		t.Errorf("a finding the branch brings in is only warned about:\n%s", out)
	}
	if !strings.Contains(out, "  GO-2026-0002\n") || strings.Contains(out, "  GO-2026-0001\n") {
		t.Errorf("the stop does not name the new id alone:\n%s", out)
	}
	if !strings.Contains(out, "example.com/dep@v1.2.4") {
		t.Errorf("the stop does not name the fix:\n%s", out)
	}
	r.noCopyLeft()
}

// TestVulncheckStopsWhenTheForkPointCannotBeScanned: with no scan of the
// fork point there is nothing to show the default branch carries the
// finding, so it stops as any finding does.
func TestVulncheckStopsWhenTheForkPointCannotBeScanned(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", `if [ -f feature.txt ]; then echo 'Vulnerability #1: GO-2026-0001'; exit 3; fi
echo 'go: cannot load the module' >&2; exit 1`)
	code, out := r.run("vulncheck")
	if code != 1 {
		t.Fatalf("a finding with the fork point unscanned: exit %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "WARNING") || !strings.Contains(out, "could not be scanned") {
		t.Errorf("the unscanned fork point is not said, or the finding is only warned about:\n%s", out)
	}
	r.noCopyLeft()
}

// interruptHangGuard turns a scan that never ends into this test's failure
// rather than the whole test binary's timeout. It says nothing about how fast
// an interrupt is: nothing but the interrupt can end the scan the test blocks,
// so no length of this guard lets a scan that ignored it pass, and it races
// nothing on a slow machine. A healthy run reaches the scan in under a second
// and ends milliseconds after the signal.
const interruptHangGuard = 5 * time.Minute

// TestVulncheckInterruptedLeavesNoCopyBehind: an interrupt while the fork
// point's copy is being scanned, as a terminal's Ctrl-C sends it to the whole
// process group, ends the script through its INT trap and still removes the
// copy.
//
// The stub's scan is a cat of a FIFO this test holds open, so it cannot finish
// by itself: the script ending at all is the interrupt's doing. The signal
// waits for a reader on the FIFO, which only a process past its exec can be.
// A marker the stub's shell wrote before starting the process it blocks on left
// a window (iss-2611011153130169): the process it was meant to stop was seen to
// run on, orphaned, holding the pipe of the script's command substitution, whose
// read defers the INT trap. The likely cause, read from how shells work rather
// than observed, is a signal landing between that shell's fork and its exec.
func TestVulncheckInterruptedLeavesNoCopyBehind(t *testing.T) {
	r := newScanRepo(t, false)
	blocker := filepath.Join(r.stubs, "fork-scan-blocks-here")
	if err := syscall.Mkfifo(blocker, 0o600); err != nil {
		t.Fatal(err)
	}
	r.stub("go", `if [ -f feature.txt ]; then echo 'Vulnerability #1: GO-2026-0001'; exit 3; fi
exec cat "`+blocker+`"`)
	cmd := r.command("vulncheck")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	hung := time.After(interruptHangGuard)
	// A write-only, non-blocking open of a FIFO fails with ENXIO until it has a
	// reader. Once it succeeds, holding it keeps the reader blocked; closing it
	// when the test returns ends any reader still there.
	var held *os.File
	for held == nil {
		f, err := os.OpenFile(blocker, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			held = f
			break
		}
		if !errors.Is(err, syscall.ENXIO) {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			t.Fatal(err)
		}
		select {
		case err := <-done:
			t.Fatalf("the script ended before the fork point's scan started: %v", err)
		case <-hung:
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			t.Fatal("the fork point's scan never started")
		case <-time.After(5 * time.Millisecond):
		}
	}
	defer held.Close()
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-hung:
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the interrupted scan did not end")
	}
	if code := cmd.ProcessState.ExitCode(); code != 130 {
		t.Errorf("the interrupted scan exited %d, want 130, its INT trap's", code)
	}
	r.noCopyLeft()
}

func TestVulncheckOnABranchThatMovesADependencyBlamesNoOne(t *testing.T) {
	r := newScanRepo(t, true)
	r.stub("go", govulncheckFinding)
	code, out := r.run("vulncheck")
	if code != 1 {
		t.Fatalf("a finding: exit %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "neither go.mod nor go.sum") {
		t.Errorf("a branch that changes go.mod is told the finding is the default branch's:\n%s", out)
	}
	if !strings.Contains(out, "net/http@go1.26.10") {
		t.Errorf("the refusal does not name the fix:\n%s", out)
	}
}

// TestVulncheckThatCannotRunSaysWhatWasNotChecked: offline, the scan cannot
// fetch the vulnerability database; that is said, loudly, and does not stop
// the preflight, because CI's scan still runs.
func TestVulncheckThatCannotRunSaysWhatWasNotChecked(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", "echo 'govulncheck: fetching vulnerabilities: dial tcp: lookup vuln.go.dev: no such host' >&2; exit 1")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a scan that could not run: exit %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "not checked") {
		t.Fatalf("the skip is not said loudly:\n%s", out)
	}
}

func TestSecretsCheckScansTheBranchsOwnCommits(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.24.3 ;; esac; exit 0`)
	code, out := r.run("secrets")
	if code != 0 {
		t.Fatalf("a clean scan: exit %d\n%s", code, out)
	}
	base := r.git("merge-base", "HEAD", "refs/remotes/origin/main")
	if !strings.Contains(r.args("gitleaks"), "git --redact --no-banner --exit-code 77 --log-opts="+base+"..HEAD .") {
		t.Fatalf("gitleaks ran %q, want CI's scan over %s..HEAD", r.args("gitleaks"), base)
	}
	if strings.Contains(out, "not the") {
		t.Errorf("CI's own version is warned about:\n%s", out)
	}
}

func TestSecretsCheckRefusesAFinding(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.24.3; exit 0 ;; esac; echo 'leaks found: 1'; exit 77`)
	if code, out := r.run("secrets"); code != 1 {
		t.Fatalf("a finding: exit %d, want 1\n%s", code, out)
	}
}

// TestSecretsCheckThatCannotScanIsNotCalledAFinding: gitleaks exits 1 on its
// own errors as on a leak, so an error must not read as a secret in the
// history; it stops the preflight as the fault CI's required job would hit.
func TestSecretsCheckThatCannotScanIsNotCalledAFinding(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.24.3; exit 0 ;; esac; echo 'error: unable to read config' >&2; exit 1`)
	code, out := r.run("secrets")
	if code != 2 {
		t.Fatalf("a scan that could not run: exit %d, want 2\n%s", code, out)
	}
	if strings.Contains(out, "found a secret") || !strings.Contains(out, "could not scan") {
		t.Errorf("an error is reported as a finding, or not said:\n%s", out)
	}
}

// The secret scan judges with the gitleaks ci.yml pins (iss-2611081341084748):
// a gitleaks on PATH at another version passed a token fixture CI's 8.24.3
// refused, and the skew only warned. Now a PATH copy at another version, or
// none, is replaced by CI's release, fetched once into the abcd home and
// checked against the digest the repository holds in
// scripts/gitleaks-checksums.txt; what cannot be fetched or verified refuses.

// pinnedRelease is a release tarball whose gitleaks reports version and logs
// its arguments to log; it returns the tarball's path and sha256.
func (r *scanRepo) pinnedRelease(version, log string) (string, string) {
	r.t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"" + log + "\"\ncase \"$1\" in version) echo " + version + " ;; esac\nexit 0\n"
	for _, f := range []struct{ name, body string }{{"LICENSE", "license\n"}, {"gitleaks", body}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
			r.t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			r.t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		r.t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		r.t.Fatal(err)
	}
	p := filepath.Join(r.t.TempDir(), "release.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		r.t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return p, hex.EncodeToString(sum[:])
}

// pinDigest writes the repository's digest file with digest for every
// platform the script runs on at version.
func (r *scanRepo) pinDigest(version, digest string) {
	r.t.Helper()
	var b strings.Builder
	b.WriteString("# a comment line\n")
	for _, p := range []string{"darwin_arm64", "darwin_x64", "linux_arm64", "linux_x64"} {
		b.WriteString(digest + "  gitleaks_" + version + "_" + p + ".tar.gz\n")
	}
	r.write("scripts/gitleaks-checksums.txt", b.String())
}

// serve stubs curl to write the file at path to its -o argument, or to fail
// as curl -f does on a refused download when path is empty.
func (r *scanRepo) serve(path string) {
	r.t.Helper()
	if path == "" {
		r.stub("curl", `echo 'curl: (6) Could not resolve host' >&2; exit 6`)
		return
	}
	r.stub("curl", `while [ $# -gt 0 ]; do case "$1" in -o) out="$2"; shift ;; esac; shift; done
cat '`+path+`' >"$out"`)
}

func TestSecretsCheckFetchesCIsVersionWhenPATHHoldsAnother(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	log := filepath.Join(t.TempDir(), "pinned.args")
	tarball, digest := r.pinnedRelease("8.24.3", log)
	r.pinDigest("8.24.3", digest)
	r.serve(tarball)
	home := t.TempDir()
	code, out := r.runHome("secrets", home)
	if code != 0 {
		t.Fatalf("a clean scan with CI's version fetched: exit %d\n%s", code, out)
	}
	if strings.Contains(r.args("gitleaks"), "git ") {
		t.Errorf("the gitleaks on PATH (8.30.1) scanned the branch:\n%s", r.args("gitleaks"))
	}
	ran, _ := os.ReadFile(log)
	base := r.git("merge-base", "HEAD", "refs/remotes/origin/main")
	if !strings.Contains(string(ran), "git --redact --no-banner --exit-code 77 --log-opts="+base+"..HEAD .") {
		t.Fatalf("CI's gitleaks ran %q, want the scan over %s..HEAD\n%s", ran, base, out)
	}
	if !strings.Contains(r.args("curl"), "/releases/download/v8.24.3/gitleaks_8.24.3_") {
		t.Errorf("the fetch was not CI's release: %q", r.args("curl"))
	}
	if strings.Contains(r.args("curl"), "checksums.txt") {
		t.Errorf("the digest was taken from the network: %q", r.args("curl"))
	}
	if !strings.Contains(out, "8.30.1") || !strings.Contains(out, "8.24.3") {
		t.Errorf("the run does not say which gitleaks it set aside for which:\n%s", out)
	}
	cached, _ := filepath.Glob(abcdhome.Path(home, "gate-tools", "gitleaks", "8.24.3", "gitleaks_8.24.3_*.tar.gz"))
	if len(cached) != 1 {
		t.Fatalf("the verified release is not cached in the abcd home: %v", cached)
	}
	r.noCopyLeft()
	// A second run needs no network: the cached release is verified again.
	r.serve("")
	if code, out := r.runHome("secrets", home); code != 0 {
		t.Fatalf("a second run from the cache: exit %d\n%s", code, out)
	}
	// A cached release that no longer matches the pin is not run.
	if err := os.WriteFile(cached[0], []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := r.runHome("secrets", home); code != 2 || !strings.Contains(out, "REFUSING") {
		t.Fatalf("a tampered cache with no network: exit %d, want 2 and a refusal\n%s", code, out)
	}
}

func TestSecretsCheckRefusesADownloadThatDoesNotMatchThePin(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	log := filepath.Join(t.TempDir(), "pinned.args")
	tarball, _ := r.pinnedRelease("8.24.3", log)
	r.pinDigest("8.24.3", strings.Repeat("0", 64))
	r.serve(tarball)
	home := t.TempDir()
	code, out := r.runHome("secrets", home)
	if code != 2 {
		t.Fatalf("an unverified download: exit %d, want 2\n%s", code, out)
	}
	if !strings.Contains(out, "REFUSING") || !strings.Contains(out, "8.30.1") || !strings.Contains(out, "8.24.3") {
		t.Errorf("the refusal does not name the skew:\n%s", out)
	}
	if ran, _ := os.ReadFile(log); len(ran) != 0 {
		t.Errorf("the unverified binary ran: %q", ran)
	}
	if strings.Contains(r.args("gitleaks"), "git ") {
		t.Errorf("the scan fell back to the gitleaks on PATH:\n%s", r.args("gitleaks"))
	}
	if left, _ := filepath.Glob(abcdhome.Path(home, "gate-tools", "gitleaks", "8.24.3", "*")); len(left) != 0 {
		t.Errorf("an unverified download was kept: %v", left)
	}
	r.noCopyLeft()
}

func TestSecretsCheckRefusesWhenCIsVersionCannotBeFetched(t *testing.T) {
	for name, path := range map[string]string{"another version": `case "$1" in version) echo 8.30.1 ;; esac; exit 0`, "none": ""} {
		t.Run(name, func(t *testing.T) {
			r := newScanRepo(t, false)
			if path != "" {
				r.stub("gitleaks", path)
			}
			r.pinDigest("8.24.3", strings.Repeat("a", 64))
			r.serve("")
			code, out := r.run("secrets")
			if code != 2 || !strings.Contains(out, "REFUSING") || !strings.Contains(out, "8.24.3") {
				t.Fatalf("no network: exit %d, want 2 and a refusal naming CI's version\n%s", code, out)
			}
			if strings.Contains(r.args("gitleaks"), "git ") {
				t.Errorf("the scan fell back to the gitleaks on PATH:\n%s", r.args("gitleaks"))
			}
		})
	}
}

func TestSecretsCheckRefusesWithoutAPinnedDigest(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	r.pinDigest("8.20.0", strings.Repeat("a", 64))
	tarball, _ := r.pinnedRelease("8.24.3", filepath.Join(t.TempDir(), "pinned.args"))
	r.serve(tarball)
	code, out := r.run("secrets")
	if code != 2 || !strings.Contains(out, "REFUSING") || !strings.Contains(out, "scripts/gitleaks-checksums.txt") {
		t.Fatalf("no digest for CI's version: exit %d, want 2 and a refusal naming the digest file\n%s", code, out)
	}
	if r.args("curl") != "" {
		t.Errorf("a release with no pinned digest was fetched: %q", r.args("curl"))
	}
}

// TestGitleaksDigestsCoverCIsPin: the digest file holds a digest for every
// platform the script fetches for, at the version ci.yml pins, and CI's own
// install checks its download against that file rather than against a
// checksum list fetched beside it, so a version bump that leaves the digests
// behind fails here and in CI's install, not first on a developer's machine.
func TestGitleaksDigestsCoverCIsPin(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ci := readRepoFile(t, root, ".github/workflows/ci.yml")
	pin := regexp.MustCompile(`(?m)^\s*GITLEAKS_VERSION:\s*([0-9][0-9.]*)\s*$`).FindStringSubmatch(ci)
	if pin == nil {
		t.Fatal("ci.yml pins no gitleaks version")
	}
	sums := readRepoFile(t, root, "scripts/gitleaks-checksums.txt")
	for _, p := range []string{"darwin_arm64", "darwin_x64", "linux_arm64", "linux_x64"} {
		asset := "gitleaks_" + pin[1] + "_" + p + ".tar.gz"
		if !regexp.MustCompile(`(?m)^[0-9a-f]{64}  ` + regexp.QuoteMeta(asset) + `$`).MatchString(sums) {
			t.Errorf("scripts/gitleaks-checksums.txt holds no sha256 for %s, the release ci.yml pins", asset)
		}
	}
	if !strings.Contains(ci, "scripts/gitleaks-checksums.txt") || strings.Contains(ci, "_checksums.txt\"") {
		t.Error("ci.yml's gitleaks install does not verify against scripts/gitleaks-checksums.txt alone")
	}
}

// keptRelease runs one clean scan that fetches CI's release into home's abcd
// home, then takes the network away, so a later run can only use what was
// kept. It returns the kept directory and the pinned binary's argument log.
func (r *scanRepo) keptRelease(home string) (string, string) {
	r.t.Helper()
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	log := filepath.Join(r.t.TempDir(), "pinned.args")
	tarball, digest := r.pinnedRelease("8.24.3", log)
	r.pinDigest("8.24.3", digest)
	r.serve(tarball)
	if code, out := r.runHome("secrets", home); code != 0 {
		r.t.Fatalf("the first scan, which keeps CI's release: exit %d\n%s", code, out)
	}
	r.serve("")
	if err := os.Remove(log); err != nil {
		r.t.Fatal(err)
	}
	return abcdhome.Path(home, "gate-tools", "gitleaks", "8.24.3"), log
}

// keptArchive is the one tarball in the kept directory.
func keptArchive(t *testing.T, cache string) string {
	t.Helper()
	kept, _ := filepath.Glob(filepath.Join(cache, "gitleaks_8.24.3_*.tar.gz"))
	if len(kept) != 1 {
		t.Fatalf("the kept directory holds %v, want one release", kept)
	}
	return kept[0]
}

// refusedKept asserts a run refused naming path, and ran no pinned gitleaks.
func (r *scanRepo) refusedKept(home, log, path string) {
	r.t.Helper()
	code, out := r.runHome("secrets", home)
	if code != 2 || !strings.Contains(out, "REFUSING") || !strings.Contains(out, path) {
		r.t.Fatalf("exit %d, want 2 and a refusal naming %s\n%s", code, path, out)
	}
	if ran, _ := os.ReadFile(log); len(ran) != 0 {
		r.t.Errorf("CI's gitleaks ran from a kept copy that was refused: %q", ran)
	}
	if strings.Contains(r.args("gitleaks"), "git ") {
		r.t.Errorf("the scan fell back to the gitleaks on PATH:\n%s", r.args("gitleaks"))
	}
	r.noCopyLeft()
}

// TestSecretsCheckKeepsCIsReleasePrivate: the kept directory is made 0700,
// and a kept directory or tarball another user owns, or anyone else can write,
// or that is a symlink, is refused by name rather than verified and run.
func TestSecretsCheckKeepsCIsReleasePrivate(t *testing.T) {
	t.Run("the kept directory is made 0700", func(t *testing.T) {
		r := newScanRepo(t, false)
		cache, _ := r.keptRelease(t.TempDir())
		st, err := os.Stat(cache)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0o700 {
			t.Errorf("the kept directory is %o, want 700", st.Mode().Perm())
		}
	})
	t.Run("a group-writable directory", func(t *testing.T) {
		r := newScanRepo(t, false)
		home := t.TempDir()
		cache, log := r.keptRelease(home)
		if err := os.Chmod(cache, 0o770); err != nil {
			t.Fatal(err)
		}
		r.refusedKept(home, log, cache)
	})
	t.Run("an others-writable tarball", func(t *testing.T) {
		r := newScanRepo(t, false)
		home := t.TempDir()
		cache, log := r.keptRelease(home)
		archive := keptArchive(t, cache)
		if err := os.Chmod(archive, 0o602); err != nil {
			t.Fatal(err)
		}
		r.refusedKept(home, log, archive)
	})
	t.Run("another user's directory", func(t *testing.T) {
		r := newScanRepo(t, false)
		home := t.TempDir()
		cache, log := r.keptRelease(home)
		// The script asks id who it runs as; an answer that is not the
		// directory's owner makes the kept copy another user's.
		r.stub("id", `case "$1" in -u) echo 4242424 ;; *) exec /usr/bin/id "$@" ;; esac`)
		r.refusedKept(home, log, cache)
	})
	t.Run("a symlinked directory", func(t *testing.T) {
		r := newScanRepo(t, false)
		home := t.TempDir()
		cache, log := r.keptRelease(home)
		elsewhere := filepath.Join(t.TempDir(), "elsewhere")
		if err := os.Rename(cache, elsewhere); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, cache); err != nil {
			t.Fatal(err)
		}
		r.refusedKept(home, log, cache)
	})
	t.Run("a symlinked tarball", func(t *testing.T) {
		r := newScanRepo(t, false)
		home := t.TempDir()
		cache, log := r.keptRelease(home)
		archive := keptArchive(t, cache)
		elsewhere := filepath.Join(t.TempDir(), "release.tar.gz")
		if err := os.Rename(archive, elsewhere); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, archive); err != nil {
			t.Fatal(err)
		}
		r.refusedKept(home, log, archive)
	})
}

// TestSecretsCheckVerifiesAndUnpacksOnePrivateCopy: the tarball that is
// hashed and the one unpacked are the same copy, in the script's own
// temporary directory, never the kept file in the abcd home.
func TestSecretsCheckVerifiesAndUnpacksOnePrivateCopy(t *testing.T) {
	r := newScanRepo(t, false)
	home := t.TempDir()
	r.keptRelease(home)
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		t.Skip("tar unavailable")
	}
	r.stub("tar", `exec '`+tarPath+`' "$@"`)
	r.stub("sha256sum", `if command -v shasum >/dev/null 2>&1; then exec shasum -a 256 "$@"; fi; exec /usr/bin/sha256sum "$@"`)
	if code, out := r.runHome("secrets", home); code != 0 {
		t.Fatalf("a scan from the kept release: exit %d\n%s", code, out)
	}
	hashed, unpacked := strings.TrimSpace(r.args("sha256sum")), r.args("tar")
	if !strings.HasPrefix(hashed, r.tmp+"/") {
		t.Errorf("the hash was taken of %q, not of a copy in the script's TMPDIR %s", hashed, r.tmp)
	}
	if !strings.Contains(unpacked, "-xzf "+hashed+" ") {
		t.Errorf("tar unpacked %q, not the copy that was hashed, %s", unpacked, hashed)
	}
	r.noCopyLeft()
}

// TestSecretsCheckRefusesWithNoHasher: with neither sha256sum nor shasum
// there is no way to verify CI's release, so the scan refuses with exit 2 and
// says so, before fetching anything.
func TestSecretsCheckRefusesWithNoHasher(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	tarball, digest := r.pinnedRelease("8.24.3", filepath.Join(t.TempDir(), "pinned.args"))
	r.pinDigest("8.24.3", digest)
	r.serve(tarball)
	// The system's tools, every one but the two hashers.
	sys := t.TempDir()
	for _, d := range []string{"/usr/bin", "/bin"} {
		entries, _ := os.ReadDir(d)
		for _, e := range entries {
			name := e.Name()
			if name == "sha256sum" || name == "shasum" {
				continue
			}
			if _, err := os.Lstat(filepath.Join(sys, name)); err == nil {
				continue
			}
			if err := os.Symlink(filepath.Join(d, name), filepath.Join(sys, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	cmd := r.command("secrets")
	for i, e := range cmd.Env {
		if strings.HasPrefix(e, "PATH=") {
			cmd.Env[i] = strings.Replace(e, ":/usr/bin:/bin", ":"+sys, 1)
		}
	}
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code != 2 || !strings.Contains(string(out), "REFUSING") || !strings.Contains(string(out), "sha256sum") {
		t.Fatalf("no hasher: exit %d, want 2 and a refusal naming the missing tool\n%s", code, out)
	}
	if r.args("curl") != "" {
		t.Errorf("a release was fetched with no way to verify it: %q", r.args("curl"))
	}
}

// TestSecretsCheckMatchesTheAssetNameLiterally: a digest line for a name that
// differs from the asset only where the asset has a dot is not its digest.
func TestSecretsCheckMatchesTheAssetNameLiterally(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	tarball, digest := r.pinnedRelease("8.24.3", filepath.Join(t.TempDir(), "pinned.args"))
	var b strings.Builder
	for _, p := range []string{"darwin_arm64", "darwin_x64", "linux_arm64", "linux_x64"} {
		b.WriteString(digest + "  gitleaks_8x24x3_" + p + "xtarxgz\n")
	}
	r.write("scripts/gitleaks-checksums.txt", b.String())
	r.serve(tarball)
	code, out := r.run("secrets")
	if code != 2 || !strings.Contains(out, "holds no single sha256") {
		t.Fatalf("a near-miss name: exit %d, want 2 and a refusal for no digest\n%s", code, out)
	}
	if r.args("curl") != "" {
		t.Errorf("a near-miss line was taken as the asset's digest, and the release fetched: %q", r.args("curl"))
	}
}

// TestCIsGitleaksInstallMatchesTheAssetNameLiterally: ci.yml picks the
// digest line by the whole asset name, never by a pattern whose dots match
// any character.
func TestCIsGitleaksInstallMatchesTheAssetNameLiterally(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ci := readRepoFile(t, root, ".github/workflows/ci.yml")
	if strings.Contains(ci, `grep "  $base\$"`) || !strings.Contains(ci, `awk -v a="$base" 'NF == 2 && $2 == a'`) {
		t.Error("ci.yml's gitleaks install does not pick its digest line by the whole asset name")
	}
}
