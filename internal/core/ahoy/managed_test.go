package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// committedRepo stands up a git checkout with one commit, so it has a root
// commit the history index can register.
func committedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(gittest.Env(t),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v unavailable: %v (%s)", args, err, out)
		}
	}
	run("init", "-q", "--initial-branch=main")
	// The content is the directory itself, so two fixtures made in the same
	// second by the same identity do not hash to the same root commit.
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte(dir), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "f")
	run("commit", "-q", "-m", "one")
	return dir
}

// TestManagedAgreesWithClassify pins the classifier's contract: Managed is
// true exactly where classify says ManagedRepo — a marker block in CLAUDE.md
// or AGENTS.md at the checkout root, or registration in the history index —
// and false everywhere else, a bare `.abcd/` and a plain git checkout
// included (iss-88). It is asserted against classify itself on every fixture,
// so the two cannot drift.
func TestManagedAgreesWithClassify(t *testing.T) {
	setupHermetic(t)

	plain := t.TempDir()
	bareAbcd := t.TempDir()
	if err := os.Mkdir(filepath.Join(bareAbcd, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	agents := t.TempDir()
	if err := os.WriteFile(filepath.Join(agents, "AGENTS.md"), []byte("# P\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unregistered := committedRepo(t)
	registered := committedRepo(t)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}
	sha := gitutil.RootCommit(registered)
	if sha == "" {
		t.Fatal("the registered fixture has no root commit")
	}
	if err := writeHistoryIndex(&historyIndex{Schema: 1, Repos: []historyRepo{{RootCommit: sha, Path: registered, Status: "active"}}}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		dir  string
		want bool
	}{
		{"plain directory", plain, false},
		{"bare .abcd is not a managed signal", bareAbcd, false},
		{"marker in CLAUDE.md", managedRepo(t), true},
		{"marker in AGENTS.md", agents, true},
		{"git checkout, no marker, not registered", unregistered, false},
		{"git checkout registered in the history index", registered, true},
	}
	idx, _ := loadHistoryIndex()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, _ := classify(tc.dir, deriveIdentity(tc.dir), idx)
			if (kind == ManagedRepo) != tc.want {
				t.Fatalf("fixture is wrong: classify = %s, the case expects managed=%v", kind, tc.want)
			}
			if got := Managed(tc.dir); got != tc.want {
				t.Errorf("Managed = %v, want %v (classify says %s)", got, tc.want, kind)
			}
		})
	}
}

// TestManagedIsCheapWhereTheMarkerAnswers: a marker block at the root settles
// the question before any git subprocess is needed, so the status verb —
// which calls this on every refresh — never pays for a git call in the
// ordinary managed case. Proven by making git unreachable and asserting the
// marker still answers.
func TestManagedIsCheapWhereTheMarkerAnswers(t *testing.T) {
	setupHermetic(t)
	dir := managedRepo(t)
	t.Setenv("PATH", t.TempDir())
	if !Managed(dir) {
		t.Fatal("a marker block must answer without git")
	}
	if strings.Contains(os.Getenv("PATH"), "usr") {
		t.Fatal("test precondition: PATH still reaches system tools")
	}
}

// TestManagedNeverWrites: Managed is a read on every status refresh, and the
// tree it reads must be exactly as it found it.
func TestManagedNeverWrites(t *testing.T) {
	setupHermetic(t)
	dir := committedRepo(t)
	before := treeListing(t, dir)
	_ = Managed(dir)
	if after := treeListing(t, dir); after != before {
		t.Fatalf("Managed changed the tree:\nbefore: %s\nafter:  %s", before, after)
	}
}

func treeListing(t *testing.T, dir string) string {
	t.Helper()
	var names []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			names = append(names, p+":"+fi.ModTime().String())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(names, "\n")
}
