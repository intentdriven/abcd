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
	if open, err := mode.QuestionOpenForTest(root); err != nil || !open {
		t.Fatalf("QuestionOpenForTest = %v, %v; want true, nil", open, err)
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

// TestResetOnAnswerRefusesAMarkerItCannotRemove (iss-2609260100393814): a
// directory planted at the marker's path is nothing the gate wrote and nothing
// the reset can clear, so resetting on it would reset a hand-set mode on every
// message that follows and never clear the cause. The reset refuses it, names
// it, and leaves the mode as it was, on every message.
func TestResetOnAnswerRefusesAMarkerItCannotRemove(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)
	if err := os.MkdirAll(filepath.Join(markerPath(root), "planted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		reset, err := mode.ResetOnAnswer(root)
		if err == nil || reset {
			t.Fatalf("message %d: ResetOnAnswer on a planted directory = %v, %v; want false and an error", i+1, reset, err)
		}
		if got, _ := mode.ReadAt(root); got != mode.Facilitator {
			t.Fatalf("message %d: the hand-set mode moved to %q", i+1, got)
		}
	}
}

// TestResetOnAnswerClearsAStaleMarker: the reset applies only to the mode the
// question was asked under. A question refused under the product thinker and
// abandoned leaves its marker; the mode is then set to managed and later to
// the facilitator by hand. The next message must not reset that hand-set
// mode: the marker names another mode, so it is stale, and is cleared with
// the mode left as set. A marker that is not a regular file is nothing the
// gate wrote, and is cleared the same way.
func TestResetOnAnswerClearsAStaleMarker(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	for _, st := range []mode.State{mode.Managed, mode.Facilitator} {
		if err := mode.SetAt(root, st); err != nil {
			t.Fatal(err)
		}
	}
	reset, err := mode.ResetOnAnswer(root)
	if err != nil || reset {
		t.Fatalf("ResetOnAnswer over a stale marker = %v, %v; want false, nil", reset, err)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Errorf("a stale marker reset the hand-set mode to %q", got)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the stale marker survived: %v", err)
	}

	if err := os.Symlink(filepath.Join(root, "elsewhere"), markerPath(root)); err != nil {
		t.Fatal(err)
	}
	reset, err = mode.ResetOnAnswer(root)
	if err != nil || reset {
		t.Fatalf("ResetOnAnswer over a symlinked marker = %v, %v; want false, nil", reset, err)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Errorf("a symlinked marker reset the mode to %q", got)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the symlinked marker survived: %v", err)
	}
}

// TestResetOnAnswerResetsOverAnUnreadableStore: a store that cannot be read
// is no evidence of a mode set since the question, so the marker is not
// stale: the reset writes managed and clears the marker, as it did before the
// marker's word was compared. Where the store cannot be written either, the
// reset says so and keeps the marker for the next message, never clearing it
// silently with the mode left parked.
func TestResetOnAnswerResetsOverAnUnreadableStore(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(storePath(root), []byte("bogus\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mode.ReadAt(root); err == nil {
		t.Fatal("the fixture's store reads; want an unknown word it refuses")
	}
	reset, err := mode.ResetOnAnswer(root)
	if err != nil || !reset {
		t.Fatalf("ResetOnAnswer over a store holding an unknown word = %v, %v; want true, nil", reset, err)
	}
	if got, err := mode.ReadAt(root); err != nil || got != mode.Managed {
		t.Errorf("mode after the reset = %q, %v; want managed", got, err)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the marker survived the reset: %v", err)
	}

	if err := os.Remove(storePath(root)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(storePath(root), "planted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	reset, err = mode.ResetOnAnswer(root)
	if err == nil || reset {
		t.Fatalf("ResetOnAnswer over a store it can neither read nor write = %v, %v; want false and an error", reset, err)
	}
	if _, err := os.Lstat(markerPath(root)); err != nil {
		t.Errorf("the marker was cleared with the mode not reset: %v", err)
	}
}

// TestClearQuestionOpenOtherThanKeepsAMarkerForTheStateSet: an explicit mode
// set clears a marker naming another state, and changes no mode; a marker
// naming the state being set is kept, so a retry that sets the mode again
// beside the question it asks still resets on the answer.
func TestClearQuestionOpenOtherThanKeepsAMarkerForTheStateSet(t *testing.T) {
	root := newRepo(t)
	if err := mode.ClearQuestionOpenOtherThan(root, mode.Managed); err != nil {
		t.Fatalf("with no tier: %v", err)
	}
	makeTier(t, root)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.ProductThinker); err != nil {
		t.Fatalf("with no marker: %v", err)
	}
	if err := mode.MarkQuestionOpen(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(markerPath(root)); err != nil {
		t.Errorf("a marker naming the state being set was cleared: %v", err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a marker naming another state survived: %v", err)
	}
	if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
		t.Errorf("clearing the marker moved the mode to %q", got)
	}
}

// TestClearQuestionOpenOtherThanChecksItsTarget: like MarkQuestionOpen, the
// clear acts only where the tier is a real directory, so a tier symlinked to
// another directory of the checkout has that directory's question_open left
// alone. At the marker's path it removes a symlink as itself, and names and
// leaves a directory, which the gate never writes.
func TestClearQuestionOpenOtherThanChecksItsTarget(t *testing.T) {
	root := newRepo(t)
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	bystander := filepath.Join(other, "question_open")
	if err := os.WriteFile(bystander, []byte("facilitator\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tier := filepath.Join(root, filepath.FromSlash(mode.TierRelPath))
	if err := os.Symlink(other, tier); err != nil {
		t.Fatal(err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.Managed); err != nil {
		t.Fatalf("behind a symlinked tier: %v", err)
	}
	if _, err := os.Lstat(bystander); err != nil {
		t.Errorf("the clear removed a file behind a symlinked tier: %v", err)
	}
	if err := os.Remove(tier); err != nil {
		t.Fatal(err)
	}

	makeTier(t, root)
	target := filepath.Join(root, "kept")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, markerPath(root)); err != nil {
		t.Fatal(err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.Managed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(markerPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a symlink at the marker's path survived: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("the clear removed what the symlink pointed at: %v", err)
	}

	if err := os.Mkdir(markerPath(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.ClearQuestionOpenOtherThan(root, mode.Managed); err == nil {
		t.Error("a directory at the marker's path was not named")
	}
	if fi, err := os.Lstat(markerPath(root)); err != nil || !fi.IsDir() {
		t.Errorf("the directory at the marker's path was removed: %v", err)
	}
}
