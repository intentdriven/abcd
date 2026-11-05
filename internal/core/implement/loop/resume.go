package loop

// resume.go is the take-over (itd-2611041549461012, spc-2611041706004822): when
// the session coordinating a run is gone, the next one reads what the run wrote
// down and steps on, and nothing is started twice. ListRuns is the reading: it
// writes nothing and uses no network. TakeRun acts on it, in the order that makes
// it safe: an arrived receipt is collected before anything else, an agent gone
// has its uncommitted work saved aside and named before its await is re-told, and
// a reservation a stop released is taken again before any of that lets a stage
// start. TakeRun itself starts no agent; that stays with the next step.

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageResume is the refusal stage and the run record's stage for a take-over.
const StageResume = "resume"

// whyDriverGone marks an await kept because its driver is gone, so a restart
// re-tells it.
const whyDriverGone = "its driver is gone"

// ListedAgent is one agent a lane awaits, as the record can judge it.
type ListedAgent struct {
	Role  string     `json:"role"`
	State AgentState `json:"state"`
	// Receipt is the path its receipt lands at, and ReceiptArrived whether a file
	// stands there uncollected.
	Receipt        string `json:"receipt"`
	ReceiptArrived bool   `json:"receipt_arrived"`
	// Route names the runner a recorded launch started it on; "" for the host's.
	Route string `json:"route,omitempty"`
}

// ListedLane is one lane with anything left.
type ListedLane struct {
	ID     string        `json:"id"`
	Stage  Stage         `json:"stage"`
	Branch string        `json:"branch,omitempty"`
	Agents []ListedAgent `json:"agents"`
	// PR is the lane's pull request as the state names it; 0 for none.
	PR int `json:"pr,omitempty"`
	// BaseMoved is how many commits origin's default branch, as last fetched, is
	// past the lane's base; nil when that cannot be told.
	BaseMoved *int `json:"base_moved,omitempty"`
	// Uncommitted is how many files the lane's worktree holds uncommitted; nil
	// when it was not read, with UnreadWhy saying why.
	Uncommitted *int   `json:"uncommitted,omitempty"`
	UnreadWhy   string `json:"unread_why,omitempty"`
	// Asides are the saved-aside directories of the lane, oldest first.
	Asides []string `json:"asides,omitempty"`
}

// ListedDriver is a run's recorded driver, with whether its process still runs.
type ListedDriver struct {
	Session string `json:"session,omitempty"`
	PID     int    `json:"pid"`
	// State is alive, gone, or unknown when the process table could not be read.
	State string    `json:"state"`
	At    time.Time `json:"at"`
}

// ListedRun is one run with anything left.
type ListedRun struct {
	Checkout string        `json:"checkout"`
	RunID    string        `json:"run_id"`
	Key      string        `json:"key"`
	Driver   *ListedDriver `json:"driver,omitempty"`
	// Reserved says the run holds its record's reservation as its state has it.
	Reserved bool `json:"reserved"`
	// Stop is what a stop did to the run, and left.
	Stop  *RunStop     `json:"stop,omitempty"`
	Lanes []ListedLane `json:"lanes"`
	// Next is the one move this run wants of whoever reads the listing.
	Next string `json:"next"`
}

// ListedDrain is a checkout's drain in progress.
type ListedDrain struct {
	Checkout string `json:"checkout"`
	// Judging is the issue a judgement is awaited over, "" for none.
	Judging string `json:"judging,omitempty"`
	Next    string `json:"next"`
}

// Listing is `abcd implement resume`: every run this account can read on the
// machine with anything left. The record is the whole of it, so a session that
// lost its context needs nothing else.
type Listing struct {
	Runs   []ListedRun   `json:"runs"`
	Drains []ListedDrain `json:"drains,omitempty"`
	// Stop is the stop in force; Refused names each file the read refused.
	Stop    *runstop.Stop `json:"stop,omitempty"`
	Refused []string      `json:"refused,omitempty"`
	// Unreadable names each run state that could not be read, and why.
	Unreadable []string `json:"unreadable,omitempty"`
	// Notices names each stop another account left, which is not honoured here.
	Notices []string `json:"notices,omitempty"`
}

// ListRuns reads every run this account can on the machine: this repository's
// worktrees and each checkout a driver noted. It changes nothing and uses no
// network. One naming run keeps to that run.
func ListRuns(repoRoot, only string, o Options) (Listing, error) {
	home := o.stopHomeOf()
	r := o.stopRead(nil)
	out := Listing{Runs: []ListedRun{}, Stop: r.Stop}
	for _, line := range r.Refused {
		out.Refused = append(out.Refused, termsafe.Sanitize(line))
	}
	notice := func(lines []string) {
		for _, line := range lines {
			if line = termsafe.Sanitize(line); !slices.Contains(out.Notices, line) {
				out.Notices = append(out.Notices, line)
			}
		}
	}
	notice(r.Foreign)
	var seen *sessionname.Table
	if tab, err := readProcs(); err == nil {
		seen = &tab
	}
	now := o.now()
	found := false
	for _, checkout := range machineCheckouts(repoRoot, home) {
		root, err := os.OpenRoot(checkout)
		if err != nil {
			continue
		}
		ids, err := runIDs(root)
		if err != nil {
			out.Unreadable = append(out.Unreadable, fmt.Sprintf("%s: %s", fsutil.RedactHome(checkout), termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
			root.Close()
			continue
		}
		for _, id := range ids {
			if only != "" && id != only {
				continue
			}
			found = found || id == only
			st, err := readListedState(root, id)
			if err != nil {
				out.Unreadable = append(out.Unreadable, fmt.Sprintf("%s in %s: %s", id, fsutil.RedactHome(checkout), termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
				continue
			}
			if st.Complete() {
				continue
			}
			c, _ := readControl(root, id)
			pinned := o.stopRead(c.Shared)
			notice(pinned.Foreign)
			if out.Stop == nil {
				out.Stop = pinned.Stop
			}
			out.Runs = append(out.Runs, listRun(checkout, o, root, st, seen, r.Stop != nil || pinned.Stop != nil, now))
		}
		root.Close()
		if only == "" {
			if d, live, err := readDrain(checkout); err == nil && live {
				out.Drains = append(out.Drains, listDrain(checkout, d))
			}
		}
	}
	if only != "" && !found {
		return Listing{}, refuse(StageResume, "", "", "no run "+termsafe.DescribeRefused(only)+" on this machine",
			"name a run `abcd implement resume` lists")
	}
	return out, nil
}

func listDrain(checkout string, d DrainState) ListedDrain {
	l := ListedDrain{Checkout: termsafe.Sanitize(fsutil.RedactHome(checkout)), Next: "run `abcd drain` in that checkout, which picks up from its state"}
	if d.Judging != nil {
		l.Judging = termsafe.Sanitize(d.Judging.Issue)
		l.Next = "give the judgement over " + l.Judging + " with `abcd drain --judgement`, in that checkout"
	}
	return l
}

// listRun is one run's whole account.
func listRun(checkout string, o Options, root *os.Root, st State, tab *sessionname.Table, stopHolds bool, now time.Time) ListedRun {
	out := ListedRun{Checkout: termsafe.Sanitize(fsutil.RedactHome(checkout)), RunID: termsafe.Sanitize(st.RunID), Key: termsafe.Sanitize(st.Key), Reserved: st.ReservedAt != nil, Lanes: []ListedLane{}}
	c, err := readControl(root, st.RunID)
	if err != nil {
		c = control{}
	}
	out.Stop = safeStop(c.Stop)
	held := o.stopRead(c.Shared).Held
	alive := false
	if d := c.Driver; d != nil {
		ld := &ListedDriver{Session: termsafe.Sanitize(d.Session), PID: d.PID, State: "unknown", At: d.At}
		if tab != nil {
			ld.State = "gone"
			if tab.Alive(d.host()) {
				ld.State, alive = "alive", true
			}
		}
		out.Driver = ld
	}
	view, _, _ := lastFetchedOrigin(checkout)
	for _, lane := range st.Lanes {
		if stopFinished(lane) {
			continue
		}
		out.Lanes = append(out.Lanes, listLane(checkout, root, st.RunID, lane, c, tab, view, now))
	}
	switch {
	case stopHolds:
		out.Next = "a stop holds: nothing moves until the person lifts it"
	case held != "":
		out.Next = termsafe.Sanitize(held) + ": nothing moves until the person puts it back"
	case alive && c.Stop != nil && c.Stop.HaltedAt != nil && c.Driver.At.After(*c.Stop.HaltedAt):
		out.Next = "a take-over from " + driverName(*c.Driver) + " holds it: if that take-over was refused, run `abcd implement resume --take " + out.RunID + "` again from that session, or end that session and take the run over from another, in " + out.Checkout
	case c.Stop != nil && c.Stop.HaltedAt != nil:
		out.Next = "a stop halted it, every lane has settled, and no agent is in its call: take it over with `abcd implement resume --take " + out.RunID + "`, in " + out.Checkout
	case alive:
		out.Next = "its driver is alive: leave it"
	default:
		out.Next = "take it over with `abcd implement resume --take " + out.RunID + "`, in " + out.Checkout
	}
	return out
}

// safeStop is a stop's account with what it was read with made safe to write out.
func safeStop(s *RunStop) *RunStop {
	if s == nil {
		return nil
	}
	out := *s
	out.SetBy, out.Released = termsafe.Sanitize(s.SetBy), termsafe.Sanitize(s.Released)
	out.Left = slices.Clone(s.Left)
	for i, l := range out.Left {
		out.Left[i].Lane, out.Left[i].Stage = termsafe.Sanitize(l.Lane), Stage(termsafe.Sanitize(string(l.Stage)))
		out.Left[i].Note, out.Left[i].Aside = termsafe.Sanitize(l.Note), termsafe.Sanitize(l.Aside)
	}
	return &out
}

func listLane(checkout string, root *os.Root, runID string, lane Lane, c control, tab *sessionname.Table, view originView, now time.Time) ListedLane {
	out := ListedLane{ID: termsafe.Sanitize(lane.ID), Stage: Stage(termsafe.Sanitize(string(lane.Stage))), Branch: termsafe.Sanitize(lane.Branch), PR: lane.PR, Agents: []ListedAgent{}}
	for _, a := range lane.Awaits {
		la := ListedAgent{Role: termsafe.Sanitize(a.Role), State: agentState(a, c, tab, now), Receipt: termsafe.Sanitize(a.Receipt), ReceiptArrived: receiptArrived(checkout, runID, lane.ID, root, a)}
		if r := a.Running; r != nil {
			la.Route = termsafe.Sanitize(r.Runner)
		}
		out.Agents = append(out.Agents, la)
	}
	if view.commit != "" && gitutil.IsFullSHA(lane.BaseSHA) {
		if n, err := gitutil.Run(checkout, "rev-list", "--count", lane.BaseSHA+".."+view.commit, "--"); err == nil {
			if moved, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
				out.BaseMoved = &moved
			}
		}
	}
	if lane.Worktree != "" {
		lw, err := derivedLaneWorktree(checkout, runID, lane, StageResume)
		switch {
		case err != nil:
			out.UnreadWhy = "the state names a worktree the loop did not derive, so it is not read"
		case !fsutil.IsRealDir(lw.Path):
			out.UnreadWhy = "the lane's worktree is gone"
		default:
			s, err := gitutil.Run(lw.Path, "--no-optional-locks", "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all")
			if err != nil {
				out.UnreadWhy = "git could not read the lane's worktree"
				break
			}
			n := 0
			for _, rec := range strings.Split(s, "\x00") {
				if rec != "" {
					n++
				}
			}
			out.Uncommitted = &n
		}
	}
	if dir, err := laneRel(runID, lane.ID, stageRestart); err == nil {
		if f, err := root.Open(dir + "/" + AsideDirName); err == nil {
			names, _ := f.Readdirnames(-1)
			f.Close()
			slices.Sort(names)
			for _, n := range names {
				out.Asides = append(out.Asides, termsafe.Sanitize(path.Join(dir, AsideDirName, n)))
			}
		}
	}
	return out
}

// TakeResult is what a take-over did.
type TakeResult struct {
	RunID string `json:"run_id"`
	// Collected are the receipts it handed back, by lane.
	Collected map[string]string `json:"collected,omitempty"`
	// Saved are the lanes with no agent whose uncommitted work it saved aside.
	Saved []RestartAside `json:"saved,omitempty"`
	// Restarted are the lanes whose lost agents it re-told, with the aside each
	// saved; each await named there wants a fresh agent.
	Restarted []RestartResult `json:"restarted,omitempty"`
	// Left says what it could not do, and why.
	Left []string `json:"left,omitempty"`
	// Reservation is what became of a reservation a stop released.
	Reservation string `json:"reservation,omitempty"`
	// Run is the run as it now stands.
	Run  *ListedRun `json:"run,omitempty"`
	Next string     `json:"next"`
}

// safeTake is a take-over's answer with what the run file named made safe to write
// out: a state's lane ids and roles are not validated when it is read.
func safeTake(res TakeResult) TakeResult {
	safe := func(line string) string { return termsafe.Sanitize(line) }
	res.RunID, res.Reservation, res.Next = safe(res.RunID), safe(res.Reservation), safe(res.Next)
	if res.Collected != nil {
		collected := make(map[string]string, len(res.Collected))
		for lane, note := range res.Collected {
			collected[safe(lane)] = safe(note)
		}
		res.Collected = collected
	}
	res.Left = slices.Clone(res.Left)
	for i, l := range res.Left {
		res.Left[i] = safe(l)
	}
	aside := func(a RestartAside) RestartAside {
		a.RunID, a.Lane, a.Path, a.Why, a.Receipt = safe(a.RunID), safe(a.Lane), safe(a.Path), safe(a.Why), safe(a.Receipt)
		a.Files = slices.Clone(a.Files)
		for i, f := range a.Files {
			a.Files[i] = safe(f)
		}
		return a
	}
	res.Saved = slices.Clone(res.Saved)
	for i, a := range res.Saved {
		res.Saved[i] = aside(a)
	}
	res.Restarted = slices.Clone(res.Restarted)
	for i, r := range res.Restarted {
		r.Lane, r.Next, r.Aside = safe(r.Lane), safe(r.Next), aside(r.Aside)
		res.Restarted[i] = r
	}
	return res
}

// takeFailed is the refusal of a take-over that left work saved nowhere: its reason goes
// out as written, under --json too, and a state's lane ids are not validated when read.
func takeFailed(runID string, failed []string) error {
	return refuse(StageResume, "", "", termsafe.Sanitize(strings.Join(failed, "; ")),
		"settle what the reason names, then run `abcd implement resume --take "+termsafe.Sanitize(runID)+"` again; do not step the run")
}

// tookFirst says, on a refusal after the take-over began, what it did first.
func tookFirst(err error, res TakeResult) error {
	rf, ok := AsRefusal(err)
	if !ok {
		return err
	}
	said := "it claimed the driver"
	lanes := make([]string, 0, len(res.Collected))
	for lane := range res.Collected {
		lanes = append(lanes, lane)
	}
	slices.Sort(lanes)
	for _, lane := range lanes {
		said += "; " + lane + ": " + res.Collected[lane]
	}
	for _, a := range res.Saved {
		said += fmt.Sprintf("; %s: %d file(s) saved aside at %s", a.Lane, len(a.Files), a.Path)
	}
	for _, r := range res.Restarted {
		said += "; " + r.Lane + " was restarted"
	}
	out := *rf
	out.Wrote = termsafe.Sanitize(said)
	return &out
}

// finishedKey is the record the run's state names, "" when it cannot be read.
func finishedKey(repoRoot, runID string) string {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return ""
	}
	return st.Key
}

// TakeRun takes a run over. It is refused, changing nothing, while a stop holds
// and while another driver's process is alive; and, having collected what had
// already arrived and changing nothing else, when a reservation a stop released
// cannot be taken again.
func TakeRun(repoRoot, runID string, o Options) (TakeResult, error) {
	res, err := takeRun(repoRoot, runID, o)
	return safeTake(res), err
}

func takeRun(repoRoot, runID string, o Options) (TakeResult, error) {
	mine, err := runControl(repoRoot, runID)
	if err != nil {
		return TakeResult{}, err
	}
	if r := o.stopRead(mine.Shared); r.Stop != nil {
		return TakeResult{}, contend(StageStop, "", "", fmt.Sprintf("a stop set by %s at %s holds, so nothing is taken over",
			termsafe.Sanitize(r.Stop.SetBy), r.Stop.SetAt.UTC().Format(time.RFC3339)),
			"the person's own `abcd implement stop --lift` lifts it")
	} else if r.Held != "" {
		return TakeResult{}, heldRefusal(r.Held)
	} else if len(r.Refused) > 0 {
		// Only a read that saw clearly and found no stop lets a run be taken over.
		return TakeResult{}, heldRefusal(strings.Join(r.Refused, "; "))
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return TakeResult{}, err
	}
	defer root.Close()
	if _, err := ReadState(repoRoot, runID); err != nil {
		return TakeResult{}, err
	}
	// Everything below may run git, so a configuration changed during the run
	// refuses the take-over before it does anything.
	if err := configTripwire(repoRoot, runID); err != nil {
		return TakeResult{}, err
	}
	// The record as the gone driver left it: what is lost is judged by that, not
	// by the driver about to be claimed.
	prev, err := readControl(root, runID)
	if err != nil {
		return TakeResult{}, err
	}
	var seen *sessionname.Table
	if tab, err := readProcs(); err == nil {
		seen = &tab
	}
	if seen == nil && prev.Driver != nil {
		// A take-over resets worktrees: a driver that cannot be judged is not
		// taken for gone.
		return TakeResult{}, contend(StageDriver, "", "", fmt.Sprintf("the process table cannot be read, so whether %s still drives %s cannot be told, and nothing was changed",
			driverName(*prev.Driver), runID), "run `abcd implement resume --take "+runID+"` again")
	}
	if err := claimDriver(repoRoot, runID, o, true); err != nil {
		return TakeResult{}, err
	}
	res := TakeResult{RunID: runID}
	res.Collected = collectArrived(repoRoot, runID, DefaultStages(), seen, o)

	if prev.Stop != nil && prev.Stop.HaltedAt != nil && prev.Stop.Released != noReservationHeld {
		start := StartResult{RunID: runID}
		switch err := reserveResumed(repoRoot, &start, o); {
		case err == nil:
			res.Reservation = start.Reservation
		case claim.Finished(repoRoot, finishedKey(repoRoot, runID)):
			// An armed pull request merged after the stop let the claim go: there
			// is nothing to reserve, and the run ends as it would have.
			res.Reservation = "the record has finished on the default branch, so there is nothing to reserve"
		default:
			return res, tookFirst(err, res)
		}
	}
	if prev.Stop != nil {
		if err := clearStop(repoRoot, runID, "taken over after the stop lifted", o); err != nil {
			return res, tookFirst(err, res)
		}
	}

	now := o.now()
	var lost, failed []string
	err = mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		changed := false
		// Saving a lane runs git in its worktree: the run tripwire is checked
		// first, as a restart checks it.
		tripped := configTripwireIn(repoRoot, root, runID)
		for i := range st.Lanes {
			lane := st.Lanes[i]
			if stopFinished(lane) {
				continue
			}
			if len(lane.Awaits) == 0 {
				if lane.Worktree == "" {
					continue
				}
				if laneWorktreeGone(repoRoot, runID, lane) {
					res.Left = append(res.Left, lane.ID+"'s worktree is gone, so it held nothing to save")
					continue
				}
				if tripped != nil {
					failed = append(failed, fmt.Sprintf("%s's uncommitted work was not read: %s", lane.ID, termsafe.Sanitize(fsutil.RedactHome(tripped.Error()))))
					continue
				}
				p := provedAside{reset: true, files: []string{}}
				if err := proveLaneWork(repoRoot, runID, lane, &p); err != nil {
					failed = append(failed, fmt.Sprintf("%s's uncommitted work could not be read: %s", lane.ID, termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
					continue
				}
				if len(p.patch) == 0 {
					continue
				}
				aside, err := saveAside(repoRoot, root, runID, lane.ID, Await{}, p, "taken over: "+WhyDied, now)
				if err != nil {
					failed = append(failed, fmt.Sprintf("%s's uncommitted work could not be saved aside: %s", lane.ID, termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
					continue
				}
				res.Saved = append(res.Saved, aside)
				st.Record = append(st.Record, Entry{At: now, Lane: lane.ID, Stage: StageResume,
					Note: fmt.Sprintf("%d uncommitted file(s) a gone agent left saved aside at %s", len(aside.Files), aside.Path)})
				changed = true
				continue
			}
			any := false
			lane.Awaits = slices.Clone(lane.Awaits)
			for k, a := range lane.Awaits {
				switch agentState(a, prev, seen, now) {
				case AgentLost:
					any = true
					if a.Role != RoleImplementer && a.Kept == "" {
						a.Kept = whyDriverGone
						lane.Awaits[k] = a
						changed = true
					}
				case AgentUnknown:
					res.Left = append(res.Left, fmt.Sprintf("%s's %s may still be in its call, so it is not restarted: run `abcd implement step --restart %s` once it is known to be gone", lane.ID, a.Role, lane.ID))
				}
			}
			st.Lanes[i] = lane
			if any {
				lost = append(lost, lane.ID)
			}
		}
		if changed {
			st.UpdatedAt = now
		}
		return changed, nil
	})
	if err != nil {
		return res, tookFirst(err, res)
	}
	for _, laneID := range lost {
		r, err := restartLane(repoRoot, runID, laneID, "", o)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s was not restarted: %s", laneID, termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
			continue
		}
		res.Restarted = append(res.Restarted, r)
	}
	// Work saved nowhere is never stepped over: the take-over says so rather than
	// naming a step that would build on it.
	if len(failed) > 0 {
		return res, tookFirst(takeFailed(runID, failed), res)
	}
	res.Next = "run `abcd implement step --run " + runID + "`"
	if listed, err := ListRuns(repoRoot, runID, o); err == nil && len(listed.Runs) == 1 {
		res.Run = &listed.Runs[0]
	}
	return res, nil
}
