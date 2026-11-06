package loop

// stop.go is what a run does with a stop (itd-2611041549471712,
// spc-2611041705580524; dcn-2611041550139264): one check any driver calls before
// a move, and the order the stop then takes. A person cannot interrupt a tool
// call as it runs, so a stop acts at the boundary between moves: no lane opens,
// no stage is handed out and no agent starts; an agent in a call finishes it and
// its receipt is collected, and goes no further; and only then, with no agent
// live in a lane, is the lane's uncommitted work saved aside as a restart saves
// it. When every lane has settled the run's reservation is released so a peer
// may take the work, and the day's page is rewritten whole. A run halted stays
// halted when the stop lifts, until `abcd implement resume --take` reserves
// again.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageStop is the refusal stage and the run record's stage for a stop.
const StageStop = "stop"

// noReservationHeld is a stop's account of a run that held no reservation, which
// a take-over therefore does not take again.
const noReservationHeld = "the run held no reservation"

// inItsCall is what a lane's account says while an agent holds it back.
const inItsCall = "an agent is still in its call"

// stopKeptPrefix opens the reason a stop marks an agent lost with.
const stopKeptPrefix = "stopped: a stop set by "

// defaultStopHome is the home a stop is read from; TestMain empties it, which
// reads as no stop, so a stop on the developer's machine stops no test.
var defaultStopHome = runstop.MachineStopHome

// accountStopHome is the account's own home (runstop.AccountHome), which is where the
// verb writes a stop and which the variable a process inherits does not move. A test
// names a fixture.
var accountStopHome = runstop.ThisAccountHome

// afterStopCheck runs between a move's check and its lock; a test sets a stop there.
var afterStopCheck = func() {}

// stopReadingOff is set by the package's TestMain alone: a stop on the machine
// running the tests stops no test, and a test of the stop names its own home.
var stopReadingOff bool

// stopRead is the one read of a stop in this package, for a run started under pin.
func (o Options) stopRead(pin *runstop.Pin) runstop.Reading {
	if stopReadingOff && o.stopHome == "" {
		return runstop.Reading{}
	}
	account := ""
	if o.stopHome == "" {
		account = accountStopHome()
	}
	return runstop.ReadHomes(o.stopHomeOf(), account, pin)
}

// stopHomeOf is the home this call reads a stop from.
func (o Options) stopHomeOf() string {
	if o.stopHome != "" {
		return o.stopHome
	}
	return defaultStopHome()
}

// stopCheck is the check every move makes: nil when the run may move. With a stop
// in force it applies the stop to the run as far as it can this call (haltRun), and
// refuses at exit 3 at the stage `stop`, so a driver reads the refusal as the end of
// its work. A run a stop halted is refused the same way once the stop lifts, until a
// take-over reserves it again. A file the read refused is no stop, and the returned
// note names it for the move's answer.
func stopCheck(repoRoot, runID string, steps Stages, o Options) (string, error) {
	c, err := runControl(repoRoot, runID)
	if err != nil {
		return "", err
	}
	r := o.stopRead(c.Shared)
	note := readingNote(r)
	settle := func(s runstop.Stop) (string, error) {
		if driveStopped(runID) {
			return note, stopOverDriveStop(runID, s)
		}
		rec, err := haltRun(repoRoot, runID, s, steps, o)
		if err != nil || rec == nil {
			return note, err
		}
		return note, stopRefusal(runID, *rec)
	}
	if r.Stop != nil {
		return settle(*r.Stop)
	}
	if r.Held != "" {
		// A read that cannot be trusted is neither a stop nor its lift: the run
		// keeps what it holds and moves nothing.
		return note, heldRefusal(r.Held)
	}
	if c.Shared == nil {
		pinShared(repoRoot, runID, o)
	}
	mark := c.Stop
	switch {
	case mark == nil:
		return note, nil
	case mark.HaltedAt != nil:
		said := "the stop has lifted"
		switch {
		case !mark.FromFile:
			said = "it came from a signal"
		case len(r.Refused) > 0:
			said = "the read of where it stood was refused, so whether it lifted cannot be told"
		}
		return note, contend(StageStop, "", "", fmt.Sprintf("%s was halted by a stop set by %s at %s, and let its claims go; %s",
			runID, termsafe.Sanitize(mark.SetBy), mark.SetAt.UTC().Format(time.RFC3339), said),
			"run `abcd implement resume --take "+runID+"`, which reserves the work again before anything steps")
	case !mark.FromFile:
		// A signal's stop has no file to lift: it settles, and stays.
		return settle(runstop.Stop{SetBy: mark.SetBy, SetAt: mark.SetAt})
	case len(r.Refused) > 0:
		// Only a read that saw clearly and found no stop is a lift.
		return note, stopRefusal(runID, *mark)
	}
	// Lifted before the run let its claims go: it holds them still, so it just
	// goes on.
	return note, goOnAfterLift(repoRoot, runID, o)
}

// readingNote is what a read has to say beyond its answer: each file it refused and
// each stop another account left.
func readingNote(r runstop.Reading) string {
	return termsafe.Sanitize(strings.Join(append(slices.Clone(r.Refused), r.Foreign...), "; "))
}

// runControl is the run's control record, read without the lock; an empty one for
// a run with none. One that cannot be read is a refusal, never an empty record: it
// names the folder the run reads a stop from.
func runControl(repoRoot, runID string) (control, error) {
	if !ValidRunID(runID) {
		return control{}, nil
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return control{}, nil
	}
	defer root.Close()
	return readControl(root, runID)
}

// heldRefusal is the answer a move gets while the read cannot be trusted.
func heldRefusal(why string) error {
	return contend(StageStop, "", "", termsafe.Sanitize(why)+", so nothing moves and nothing is let go",
		"tell the person; they put it back with `abcd ahoy shared-folder`, naming the folder and account the run started under")
}

// pinShared fixes the shared folder the setting names as the run's for its life.
// A setting that cannot be read pins nothing, and the read says why.
func pinShared(repoRoot, runID string, o Options) {
	home := o.stopHomeOf()
	if home == "" {
		return
	}
	pin := runstop.PinOf(home)
	if pin == nil {
		return
	}
	_ = mutate(repoRoot, runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Shared != nil {
			return false, err
		}
		c.Shared = pin
		return false, writeControl(root, runID, c)
	})
}

// clearStop removes the run's stop mark, with why in its record.
func clearStop(repoRoot, runID, why string, o Options) error {
	return mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Stop == nil {
			return false, err
		}
		c.Stop = nil
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		now := o.now()
		st.Record = append(st.Record, Entry{At: now, Stage: StageStop, Note: why})
		st.UpdatedAt = now
		return true, nil
	})
}

// goOnAfterLift removes the mark of a stop lifted before the run halted. A lane
// that had settled has its worktree at its last commit and its agent marked lost:
// that await is told again, so the next step names a fresh agent and the mark does
// not outlive it, reading a live agent as lost.
func goOnAfterLift(repoRoot, runID string, o Options) error {
	return mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Stop == nil {
			return false, err
		}
		c.Stop = nil
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		now := o.now()
		note := "the stop lifted before the run halted, so it goes on"
		for i, lane := range st.Lanes {
			retold := false
			lane.Awaits = slices.Clone(lane.Awaits)
			for k, a := range lane.Awaits {
				if strings.HasPrefix(a.Kept, stopKeptPrefix) {
					// What a restart clears: the fresh agent's return is its own.
					a.Since, a.Kept, a.Answered, a.Paused, a.Running = now, "", nil, nil, nil
					lane.Awaits[k] = a
					retold = true
				}
			}
			if retold {
				st.Lanes[i] = lane
				note += "; " + lane.ID + "'s agent is told again"
			}
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageStop, Note: note})
		st.UpdatedAt = now
		return true, nil
	})
}

// stopRefusal is the answer a move gets while a stop holds.
func stopRefusal(runID string, rec RunStop) error {
	state := "with an agent still in its call; a later check settles it"
	switch {
	case rec.HaltedAt != nil:
		state = "and has let its claims go"
	case heldBySave(&rec):
		state = "with work that could not be saved aside still in place, so it keeps its claims until the work is saved; the run record names it"
	}
	remedy := "the person's own `abcd implement stop --lift` lifts it; hand back any receipt you hold with `abcd implement receipt`, then stop"
	if !rec.FromFile {
		remedy = "it came from a signal, and `abcd implement resume --take " + runID + "` takes the run over once it has settled; hand back any receipt you hold with `abcd implement receipt`, then stop"
	}
	reason := fmt.Sprintf("a stop set by %s at %s holds: %s is halted %s, and nothing further starts",
		termsafe.Sanitize(rec.SetBy), rec.SetAt.UTC().Format(time.RFC3339), runID, state)
	if rec.PageFailed != "" {
		reason += "; " + termsafe.Sanitize(rec.PageFailed)
	}
	return contend(StageStop, "", "", reason, remedy)
}

// stopOverDriveStop is the refusal where both a stop and a drive's own stop stand on a
// run (drive_stop.go). They are two things, and neither answers for the other. Nothing
// is settled while both stand, since what the run says is not known to be the driver's:
// its work is not saved aside and its reservation is not released, which lapses with its
// lease. Lifting the stop leaves the drive's stop, and removing that leaves the stop, which
// the run then takes in its order.
func stopOverDriveStop(runID string, s runstop.Stop) error {
	return contend(StageStop, "", "", fmt.Sprintf("a stop set by %s at %s holds, and a drive stopped on %s because its state changed, so nothing is settled from what it now says",
		termsafe.Sanitize(s.SetBy), s.SetAt.UTC().Format(time.RFC3339), runID),
		"the person's own `abcd implement stop --lift` lifts the stop and leaves the other standing. "+driveStopRemedy(runID))
}

// haltedByPerson reports whether a person's stop halted the run, which then waits for an
// explicit take-over; a drive's own signal is not one.
func haltedByPerson(repoRoot, runID string) bool {
	c, err := runControl(repoRoot, runID)
	return err == nil && c.Stop != nil && c.Stop.FromFile && c.Stop.HaltedAt != nil
}

// waitsForTakeOver says what a run a person's stop halted waits for.
func waitsForTakeOver(runID string) string {
	return fmt.Sprintf("a person's stop halted run %s, which waits for `abcd implement resume --take %s`", termsafe.Sanitize(runID), termsafe.Sanitize(runID))
}

// waitsForItsTakeOver says what a run a stop halted waits for, "" for a run no stop halted;
// a record that cannot be read is said, and nothing is done on it.
func waitsForItsTakeOver(repoRoot, runID string) string {
	c, err := runControl(repoRoot, runID)
	switch {
	case err != nil:
		return fmt.Sprintf("the control record of run %s cannot be read, so nothing is done for it: %s", termsafe.Sanitize(runID), termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	case c.Stop == nil || c.Stop.HaltedAt == nil:
		return ""
	case c.Stop.FromFile:
		return waitsForTakeOver(runID)
	}
	return fmt.Sprintf("a signal to a drive halted run %s, which waits for `abcd implement resume --take %s`", termsafe.Sanitize(runID), termsafe.Sanitize(runID))
}

// stopMarkNote says why a stop's mark leaves nothing to be done for a run, "" for a run with
// none: what a halted run waits for, or what takes up one that has not settled. A record
// that cannot be read is said, and nothing is done on it.
func stopMarkNote(repoRoot, runID string) string {
	if note := waitsForItsTakeOver(repoRoot, runID); note != "" {
		return note
	}
	c, err := runControl(repoRoot, runID)
	if err != nil || c.Stop == nil {
		return ""
	}
	id := termsafe.Sanitize(runID)
	if !c.Stop.FromFile {
		// A signal has nothing to lift, so its mark is only ever taken over.
		return fmt.Sprintf("a signal to a drive marked run %s, which has not settled, so nothing is done for it: `abcd implement resume --take %s` takes it over", id, id)
	}
	return fmt.Sprintf("a stop set by %s marked run %s, which has not settled, so nothing is done for it: `abcd implement step --run %s` takes that up once the stop lifts",
		termsafe.Sanitize(c.Stop.SetBy), id, id)
}

// parkedStopNote is stopMarkNote for a drain's move over a parked run. A mark that has not
// settled first takes the check every move makes, so one from a stop lifted since goes, and
// the run is then as any other.
func parkedStopNote(repoRoot, runID string, o Options) (string, error) {
	c, err := runControl(repoRoot, runID)
	if err == nil && c.Stop != nil && c.Stop.HaltedAt == nil {
		if _, err := stopCheck(repoRoot, runID, DefaultStages(), o); err != nil {
			if _, refused := AsRefusal(err); !refused {
				return "", err
			}
		}
	}
	return stopMarkNote(repoRoot, runID), nil
}

// waitsUntaken says what a run no later drive takes up waits for: one a person's stop halted,
// or a parked one either kind of stop did. It is "" for any other.
func waitsUntaken(repoRoot string, run State) string {
	if run.Complete() || (!haltedByPerson(repoRoot, run.RunID) && parkedAtLand(run) < 0) {
		return ""
	}
	return waitsForItsTakeOver(repoRoot, run.RunID)
}

// haltRun applies a stop in its order, one read from the file or one that did not
// come from it: the caller names who set it. A later call takes up what an agent
// still in its call held back. The result's HaltedAt is set once every lane has
// settled; nil for a run with nothing left, which a stop has no claim on.
func haltRun(repoRoot, runID string, s runstop.Stop, steps Stages, o Options) (*RunStop, error) {
	now := o.now()
	var rec *RunStop
	var driver *LiveDriver
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		if st.Complete() {
			return false, nil
		}
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		driver = c.Driver
		if c.Stop != nil {
			rec = c.Stop
			return false, nil
		}
		c.Stop = &RunStop{SetBy: termsafe.Sanitize(s.SetBy), SetAt: s.SetAt, NotedAt: now, FromFile: s.Path != ""}
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		rec = c.Stop
		st.Record = append(st.Record, Entry{At: now, Stage: StageStop,
			Note: fmt.Sprintf("a stop set by %s at %s: no lane opens, no stage is handed out and no agent starts", c.Stop.SetBy, s.SetAt.UTC().Format(time.RFC3339))})
		st.UpdatedAt = now
		if configTripwireIn(repoRoot, root, runID) == nil {
			logStopEvent(repoRoot, driver, c.Stop.SetBy, s.SetAt)
		}
		return true, nil
	})
	if err != nil || rec == nil {
		return nil, err
	}
	if rec.HaltedAt != nil {
		if rec.FromFile && rec.PageFailed != "" {
			return notePage(repoRoot, runID, rec, s, o), nil
		}
		return rec, nil
	}
	tab, tabErr := readProcs()
	var seen *sessionname.Table
	if tabErr == nil {
		seen = &tab
	}
	// Collecting a receipt verifies it with git: a configuration changed during the
	// run holds that too, and settleStop says so.
	collected := map[string]string{}
	if configTripwire(repoRoot, runID) == nil {
		collected = collectArrived(repoRoot, runID, steps, seen, o)
	}
	err = mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		if c.Stop == nil {
			// A take-over cleared it between: the run is its driver's again.
			return false, nil
		}
		if c.Stop.HaltedAt != nil {
			// Another check halted it between: what that one let go stands.
			rec = c.Stop
			return false, nil
		}
		// A mark from a stop since lifted settles under the stop in force, which is the one
		// that halts it and the one whose page names it.
		if c.Stop.FromFile && s.Path != "" && !c.Stop.SetAt.Equal(s.SetAt) {
			c.Stop.SetBy, c.Stop.SetAt = termsafe.Sanitize(s.SetBy), s.SetAt
		}
		settleStop(repoRoot, root, st, &c, seen, collected, now)
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		rec = c.Stop
		st.UpdatedAt = now
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	// The page says the person's stop holds and how to lift it, which a signal is not.
	if rec.HaltedAt != nil && rec.FromFile {
		return notePage(repoRoot, runID, rec, s, o), nil
	}
	return rec, nil
}

// logStopEvent writes the stop to the shared run log where the driver's session
// joined one; the run's own record has it either way.
func logStopEvent(repoRoot string, driver *LiveDriver, setBy string, setAt time.Time) {
	if driver == nil || driver.Session == "" {
		return
	}
	sha, err := gitutil.RootCommit(repoRoot)
	if err != nil {
		return
	}
	run, err := implement.Peek(sha)
	if err != nil {
		return
	}
	_, _ = run.Log(driver.Session, implement.EventStop, map[string]string{
		"cause": fmt.Sprintf("a stop set by %s at %s", setBy, setAt.UTC().Format(time.RFC3339)),
	})
}

// AgentState is what the record can say of an agent a lane awaits.
type AgentState string

const (
	// AgentAlive is an agent a recorded launch's process, or its runner claim's
	// term, says is running.
	AgentAlive AgentState = "alive"
	// AgentLost is one whose claim's term has ended or whose driver is gone, or a
	// host's agent with no recorded driver older than one reservation lease.
	AgentLost AgentState = "lost"
	// AgentUnknown is one no process of its own is recorded for and nothing yet
	// says is gone, so it is taken to be in its call: it is never restarted, and a
	// stop waits for it.
	AgentUnknown AgentState = "unknown"
	// AgentPaused is a role waiting for its runner: no agent is out.
	AgentPaused AgentState = "paused"
)

// agentState judges one await. tab is nil when the process table could not be
// read, which leaves only the claim's term and the lease to go on.
func agentState(a Await, c control, tab *sessionname.Table, now time.Time) AgentState {
	switch {
	case a.Kept != "":
		return AgentLost
	case a.Paused != nil:
		return AgentPaused
	}
	if r := a.Running; r != nil {
		// A launch's process is the step that started it, and the agent leads a
		// process group of its own, so a step killed outright can leave the agent
		// running: its process gone says only "not known to be alive" until the
		// claim's term ends.
		if tab != nil {
			for _, l := range c.Launches {
				if l.Session == r.Session && l.PID > 0 && tab.Alive(l.host()) {
					return AgentAlive
				}
			}
		}
		if now.Before(r.Until) {
			if tab != nil && slices.ContainsFunc(c.Launches, func(l Launch) bool { return l.Session == r.Session && l.PID > 0 }) {
				return AgentUnknown
			}
			return AgentAlive
		}
		return AgentLost
	}
	if c.Driver != nil {
		// A host's agent lives in its driver's session: gone with it, and otherwise
		// taken to be in its call for as long as the driver lives or cannot be
		// judged. A driver no session names may be a program that has simply
		// exited, so its going says nothing and the lease decides.
		if tab == nil || tab.Alive(c.Driver.host()) {
			return AgentUnknown
		}
		if c.Driver.Session != "" {
			return AgentLost
		}
	}
	if now.Sub(a.Since) < lease.Default {
		return AgentUnknown
	}
	return AgentLost
}

// receiptArrived reports whether a file stands at the await's own receipt path.
func receiptArrived(repoRoot, runID, laneID string, root *os.Root, a Await) bool {
	if a.Receipt == "" || laneOwnReceipt(repoRoot, runID, laneID, a.Receipt) != nil {
		return false
	}
	fi, err := localtier.LstatInRoot(root, a.Receipt)
	return err == nil && fi.Mode().IsRegular()
}

// collectArrived hands back every receipt standing at its await's path whose
// agent no step still holds a running claim on: an agent's finished work is
// collected before anything else is done about its lane, never started again
// (iss-2611031337115764). Each is verified as any receipt is, so one that does
// not verify is named and left. It returns a note per lane.
func collectArrived(repoRoot, runID string, steps Stages, tab *sessionname.Table, o Options) map[string]string {
	notes := map[string]string{}
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return notes
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return notes
	}
	defer root.Close()
	c, err := readControl(root, runID)
	if err != nil {
		return notes
	}
	now := o.now()
	for _, lane := range st.Lanes {
		var said []string
		for _, a := range lane.Awaits {
			if !receiptArrived(repoRoot, runID, lane.ID, root, a) || agentState(a, c, tab, now) == AgentAlive {
				continue
			}
			// An implementer's receipt is taken as a step takes one where it stood, so
			// one that hands the lane back is left for a person, and its agent's own
			// hand-in later gets an answer. Validators' are handed in as ever.
			if _, err := handIn(repoRoot, runID, a.Receipt, steps, o, nil, a.Role == RoleImplementer); err != nil {
				said = append(said, fmt.Sprintf("the %s's receipt at %s did not verify: %s", a.Role, a.Receipt, fsutil.RedactHome(err.Error())))
				continue
			}
			said = append(said, "the "+a.Role+"'s receipt was collected")
		}
		if len(said) > 0 {
			notes[lane.ID] = termsafe.Sanitize(strings.Join(said, "; "))
		}
	}
	return notes
}

// stopFinished reports whether a stop has nothing to do for the lane.
func stopFinished(l Lane) bool {
	return l.Stage == StageDone || l.Stage == StageDiscarded || l.Stage == StageHandedBack
}

// settleStop takes each lane as far as its agents allow, under the run's lock.
// A lane with an agent live is left exactly as it stands. Once none is, its lost
// agents' awaits are marked kept, what is uncommitted is saved aside, and the
// worktree is reset as a restart resets it; a save that cannot be proved leaves
// the work where it is and says so. With every lane settled the reservation is
// released and the run is halted.
func settleStop(repoRoot string, root *os.Root, st *State, c *control, tab *sessionname.Table, collected map[string]string, now time.Time) {
	rec := c.Stop
	prior := map[string]StopLeft{}
	for _, l := range rec.Left {
		prior[l.Lane] = l
	}
	left := []StopLeft{}
	waiting := false
	why := stopKeptPrefix + rec.SetBy
	// Saving a lane runs git in its worktree, so a configuration changed during
	// the run holds every lane still to save, as it refuses a restart.
	tripped := configTripwireIn(repoRoot, root, st.RunID)
	for i := range st.Lanes {
		lane := st.Lanes[i]
		if stopFinished(lane) {
			continue
		}
		if p, ok := prior[lane.ID]; ok && p.Settled {
			left = append(left, p)
			continue
		}
		out := StopLeft{Lane: lane.ID, Stage: lane.Stage, Note: collected[lane.ID]}
		if lane.PR > 0 && lane.Landing != nil && lane.Landing.Armed {
			out.ArmedPR = lane.PR
		}
		live := slices.ContainsFunc(lane.Awaits, func(a Await) bool {
			s := agentState(a, *c, tab, now)
			return s == AgentAlive || s == AgentUnknown
		})
		if live {
			waiting = true
			out.Note = joinNext(out.Note, inItsCall)
			left = append(left, out)
			continue
		}
		if tripped != nil {
			waiting = true
			out.Note = joinNext(out.Note, "nothing was saved aside: "+termsafe.Sanitize(fsutil.RedactHome(tripped.Error())))
			left = append(left, out)
			continue
		}
		// A save that fails holds the lane, and so the claims, for a later check:
		// releasing would let a peer in over work saved nowhere.
		out.Settled = saveLaneForStop(repoRoot, root, st, i, why, &out, now)
		waiting = waiting || !out.Settled
		left = append(left, out)
		if out.Aside != "" {
			st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageStop,
				Note: fmt.Sprintf("%d uncommitted file(s) saved aside at %s", out.Files, out.Aside)})
		}
	}
	rec.Left = left
	// A release runs git too, so a changed configuration holds it.
	if waiting || tripped != nil {
		return
	}
	rec.Released = noReservationHeld
	if st.ReservedAt != nil {
		clause, failed := releaseReservation(repoRoot, st.Key, "", now)
		stage := StageReservationReleased
		if failed {
			clause += ", so it lapses with its lease"
			stage = StageReservationReleaseFailed
		}
		// Failed or not the run no longer holds it: a renewal now would keep the
		// claim the stop let go.
		st.ReservedAt = nil
		rec.Released = clause
		st.Record = append(st.Record, Entry{At: now, Stage: stage, Note: clause})
	}
	rec.HaltedAt = &now
	st.Record = append(st.Record, Entry{At: now, Stage: StageStop, Note: "halted: every lane has settled and the run's claims are let go"})
}

// saveLaneForStop is one lane with no agent live: each await marked kept with its
// partial return saved aside, and the lane's uncommitted work saved aside once.
func saveLaneForStop(repoRoot string, root *os.Root, st *State, i int, why string, out *StopLeft, now time.Time) (saved bool) {
	saved = true
	lane := st.Lanes[i]
	runID := st.RunID
	lane.Awaits = slices.Clone(lane.Awaits)
	worktreeSaved := false
	// A worktree that no longer stands holds nothing to save, and must not hold
	// the run's claims waiting for a save that cannot come.
	gone := laneWorktreeGone(repoRoot, runID, lane)
	if gone {
		out.Note = joinNext(out.Note, "the lane's worktree is gone, so it held nothing to save")
	}
	for k, a := range lane.Awaits {
		if a.Paused != nil || a.Kept != "" {
			continue
		}
		reset := a.Role == RoleImplementer && !gone
		p, err := proveAside(repoRoot, root, runID, lane, a, reset)
		if err == nil {
			var aside RestartAside
			if aside, err = saveAside(repoRoot, root, runID, lane.ID, a, p, why, now); err == nil {
				worktreeSaved = worktreeSaved || reset
				if len(aside.Files) > 0 {
					out.Aside, out.Files = aside.Path, len(aside.Files)
				}
			}
		}
		if err != nil {
			out.Note = joinNext(out.Note, "the "+a.Role+"'s leavings could not be saved aside and stay where they are: "+termsafe.Sanitize(fsutil.RedactHome(err.Error())))
			saved = false
			continue
		}
		a.Kept = why
		lane.Awaits[k] = a
		out.Note = joinNext(out.Note, "the "+a.Role+" is lost; a take-over restarts it")
	}
	st.Lanes[i] = lane
	if worktreeSaved || !saved || lane.Worktree == "" || gone {
		return saved
	}
	p := provedAside{reset: true, files: []string{}}
	err := proveLaneWork(repoRoot, runID, lane, &p)
	if err == nil && len(p.patch) > 0 {
		var aside RestartAside
		if aside, err = saveAside(repoRoot, root, runID, lane.ID, Await{}, p, why, now); err == nil {
			out.Aside, out.Files = aside.Path, len(aside.Files)
		}
	}
	if err != nil {
		out.Note = joinNext(out.Note, "the lane's uncommitted work could not be saved aside and stays where it is: "+termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return false
	}
	return true
}

// laneWorktreeGone reports that nothing stands where the loop derives the lane's
// worktree. Anything else there, a link included, is for the save to refuse.
func laneWorktreeGone(repoRoot, runID string, lane Lane) bool {
	if lane.Worktree == "" {
		return false
	}
	lw, err := derivedLaneWorktree(repoRoot, runID, lane, StageStop)
	if err != nil {
		return false
	}
	_, err = os.Lstat(lw.Path)
	return errors.Is(err, fs.ErrNotExist)
}

// stopEndsRenewal reports whether a stop has ended the run's right to renew its
// reservation: one is in force or the run took one, no agent may still be in its
// call, and no lane holds work a save could not take. A claim a stop is letting go
// is not kept alive; one with an agent in its call stays renewable until that agent
// is done, and one over work saved nowhere until the work is saved. That holds only
// where something still renews: under a stop in force a move and a step refuse before
// any renewal and a parked run is not renewed where it waits, so such a run's claim
// lapses at its lease, with the lane's work still on its branch and in its worktree.
func stopEndsRenewal(repoRoot string, st State, now time.Time) bool {
	c, err := runControl(repoRoot, st.RunID)
	if err != nil {
		return false
	}
	if c.Stop == nil && (Options{}).stopRead(c.Shared).Stop == nil {
		return false
	}
	var seen *sessionname.Table
	if tab, err := readProcs(); err == nil {
		seen = &tab
	}
	for _, lane := range st.Lanes {
		if stopFinished(lane) {
			continue
		}
		for _, a := range lane.Awaits {
			if s := agentState(a, c, seen, now); s == AgentAlive || s == AgentUnknown {
				return false
			}
		}
	}
	return !heldBySave(c.Stop)
}

// settlesHere reports whether this binary may settle st: it writes the state at its
// own schema, so a run at another stays with its driver's own check, and its lease
// lapses if that check does not come first.
func settlesHere(st State) bool { return st.SchemaVersion == SchemaVersion }

// notStartedUnderStop answers an await this very call handed out or re-told, which a
// stop refused before it began: no agent can have it yet, so it is marked as not
// started and takes the stop as one that was lost, which a take-over re-tells. It
// returns the refusal the run now gives. An older await, which a host's agent may be
// working on, never comes here.
func notStartedUnderStop(repoRoot, runID, receiptPath string, steps Stages, o Options, refused error) error {
	if rf, ok := AsRefusal(refused); !ok || rf.Stage != StageStop {
		return refused
	}
	err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Stop == nil {
			return false, err
		}
		i, k := st.findAwait(repoRoot, receiptPath)
		if i < 0 {
			return false, nil
		}
		a := st.Lanes[i].Awaits[k]
		if a.Running != nil || a.Paused != nil || a.Kept != "" {
			return false, nil
		}
		a.Kept = stopKeptPrefix + c.Stop.SetBy + ": it had not started"
		lane := st.Lanes[i]
		lane.Awaits = slices.Clone(lane.Awaits)
		lane.Awaits[k] = a
		st.Lanes[i] = lane
		st.UpdatedAt = o.now()
		return true, nil
	})
	if err != nil {
		return err
	}
	if _, again := stopCheck(repoRoot, runID, steps, o); again != nil {
		return again
	}
	return refused
}

// heldBySave reports whether a lane of the stop has work a save could not take, with
// no agent left to wait for.
func heldBySave(rec *RunStop) bool {
	if rec == nil || rec.HaltedAt != nil {
		return false
	}
	return slices.ContainsFunc(rec.Left, func(l StopLeft) bool {
		return !l.Settled && !strings.Contains(l.Note, inItsCall)
	})
}

// settleDueStop takes the run's stop when its renewal falls due with no one stepping
// it, so its work is saved before its claim goes. It reports whether one was in force.
func settleDueStop(repoRoot, runID string) bool {
	o := Options{}
	c, err := runControl(repoRoot, runID)
	if err != nil || (c.Stop == nil && o.stopRead(c.Shared).Stop == nil) {
		return false
	}
	_, _ = stopCheck(repoRoot, runID, DefaultStages(), o)
	return true
}

// composeStopPage rewrites this account's page for the day whole from the records: the
// stop, and every run this account can read that it halted, with what each left. Each
// halting rewrites it, so the last run to end leaves a page naming them all. Every value
// is masked as it is written, so none adds a line. It holds no account's name, which a
// page never does (itd-2611041549483696). Without a shared folder nothing is written; a
// page that cannot be written changes nothing about the stop, and the error says why.
func composeStopPage(repoRoot string, rec RunStop, o Options) error {
	home := o.stopHomeOf()
	if home == "" {
		return nil
	}
	now := o.now()
	var b strings.Builder
	armed := false
	for _, checkout := range machineCheckouts(repoRoot, home) {
		root, err := os.OpenRoot(checkout)
		if err != nil {
			continue
		}
		ids, err := runIDs(root)
		if err != nil {
			root.Close()
			continue
		}
		for _, id := range ids {
			c, err := readControl(root, id)
			if err != nil || c.Stop == nil || !c.Stop.FromFile || c.Stop.HaltedAt == nil || !c.Stop.SetAt.Equal(rec.SetAt) {
				continue
			}
			st, err := readListedState(root, id)
			if err != nil {
				continue
			}
			fmt.Fprintf(&b, "- %s (%s, in %s): halted, its claims let go\n", termsafe.Sanitize(st.RunID), termsafe.Sanitize(st.Key), termsafe.Sanitize(fsutil.RedactHome(checkout)))
			for _, l := range c.Stop.Left {
				fmt.Fprintf(&b, "  - %s at %s", termsafe.Sanitize(l.Lane), termsafe.Sanitize(string(l.Stage)))
				if l.Aside != "" {
					fmt.Fprintf(&b, ": %d file(s) saved aside at %s", l.Files, termsafe.Sanitize(l.Aside))
				}
				if l.Note != "" {
					fmt.Fprintf(&b, ": %s", termsafe.Sanitize(l.Note))
				}
				b.WriteString("\n")
				if l.ArmedPR > 0 {
					armed = true
					fmt.Fprintf(&b, "  - %s\n", ArmedLine(l.ArmedPR))
				}
			}
		}
		root.Close()
	}
	still := ""
	if armed {
		still = " A pull request armed before the stop can still merge, and each is named below."
	}
	page := fmt.Sprintf("# abcd, %s\n\n## Stop\n\nThe person's stop, set at %s, holds. Nothing further starts until they lift it.%s\n\n## Runs it halted\n\n%s",
		now.UTC().Format(time.DateOnly), rec.SetAt.UTC().Format(time.RFC3339), still, b.String())
	_, err := runstop.WritePage(home, now, []byte(page))
	return err
}

// ArmedLine says what a stop leaves of a pull request armed before it, which nothing disarms
// on its own.
func ArmedLine(pr int) string {
	return fmt.Sprintf("pull request #%d was armed before the stop and can still merge: disarm it with `gh pr merge %d --disable-auto`, and take it out of the merge queue on its page if it is there", pr, pr)
}

// notePage writes the page for a run the stop halted and keeps in the run's account whether
// it could, so the stop's answer can say; a later check tries again.
func notePage(repoRoot, runID string, rec *RunStop, s runstop.Stop, o Options) *RunStop {
	why := ""
	inForce := *rec
	if s.Path != "" {
		inForce.SetBy, inForce.SetAt = termsafe.Sanitize(s.SetBy), s.SetAt
	}
	if err := composeStopPage(repoRoot, inForce, o); err != nil {
		why = "the day's page could not be written: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	}
	if why == rec.PageFailed {
		return rec
	}
	now := o.now()
	_ = mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Stop == nil {
			return false, err
		}
		c.Stop.PageFailed = why
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		note := why
		if note == "" {
			note = "the day's page was written"
		}
		st.Record = append(st.Record, Entry{At: now, Stage: StageStop, Note: note})
		st.UpdatedAt = now
		return true, nil
	})
	out := *rec
	out.PageFailed = why
	return &out
}

// machineCheckouts is every checkout whose runs this account can read: this
// repository's worktrees and each checkout a driver noted, those that still
// stand, this one first. A lane's own worktree is none of them: what it holds is
// its agent's to write, and a run's state lives in the checkout that started it.
func machineCheckouts(repoRoot, home string) []string {
	out := []string{}
	if !isLaneWorktree(repoRoot) {
		out = append(out, repoRoot)
	}
	add := func(p string) {
		if p != "" && !isLaneWorktree(p) && fsutil.IsRealDir(p) && !slices.ContainsFunc(out, func(have string) bool { return samePath(repoRoot, have, p) }) {
			out = append(out, p)
		}
	}
	if wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing); err == nil {
		for _, wt := range wts {
			if !wt.Bare {
				add(wt.Path)
			}
		}
	}
	for _, p := range runCheckouts(home) {
		add(p)
	}
	return out
}

// drainStop is a drain's move under a stop: every lane's run takes the stop, and
// a judgement the drain awaits is let go with its hold, so the issue is a peer's
// to take. It refuses the move; the drain keeps its state and goes on when the
// stop lifts. The caller's deferred settle releases the hold st no longer names.
func drainStop(repoRoot string, st *DrainState, o Options) (string, error) {
	r := o.stopRead(nil)
	notes := []string{}
	note := func(line string) {
		if line != "" && !slices.Contains(notes, line) {
			notes = append(notes, line)
		}
	}
	note(readingNote(r))
	runs, err := Runs(repoRoot)
	if err != nil && r.Stop != nil {
		return "", err
	}
	// Each run reads the folder it started under, so a stop or a held read of any
	// one of them is the drain's too.
	for _, run := range runs {
		if run.Complete() {
			continue
		}
		c, err := runControl(repoRoot, run.RunID)
		if err != nil {
			return "", err
		}
		pinned := o.stopRead(c.Shared)
		note(readingNote(pinned))
		if r.Stop == nil {
			r.Stop = pinned.Stop
		}
		if r.Held == "" {
			r.Held = pinned.Held
		}
	}
	said := strings.Join(notes, "; ")
	if r.Stop == nil {
		if r.Held != "" {
			return said, heldRefusal(r.Held)
		}
		return said, nil
	}
	for _, run := range runs {
		if run.Complete() {
			continue
		}
		if driveStopped(run.RunID) {
			continue
		}
		if _, err := haltRun(repoRoot, run.RunID, *r.Stop, DefaultStages(), o); err != nil {
			if _, ok := AsRefusal(err); ok {
				continue
			}
			return said, err
		}
	}
	if st.Judging != nil {
		st.Judging = nil
		if err := writeDrain(repoRoot, *st); err != nil {
			return said, err
		}
	}
	return said, contend(StageStop, "", "", fmt.Sprintf("a stop set by %s at %s holds, so the drain opens no lane and asks for no judgement",
		termsafe.Sanitize(r.Stop.SetBy), r.Stop.SetAt.UTC().Format(time.RFC3339)),
		"the person's own `abcd implement stop --lift` lifts it")
}
