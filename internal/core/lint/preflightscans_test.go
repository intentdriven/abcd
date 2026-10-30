package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

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
	cmd := r.command(mode)
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

// TestVulncheckInterruptedLeavesNoCopyBehind: an interrupt while the fork
// point's copy is being scanned, as a terminal's Ctrl-C sends it to the whole
// process group, still removes the copy.
func TestVulncheckInterruptedLeavesNoCopyBehind(t *testing.T) {
	r := newScanRepo(t, false)
	started := filepath.Join(r.stubs, "fork-scan-started")
	r.stub("go", `if [ -f feature.txt ]; then echo 'Vulnerability #1: GO-2026-0001'; exit 3; fi
: >"`+started+`"
sleep 30`)
	cmd := r.command("vulncheck")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			t.Fatal("the fork point's scan never started")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the interrupted scan did not end")
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

func TestSecretsCheckWarnsOnAnotherVersion(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 'version is set by build process' ;; esac; exit 0`)
	code, out := r.run("secrets")
	if code != 0 {
		t.Fatalf("a clean scan: exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "8.24.3") || !strings.Contains(out, "not the") {
		t.Fatalf("another version is not warned about:\n%s", out)
	}
}

func TestSecretsCheckWithoutGitleaksSaysWhatWasNotChecked(t *testing.T) {
	r := newScanRepo(t, false)
	code, out := r.run("secrets")
	if code != 0 {
		t.Fatalf("no gitleaks: exit %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "not scanned") {
		t.Fatalf("the skip is not said loudly:\n%s", out)
	}
}
