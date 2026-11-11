package loop

// reservation_save.go is the run state's one save a lane's own abcd can make
// (iss-2611020719494343). A run's state lives in the checkout that drives it,
// and the abcd driving it is whatever that checkout runs. A lane's process runs
// the lane's own abcd, so a lane whose change raises SchemaVersion runs a newer
// one, and three writers reach the driver's file from there: the gated waiter
// its go test and make preflight run under (RenewRuns, across worktrees),
// implement status (RenewRunsIn), and implement widen, which the lane's brief
// runs from the driver's checkout. Each used to save the whole state at its own
// binary's version, after which the driver, reading up to the version it was
// built with, refused every step and receipt until someone set the number back
// by hand.
//
// None of those writers has anything to add but a line in the run's record, so
// each saves only that: the file is read, and rewritten with its record grown
// and its updated_at moved. Every other byte stays what the writer it came from
// wrote, its schema_version first, so a lane's work never raises the run that
// drives it, whatever the lane's abcd knows.
//
// The refusal the other way round is unchanged, on purpose. A file above this
// abcd's own version is still refused as a newer abcd's before anything decodes
// it, whichever writer reads it: nothing is lowered and no field this abcd
// cannot name is dropped. A driver stepping a run takes the file up to its own
// version as it always has (mutate), since a driver is the file's owner and may
// write the fields that version adds.
//
// Every renewal saves this way, a parked lane's included (drain_finish.go,
// drain_lease.go). A settle under a stop cannot: it saves the lanes, so a process
// that is not the run's driver leaves the stop's halt and a stop's unsettled mark
// on a run whose file states another version to that driver (settlesHere): the
// renewal does, and so does a drain's move over a run not among its lanes, which
// neither halts it (drainStop) nor takes up its mark (parkedStopNote).
//
// Two limits stand, and nothing enforces either yet. First, a step under a stop
// (ClaimDriver, then the move's own check) and a drain's move over its own lanes
// must not raise a run another version's driver writes, and both settle it at
// this binary's version. Second, a drain's move over a parked run with no stop's
// mark must not raise one another version's driver writes, among its lanes or
// not, and four saves in it take the whole state to this binary's version with
// no stop in force: the landing's last step on a merged lane (finishMerged),
// the sync entered again over a conflict (resyncParked), the merge-queue watch
// (watchParked) and the hand-back of a closed pull request (handBackClosed).
// A step from a newer abcd is how a driver resumes after an upgrade, a newer
// drain in the checkout reads the same drain state as the drain it resumes, and
// a new drain, which starts with no lanes, finishing the merged lanes an earlier
// drain left is the ordinary path after one; under a stop ClaimDriver records no
// driver, and the driver it recorded before names a process a driver's own new
// session does not match, so nothing the run holds tells its driver from a
// second session or drain. Closing either needs the run to name its driver in a
// form a new session of it can prove.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The state file's two fields a reservation save rewrites.
const (
	stateFieldRecord  = "record"
	stateFieldUpdated = "updated_at"
)

// mutateReservation is mutate for a writer that may be running a lane's abcd:
// the same frame and the same strict read, and fn may change the run's record,
// by appending to it, and its updated_at, and nothing else. A fn that changed
// anything else is refused, not saved in part: the change would otherwise be
// dropped in silence.
func mutateReservation(repoRoot, runID string, fn func(st *State) (bool, error)) error {
	return withRun(repoRoot, runID, func(root *os.Root) error {
		st, err := readStateIn(root, runID)
		if err != nil {
			return err
		}
		changed, err := fn(&st)
		if err != nil || !changed {
			return err
		}
		// Under the lock the file is still what fn was handed, so it is what
		// the change is judged against and what the save rewrites.
		read, err := readStateIn(root, runID)
		if err != nil {
			return err
		}
		if !onlyRecordChanged(read, st) {
			return refuse("state", "", "", "a reservation save changes only the run's record and its date, and this one changed more of "+runID,
				"nothing was written; this is a fault in abcd, so report it")
		}
		rel := StateRelPath(runID)
		data, err := fsutil.ReadGuardedInRoot(root, rel, maxStateBytes)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rel, err)
		}
		saved, err := withRecordGrown(data, st.UpdatedAt, st.masked().Record[len(read.Record):])
		if err != nil {
			return fmt.Errorf("saving %s: %w", rel, err)
		}
		if err := stateFits(runID, saved); err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomicInRoot(root, rel, saved, filePerm); err != nil {
			return fmt.Errorf("writing %s: %w", rel, err)
		}
		return nil
	})
}

// onlyRecordChanged reports whether after is read with its record grown and
// its date moved, and otherwise equal. Both are taken by value: the slices and
// pointers they share with their callers are only read.
func onlyRecordChanged(read, after State) bool {
	n := len(read.Record)
	if len(after.Record) < n || !reflect.DeepEqual(read.Record, slices.Clip(after.Record[:n])) {
		return false
	}
	read.Record, after.Record = nil, nil
	read.UpdatedAt, after.UpdatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(read, after)
}

// withRecordGrown is the state file data with added appended to its record and
// updated as its updated_at. Each top-level field keeps its place, and every
// byte of every other field and of the record's earlier entries is kept, so
// the version, and any field of a later version than this abcd, is untouched.
func withRecordGrown(data []byte, updated time.Time, added []Entry) ([]byte, error) {
	at, err := json.Marshal(updated)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("the file is not one JSON object")
	}
	var out bytes.Buffer
	out.WriteByte('{')
	put := func(key string, value []byte) error {
		name, err := json.Marshal(key)
		if err != nil {
			return err
		}
		if out.Len() > 1 {
			out.WriteByte(',')
		}
		out.Write(name)
		out.WriteByte(':')
		out.Write(value)
		return nil
	}
	sawRecord, sawUpdated := false, false
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("the file has a field with no name")
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		switch key {
		case stateFieldRecord:
			sawRecord = true
			if value, err = recordGrown(value, added); err != nil {
				return nil, err
			}
		case stateFieldUpdated:
			sawUpdated = true
			value = at
		}
		if err := put(key, value); err != nil {
			return nil, err
		}
	}
	if !sawRecord {
		value, err := recordGrown(nil, added)
		if err != nil {
			return nil, err
		}
		if err := put(stateFieldRecord, value); err != nil {
			return nil, err
		}
	}
	if !sawUpdated {
		if err := put(stateFieldUpdated, at); err != nil {
			return nil, err
		}
	}
	out.WriteByte('}')
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, out.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// recordGrown is the record's JSON with added appended, its earlier entries as
// they stood.
func recordGrown(value json.RawMessage, added []Entry) ([]byte, error) {
	var entries []json.RawMessage
	if len(value) > 0 {
		if err := json.Unmarshal(value, &entries); err != nil {
			return nil, fmt.Errorf("its record is not a list: %w", err)
		}
	}
	for _, e := range added {
		raw, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		entries = append(entries, raw)
	}
	var out bytes.Buffer
	out.WriteByte('[')
	for i, e := range entries {
		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(e)
	}
	out.WriteByte(']')
	return out.Bytes(), nil
}
