package ahoy

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/vintage"
)

// gatePrompter answers every approval with confirm and every value question
// with its default, keeping the config value keys it was asked, as
// valueKeyPrompter does.
type gatePrompter struct {
	valueKeyPrompter
	confirm bool
}

func (p *gatePrompter) Confirm(string) bool { return p.confirm }

// TestWalkAndInstallStopAtTheSameEarlyGates holds the walk's second copy of
// install's early gates in step with install's own: every refusal install
// makes before its first value question (an unmanaged folder, the .abcd
// hazard, a stale binary, a retired docs target, a declined adoption, an
// unapproved settings change) is fed to both, and neither puts a value
// question. A gate added to install and left out of the walk shows here as
// the walk putting a question the install never asks. The control case, no
// gate, is the proof that both would ask.
func TestWalkAndInstallStopAtTheSameEarlyGates(t *testing.T) {
	yes := true
	for name, c := range map[string]struct {
		// arrange lays the gate out in repo, or returns false when the run is
		// in a folder that is not a repository.
		arrange func(t *testing.T, repo string) (gitRepo bool)
		opts    InstallOptions
		approve bool
		asked   bool
	}{
		"no gate (control)": {
			arrange: func(*testing.T, string) bool { return true },
			opts:    InstallOptions{Adopt: &yes, Yes: true}, approve: true, asked: true,
		},
		"an unmanaged folder": {
			arrange: func(*testing.T, string) bool { return false },
			opts:    InstallOptions{Adopt: &yes, Yes: true}, approve: true,
		},
		"the .abcd hazard": {
			arrange: func(t *testing.T, repo string) bool {
				if err := os.WriteFile(filepath.Join(repo, ".abcd"), []byte("not a directory\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return true
			},
			opts: InstallOptions{Adopt: &yes, Yes: true}, approve: true,
		},
		"a stale binary": {
			arrange: func(t *testing.T, _ string) bool {
				t.Cleanup(SetCurrentVintageForTest(func() vintage.Current { return vintage.Current{} }))
				return true
			},
			opts: InstallOptions{Adopt: &yes, Yes: true}, approve: true,
		},
		"a retired docs target": {
			arrange: func(*testing.T, string) bool { return true },
			opts:    InstallOptions{Adopt: &yes, Yes: true, ValueOverrides: map[string]string{"docs_target": "claude_md"}}, approve: true,
		},
		"a declined adoption": {
			arrange: func(*testing.T, string) bool { return true },
			opts:    InstallOptions{Yes: true}, approve: false,
		},
		"an unapproved settings change": {
			arrange: func(*testing.T, string) bool { return true },
			opts:    InstallOptions{Adopt: &yes}, approve: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			trufflehogResolves(t, false)
			repo := t.TempDir()
			if c.arrange(t, repo) {
				idMustGit(t, repo, "init")
			}
			answers := map[string]string{"visibility": "private"}
			walked := &gatePrompter{valueKeyPrompter{answers: answers}, c.approve}
			if err := WalkConfigValueQuestions(repo, c.opts, func(string) bool { return c.approve }, walked.Prompt); err != nil {
				t.Fatal(err)
			}
			installed := &gatePrompter{valueKeyPrompter{answers: answers}, c.approve}
			if _, err := Install(repo, c.opts, installed); err != nil {
				t.Fatal(err)
			}
			want := []string(nil)
			if c.asked {
				want = []string{"visibility", "docs_target"}
			}
			if !slices.Equal(walked.keys, want) || !slices.Equal(installed.keys, want) {
				t.Fatalf("walked %v, the install asked %v; want %v from both", walked.keys, installed.keys, want)
			}
		})
	}
}
