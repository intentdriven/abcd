package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/cienv"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The waiter (`abcd implement load --wait`, iss-2610111826193257) is the one
// wait-for-capacity primitive concurrent sessions use instead of writing their
// own, and `make preflight` runs its whole run through its command form, so a
// queued preflight does no work until its turn and the turn holds until the
// run ends (itd-2610262127138979). The slot count is read at every look
// (PreflightSlots), so a change to it reaches the waiters already queued. It
// blocks until the one-minute load average is strictly below the machine's wait limit
// (wait-load in ~/.abcd.noindex/load-limits, twice the online core count by
// default), no other `go test -race` run is live, no other `make preflight` is
// running and no waiter is queued ahead of it or running its command, every one
// judged from the process table and each candidate's argv, under any account,
// with the waiter's own process and ancestors excluded (machineload.RaceRuns,
// machineload.QueueAhead). It never signals anything.
//
// That is the first gate with one slot, the default. How many preflights the
// first gate admits at once, each as one whole run, is the person's own
// setting (PreflightSlots, load_slots.go); above one, the first gate opens
// while fewer than that many other units of work are running past it
// (machineload.Ahead.Shut), a waiter queued ahead still shuts it, a live race
// run does not shut it by itself, and a preflight queued at its race gate,
// running nothing, takes no slot until its race run starts
// (machineload.Ahead.TakingSlots, iss-2610282016250805). The race gate
// (WaitRequest.Race, `--race-gate`), which `make preflight` runs before its
// race stage, opens at once with one slot, so a preflight's race stage
// follows its tests exactly as before; above one it is shut only by another
// race run, a race waiter ahead (machineload.RaceAhead), the load and the
// race gate's own hold. That keeps race stages apart among the preflights
// whose checkouts have the race gate (a checkout older than the gate runs its
// race stage ungated), within one account by the hold, and across accounts
// only by the process table, the jitter and the re-check.
//
// The process table is the cross-account half (iss-2610140638074587): it
// shows every account's processes: no file another account can write is read;
// the gate reads only the process table (and the caller's own load-limits and
// hold). Another account's processes and their arguments are read, and that
// account wrote them, which is why a waiter ahead can delay this one. A
// waiter queued ahead still delays the ones behind it, under any account, until
// it opens or reaches its own timeout, even when only its own account's
// stricter wait-load keeps it shut: it cannot yield its place, because the
// others cannot see why it is shut, and telling them would need the shared
// writable file this design rules out. A waiter running its command holds every
// later gate for the command's length, by design. A look that finds the
// machine free does not open the gate: the waiter waits a random 15 to 60
// seconds and checks everything again, and opens only if that re-check also
// finds it free, so two starters released by one quiet minute do not start
// together (iss-2610102020442624).
//
// The queue is ordered by when each waiter started, whatever its pid
// (machineload.QueueAhead, iss-2610222125175640). A waiter queued ahead by
// start with the higher pid, as after the process ids wrap, is crossed: one
// built before the start order (v0.15.0 and earlier) waits on every lower
// pid, so it may be waiting on this waiter while this one waits on it, and
// nothing in the table tells it from one that is not. So a wait stops
// counting crossed waiters once they alone have held its gate shut for
// WaitCrossedCap on end, a step more for each crossed waiter ahead beyond the
// first (crossedCapFor), and opens past them; anything else that shuts the
// gate in between, a crossed waiter running its command among them, and a
// stand-back at the open look, start the count again. No back-off breaks that
// cycle: WaitBackoffCap bounds only the open look, which a queued wait never
// reaches.
//
// Across accounts no hold keeps two waiters apart: re-checks that fall in one
// moment, before either shows in the table as running its command, both open.
// Under `--wait -- make preflight` each preflight's own gate then waits on the
// other's running waiter, and both stand until their timeouts. So the command
// form shows itself in the table as running from the moment it opens (a
// short-lived child, its mark), waits WaitOpenPause and reads the table once
// more before it starts its command; a waiter that finds another opened
// within that moment stands back from it unless it precedes it by start and
// by pid alike (machineload.StandBackFrom): it gives up its hold, re-draws
// its jitter and waits again, while the other goes on. So the later start
// stands back from the earlier whatever their pids, and the earlier from a
// lower pid too, which an older abcd (v0.15.1 and earlier) goes on past. It
// stands back at most WaitBackoffCap times. It stands back only from a waiter
// at its own gate (machineload.JustOpened), and at the first gate whatever the
// slots: above one, the other keeps going and this one's next look counts it,
// and its mark, among the units running past the gate (machineload.Ahead.Shut).
// With one slot the race gate opens before any look, so it never marks or
// stands back.
//
// A falling load opens every waiter watching it at once, which on 2026-10-11
// put four race runs on a 16-core machine together. So within one account the
// waiter that opens also takes a hold, a record under ~/.abcd.noindex/ written
// under a flock, one record for each gate, and every other waiter of that
// account at the same gate stays shut while the hold is in force (a hold
// whose holder is the waiter's own ancestor, as when `--wait -- make
// preflight` runs a preflight that gates itself, is not; that inner waiter
// also takes its ancestor's place in the process-table queue,
// machineload.QueueAhead):
//
//   - the plain form (`--wait` alone) holds for WaitSettle after opening, long
//     enough for the one-minute load average to show the work it admitted. Its
//     holder exits right after opening, by design, so its life is not asked.
//   - the command form (`--wait -- <command>`) holds while its holder (the
//     waiter, which runs the command) is alive as the same process (pid and
//     start time), until the command has started its own race run, the
//     command exits, or WaitCommandCap passes, whichever comes first. A holder
//     that has died, or whose pid now belongs to another process, is not
//     trusted, and its hold is reclaimed by the next waiter that looks. Above
//     one slot a command that is not a race run takes the plain form's settle
//     hold instead, so the account's next slot opens once the load shows the
//     work, as it did when a preflight's own plain gate admitted its tests.

// The waiter's statuses.
const (
	// WaitOpen: every condition held open and the hold was taken.
	WaitOpen = "open"
	// WaitTimedOut: the timeout passed with a condition still shut.
	WaitTimedOut = "timeout"
	// WaitSkipped: a CI runner, where there is no one else's work to wait for.
	WaitSkipped = "skipped"
	// WaitUnchecked: a platform whose load and processes abcd cannot read; the
	// waiter opens without judging, and says so.
	WaitUnchecked = "unchecked"
)

// The conditions that hold the gate shut.
const (
	// WaitOnLoad: the one-minute load average is not below the wait limit.
	WaitOnLoad = "load"
	// WaitOnRaceRun: another go test -race run is live (at the first gate,
	// only while one preflight runs at a time).
	WaitOnRaceRun = "race-run"
	// WaitOnLock: another waiter's hold is in force.
	WaitOnLock = "lock"
	// WaitOnUnreadable: the load or the process table could not be read.
	WaitOnUnreadable = "unreadable"
	// WaitOnPreflight: another make preflight, of any account, is running (with
	// more than one slot: every slot is taken, by at least one preflight).
	WaitOnPreflight = "preflight"
	// WaitOnWaiter: another waiter, of any account, is queued ahead of this one
	// or running the command it gates.
	WaitOnWaiter = "waiter"
)

// The two hold forms.
const (
	HoldSettle  = "settle"
	HoldCommand = "command"
)

// The waiter's intervals. They are variables only so the front door can read
// them into its help; nothing reassigns them.
var (
	// WaitDefaultTimeout is the front door's --timeout default.
	WaitDefaultTimeout = 30 * time.Minute
	// WaitPollInterval is how often the machine is read while waiting.
	WaitPollInterval = 15 * time.Second
	// WaitSettle is how long the plain form's hold lasts.
	WaitSettle = 2 * time.Minute
	// WaitCommandCap is the longest the command form's hold lasts. It only
	// bounds a hold nothing else releases: the command starting its own race
	// run, the command exiting, or its holder dying or its pid being reused
	// each end the hold first. Twenty minutes was too short: a preflight on a
	// loaded machine takes longer than that to reach its race step, so the hold
	// lapsed while its holder was alive and a second preflight was admitted.
	WaitCommandCap = 60 * time.Minute
	// WaitJitterMin and WaitJitterMax bound the random wait between a look
	// that finds the machine free and the re-check that opens the gate.
	WaitJitterMin = 15 * time.Second
	WaitJitterMax = 60 * time.Second
	// WaitOpenPause is how long the command form waits after it opens, showing
	// in the process table as running its command, before it looks once more
	// for another waiter that opened in the same moment (WaitLoad).
	WaitOpenPause = 2 * time.Second
)

// WaitBackoffCap bounds how many times one wait stands back from another
// waiter that opened in the same moment (machineload.StandBackFrom). A waiter
// that has stood back this many times opens without that last look, as it did
// before the look was added, so no wait goes round for ever.
const WaitBackoffCap = 3

// WaitCrossedCap is how long crossed waiters (machineload.Ahead.Crossed),
// queued ahead by start with a higher pid, may alone hold a wait's gate shut
// on end before it stops counting them. One of them may be a waiter built
// before the start order, which waits on every lower pid and so on this one,
// and each would wait on the other until its timeout. A waiter of this code
// or v0.15.1's waits on no later start, so another account sees it queued
// while the machine is free only for what its own account alone sees: its
// account's settle hold (WaitSettle), the poll after it, its jitter and its
// open look, which the cap outlasts. One held shut longer by its own account's
// stricter wait limit is passed, as v0.15.1 passes every crossed waiter.
var WaitCrossedCap = 5 * time.Minute

// crossedLook is what one look read of the crossed waiters, when they alone
// held the gate shut (whether or not they were counted): n of them queued
// ahead, every waiter queued ahead then being crossed, the crossed pids, and
// every waiter queued at the gate, ahead or not. The zero value is a look at
// which they did not.
type crossedLook struct {
	n       int
	crossed []int
	queued  []int
}

// crossedCapFor is the cap for a wait with n crossed waiters queued ahead of
// it: WaitCrossedCap, and one stagger step more for each beyond the first.
// Each waiter judges the cap from its own looks, so with one cap for all, two
// waiters of this code queued behind the same pid-ordered waiter, the later of
// them crossed for the earlier too (after a wrap the later start can hold the
// lower pid), would stop counting crossed waiters in the same few seconds, and
// the later, re-checking first, would overtake the earlier. The later start
// counts every crossed waiter the earlier counts and the earlier one as well,
// so its n is greater, and its cap ends a step after the earlier's. A step
// outlasts what the earlier's open can take as the later sees it: its first
// look past its cap up to a poll after the cap, its jitter, its open look, and
// a further poll for the two waiters' runs of looks starting up to a poll
// apart; the later's own re-check waits at least WaitJitterMin more. By then
// the earlier shows running, which ends the later's run of looks.
//
// The step does not budget slow reads of the process table. A look whose ps
// run takes 8 to 12 seconds on a loaded machine delays the earlier waiter's
// open by that much, and two or more such looks on the earlier waiter's side
// can use up the step's margin, so the later one can still overtake; the open
// look's stand-back (machineload.StandBackFrom) is then the backstop for the
// command form.
func crossedCapFor(n int) time.Duration {
	step := 2*WaitPollInterval + WaitJitterMax + WaitOpenPause
	return WaitCrossedCap + time.Duration(max(n-1, 0))*step
}

// waitJitter is a random wait from WaitJitterMin to WaitJitterMax.
func waitJitter() time.Duration {
	return WaitJitterMin + rand.N(WaitJitterMax-WaitJitterMin+1)
}

// WaitHoldFileName is the first gate's hold record under ~/.abcd.noindex/,
// and WaitRaceHoldFileName the race gate's. Each gate judges and takes only
// its own (holdFileName); the first gate's keeps the name it had before the
// race gate existed, so an abcd released before it reads the same record.
const (
	WaitHoldFileName     = "load-wait.json"
	WaitRaceHoldFileName = "load-wait-race.json"
)

// holdGates are the gates whose hold a wait judges and takes: its own, and
// for a race command's waiter above one slot both, since its command is a
// race run (raceCommandAtBothGates).
func holdGates(req WaitRequest, res *WaitResult) []string {
	if raceCommandAtBothGates(req, res) {
		return []string{GateTests, GateRace}
	}
	return []string{res.Gate}
}

// raceCommandAtBothGates reports whether the wait is the command form of a
// race run at the first gate (`--wait -- go test -race`) with more than one
// slot, where a race run does not shut the first gate by itself. Its command
// is a race run, so it is judged as one for both gates: a live race run, or
// another race command's waiter running its command, shuts it, and it judges
// and takes the race gate's hold as well as its own; the race gate counts it
// as a race run from the moment it opens (machineload.RaceCommandWaiters).
// With one slot the first gate is already shut by every race run and every
// waiter running its command, and the race gate opens at once, so nothing
// changes there.
func raceCommandAtBothGates(req WaitRequest, res *WaitResult) bool {
	return req.RaceCommand && req.Command && !req.Race && res.Slots > 1
}

// HoldFileName is the hold record of gate, GateTests or GateRace, as the
// front door names it in a message.
func HoldFileName(gate string) string { return holdFileName(gate) }

// holdFileName is gate's hold record.
func holdFileName(gate string) string {
	if gate == GateRace {
		return WaitRaceHoldFileName
	}
	return WaitHoldFileName
}

// waitLockFileName is the flock every waiter takes to judge and write the hold.
const waitLockFileName = ".load-wait.lock"

// waitLockTimeout bounds one poll's wait for the flock; another waiter holds it
// only while it reads and writes one small record.
const waitLockTimeout = 10 * time.Second

// holdStartSlack is how far a holder's start time, as the process table gives it
// now, may differ from the one its record carries and still be the same
// process: macOS reports a process's age to the second.
const holdStartSlack = 3

// maxHoldBytes caps the hold record.
const maxHoldBytes = 4 << 10

// WaitRequest is one wait. Every field has a working zero value but Timeout,
// whose zero is a single look; the seams exist so a test never depends on the
// real machine, clock, environment or home.
type WaitRequest struct {
	// Getenv reads the environment; nil is os.Getenv.
	Getenv func(string) string
	// Read reads the machine; nil is machineload.Read.
	Read func() (machineload.Snapshot, error)
	// ReadArgs reads the go processes' arguments; nil is machineload.ReadArgs.
	ReadArgs func([]int) (map[int][]string, error)
	// Home is the caller's home; "" resolves it.
	Home string
	// Self is the waiter; the zero value is this process.
	Self machineload.Self
	// Now and Sleep are the clock; nil is the real one.
	Now   func() time.Time
	Sleep func(time.Duration)
	// Timeout is how long to wait; zero looks once.
	Timeout time.Duration
	// Poll is the interval between looks; zero is WaitPollInterval.
	Poll time.Duration
	// Command asks for the command form's hold.
	Command bool
	// Race waits at the race gate (`--race-gate`), before a race stage,
	// instead of the first gate. With one slot (PreflightSlots) it opens at
	// once, without a look, a hold or a jitter; above one it is shut only by
	// another go test -race run, a race waiter ahead of it, the load and the
	// race gate's own hold.
	Race bool
	// RaceCommand says the command form's command is a race run
	// (machineload.IsRaceCommand).
	RaceCommand bool
	// OnWait is called once, at the first look that finds the gate shut.
	OnWait func(WaitResult)
	// OnSettle is called once, at the first look that finds the machine free,
	// with the jitter the waiter waits before it checks again.
	OnSettle func(time.Duration)
	// Jitter is the wait between a look that finds the machine free and the
	// re-check that opens the gate; nil is a random WaitJitterMin to
	// WaitJitterMax.
	Jitter func() time.Duration
	// Mark shows the command form's waiter in the process table as running its
	// command from the moment it opens until its command starts: it starts a
	// child of the waiter and returns the function that stops it. nil is a
	// short-lived /bin/sleep.
	Mark func() (stop func(), err error)
	// OnBackoff is called at each back-off, with the waiters that opened in the
	// same moment that this one stood back from (machineload.StandBackFrom) and
	// the jitter waited before looking again.
	OnBackoff func(first []int, jitter time.Duration)
}

// WaitHolder is a hold: the record under ~/.abcd.noindex/ and what the report
// carries.
type WaitHolder struct {
	PID int `json:"pid"`
	// StartUnix is the holder's start time, in Unix seconds; zero when it could
	// not be read.
	StartUnix int64     `json:"start_unix"`
	Opened    time.Time `json:"opened"`
	Until     time.Time `json:"until"`
	Form      string    `json:"form"`
	// Gate is the gate whose record this is, GateTests or GateRace; a
	// record without one, as an abcd released before the race gate writes
	// it, is the first gate's.
	Gate string `json:"gate,omitempty"`
}

// WaitResult is the waiter's report.
type WaitResult struct {
	Status  string  `json:"status"`
	WaitedS float64 `json:"waited_s"`
	// WaitedOn is every condition that held the gate shut at some look, in the
	// order first seen; empty is opened at the first look.
	WaitedOn []string `json:"waited_on"`
	// Still is what held the gate shut at the last look, when it is shut.
	Still []string `json:"still,omitempty"`
	Polls int      `json:"polls"`
	// Load, WaitLoad and Cores are the last reading and the limit it was judged
	// against; LimitSource says where the limit came from.
	Load        float64 `json:"load"`
	WaitLoad    float64 `json:"wait_load"`
	LimitSource string  `json:"limit_source"`
	Malformed   string  `json:"malformed,omitempty"`
	Cores       int     `json:"cores"`
	// RaceRuns is the race runs live at the last look, by pid.
	RaceRuns []int `json:"race_runs"`
	// Holder is the hold in force at the last look, when one was.
	Holder *WaitHolder `json:"holder,omitempty"`
	// Hold is the hold this wait took, when it opened.
	Hold   *WaitHolder `json:"hold,omitempty"`
	Reason string      `json:"reason,omitempty"`
	// Preflights and Waiters are what the process table held ahead of the
	// waiter at the last look: make preflights running and waiters queued
	// ahead or running their command, by pid (machineload.QueueAhead). They
	// name the pids in the text report; the JSON report carries the condition.
	Preflights []int `json:"-"`
	Waiters    []int `json:"-"`
	// UngatedTests are the go test runs no gate admitted
	// (machineload.UngatedTests), named beside the load they add to; they
	// shut nothing (iss-2610241130157455).
	UngatedTests []int `json:"ungated_tests,omitempty"`
	// Gate is the gate waited at, GateTests or GateRace.
	Gate string `json:"gate"`
	// Slots is how many preflights the first gate admits at once, and
	// SlotsSource where that came from (PreflightSlots). With one, the race
	// gate opens at once.
	Slots       int    `json:"slots,omitempty"`
	SlotsSource string `json:"slots_source,omitempty"`
	// SlotsNote says why the machine configuration was read past for the
	// count, when it was (PreflightSlots); the front door prints it once per
	// waiter, so a preflight, which waits at both gates, shows it twice.
	SlotsNote string `json:"slots_note,omitempty"`
	// Backoffs is how many times the waiter stood back from another that opened
	// in the same moment (machineload.StandBackFrom).
	Backoffs int `json:"backoffs,omitempty"`
	// Unmark stops the command form's mark (WaitRequest.Mark). The caller calls
	// it once its command has started, or could not be started; nil when there
	// is no mark.
	Unmark func() `json:"-"`
}

// WaitLoad waits until the gate opens or the timeout passes. Its error is a
// refusal before any wait (a home it cannot resolve, a home inside a git
// working tree that is not the account's own, a symlinked ~/.abcd.noindex, or
// a hold lock that is not a regular file), or a hold it
// could not give up when standing back after a wait. A timeout is a status,
// not an error.
func WaitLoad(req WaitRequest) (WaitResult, error) {
	getenv := req.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	now, sleep := req.Now, req.Sleep
	if now == nil {
		now = time.Now
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	if req.Read == nil {
		req.Read = machineload.Read
	}
	if req.ReadArgs == nil {
		req.ReadArgs = machineload.ReadArgs
	}
	if req.Self == (machineload.Self{}) {
		req.Self = machineload.Self{PID: os.Getpid(), UID: uint32(os.Geteuid())}
	}
	pollEvery := req.Poll
	if pollEvery <= 0 {
		pollEvery = WaitPollInterval
	}
	jitter := req.Jitter
	if jitter == nil {
		jitter = waitJitter
	}
	res := WaitResult{WaitedOn: []string{}, RaceRuns: []int{}, Gate: GateTests}
	if req.Race {
		res.Gate = GateRace
	}
	if reason, ci := cienv.Runner(getenv); ci {
		res.Status = WaitSkipped
		res.Reason = "skipped on a CI runner (" + reason + "): a fresh runner runs no other session's work, so there is nothing to wait for"
		return res, nil
	}
	home := req.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil || h == "" {
			return res, errors.New("the home directory could not be resolved, so there is nowhere to keep the hold; nothing was waited for")
		}
		home = h
	}
	// A home inside a git working tree that is not the account's own is
	// refused before anything is created under it (layered.MachineRoots): a
	// hold kept there would queue apart from the machine's own queue.
	if roots, _ := layered.MachineRoots(home); roots.HomeRefused {
		return res, errors.New("the home directory lies inside a working tree, so the load gate keeps no hold there; run with your own home; nothing was waited for")
	}
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return res, fmt.Errorf("nothing was waited for: %w", err)
	}
	defer root.Close()
	start := now()
	// readSlots reads the count at every look, before the machine is read, so
	// a change to it reaches a waiter already queued at its next look, and the
	// race gate's shortcut judges by the same value as the rest of that look
	// (itd-2610262127138979). A count that has become unusable refuses at the
	// look that finds it, naming where it was set, as it does before any wait.
	// open reports the race gate opening at once with one slot: the first gate
	// admitted this preflight's tests alone, so its race stage follows them at
	// once, as it did before the race gate existed. No look, no hold, no
	// jitter.
	readSlots := func() (open bool, err error) {
		n, src, note, err := PreflightSlots(getenv, home)
		if err != nil {
			if res.Polls == 0 {
				return false, fmt.Errorf("nothing was waited for: %w", err)
			}
			return false, fmt.Errorf("the wait stopped after %d looks: %w", res.Polls, err)
		}
		res.Slots, res.SlotsSource, res.SlotsNote = n, src, note
		if req.Race && n <= 1 {
			res.Status = WaitOpen
			res.Reason = "one preflight runs at a time, so the race stage follows its tests at once"
			// An open gate names nothing that held it shut; what an earlier
			// look found is WaitedOn's.
			res.Still, res.RaceRuns, res.Holder = nil, []int{}, nil
			res.Preflights, res.Waiters, res.UngatedTests = nil, nil, nil
			res.WaitedS = now().Sub(start).Seconds()
			return true, nil
		}
		return false, nil
	}
	deadline := start.Add(req.Timeout)
	announced, settled := false, false
	// crossedSince is the first look of the run of looks at which crossed
	// waiters alone held the gate shut, zero outside one, crossedN how many
	// were queued ahead at the last of them, and crossedSeen every waiter
	// crossed at some look of the run; past their cap (crossedCapFor, from
	// the last look's count) the looks stop counting them.
	var crossedSince time.Time
	crossedN := 0
	var crossedSeen []int
	pastCap := func() bool { return !crossedSince.IsZero() && now().Sub(crossedSince) >= crossedCapFor(crossedN) }
	// A crossed waiter that has left the queue (exited, or opened) starts the
	// count again: a waiter it held back may now open, and a count kept
	// against the smaller cap would already have run out, re-opening this one
	// past it at once. A fall in the count alone does not: a waiter whose
	// start lies at the tie tolerance's edge reads, in whole seconds, as
	// crossed at some looks and tied behind this one at others, and restarting
	// on each fall kept the count from ever reaching its cap. A rise, the same
	// waiter read crossed again, only lengthens the cap.
	noteCrossed := func(c crossedLook) {
		if c.n == 0 {
			crossedSince, crossedN, crossedSeen = time.Time{}, 0, nil
			return
		}
		left := false
		crossedSeen = slices.DeleteFunc(crossedSeen, func(pid int) bool {
			// A departed pid reused by a new waiter at this gate within one poll
			// would hide the departure; macOS's sequential pids make that
			// unreachable (about 99,998 forks in 15 s), and the open look
			// backstops it.
			gone := !slices.Contains(c.queued, pid)
			left = left || gone
			return gone
		})
		if crossedSince.IsZero() || left {
			crossedSince = now()
		}
		for _, pid := range c.crossed {
			if !slices.Contains(crossedSeen, pid) {
				crossedSeen = append(crossedSeen, pid)
			}
		}
		crossedN = c.n
	}
	for {
		if open, err := readSlots(); err != nil || open {
			return res, err
		}
		res.Polls++
		shut, done, only, err := waitLook(req, root, home, &res, now, false, pastCap())
		if err != nil {
			return res, err
		}
		noteCrossed(only)
		if !done && len(shut) == 0 {
			// Free: wait the jitter, then check everything again, and open only
			// if the machine is still free. The jitter never runs past the
			// deadline; with no time left the re-check is immediate.
			j := min(jitter(), max(deadline.Sub(now()), 0))
			if !settled {
				settled = true
				if req.OnSettle != nil {
					req.OnSettle(j)
				}
			}
			sleep(j)
			if open, err := readSlots(); err != nil || open {
				return res, err
			}
			res.Polls++
			shut, done, only, err = waitLook(req, root, home, &res, now, true, pastCap())
			if err != nil {
				return res, err
			}
			noteCrossed(only)
		}
		if done && res.Status == WaitOpen && req.Command && res.Backoffs < WaitBackoffCap {
			first, stop := openLook(req, sleep)
			if len(first) > 0 {
				// Stand back: another opened in the same moment, and it goes
				// first. Give up the hold, re-draw the jitter, and wait again.
				stop()
				if res.Hold != nil {
					if err := releaseHoldIn(root, *res.Hold); err != nil {
						return res, fmt.Errorf("the hold %s could not be given up: %w", abcdhome.Display(holdFileName(res.Hold.Gate)), err)
					}
				}
				res.Status, res.Hold, res.Waiters = "", nil, first
				res.Backoffs++
				// The run of looks ends here. A waiter that opened beside this
				// one past its cap is one this wait had stopped counting, ahead
				// of it, or one that opened as this one did; counting the cap
				// afresh hands the next moment to the waiter ahead, where
				// keeping the count would re-open this one past it at once.
				noteCrossed(crossedLook{})
				if !slices.Contains(res.WaitedOn, WaitOnWaiter) {
					res.WaitedOn = append(res.WaitedOn, WaitOnWaiter)
				}
				j := min(jitter(), max(deadline.Sub(now()), 0))
				if req.OnBackoff != nil {
					req.OnBackoff(first, j)
				}
				sleep(j)
				continue
			}
			res.Unmark = stop
		}
		res.WaitedS = now().Sub(start).Seconds()
		if done {
			// What held an earlier look shut is history (WaitedOn), not the
			// state of a gate that has now opened.
			res.Still = nil
			return res, nil
		}
		res.Still = shut
		for _, c := range shut {
			if !slices.Contains(res.WaitedOn, c) {
				res.WaitedOn = append(res.WaitedOn, c)
			}
		}
		if !announced {
			announced = true
			if req.OnWait != nil {
				req.OnWait(res)
			}
		}
		left := deadline.Sub(now())
		if left <= 0 {
			res.Status = WaitTimedOut
			return res, nil
		}
		sleep(min(pollEvery, left))
	}
}

// openLook is the command form's last look, once it has opened and before it
// starts its command. Across accounts no hold keeps two waiters apart, so
// both can open in one moment, before either shows in the table as running
// its command; under `--wait -- make preflight` each preflight's own gate then
// waits on the other's running waiter until its timeout. So the waiter marks
// itself as running (req.Mark), waits WaitOpenPause, and reads the table once
// more: the waiters it returns opened within that moment, and this one does
// not precede them by start and by pid alike (machineload.StandBackFrom), so
// it stands back from them. The mark is left running for the
// caller to stop once the command has started (WaitResult.Unmark); a mark that
// cannot be started leaves this waiter unseen by the others' look, and its own
// look still runs. A table that cannot be read finds no one
// (dcn-2610162257477571).
func openLook(req WaitRequest, sleep func(time.Duration)) ([]int, func()) {
	mark := req.Mark
	if mark == nil {
		mark = sleepMark
	}
	stop, err := mark()
	if err != nil || stop == nil {
		stop = func() {}
	}
	var once sync.Once
	stopOnce := func() { once.Do(stop) }
	sleep(WaitOpenPause)
	snap, err := req.Read()
	if err != nil || !snap.HasProcs {
		return nil, stopOnce
	}
	args, err := req.ReadArgs(machineload.GateCandidates(snap, req.Self))
	if err != nil {
		return nil, stopOnce
	}
	window := WaitOpenPause + machineload.StartTolerance*time.Second
	opened := machineload.JustOpened(snap, req.Self, args, window, req.Race)
	return machineload.StandBackFrom(snap, req.Self, opened), stopOnce
}

// sleepMark is the default mark: a /bin/sleep child of the waiter, which the
// process table shows as the waiter running its command. It outlives no
// waiter for long: the waiter stops it once its command has started, and a
// waiter that dies first leaves it to end on its own.
func sleepMark() (func(), error) {
	c := exec.Command("/bin/sleep", "30")
	if err := c.Start(); err != nil {
		return nil, err
	}
	return func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	}, nil
}

// waitLook is one look at the machine and the hold. It reports the conditions
// that hold the gate shut, or done when the wait is over (opened, or a platform
// it cannot judge). Only a look with take set opens the gate and takes the
// hold; one without it that finds nothing shut reports no condition and not
// done, and the caller re-checks after the jitter. With uncrossed set it does
// not count crossed waiters (machineload.Ahead.Crossed, WaitCrossedCap).
// crossedOnly says what crossedLook holds.
func waitLook(req WaitRequest, root *os.Root, home string, res *WaitResult, now func() time.Time, take, uncrossed bool) (shut []string, done bool, crossedOnly crossedLook, err error) {
	snap, readErr := req.Read()
	if errors.Is(readErr, machineload.ErrUnsupported) {
		res.Status = WaitUnchecked
		res.Reason = "abcd reads load and processes on macOS and Linux only, and this is " + termsafe.Sanitize(runtime.GOOS) + "; opening without judging"
		return nil, true, crossedLook{}, nil
	}
	t := now()
	res.Holder, res.RaceRuns, res.Reason = nil, []int{}, ""
	res.Preflights, res.Waiters, res.UngatedTests = nil, nil, nil
	if readErr != nil {
		res.Reason = termsafe.Sanitize(readErr.Error())
	}
	if !snap.HasLoad {
		shut = append(shut, WaitOnUnreadable)
	} else {
		if snap.Cores < 1 {
			snap.Cores = runtime.NumCPU()
		}
		lim, limits := readLimits(home, snap.Cores)
		res.Load, res.WaitLoad, res.Cores = snap.Load1, lim.WaitLoad, snap.Cores
		res.LimitSource, res.Malformed = limits.Source, limits.Malformed
		if !(snap.Load1 < lim.WaitLoad) {
			shut = append(shut, WaitOnLoad)
		}
	}
	procsOK := snap.HasProcs
	raceCmd := raceCommandAtBothGates(req, res)
	var runs, racing []int
	var ahead machineload.Ahead
	var raceAhead, crossed, queued []int
	if procsOK {
		args, err := req.ReadArgs(machineload.GateCandidates(snap, req.Self))
		if err != nil {
			procsOK = false
			res.Reason = termsafe.Sanitize(err.Error())
		} else {
			runs = machineload.RaceRuns(snap, req.Self, args)
			racing = runs
			if req.Race || raceCmd {
				// A race command's waiter running its command is a race run
				// from its mark onwards, before its go test shows.
				racing = append(append([]int(nil), runs...), machineload.RaceCommandWaiters(snap, req.Self, args)...)
				slices.Sort(racing)
			}
			ahead = machineload.QueueAhead(snap, req.Self, args)
			// Named only while the load is what shuts the gate, the condition
			// they add to, as the race runs are named only when they shut it.
			if slices.Contains(shut, WaitOnLoad) {
				res.UngatedTests = machineload.UngatedTests(snap, req.Self, args)
			}
			var raceCrossed []int
			raceAhead, raceCrossed = machineload.RaceAhead(snap, req.Self, args)
			crossed = ahead.Crossed
			if req.Race {
				crossed = raceCrossed
			}
			queued = machineload.QueuedAt(snap, req.Self, args, req.Race)
		}
	}
	notCrossed := func(pids []int) []int {
		return slices.DeleteFunc(slices.Clone(pids), func(pid int) bool { return slices.Contains(crossed, pid) })
	}
	if uncrossed {
		ahead, raceAhead = ahead.Uncrossed(), notCrossed(raceAhead)
	}
	// crossedShut is the waiter condition resting on crossed waiters alone.
	crossedShut := false
	if !procsOK {
		if !slices.Contains(shut, WaitOnUnreadable) {
			shut = append(shut, WaitOnUnreadable)
		}
	} else {
		// A live race run shuts the race gate always, and the first gate
		// only while one preflight runs at a time, or for a waiter whose own
		// command is a race run: with more, the race gate is what keeps race
		// stages apart, and the load limit what keeps the machine from
		// saturating.
		if len(racing) > 0 && (req.Race || res.Slots <= 1 || raceCmd) {
			shut = append(shut, WaitOnRaceRun)
			res.RaceRuns = racing
		}
		if req.Race {
			if len(raceAhead) > 0 {
				shut = append(shut, WaitOnWaiter)
				res.Waiters = raceAhead
				crossedShut = len(notCrossed(raceAhead)) == 0
			}
		} else {
			preflight, waiter := ahead.Shut(res.Slots)
			if preflight {
				shut = append(shut, WaitOnPreflight)
				res.Preflights = ahead.TakingSlots(res.Slots).Preflights
			}
			if waiter {
				shut = append(shut, WaitOnWaiter)
				res.Waiters = ahead.TakingSlots(res.Slots).Waiters
				_, without := ahead.Uncrossed().Shut(res.Slots)
				crossedShut = !without
			}
		}
	}

	// The hold is judged at every look, under the flock, so a command hold
	// whose work has started its race run is retired by whichever waiter sees
	// it, and two waiters opened by one load drop cannot both take it. A wait
	// judges and takes only its own gate's hold: a first gate's command hold
	// lasts until its preflight starts its race run, so were it to shut the
	// race gate too, an earlier preflight's race waiter would wait on it while
	// that preflight's own race waiter queued behind the earlier one, and
	// neither would move until its timeout.
	err = fsutil.WithFileLockIn(root, waitLockFileName, waitLockTimeout, func() error {
		var free []string
		for _, gate := range holdGates(req, res) {
			name := holdFileName(gate)
			if h, ok := readHoldIn(root, gate); ok {
				// A hold taken by this waiter's own ancestor (`--wait -- make
				// preflight`, whose preflight gates itself) admitted the work
				// this waiter is part of: it neither shuts this waiter nor is
				// replaced.
				if procsOK && machineload.Descends(snap.Procs, req.Self.PID, h.PID) {
					continue
				}
				if holdInForce(h, snap, procsOK, runs, t) {
					if res.Holder == nil {
						res.Holder = &h
					}
					if !slices.Contains(shut, WaitOnLock) {
						shut = append(shut, WaitOnLock)
					}
					continue
				}
				if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%s: %w", abcdhome.Display(name), err)
				}
			}
			free = append(free, gate)
		}
		if len(shut) > 0 || !take || len(free) == 0 {
			return nil
		}
		own := WaitHolder{PID: req.Self.PID, Opened: t, Until: t.Add(WaitSettle), Form: HoldSettle, Gate: res.Gate}
		if s, ok := procStart(snap, req.Self.PID, t); ok {
			own.StartUnix = s
			// The command form rests on its holder's life, so it needs the
			// holder's start time to be proven later; without one it takes the
			// plain form's bounded hold instead. Above one slot a command that
			// is not a race run (a whole preflight, `--wait -- make preflight`)
			// takes the settle hold, as the plain first gate inside a preflight
			// did: the units in their test stages are counted from the process
			// table (machineload.Ahead.Shut), and a command hold would keep the
			// account's next slot shut until this run reached its race stage.
			if req.Command && (res.Slots <= 1 || raceCmd) {
				own.Until, own.Form = t.Add(WaitCommandCap), HoldCommand
			}
		}
		for _, gate := range free {
			rec := own
			rec.Gate = gate
			raw, err := json.Marshal(rec)
			if err != nil {
				return err
			}
			if err := fsutil.WriteFileAtomicInRoot(root, holdFileName(gate), append(raw, '\n'), abcdhome.FileMode); err != nil {
				return fmt.Errorf("%s: %w", abcdhome.Display(holdFileName(gate)), err)
			}
		}
		res.Hold = &own
		return nil
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		shut = append(shut, WaitOnLock)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return nil, false, crossedLook{}, fmt.Errorf("the hold's lock %s is not a regular file (a symlink, or something else), so it is refused; remove it, and the next wait creates it afresh",
			abcdhome.Display(waitLockFileName))
	case err != nil:
		return nil, false, crossedLook{}, fmt.Errorf("the hold could not be judged or written: %w", err)
	}
	if len(crossed) > 0 && (len(shut) == 0 || (crossedShut && slices.Equal(shut, []string{WaitOnWaiter}))) {
		crossedOnly = crossedLook{n: len(crossed), crossed: crossed, queued: queued}
	}
	if len(shut) > 0 || !take {
		return shut, false, crossedOnly, nil
	}
	res.Status = WaitOpen
	return nil, true, crossedOnly, nil
}

// holdInForce judges a hold at time t against the snapshot. Past its Until it
// has lapsed. A settle hold is otherwise in force. A command hold is in force
// while its holder is alive as the same process (its pid in the table with the
// start time the record carries) and has not started its own race run; a table
// that could not be read cannot disprove the holder, so the hold stands until
// its Until.
func holdInForce(h WaitHolder, snap machineload.Snapshot, procsOK bool, runs []int, t time.Time) bool {
	if !t.Before(h.Until) {
		return false
	}
	if h.Form != HoldCommand {
		return true
	}
	if !procsOK {
		return true
	}
	start, ok := procStart(snap, h.PID, t)
	if !ok || h.StartUnix == 0 || start-h.StartUnix > holdStartSlack || h.StartUnix-start > holdStartSlack {
		return false
	}
	for _, r := range runs {
		if machineload.Descends(snap.Procs, r, h.PID) {
			return false
		}
	}
	return true
}

// procStart is pid's start time in Unix seconds, from its age in the snapshot
// read at t.
func procStart(snap machineload.Snapshot, pid int, t time.Time) (int64, bool) {
	if !snap.HasProcs {
		return 0, false
	}
	for _, p := range snap.Procs {
		if p.PID == pid {
			return t.Add(-p.Age).Unix(), true
		}
	}
	return 0, false
}

// readHoldIn reads gate's hold record through root. A record that is absent,
// not a regular file, too large or unreadable holds nothing, and is replaced by
// the next hold taken. A record is its file's gate's, whatever it says.
func readHoldIn(root *os.Root, gate string) (WaitHolder, bool) {
	name := holdFileName(gate)
	fi, err := root.Lstat(name)
	if err != nil || !fi.Mode().IsRegular() {
		return WaitHolder{}, false
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return WaitHolder{}, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxHoldBytes+1))
	if err != nil || len(raw) > maxHoldBytes {
		return WaitHolder{}, false
	}
	var h WaitHolder
	if json.Unmarshal(raw, &h) != nil || h.PID <= 0 || (h.Form != HoldSettle && h.Form != HoldCommand) {
		return WaitHolder{}, false
	}
	h.Gate = gate
	return h, true
}

// ReleaseWaitHold removes the hold h wherever it is still the record in
// force: the command form calls it when its command exits. A hold another
// waiter has since taken is left alone.
func ReleaseWaitHold(home string, h WaitHolder) error {
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel())
	if err != nil {
		return err
	}
	defer root.Close()
	return releaseHoldIn(root, h)
}

// releaseHoldIn removes the hold h through root from each gate's record that
// is still h, whatever gate h names: one wait may hold more than one gate.
func releaseHoldIn(root *os.Root, h WaitHolder) error {
	return fsutil.WithFileLockIn(root, waitLockFileName, waitLockTimeout, func() error {
		for _, gate := range []string{GateTests, GateRace} {
			cur, ok := readHoldIn(root, gate)
			if !ok || cur.PID != h.PID || cur.StartUnix != h.StartUnix || !cur.Opened.Equal(h.Opened) {
				continue
			}
			if err := root.Remove(holdFileName(gate)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	})
}

// PreflightParallelism is the -p `make preflight` gives its test and race
// stages (cmd/test-parallelism prints it), derived from the online cores and
// the one-minute load as they are when the stage starts, the cores divided
// among the preflights the first gate admits at once
// (machineload.GoTestParallelism, PreflightSlots). On a CI runner the machine
// is not read and go's own default, GOMAXPROCS, stands, as it does when the
// machine cannot be read.
func PreflightParallelism(getenv func(string) string, read func() (machineload.Snapshot, error), slots int) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	if read == nil {
		read = machineload.Read
	}
	if _, ci := cienv.Runner(getenv); ci {
		return runtime.GOMAXPROCS(0)
	}
	snap, err := read()
	if !snap.HasLoad || (err != nil && errors.Is(err, machineload.ErrUnsupported)) {
		return runtime.GOMAXPROCS(0)
	}
	return machineload.GoTestParallelism(snap, slots)
}
