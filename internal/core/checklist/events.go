package checklist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
// plain file this account owns with no bit for group or others: something
// other than the hook made it, and nothing is appended to it.
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
func RecordCompaction(repoRoot, trigger string, at time.Time) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("checklist: opening the checkout to record the compaction: %w", err)
	}
	defer root.Close()
	if tierState(root) != tierPlain {
		return fmt.Errorf("%w: %s/ is not a plain directory in this checkout", ErrNoLocalTier, TierRelPath)
	}
	if fi, err := root.Lstat(EventsRelPath); err == nil {
		if !fi.Mode().IsRegular() || fileGate(root, repoRoot, EventsRelPath) != "" {
			return ErrEventsNotKept
		}
	}
	line, err := json.Marshal(Event{Schema: EventsSchema, At: at.UTC(), Event: EventCompact, Trigger: compactTrigger(trigger)})
	if err != nil {
		return fmt.Errorf("checklist: encoding the compaction: %w", err)
	}
	f, err := root.OpenFile(EventsRelPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, storePerm)
	if err != nil {
		return fmt.Errorf("checklist: opening %s: %w", EventsRelPath, err)
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
