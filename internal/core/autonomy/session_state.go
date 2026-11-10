package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// SessionsDir is the session-state folder below a working copy's root: one
// file per session id, in the copy's local tier.
const SessionsDir = mode.TierRelPath + "/autonomy/sessions"

// SessionState is one session's stop-check state, kept between hook runs in
// SessionsDir/<session-id>.json.
type SessionState struct {
	SchemaVersion int `json:"schema_version"`
	// State is working, waiting or stopped, as the last verdict left it.
	State string `json:"state,omitempty"`
	// NextLook is the qualifying wake's next fire time while waiting.
	NextLook time.Time `json:"next_look,omitempty"`
	// Fingerprint is the owed set's at the last check (Owed.Fingerprint).
	Fingerprint string `json:"fingerprint,omitempty"`
	// QuestionHash is the last prose question blocked, hashed.
	QuestionHash string `json:"question_hash,omitempty"`
	// Blocks is the count of consecutive blocks, which stop_hook_active alone
	// drives (dcn-2610181003484631).
	Blocks int `json:"blocks"`
	// Wakes is how many times in a row a wake released an unchanged set of
	// capped items, and WakeFingerprint those items' fingerprint
	// (dcn-2610181035361997). A wake that releases a set waiting on live work
	// does not add to it.
	Wakes           int    `json:"wakes"`
	WakeFingerprint string `json:"wake_fingerprint,omitempty"`
	// WaitingSince is the first check that found live work standing for
	// every capped item of the set WakeFingerprint names (Owed.CappedAllLive),
	// zero before one; reset with WakeFingerprint. The wait it starts is
	// bounded by LiveWorkBound, measured from it rather than from the run's
	// age, so a run restarted at each wake cannot renew it.
	WaitingSince time.Time `json:"waiting_since,omitempty"`
	// ChecklistBlocked is the behind set (Behind.Fingerprint) a stop was
	// last blocked on while away, so a set blocks once per consecutive run of
	// stops on it; "" once the
	// checklist caught up.
	ChecklistBlocked string `json:"checklist_blocked,omitempty"`
	// ChecklistNoted is the behind set last left for the prompt hook while the
	// person was present, and ChecklistNote that note, until the next prompt
	// takes it into the session's context.
	ChecklistNoted string `json:"checklist_noted,omitempty"`
	ChecklistNote  string `json:"checklist_note,omitempty"`
}

// sessionStatePath is the session's state file below the working copy at
// copyRoot; an id outside [A-Za-z0-9_-]{1,128} is ErrSessionID.
func sessionStatePath(copyRoot, sessionID string) (string, error) {
	if !ValidSessionID(sessionID) {
		return "", ErrSessionID
	}
	return filepath.Join(copyRoot, filepath.FromSlash(SessionsDir), sessionID+".json"), nil
}

// LoadSession reads the session's state below copyRoot, through an os.Root at
// copyRoot so nothing outside the copy is read; an absent file is the zero
// state. A file over MaxFileBytes, or one that does not parse, is an error.
// Its one git call, the tracked check, is bound to ctx: a hook reads it under
// its own deadline, and a ctx that ends first refuses, as git not answering
// does.
func LoadSession(ctx context.Context, copyRoot, sessionID string) (SessionState, error) {
	if !ValidSessionID(sessionID) {
		return SessionState{}, ErrSessionID
	}
	root, err := os.OpenRoot(copyRoot)
	if err != nil {
		return SessionState{}, err
	}
	defer root.Close()
	// A state a commit carries was written by whoever made the commit, not by
	// this copy's session, and is refused rather than read as its wake state.
	if err := localtier.UntrackedContext(ctx, copyRoot, path.Join(SessionsDir, sessionID+".json")); err != nil {
		return SessionState{}, err
	}
	f, err := root.Open(path.Join(SessionsDir, sessionID+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return SessionState{SchemaVersion: SchemaVersion}, nil
	}
	if err != nil {
		return SessionState{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return SessionState{}, err
	}
	if len(raw) > MaxFileBytes {
		return SessionState{}, fmt.Errorf("the session state %s is over %d KiB", sessionID, MaxFileBytes>>10)
	}
	var st SessionState
	if err := json.Unmarshal(raw, &st); err != nil || st.SchemaVersion != SchemaVersion {
		return SessionState{}, fmt.Errorf("the session state %s does not parse", sessionID)
	}
	return st, nil
}

// SaveSession writes the session's state below copyRoot through fsutil's
// atomic writer, making the folders owner-only on the way.
func SaveSession(copyRoot, sessionID string, st SessionState) error {
	if !ValidSessionID(sessionID) {
		return ErrSessionID
	}
	root, err := os.OpenRoot(copyRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(SessionsDir, 0o700); err != nil {
		return err
	}
	st.SchemaVersion = SchemaVersion
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(root, path.Join(SessionsDir, sessionID+".json"), append(raw, '\n'), 0o600)
}
