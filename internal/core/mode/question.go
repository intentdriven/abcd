package mode

// The question marker: how the badge stays true across the two moments it must
// change (itd-2609212130146198).
//
// When the guard admits abcd's question on the host's question tool, it writes
// this marker and sets the state from the question's chip, naming whom the
// question is for (iss-2610100626211810). The answer resets the state to
// Managed and clears the marker: the question tool's PostToolUse hook runs the
// reset when the answer comes back, and the prompt hook runs it on the next
// human message, for a host that does not run PostToolUse. Without the marker
// neither hook changes anything, so a state the human set by hand to say which
// hat they wear is not clobbered by the message they type next.
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
	"github.com/intentdriven/abcd/internal/localtier"
)

// QuestionOpenRelPath is the marker, repo-relative and slash-separated. Its
// content is the state the question was admitted under, one word; only its
// presence is load-bearing.
const QuestionOpenRelPath = TierRelPath + "/question_open"

// HasTier reports whether repoRoot holds the local-ephemeral tier as a real
// directory — the same presence test SetAt applies before it writes. It says
// the tier is HERE, not that it can be written: a read-only tier passes it.
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

// QuestionOpenForTest reports whether a question is marked open in repoRoot.
// It is a declared test seam: the mode tests read the marker through it, and
// no production path calls it.
func QuestionOpenForTest(repoRoot string) (bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening the checkout to read the question marker: %w", err)
	}
	defer root.Close()
	return markerIn(root)
}

func markerIn(root *os.Root) (bool, error) {
	_, err := localtier.LstatInRoot(root, QuestionOpenRelPath)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	}
	return false, fmt.Errorf("reading the question marker at %s: %w", QuestionOpenRelPath, err)
}

// ResetOnAnswer is the reset the answer triggers: when a question
// is marked open it records Managed, then clears the marker, and reports true.
// With no question open it changes nothing and reports false.
//
// The state is written before the marker is removed, so a failure between the
// two leaves the marker in place and the next message retries the reset; the
// opposite order could clear the marker and leave the badge parked with nothing
// left to reset it.
//
// That ordering is only safe for a marker the reset CAN remove, so it refuses
// to act on one it cannot: a directory at the marker's path is nothing the gate
// wrote (the gate writes one file), and resetting on it would reset a hand-set
// state on every message that follows while the cause stayed put
// (iss-2609260100393814). The refusal changes nothing and names the path. A
// file, a symlink or a FIFO is removed as itself, as before.
//
// The reset applies only to the mode the question was asked under: the
// marker's word is that mode, and the state is reset only while it still
// reads the same. A marker left by a question that never reached an answer
// (one refused and abandoned) whose mode has since been set again, by
// `abcd mode` or by hand, is stale: it is cleared and the mode is left as it
// was set, so a stale marker never resets a state somebody set after it. A
// marker the reset cannot read as one word (a symlink, a FIFO, an oversize or
// unknown word) is nothing the gate wrote, and is cleared the same way.
//
// That comparison needs the store read. A store that cannot be read (an
// unknown word in it, a store git tracks, one that is not a regular file) is
// no evidence of a mode set since, so the reset goes ahead as it always did:
// it writes Managed, and a write that fails is returned, the marker kept, for
// the hook to name.
func ResetOnAnswer(repoRoot string) (bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening the checkout to reset %s: %w", StoreName, err)
	}
	defer root.Close()
	// A marker git tracks came with a commit, not from a question asked here,
	// and resets nothing.
	fi, err := localtier.LstatInRoot(root, QuestionOpenRelPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading the question marker at %s, so the mode was not reset: %w", QuestionOpenRelPath, err)
	case fi.IsDir():
		return false, fmt.Errorf("a directory stands at the question marker's path %s, which the gate never writes and the reset cannot clear, so the mode was not reset; remove it by hand", QuestionOpenRelPath)
	}
	if markerStale(root, repoRoot) {
		if err := root.Remove(QuestionOpenRelPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("a stale question marker at %s, left under a mode since set again, could not be cleared: %w", QuestionOpenRelPath, err)
		}
		return false, nil
	}
	if err := SetAt(repoRoot, Managed); err != nil {
		return false, fmt.Errorf("the mode was not reset: %w", err)
	}
	if err := root.Remove(QuestionOpenRelPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return true, fmt.Errorf("the mode was reset to managed, but the question marker at %s could not be cleared, so the next message resets it again: %w", QuestionOpenRelPath, err)
	}
	return true, nil
}

// markerStale reports whether the marker is stale: the store reads, and the
// marker is not one word naming the state it reads. A store that cannot be
// read leaves nothing to compare against, so the marker is not stale and the
// reset goes ahead.
func markerStale(root *os.Root, repoRoot string) bool {
	current, err := ReadAt(repoRoot)
	if err != nil {
		return false
	}
	marked, ok := markerWord(root)
	return !ok || marked != current
}

// markerWord is the marker's one word, read as a regular file within the
// store's cap; ok is false for anything else (no marker, a symlink, a FIFO,
// an oversize file or a word outside the vocabulary).
func markerWord(root *os.Root) (State, bool) {
	data, err := fsutil.ReadGuardedInRoot(root, QuestionOpenRelPath, maxStoreBytes)
	if err != nil {
		return "", false
	}
	marked, err := ParseState(string(data))
	return marked, err == nil
}

// ClearQuestionOpenOtherThan removes the open-question marker unless it names
// s, and changes no mode. An explicit `abcd mode <state>` calls it with the
// state it set: a state set by hand supersedes a question still marked open
// under another, so the next message's reset never undoes it, while a marker
// naming the state just set is kept, since it then matches at the reset (a
// retry that sets the mode again in the same batch as the question it asks
// still resets on the answer).
//
// It acts only where the tier is a real directory, the test MarkQuestionOpen
// writes behind: a missing tier, or a symlink standing in for it, holds no
// marker the gate wrote and is left as it is. At the marker's path it removes
// only a regular file or a symlink (the symlink as itself, never what it
// points at); anything else there, a directory among them, is nothing the gate
// wrote, and is named and left, as is a marker git tracks.
func ClearQuestionOpenOtherThan(repoRoot string, s State) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout to clear the question marker: %w", err)
	}
	defer root.Close()
	if !tierIn(root) {
		return nil
	}
	// A marker git tracks came with a commit, and is named, never removed.
	fi, err := localtier.LstatInRoot(root, QuestionOpenRelPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("reading the question marker at %s, so it was left: %w", QuestionOpenRelPath, err)
	case fi.Mode().IsRegular():
		if marked, ok := markerWord(root); ok && marked == s {
			return nil
		}
	case fi.Mode()&fs.ModeSymlink == 0:
		return fmt.Errorf("something other than a file stands at the question marker's path %s, which the gate never writes, so it was left; remove it by hand", QuestionOpenRelPath)
	}
	if err := root.Remove(QuestionOpenRelPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clearing the question marker at %s: %w", QuestionOpenRelPath, err)
	}
	return nil
}
