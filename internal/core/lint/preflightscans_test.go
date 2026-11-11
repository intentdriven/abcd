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

// stubGovulncheck makes govulncheck a tool that runs body, behind a go stub
// that treats it as the real go does: `go install <pkg>@<version>` writes the
// tool into $GOBIN, and `go run <pkg>@<version> <args>` runs it and turns any
// exit but 0 into exit 1 with an "exit status N" line, so a finding (exit 3)
// reaches a `go run` caller as the same exit 1 an offline run gives. A stub
// that passed the tool's exit straight through hid that the script read every
// finding as a skip. The tool logs its arguments as "govulncheck.args".
//
// The install arm also holds where the binary is built: only in a private
// directory of the script's own, under TMPDIR, a directory, mode 0700, which is
// what mktemp -d makes. A fixed path, one outside TMPDIR or one others can
// write is a place another account could put a binary first; the stub refuses
// each, so the scan is skipped and every test expecting a result fails.
func (r *scanRepo) stubGovulncheck(body string) {
	r.t.Helper()
	tool := filepath.Join(r.stubs, "govulncheck.tool")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"" + filepath.Join(r.stubs, "govulncheck.args") + "\"\n" + body + "\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		r.t.Fatal(err)
	}
	r.stub("go", `case "$1" in
install)
	case "$GOBIN" in /*) ;; *) echo 'go: GOBIN is not an absolute path' >&2; exit 1 ;; esac
	g="$(cd "$GOBIN" 2>/dev/null && pwd -P)" && t="$(cd "${TMPDIR:?}" 2>/dev/null && pwd -P)" ||
		{ echo "go stub: GOBIN $GOBIN or TMPDIR $TMPDIR names no directory" >&2; exit 2; }
	case "$g" in "$t"/*) ;; *) echo "go stub: GOBIN $GOBIN is not under TMPDIR $TMPDIR" >&2; exit 2 ;; esac
	[ -n "$(find "$GOBIN" -prune -type d -perm 700 -print)" ] ||
		{ echo "go stub: GOBIN $GOBIN is not a directory of mode 0700" >&2; exit 2; }
	cat "`+tool+`" >"$GOBIN/govulncheck" && chmod 755 "$GOBIN/govulncheck"
	;;
# The script no longer calls go run. This arm stays because it is what makes
# TestVulncheckFindingIsJudgedNotSkipped fail against a script that goes back
# to go run: like the real go run, it turns the finding's exit 3 into exit 1.
run)
	shift 2
	"`+tool+`" "$@"
	rc=$?
	[ "$rc" -eq 0 ] || { echo "exit status $rc" >&2; exit 1; }
	;;
*) echo "go stub: unexpected $*" >&2; exit 2 ;;
esac`)
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

// runTMPDIR is run with TMPDIR set to tmpdir, which may be relative to the
// repository.
func (r *scanRepo) runTMPDIR(mode, tmpdir string) (int, string) {
	r.t.Helper()
	cmd := r.command(mode)
	for i, e := range cmd.Env {
		if strings.HasPrefix(e, "TMPDIR=") {
			cmd.Env[i] = "TMPDIR=" + tmpdir
		}
	}
	out, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), string(out)
	} else if err != nil {
		r.t.Fatal(err)
	}
	return 0, string(out)
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
	r.stubGovulncheck("echo 'No vulnerabilities found.'; exit 0")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a clear module: exit %d\n%s", code, out)
	}
	if got := r.args("go"); got != "install golang.org/x/vuln/cmd/govulncheck@v1.7.0\n" {
		t.Fatalf("go ran %q, want CI's govulncheck version built", got)
	}
	if got := r.args("govulncheck"); got != "./...\n" {
		t.Fatalf("the scan ran %q, want CI's scan of ./...", got)
	}
	r.noCopyLeft()
}

// TestVulncheckFindingIsJudgedNotSkipped: govulncheck exits 3 on a finding,
// and `go run` turns that into exit 1, the same exit an offline run gives, so a
// scan run through `go run` reported every real finding as SKIPPED and the
// preflight passed while CI's scan failed. The tool runs as its own binary, so
// its finding exit reaches the judgement.
func TestVulncheckFindingIsJudgedNotSkipped(t *testing.T) {
	r := newScanRepo(t, true)
	r.stubGovulncheck(govulncheckFinding)
	code, out := r.run("vulncheck")
	if code != 1 {
		t.Fatalf("a finding: exit %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "SKIPPED") || strings.Contains(out, "could not run") {
		t.Errorf("a finding is reported as a scan that could not run:\n%s", out)
	}
	if !strings.Contains(out, "found a known vulnerability") || !strings.Contains(out, "GO-2026-0001") {
		t.Errorf("the finding is not reported:\n%s", out)
	}
	r.noCopyLeft()
}

// TestVulncheckThatCannotBeBuiltIsSkippedLoudly: offline on a machine that
// has never built CI's govulncheck, go cannot fetch its module; that is a
// scan that could not run, said loudly, not a finding.
func TestVulncheckThatCannotBeBuiltIsSkippedLoudly(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", "echo 'go: golang.org/x/vuln/cmd/govulncheck@v1.7.0: dial tcp: lookup proxy.golang.org: no such host' >&2; exit 1")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a govulncheck that could not be built: exit %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "not checked") || !strings.Contains(out, "proxy.golang.org") {
		t.Fatalf("the skip is not said loudly, with go's reason:\n%s", out)
	}
	// The cause is the build, not a database the tool never reached.
	if !strings.Contains(out, "could not be built (go install exit 1)") || strings.Contains(out, "vulnerability database") {
		t.Errorf("the skip does not say the build failed, or blames the database:\n%s", out)
	}
	r.noCopyLeft()
}

// TestVulncheckBuildThatLeavesNoBinaryIsSkippedLoudly: a go install that
// exits 0 and writes nothing leaves no tool to run; that is said as a build
// that failed, not as a binary that could not run.
func TestVulncheckBuildThatLeavesNoBinaryIsSkippedLoudly(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("go", "exit 0")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a build that left no binary: exit %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "go install exited 0 and left no govulncheck") {
		t.Fatalf("the skip does not say the build left no binary:\n%s", out)
	}
	if strings.Contains(out, "could not be executed") || strings.Contains(out, "no known vulnerability") {
		t.Errorf("a missing binary is reported as one that ran, or as clean:\n%s", out)
	}
	r.noCopyLeft()
}

// TestVulncheckBuildThatFailsItsChecksumRefuses: a module go fetched for the
// tool that does not match the checksum database, or an altered copy in the
// module cache, is not a machine that is offline. What would run is not what
// CI runs, so the scan refuses (exit 2), as a gitleaks release that fails its
// sha256 check does, and never reads as a skip. The match ignores case: the
// checksum database's client words a server that fails its own proofs in
// lower case ("security error: misbehaving server").
func TestVulncheckBuildThatFailsItsChecksumRefuses(t *testing.T) {
	for name, said := range map[string]string{
		"the checksum database":          "verifying golang.org/x/vuln@v1.7.0/go.mod: checksum mismatch\n\tdownloaded: h1:AAAA\n\tsum.golang.org: h1:BBBB\n\nSECURITY ERROR\nThis download does NOT match the one reported by the checksum server.",
		"the module cache":               "verifying golang.org/x/vuln@v1.7.0: checksum mismatch",
		"the checksum database's proofs": "verifying golang.org/x/vuln@v1.7.0/go.mod: security error: misbehaving server",
	} {
		t.Run(name, func(t *testing.T) {
			r := newScanRepo(t, false)
			r.stub("go", "printf '"+said+"\\n' >&2; exit 1")
			code, out := r.run("vulncheck")
			if code != 2 {
				t.Fatalf("a build that failed its checksum: exit %d, want 2\n%s", code, out)
			}
			if !strings.Contains(out, "REFUSING") || !strings.Contains(out, "checksum verification") || !strings.Contains(out, strings.SplitN(said, "\n", 2)[0]) {
				t.Errorf("the refusal does not say the checksum failed, with go's lines:\n%s", out)
			}
			if strings.Contains(out, "SKIPPED") {
				t.Errorf("a failed checksum reads as a skip:\n%s", out)
			}
			r.noCopyLeft()
		})
	}
}

// TestVulncheckChecksumDatabaseUnreachableIsSkipped: offline, or with the
// checksum database answering a lookup with 404 or 410, go cannot verify a
// download, but nothing failed a verification: that is a build that could not
// be done, skipped loudly, and never the refusal a failed checksum earns.
func TestVulncheckChecksumDatabaseUnreachableIsSkipped(t *testing.T) {
	for name, said := range map[string]string{
		"no such host": `verifying golang.org/x/vuln@v1.7.0: golang.org/x/vuln@v1.7.0: Get "https://sum.golang.org/lookup/golang.org/x/vuln@v1.7.0": dial tcp: lookup sum.golang.org: no such host`,
		"404":          "verifying golang.org/x/vuln@v1.7.0: golang.org/x/vuln@v1.7.0: reading https://sum.golang.org/lookup/golang.org/x/vuln@v1.7.0: 404 Not Found",
		"410":          "verifying golang.org/x/vuln@v1.7.0/go.mod: golang.org/x/vuln@v1.7.0/go.mod: reading https://sum.golang.org/tile/8/0/x123/456: 410 Gone",
	} {
		t.Run(name, func(t *testing.T) {
			r := newScanRepo(t, false)
			r.stub("go", "printf '%s\\n' '"+said+"' >&2; exit 1")
			code, out := r.run("vulncheck")
			if code != 0 {
				t.Fatalf("a checksum database go could not reach: exit %d, want 0\n%s", code, out)
			}
			if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "could not be built (go install exit 1)") || !strings.Contains(out, "sum.golang.org") {
				t.Errorf("the skip is not said loudly, with go's lines:\n%s", out)
			}
			if strings.Contains(out, "REFUSING") {
				t.Errorf("an unreachable checksum database reads as a failed checksum:\n%s", out)
			}
			r.noCopyLeft()
		})
	}
}

// TestVulncheckThatCannotFinishSaysWhy: a built binary the system would not
// execute, or one a signal ended, gave no result; each is skipped loudly with
// its own cause and its exit, and never read as clean.
func TestVulncheckThatCannotFinishSaysWhy(t *testing.T) {
	for _, c := range []struct {
		name, body string
		want       []string
	}{
		{"ended by a signal", "echo 'Scanning your code'; kill -KILL $$", []string{"ended by signal 9 (exit 137)", "Nothing it printed is a result"}},
		{"not executable", "echo 'permission denied' >&2; exit 126", []string{"could not be executed (exit 126)", "noexec"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newScanRepo(t, false)
			r.stubGovulncheck(c.body)
			code, out := r.run("vulncheck")
			if code != 0 {
				t.Fatalf("%s: exit %d, want 0\n%s", c.name, code, out)
			}
			if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "not checked") {
				t.Errorf("%s is not skipped loudly:\n%s", c.name, out)
			}
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("%s: the skip does not say %q:\n%s", c.name, w, out)
				}
			}
			if strings.Contains(out, "no known vulnerability") || strings.Contains(out, "vulnerability database") {
				t.Errorf("%s is read as clean, or blamed on the database:\n%s", c.name, out)
			}
			r.noCopyLeft()
		})
	}
}

// TestVulncheckWithARelativeTMPDIRStillScans: go install refuses a relative
// GOBIN, so a relative TMPDIR is made absolute rather than skipping the scan
// on every run; a TMPDIR that names no directory refuses, naming it.
func TestVulncheckWithARelativeTMPDIRStillScans(t *testing.T) {
	r := newScanRepo(t, false)
	r.stubGovulncheck("echo 'No vulnerabilities found.'; exit 0")
	rel := filepath.Join(r.dir, "scratch-tmp")
	if err := os.Mkdir(rel, 0o700); err != nil {
		t.Fatal(err)
	}
	code, out := r.runTMPDIR("vulncheck", "scratch-tmp")
	if code != 0 || !strings.Contains(out, "no known vulnerability") {
		t.Fatalf("a relative TMPDIR: exit %d, want a clean scan\n%s", code, out)
	}
	if left, err := os.ReadDir(rel); err != nil || len(left) != 0 {
		t.Errorf("the scan left %v behind in its relative TMPDIR (%v)", left, err)
	}
	code, out = r.runTMPDIR("vulncheck", "no-such-dir")
	if code != 2 || !strings.Contains(out, "TMPDIR is no-such-dir") {
		t.Errorf("a relative TMPDIR naming no directory: exit %d, want 2 naming it\n%s", code, out)
	}
}

// TestVulncheckWithARelativeTMPDIRScansTheForkPoint: the fork point's scan
// runs the tool from inside the extracted tree, where a relative TMPDIR names
// no directory. The absolute one is what the tool is handed, so a tool that
// needs a temporary file there still scans the fork point, and a finding the
// default branch carries too warns rather than stops.
func TestVulncheckWithARelativeTMPDIRScansTheForkPoint(t *testing.T) {
	r := newScanRepo(t, false)
	r.stubGovulncheck(`[ -d "${TMPDIR:?}" ] || { echo "govulncheck: TMPDIR $TMPDIR names no directory" >&2; exit 1; }
` + govulncheckFinding)
	rel := filepath.Join(r.dir, "scratch-tmp")
	if err := os.Mkdir(rel, 0o700); err != nil {
		t.Fatal(err)
	}
	code, out := r.runTMPDIR("vulncheck", "scratch-tmp")
	if code != 0 || !strings.Contains(out, "WARNING") {
		t.Fatalf("a finding the fork point carries, under a relative TMPDIR: exit %d, want 0 and a warning\n%s", code, out)
	}
	if n := strings.Count(r.args("govulncheck"), "./..."); n != 2 {
		t.Errorf("the scan ran %d time(s), want twice, the head and the fork point", n)
	}
	if left, err := os.ReadDir(rel); err != nil || len(left) != 0 {
		t.Errorf("the scan left %v behind in its relative TMPDIR (%v)", left, err)
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
	r.stubGovulncheck(govulncheckFinding)
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a finding the branch did not bring in: exit %d, want 0\n%s", code, out)
	}
	for _, want := range []string{"WARNING", "origin/main", "neither go.mod nor go.sum", "fork point", "rebase first", "net/http@go1.26.10"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, out)
		}
	}
	if n := strings.Count(r.args("govulncheck"), "./..."); n != 2 {
		t.Errorf("the scan ran %d time(s), want twice, the head and the fork point:\n%s", n, r.args("govulncheck"))
	}
	if n := strings.Count(r.args("go"), "install golang.org/x/vuln/cmd/govulncheck@v1.7.0"); n != 1 {
		t.Errorf("CI's govulncheck was built %d time(s), want once for both scans:\n%s", n, r.args("go"))
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
	r.stubGovulncheck(`echo 'Vulnerability #1: GO-2026-0001'
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
	r.stubGovulncheck(`if [ -f feature.txt ]; then echo 'Vulnerability #1: GO-2026-0001'; exit 3; fi
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
//
// A hangup (a closed terminal) takes the same route through the HUP trap: a
// signal with no trap ends the script without its EXIT trap, leaving the copy
// and the private directory the tool was built in.
//
// A second signal while the removals run changes nothing
// (iss-2611111850449220): each signal trap ignores the three signals before it
// exits. Without that the second signal took the same trap, which exited at
// once with the first trap's code and left whatever was not yet removed. The
// last case sends that second hangup from inside the first removal: rm is a
// stub that signals the script, its parent, once, and then removes as asked.
func TestVulncheckInterruptedLeavesNoCopyBehind(t *testing.T) {
	for _, c := range []struct {
		name  string
		sig   syscall.Signal
		want  int
		again bool
	}{
		{"INT", syscall.SIGINT, 130, false},
		{"HUP", syscall.SIGHUP, 129, false},
		{"HUP, and again as the removals start", syscall.SIGHUP, 129, true},
	} {
		t.Run(c.name, func(t *testing.T) { interruptedScanLeavesNoCopy(t, c.sig, c.want, c.again) })
	}
}

func interruptedScanLeavesNoCopy(t *testing.T, sig syscall.Signal, want int, again bool) {
	r := newScanRepo(t, false)
	if again {
		rmPath, err := exec.LookPath("rm")
		if err != nil {
			t.Skip("rm unavailable")
		}
		sent := filepath.Join(r.stubs, "rm-signalled-its-parent")
		r.stub("rm", `if [ ! -e "`+sent+`" ]; then : >"`+sent+`"; kill -HUP "$PPID"; fi
exec '`+rmPath+`' "$@"`)
		t.Cleanup(func() {
			if _, err := os.Stat(sent); err != nil {
				t.Errorf("the removals never ran through the rm stub, so no second signal was sent: %v", err)
			}
		})
	}
	blocker := filepath.Join(r.stubs, "fork-scan-blocks-here")
	if err := syscall.Mkfifo(blocker, 0o600); err != nil {
		t.Fatal(err)
	}
	r.stubGovulncheck(`if [ -f feature.txt ]; then echo 'Vulnerability #1: GO-2026-0001'; exit 3; fi
exec cat "` + blocker + `"`)
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
	if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-hung:
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the interrupted scan did not end")
	}
	if code := cmd.ProcessState.ExitCode(); code != want {
		t.Errorf("the scan sent %v exited %d, want %d, its trap's", sig, code, want)
	}
	r.noCopyLeft()
}

func TestVulncheckOnABranchThatMovesADependencyBlamesNoOne(t *testing.T) {
	r := newScanRepo(t, true)
	r.stubGovulncheck(govulncheckFinding)
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
	r.noCopyLeft()
}

// TestVulncheckThatCannotRunSaysWhatWasNotChecked: offline, the scan cannot
// fetch the vulnerability database; that is said, loudly, and does not stop
// the preflight, because CI's scan still runs.
func TestVulncheckThatCannotRunSaysWhatWasNotChecked(t *testing.T) {
	r := newScanRepo(t, false)
	r.stubGovulncheck("echo 'govulncheck: fetching vulnerabilities: dial tcp: lookup vuln.go.dev: no such host' >&2; exit 1")
	code, out := r.run("vulncheck")
	if code != 0 {
		t.Fatalf("a scan that could not run: exit %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "SKIPPED") || !strings.Contains(out, "not checked") {
		t.Fatalf("the skip is not said loudly:\n%s", out)
	}
	if !strings.Contains(out, "ran and stopped with exit 1") || !strings.Contains(out, "vulnerability database") {
		t.Errorf("the skip does not say the tool ran and could not reach its database:\n%s", out)
	}
	r.noCopyLeft()
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
	return r.pinnedReleaseMode(version, log, 0o755)
}

// pinnedReleaseMode is pinnedRelease with the gitleaks in it unpacked at mode:
// one without execute bits stands in for a TMPDIR mounted noexec, where the
// shell refuses to run the binary with exit 126.
func (r *scanRepo) pinnedReleaseMode(version, log string, mode int64) (string, string) {
	r.t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"" + log + "\"\ncase \"$1\" in version) echo " + version + " ;; esac\nexit 0\n"
	for _, f := range []struct{ name, body string }{{"LICENSE", "license\n"}, {"gitleaks", body}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: mode, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
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

// A verified gitleaks the system will not execute, as from a TMPDIR mounted
// noexec, reports no version: the refusal says the binary could not be
// executed, names the temporary directory and the TMPDIR to set, rather than
// calling the verified release one that reports nothing.
func TestSecretsCheckThatCannotExecuteTheVerifiedBinarySaysSo(t *testing.T) {
	r := newScanRepo(t, false)
	r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
	log := filepath.Join(t.TempDir(), "pinned.args")
	tarball, digest := r.pinnedReleaseMode("8.24.3", log, 0o644)
	r.pinDigest("8.24.3", digest)
	r.serve(tarball)
	code, out := r.run("secrets")
	if code != 2 || !strings.Contains(out, "REFUSING") {
		t.Fatalf("a verified binary that cannot be executed: exit %d, want 2 and a refusal\n%s", code, out)
	}
	if strings.Contains(out, "reports 'nothing'") {
		t.Errorf("the refusal blames the release for reporting nothing:\n%s", out)
	}
	for _, want := range []string{"could not be executed", r.tmp, "TMPDIR"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(r.args("gitleaks"), "git ") {
		t.Errorf("the scan fell back to the gitleaks on PATH:\n%s", r.args("gitleaks"))
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

// TestSecretsCheckRefusesWhenTheCheckCannotRun: a hasher that is on PATH but
// fails, or a digest file that cannot be read, is a fault in the check, not a
// finding: the scan refuses with exit 2 and names it, never exit 1 in silence.
func TestSecretsCheckRefusesWhenTheCheckCannotRun(t *testing.T) {
	for name, want := range map[string]string{"a failing hasher": "could not hash", "an unreadable digest file": "could not be read"} {
		t.Run(name, func(t *testing.T) {
			r := newScanRepo(t, false)
			r.stub("gitleaks", `case "$1" in version) echo 8.30.1 ;; esac; exit 0`)
			tarball, digest := r.pinnedRelease("8.24.3", filepath.Join(t.TempDir(), "pinned.args"))
			r.pinDigest("8.24.3", digest)
			r.serve(tarball)
			if want == "could not hash" {
				r.stub("sha256sum", `echo 'sha256sum: read error' >&2; exit 1`)
				r.stub("shasum", `echo 'shasum: read error' >&2; exit 1`)
			} else {
				sums := filepath.Join(r.dir, "scripts", "gitleaks-checksums.txt")
				if err := os.Chmod(sums, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(sums, 0o644) })
			}
			code, out := r.runHome("secrets", t.TempDir())
			if code != 2 || !strings.Contains(out, "REFUSING") || !strings.Contains(out, want) {
				t.Fatalf("%s: exit %d, want 2 and a refusal saying %q\n%s", name, code, want, out)
			}
		})
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
