package landing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The board's merging element, read live (itd-2610150602547271,
// spc-2610161609350616 step 7). LiveMerging answers auto or human as the merge
// gate resolves facilitator.mode (the default branch's committed
// .abcd/config.json and the machine layer, never the working tree's copy), and
// not known where the mode cannot be read. How the row places the element, and
// that its switch stops the read, is core/statusline's to prove, with a fake
// reader; the status line's own reading, kept, is keptmode_test.go's.

// mergingRepo is a checkout whose default branch, main, commits config as its
// .abcd/config.json ("" commits none), with HOME sandboxed so the machine
// layer is the test's own. It returns the repository and its root as git
// names it (the temp area is reached through a symlink on macOS, and the note
// masks the root the status line resolved, which is git's).
func mergingRepo(t *testing.T, config string) (*gittest.Repo, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	if config != "" {
		repo.Write(".abcd/config.json", config)
	}
	repo.Commit("root")
	return repo, repo.Git("rev-parse", "--show-toplevel")
}

// writeMachineConfig writes config as the machine layer of the test's HOME.
func writeMachineConfig(t *testing.T, config string) {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLiveMergingIsReadAsAutoFromTheDefaultBranch(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	m, note := LiveMerging(context.Background(), root)
	if m.State != statusline.MergingAuto {
		t.Errorf("auto committed on the default branch read as %q", m)
	}
	if note != "" {
		t.Errorf("note = %q, want none", note)
	}
}

func TestLiveMergingIsHumanOrNotKnownOtherwise(t *testing.T) {
	cases := map[string]func(t *testing.T) string{
		"human committed": func(t *testing.T) string {
			_, root := mergingRepo(t, humanConfig)
			return root
		},
		"no mode anywhere": func(t *testing.T) string {
			_, root := mergingRepo(t, "")
			return root
		},
		"auto only in the working tree": func(t *testing.T) string {
			repo, root := mergingRepo(t, humanConfig)
			repo.Write(".abcd/config.json", autoConfig)
			return root
		},
		"human committed over auto in the machine layer": func(t *testing.T) string {
			_, root := mergingRepo(t, humanConfig)
			writeMachineConfig(t, autoConfig)
			return root
		},
		"a default branch that cannot be told": func(t *testing.T) string {
			repo, root := mergingRepo(t, autoConfig)
			repo.Git("branch", "-m", "side")
			return root
		},
		"no commit yet": func(t *testing.T) string {
			t.Setenv("HOME", t.TempDir())
			repo := gittest.NewRepo(t)
			repo.Write(".abcd/config.json", autoConfig)
			return repo.Git("rev-parse", "--show-toplevel")
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := setup(t)
			m, note := LiveMerging(context.Background(), root)
			want := statusline.MergingHuman
			if strings.Contains(name, "cannot be told") {
				want = statusline.MergingUnknown
			}
			if m.State != want {
				t.Errorf("read as %q, want %q", m.State, want)
			}
			if note != "" {
				t.Errorf("note = %q, want none", note)
			}
		})
	}
}

func TestAnUnreadableModeDropsTheMergingElementAndSaysSo(t *testing.T) {
	cases := map[string]string{
		"a value that is not a mode": `{"facilitator": {"mode": "sometimes"}}`,
		"a hostile key":              `{"facilitator": {"mode": "auto", "mo\u001b[31mde": "auto"}}`,
	}
	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			_, root := mergingRepo(t, config)
			m, note := LiveMerging(context.Background(), root)
			if m.State != statusline.MergingUnknown {
				t.Errorf("a mode that does not read read as %q", m)
			}
			if !strings.Contains(note, "facilitator.mode was not read") || !strings.Contains(note, "facilitator.mode") {
				t.Fatalf("note = %q, want one saying facilitator.mode was not read", note)
			}
			if strings.ContainsRune(note, 0x1b) {
				t.Errorf("the note carries a raw escape byte: %q", note)
			}
			if home := os.Getenv("HOME"); strings.Contains(note, home) || strings.Contains(note, root) {
				t.Errorf("the note carries a local path: %q", note)
			}
		})
	}
}

// TestTheMergingElementIsNotReadWhenGitRunsOut: the mode is read with git's
// questions bounded by the board's own deadline; with that time spent the
// mode is not read and the note says why.
func TestTheMergingElementIsNotReadWhenGitRunsOut(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, note := LiveMerging(ctx, root)
	if m.State != statusline.MergingUnknown {
		t.Errorf("with git's time spent the mode read as %q", m)
	}
	if !strings.Contains(note, "facilitator.mode was not read") {
		t.Errorf("note = %q, want the unread mode named", note)
	}
}
