package mode

// The question marker: how the badge stays true across the two moments it must
// change (itd-2609212130146198).
//
// The agent sets the state with the verb when it stops to ask, naming whom it
// addresses; the guard admits a question on the host's question tool only once
// that state names somebody, and when it admits one it writes this marker. The
// next human message is the answer, and the prompt hook that sees it resets the
// state to Managed and clears the marker. Without the marker the prompt hook
// changes nothing, so a state the human set by hand to say which hat they wear
// is not clobbered by the message they type next.
//
// The marker lives beside the store, in the same local-ephemeral tier, and
// follows the store's rules: it is written only where the tier already is, the
// tier is never created on the way to a write, and every read and write goes
// through an os.Root so a symlinked ancestor cannot walk it out of the checkout.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// QuestionOpenRelPath is the marker, repo-relative and slash-separated. Its
// content is the state the question was admitted under, one word; only its
// presence is load-bearing.
const QuestionOpenRelPath = TierRelPath + "/question_open"

// HasTier reports whether repoRoot holds the local-ephemeral tier as a real
// directory — the same test SetAt applies before it writes, so a caller that
// gates on it gates on exactly "the verb can set the state here".
func HasTier(repoRoot string) bool {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false
	}
	defer root.Close()
	return tierIn(root)
}

// tierIn is HasTier against an open root. Lstat, not Stat: a symlink standing in
// for the tier is not the tier.
func tierIn(root *os.Root) bool {
	fi, err := root.Lstat(TierRelPath)
	return err == nil && fi.IsDir()
}

// MarkQuestionOpen records that a question was admitted while the loop was
// parked on s. It refuses, writing nothing, where the tier is absent.
func MarkQuestionOpen(repoRoot string, s State) error {
	if !s.Valid() {
		return unknown(s)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout to mark a question open: %w", err)
	}
	defer root.Close()
	if !tierIn(root) {
		return fmt.Errorf("%w: %s/ is not a directory in this checkout, so there is nowhere to mark a question open",
			ErrNoLocalTier, TierRelPath)
	}
	if err := fsutil.WriteFileAtomicInRoot(root, QuestionOpenRelPath, []byte(string(s)+"\n"), storePerm); err != nil {
		return fmt.Errorf("writing the question marker at %s: %w", QuestionOpenRelPath, err)
	}
	return nil
}

// QuestionOpen reports whether a question is marked open in repoRoot.
func QuestionOpen(repoRoot string) (bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening the checkout to read the question marker: %w", err)
	}
	defer root.Close()
	return markerIn(root)
}

func markerIn(root *os.Root) (bool, error) {
	_, err := root.Lstat(QuestionOpenRelPath)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	}
	return false, fmt.Errorf("reading the question marker at %s: %w", QuestionOpenRelPath, err)
}

// ResetOnAnswer is the reset the next human message triggers: when a question
// is marked open it records Managed, then clears the marker, and reports true.
// With no question open it changes nothing and reports false.
//
// The state is written before the marker is removed, so a failure between the
// two leaves the marker in place and the next message retries the reset; the
// opposite order could clear the marker and leave the badge parked with nothing
// left to reset it.
func ResetOnAnswer(repoRoot string) (bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening the checkout to reset %s: %w", StoreName, err)
	}
	defer root.Close()
	open, err := markerIn(root)
	if err != nil || !open {
		return false, err
	}
	if err := SetAt(repoRoot, Managed); err != nil {
		return false, err
	}
	if err := root.Remove(QuestionOpenRelPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("clearing the question marker at %s: %w", QuestionOpenRelPath, err)
	}
	return true, nil
}
