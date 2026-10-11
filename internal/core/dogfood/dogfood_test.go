package dogfood

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// git runs git in dir under the isolated environment with a fixed identity,
// failing the test on error, and returns trimmed stdout.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "init.defaultBranch=main", "-c", "protocol.file.allow=always"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(gittest.Env(t),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// realDir is a fresh temp directory with its symlinks resolved, the spelling
// git reports back.
func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// goVersion is the running toolchain's version as a go.mod line takes it, so
// the fixture's build pins the toolchain already present.
func goVersion() string { return strings.TrimPrefix(runtime.Version(), "go") }

// mainSource is a cmd/abcd that prints marker, so a test can tell which
// commit a build was made from by running it.
func mainSource(marker string) string {
	return "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"" + marker + "\") }\n"
}

// fixture is an upstream repository holding an abcd-shaped module and a
// clone of it, the checkout under test, whose origin/main is the upstream's
// main.
type fixture struct {
	upstream, checkout, home string
}

// goCaches are the build and module caches of the account running the tests,
// read once before any test repoints HOME: gittest.Env gives each test a
// fresh HOME, and a build under it would otherwise start from empty caches
// and fetch its toolchain again.
var goCaches = sync.OnceValues(func() ([]string, error) {
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
})

func newFixture(t *testing.T, module string) fixture {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	caches, err := goCaches()
	if err != nil || len(caches) != 2 {
		t.Fatalf("reading the go caches: %v %q", err, caches)
	}
	t.Setenv("GOCACHE", caches[0])
	t.Setenv("GOMODCACHE", caches[1])
	up := realDir(t)
	git(t, up, "init", "-q")
	write(t, filepath.Join(up, "go.mod"), "module "+module+"\n\ngo "+goVersion()+"\n")
	write(t, filepath.Join(up, "cmd", "abcd", "main.go"), mainSource("reviewed-1"))
	git(t, up, "add", ".")
	git(t, up, "commit", "-q", "-m", "first")
	parent := realDir(t)
	git(t, parent, "clone", "-q", up, "checkout")
	return fixture{upstream: up, checkout: filepath.Join(parent, "checkout"), home: realDir(t)}
}

// list writes the person's opt-in list naming paths, at mode 0600.
func (f fixture) list(t *testing.T, paths ...string) {
	t.Helper()
	dir := abcdhome.Path(f.home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "# dogfood\n" + strings.Join(paths, "\n") + "\n"
	if err := os.WriteFile(abcdhome.Path(f.home, "dogfood-roots"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// land commits a new main.go printing marker on the upstream and fetches it
// into the checkout, as a merge through review would arrive.
func (f fixture) land(t *testing.T, source string) string {
	t.Helper()
	write(t, filepath.Join(f.upstream, "cmd", "abcd", "main.go"), source)
	git(t, f.upstream, "commit", "-q", "-am", "landed")
	git(t, f.checkout, "fetch", "-q", "origin")
	return git(t, f.upstream, "rev-parse", "HEAD")
}

func run(t *testing.T, bin string) string {
	t.Helper()
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("running the build %s: %v", bin, err)
	}
	return strings.TrimSpace(string(out))
}

func notes(p Plan) string { return strings.Join(p.Notes, "\n") }

// buildFor resolves cwd and builds what it asks for, failing the test unless
// the plan asked for a build.
func buildFor(t *testing.T, home, cwd string) string {
	t.Helper()
	p := Resolve(home, cwd)
	if !p.StartBuild || p.Checkout == nil {
		t.Fatalf("Resolve asked for no build: %+v", p)
	}
	bin, err := Build(home, *p.Checkout)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return bin
}

// TestNoListRunsTheRelease: without a list, nothing is resolved and nothing
// is said.
func TestNoListRunsTheRelease(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	if p := Resolve(f.home, f.checkout); p.Binary != "" || p.StartBuild || p.Checkout != nil || len(p.Notes) != 0 {
		t.Fatalf("Resolve with no list = %+v, want the installed release and silence", p)
	}
}

// TestUnlistedCheckoutRunsTheRelease: a list naming another checkout leaves
// this one on the installed release, silently.
func TestUnlistedCheckoutRunsTheRelease(t *testing.T) {
	f := newFixture(t, "example.com/other")
	f.list(t, realDir(t))
	if p := Resolve(f.home, f.checkout); p.Binary != "" || p.StartBuild || p.Checkout != nil || len(p.Notes) != 0 {
		t.Fatalf("Resolve in an unlisted checkout = %+v, want the installed release and silence", p)
	}
}

// TestPosingCheckoutNeverReachesTheBuild: a repository carrying abcd's module
// path and its cmd/abcd, but not listed, is not built and not run, whatever it
// claims to be. Recognising abcd's source automatically was rejected for
// exactly this repository.
func TestPosingCheckoutNeverReachesTheBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, realDir(t))
	p := Resolve(f.home, f.checkout)
	if p.Binary != "" || p.StartBuild || p.Checkout != nil {
		t.Fatalf("Resolve in an unlisted checkout posing as abcd = %+v, want the installed release", p)
	}
	if _, err := os.Lstat(abcdhome.Path(f.home, "dogfood")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolving an unlisted checkout touched the build store: %v", err)
	}
}

// TestListedCheckoutBuildsTheReviewedTip: a listed checkout asks for a build
// of origin/main, the build runs that commit's code, and once it is there
// Resolve names it without asking again.
func TestListedCheckoutBuildsTheReviewedTip(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	p := Resolve(f.home, f.checkout)
	if p.Binary != "" || !p.StartBuild || p.Checkout == nil {
		t.Fatalf("first Resolve = %+v, want a build asked for and the installed release meanwhile", p)
	}
	if !strings.Contains(notes(p), "building it in the background") {
		t.Fatalf("the first Resolve did not say a build was starting: %q", notes(p))
	}
	tip := git(t, f.checkout, "rev-parse", "origin/main")
	if p.Checkout.Commit != tip || p.Checkout.Ref != "refs/remotes/origin/main" {
		t.Fatalf("source = %s %s, want refs/remotes/origin/main %s", p.Checkout.Ref, p.Checkout.Commit, tip)
	}
	bin, err := Build(f.home, *p.Checkout)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if want := abcdhome.Path(f.home, "dogfood", p.Checkout.RootSHA, tip, BinaryName); bin != want {
		t.Fatalf("build at %s, want %s", bin, want)
	}
	if got := run(t, bin); got != "reviewed-1" {
		t.Fatalf("the build printed %q, want the reviewed tip's reviewed-1", got)
	}
	again := Resolve(f.home, f.checkout)
	if again.Binary != bin || again.StartBuild || len(again.Notes) != 0 {
		t.Fatalf("Resolve after the build = %+v, want the build named and silence", again)
	}
}

// TestHeadCommitNeverReachesTheBuild: a commit on the checkout's HEAD that is
// not on origin/main, and an uncommitted change, never reach the build: an
// agent's commit cannot change the guard that judges its next command.
func TestHeadCommitNeverReachesTheBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	write(t, filepath.Join(f.checkout, "cmd", "abcd", "main.go"), mainSource("agent-head"))
	git(t, f.checkout, "commit", "-q", "-am", "an agent's own commit")
	write(t, filepath.Join(f.checkout, "cmd", "abcd", "main.go"), mainSource("agent-uncommitted"))
	f.list(t, f.checkout)
	p := Resolve(f.home, f.checkout)
	head := git(t, f.checkout, "rev-parse", "HEAD")
	if p.Checkout == nil || p.Checkout.Commit == head {
		t.Fatalf("Resolve chose HEAD %s as the source: %+v", head, p)
	}
	bin := buildFor(t, f.home, f.checkout)
	if got := run(t, bin); got != "reviewed-1" {
		t.Fatalf("the build printed %q, want the reviewed tip's reviewed-1, not the checkout's own work", got)
	}
}

// TestMovedTipRebuilds: when origin/main moves, Resolve asks for a build of
// the new tip, and the new build runs the new code.
func TestMovedTipRebuilds(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	first := buildFor(t, f.home, f.checkout)
	moved := f.land(t, mainSource("reviewed-2"))
	p := Resolve(f.home, f.checkout)
	if p.Binary != "" || !p.StartBuild || p.Checkout == nil || p.Checkout.Commit != moved {
		t.Fatalf("Resolve after the tip moved = %+v, want a build of %s asked for", p, moved)
	}
	second, err := Build(f.home, *p.Checkout)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if second == first || run(t, second) != "reviewed-2" {
		t.Fatalf("the rebuild at %s printed %q, want a new build printing reviewed-2", second, run(t, second))
	}
}

// TestFailedBuildFallsBackLoudly: a tip that does not compile leaves no build,
// the installed release runs, a note says the build FAILED and where its log
// is, and no build is asked for again until the retry window has passed.
func TestFailedBuildFallsBackLoudly(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	f.land(t, "package main\n\nfunc main() { this does not compile }\n")
	p := Resolve(f.home, f.checkout)
	if !p.StartBuild || p.Checkout == nil {
		t.Fatalf("Resolve = %+v, want a build asked for", p)
	}
	if bin, err := Build(f.home, *p.Checkout); err == nil || bin != "" {
		t.Fatalf("Build of a tip that does not compile = %q, %v; want an error and no build", bin, err)
	}
	after := Resolve(f.home, f.checkout)
	if after.Binary != "" || after.StartBuild {
		t.Fatalf("Resolve after a failed build = %+v, want the installed release and no retry yet", after)
	}
	if n := notes(after); !strings.Contains(n, "FAILED") || !strings.Contains(n, "installed release runs") || !strings.Contains(n, ".log") {
		t.Fatalf("the failure note does not say it failed, what runs, and where the log is: %q", n)
	}
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return time.Now().Add(retryAfter + time.Minute) }
	if later := Resolve(f.home, f.checkout); !later.StartBuild {
		t.Fatalf("Resolve after the retry window = %+v, want the build asked for again", later)
	}
}

// TestRunningBuildIsNotRaced: while one build holds the store's lock, a second
// Build returns ErrBuildRunning at once and Resolve asks for none.
func TestRunningBuildIsNotRaced(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil {
		t.Fatalf("Resolve = %+v", p)
	}
	store := abcdhome.Path(f.home, "dogfood", p.Checkout.RootSHA)
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	held, release := make(chan struct{}), make(chan struct{})
	done := make(chan error)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(store, lockName), time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	if _, err := Build(f.home, *p.Checkout); !errors.Is(err, ErrBuildRunning) {
		t.Errorf("Build under a held lock = %v, want ErrBuildRunning", err)
	}
	if q := Resolve(f.home, f.checkout); q.StartBuild || !strings.Contains(notes(q), "still running") {
		t.Errorf("Resolve under a held lock = %+v, want no build asked for and a still-running note", q)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// TestListedMainCoversItsWorktrees: listing the main checkout covers a linked
// worktree it records, and the build still comes from origin/main.
func TestListedMainCoversItsWorktrees(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	wt := filepath.Join(realDir(t), "wt")
	git(t, f.checkout, "worktree", "add", "-q", "-b", "lane", wt)
	p := Resolve(f.home, wt)
	if p.Checkout == nil || !p.StartBuild || !same(p.Checkout.Repo, f.checkout) || !same(p.Checkout.Toplevel, wt) {
		t.Fatalf("Resolve in a worktree of the listed checkout = %+v, want it covered", p)
	}
}

// TestBorrowedGitDirIsRefused: a directory whose .git file points into the
// listed checkout's worktree records, but which that checkout does not record
// as a worktree, is refused out loud and never built for.
func TestBorrowedGitDirIsRefused(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	wt := filepath.Join(realDir(t), "wt")
	git(t, f.checkout, "worktree", "add", "-q", "-b", "lane", wt)
	poser := realDir(t)
	gitfile, err := os.ReadFile(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(poser, ".git"), string(gitfile))
	write(t, filepath.Join(poser, "go.mod"), "module github.com/intentdriven/abcd\n")
	p := Resolve(f.home, poser)
	if p.Checkout != nil || p.StartBuild || p.Binary != "" || !strings.Contains(notes(p), "REFUSED") {
		t.Fatalf("Resolve in a directory borrowing a listed repository's git dir = %+v, want a loud refusal", p)
	}
}

// TestNoRemoteTipRunsTheRelease: a listed checkout with no origin default
// branch builds nothing, never falls back to a local branch, and says so.
func TestNoRemoteTipRunsTheRelease(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	git(t, f.checkout, "remote", "remove", "origin")
	f.list(t, f.checkout)
	p := Resolve(f.home, f.checkout)
	if p.Checkout != nil || p.StartBuild || p.Binary != "" || !strings.Contains(notes(p), "git fetch origin") {
		t.Fatalf("Resolve with no remote tip = %+v, want the installed release and a note naming the fetch", p)
	}
}

// TestUntrustedListIsIgnoredLoudly: a list others can write, or one behind a
// symlinked ~/.abcd.noindex, is refused by the shared declaration reader and
// said out loud; nothing is built.
func TestUntrustedListIsIgnoredLoudly(t *testing.T) {
	t.Run("writable by others", func(t *testing.T) {
		f := newFixture(t, "github.com/intentdriven/abcd")
		f.list(t, f.checkout)
		if err := os.Chmod(abcdhome.Path(f.home, "dogfood-roots"), 0o666); err != nil {
			t.Fatal(err)
		}
		p := Resolve(f.home, f.checkout)
		if p.Checkout != nil || p.StartBuild || !strings.Contains(notes(p), "IGNORED "+RootsDisplay) {
			t.Fatalf("Resolve with a list others can write = %+v, want it ignored out loud", p)
		}
	})
	t.Run("not a regular file", func(t *testing.T) {
		f := newFixture(t, "github.com/intentdriven/abcd")
		if err := os.MkdirAll(abcdhome.Path(f.home, "dogfood-roots"), 0o700); err != nil {
			t.Fatal(err)
		}
		p := Resolve(f.home, f.checkout)
		if p.Checkout != nil || p.StartBuild || !strings.Contains(notes(p), "IGNORED "+RootsDisplay) {
			t.Fatalf("Resolve with a directory for a list = %+v, want it ignored out loud", p)
		}
	})
	t.Run("behind a symlinked home folder", func(t *testing.T) {
		f := newFixture(t, "github.com/intentdriven/abcd")
		real := realDir(t)
		body := f.checkout + "\n"
		if err := os.WriteFile(filepath.Join(real, "dogfood-roots"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, abcdhome.Path(f.home)); err != nil {
			t.Fatal(err)
		}
		p := Resolve(f.home, f.checkout)
		if p.Checkout != nil || p.StartBuild || !strings.Contains(notes(p), "IGNORED "+RootsDisplay) {
			t.Fatalf("Resolve with the list behind a symlinked home folder = %+v, want it ignored out loud", p)
		}
	})
}

// TestExportIsTheCommitExactly: repository-local attributes an agent can
// write (.git/info/attributes) must not shape the build. An export-ignore on a
// file of the reviewed commit, which would build the binary without it, makes
// the build refuse rather than build something other than the commit.
func TestExportIsTheCommitExactly(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.list(t, f.checkout)
	main := "package main\n\nimport \"fmt\"\n\nvar marker = \"unguarded\"\n\nfunc main() { fmt.Println(marker) }\n"
	write(t, filepath.Join(f.upstream, "cmd", "abcd", "guard.go"), "package main\n\nfunc init() { marker = \"guarded\" }\n")
	git(t, f.upstream, "add", ".")
	f.land(t, main)
	write(t, filepath.Join(f.checkout, ".git", "info", "attributes"), "cmd/abcd/guard.go export-ignore\n")
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil {
		t.Fatalf("Resolve = %+v", p)
	}
	bin, err := Build(f.home, *p.Checkout)
	if err == nil {
		t.Fatalf("Build with an export-ignore over a reviewed file succeeded; the build at %s prints %q", bin, run(t, bin))
	}
	if !strings.Contains(err.Error(), "cmd/abcd/guard.go") {
		t.Fatalf("Build error %q does not name the file the export dropped", err)
	}
}
