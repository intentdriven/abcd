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
	"crypto/rand"
	"encoding/hex"
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
// directory — the same presence test SetAt applies before it writes. It says
// the tier is HERE, not that it can be written: a read-only tier passes it.
// A caller that needs "the verb can set the state here" asks CanSet.
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
//
// That ordering is only safe for a marker the reset CAN remove, so it refuses
// to act on one it cannot: a directory at the marker's path is nothing the gate
// wrote (the gate writes one file), and resetting on it would reset a hand-set
// state on every message that follows while the cause stayed put
// (iss-2609260100393814). The refusal changes nothing and names the path. A
// file, a symlink or a FIFO is removed as itself, as before.
func ResetOnAnswer(repoRoot string) (bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return false, fmt.Errorf("opening the checkout to reset %s: %w", StoreName, err)
	}
	defer root.Close()
	fi, err := root.Lstat(QuestionOpenRelPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("reading the question marker at %s, so the mode was not reset: %w", QuestionOpenRelPath, err)
	case fi.IsDir():
		return false, fmt.Errorf("a directory stands at the question marker's path %s, which the gate never writes and the reset cannot clear, so the mode was not reset; remove it by hand", QuestionOpenRelPath)
	}
	if err := SetAt(repoRoot, Managed); err != nil {
		return false, fmt.Errorf("the mode was not reset: %w", err)
	}
	if err := root.Remove(QuestionOpenRelPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return true, fmt.Errorf("the mode was reset to managed, but the question marker at %s could not be cleared, so the next message resets it again: %w", QuestionOpenRelPath, err)
	}
	return true, nil
}

// probePrefix names the probe file CanSet creates and removes. It sits beside
// the atomic writer's own temp names in the tier and is never left behind.
const probePrefix = ".mode-probe-"

// CanSet reports whether SetAt could record a state in repoRoot right now, and
// why not when it could not. It is the question gate's check before it names
// `abcd mode` as the remedy for a refused question: a refusal whose remedy
// cannot run refuses forever (iss-2609260100382261).
//
// It probes what the write needs rather than what the permission bits say:
// the tier is a real directory (ErrNoLocalTier otherwise, as SetAt refuses),
// nothing stands at the store's path that the writer's rename cannot replace,
// and a file can be created in the tier — the atomic writer's first step. The
// probe file is removed before CanSet returns, so a successful probe leaves the
// tier as it found it, and a failed create leaves nothing to remove. Creating
// is the test because it is what fails on a read-only mount, an unwritable
// directory and a foreign owner alike, where a mode-bit check answers only the
// second.
func CanSet(repoRoot string) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout to probe %s: %w", StoreName, err)
	}
	defer root.Close()
	if !tierIn(root) {
		return fmt.Errorf("%w: %s/ is not a directory in this checkout, so there is nowhere for %s to live",
			ErrNoLocalTier, TierRelPath, StoreName)
	}
	if fi, err := root.Lstat(FileRelPath); err == nil && fi.IsDir() {
		return fmt.Errorf("%s at %s is a directory, which the writer cannot replace", StoreName, FileRelPath)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspecting %s at %s: %w", StoreName, FileRelPath, err)
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Errorf("naming the probe for %s: %w", StoreName, err)
	}
	probe := TierRelPath + "/" + probePrefix + hex.EncodeToString(buf[:])
	f, err := root.OpenFile(probe, os.O_WRONLY|os.O_CREATE|os.O_EXCL, storePerm)
	if err != nil {
		return fmt.Errorf("%s/ is not writable, so %s cannot be recorded here: %w", TierRelPath, StoreName, err)
	}
	closeErr := f.Close()
	if err := root.Remove(probe); err != nil {
		return fmt.Errorf("removing the probe %s: %w", probe, err)
	}
	if closeErr != nil {
		return fmt.Errorf("closing the probe %s: %w", probe, closeErr)
	}
	return nil
}
