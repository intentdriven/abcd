package loop

// control.go is what a run records about who drives it, beside its state
// (itd-2611041549461012, spc-2611041706004822): the live driver, each in-flight
// launch with its route, and the stop that halted the run. A coordinator that
// dies leaves these for the next session to read. They live in a file of their
// own under the run's directory, written under the run's lock, so the state's
// schema does not move for them and an abcd that predates them reads the run
// as it always did.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// ControlFileName is the record, under the run's directory.
	ControlFileName = "control.json"
	controlVersion  = 1
	// maxControlBytes bounds a read of it: a driver, a stop naming each lane and
	// a launch per await.
	maxControlBytes = 1 << 20
	// StageDriver is the refusal stage and the record's stage for the driver.
	StageDriver = "driver"
	// runCheckoutsFile lists, in abcd's home, the checkouts a driver worked in,
	// which is how a listing finds a repository other than its own.
	runCheckoutsFile     = "run-checkouts"
	maxRunCheckoutsBytes = 256 << 10
)

// LiveDriver is the process driving a run: the host session a step runs under,
// or the plain terminal's own.
type LiveDriver struct {
	// Session is the host session's id where the session-start hook recorded
	// one; "" for a plain terminal.
	Session   string    `json:"session,omitempty"`
	PID       int       `json:"pid"`
	StartUnix int64     `json:"start_unix"`
	SlackMS   int64     `json:"slack_ms,omitempty"`
	At        time.Time `json:"at"`
}

func (d LiveDriver) host() sessionname.Host {
	return sessionname.Host{PID: d.PID, StartUnix: d.StartUnix, Slack: time.Duration(d.SlackMS) * time.Millisecond}
}

// Launch is one in-flight launch: an agent a step started through a runner, the
// route it runs on, and the process that started it. It stands while the await
// names the same runner claim; a launch's own process is what says it is alive.
type Launch struct {
	Lane      string    `json:"lane"`
	Role      string    `json:"role"`
	Runner    string    `json:"runner"`
	Route     string    `json:"route,omitempty"`
	Session   string    `json:"session"`
	PID       int       `json:"pid,omitempty"`
	StartUnix int64     `json:"start_unix,omitempty"`
	SlackMS   int64     `json:"slack_ms,omitempty"`
	At        time.Time `json:"at"`
}

func (l Launch) host() sessionname.Host {
	return sessionname.Host{PID: l.PID, StartUnix: l.StartUnix, Slack: time.Duration(l.SlackMS) * time.Millisecond}
}

// RunStop is the stop a run took: who set it and when, what is left of the
// halting, and what each lane left once it settled.
type RunStop struct {
	SetBy   string    `json:"set_by"`
	SetAt   time.Time `json:"set_at"`
	NotedAt time.Time `json:"noted_at"`
	// HaltedAt is when the last lane settled and the claims were let go; nil
	// while an agent is still in its call.
	HaltedAt *time.Time `json:"halted_at,omitempty"`
	// Released says what became of the run's reservation.
	Released string     `json:"released,omitempty"`
	Left     []StopLeft `json:"left,omitempty"`
	// FromFile says the stop was read from a stop file, which a lift removes; a
	// signal's stop has none, and stays until a take-over.
	FromFile bool `json:"from_file,omitempty"`
}

// StopLeft is what the stop left in one lane.
type StopLeft struct {
	Lane  string `json:"lane"`
	Stage Stage  `json:"stage"`
	// Settled is false while the lane's agent is still in its call.
	Settled bool `json:"settled"`
	// Aside is where its uncommitted work was saved, and Files how many files
	// that holds.
	Aside string `json:"aside,omitempty"`
	Files int    `json:"files,omitempty"`
	// Note says anything else: a receipt collected, an agent lost, work that
	// could not be saved.
	Note string `json:"note,omitempty"`
}

// control is the file's whole shape.
type control struct {
	SchemaVersion int         `json:"schema_version"`
	Driver        *LiveDriver `json:"driver,omitempty"`
	Launches      []Launch    `json:"launches,omitempty"`
	Stop          *RunStop    `json:"stop,omitempty"`
	// Shared is the shared folder the run started under (runstop.Pin).
	Shared *runstop.Pin `json:"shared,omitempty"`
}

func controlRel(runID string) string { return runRel(runID) + "/" + ControlFileName }

// readControl reads the run's control record; a run with none has an empty one.
func readControl(root *os.Root, runID string) (control, error) {
	rel := controlRel(runID)
	data, err := localtier.ReadInRoot(root, rel, maxControlBytes)
	if rf := tierRefusal("state", "", err); rf != nil {
		return control{}, rf
	}
	if errors.Is(err, fs.ErrNotExist) {
		return control{SchemaVersion: controlVersion}, nil
	}
	if err != nil {
		return control{}, refuse("state", "", "", fmt.Sprintf("%s cannot be read: %s", rel, fsutil.RedactHome(err.Error())),
			"the loop is the file's only writer; restore it or remove it")
	}
	var c control
	if err := json.Unmarshal(data, &c); err != nil || c.SchemaVersion != controlVersion {
		return control{}, refuse("state", "", "", rel+" does not parse as this abcd's control record",
			"the loop is the file's only writer; restore it or remove it")
	}
	return c, nil
}

// writeControl replaces the record whole; the caller holds the run's lock.
func writeControl(root *os.Root, runID string, c control) error {
	c.SchemaVersion = controlVersion
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	rel := controlRel(runID)
	if err := fsutil.WriteFileAtomicInRoot(root, rel, append(data, '\n'), filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// readProcs reads the machine's process table; a test stands one in.
var readProcs = func() (sessionname.Table, error) { return sessionname.Live().ReadTable() }

// hereFrom is the process a host walk starts from; a test stands one in.
var hereFrom = os.Getppid

// procOf is the table's reading of pid, as a Host a later read can judge; ok is
// false when the table does not hold it.
func procOf(tab sessionname.Table, pid int) (sessionname.Host, bool) {
	for _, p := range tab.Procs {
		if p.PID == pid {
			return sessionname.Host{PID: pid, StartUnix: tab.ReadAt.Add(-p.Age).Unix(), Slack: tab.Took}, true
		}
	}
	return sessionname.Host{}, false
}

// ClaimDriver is the one call a driver makes before it steps a run: it records
// the host process this call runs under as the run's live driver, and refuses at
// exit 3 while another driver's process is alive (iss-2611022247128872). One gone
// is replaced, with an entry in the run's record. A table that cannot be read
// claims nothing and refuses nothing: a step is never stopped by the machine's
// process listing. A walk that names no host is a loop its session left behind
// (iss-2611022247128872): it claims nothing, and backs off from a live driver.
// Under a stop it claims nothing either: the stop is settled by the driver the run's
// agents ran under, and the move that follows gets the stop's refusal.
func ClaimDriver(repoRoot, runID string, o Options) error {
	c, err := runControl(repoRoot, runID)
	if err != nil {
		return err
	}
	if o.stopRead(c.Shared).Stop != nil || (c.Stop != nil && !c.Stop.FromFile) {
		if _, err := stopCheck(repoRoot, runID, DefaultStages(), o); err != nil {
			if _, refused := AsRefusal(err); !refused {
				return err
			}
		}
		return nil
	}
	return claimDriver(repoRoot, runID, o, false)
}

// driverOf is the process a step's run is driven by: the nearest one above it that
// holds a session, so a program a host wraps one step in does not stand as the driver,
// and without one the host walk's answer.
func driverOf(tab sessionname.Table, from int, home string) (sessionname.Host, string, error) {
	if home != "" {
		for _, h := range sessionname.Lineage(tab, from) {
			if s := occupancy.SessionOn(home, h); s != "" {
				return h, s, nil
			}
		}
	}
	h, err := sessionname.FindHost(tab, from)
	return h, "", err
}

// claimDriver is ClaimDriver's claim, which a take-over makes directly. over says a
// live driver gives way: a run every lane of which has settled has no agent in its
// call, and an idle session does not hold it from being taken over.
func claimDriver(repoRoot, runID string, o Options, over bool) error {
	tab, err := readProcs()
	if err != nil {
		return nil
	}
	home := o.stopHomeOf()
	here, marked, hostErr := driverOf(tab, hereFrom(), home)
	session := o.Session
	if session == "" {
		session = marked
	}
	return mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		now := o.now()
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		changed := false
		if d := c.Driver; d != nil {
			// Each reading measures a start time back from itself, so two of one
			// driver differ by a second: this process is the driver when it
			// holds the id and the table says the driver is alive.
			alive := tab.Alive(d.host())
			if hostErr == nil && d.PID == here.PID && alive {
				return false, nil
			}
			if !alive {
				// The table was read before the lock, so a driver that claimed
				// since is not in it and is not gone.
				fresh, err := readProcs()
				if err != nil {
					return false, nil
				}
				alive = fresh.Alive(d.host())
			}
			if alive && !over {
				return false, contend(StageDriver, "", "", fmt.Sprintf("%s is driven by %s, which has been alive since it took the run at %s",
					st.RunID, driverName(*d), d.At.UTC().Format(time.RFC3339)),
					"back off: only one driver steps a run; once that driver is gone, `abcd implement resume` lists what it left")
			}
			if hostErr != nil {
				return false, nil
			}
			st.Record = append(st.Record, Entry{At: now, Stage: StageDriver,
				Note: fmt.Sprintf("the driver %s is gone; %s took the run", driverName(*d), driverName(LiveDriver{Session: session, PID: here.PID}))})
			st.UpdatedAt = now
			changed = true
		}
		if hostErr != nil {
			return false, nil
		}
		c.Driver = &LiveDriver{Session: session, PID: here.PID, StartUnix: here.StartUnix, SlackMS: here.Slack.Milliseconds(), At: now}
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		if home != "" {
			noteRunCheckout(home, repoRoot)
		}
		return changed, nil
	})
}

// driverName names a driver for a line: its session where there is one, and its
// process.
func driverName(d LiveDriver) string {
	if d.Session != "" {
		return fmt.Sprintf("session %s (process %d)", termsafe.Sanitize(d.Session), d.PID)
	}
	return fmt.Sprintf("process %d", d.PID)
}

// recordLaunch adds a launch to the run's record, under the run's lock, dropping
// any whose await no longer names its claim. A failure changes nothing about the
// launch: the await's own claim still holds the slot, and the agent then reads
// as alive for that claim's term.
func recordLaunch(root *os.Root, st *State, l Launch) {
	c, err := readControl(root, st.RunID)
	if err != nil {
		return
	}
	c.Launches = slices.DeleteFunc(c.Launches, func(old Launch) bool { return !st.claimStands(old) })
	if tab, err := readProcs(); err == nil {
		if h, ok := procOf(tab, os.Getpid()); ok {
			l.PID, l.StartUnix, l.SlackMS = h.PID, h.StartUnix, h.Slack.Milliseconds()
		}
	}
	c.Launches = append(c.Launches, l)
	_ = writeControl(root, st.RunID, c)
}

// claimStands reports whether an await still names the launch's runner claim.
func (s State) claimStands(l Launch) bool {
	for _, lane := range s.Lanes {
		if lane.ID != l.Lane {
			continue
		}
		for _, a := range lane.Awaits {
			if a.Running != nil && a.Running.Session == l.Session {
				return true
			}
		}
	}
	return false
}

// noteRunCheckout adds the checkout to the account's list once. The list is a
// convenience for a reader on the machine: a write that fails is no refusal.
func noteRunCheckout(home, repoRoot string) {
	for _, p := range runCheckouts(home) {
		if p == repoRoot {
			return
		}
	}
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return
	}
	defer root.Close()
	_ = fsutil.AppendLineIn(root, runCheckoutsFile, []byte(repoRoot), abcdhome.FileMode)
}

// runCheckouts is the account's list, read as its own word; nothing when it
// cannot be.
func runCheckouts(home string) []string {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(runCheckoutsFile), maxRunCheckoutsBytes)
	if err != nil || refusal != fsutil.DeclarationOK {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if filepath.IsAbs(line) && filepath.Clean(line) == line {
			out = append(out, line)
		}
	}
	return out
}
