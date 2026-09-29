package ahoy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/gittest"
)

// noWrites asserts the install left none of its evidence artefacts — a refusal
// before the first apply step must not touch the repo.
func noWrites(t *testing.T, repo string) {
	t.Helper()
	for _, rel := range []string{".abcd/config.json", ".abcd/rules.json", "CLAUDE.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(repo, rel)); err == nil {
			t.Errorf("refusal still wrote %s", rel)
		}
	}
}

func TestInstallRefusesUnknownVintage(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := currentVintage
	t.Cleanup(func() { currentVintage = orig })
	currentVintage = func() vintage.Current { return vintage.Current{Known: false} }

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" {
		t.Fatalf("status = %q, want refused", res.Status)
	}
	if len(res.Writes) != 0 {
		t.Fatalf("refusal reported writes: %v", res.Writes)
	}
	if len(res.Notes) == 0 {
		t.Fatal("a refusal must record its reason in Notes")
	}
	noWrites(t, repo)
}

func TestInstallRefusesStaleAgainstTip(t *testing.T) {
	setupHermetic(t)
	r := gittest.NewRepo(t)
	r.Commit("first")
	first := r.Git("rev-parse", "HEAD")
	r.Commit("second") // advance the tip past the binary's revision

	orig := currentVintage
	t.Cleanup(func() { currentVintage = orig })
	currentVintage = func() vintage.Current { return vintage.Current{Revision: first, Known: true} }

	res, err := Install(r.Root(), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" {
		t.Fatalf("status = %q, want refused", res.Status)
	}
	if len(res.Writes) != 0 {
		t.Fatalf("refusal reported writes: %v", res.Writes)
	}
	noWrites(t, r.Root())
}

func TestInstallOverrideProceedsThroughStaleBinary(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := currentVintage
	t.Cleanup(func() { currentVintage = orig })
	currentVintage = func() vintage.Current { return vintage.Current{Known: false} }

	opts := installOpts()
	opts.AllowStaleBinary = true
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "refused" {
		t.Fatalf("override should proceed, got refused (notes=%v)", res.Notes)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "config.json")); err != nil {
		t.Errorf("override install wrote nothing: %v", err)
	}
}

func TestInstallProceedsThroughFreshBinary(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	// A bare .git dir: adoptable, but git cannot answer for a tip, so the binary
	// is not a dogfood-stale one. A determinable vintage here must proceed.
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := currentVintage
	t.Cleanup(func() { currentVintage = orig })
	currentVintage = func() vintage.Current {
		return vintage.Current{Revision: "1111111111111111111111111111111111111111", Known: true}
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "refused" {
		t.Fatalf("a fresh, determinable binary must proceed, got refused (notes=%v)", res.Notes)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "config.json")); err != nil {
		t.Errorf("fresh install wrote nothing: %v", err)
	}
}

// TestStaleRefusalPrecedesTheBinDirProbe pins AC2's "refuses before any write"
// under an explicit --bin-dir (iss-2609291942529461). The writability probe
// creates and removes a temp file in the named directory (or, when it does not
// exist yet, in its nearest existing parent), so a stale binary that probed
// first touched the filesystem before refusing. Creating or removing an entry
// moves the directory's modification time, so a watched directory whose mtime
// is pinned in the past shows whether the probe ever ran.
func TestStaleRefusalPrecedesTheBinDirProbe(t *testing.T) {
	cases := []struct {
		name string
		// binDir returns the --bin-dir to pass and the directory the probe
		// would write into.
		binDir func(root string) (binDir, watched string)
	}{
		{"existing bin dir", func(root string) (string, string) { return root, root }},
		{"bin dir to be created", func(root string) (string, string) { return filepath.Join(root, "opt", "bin"), root }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			binDir, watched := tc.binDir(t.TempDir())
			past := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
			if err := os.Chtimes(watched, past, past); err != nil {
				t.Fatal(err)
			}
			orig := currentVintage
			t.Cleanup(func() { currentVintage = orig })
			currentVintage = func() vintage.Current { return vintage.Current{Known: false} }

			opts := installOpts()
			opts.BinDir = binDir
			res, err := Install(repo, opts, RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != "refused" {
				t.Fatalf("status = %q, want refused", res.Status)
			}
			fi, err := os.Stat(watched)
			if err != nil {
				t.Fatal(err)
			}
			if !fi.ModTime().Equal(past) {
				t.Errorf("the --bin-dir probe touched %s before the stale refusal (mtime moved to %v)", watched, fi.ModTime())
			}
			entries, err := os.ReadDir(watched)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("the refusal left entries in %s: %v", watched, entries)
			}
			noWrites(t, repo)
		})
	}
}
