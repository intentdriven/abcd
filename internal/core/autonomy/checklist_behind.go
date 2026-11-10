package autonomy

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// checklist_behind.go is the checklist's behind check (spc-2610230805429429
// step 4, "Keeping it current"). No hook can make a session bring its
// checklist up to date before an automatic compaction, so currency is held as
// the work happens: the checklist is behind when the work the session owns,
// read as the stop check reads owed work (its reservations, its pull requests,
// its branches), holds a reference no item's owner_refs names, or moved after
// the checklist's updated_at. While the person is away the stop is blocked
// once per consecutive set, inside the host's continuation cap; while present
// the stop is never blocked, and the next prompt carries the note instead,
// since a stop's message reaches the person and not the model. A step the
// session never recorded and that left no reference cannot be seen.

// maxBehindNamed is how many references the note names before it counts the
// rest.
const maxBehindNamed = 10

// notePrefix opens every note Note words; a note read back without it was not
// abcd's and is dropped.
const notePrefix = "abcd: your checklist is behind the work this session owns: "

// maxNoteBytes bounds a note read back from the session's state: the fixed
// words, under 512 bytes, and the two lists' maxBehindNamed names each, a name
// at most clean's 2*echoCap bytes with its separator. A note Note words is
// never longer, so a real one loses nothing.
const maxNoteBytes = 512 + 2*maxBehindNamed*(2*echoCap+2)

// Behind is what a checklist does not keep up with.
type Behind struct {
	// Unlisted are the owned references no item names.
	Unlisted []string
	// Moved are the listed references whose work moved after the checklist
	// was last written.
	Moved []string
}

// Empty reports whether the checklist keeps up with the owned work.
func (b Behind) Empty() bool { return len(b.Unlisted) == 0 && len(b.Moved) == 0 }

// Fingerprint identifies the set by its references, so a branch that moves
// again is not a new set until the checklist has caught up once.
func (b Behind) Fingerprint() string {
	if b.Empty() {
		return ""
	}
	return hashText("unlisted\x00" + strings.Join(b.Unlisted, "\x00") + "\x00moved\x00" + strings.Join(b.Moved, "\x00"))
}

// CheckBehind compares the owned work with the checklist a Read returned. A
// read that found no checklist lists nothing, so every owned reference is
// unlisted.
func CheckBehind(owned []OwnedRef, res checklist.Result) Behind {
	listed := map[string]bool{}
	var updated time.Time
	if res.Source != checklist.FromNothing {
		for _, it := range res.Checklist.Items {
			for _, ref := range it.OwnerRefs {
				listed[ref] = true
			}
		}
		// Validate holds updated_at to RFC 3339; one that does not parse
		// reads as never written, so every moved reference counts.
		updated, _ = time.Parse(time.RFC3339Nano, res.Checklist.UpdatedAt)
	}
	var b Behind
	for _, o := range owned {
		named := listed[o.Ref]
		for _, a := range o.Aliases {
			named = named || listed[a]
		}
		switch {
		case !named:
			b.Unlisted = append(b.Unlisted, o.Ref)
		case !o.Changed.IsZero() && o.Changed.After(updated):
			b.Moved = append(b.Moved, o.Ref)
		}
	}
	sort.Strings(b.Unlisted)
	sort.Strings(b.Moved)
	return b
}

// Note is the behind set in plain words, for the agent: what the checklist
// does not list, what moved since it was written, and how to catch up.
func (b Behind) Note() string {
	if b.Empty() {
		return ""
	}
	var parts []string
	if len(b.Unlisted) > 0 {
		parts = append(parts, "it does not list "+namedRefs(b.Unlisted))
	}
	if len(b.Moved) > 0 {
		parts = append(parts, namedRefs(b.Moved)+" moved after it was last written")
	}
	return notePrefix + strings.Join(parts, "; ") +
		". Bring it up to date now, with `abcd checklist add` or `abcd checklist set`, naming each in an item's owner_refs, so it is current before a compaction can come."
}

// namedRefs names at most maxBehindNamed references, each cleaned to one
// line, and counts the rest.
func namedRefs(refs []string) string {
	shown := refs
	if len(shown) > maxBehindNamed {
		shown = shown[:maxBehindNamed]
	}
	names := make([]string, len(shown))
	for i, r := range shown {
		names[i] = clean(r)
	}
	s := strings.Join(names, ", ")
	if rest := len(refs) - len(shown); rest > 0 {
		s += fmt.Sprintf(" and %d more", rest)
	}
	return s
}

// JudgeChecklist folds the behind check into the stop verdict v that Judge
// returned over st, the session's state before it.
//
// While away (v.Save), a set other than the one last blocked on blocks the
// stop once, naming
// it: added to a block's reason, or, on a stop Judge allowed, as a block of
// its own, counted toward the host's continuation cap as every block is and
// never past it. Only the last set is kept, so the block is once per
// consecutive set: a set that comes back after another blocks again, within
// the cap. The set last blocked on, or a block the cap would override,
// leaves v as it was. While present the stop is never blocked: a set not yet
// noted is kept in the session's state for the next prompt (TakeChecklistNote),
// and save says the state must be written.
func JudgeChecklist(v Verdict, b Behind, in StopInput, st SessionState, away bool) (out Verdict, present SessionState, save bool) {
	fp := b.Fingerprint()
	if !away {
		next := st
		switch {
		case fp == "":
			next.ChecklistNoted, next.ChecklistNote = "", ""
		case fp != st.ChecklistNoted:
			next.ChecklistNoted, next.ChecklistNote = fp, b.Note()
		}
		return v, next, next != st
	}
	if !v.Save {
		// The reading was cut short: the stop goes as Judge allowed it.
		return v, st, false
	}
	if fp == "" {
		v.State.ChecklistBlocked = ""
		return v, st, false
	}
	if fp == v.State.ChecklistBlocked || v.Stop != nil {
		return v, st, false
	}
	if v.Block {
		v.Reason += "\n" + b.Note()
		v.State.ChecklistBlocked = fp
		return v, st, false
	}
	// Counted from the state before this check, as Judge counts: an allowed
	// stop has already reset the verdict's own count.
	blocks := 1
	if in.StopHookActive {
		blocks = st.Blocks + 1
	}
	limit := in.Cap
	if limit <= 0 {
		limit = DefaultBlockCap
	}
	if blocks > limit {
		return v, st, false
	}
	v.Block, v.Reason = true, b.Note()
	v.State.State, v.State.NextLook, v.State.Blocks = StateWorking, time.Time{}, blocks
	v.State.ChecklistBlocked = fp
	return v, st, false
}

// ErrChecklistNoteForeign is TakeChecklistNote's word for a note in the
// session's state that abcd did not word: it is cleared and nothing is shown.
var ErrChecklistNoteForeign = errors.New("the behind note in the session's state is not one abcd words, so it was dropped")

// TakeChecklistNote is the behind note the stop hook left for this session,
// cleared as it is taken, so one note reaches one prompt; "" when none waits.
// The noted set is kept, so the same set is not noted again until the
// checklist has caught up.
//
// The note is read back from a file any process of this account can write, so
// it is cleaned as it is read, not only as it was written: one line, no
// control or bidi character, within maxNoteBytes, and opening with the words
// Note opens with, so it can never start a heading of its own in the prompt.
func TakeChecklistNote(copyRoot, sessionID string) (string, error) {
	st, err := LoadSession(copyRoot, sessionID)
	if err != nil || st.ChecklistNote == "" {
		return "", err
	}
	note := termsafe.CleanProseLine(st.ChecklistNote, maxNoteBytes)
	st.ChecklistNote = ""
	if err := SaveSession(copyRoot, sessionID, st); err != nil {
		return "", err
	}
	if !strings.HasPrefix(note, notePrefix) {
		return "", ErrChecklistNoteForeign
	}
	return note, nil
}
