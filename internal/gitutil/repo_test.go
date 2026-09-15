package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// commitAll stages and commits everything in repo, with an isolated identity so
// no developer git config is needed.
func commitAll(t *testing.T, repo string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "-A"},
		{"-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-q", "-m", "fixture"},
	} {
		if out, err := runGit(t, repo, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func TestInRepo(t *testing.T) {
	repo := newRepo(t, "")
	if !gitutil.InRepo(repo) {
		t.Error("InRepo(git repo) = false, want true")
	}

	plain := t.TempDir() // not a git repo
	if gitutil.InRepo(plain) {
		t.Error("InRepo(non-repo) = true, want false")
	}
}

func TestTrackedFiles(t *testing.T) {
	repo := newRepo(t, "")
	// Two committed files, one untracked. TrackedFiles reports only committed.
	write := func(rel, body string) {
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.md", "x")
	write("sub/b.go", "y")
	commitAll(t, repo)
	write("untracked.txt", "z") // added after the commit, never staged

	got, err := gitutil.TrackedFiles(repo)
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, f := range got {
		set[f] = true
	}
	if !set["a.md"] || !set["sub/b.go"] {
		t.Errorf("TrackedFiles = %v, want a.md and sub/b.go", got)
	}
	if set["untracked.txt"] {
		t.Errorf("TrackedFiles included an untracked file: %v", got)
	}
}

// Outside a repo, TrackedFiles returns no files and no error (fail open) — a
// privacy scan degrades to "nothing to scan", not a crash.
func TestTrackedFilesOutsideRepo(t *testing.T) {
	got, err := gitutil.TrackedFiles(t.TempDir())
	if err != nil {
		t.Fatalf("TrackedFiles outside a repo errored: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("TrackedFiles outside a repo = %v, want empty", got)
	}
}

// Inside a repo, a failure other than not-a-repo (here a corrupt index) must be
// returned as an error, not swallowed as "nothing tracked". Otherwise a
// scanning rule (privacy hygiene) would read zero files and report the repo
// clean. InRepo still says true — it does not read the index — so the earlier
// blanket (nil, nil) hid a real read failure.
func TestTrackedFilesCorruptIndexErrors(t *testing.T) {
	repo := newRepo(t, "")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	if err := os.WriteFile(filepath.Join(repo, ".git", "index"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !gitutil.InRepo(repo) {
		t.Skip("InRepo unexpectedly false with a corrupt index")
	}
	if _, err := gitutil.TrackedFiles(repo); err == nil {
		t.Error("TrackedFiles with a corrupt index returned nil error; want the failure surfaced")
	}
}

// An inherited GIT_DIR/GIT_WORK_TREE must not redirect queries to a different
// repository: isolation strips them so the answer is always about root, not the
// repo those vars point at.
func TestIsolationIgnoresInheritedGitDir(t *testing.T) {
	writeFile := func(dir, rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	target := newRepo(t, "")
	writeFile(target, "target.md", "x")
	commitAll(t, target)

	other := newRepo(t, "")
	writeFile(other, "secret.md", "y")
	commitAll(t, other)

	// Point the environment at the OTHER repo, then ask about target.
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)

	got, err := gitutil.TrackedFiles(target)
	if err != nil {
		t.Fatalf("TrackedFiles: %v", err)
	}
	set := map[string]bool{}
	for _, f := range got {
		set[f] = true
	}
	if set["secret.md"] {
		t.Errorf("TrackedFiles followed inherited GIT_DIR into another repo: %v", got)
	}
	if !set["target.md"] {
		t.Errorf("TrackedFiles = %v, want target.md", got)
	}

	// A plain non-repo directory must not read as a repo just because GIT_DIR
	// is set in the environment.
	if gitutil.InRepo(t.TempDir()) {
		t.Error("InRepo(non-repo with inherited GIT_DIR) = true, want false")
	}
}

// TestRunCappedRefusesOversizeOutput pins the difference between the two bounded
// readers. RunLimited degrades — its callers (the lifeboat probe over an
// archived repo) would rather have a partial answer than none. RunCapped
// refuses, for callers whose answer is only correct if it is complete: a
// truncated `git log` is not a shorter history, it is a wrong one, and a caller
// that cannot tell the two apart publishes the wrong one.
func TestRunCappedRefusesOversizeOutput(t *testing.T) {
	repo := newRepo(t, "")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)

	// A cap far below one line of `git log` output.
	_, err := gitutil.RunCapped(repo, 8, "log", "--pretty=format:%H %s")
	if err == nil {
		t.Fatal("RunCapped returned a silently truncated answer")
	}
	for _, want := range []string{"8", "log"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}

	// Under the cap, it is Run with a ceiling.
	out, err := gitutil.RunCapped(repo, 1<<20, "log", "--pretty=format:%s")
	if err != nil {
		t.Fatal(err)
	}
	if out != "fixture" {
		t.Errorf("RunCapped = %q, want %q", out, "fixture")
	}

	// RunLimited keeps its documented degradation, so the two are not the same
	// function with a different name.
	if _, err := gitutil.RunLimited(repo, 8, "log", "--pretty=format:%H %s"); err != nil {
		t.Errorf("RunLimited must still degrade rather than refuse: %v", err)
	}
}

// TestTrackedFilesRepoShapedButUnanswerable pins the three-state contract: a
// plain directory is "nothing tracked, not an error", but a repo-shaped tree
// git cannot answer for (here: a corrupt .git) is an error — never a silent
// zero-file result a scanning rule would report as a clean pass.
func TestTrackedFilesRepoShapedButUnanswerable(t *testing.T) {
	plain := t.TempDir()
	if got, err := gitutil.TrackedFiles(plain); err != nil || got != nil {
		t.Fatalf("TrackedFiles(plain dir) = %v, %v; want nil, nil", got, err)
	}
	if gitutil.RepoShaped(plain) {
		t.Error("RepoShaped(plain dir) = true, want false")
	}

	shaped := t.TempDir()
	if err := os.Mkdir(filepath.Join(shaped, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !gitutil.RepoShaped(shaped) {
		t.Error("RepoShaped(dir with .git) = false, want true")
	}
	if _, err := gitutil.TrackedFiles(shaped); err == nil {
		t.Error("TrackedFiles over a repo-shaped tree git cannot answer for returned no error — a caller would report a scan clean after reading zero files")
	}

	// The subdirectory of a repo-shaped tree is still repo-shaped: the walk
	// climbs, because git does.
	sub := filepath.Join(shaped, "docs")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if !gitutil.RepoShaped(sub) {
		t.Error("RepoShaped(subdirectory of a repo-shaped tree) = false, want true")
	}
}

// TestRootCommit pins the identity probe: the first root commit of a
// repository with history, and "" — never an error — outside one or before the
// first commit.
func TestRootCommit(t *testing.T) {
	repo := newRepo(t, "")
	if got := gitutil.RootCommit(repo); got != "" {
		t.Errorf("RootCommit(repo with no commits) = %q, want empty", got)
	}
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	want, err := runGit(t, repo, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := gitutil.RootCommit(repo); got != strings.TrimSpace(want) {
		t.Errorf("RootCommit = %q, want %q", got, strings.TrimSpace(want))
	}
	if got := gitutil.RootCommit(t.TempDir()); got != "" {
		t.Errorf("RootCommit(non-repo) = %q, want empty", got)
	}
}
