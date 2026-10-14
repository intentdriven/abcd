package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/cienv"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The waiter (`abcd implement load --wait`, iss-2610111826193257) is the one
// wait-for-capacity primitive concurrent sessions use instead of writing their
// own, and `make preflight` runs its test stages through it. It blocks until
// the one-minute load average is strictly below the machine's wait limit
// (wait-load in ~/.abcd.noindex/load-limits, twice the online core count by
// default), no other `go test -race` run is live, no other `make preflight` is
// running and no waiter is queued ahead of it or running its command, every one
// judged from the process table and each candidate's argv, under any account,
// with the waiter's own process and ancestors excluded (machineload.RaceRuns,
// machineload.QueueAhead). It never signals anything.
//
// The process table is the cross-account half (iss-2610140638074587): it
// shows every account's processes, and nothing in it is a file another account
// can write, so no account can hold another's gate shut. A look that finds the
// machine free does not open the gate: the waiter waits a random 15 to 60
// seconds and checks everything again, and opens only if that re-check also
// finds it free, so two starters released by one quiet minute do not start
// together (iss-2610102020442624).
//
// A falling load opens every waiter watching it at once, which on 2026-10-11
// put four race runs on a 16-core machine together. So within one account the
// waiter that opens also takes a hold, a record under ~/.abcd.noindex/ written
// under a flock, and every other waiter of that account stays shut while the
// hold is in force (a hold whose holder is the waiter's own ancestor, as when
// `--wait -- make preflight` runs a preflight that gates itself, is not):
//
//   - the plain form (`--wait` alone) holds for WaitSettle after opening, long
//     enough for the one-minute load average to show the work it admitted. Its
//     holder exits right after opening, by design, so its life is not asked.
//   - the command form (`--wait -- <command>`) holds while its holder (the
//     waiter, which runs the command) is alive as the same process (pid and
//     start time), until the command has started its own race run, the
//     command exits, or WaitCommandCap passes, whichever comes first. A holder
//     that has died, or whose pid now belongs to another process, is not
//     trusted, and its hold is reclaimed by the next waiter that looks.

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
	// WaitOnRaceRun: another go test -race run is live.
	WaitOnRaceRun = "race-run"
	// WaitOnLock: another waiter's hold is in force.
	WaitOnLock = "lock"
	// WaitOnUnreadable: the load or the process table could not be read.
	WaitOnUnreadable = "unreadable"
	// WaitOnPreflight: another make preflight, of any account, is running.
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
)

// waitJitter is a random wait from WaitJitterMin to WaitJitterMax.
func waitJitter() time.Duration {
	return WaitJitterMin + rand.N(WaitJitterMax-WaitJitterMin+1)
}

// WaitHoldFileName is the hold record under ~/.abcd.noindex/.
const WaitHoldFileName = "load-wait.json"

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
	// OnWait is called once, at the first look that finds the gate shut.
	OnWait func(WaitResult)
	// OnSettle is called once, at the first look that finds the machine free,
	// with the jitter the waiter waits before it checks again.
	OnSettle func(time.Duration)
	// Jitter is the wait between a look that finds the machine free and the
	// re-check that opens the gate; nil is a random WaitJitterMin to
	// WaitJitterMax.
	Jitter func() time.Duration
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
}

// WaitLoad waits until the gate opens or the timeout passes. Its error is a
// refusal before any wait: a home it cannot resolve, a symlinked
// ~/.abcd.noindex, or a hold lock that is not a regular file. A timeout is a
// status, not an error.
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
	res := WaitResult{WaitedOn: []string{}, RaceRuns: []int{}}
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
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return res, fmt.Errorf("nothing was waited for: %w", err)
	}
	defer root.Close()

	start := now()
	deadline := start.Add(req.Timeout)
	announced, settled := false, false
	for {
		res.Polls++
		shut, done, err := waitLook(req, root, home, &res, now, false)
		if err != nil {
			return res, err
		}
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
			res.Polls++
			shut, done, err = waitLook(req, root, home, &res, now, true)
			if err != nil {
				return res, err
			}
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

// waitLook is one look at the machine and the hold. It reports the conditions
// that hold the gate shut, or done when the wait is over (opened, or a platform
// it cannot judge). Only a look with take set opens the gate and takes the
// hold; one without it that finds nothing shut reports no condition and not
// done, and the caller re-checks after the jitter.
func waitLook(req WaitRequest, root *os.Root, home string, res *WaitResult, now func() time.Time, take bool) ([]string, bool, error) {
	snap, readErr := req.Read()
	if errors.Is(readErr, machineload.ErrUnsupported) {
		res.Status = WaitUnchecked
		res.Reason = "abcd reads load and processes on macOS and Linux only, and this is " + termsafe.Sanitize(runtime.GOOS) + "; opening without judging"
		return nil, true, nil
	}
	t := now()
	var shut []string
	res.Holder, res.RaceRuns, res.Reason = nil, []int{}, ""
	res.Preflights, res.Waiters = nil, nil
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
	var runs []int
	var ahead machineload.Ahead
	if procsOK {
		args, err := req.ReadArgs(machineload.GateCandidates(snap, req.Self))
		if err != nil {
			procsOK = false
			res.Reason = termsafe.Sanitize(err.Error())
		} else {
			runs = machineload.RaceRuns(snap, req.Self, args)
			ahead = machineload.QueueAhead(snap, req.Self, args)
		}
	}
	if !procsOK {
		if !slices.Contains(shut, WaitOnUnreadable) {
			shut = append(shut, WaitOnUnreadable)
		}
	} else {
		if len(runs) > 0 {
			shut = append(shut, WaitOnRaceRun)
			res.RaceRuns = runs
		}
		if len(ahead.Preflights) > 0 {
			shut = append(shut, WaitOnPreflight)
			res.Preflights = ahead.Preflights
		}
		if len(ahead.Waiters) > 0 {
			shut = append(shut, WaitOnWaiter)
			res.Waiters = ahead.Waiters
		}
	}

	// The hold is judged at every look, under the flock, so a command hold whose
	// work has started its race run is retired by whichever waiter sees it, and
	// two waiters opened by one load drop cannot both take it.
	err := fsutil.WithFileLockIn(root, waitLockFileName, waitLockTimeout, func() error {
		if h, ok := readHoldIn(root); ok {
			// A hold taken by this waiter's own ancestor (`--wait -- make
			// preflight`, whose preflight gates itself) admitted the work this
			// waiter is part of: it neither shuts this waiter nor is replaced.
			if procsOK && machineload.Descends(snap.Procs, req.Self.PID, h.PID) {
				return nil
			}
			if holdInForce(h, snap, procsOK, runs, t) {
				res.Holder = &h
				shut = append(shut, WaitOnLock)
				return nil
			}
			if err := root.Remove(WaitHoldFileName); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if len(shut) > 0 || !take {
			return nil
		}
		own := WaitHolder{PID: req.Self.PID, Opened: t, Until: t.Add(WaitSettle), Form: HoldSettle}
		if s, ok := procStart(snap, req.Self.PID, t); ok {
			own.StartUnix = s
			// The command form rests on its holder's life, so it needs the
			// holder's start time to be proven later; without one it takes the
			// plain form's bounded hold instead.
			if req.Command {
				own.Until, own.Form = t.Add(WaitCommandCap), HoldCommand
			}
		}
		raw, err := json.Marshal(own)
		if err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomicInRoot(root, WaitHoldFileName, append(raw, '\n'), abcdhome.FileMode); err != nil {
			return err
		}
		res.Hold = &own
		return nil
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		shut = append(shut, WaitOnLock)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return nil, false, fmt.Errorf("the hold's lock %s is not a regular file (a symlink, or something else), so it is refused; remove it, and the next wait creates it afresh",
			abcdhome.Display(waitLockFileName))
	case err != nil:
		return nil, false, fmt.Errorf("the hold %s could not be judged or written: %w", abcdhome.Display(WaitHoldFileName), err)
	}
	if len(shut) > 0 || !take {
		return shut, false, nil
	}
	res.Status = WaitOpen
	return nil, true, nil
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

// readHoldIn reads the hold record through root. A record that is absent, not
// a regular file, too large or unreadable holds nothing, and is replaced by the
// next hold taken.
func readHoldIn(root *os.Root) (WaitHolder, bool) {
	fi, err := root.Lstat(WaitHoldFileName)
	if err != nil || !fi.Mode().IsRegular() {
		return WaitHolder{}, false
	}
	f, err := root.OpenFile(WaitHoldFileName, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
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
	return h, true
}

// ReleaseWaitHold removes the hold h when it is still the record in force: the
// command form calls it when its command exits. A hold another waiter has since
// taken is left alone.
func ReleaseWaitHold(home string, h WaitHolder) error {
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel())
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WithFileLockIn(root, waitLockFileName, waitLockTimeout, func() error {
		cur, ok := readHoldIn(root)
		if !ok || cur.PID != h.PID || cur.StartUnix != h.StartUnix || !cur.Opened.Equal(h.Opened) {
			return nil
		}
		if err := root.Remove(WaitHoldFileName); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}

// PreflightParallelism is the -p `make preflight` gives its test and race
// stages (cmd/test-parallelism prints it), derived from the online cores and
// the one-minute load as they are when the stage starts
// (machineload.GoTestParallelism). On a CI runner the machine is not read and
// go's own default, GOMAXPROCS, stands, as it does when the machine cannot be
// read.
func PreflightParallelism(getenv func(string) string, read func() (machineload.Snapshot, error)) int {
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
	return machineload.GoTestParallelism(snap)
}
