package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// stageFiles is a walled stage's reading: the person's home, a walled directory,
// and a shell home apart from it.
func stageFiles(t *testing.T) (f *Files, profile, shellHome string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	profile = abcdhome.Path(home, "profiles", "walled")
	shellHome = abcdhome.Path(home, "stage-homes", "walled")
	return NewFiles(t.TempDir()).WithStage(shellHome, []string{profile}, []string{"docs/page.md", ".abcd/work/issues"}), profile, shellHome
}

// TestAStagesHomeIsTheOneItsShellExpands: a word that opens with the home names
// the stage's, and the walled directory is protected however it is spelled.
func TestAStagesHomeIsTheOneItsShellExpands(t *testing.T) {
	f, profile, _ := stageFiles(t)
	for _, c := range []struct {
		line    string
		allowed bool
	}{
		{`printf x > ~/notes.txt`, true},
		{`printf x > ~/.gitconfig`, false},
		{`printf x > ~/.zshrc`, false},
		{`printf x > ~/../settings.json`, false},
		{`printf x > "$HOME/../../profiles/walled/settings.json"`, false},
		{`printf x > ~/../../profiles/walled/settings.json`, false},
		{`printf x > ` + filepath.Join(profile, "settings.json"), false},
		{`printf x > ` + filepath.Join(profile, "stage-env.sh"), false},
	} {
		if a := approveShell(clean(), f, c.line); a.Allow != c.allowed {
			t.Errorf("%q: allowed %v, want %v (%s)", c.line, a.Allow, c.allowed, a.Reason)
		}
	}
}

// TestAStageStartsNoFurtherAgent: no second harness and no drive from a stage's
// shell, whatever precedes the program; another session is not judged this way.
func TestAStageStartsNoFurtherAgent(t *testing.T) {
	f, _, _ := stageFiles(t)
	for _, line := range []string{
		`claude -p hello`,
		`/usr/local/bin/claude -p hello`,
		`env A=1 claude -p hello`,
		`nohup claude -p hello`,
		`abcd drain --drive --yes`,
		`/opt/bin/abcd drain --max 3 --drive`,
		`true && claude --version`,
	} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("%q was approved on a walled stage", line)
		}
	}
	for _, line := range []string{`git log --grep claude`, `abcd drain --dry-run`, `printf claude`} {
		if a := approveShell(clean(), f, line); !a.Allow {
			t.Errorf("%q was refused on a walled stage: %s", line, a.Reason)
		}
	}
	if a := approveShell(clean(), NewFiles(t.TempDir()), `claude --version`); !a.Allow {
		t.Errorf("another session is not judged this way: %s", a.Reason)
	}
}

// TestAStageWritesNoGateFileItDoesNotDeclare: what decides how a lane is judged
// is not a stage's to write, unless its footprint declares it; its own local tier
// is.
func TestAStageWritesNoGateFileItDoesNotDeclare(t *testing.T) {
	f, _, _ := stageFiles(t)
	for _, c := range []struct {
		line    string
		allowed bool
	}{
		{`printf x > Makefile`, false},
		{`printf x > .abcd/rules.json`, false},
		{`printf x > .abcd/guard.json`, false},
		{`printf x > .githooks/pre-push`, false},
		{`printf x > hooks/pre-tool-use.sh`, false},
		{`printf x > src/hooks/use.ts`, true},
		{`printf x > docs/page.md`, true},
		{`printf x > .abcd/work/issues/a.md`, true},
		{`printf x > .abcd/.work.local/scratch.txt`, true},
	} {
		if a := approveShell(clean(), f, c.line); a.Allow != c.allowed {
			t.Errorf("%q: allowed %v, want %v (%s)", c.line, a.Allow, c.allowed, a.Reason)
		}
	}
	dir := f.dir
	declared := []string{"docs/page.md"}
	for _, c := range []struct {
		path string
		ok   bool
	}{
		{"a.go", true},
		{"Makefile", false},
		{".abcd/rules.json", false},
		{".abcd/.work.local/x", true},
		{"docs/page.md", true},
	} {
		why := StageWriteRefusal([]string{dir}, declared, filepath.Join(dir, c.path))
		if (why == "") != c.ok {
			t.Errorf("%s: %q", c.path, why)
		}
	}
}

// TestAStageNeverMovesItsHomeOrReachesTheKeychain: a line that sets HOME or names
// the person's home is refused, as is the keychain tool, and a path spelled with
// $HOME is refused as unfixed rather than judged.
func TestAStageNeverMovesItsHomeOrReachesTheKeychain(t *testing.T) {
	f, _, _ := stageFiles(t)
	for _, line := range []string{
		`HOME=/x git status`,
		`export HOME=/x`,
		`env HOME=/x git status`,
		`printf %s "$ABCD_PERSON_HOME"`,
		`security find-generic-password -s x`,
		`/usr/bin/security list-keychains`,
		`printf x > "$HOME/notes.txt"`,
	} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("%q was approved on a stage", line)
		}
	}
	if a := approveShell(clean(), f, `MY_HOME=/x git status`); !a.Allow {
		t.Errorf("another variable is no home: %s", a.Reason)
	}
	if a := approveShell(clean(), NewFiles(t.TempDir()), `HOME=/x git status`); !a.Allow {
		t.Errorf("another session is not judged this way: %s", a.Reason)
	}
}

// TestAStageWritesOnlyWhereItWorks: its tree, its run directories, its home and
// temporary files, and nowhere else, another checkout's run state among them. A
// test's own directory lies in the temporary directory on every system, /tmp being
// open whatever TMPDIR says, so each place here is named outside it; none need
// exist, since a write is judged by where its path lies.
func TestAStageWritesOnlyWhereItWorks(t *testing.T) {
	const (
		tree      = "/nonexistent-stage/tree"
		run       = "/nonexistent-stage/run"
		shellHome = "/nonexistent-stage/home"
		temporary = "/nonexistent-stage/temporary"
		other     = "/nonexistent-stage/other-checkout"
	)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TMPDIR", temporary)
	f := NewFiles(tree).WithStage(shellHome, nil, nil).WithRunDirs(run)
	for _, c := range []struct {
		line    string
		allowed bool
	}{
		{`printf x > a.go`, true},
		{`printf x > ` + filepath.Join(tree, "a.go"), true},
		{`printf x > ` + filepath.Join(run, "receipt.json"), true},
		{`printf x > ~/notes.txt`, true},
		{`printf x > ` + filepath.Join(temporary, "scratch"), true},
		{`printf x > /tmp/scratch`, true},
		{`printf x > /dev/null`, true},
		{`printf x > ` + filepath.Join(other, ".abcd", ".work.local", "run", "state.json"), false},
		{`printf x > ` + filepath.Join(other, "a.go"), false},
	} {
		if a := approveShell(clean(), f, c.line); a.Allow != c.allowed {
			t.Errorf("%q: allowed %v, want %v (%s)", c.line, a.Allow, c.allowed, a.Reason)
		}
	}
}

// TestAWorksFootprintNeverOpensAGateFile: what a record's found_at names is
// declared, and that opens records under .abcd, never what judges a lane.
func TestAWorksFootprintNeverOpensAGateFile(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []struct {
		path string
		ok   bool
	}{
		{".githooks/commit-msg", false},
		{"hooks/pre-push", false},
		{"Makefile", false},
		{".abcd/rules.json", false},
		{".abcd/guard.json", false},
		{".abcd/config.json", false},
		{".abcd/.work.local/preflight-receipts/abc", false},
		{".abcd/work/issues/open/a.md", true},
	} {
		why := StageWriteRefusal([]string{dir}, []string{c.path}, filepath.Join(dir, c.path))
		if (why == "") != c.ok {
			t.Errorf("%s declared: %q", c.path, why)
		}
	}
	if why := StageWriteRefusal([]string{dir}, nil, filepath.Join(dir, ".abcd/work/issues/open/a.md")); why == "" {
		t.Error("a record its work does not declare stays refused")
	}
}

// TestAConfiguredHooksPathIsProtectedFromAStage: both ways of writing, whatever
// the repository calls its hooks.
func TestAConfiguredHooksPathIsProtectedFromAStage(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Git("config", "core.hooksPath", "tools/hooks")
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := NewFiles(repo.Root()).WithStage(abcdhome.Path(home, "stage-homes", "walled", "none", "home"), nil, nil)
	if a := approveShell(clean(), f, `printf x > tools/hooks/commit-msg`); a.Allow {
		t.Error("a line wrote the configured hooks")
	}
	if why := StageWriteRefusal([]string{repo.Root()}, nil, filepath.Join(repo.Root(), "tools", "hooks", "commit-msg")); why == "" {
		t.Error("a file tool wrote the configured hooks")
	}
}

// TestAStageDoesNotWriteWhatArmsAMerge: a local tier deeper in the tree is judged the
// same.
func TestAStageDoesNotWriteWhatArmsAMerge(t *testing.T) {
	tree := t.TempDir()
	for _, rel := range []string{
		".abcd/.work.local/approvals/abc",
		".abcd/.work.local/review-receipts/abc",
		".abcd/.work.local/preflight-receipts/abc",
		"nested/.abcd/.work.local/approvals/abc",
		"nested/deeper/.abcd/.work.local/review-receipts/abc",
	} {
		if StageWriteRefusal([]string{tree}, nil, filepath.Join(tree, rel)) == "" {
			t.Errorf("a stage may write %s", rel)
		}
	}
	for _, rel := range []string{".abcd/.work.local/scratch/abc", "internal/approvals/abc", "nested/review-receipts/abc"} {
		if why := StageWriteRefusal([]string{tree}, nil, filepath.Join(tree, rel)); why != "" {
			t.Errorf("%s is refused: %s", rel, why)
		}
	}
}

// TestAStageDoesNotLand: the loop lands, a stage does not; the verbs a stage's work
// needs stay open.
func TestAStageDoesNotLand(t *testing.T) {
	home := t.TempDir()
	stage := NewFiles(t.TempDir()).WithStage(filepath.Join(home, "stage"), nil, nil)
	for _, line := range []string{
		`abcd land`,
		`abcd land --record-review return.md`,
		`abcd land --approve 12`,
		`abcd --json land`,
		`scripts/abcd.sh land --record-review return.md`,
		`true && abcd land`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage may run %q", line)
		}
	}
	for _, line := range []string{`abcd rules land`, `abcd capture "land the thing"`, `abcd implement load --wait --narrow -- go test ./x`, `abcd peers`} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
	if a := approveShell(clean(), NewFiles(t.TempDir()), `abcd land`); !a.Allow {
		t.Errorf("outside a stage the verb is the person's: %s", a.Reason)
	}
}

// TestAStageMovesNoRun: `implement load` is how its tests run and `implement widen` how an
// issue lane holds a file its brief lets it reach, so both stay open; every other
// `implement` verb is the loop's, and outside a stage nothing changes.
func TestAStageMovesNoRun(t *testing.T) {
	home := t.TempDir()
	stage := NewFiles(t.TempDir()).WithStage(filepath.Join(home, "stage"), nil, nil)
	for _, line := range []string{
		`abcd implement receipt return.md`,
		`abcd implement step`,
		`abcd implement step --run run-1 --discard lane-1`,
		`abcd --json implement status`,
		`abcd implement`,
		`scripts/abcd.sh implement receipt return.md`,
		`true && abcd implement step`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage may run %q", line)
		}
	}
	for _, line := range []string{`abcd implement load --wait --narrow -- go test ./x`, `scripts/abcd.sh implement load --wait -- make check`, `abcd rules implement`,
		`abcd implement widen --run run-1 --lane lane-1 docs/page.md`, `/opt/bin/abcd implement widen --run run-1 --lane lane-1 docs/page.md`,
		`scripts/abcd.sh implement widen --run run-1 --lane lane-1 docs/page.md`} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
	if a := approveShell(clean(), NewFiles(t.TempDir()), `abcd implement receipt return.md`); !a.Allow {
		t.Errorf("a host session's stage hands back as it did: %s", a.Reason)
	}
}

// TestAStageAddsNoWorktree: a lane has its own worktree.
func TestAStageAddsNoWorktree(t *testing.T) {
	f, _, _ := stageFiles(t)
	for _, line := range []string{
		`git worktree add ../other`,
		`git worktree move ../one ../other`,
		`git -C /tmp/lane worktree add ../other`,
		`git -C /tmp/lane worktree move ../one ../other`,
		`git --git-dir=/tmp/lane/.git worktree add ../other`,
		`git -c core.pager=cat worktree add -b new ../other`,
		`/usr/bin/git worktree add ../other`,
		`env A=1 git worktree add ../other`,
		`true && git worktree add ../other`,
		`git worktree "$WHAT" ../other`,
		`git "$WHAT" add ../other`,
		`git --attr-source HEAD worktree add ../other`,
		`git WORKTREE add ../other`,
		`git-worktree add ../other`,
		`find . -exec git worktree add ../other ;`,
	} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("%q was approved on a walled stage", line)
		}
	}
	// The rule itself, whatever an earlier reading of the line makes of a change of directory.
	for line, want := range map[string]bool{
		`git worktree add x`:                 true,
		`git worktree move a b`:              true,
		`git -C /tmp/lane worktree add x`:    true,
		`git -C /tmp/lane worktree move a b`: true,
		`git-worktree add x`:                 true,
		`git worktree list`:                  false,
		`git -C /tmp/lane status`:            false,
		`git log --grep worktree`:            false,
		`git add worktree`:                   false,
		`printf git worktree add`:            false,
	} {
		if got := addsAWorktree(strings.Fields(line), 0); got != want {
			t.Errorf("%q: addsAWorktree = %v, want %v", line, got, want)
		}
	}
	for _, line := range []string{`git worktree list`, `git log --grep worktree`, `git add worktree`} {
		if a := approveShell(clean(), f, line); !a.Allow {
			t.Errorf("%q was refused on a walled stage: %s", line, a.Reason)
		}
	}
	if a := approveShell(clean(), NewFiles(t.TempDir()), `git worktree list`); !a.Allow {
		t.Errorf("another session is not judged this way: %s", a.Reason)
	}
}

// TestAStageOnOneProfileWritesNoSiblingProfile: a stage names only its own profile to
// protect, and a sibling's login and settings stay out of its reach all the same,
// since a stage writes only under its own directories.
func TestAStageOnOneProfileWritesNoSiblingProfile(t *testing.T) {
	f, profile, _ := stageFiles(t)
	sibling := filepath.Join(filepath.Dir(profile), "sibling")
	if err := os.MkdirAll(sibling, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		`printf x > ` + filepath.Join(sibling, "settings.json"),
		`printf x > ~/../../profiles/sibling/settings.json`,
		`rm -rf ` + sibling,
		`cp /dev/null ` + filepath.Join(sibling, ".credentials.json"),
	} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("%q was approved on a walled stage", line)
		}
	}
}
