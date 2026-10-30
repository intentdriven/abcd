package landing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The status line's merging element, read (itd-2610150602547271,
// spc-2610161609350616 step 7). While the repository's facilitator.mode is
// auto, AutonomousMerging says merging is autonomous; under human, or where the
// mode cannot be read, it says nothing. The mode is read as the merge gate
// reads it: the default branch's committed .abcd/config.json and the machine
// layer, never the working tree's copy. How the row places the element, and
// that its switch stops the read, is core/statusline's to prove, with a fake
// reader (statusline.MergingMode); what is proved here is the reading.

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

func TestAutonomousMergingIsReadAsAutoFromTheDefaultBranch(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	auto, note := AutonomousMerging(context.Background(), root)
	if !auto {
		t.Error("auto committed on the default branch did not read as autonomous merging")
	}
	if note != "" {
		t.Errorf("note = %q, want none", note)
	}
}

func TestAutonomousMergingSaysNothingOtherwise(t *testing.T) {
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
		"human in the machine layer": func(t *testing.T) string {
			_, root := mergingRepo(t, autoConfig)
			writeMachineConfig(t, humanConfig)
			return root
		},
		"no default branch yet": func(t *testing.T) string {
			t.Setenv("HOME", t.TempDir())
			repo := gittest.NewRepo(t)
			repo.Write(".abcd/config.json", autoConfig)
			return repo.Git("rev-parse", "--show-toplevel")
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := setup(t)
			auto, note := AutonomousMerging(context.Background(), root)
			if auto {
				t.Error("read as autonomous merging")
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
			auto, note := AutonomousMerging(context.Background(), root)
			if auto {
				t.Error("read as autonomous merging over a mode that does not read")
			}
			if !strings.Contains(note, "DROPPED the merging element") || !strings.Contains(note, "facilitator.mode") {
				t.Fatalf("note = %q, want one dropping the merging element over facilitator.mode", note)
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

// TestTheMergingElementDropsWhenGitRunsOut: the mode is read with git's
// questions bounded by the row's own deadline; with that time spent the
// element drops and says why.
func TestTheMergingElementDropsWhenGitRunsOut(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	auto, note := AutonomousMerging(ctx, root)
	if auto {
		t.Error("read as autonomous merging with git's time spent")
	}
	if !strings.Contains(note, "DROPPED the merging element") {
		t.Errorf("note = %q, want the drop named", note)
	}
}
