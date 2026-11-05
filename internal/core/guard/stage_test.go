package guard

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
