package mode_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

func markerPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(mode.QuestionOpenRelPath))
}

// TestHasTierIsTheWritersOwnTest: the tier is present only as a real directory
// in this checkout. A symlink standing in for it is not the tier, for the same
// reason SetAt refuses one.
func TestHasTierIsTheWritersOwnTest(t *testing.T) {
	root := newRepo(t)
	if mode.HasTier(root) {
		t.Fatal("HasTier is true in a checkout with no tier")
	}
	makeTier(t, root)
	if !mode.HasTier(root) {
		t.Fatal("HasTier is false in a checkout that has the tier")
	}

	linked := newRepo(t)
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(linked, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(linked, filepath.FromSlash(mode.TierRelPath))); err != nil {
		t.Fatal(err)
	}
	if mode.HasTier(linked) {
		t.Fatal("HasTier accepted a symlink standing in for the tier")
	}
}

// TestMarkQuestionOpenWritesTheMarkerInTheTier: the marker is one file in the
// local tier, and it is never created on the way to a write — a checkout with
// no tier is refused with the tier's own error and nothing is minted.
func TestMarkQuestionOpenWritesTheMarkerInTheTier(t *testing.T) {
	root := newRepo(t)
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); !errors.Is(err, mode.ErrNoLocalTier) {
		t.Fatalf("MarkQuestionOpen without the tier = %v, want ErrNoLocalTier", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused mark created .abcd/: %v", err)
	}

	makeTier(t, root)
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); err != nil {
		t.Fatalf("MarkQuestionOpen: %v", err)
	}
	data, err := os.ReadFile(markerPath(root))
	if err != nil {
		t.Fatalf("the marker was not written: %v", err)
	}
	if string(data) != "product-thinker\n" {
		t.Fatalf("marker = %q, want the addressed state", data)
	}
	if open, err := mode.QuestionOpen(root); err != nil || !open {
		t.Fatalf("QuestionOpen = %v, %v; want true, nil", open, err)
	}
}

// TestResetOnAnswerResetsAndClears is criterion 4 at the store: with a
// question open, the answer resets the mode to managed and clears the marker,
// and says it did; with none open it changes nothing, so a state the human set
// by hand survives their next message.
func TestResetOnAnswerResetsAndClears(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}

	reset, err := mode.ResetOnAnswer(root)
	if err != nil || reset {
		t.Fatalf("ResetOnAnswer with no question open = %v, %v; want false, nil", reset, err)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Fatalf("no question was open, yet the mode moved to %q", got)
	}

	if err := mode.MarkQuestionOpen(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	reset, err = mode.ResetOnAnswer(root)
	if err != nil || !reset {
		t.Fatalf("ResetOnAnswer with a question open = %v, %v; want true, nil", reset, err)
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Fatalf("mode after the answer = %q, want managed", got)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the marker survived the reset: %v", err)
	}

	// A second message is not a second answer.
	if reset, err := mode.ResetOnAnswer(root); err != nil || reset {
		t.Fatalf("second ResetOnAnswer = %v, %v; want false, nil", reset, err)
	}
}

// TestResetOnAnswerOutsideAManagedTree: a checkout with no tier has no marker
// and nothing to reset, and the reset creates nothing there.
func TestResetOnAnswerOutsideAManagedTree(t *testing.T) {
	root := newRepo(t)
	if reset, err := mode.ResetOnAnswer(root); err != nil || reset {
		t.Fatalf("ResetOnAnswer without a tier = %v, %v; want false, nil", reset, err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the reset created .abcd/: %v", err)
	}
}
