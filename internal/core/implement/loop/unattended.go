package loop

// unattended.go answers the host's approval-prompt hook
// (iss-2610171110154902): is a build or a drain running for the session's
// checkout, so that nobody is at the terminal to answer a prompt? While one
// is, the hook refuses the prompt rather than let the run wait on a person who
// is not there; otherwise it says nothing and the host's prompt shows.

import (
	"errors"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The kinds of unattended work.
const (
	UnattendedBuild = "build"
	UnattendedDrain = "drain"
)

// Unattended is the build or drain a session's checkout is running.
type Unattended struct {
	// Kind is UnattendedBuild or UnattendedDrain.
	Kind string
	// RunID and Key name a build's run and the record it builds; both are
	// empty for a drain. Key is empty too when the state names no record id
	// (an itd- or an iss- id): it is read from a local-tier file and quoted
	// to the agent, so nothing else is passed on.
	RunID string
	Key   string
}

// RunningUnattended reports the build or drain running for the session whose
// checkout is sessionRoot, as of now. A build counts while its run is not
// complete and its state was written within lease.Default of now; a drain
// while it has not stopped and its state was written within the same window.
// The window is the reservation lease's: a run idle past it is the run the
// reservation system already treats as abandoned, so a checkout holding old
// unfinished runs (the main checkout keeps many from 2026-10-12) does not
// count as running one. A state dated more than lease.Skew ahead of now is
// not counted either, so a planted far-future date cannot hold it on. Nor is
// a run waiting on the person (waitsOnPerson): the person is at the terminal
// by then, taking the work over, and their own prompts must show.
//
// The session also counts when sessionRoot is the worktree of a lane of a
// counting build in any other worktree of the same repository: a lane's
// checkout lives in the machine's worktree store, while its run's state lives
// in the checkout that started it.
//
// A run or drain state that cannot be read does not count and is passed over.
// The error is returned only when nothing could be read: the checkout cannot
// be opened, or every state it holds failed and no lane matched. The caller
// reads an error as no answer, never as a run.
func RunningUnattended(sessionRoot string, now time.Time) (Unattended, bool, error) {
	u, ok, read, err := runningIn(sessionRoot, now, nil)
	if ok {
		return u, true, nil
	}
	if wts, lerr := gitutil.ListWorktrees(sessionRoot, gitutil.MaxWorktreeListing); lerr == nil {
		for _, wt := range wts {
			if wt.Bare || wt.Path == "" || samePath(sessionRoot, wt.Path, sessionRoot) {
				continue
			}
			// A lane counts only while it has work left: a session opened in
			// a finished or discarded lane's worktree is a person looking at
			// it, even while a sibling lane keeps the run unfinished.
			lane := func(st State) bool {
				for _, l := range st.Lanes {
					if l.Stage == StageDone || l.Stage == StageDiscarded {
						continue
					}
					if samePath(sessionRoot, l.Worktree, sessionRoot) {
						return true
					}
				}
				return false
			}
			if u, ok, _, _ := runningIn(wt.Path, now, lane); ok {
				return u, true, nil
			}
		}
	}
	if read == 0 && err != nil {
		return Unattended{}, false, err
	}
	return Unattended{}, false, nil
}

// runningIn is the counting build or drain in the checkout at repoRoot. With
// a lane filter, only a build whose state it accepts counts, and the drain is
// not consulted: a drain's lanes are builds of its own checkout. read counts
// the states read cleanly, and err is the first that was not.
func runningIn(repoRoot string, now time.Time, lane func(State) bool) (_ Unattended, _ bool, read int, _ error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Unattended{}, false, 0, err
	}
	defer root.Close()
	ids, err := runIDs(root)
	if err != nil {
		return Unattended{}, false, 0, err
	}
	var first error
	for _, id := range ids {
		st, err := readStateIn(root, id)
		if err != nil {
			first = errors.Join(first, err)
			continue
		}
		read++
		if st.Complete() || st.waitsOnPerson() || !recent(st.UpdatedAt, now) || (lane != nil && !lane(st)) {
			continue
		}
		u := Unattended{Kind: UnattendedBuild, RunID: st.RunID}
		if _, ok := keyCheck(st.Key); ok {
			u.Key = st.Key
		}
		return u, true, read, nil
	}
	if lane != nil {
		return Unattended{}, false, read, first
	}
	d, live, err := readDrain(repoRoot)
	switch {
	case err != nil:
		first = errors.Join(first, err)
	case live:
		read++
		if recent(d.UpdatedAt, now) {
			return Unattended{Kind: UnattendedDrain}, true, read, nil
		}
	}
	return Unattended{}, false, read, first
}

// waitsOnPerson reports whether any lane of the run waits on the person: one
// handed back (its intent is theirs to replan; the run stays incomplete), one
// held until they release or discard it, or one carrying a hold not yet
// released. It errs towards yes, since a run that does not count leaves the
// host's own prompt to show.
func (s State) waitsOnPerson() bool {
	for _, l := range s.Lanes {
		if l.Stage == StageHandedBack || l.Stage == StageHeld || l.HandBack != nil || (l.Hold != nil && !l.Hold.Released) {
			return true
		}
	}
	return false
}

// recent reports whether a state written at t is inside the reservation
// lease of now, and not dated past the allowed clock skew ahead of it.
func recent(t, now time.Time) bool {
	age := now.Sub(t)
	return age < lease.Default && age >= -lease.Skew
}
