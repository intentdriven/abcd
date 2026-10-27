package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countingPrompter records every question it is asked and declines each.
type countingPrompter struct{ asked []string }

func (p *countingPrompter) Confirm(q string) bool { p.asked = append(p.asked, q); return false }
func (p *countingPrompter) Prompt(key string, _ []string, def string) string {
	p.asked = append(p.asked, key)
	return def
}
func (p *countingPrompter) AtTerminal() bool { return true }

// TestSetupStopsBeforeAnyQuestionWithoutTheBasics is itd-2610240823055889
// criterion 1: with no git, no repository or no agent harness there is
// nothing to set up, so setup stops before any question with one message
// naming what is missing, and writes nothing.
func TestSetupStopsBeforeAnyQuestionWithoutTheBasics(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (repo string, restore func())
		names string
	}{
		{"no git", func(t *testing.T) (string, func()) {
			return gitRepo(t), swapSetupBasics(func() bool { return false }, nil)
		}, "git is not installed"},
		{"no repository", func(t *testing.T) (string, func()) {
			return t.TempDir(), func() {}
		}, "not the top folder of a git repository"},
		{"no harness", func(t *testing.T) (string, func()) {
			return gitRepo(t), swapSetupBasics(nil, func() bool { return false })
		}, "no agent harness"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo, restore := tc.setup(t)
			defer restore()
			p := &countingPrompter{}
			res, err := Install(repo, InstallOptions{}, p)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.asked) != 0 {
				t.Errorf("asked %q before stopping", p.asked)
			}
			if res.Status != "refused" || len(res.Notes) != 1 || !strings.Contains(res.Notes[0], tc.names) {
				t.Fatalf("status %q, notes %q: want refused with one note naming %q", res.Status, res.Notes, tc.names)
			}
			if len(res.Writes) != 0 {
				t.Errorf("wrote %v", res.Writes)
			}
			if _, err := os.Stat(filepath.Join(repo, ".abcd")); err == nil {
				t.Error(".abcd/ was created")
			}
		})
	}
}

// TestGitInstalledReadsPath: the real check is the git command on PATH.
func TestGitInstalledReadsPath(t *testing.T) {
	restore := swapSetupBasics(nil, nil)
	defer restore()
	gitInstalled = shippedGitInstalled
	t.Setenv("PATH", t.TempDir())
	if gitInstalled() {
		t.Error("git reported installed with no git on PATH")
	}
}
