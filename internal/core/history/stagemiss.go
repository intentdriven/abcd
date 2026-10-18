package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// stageMissFilename is the marker, beside the staging directory, that counts
// the transcripts a staging hook (sub-agent stop, session end) read but could
// not stage (iss-2610051434156079). Both hooks exit 0 whatever happens, and
// their stderr reaches only a debug log, so without the marker a lost
// transcript would be seen by no one.
const stageMissFilename = "stage-misses.json"

// stageMissLockTimeout bounds the wait for the marker's own lock. A burst of
// completions that all miss writes the marker once each, and the critical
// section is one small read and write.
const stageMissLockTimeout = 5 * time.Second

// stageMissReasonMax bounds the last reason kept, so the marker stays small
// whatever the failure said.
const stageMissReasonMax = 300

// StageMissNote is the marker: how many transcripts were read but not staged,
// when the first and the last were, and whose the last was (a sub-agent or a
// session) and why.
type StageMissNote struct {
	Schema     int       `json:"schema"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	Count      int       `json:"count"`
	LastWho    string    `json:"last_who"`
	LastReason string    `json:"last_reason"`
	// EarlierLostAt is when an unreadable marker was replaced: misses before
	// it were counted there and are not in Count. Zero when none was.
	EarlierLostAt time.Time `json:"earlier_lost_at,omitzero"`
}

// stageMissPath returns the marker's path beside the staging directory.
func stageMissPath(repoRoot, rootSHA string) (string, error) {
	lane, err := laneDir(repoRoot, rootSHA)
	if err != nil {
		return "", err
	}
	return filepath.Join(lane, stageMissFilename), nil
}

// NoteStageMiss records one transcript that was read and could not be staged,
// under the marker's own lock so concurrent misses each count. who names it
// ("sub-agent <id>" or "session <id>"). who and reason are stored as given,
// apart from a bound on the reason, and every reader sanitises them before
// printing.
func NoteStageMiss(repoRoot, rootSHA, who, reason string) error {
	if !rootSHARe.MatchString(rootSHA) {
		return errors.New(rootSHAErrMsg)
	}
	path, err := stageMissPath(repoRoot, rootSHA)
	if err != nil {
		return err
	}
	if len(reason) > stageMissReasonMax {
		reason = reason[:stageMissReasonMax]
	}
	reason = strings.ReplaceAll(strings.ReplaceAll(reason, "\r", " "), "\n", " ")
	return fsutil.WithFileLock(path+".lock", stageMissLockTimeout, func() error {
		now := time.Now().UTC()
		note := StageMissNote{Schema: 1, FirstSeen: now}
		prior, ok, rerr := readStageMiss(path)
		switch {
		case rerr != nil:
			// A marker that cannot be read is replaced, and the replacement
			// says when, for as long as it stands, so the count restarting is
			// never read as the first miss.
			note.EarlierLostAt = now
		case ok:
			note.FirstSeen, note.Count, note.EarlierLostAt = prior.FirstSeen, prior.Count, prior.EarlierLostAt
		}
		note.LastSeen, note.LastWho, note.LastReason = now, who, reason
		note.Count++
		data, err := json.Marshal(note)
		if err != nil {
			return err
		}
		return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
	})
}

// StageMiss reads the marker. A missing marker is not an error: no transcript
// has been missed.
func StageMiss(repoRoot, rootSHA string) (StageMissNote, bool, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return StageMissNote{}, false, errors.New(rootSHAErrMsg)
	}
	path, err := stageMissPath(repoRoot, rootSHA)
	if err != nil {
		return StageMissNote{}, false, err
	}
	return readStageMiss(path)
}

func readStageMiss(path string) (StageMissNote, bool, error) {
	data, err := fsutil.ReadGuarded(path, maxStageSidecarBytes)
	if errors.Is(err, os.ErrNotExist) {
		return StageMissNote{}, false, nil
	}
	if err != nil {
		return StageMissNote{}, false, err
	}
	var note StageMissNote
	if err := json.Unmarshal(data, &note); err != nil {
		return StageMissNote{}, false, fmt.Errorf("history: the stage-miss marker does not parse: %v", err)
	}
	return note, true, nil
}
