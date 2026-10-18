package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// end.go — ending the away mode whatever state its record is in (the diff
// review's major 3, spc-2610180914469489 "Diff review round 1"): a record
// past its bound, partly or wholly unparsable, not safely this account's, or
// behind a lock held past its timeout never keeps the person from ending
// the mode. Setting the mode again carries a record past its bound too.

// lenientCap bounds the lenient read: far past MaxFileBytes, so a record a
// harness or an older build left over its bound is still read and closed.
const lenientCap = 4 << 20

// lenientRead is the record as far as it could be read, and notes saying
// what could not.
type lenientRead struct {
	rec *Record
	// unreadable is a file that stands there but yielded no record to trust:
	// it does not parse, names another schema, is past lenientCap, or is not
	// safely this account's. Nothing from it is printed.
	unreadable bool
	// partial is a record that parsed with parts of it of the wrong shape,
	// which were left out.
	partial bool
	notes   []string
}

// readLenient reads the record as Back needs it: whatever can be trusted,
// with a note for everything else. Only an absent record reads as none.
func (s Store) readLenient() lenientRead {
	name := s.Display(RecordFile)
	raw, refusal, err := fsutil.ReadHomeDeclaration(s.home, s.Rel()+"/"+RecordFile, lenientCap)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, fs.ErrNotExist):
		return lenientRead{}
	case refusal == fsutil.DeclarationOK:
	case refusal == fsutil.DeclarationAbsent || refusal == fsutil.DeclarationUnreadable:
		return lenientRead{unreadable: true, notes: []string{fmt.Sprintf("%s could not be read (%s), so nothing from it is printed", name, refusalWords(refusal, err))}}
	default:
		return lenientRead{unreadable: true, notes: []string{fmt.Sprintf("%s is not safely this account's (it %s), so nothing from it is printed", name, refusalWords(refusal, err))}}
	}
	var r Record
	err = json.Unmarshal(raw, &r)
	var typeErr *json.UnmarshalTypeError
	out := lenientRead{rec: &r}
	switch {
	case errors.As(err, &typeErr):
		out.partial = true
		out.notes = append(out.notes, fmt.Sprintf("parts of %s could not be read (the field %q is not of its shape) and are left out", name, typeErr.Field))
	case err != nil:
		return lenientRead{unreadable: true, notes: []string{fmt.Sprintf("%s could not be read (it does not parse), so nothing from it is printed", name)}}
	}
	if r.SchemaVersion != SchemaVersion {
		return lenientRead{unreadable: true, notes: []string{fmt.Sprintf("%s could not be read (schema_version %d, this abcd reads %d), so nothing from it is printed", name, r.SchemaVersion, SchemaVersion)}}
	}
	if len(raw) > MaxFileBytes {
		out.notes = append(out.notes, fmt.Sprintf("%s held %d bytes, past its %d KiB; every line is printed, and the oldest are let go from the record kept", name, len(raw), MaxFileBytes>>10))
	}
	return out
}

// closedStub is the least closed record that ends the mode: prev's who,
// from where and window when prev is known, and the closing.
func closedStub(prev *Record, who Who, now time.Time) *Record {
	r := &Record{SchemaVersion: SchemaVersion, Start: now.Truncate(time.Second), Until: now.Truncate(time.Second)}
	if prev != nil {
		r.SetBy, r.SetFrom, r.Start, r.Until, r.OpenEnded, r.HoldPushes = prev.SetBy, prev.SetFrom, prev.Start, prev.Until, prev.OpenEnded, prev.HoldPushes
	}
	r.Close(who, now)
	return r
}

func (s Store) writeRecord(r *Record) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return s.WriteFile(RecordFile, append(raw, '\n'))
}

// endRecord is Back's work on the record, under the lock or, when the lock
// cannot be had, without it: it prints what can be read, closes, fits and
// writes; a record it cannot write whole, or cannot read at all, is replaced
// by a closed stub.
func (s Store) endRecord(res *BackResult, who Who, now time.Time) (*Record, error) {
	lr := s.readLenient()
	res.Notes = append(res.Notes, lr.notes...)
	if lr.unreadable {
		res.State = "unreadable"
		stub := closedStub(nil, who, now)
		if err := s.writeRecord(stub); err != nil {
			return nil, err
		}
		res.Notes = append(res.Notes, "it was replaced with a closed record")
		return stub, nil
	}
	cur := lr.rec
	if cur == nil || cur.Closed != nil {
		res.State = "none"
		return cur, nil
	}
	res.State = "lapsed"
	if cur.Holds(now) {
		res.State = "away"
	}
	res.Lines = cur.Lines()
	cur.Close(who, now)
	cur.Fit()
	if err := s.writeRecord(cur); err != nil {
		stub := closedStub(cur, who, now)
		if err2 := s.writeRecord(stub); err2 != nil {
			return nil, err
		}
		res.Notes = append(res.Notes, fmt.Sprintf("the closed record could not be written whole (%s); a closed record without its lines was written instead", err))
		return stub, nil
	}
	return cur, nil
}

// endUnlocked is endRecord for a lock that could not be had: the write is
// atomic, and a writer that held the lock and wrote after it is caught by
// re-reading and closing again, a few times.
func (s Store) endUnlocked(res *BackResult, who Who, now time.Time) (*Record, error) {
	var rec *Record
	var err error
	for try := 0; try < 3; try++ {
		*res = BackResult{State: res.State, Lines: res.Lines, Notes: res.Notes}
		if try > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		first := try == 0
		var r BackResult
		rec, err = s.endRecord(&r, who, now)
		if err != nil {
			return nil, err
		}
		if first {
			*res = BackResult{State: r.State, Lines: r.Lines, Notes: append(res.Notes, r.Notes...)}
		}
		time.Sleep(50 * time.Millisecond)
		if again := s.readLenient(); again.unreadable || again.rec == nil || again.rec.Closed != nil {
			return rec, nil
		}
	}
	return rec, nil
}

// carryRead is SetAway's read under the lock: a record past its bound is
// carried (Begin fits it); one that is unreadable, partial or not safely
// this account's is refused, naming the file, for `abcd mode back` to end.
func (s Store) carryRead() (*Record, error) {
	lr := s.readLenient()
	if lr.unreadable || lr.partial {
		return nil, fmt.Errorf("%w: %s; `abcd mode back` ends it", ErrUnsafe, lr.notes[0])
	}
	if lr.rec != nil {
		if err := s.vetLock(); err != nil {
			return nil, err
		}
	}
	return lr.rec, nil
}
