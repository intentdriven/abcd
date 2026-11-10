package checklist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"
)

// events.go is the checklist's event sidecar (spc-2610230805429429 step 6,
// "Keeping it current"). No hook can make a session bring its checklist up to
// date before a compaction: the host's PreCompact hook can run a command or
// block, and no more. So the hook records that a compaction came, its time
// and its trigger, one JSON line appended per compaction, and never blocks
// it. The checklist itself is never touched here.

// EventsRelPath is the sidecar, beside the checklist in the local tier.
const EventsRelPath = TierRelPath + "/checklist.events.jsonl"

// EventsSchema names each line's shape.
const EventsSchema = "abcd/checklist-events/v1"

// EventCompact is the one event recorded: the host is about to compact the
// session's context.
const EventCompact = "compact"

// The triggers a compaction line records. The host names "manual" (the
// person's /compact) or "auto" (a full context); any other value is recorded
// as TriggerOther, so text from the payload is never copied into the file.
const (
	TriggerManual = "manual"
	TriggerAuto   = "auto"
	TriggerOther  = "other"
)

// Event is one line of the sidecar.
type Event struct {
	Schema  string    `json:"schema"`
	At      time.Time `json:"at"`
	Event   string    `json:"event"`
	Trigger string    `json:"trigger"`
}

// ErrEventsNotKept is RecordCompaction's refusal for a sidecar that is not a
// plain file this account owns, with one name and no bit for group or others:
// something other than the hook made it, and nothing is appended to it.
var ErrEventsNotKept = errors.New("checklist: checklist.events.jsonl is not a plain file this account owns, readable by it alone")

// compactTrigger is the trigger a compaction line records for the host's.
func compactTrigger(t string) string {
	switch t {
	case TriggerManual, TriggerAuto:
		return t
	}
	return TriggerOther
}

// RecordCompaction appends one compaction line, at and the host's trigger, to
// the sidecar under an already-resolved checkout root, creating the file 0600
// in a plain local tier, never the tier itself. It goes through the
// checkout's os.Root, so a tier swapped for a link cannot carry the write out
// of the checkout, and one write of one short line with O_APPEND, so two
// sessions compacting at once each add their own whole line. Every refusal
// comes before any byte is written: an absent or linked tier
// (ErrNoLocalTier) and a sidecar something else made (ErrEventsNotKept).
//
// The opened file is checked, not only its path: a sidecar already there is
// judged by its Lstat, opened without creating, and its handle must be that
// same file and pass the same judgement; an absent one is created exclusively,
// so the open never lands on a file something else put there meanwhile, and
// its handle is judged too. O_NONBLOCK keeps a FIFO from holding the open.
func RecordCompaction(repoRoot, trigger string, at time.Time) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("checklist: opening the checkout to record the compaction: %w", err)
	}
	defer root.Close()
	if tierState(root) != tierPlain {
		return fmt.Errorf("%w: %s/ is not a plain directory in this checkout", ErrNoLocalTier, TierRelPath)
	}
	line, err := json.Marshal(Event{Schema: EventsSchema, At: at.UTC(), Event: EventCompact, Trigger: compactTrigger(trigger)})
	if err != nil {
		return fmt.Errorf("checklist: encoding the compaction: %w", err)
	}
	f, err := openSidecar(root)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return fmt.Errorf("checklist: appending to %s: %w", EventsRelPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("checklist: closing %s: %w", EventsRelPath, err)
	}
	return nil
}

// sidecarAttempts bounds openSidecar's retries when another session creates
// the sidecar between this one's Lstat and its exclusive create.
const sidecarAttempts = 3

// openSidecar opens the sidecar for one append, as RecordCompaction says, and
// answers a handle that has passed the judgement or ErrEventsNotKept.
func openSidecar(root *os.Root) (*os.File, error) {
	for attempt := 1; ; attempt++ {
		var vetted os.FileInfo
		flags := os.O_WRONLY | os.O_APPEND | syscall.O_NONBLOCK
		fi, err := root.Lstat(EventsRelPath)
		switch {
		case err == nil:
			if fileVerdict(fi) != "" {
				return nil, ErrEventsNotKept
			}
			vetted = fi
		case errors.Is(err, fs.ErrNotExist):
			flags |= os.O_CREATE | os.O_EXCL
		default:
			return nil, fmt.Errorf("checklist: reading %s: %w", EventsRelPath, err)
		}
		f, err := root.OpenFile(EventsRelPath, flags, storePerm)
		switch {
		case err == nil:
		case attempt < sidecarAttempts && (errors.Is(err, fs.ErrExist) || errors.Is(err, fs.ErrNotExist)):
			// Made, or removed, by something else between the Lstat and the
			// open: judge what is there now.
			continue
		default:
			return nil, fmt.Errorf("checklist: opening %s: %w", EventsRelPath, err)
		}
		st, err := f.Stat()
		if err != nil || fileVerdict(st) != "" || (vetted != nil && !os.SameFile(vetted, st)) {
			f.Close()
			return nil, ErrEventsNotKept
		}
		return f, nil
	}
}
