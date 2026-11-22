package autonomy

// heartbeat.go is the memory of `abcd implement heartbeat`
// (iss-2611220942554289): an autonomous run's half-hourly check, which a run
// used to compose by hand from a handful of shell commands each time. The
// verb renews the run's reservations, reads each tracked pull request and the
// free disk, and reports what changed since the last heartbeat. This file
// keeps the last heartbeat's facts in the checkout's local tier and tells
// what changed; the reading itself is the front door's, which holds the
// reservation, forge and load-gate packages.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// HeartbeatRel is where the last heartbeat's facts are kept, in the
// checkout's local tier.
const HeartbeatRel = ".abcd/.work.local/heartbeat.json"

// heartbeatSchema is the state file's schema_version.
const heartbeatSchema = 1

// maxHeartbeatBytes bounds the state file a heartbeat reads back.
const maxHeartbeatBytes = 64 << 10

// The kinds of fact a heartbeat reports.
const (
	BeatReservation = "reservation"
	BeatPullRequest = "pull_request"
	BeatDisk        = "disk"
	// BeatForge is a read the heartbeat could not make at all (the
	// reservations' fetch, the listing of the account's pull requests).
	BeatForge = "forge"
)

// BeatItem is one fact a heartbeat read: its kind, its key within the kind
// (a record id, "#N", the disk), the state it is compared on, and the line a
// person reads. Terminal marks a pull request merged or closed, which the
// next heartbeat no longer reads unless asked.
type BeatItem struct {
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	State    string `json:"state"`
	Line     string `json:"line"`
	Terminal bool   `json:"terminal,omitempty"`
}

// BeatChange is a fact that changed since the last heartbeat: new, in
// another state (Was names the state it was in), or gone (a reservation no
// longer held, or a read that now succeeds).
type BeatChange struct {
	BeatItem
	Was  string `json:"was,omitempty"`
	Gone bool   `json:"gone,omitempty"`
}

// beatFile is the state file's content.
type beatFile struct {
	SchemaVersion int        `json:"schema_version"`
	At            time.Time  `json:"at"`
	Items         []BeatItem `json:"items"`
}

// ErrHeartbeatNoTier is a checkout without its local tier, where the heartbeat
// keeps its facts.
var ErrHeartbeatNoTier = errors.New("this checkout has no .abcd/.work.local/, so the heartbeat keeps nothing between runs; run abcd ahoy to set it up")

// ReadHeartbeat is the last heartbeat's facts and when it ran, kept in the
// checkout at root; none and the zero time when no heartbeat has run here. A
// file that cannot be read or does not parse is an error, and the caller
// judges every fact new.
func ReadHeartbeat(root string) ([]BeatItem, time.Time, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer r.Close()
	data, err := localtier.ReadInRoot(r, HeartbeatRel, maxHeartbeatBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	var f beatFile
	if json.Unmarshal(data, &f) != nil || f.SchemaVersion != heartbeatSchema {
		return nil, time.Time{}, fmt.Errorf("%s does not parse as a heartbeat's facts", HeartbeatRel)
	}
	return f.Items, f.At, nil
}

// WriteHeartbeat keeps items as the heartbeat's facts at at, atomically, in
// the checkout at root's local tier, which it never creates.
func WriteHeartbeat(root string, at time.Time, items []BeatItem) error {
	fi, err := os.Lstat(filepath.Join(root, ".abcd", ".work.local"))
	if err != nil || !fi.IsDir() {
		return ErrHeartbeatNoTier
	}
	data, err := json.MarshalIndent(beatFile{SchemaVersion: heartbeatSchema, At: at.UTC(), Items: items}, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxHeartbeatBytes {
		return fmt.Errorf("the heartbeat's facts would pass %d bytes", maxHeartbeatBytes)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	return fsutil.WriteFileAtomicInRoot(r, HeartbeatRel, append(data, '\n'), 0o600)
}

// HeartbeatChanges is every fact of now that is new or in another state than
// in prev, then every fact of prev gone from now: a reservation, a disk
// reading or a failed read no longer there, and a pull request not yet
// terminal that is no longer read. A terminal pull request left out of now is
// no change: the heartbeat stops reading it once it has reported it. Facts
// come in now's order, then prev's.
func HeartbeatChanges(prev, now []BeatItem) []BeatChange {
	was := map[string]BeatItem{}
	for _, it := range prev {
		was[it.Kind+"\x00"+it.Key] = it
	}
	seen := map[string]bool{}
	var out []BeatChange
	for _, it := range now {
		k := it.Kind + "\x00" + it.Key
		seen[k] = true
		p, ok := was[k]
		switch {
		case !ok:
			out = append(out, BeatChange{BeatItem: it})
		case p.State != it.State:
			out = append(out, BeatChange{BeatItem: it, Was: p.State})
		}
	}
	for _, it := range prev {
		if seen[it.Kind+"\x00"+it.Key] || it.Kind == BeatPullRequest && it.Terminal {
			continue
		}
		g := BeatChange{BeatItem: it, Gone: true}
		g.Line = goneLine(it)
		out = append(out, g)
	}
	return out
}

// goneLine is the line a fact gone since the last heartbeat prints.
func goneLine(it BeatItem) string {
	switch it.Kind {
	case BeatReservation:
		return it.Key + ": no longer held by you from this checkout"
	case BeatPullRequest:
		return it.Key + ": no longer tracked"
	case BeatForge:
		return it.Key + ": read again"
	}
	return it.Key + ": no longer reported"
}

// TrackedPRs is each pull request number prev read that is not yet terminal,
// ascending: the next heartbeat reads them again, so a pull request that
// merged or closed is reported once even after it leaves the account's open
// listing.
func TrackedPRs(prev []BeatItem) []int {
	var out []int
	for _, it := range prev {
		if it.Kind != BeatPullRequest || it.Terminal {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(it.Key, "#")); err == nil && n > 0 && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}
