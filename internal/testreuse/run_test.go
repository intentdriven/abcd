package testreuse

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// world is one repository with an origin, its fixture packages, and the home
// that holds its check directory. Its tests run real go test under the person's
// own build cache, so the standard library compiles once.
type world struct {
	t      *testing.T
	repo   string
	home   string
	origin string
}

// goCaches are the build cache, module cache and path the test binary started
// with, read before any test moves HOME.
var goCaches []string

func init() {
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH").Output()
	if err != nil {
		return
	}
	vals := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(vals) == 3 {
		goCaches = []string{"GOCACHE=" + vals[0], "GOMODCACHE=" + vals[1], "GOPATH=" + vals[2]}
	}
}

const fixtureGoMod = "module example.com/fixture\n\ngo 1.21\n"

// Each package's test states what it reads, so an edit decides its result.
const (
	readsTest = `package reads
import (
	"os"
	"testing"
)
func TestReadsATrackedFile(t *testing.T) {
	raw, err := os.ReadFile("data.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "good\n" {
		t.Fatalf("data.txt is %q", raw)
	}
}
`
	pureTest = `package pure
import "testing"
func TestNothingAtAll(t *testing.T) { t.Log("TOKEN") }
`
	// The file is read by bash, so the test cache never sees it; nothing
	// declares this test, and the classifier finds it.
	subprocessTest = `package subprocess
import (
	"os/exec"
	"strings"
	"testing"
)
func TestReadsThroughASubprocess(t *testing.T) {
	out, err := exec.Command("bash", "-c", "cat data.txt").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "good" {
		t.Fatalf("data.txt is %q", out)
	}
}
func TestStaysReusable(t *testing.T) { t.Log("TOKEN") }
`
)

func newWorld(t *testing.T) *world {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	// A fixture's git runs under a home of its own, and go test must keep the
	// person's build cache and toolchain, not a cold one under that home.
	gittest.Env(t)
	for _, kv := range goCaches {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}
	base := t.TempDir()
	// macOS's temp directory is reached through a symlink, and git and go
	// report the resolved path.
	base, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	w := &world{t: t, repo: filepath.Join(base, "repo"), home: filepath.Join(base, "home"), origin: filepath.Join(base, "origin.git")}
	if err := os.MkdirAll(w.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(w.home, 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.InitBare(t, w.origin)
	runGit(t, w.repo, "init", "-q", "-b", "main")
	runGit(t, w.repo, "remote", "add", "origin", w.origin)

	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "preflight-receipt.sh"))
	if err != nil {
		t.Fatal(err)
	}
	w.write("go.mod", fixtureGoMod)
	w.write("scripts/preflight-receipt.sh", string(script))
	w.write(".gitignore", ".abcd/.work.local/\n")
	w.write(ListPath, "# nothing declared yet\n")
	w.write("reads/reads_test.go", readsTest)
	w.write("reads/data.txt", "good\n")
	// A token in each binary keeps this run's results apart from every earlier
	// run's in the person's build cache: under -trimpath a package that reads
	// nothing has one result wherever it sits, which is the point of the check.
	token := t.Name() + strconv.FormatInt(time.Now().UnixNano(), 10)
	w.write("pure/pure_test.go", strings.ReplaceAll(pureTest, "TOKEN", token))
	w.write("subprocess/subprocess_test.go", strings.ReplaceAll(subprocessTest, "TOKEN", token))
	w.write("subprocess/data.txt", "good\n")
	runGit(t, w.repo, "add", "-A")
	runGit(t, w.repo, "commit", "-q", "-m", "the fixture")
	runGit(t, w.repo, "push", "-q", "origin", "main")
	runGit(t, w.repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return w
}

func (w *world) write(rel, body string) {
	w.t.Helper()
	writeFile(w.t, filepath.Join(w.repo, rel), body, 0o644)
}

func (w *world) commit(msg string) {
	w.t.Helper()
	runGit(w.t, w.repo, "add", "-A")
	runGit(w.t, w.repo, "commit", "-q", "-m", msg)
}

// worktree adds a fresh linked worktree at the commit in rev.
func (w *world) worktree(name, rev string) string {
	w.t.Helper()
	path := filepath.Join(filepath.Dir(w.repo), name)
	runGit(w.t, w.repo, "worktree", "add", "-q", "--detach", path, rev)
	return path
}

// result is one stage's run.
type result struct {
	code int
	out  string
	err  string
}

func (r result) String() string { return r.out + r.err }

// ran reports whether a package executed tests on this run, not replayed them.
func (r result) ran(pkg string) bool {
	return regexp.MustCompile(`(?m)^ok\s+example\.com/fixture/` + pkg + `\s+[0-9.]+s`).MatchString(r.out)
}

func (r result) cached(pkg string) bool {
	return regexp.MustCompile(`(?m)^ok\s+example\.com/fixture/` + pkg + `\s+\(cached\)`).MatchString(r.out)
}

func (w *world) stage(dir string, mod func(*Config), args ...string) result {
	w.t.Helper()
	var out, errb bytes.Buffer
	cfg := Config{
		Stage: "unit", Args: args, Worktree: dir, Home: w.home,
		Stdout: &out, Stderr: &errb, LockWait: time.Minute, Uid: os.Geteuid(),
	}
	if mod != nil {
		mod(&cfg)
	}
	code := Run(context.Background(), cfg)
	return result{code, out.String(), errb.String()}
}

var allPackages = []string{"-timeout", "5m", "./..."}

// TestAFreshWorktreeReusesWhatAnotherWorktreeChecked: criterion 1. The same
// commit checked in a second, fresh worktree replays every result whose inputs
// are unchanged, a test that opens a tracked file among them, and the report
// counts them. The test a subprocess reads for runs again in both.
func TestAFreshWorktreeReusesWhatAnotherWorktreeChecked(t *testing.T) {
	w := newWorld(t)
	first := w.stage(w.repo, nil, allPackages...)
	if first.code != 0 {
		t.Fatalf("the first check failed: %s", first)
	}
	if !first.ran("reads") || !first.ran("pure") {
		t.Fatalf("the first check should run its packages:\n%s", first)
	}

	second := w.stage(w.worktree("fresh", "HEAD"), nil, allPackages...)
	if second.code != 0 {
		t.Fatalf("the second check failed: %s", second)
	}
	for _, pkg := range []string{"reads", "pure"} {
		if !second.cached(pkg) {
			t.Errorf("%s was not reused in a fresh worktree:\n%s", pkg, second)
		}
	}
	if !second.ran("subprocess") {
		t.Errorf("the test that reads through a subprocess was reused:\n%s", second)
	}
	if !strings.Contains(second.out, "reused 3 of 3 packages (2 with no test run, 1 with only their always-rerun tests run)") {
		t.Errorf("the report does not count the reuse:\n%s", second)
	}
	if !strings.Contains(second.out, "TestReadsThroughASubprocess") || !strings.Contains(second.out, "undeclared") {
		t.Errorf("the report does not say why the undeclared test ran:\n%s", second)
	}
}

// TestAResultDoesNotHangOnTheWorktreesDirectory: a test that reads the working
// directory's variable has its value in go test's key, and go hashes the value
// it was started with, not the one it gave the test. Started from the worktree,
// every such package misses in another worktree: on this repository, most of what
// links the guard, the CLI's 670 seconds among them.
func TestAResultDoesNotHangOnTheWorktreesDirectory(t *testing.T) {
	w := newWorld(t)
	w.write("readspwd/readspwd_test.go", `package readspwd
import (
	"os"
	"testing"
)
func TestReadsTheDirectoryVariables(t *testing.T) {
	t.Log(os.Getenv("PWD"), os.Getenv("OLDPWD"))
}
`)
	w.commit("a test that reads the working directory's variables")
	t.Setenv("OLDPWD", "/first")
	t.Setenv("PWD", w.repo)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 || !r.ran("readspwd") {
		t.Fatalf("the first check: %s", r)
	}
	second := w.worktree("fresh", "HEAD")
	t.Setenv("OLDPWD", "/second")
	t.Setenv("PWD", second)
	r := w.stage(second, nil, allPackages...)
	if !r.cached("readspwd") {
		t.Errorf("a test reading PWD and OLDPWD missed in another worktree:\n%s", r)
	}
}

// TestASameSizeEditFailsTheTestThatReadsIt: the hazard itself. A same-size edit
// to a file a test opens, amended into the commit or committed after it, within
// the same second by every date git records, must not replay the pass: the test
// reruns and fails. Everything else stays reused. The commit under the edit is
// not the root commit, whose amendment would name another check directory.
func TestASameSizeEditFailsTheTestThatReadsIt(t *testing.T) {
	for _, how := range []string{"amended", "committed"} {
		t.Run(how, func(t *testing.T) {
			const second = "2026-11-05T00:00:00+0000"
			t.Setenv("GIT_AUTHOR_DATE", second)
			t.Setenv("GIT_COMMITTER_DATE", second)
			w := newWorld(t)
			w.write("unrelated.txt", "x\n")
			w.commit("not the root commit")
			root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
			if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
				t.Fatalf("the first check failed: %s", r)
			}
			w.write("reads/data.txt", "evil\n")
			runGit(t, w.repo, "add", "-A")
			if how == "amended" {
				runGit(t, w.repo, "commit", "-q", "--amend", "--no-edit")
			} else {
				runGit(t, w.repo, "commit", "-q", "-m", "a same-size edit")
			}
			if now := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD"); now != root {
				t.Fatalf("the root commit moved, so this is another check directory: %s to %s", root, now)
			}

			r := w.stage(w.repo, nil, allPackages...)
			if r.code == 0 {
				t.Fatalf("a same-size edit to the file a test reads replayed its pass:\n%s", r)
			}
			if !strings.Contains(r.out, "data.txt is") {
				t.Errorf("the failure is not the test's:\n%s", r)
			}
			if !r.cached("pure") {
				t.Errorf("an unrelated package was not reused:\n%s", r)
			}
		})
	}
}

// TestAnEditReachingATestOnlyThroughASubprocessFailsIt: the classifier's half of
// fail-closed. Nothing declares this test and the cache cannot see its input, and
// its pass must not stand in for it when that input changes.
func TestAnEditReachingATestOnlyThroughASubprocessFailsIt(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	w.write("subprocess/data.txt", "evil\n")
	w.commit("an input only a subprocess reads")
	r := w.stage(w.repo, nil, allPackages...)
	if r.code == 0 {
		t.Fatalf("an edit a subprocess reads replayed the pass:\n%s", r)
	}
	if !strings.Contains(r.out, "--- FAIL: TestReadsThroughASubprocess") {
		t.Errorf("the failure is not the test's:\n%s", r)
	}
}

// TestADeclaredPackageAlwaysReruns: a package on the default branch's list runs
// on every check even with nothing changed, whatever the classifier finds.
func TestADeclaredPackageAlwaysReruns(t *testing.T) {
	w := newWorld(t)
	w.write(ListPath, "pure # read from the machine by a program its source does not spell\n")
	w.commit("declare a package")
	runGit(t, w.repo, "push", "-q", "origin", "main")
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	r := w.stage(w.repo, nil, allPackages...)
	if !r.ran("pure") {
		t.Errorf("a declared package was reused:\n%s", r)
	}
	if !r.cached("reads") {
		t.Errorf("an undeclared package was not reused:\n%s", r)
	}
	if !strings.Contains(r.out, "on the always-rerun list: read from the machine") {
		t.Errorf("the report does not say why it ran:\n%s", r)
	}
}

// TestAChangeCannotTakeItsTestOffTheList: the change's own copy of the list does
// not count, and the check reads the default branch's.
func TestAChangeCannotTakeItsTestOffTheList(t *testing.T) {
	w := newWorld(t)
	w.write(ListPath, "pure # read from the machine by a program its source does not spell\n")
	w.commit("declare a package")
	runGit(t, w.repo, "push", "-q", "origin", "main")
	runGit(t, w.repo, "checkout", "-q", "-b", "change")
	w.write(ListPath, "# nothing here\n")
	w.commit("take it off")
	w.stage(w.repo, nil, allPackages...)
	if r := w.stage(w.repo, nil, allPackages...); !r.ran("pure") {
		t.Errorf("a change took its own package off the list:\n%s", r)
	}
}

// TestNoListMeansNoReuse: a default branch with no list, or one that does not
// read, reuses nothing and says why.
func TestNoListMeansNoReuse(t *testing.T) {
	w := newWorld(t)
	runGit(t, w.repo, "rm", "-q", ListPath)
	w.commit("no list")
	runGit(t, w.repo, "push", "-q", "origin", "main")
	w.stage(w.repo, nil, allPackages...)
	r := w.stage(w.repo, nil, allPackages...)
	if r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	if !r.ran("pure") || !r.ran("reads") {
		t.Errorf("a repository with no list reused a result:\n%s", r)
	}
	if !strings.Contains(r.out, "reuses nothing") || !strings.Contains(r.out, ListPath) {
		t.Errorf("the report does not say why nothing was reused:\n%s", r)
	}
	if _, err := os.Stat(abcdhome.Path(w.home, "tests")); !os.IsNotExist(err) {
		t.Errorf("a check that reuses nothing made the check directory: %v", err)
	}
}

// TestAnUncommittedEditIsWhatGetsTested: the check directory holds the commit,
// so a tree with edits runs where they are, and an edit that breaks a test fails
// the stage rather than passing on the commit's copy.
func TestAnUncommittedEditIsWhatGetsTested(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	w.write("reads/data.txt", "evil\n")
	r := w.stage(w.repo, nil, allPackages...)
	if r.code == 0 {
		t.Fatalf("an edit in the working tree passed on the commit's copy:\n%s", r)
	}
	if !strings.Contains(r.out, "differs from HEAD") {
		t.Errorf("the report does not say why nothing was reused:\n%s", r)
	}
}

// TestAReleaseCutReusesNothing: criterion 2's second half.
func TestAReleaseCutReusesNothing(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	r := w.stage(w.repo, func(c *Config) { c.Fresh = true }, allPackages...)
	if r.code != 0 {
		t.Fatalf("the cut's check failed: %s", r)
	}
	for _, pkg := range []string{"reads", "pure", "subprocess"} {
		if !r.ran(pkg) {
			t.Errorf("a release cut did not run %s:\n%s", pkg, r)
		}
	}
	if !strings.Contains(r.out, "reused 0 of 3 packages") {
		t.Errorf("the report misstates a cut:\n%s", r)
	}
}

// TestARaceStageReusesOnlyRaceResults: a race build is another binary, so the
// unit stage's results do not serve it, and its own do.
func TestARaceStageReusesOnlyRaceResults(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no race detector here")
	}
	w := newWorld(t)
	race := []string{"-race", "-timeout", "5m", "./..."}
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the unit stage failed: %s", r)
	}
	first := w.stage(w.repo, func(c *Config) { c.Stage = "race" }, race...)
	if first.code != 0 {
		t.Fatalf("the race stage failed: %s", first)
	}
	if !first.ran("pure") || !first.ran("reads") {
		t.Errorf("the race stage replayed the unit stage's results:\n%s", first)
	}
	second := w.stage(w.worktree("fresh", "HEAD"), func(c *Config) { c.Stage = "race" }, race...)
	if !second.cached("pure") || !second.cached("reads") {
		t.Errorf("a second race stage did not reuse the first's results:\n%s", second)
	}
	if !second.ran("subprocess") {
		t.Errorf("an always-rerun test was reused under the race detector:\n%s", second)
	}
}

// TestAFixedTimeoutKeepsAResultReusableUnderLoad: go test's cache key holds the
// timeout, and a stage's own moves with the load. A result reused at one must be
// reused at the other.
func TestAFixedTimeoutKeepsAResultReusableUnderLoad(t *testing.T) {
	w := newWorld(t)
	fixed := func(c *Config) { c.CachedTimeout = "20m" }
	if r := w.stage(w.repo, fixed, "-timeout", "5m", "./..."); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	r := w.stage(w.repo, fixed, "-timeout", "10m", "./...")
	if !r.cached("pure") {
		t.Errorf("a doubled budget under load lost the reuse:\n%s", r)
	}
}

// TestASecondCheckWaitsOrRefuses: one check at a time. A second waits for the
// lock, and past its bound refuses with exit 3 having run nothing.
func TestASecondCheckWaitsOrRefuses(t *testing.T) {
	w := newWorld(t)
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	dir, err := checkDir(w.home, root, os.Geteuid())
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, lockName)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Minute, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	noteHolder(dir, "/the/other/checkout")

	refused := w.stage(w.repo, func(c *Config) { c.LockWait = 100 * time.Millisecond }, allPackages...)
	if refused.code != ExitBusy {
		t.Errorf("a check past its wait exited %d, want %d:\n%s", refused.code, ExitBusy, refused)
	}
	if !strings.Contains(refused.err, "/the/other/checkout") {
		t.Errorf("the refusal does not name who holds the directory:\n%s", refused)
	}
	if _, err := os.Stat(filepath.Join(dir, treeDirName)); !os.IsNotExist(err) {
		t.Errorf("a refused check touched the tree: %v", err)
	}
	if strings.Contains(refused.out, "ok  \t") {
		t.Errorf("a refused check ran tests:\n%s", refused)
	}

	got := make(chan result)
	go func() { got <- w.stage(w.repo, nil, allPackages...) }()
	time.Sleep(1500 * time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	waited := <-got
	if waited.code != 0 {
		t.Fatalf("the waiting check failed: %s", waited)
	}
	if !strings.Contains(waited.out, "waiting for another check") || !waited.ran("pure") {
		t.Errorf("the second check did not wait and then run:\n%s", waited)
	}
}

// TestAnotherAccountsDirectoryIsRefused: two accounts share this machine, each
// keeps its own, and a directory this account does not own or that anyone else
// can write is not used. The check still runs, reusing nothing.
func TestAnotherAccountsDirectoryIsRefused(t *testing.T) {
	w := newWorld(t)
	r := w.stage(w.repo, func(c *Config) { c.Uid = os.Geteuid() + 1 }, allPackages...)
	if r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	if !strings.Contains(r.out, "belongs to another account") || !r.ran("pure") {
		t.Errorf("a directory of another account was used:\n%s", r)
	}

	dir, err := checkDir(w.home, "abc", os.Geteuid())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := checkDir(w.home, "abc", os.Geteuid()); err == nil || !strings.Contains(err.Error(), "written by another account") {
		t.Errorf("a directory anyone can write: %v", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := checkDir(w.home, "abc", os.Geteuid()); err == nil {
		t.Error("a symlinked check directory was taken")
	}
}

// TestACrashedCheckRecovers: a check killed part way leaves a directory the next
// check repairs, and what it reuses afterwards is still right.
func TestACrashedCheckRecovers(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	dir := abcdhome.Path(w.home, "tests", root)
	tree := filepath.Join(dir, treeDirName)
	writeFile(t, filepath.Join(tree, "reads", "data.txt"), "evil\n", 0o644)
	writeFile(t, filepath.Join(tree, "stray_test.go"), "package nothing\n", 0o644)
	writeFile(t, filepath.Join(dir, ".abcd-tmp-1"), "cut", 0o600)
	writeFile(t, filepath.Join(dir, "unknown.json"), "{", 0o600)

	r := w.stage(w.repo, nil, allPackages...)
	if r.code != 0 {
		t.Fatalf("the check did not recover: %s", r)
	}
	if !r.cached("reads") || !r.cached("pure") {
		t.Errorf("a repaired directory lost the reuse:\n%s", r)
	}
	if raw, _ := os.ReadFile(filepath.Join(tree, "reads", "data.txt")); string(raw) != "good\n" {
		t.Errorf("the tree was not repaired: %q", raw)
	}
}

// TestAStageWithATestThatRewritesATrackedFileFails: what the stage checked must
// be the commit from start to end, as a test that rewrites a tracked file in a
// worktree leaves that tree dirty and mints no receipt.
func TestAStageWithATestThatRewritesATrackedFileFails(t *testing.T) {
	w := newWorld(t)
	w.write("rewrites/rewrites_test.go", `package rewrites
import (
	"os"
	"testing"
)
func TestRewrites(t *testing.T) {
	if err := os.WriteFile("data.txt", []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
`)
	w.write("rewrites/data.txt", "good\n")
	w.commit("a test that rewrites its input")
	r := w.stage(w.repo, nil, allPackages...)
	if r.code == 0 || !strings.Contains(r.out, "rewrites/data.txt") {
		t.Errorf("a stage whose test rewrote a tracked file passed:\n%s", r)
	}
}

// TestAStageNamesWhatATestLeftInTheTree: a file a test leaves in the check
// directory moves the time of the directory that holds it, so every test that
// walks the tree hashes that time and never replays. The stage still passes, and
// it names the file, so the loss of reuse has a cause anyone can read.
func TestAStageNamesWhatATestLeftInTheTree(t *testing.T) {
	w := newWorld(t)
	w.write("leaves/leaves_test.go", `package leaves
import (
	"os"
	"testing"
)
func TestLeaves(t *testing.T) {
	if err := os.WriteFile("left-behind.lock", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
`)
	w.commit("a test that leaves a file")
	r := w.stage(w.repo, nil, allPackages...)
	if r.code != 0 {
		t.Fatalf("the stage failed: %s", r)
	}
	if !strings.Contains(r.out, "leaves/left-behind.lock") {
		t.Errorf("the report does not name what the test left:\n%s", r)
	}
}

// TestAGoWorkAboveTheTreeChangesNothing: go looks above the module for a go.work,
// and a test can write one there. A reused line asks for none.
func TestAGoWorkAboveTheTreeChangesNothing(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the first check failed: %s", r)
	}
	writeFile(t, abcdhome.Path(w.home, "tests", "go.work"), "this is not a go.work\n", 0o644)
	r := w.stage(w.repo, nil, allPackages...)
	if r.code != 0 || !r.cached("pure") || strings.Contains(r.out, "reuses nothing") {
		t.Errorf("a go.work above the tree reached the check:\n%s", r)
	}
}

// TestTheHolderNoteIsNotWrittenThroughALink: the note of who holds the lock goes
// to the file of that name and to nothing else.
func TestTheHolderNoteIsNotWrittenThroughALink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "elsewhere")
	writeFile(t, target, "kept\n", 0o644)
	if err := os.Symlink(target, filepath.Join(dir, holderName)); err != nil {
		t.Fatal(err)
	}
	noteHolder(dir, "/a/checkout")
	if raw, _ := os.ReadFile(target); string(raw) != "kept\n" {
		t.Errorf("the note went through the link: %q", raw)
	}
	if got := holder(dir); strings.Contains(got, "kept") {
		t.Errorf("the note read through the link: %q", got)
	}
}

// TestCancellingEndsTheWait: a check waiting for another stops when it is told to,
// having run nothing, rather than holding its caller for the rest of the bound.
func TestCancellingEndsTheWait(t *testing.T) {
	w := newWorld(t)
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	dir, err := checkDir(w.home, root, os.Geteuid())
	if err != nil {
		t.Fatal(err)
	}
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(dir, lockName), time.Minute, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	defer func() {
		close(release)
		<-done
	}()
	ctx, cancel := context.WithCancel(context.Background())
	var out, errb bytes.Buffer
	ended := make(chan int)
	go func() {
		ended <- Run(ctx, Config{Stage: "unit", Args: allPackages, Worktree: w.repo, Home: w.home, Stdout: &out, Stderr: &errb, LockWait: time.Hour, Uid: os.Geteuid()})
	}()
	time.Sleep(1500 * time.Millisecond)
	cancel()
	select {
	case code := <-ended:
		if code != ExitBusy {
			t.Errorf("a cancelled wait exited %d, want %d", code, ExitBusy)
		}
		if strings.Contains(out.String(), "ok  \t") {
			t.Errorf("a cancelled wait ran tests:\n%s", out.String())
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the wait outlived its cancellation")
	}
}

// TestAWaitTakesTheBudgetAgain: -p and -timeout are derived before a wait that can
// last hours, so what follows one takes them afresh, and the hook says which stage
// it follows so a race stage can go back through its gate. No wait, no hook.
func TestAWaitTakesTheBudgetAgain(t *testing.T) {
	w := newWorld(t)
	var called []bool
	hook := func(c *Config) {
		c.AfterWait = func(race bool) (string, string, error) {
			called = append(called, race)
			return "3", "7m", nil
		}
	}
	if r := w.stage(w.repo, hook, allPackages...); r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	if len(called) != 0 {
		t.Fatalf("a check that waited for nothing took the budget again: %v", called)
	}
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	dir, err := checkDir(w.home, root, os.Geteuid())
	if err != nil {
		t.Fatal(err)
	}
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(dir, lockName), time.Minute, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	got := make(chan result)
	go func() { got <- w.stage(w.repo, hook, "-race", "-timeout", "5m", "./...") }()
	time.Sleep(1500 * time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if r := <-got; r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	if len(called) != 1 || !called[0] {
		t.Errorf("a race stage's wait called the hook %v", called)
	}

	// A hook that refuses, as a race gate past its bound does, stops the stage.
	held2, release2, done2 := make(chan struct{}), make(chan struct{}), make(chan error)
	go func() {
		done2 <- fsutil.WithFileLock(filepath.Join(dir, lockName), time.Minute, func() error {
			close(held2)
			<-release2
			return nil
		})
	}()
	<-held2
	refusing := func(c *Config) {
		c.AfterWait = func(bool) (string, string, error) { return "", "", errors.New("the gate stayed shut") }
	}
	got2 := make(chan result)
	go func() { got2 <- w.stage(w.repo, refusing, allPackages...) }()
	time.Sleep(1500 * time.Millisecond)
	close(release2)
	if err := <-done2; err != nil {
		t.Fatal(err)
	}
	r := <-got2
	if r.code == 0 || !strings.Contains(r.String(), "the gate stayed shut") || strings.Contains(r.out, "ok  \t") {
		t.Errorf("a refusing hook did not stop the stage:\n%s", r)
	}
}

// TestTheLockIsReleasedBeforeTheRerunLines: the rerun lines run in the worktree and
// touch nothing of the check directory, so another check need not wait for them.
func TestTheLockIsReleasedBeforeTheRerunLines(t *testing.T) {
	w := newWorld(t)
	root := runGit(t, w.repo, "rev-list", "--max-parents=0", "HEAD")
	lock := filepath.Join(abcdhome.Path(w.home, "tests", root), lockName)
	t.Setenv("FIXTURE_LOCK", lock)
	w.write("holds/holds_test.go", `package holds
import (
	"os"
	"os/exec"
	"syscall"
	"testing"
)
func TestTheLockIsFree(t *testing.T) {
	_ = exec.Command("bash", "-c", "true").Run()
	f, err := os.OpenFile(os.Getenv("FIXTURE_LOCK"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("the check held its lock through the rerun lines: %v", err)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
`)
	w.commit("a test that tries the lock")
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Errorf("%s", r)
	}
}

// TestADirtyRunNamesNoTree: tests that read a working tree's edits vouch for no
// commit's tree, so their report says so and the receipt cannot name one.
func TestADirtyRunNamesNoTree(t *testing.T) {
	w := newWorld(t)
	w.write("pure/edited.txt", "an edit\n")
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	head := runGit(t, w.repo, "rev-parse", "HEAD")
	matches, err := filepath.Glob(filepath.Join(w.repo, ".abcd", ".work.local", "preflight-reuse", head, "unit-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports: %v, %v", matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	tree := runGit(t, w.repo, "rev-parse", "HEAD^{tree}")
	if strings.Contains(string(raw), tree) || !strings.Contains(string(raw), "tree none\n") || !strings.Contains(string(raw), "mode dirty\n") {
		t.Errorf("a dirty run's report names a tree:\n%s", raw)
	}
}

// TestAListLineThatResolvesToNothingReusesNothing: a renamed package leaves the list
// naming nothing, which would be its tests quietly reused.
func TestAListLineThatResolvesToNothingReusesNothing(t *testing.T) {
	for name, line := range map[string]string{
		"a package": "renamed # a subprocess reads it\n",
		"a test":    "pure TestRenamed # a subprocess reads it\n",
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			w.write(ListPath, line)
			w.commit("a stale line")
			runGit(t, w.repo, "push", "-q", "origin", "main")
			w.stage(w.repo, nil, allPackages...)
			r := w.stage(w.repo, nil, allPackages...)
			if r.code != 0 {
				t.Fatalf("the check failed: %s", r)
			}
			if !r.ran("pure") || !r.ran("reads") || !strings.Contains(r.out, "reuses nothing") {
				t.Errorf("a line naming nothing left reuse on:\n%s", r)
			}
		})
	}
}

// TestAReleaseCutSaysSoInItsReport: the receipt reads the mode, so a cut's run is
// told from one that fell back and from one that reused.
func TestAReleaseCutSaysSoInItsReport(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, func(c *Config) { c.Fresh = true }, allPackages...); r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	head := runGit(t, w.repo, "rev-parse", "HEAD")
	matches, _ := filepath.Glob(filepath.Join(w.repo, ".abcd", ".work.local", "preflight-reuse", head, "unit-*"))
	if len(matches) != 1 {
		t.Fatalf("reports: %v", matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "mode cut\n") {
		t.Errorf("a release cut's report:\n%s", raw)
	}
}

// TestATestAskingGitAboutTheRepositoryGetsNoAnswer: the check directory has no
// .git, and git may not walk above it, even when the home is itself a repository,
// so a test reading the repository through git cannot pass on state the cache
// cannot see. Written so the classifier finds nothing, as in a package's helper.
func TestATestAskingGitAboutTheRepositoryGetsNoAnswer(t *testing.T) {
	w := newWorld(t)
	runGit(t, w.home, "init", "-q")
	w.write("asks/asks_test.go", `package asks
import (
	"os/exec"
	"testing"
)
func TestAsks(t *testing.T) {
	name := "git"
	if out, err := exec.Command(name, "status").CombinedOutput(); err != nil {
		t.Fatalf("no answer: %s", out)
	}
}
`)
	w.commit("a test that asks git")
	r := w.stage(w.repo, nil, allPackages...)
	if r.code == 0 {
		t.Fatalf("a test asking git about the repository passed in the check directory:\n%s", r)
	}
	if !strings.Contains(r.out, "no answer") || !strings.Contains(r.out, "not a git repository") {
		t.Errorf("the failure is not git's refusal:\n%s", r)
	}
}

// TestARecipeFlagTheCheckDoesNotKnowIsRefused: a flag it does not read could
// change what is being checked.
func TestARecipeFlagTheCheckDoesNotKnowIsRefused(t *testing.T) {
	for _, args := range [][]string{
		{"-count=2", "./..."},
		{"-v", "./..."},
		{"-p"},
		{"-timeout", "5m"},
	} {
		var out, errb bytes.Buffer
		code := Run(context.Background(), Config{Stage: "unit", Args: args, Worktree: t.TempDir(), Stdout: &out, Stderr: &errb})
		if code != ExitUsage {
			t.Errorf("%v exited %d, want %d: %s", args, code, ExitUsage, errb.String())
		}
	}
}

// TestAStageRecordsItsTreeAndCounts: the receipt reads this, so a reused result
// counts for that exact tree.
func TestAStageRecordsItsTreeAndCounts(t *testing.T) {
	w := newWorld(t)
	if r := w.stage(w.repo, nil, allPackages...); r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	r := w.stage(w.worktree("fresh", "HEAD"), nil, allPackages...)
	if r.code != 0 {
		t.Fatalf("the check failed: %s", r)
	}
	head := runGit(t, w.repo, "rev-parse", "HEAD")
	tree := runGit(t, w.repo, "rev-parse", "HEAD^{tree}")
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(w.repo), "fresh", ".abcd", ".work.local", "preflight-reuse", head, "unit-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("reports: %v, %v", matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"commit " + head, "tree " + tree, "stage unit", "mode reuse", "reused 3", "packages 3", "result pass"} {
		if !strings.Contains(string(raw), want+"\n") {
			t.Errorf("the report lacks %q:\n%s", want, raw)
		}
	}
}

// TestTheTallyCountsWhatGoTestSaid: a replayed result, one that ran, a failure,
// and the half of a split package with nothing to run.
func TestTheTallyCountsWhatGoTestSaid(t *testing.T) {
	var tally Tally
	for _, l := range []string{
		"ok  \tm/cached\t(cached)",
		"ok  \tm/ran\t1.234s",
		"FAIL\tm/failed\t0.1s",
		"FAIL\tm/broken [build failed]",
		"ok  \tm/split\t(cached)",
		"ok  \tm/split\t0.5s",
		"ok  \tm/onlyrerun\t(cached) [no tests to run]",
		"ok  \tm/onlyrerun\t0.5s",
		"ok  \tm/nomatch\t0.1s [no tests to run]",
		"?   \tm/none\t[no test files]",
		"--- FAIL: TestX (0.00s)",
		"FAIL",
		"some output",
	} {
		tally.Line(l)
	}
	c := tally.Counts()
	if c.Packages != 6 || c.Reused != 2 || c.Whole != 1 {
		t.Errorf("counted %+v", c)
	}
	if strings.Join(c.Ran, " ") != "m/broken m/failed m/onlyrerun m/ran m/split" {
		t.Errorf("ran: %v", c.Ran)
	}
}
