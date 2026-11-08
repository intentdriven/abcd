package autonomy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// holders.go — whose an item is, when several sessions work from one copy or
// from the worktrees beside it (iss-2611020719483054, iss-2610311406481214,
// iss-2611070830056851). The owed reading finds a copy's own work; this step
// leaves out of it what another live session holds, so a session is not held
// at its stop for work it can neither push nor finish, and names each holder
// once.
//
// The boundary: a session never loses sight of its own work. An item is this
// session's whatever else is said of it when the evidence names this session:
// the guard recorded this session creating the branch, or the branch is a lane
// of a run this session drives. Only then is another session's word read, and
// an item is another's only when that session is running now:
//
//   - a lane of a run whose control record names another session as its
//     driver (the loop's own record of who steps the run);
//   - a branch the guard recorded another session creating (CreatedBranch's
//     Session);
//   - a branch checked out in a worktree another session started in (that
//     session's occupancy marker), the work a peer has taken up there;
//   - a reservation only runs another session drives are for.
//
// Whatever the evidence cannot tell stays owed: a branch recorded before
// sessions were kept, a run with no session as its driver (a plain terminal's)
// or none at all, a reservation made by hand, an item whose holder has ended,
// and every item when the stopping session has no id abcd keys on or the live
// sessions cannot be read. A peer's work returns to the sessions left once
// the peer ends.

// holding is what one other live session holds, in plain words, for its note.
type holding struct {
	session string
	items   []string
}

// claimant is a session the evidence names for an item, and how.
type claimant struct {
	session string
	why     string
}

// scope leaves out of the copy's own branches and reservations what another
// live session holds, recording each holder for one note. It runs after
// findOwn and before the items are judged.
func (r *owedReader) scope() {
	if !ValidSessionID(r.me) {
		return
	}
	for _, name := range r.ownNames() {
		lane := r.lanes[name]
		if r.recordedBy[name] == r.me || lane.driver == r.me {
			continue
		}
		var claims []claimant
		if lane.driver != "" {
			claims = append(claims, claimant{lane.driver, "its run's lane"})
		}
		if s := r.recordedBy[name]; s != "" {
			claims = append(claims, claimant{s, "it created"})
		}
		if wt := r.elsewhere[name]; wt != "" {
			for _, s := range r.startedIn(wt) {
				claims = append(claims, claimant{s, "in its worktree"})
			}
		}
		if c, ok := r.holder(claims); ok {
			delete(r.own, name)
			delete(r.lanes, name)
			r.hold(c.session, fmt.Sprintf("%s (%s)", name, c.why))
		}
	}
	for _, res := range r.mine {
		drivers := r.runDrivers[res.Record]
		if len(drivers) == 0 {
			continue
		}
		var claims []claimant
		ours := false
		for _, d := range drivers {
			if d == "" || d == r.me {
				ours = true
				break
			}
			claims = append(claims, claimant{d, "its run's"})
		}
		if ours {
			continue
		}
		if c, ok := r.holder(claims); ok {
			r.heldRes[res.Record] = c.session
			r.hold(c.session, fmt.Sprintf("the reservation of %s (%s)", res.Record, c.why))
		}
	}
	r.holdNotes()
}

// holder is the first claimant that is another session running now; false
// when none is, or the live sessions cannot be read.
func (r *owedReader) holder(claims []claimant) (claimant, bool) {
	for _, c := range claims {
		if c.session == "" || c.session == r.me {
			continue
		}
		if live, ok := r.liveSessions(); ok {
			if _, running := live[c.session]; running {
				return c, true
			}
		}
	}
	return claimant{}, false
}

// startedIn are the live sessions other than this one whose start checkout
// is the worktree at path.
func (r *owedReader) startedIn(path string) []string {
	live, ok := r.liveSessions()
	if !ok {
		return nil
	}
	var out []string
	for id, checkout := range live {
		if id != r.me && sameDir(checkout, path) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func (r *owedReader) hold(session, item string) {
	if r.held == nil {
		r.held = map[string]*holding{}
	}
	h := r.held[session]
	if h == nil {
		h = &holding{session: session}
		r.held[session] = h
	}
	h.items = append(h.items, item)
}

// maxHeldNamed bounds the items one holder's note names, so the note stays
// within the line the stop's reason gives each note.
const maxHeldNamed = 3

// holdNotes names each holder once, with what it holds, and, when another
// live session started in this very copy, the breach of one session per
// checkout that made their work meet here.
func (r *owedReader) holdNotes() {
	sessions := make([]string, 0, len(r.held))
	for s := range r.held {
		sessions = append(sessions, s)
	}
	sort.Strings(sessions)
	for _, s := range sessions {
		items := r.held[s].items
		named := items
		more := ""
		if len(named) > maxHeldNamed {
			named, more = named[:maxHeldNamed], fmt.Sprintf(" and %d more", len(items)-maxHeldNamed)
		}
		r.note("live session %s holds %s%s: that session's work, not owed here", s, strings.Join(named, ", "), more)
	}
	if !r.liveOK {
		return
	}
	var sharing []string
	for id, checkout := range r.live {
		if id != r.me && sameDir(checkout, r.root) {
			sharing = append(sharing, id)
		}
	}
	if len(sharing) == 0 {
		return
	}
	sort.Strings(sharing)
	r.note("this working copy is shared with %d other live session(s) (%s), against one session per checkout: each session works in a worktree of its own in the machine-scoped store (`abcd ahoy worktree add <name>`)",
		len(sharing), strings.Join(sharing, ", "))
}

// liveSessions is the live sessions by id, each with the checkout it started
// in, read once a check; false when they could not be read, which a note
// says, and which leaves every item owed. The read is given up when the
// check's budget ends first: the process table is read by a program of its
// own, which the check's context does not end.
func (r *owedReader) liveSessions() (map[string]string, bool) {
	if r.liveTried {
		return r.live, r.liveOK
	}
	r.liveTried = true
	type answer struct {
		sessions []occupancy.Session
		err      error
	}
	ch := make(chan answer, 1)
	go func() {
		s, err := r.liveRead()
		ch <- answer{s, err}
	}()
	var a answer
	select {
	case a = <-ch:
	case <-r.ctx.Done():
		a.err = r.ctx.Err()
	}
	if a.err != nil {
		r.note("the live sessions could not be read (%s), so no item was left out as another session's", fsutil.RedactHome(a.err.Error()))
		return nil, false
	}
	r.live = map[string]string{}
	for _, s := range a.sessions {
		if ValidSessionID(s.ID) {
			r.live[s.ID] = s.Checkout
		}
	}
	r.liveOK = true
	return r.live, true
}

// sameDir reports whether two paths name one directory: equal once their
// symlinks are resolved, or the same file to the filesystem.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ra, ea := filepath.EvalSymlinks(a)
	rb, eb := filepath.EvalSymlinks(b)
	if ea == nil && eb == nil && ra == rb {
		return true
	}
	fa, ea := os.Stat(a)
	fb, eb := os.Stat(b)
	return ea == nil && eb == nil && os.SameFile(fa, fb)
}
