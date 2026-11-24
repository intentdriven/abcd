package loop

// discard_unheld.go is what `implement step --discard` ends beyond a held lane
// (iss-2611231404319245): a lane at work whose run nothing drives, and a lane
// handed back for any reason, unachievable included. Before this, such a lane
// had no verb that ended it: it kept its worktree and branch, prune counted it
// held, and its run stayed in progress. Every discard saves the lane's commits
// not on the default branch to a bundle in the run's directory before it
// removes anything (saveLaneBundle), and once nothing of the run is left to
// land or settle it releases what the run still holds and still owns
// (releaseOnDiscard).

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// atWork reports whether l is at one of the lane stages a step performs:
// worktree, brief, implement, validate or land. An auto-plan lane's stages are
// not among them.
func atWork(l Lane) bool {
	if l.AutoPlan != nil {
		return false
	}
	for _, s := range Sequence {
		if l.Stage == s {
			return true
		}
	}
	return false
}

// unheldEnd reports whether --discard may take l beyond the held lane and the
// lanes decidable names: a lane at work, or a lane handed back for any reason.
// Whether its run lets it go is judged under the lock (mayEndUnheld).
func unheldEnd(l Lane) bool {
	return atWork(l) || (l.Stage == StageHandedBack && l.HandBack != nil)
}

// unheldOnly reports whether l is one --discard takes by unheldEnd alone: no
// other rule of decidable admits it, so mayEndUnheld must.
func unheldOnly(repoRoot string, st State, l Lane) bool {
	return unheldEnd(l) && !stuckLane(l) && !driveStopped(st.RunID) && !letGoAtLand(repoRoot, st, l)
}

// mayEndUnheld judges, under the run's lock, a lane unheldOnly admits. It is
// refused while the run is live (runLive); a handed-back lane also while a
// sibling is still at work, since the run holds its siblings only while a
// lane stands handed back; and while the lane's pull request is open or
// merged (unheldPR). closed is set when the forge reports that pull request
// closed, so the discard closes nothing.
func mayEndUnheld(c Context, root *os.Root, lane Lane, g *outageGate, o Options) (closed bool, err error) {
	if err := runLive(c.RepoRoot, root, c.State, lane.ID, o, c.Now); err != nil {
		return false, err
	}
	if lane.Stage == StageHandedBack {
		if busy := busyLanes(c.State); len(busy) > 0 {
			return false, contend(string(StageHeld), "", lane.ID, "lanes of "+c.State.RunID+" are still at work: "+strings.Join(busy, "; ")+
				"; while "+lane.ID+" stands handed back their passing rounds are held rather than landed",
				"run `abcd implement step` until they finish or are held, or discard them first; nothing was changed")
		}
	}
	return unheldPR(c, lane, g)
}

// runLive refuses, as contention, a run something still drives: a recorded
// driver that is alive and is not the session this call runs under, or an
// agent of any lane still at work that is running or may still be in its call
// (judgeAgent). The caller's own session is judged as no driver, so its own
// agents go by their runner claims and the lease. A process table that cannot
// be read is refused too: nothing then says the run is not live. A run a stop
// holds, or halted and no take-over has cleared, is refused as well: the stop
// settles its lanes, and its take-over is how the run moves on.
func runLive(repoRoot string, root *os.Root, st State, laneID string, o Options, now time.Time) error {
	again := "run `abcd implement step --run " + st.RunID + " --discard " + laneID + "` again"
	tab, err := readProcs()
	if err != nil {
		return contend(string(StageHeld), "", laneID, "the process table cannot be read, so whether anything still drives "+st.RunID+" cannot be told",
			again+" once it can be read; nothing was changed")
	}
	c, err := readControl(root, st.RunID)
	if err != nil {
		return err
	}
	if c.Stop != nil || o.stopRead(c.Shared).Stop != nil {
		return refuse(string(StageHeld), "", laneID, "a stop holds "+st.RunID+" or halted it, and its lanes are the stop's to settle",
			"once the stop is lifted, take the run over with `abcd implement resume --take "+st.RunID+"`, then "+again+"; nothing was changed")
	}
	if d := c.Driver; d != nil && tab.Alive(d.host()) {
		if here, _, herr := o.driverHere(tab); herr != nil || here.PID != d.PID {
			return contend(string(StageHeld), "", laneID, fmt.Sprintf("%s is live: it is driven by %s, which is alive", st.RunID, driverName(*d)),
				"back off: only a run nothing drives has its lanes ended this way; once that driver is gone, "+again+"; nothing was changed")
		}
		c.Driver = nil
	}
	procs := ledgerOf(o, runner.LedgerQuery{Run: st.RunID})
	var live []string
	for _, l := range st.Lanes {
		if !atWork(l) {
			continue
		}
		for _, a := range l.Awaits {
			switch judgeAgent(procs, repoRoot, st.RunID, l.ID, a, c, &tab, now) {
			case AgentAlive:
				live = append(live, l.ID+"'s "+a.Role+" is running")
			case AgentUnknown:
				live = append(live, l.ID+"'s "+a.Role+" may still be in its call")
			}
		}
	}
	if len(live) > 0 {
		return contend(string(StageHeld), "", laneID, st.RunID+" is live: "+strings.Join(live, "; "),
			"let the agent finish and hand its receipt in; an agent that is gone counts as gone once its runner's claim ends or, for a host's agent, a reservation lease after it was handed out, and then "+
				again+"; nothing was changed")
	}
	return nil
}

// unheldPR asks the forge what became of the lane's pull request: one open,
// armed or not, is refused, since its work can still merge; one merged has
// landed, which a step records; one closed is reported as such. A lane whose
// state records none but whose branch was pushed at its landing may have one
// the state never took (opened, then not read back, which the landing keeps
// for its next step to find): the forge is asked for the branch's open pull
// request, and one listed, or a listing that cannot be read, is refused. A
// lane never pushed, or handed back as closed, asks nothing. While the network
// is down either read waits on the shared probe.
func unheldPR(c Context, lane Lane, g *outageGate) (closed bool, err error) {
	again := "`abcd implement step --run " + c.State.RunID + " --discard " + lane.ID + "`"
	if lane.PR <= 0 {
		return false, unrecordedPR(c, lane, g, again)
	}
	if lane.HandBack != nil && lane.HandBack.Kind == HandBackClosed {
		return true, nil
	}
	n := strconv.Itoa(lane.PR)
	if g.down(implement.ServiceNetwork) {
		return false, g.networkWait(string(StageHeld), lane.ID, fmt.Sprintf("the discard of %s (reading pull request #%s)", lane.ID, n))
	}
	out, err := forge(c, lane, "pr", "view", n, "--json", "state", "--jq", ".state")
	if err != nil {
		if r, ok := AsRefusal(err); ok {
			g.take(r)
		}
		return false, err
	}
	switch strings.TrimSpace(out) {
	case "CLOSED":
		return true, nil
	case "MERGED":
		return false, refuse(string(StageHeld), "", lane.ID, fmt.Sprintf("%s's pull request #%s has merged, so its work has landed and is not the discard's to end", lane.ID, n),
			"run `abcd implement step --run "+c.State.RunID+"`, which records the landing; nothing was changed")
	}
	armed := ""
	if lane.Landing != nil && lane.Landing.Armed {
		armed = " and armed to merge"
	}
	return false, refuse(string(StageHeld), "", lane.ID, fmt.Sprintf("%s's pull request #%s is open%s, so its work can still merge", lane.ID, n, armed),
		"close it on the forge (`gh pr close "+n+"`), or let the run land it; then run "+again+" again; nothing was changed")
}

// unrecordedPR is unheldPR's read for a lane whose state records no pull
// request: nil for a lane never pushed at its landing, or whose branch the
// forge lists no open pull request for; a refusal for one listed, and the
// forge's own error for a listing that cannot be read.
func unrecordedPR(c Context, lane Lane, g *outageGate, again string) error {
	if lane.Branch == "" || lane.Landing == nil || lane.Landing.Pushed == "" {
		return nil
	}
	if g.down(implement.ServiceNetwork) {
		return g.networkWait(string(StageHeld), lane.ID, fmt.Sprintf("the discard of %s (listing its branch's pull requests)", lane.ID))
	}
	found, err := findPR(c, lane)
	if err != nil {
		if r, ok := AsRefusal(err); ok {
			g.take(r)
		}
		return err
	}
	if found == nil {
		return nil
	}
	n := strconv.Itoa(found.Number)
	return refuse(string(StageHeld), "", lane.ID, fmt.Sprintf("%s's branch has pull request #%s open on the forge, which its state never recorded, so its work can still merge", lane.ID, n),
		"close it on the forge (`gh pr close "+n+"`), or let the run land it; then run "+again+" again; nothing was changed")
}

// saveBundle is saveLaneBundle; a test stands one in.
var saveBundle = saveLaneBundle

// saveLaneBundle saves the commits of the lane's branch that the default
// branch does not hold (origin's, as gitutil.DefaultRef resolves it; the whole
// branch where none resolves) to a git bundle in the run's own directory,
// named for the lane and its tip, before a discard removes the branch. It
// returns the bundle's path relative to the repository root and how many
// commits it holds, "" and 0 when the lane names no branch, the branch is
// gone, or it holds no commit of its own; and tip, the branch's tip as read
// once here, which the bundle holds and at which alone the caller deletes the
// branch, "" when the branch is gone. History is read as stored, so no graft
// or replace ref hides a commit from the bundle.
//
// The file is created through root, the checkout's handle, in the run's
// directory: a name already there, a link included, refuses the save. git
// writes the bundle on its standard output to a new file of a part-written
// name the save opened itself, never opening a path of its own, and the
// bundle takes its own name only once whole. Its header must list the branch
// at tip and nothing else, or the file is removed and the save refused: the
// branch moved while it was read.
func saveLaneBundle(root *os.Root, repoRoot, runID string, lane Lane) (rel string, n int, tip string, err error) {
	if lane.Branch == "" {
		return "", 0, "", nil
	}
	ref := "refs/heads/" + lane.Branch
	tip, err = gitutil.RunCappedAsStored(repoRoot, maxGitOutput, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", 0, "", nil
	}
	not := []string{}
	if def := gitutil.DefaultRef(repoRoot); def != "" && def != ref {
		not = append(not, "^"+def)
	}
	count, err := gitutil.RunCappedAsStored(repoRoot, 64, append(append([]string{"rev-list", "--count", tip}, not...), "--")...)
	if err != nil {
		return "", 0, "", fmt.Errorf("counting %s's commits not on the default branch: %w", lane.Branch, err)
	}
	if n, err = strconv.Atoi(strings.TrimSpace(count)); err != nil {
		return "", 0, "", err
	}
	if n == 0 {
		return "", 0, tip, nil
	}
	name := "lane"
	if ValidLaneID(lane.ID) {
		name = lane.ID
	}
	file := name + "-" + tip[:12] + ".bundle"
	partial := file + ".partial"
	rel = runRel(runID) + "/" + file
	taken := fmt.Errorf("%s already exists, and a discard writes its bundle only to a file of its own making", rel)
	dir, err := root.OpenRoot(runRel(runID))
	if err != nil {
		return "", 0, "", fmt.Errorf("opening the run's directory: %w", err)
	}
	defer dir.Close()
	if _, err := dir.Lstat(file); err == nil {
		return "", 0, "", taken
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", 0, "", fmt.Errorf("reading %s: %w", rel, err)
	}
	// A part-written bundle is what a discard cut short leaves; only a discard
	// writes that name, and nothing reads it, so it is removed (a link itself,
	// never what it names) before the bundle is written afresh.
	if err := dir.Remove(partial); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", 0, "", fmt.Errorf("removing the part-written %s.partial: %w", rel, err)
	}
	f, err := dir.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, "", fmt.Errorf("creating %s.partial: %w", rel, err)
	}
	head := &bundleHeader{}
	err = gitutil.StreamAsStored(repoRoot, io.MultiWriter(f, head), append([]string{"bundle", "create", "-q", "-", ref}, not...)...)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && !head.listsOnly(tip+" "+ref) {
		err = fmt.Errorf("%s moved from %s while its commits were saved", lane.Branch, shortSHA(tip))
	}
	// The whole bundle takes its name only once checked, by a link that
	// refuses a name already there, so no name a retry refuses is ever left
	// half written.
	if err == nil {
		if err = dir.Link(partial, file); errors.Is(err, fs.ErrExist) {
			err = taken
		}
	}
	_ = dir.Remove(partial)
	if err != nil {
		return "", 0, "", fmt.Errorf("saving %s's commits to %s: %w", lane.Branch, rel, err)
	}
	return rel, n, tip, nil
}

// maxBundleHeader bounds the bundle header bundleHeader keeps.
const maxBundleHeader = 1 << 20

// bundleHeader keeps a bundle's header as git writes it: the lines before
// the first blank line, up to maxBundleHeader bytes. Everything after passes
// by.
type bundleHeader struct {
	buf  []byte
	done bool
	over bool
}

func (h *bundleHeader) Write(p []byte) (int, error) {
	if h.done || h.over {
		return len(p), nil
	}
	h.buf = append(h.buf, p...)
	if i := bytes.Index(h.buf, []byte("\n\n")); i >= 0 {
		h.buf, h.done = h.buf[:i+1], true
	} else if len(h.buf) > maxBundleHeader {
		h.buf, h.over = nil, true
	}
	return len(p), nil
}

// listsOnly reports whether the header, read whole, lists exactly one ref,
// line ("<sha> <refname>"): its signature line first, then its prerequisites
// ("-") and capabilities ("@"), which are passed over.
func (h *bundleHeader) listsOnly(line string) bool {
	if !h.done {
		return false
	}
	lines := strings.Split(strings.TrimSuffix(string(h.buf), "\n"), "\n")
	if !strings.HasPrefix(lines[0], "# v") {
		return false
	}
	var refs []string
	for _, l := range lines[1:] {
		if !strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "@") {
			refs = append(refs, l)
		}
	}
	return len(refs) == 1 && refs[0] == line
}

// bundleNote is the discard record's clause for a saved bundle.
func bundleNote(rel string, n int, branch string) string {
	return fmt.Sprintf("saved its %d commit(s) not on the default branch to %s (`git fetch %s %s` restores them)", n, rel, rel, branch)
}

// releaseOnDiscard registers, on the state a discard's callback has just
// changed, the release of every reservation the run still holds once that
// state is written, under the same lock (State.afterWritten), when nothing of
// the run is left to land or to settle: no lane at work or held, no spec step
// pending (which the next step would open a lane for), and no lane handed back
// with its pull request still armed (armingOwed), the rule completesRun keeps
// for a landing. A reservation the record already shows released since it was
// taken is not released again, and each is released only while it is provably
// still the run's own (releaseOwn). An issue run's release publishes the
// discarded lane's hand-back, where it carries one, as releaseOnHandBack would
// have. What came of each is written to the run record, and the run's
// reserved_at cleared, by a second write under the same hold, as a stop's
// release is. *said is what the releases came to, "" for none.
func releaseOnDiscard(repoRoot string, st *State, laneID string, now time.Time, said *string) {
	st.afterWritten(func(s *State, rewrite func() error) {
		if len(s.Pending) > 0 {
			return
		}
		kind := ""
		for _, l := range s.Lanes {
			switch l.Stage {
			case StageDone, StageDiscarded, StageHandedBack:
			default:
				return
			}
			if armingOwed(l) {
				return
			}
			if l.ID == laneID && l.HandBack != nil && s.Issue() != "" {
				kind = tombstoneKind(*l.HandBack)
			}
		}
		var out []string
		for _, id := range s.reservedIssues() {
			if releasedThisHoldFor(*s, id) {
				continue
			}
			clause, failed := releaseOwn(repoRoot, *s, id, kind, now)
			logged := StageReservationReleased
			if failed {
				tail, _, _ := lapseTail(repoRoot, s.forIssue(id), now)
				clause += tail
				logged = StageReservationReleaseFailed
			}
			s.Record = append(s.Record, Entry{At: now, Stage: logged, Note: clause, Issue: s.issueOn(id)})
			out = append(out, clause)
		}
		if len(out) == 0 {
			return
		}
		s.ReservedAt = nil
		for i := range s.Bundle {
			s.Bundle[i].ReservedAt = nil
		}
		s.UpdatedAt = now
		*said = strings.Join(out, "; ")
		if err := rewrite(); err != nil {
			*said += "; the run record could not take this: " + err.Error()
		}
	})
}

// releaseOwn releases the reservation on id only while it is provably still
// the run's own, and says what came of it as releaseReservation does. The tip
// is read from this checkout's view, as last fetched: one the view shows
// absent or released already is nothing of the run's to release, and nothing
// is pushed; one of another copy or person, or unreadable, is left as it
// stands; one of this copy's is released, fenced on that commit
// (releaseReservationAt, so a tip that moves on before the push is left too),
// only when its history reaches one of the run's own logged acts through
// renewals alone (provablyOwn), and otherwise left as it stands to lapse. No clock
// of the run's own is compared with the tip's: a write of the run's state is
// no reservation act, and a renewal made by another hand in this copy (a
// load-gate waiter's) is the same hold.
func releaseOwn(repoRoot string, st State, id, kind string, now time.Time) (clause string, failed bool) {
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return "the reservation on " + id + " was not released: the reservations as last fetched here could not be read (" +
			termsafe.Sanitize(fsutil.RedactHome(err.Error())) + ")", true
	}
	r, ok := snap.Lookup(id)
	switch {
	case !ok:
		return "nothing of this run's to release on " + id + ": this checkout's view shows no reservation of it", false
	case r.State == claim.StateReleased:
		return "nothing of this run's to release on " + id + ": this checkout's view shows it released already", false
	case r.Copy != claim.CopyThis || (r.State != claim.StateLive && r.State != claim.StateExpired):
		return "left the reservation on " + id + " as it stands: this checkout's view shows it held from another copy or by another person, or unreadable", false
	}
	if own, why := provablyOwn(repoRoot, st, id, r.Commit); !own {
		return "left the reservation on " + id + " as it stands, to lapse at " + stamp(r.EffectiveExpiry) +
			": it is not provably this run's, since " + why, false
	}
	return releaseReservationAt(repoRoot, id, r.Commit, kind, now)
}

// maxHoldWalk bounds the reservation commits provablyOwn reads back from a tip.
const maxHoldWalk = 64

// provablyOwn reports whether tip, a reservation commit of this copy's on id, is
// still the run's own hold: read back along first parents, as stored, from
// tip, its history reaches a commit that is one of the run's own logged
// reservation acts (runActs: the commit's claimed-at is the act's time to the
// second), through reservation commits of tip's own copy alone, none of them a
// tombstone and none claimed after the lease of the one before it had ended.
// why is the reason it is not, for the record: a release or a lapse in
// between, a take-over, a commit that cannot be read, or no anchor within
// maxHoldWalk commits.
func provablyOwn(repoRoot string, st State, id, tip string) (own bool, why string) {
	acts := runActs(st, id)
	if len(acts) == 0 {
		return false, "the run's record logs no reservation of " + id
	}
	out, err := gitutil.RunCappedAsStored(repoRoot, maxHoldWalk*48, "rev-list", "--first-parent", "--max-count="+strconv.Itoa(maxHoldWalk), tip, "--")
	if err != nil {
		return false, "its history could not be read"
	}
	copyKey := ""
	var child *claim.ReservationMessage
	for _, sha := range strings.Fields(out) {
		body, err := gitutil.RunCappedAsStored(repoRoot, claim.MaxMessageBytes+1, "log", "-1", "--format=%B", sha, "--")
		if err != nil {
			return false, "its commit " + shortSHA(sha) + " could not be read"
		}
		m, err := claim.ReadReservationMessage(body)
		switch {
		case err != nil:
			return false, "its commit " + shortSHA(sha) + " is not a reservation that can be read"
		case m.Released():
			return false, "it was released at " + stamp(m.ReleasedAt) + " and reserved again after"
		case copyKey != "" && m.Copy != copyKey:
			return false, "it was taken over from another copy after the run's reservation"
		case child != nil && child.ClaimedAt.After(heldUntil(m)):
			return false, "its lease lapsed at " + stamp(heldUntil(m)) + " and it was reserved again after"
		}
		copyKey = m.Copy
		if slices.ContainsFunc(acts, m.ClaimedAt.Equal) {
			return true, ""
		}
		child = &m
	}
	return false, "no reservation the run logged lies within its last " + strconv.Itoa(maxHoldWalk) + " commits"
}

// heldUntil is when the reservation m stopped holding: its expiry, clamped to
// a day after it was claimed, as lease.Judge clamps it.
func heldUntil(m claim.ReservationMessage) time.Time {
	if most := m.ClaimedAt.Add(lease.Max); m.ExpiresAt.After(most) {
		return most
	}
	return m.ExpiresAt
}

// runActs is the time of every reservation act on id the run's state logs,
// to the second, as a reservation commit's claimed-at carries it: the run's
// and each member's reserved_at, and each reservation and renewal line its
// record holds for id. Each was taken from the clock the act's commit was
// claimed at.
func runActs(st State, id string) []time.Time {
	var out []time.Time
	add := func(t time.Time) { out = append(out, t.UTC().Truncate(time.Second)) }
	if recordid.SameID(id, st.Key) && st.ReservedAt != nil {
		add(*st.ReservedAt)
	}
	for _, m := range st.Bundle {
		if recordid.SameID(m.Issue, id) && m.ReservedAt != nil {
			add(*m.ReservedAt)
		}
	}
	for _, e := range st.Record {
		if (e.Stage == StageReservation || e.Stage == StageReservationRenewed) && recordid.SameID(st.lineIssue(e), id) {
			add(e.At)
		}
	}
	return out
}
