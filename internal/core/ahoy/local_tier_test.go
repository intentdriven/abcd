package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestInstallCreatesTheLocalTier pins that the local-ephemeral tier the mode
// store writes into (.abcd/.work.local/) is a gap install closes: absent on an
// adoptable repo, a real directory after install, and no gap thereafter.
func TestInstallCreatesTheLocalTier(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	var gap Gap
	for _, g := range det.Gaps {
		if g.ID == localTierGapID {
			gap = g
		}
	}
	if gap.ID == "" {
		t.Fatalf("no %s gap on a repo without the tier: %+v", localTierGapID, det.Gaps)
	}
	if !gap.Required || !gap.Resolvable || gap.Category != SafeAutocreate || gap.Scope != "repo" {
		t.Errorf("gap has the wrong contract: %+v", gap)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status = %q remaining=%v notes=%v", res.Status, res.Remaining, res.Notes)
	}
	tier := filepath.Join(repo, ".abcd", ".work.local")
	if !fsutil.IsRealDir(tier) {
		t.Fatalf("%s is not a real directory after install", tier)
	}
	if !strings.Contains(strings.Join(res.Writes, "\n"), ".abcd/.work.local") {
		t.Errorf("the tier is not on the receipt: %v", res.Writes)
	}
	after, _ := Detect(repo)
	if hasGap(after.Gaps, localTierGapID) {
		t.Error("the gap persists after the tier was created")
	}
}

// TestInstallRefusesASymlinkedLocalTier: the tier is created as a real
// directory and never reached through a symlink, so a planted link at the
// tier's path is refused with a note rather than followed.
func TestInstallRefusesASymlinkedLocalTier(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(repo, ".abcd", ".work.local")); err != nil {
		t.Fatal(err)
	}
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, localTierGapID) {
		t.Fatal("a symlink standing in for the tier must read as the tier missing")
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	var refused bool
	for _, n := range res.Notes {
		if strings.Contains(n, ".work.local") {
			refused = true
		}
	}
	if !refused {
		t.Errorf("no refusal note for the symlinked tier: %v", res.Notes)
	}
	fi, err := os.Lstat(filepath.Join(repo, ".abcd", ".work.local"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced or removed; abcd must not touch what it did not write")
	}
}

// TestLocalTierFenceCoversTheModeFile proves the gitignore fence covers the
// mode store's file in every visibility, including the narrowed public fence
// of a repo that commits its record tiers.
func TestLocalTierFenceCoversTheModeFile(t *testing.T) {
	const modeFile = ".abcd/.work.local/mode"
	for _, visibility := range []string{"private", "public"} {
		repo := gittest.NewRepo(t)
		if _, err := applyVisibilityBlock(repo.Root(), visibility); err != nil {
			t.Fatalf("%s: %v", visibility, err)
		}
		if !gitCheckIgnored(t, repo, modeFile) {
			t.Errorf("%s: git does not ignore %s", visibility, modeFile)
		}
	}
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/work/DECISIONS.md", "- a decision\n")
	repo.Commit("record tier")
	if _, err := applyVisibilityBlock(repo.Root(), "public"); err != nil {
		t.Fatal(err)
	}
	if !gitCheckIgnored(t, repo, modeFile) {
		t.Errorf("public (narrowed): git does not ignore %s", modeFile)
	}
}
