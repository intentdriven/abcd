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

// tierEntries lists the tier's names, for proving a probe leaves no residue.
func tierEntries(t *testing.T, tier string) []string {
	t.Helper()
	ents, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names
}

// TestCanSetProbesWhatTheVerbNeeds (iss-2609260100382261): the gate names
// `abcd mode` as the remedy for a refused question, so it must know the verb
// could actually run here. CanSet answers exactly that — the tier is a real,
// writable directory and nothing stands at the store's path that the verb's
// rename could not replace — and it leaves nothing behind in the tier either
// way.
func TestCanSetProbesWhatTheVerbNeeds(t *testing.T) {
	root := newRepo(t)
	if err := mode.CanSet(root); !errors.Is(err, mode.ErrNoLocalTier) {
		t.Fatalf("CanSet without the tier = %v, want ErrNoLocalTier", err)
	}

	tier := makeTier(t, root)
	if err := mode.CanSet(root); err != nil {
		t.Fatalf("CanSet on a writable tier = %v, want nil", err)
	}
	if got := tierEntries(t, tier); len(got) != 0 {
		t.Fatalf("the probe left residue in the tier: %v", got)
	}

	if os.Geteuid() == 0 {
		t.Log("running as root: a read-only directory does not refuse root, so the unwritable leg is skipped")
	} else {
		if err := os.Chmod(tier, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(tier, 0o700) })
		if err := mode.CanSet(root); err == nil {
			t.Fatal("CanSet on a read-only tier = nil; the verb cannot write there")
		}
		if err := mode.SetAt(root, mode.Facilitator); err == nil {
			t.Fatal("precondition: SetAt succeeded on a read-only tier, so the probe's premise is wrong")
		}
		if err := os.Chmod(tier, 0o700); err != nil {
			t.Fatal(err)
		}
		if got := tierEntries(t, tier); len(got) != 0 {
			t.Fatalf("a failed probe left residue in the tier: %v", got)
		}
	}

	if err := os.Mkdir(storePath(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.CanSet(root); err == nil {
		t.Fatal("CanSet with a directory at the store's path = nil; the verb's rename cannot replace it")
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
