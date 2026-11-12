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
	// PageFailed says why the day's page could not be written, "" when it could.
	PageFailed string `json:"page_failed,omitempty"`
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
	// ArmedPR is the pull request armed before the stop, which the forge can still merge
	// under it, 0 when there is none.
	ArmedPR int `json:"armed_pr,omitempty"`
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
// agents ran under, and the move that follows gets the stop's refusal. Nor does it claim
// a run a stop halted, after the lift as before it: a step cannot move that run, and a
// session that tried must not stand in the way of the take-over the run waits for. The
// same holds of a run a drive stopped on (drive_stop.go), which no step moves either.
func ClaimDriver(repoRoot, runID string, o Options) error {
	c, err := runControl(repoRoot, runID)
	if err != nil {
		return err
	}
	if driveStopped(runID) {
		return nil
	}
	if o.stopRead(c.Shared).Stop != nil || (c.Stop != nil && (!c.Stop.FromFile || c.Stop.HaltedAt != nil)) {
		if _, err := stopCheck(repoRoot, runID, DefaultStages(), o); err != nil {
			if _, refused := AsRefusal(err); !refused {
				return err
			}
		}
		return nil
	}
	return claimDriver(repoRoot, runID, o, false)
}

// hereSelf is this process; a test stands one in.
var hereSelf = os.Getpid

// selfDriver is a drive as its run's driver: its own process, since one walked up to
// from its parent would be a terminal that outlives it, or the first process under nohup.
func selfDriver(tab sessionname.Table) (sessionname.Host, string, error) {
	if h, ok := procOf(tab, hereSelf()); ok {
		return h, "", nil
	}
	return sessionname.Host{}, "", errors.New("this process is not in the process table")
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

// claimDriver is ClaimDriver's claim, which a take-over makes directly. overHalted
// lets a live driver give way where the run, as read under the lock, is halted and the
// driver claimed before the halt: every lane has settled, so no agent is in its call,
// and an idle session does not hold it from being taken over. One that claimed since
// is a take-over at work, which holds the run as any driver does.
func claimDriver(repoRoot, runID string, o Options, overHalted bool) error {
	tab, err := readProcs()
	if err != nil {
		return nil
	}
	home := o.stopHomeOf()
	here, marked, hostErr := o.driverHere(tab)
	session := o.Session
	if session == "" {
		session = marked
	}
	// The claim adds a line to the run's record and nothing else of its state, so
	// the file keeps the version it states (saveRecord): a claim is what lets this
	// session save it whole after.
	return saveRecord(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
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
			over := alive && overHalted && c.Stop != nil && c.Stop.HaltedAt != nil && !d.At.After(*c.Stop.HaltedAt)
			if alive && !over {
				return false, contend(StageDriver, "", "", fmt.Sprintf("%s is driven by %s, which has been alive since it took the run at %s",
					st.RunID, driverName(*d), d.At.UTC().Format(time.RFC3339)),
					"back off: only one driver steps a run; once that driver is gone, `abcd implement resume` lists what it left")
			}
			if hostErr != nil {
				return false, nil
			}
			said := fmt.Sprintf("the driver %s is gone", driverName(*d))
			if over {
				said = fmt.Sprintf("a stop halted the run, so its idle driver %s gave way", driverName(*d))
			}
			st.Record = append(st.Record, Entry{At: now, Stage: StageDriver,
				Note: fmt.Sprintf("%s; %s took the run", said, driverName(LiveDriver{Session: session, PID: here.PID}))})
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

// driverHere is the process this call would claim a run as: a drive's own, or the host
// a step runs under (driverOf).
func (o Options) driverHere(tab sessionname.Table) (sessionname.Host, string, error) {
	if o.driverIsSelf {
		return selfDriver(tab)
	}
	return driverOf(tab, hereFrom(), o.stopHomeOf())
}

// wholeSave is the right to replace one run's state file whole (iss-2611110933196955).
// A whole save writes the file at this abcd's version, and an abcd refuses a file above
// its own, so a save over a file that states another version raises the run out of its
// driver's reach. storeState, the state file's one writer, refuses such a file unless it
// is handed the right for that run and that stated version, and mayWholeSave alone
// mints one: a field-for-field value, so no caller can grant itself one by setting a
// flag.
type wholeSave struct {
	runID  string
	stated int
}

// mayWholeSave is the one judge of whether this session may save a run's whole state:
// it returns the right storeState needs, or the refusal that leaves the run as its file
// stands. stated is the version the file states (readListedStated), never a read
// State's, which an older file reaches already raised; a version above this abcd's is
// refused by that read before anything comes here.
//
// A file at this abcd's version may be saved by any session: an older driver refuses
// it, so none drives it, and a newer one never wrote it. A file stating another
// version is saved only by the run's driver: the live process its control record
// names (claimDriver) is the one this call runs under. Nothing else tells a driver
// resumed by a newer abcd from a second session in its checkout, so a session that has
// not claimed the run leaves it, and the way to take it up is a claim: a step makes
// one, a resumed start and a drain over its lanes make one where no live driver holds
// the run, and a take-over makes one over a halted run. A renewal (o.afar) is never the
// driver: its waiter may run a lane's own abcd under the driver's own host session.
func mayWholeSave(root *os.Root, runID string, stated int, o Options) (wholeSave, error) {
	if stated == SchemaVersion {
		return wholeSave{runID: runID, stated: stated}, nil
	}
	id := termsafe.Sanitize(runID)
	takeUp := "`abcd implement step --run " + id + "` claims it for this abcd while no stop's mark stands on it, and `abcd implement resume --take " +
		id + "` takes it over while one does; either names the driver it replaces in the run's record"
	var why, remedy string
	switch c, err := readControl(root, runID); {
	case o.afar:
		why = "a renewal may run a lane's own abcd, which never drives the run"
		remedy = "nothing is needed here: the run's driver saves it at its next step"
	case err != nil:
		why = "its control record cannot be read, so whether this session drives it cannot be told"
		remedy = "restore " + controlRel(id) + ", which the loop alone writes, or remove it; then " + takeUp
	case c.Driver == nil:
		why = "its control record names no driver, and this session has not claimed it"
		remedy = takeUp
	default:
		tab, err := readProcs()
		if err != nil {
			why = "the process table cannot be read, so whether this session drives it cannot be told"
			remedy = "run it again once the process table can be read"
			break
		}
		alive := tab.Alive(c.Driver.host())
		here, _, hostErr := o.driverHere(tab)
		switch {
		case hostErr != nil:
			why = "the session this call runs under cannot be found, so whether it is the run's driver cannot be told"
			remedy = "run it from the session that is to drive the run, where " + takeUp
		case alive && here.PID == c.Driver.PID:
			return wholeSave{runID: runID, stated: stated}, nil
		case alive:
			why = "it is driven by " + driverName(*c.Driver) + ", which is alive"
			remedy = "back off: that session steps it on; once it is gone, " + takeUp
		default:
			why = "its driver " + driverName(*c.Driver) + " is gone, and this session has not claimed it"
			remedy = takeUp
		}
	}
	return wholeSave{}, leftAt(id, stated, why, remedy)
}

// leftAt is the refusal of a whole save over a file that states another version: the
// run is left as its file stands, and the refusal names the file and both versions.
func leftAt(id string, stated int, why, remedy string) *Refusal {
	r := contend(StageDriver, "", "", fmt.Sprintf("%s states schema version %d and this abcd writes %d, and %s, so the run is left as its file stands and nothing of it is written",
		StateRelPath(id), stated, SchemaVersion, why), remedy)
	r.leftAtVersion = true
	return r
}

// covers is storeState's check: nil where the file at the run's state path is absent (a
// new run), states this abcd's version, or states the version this right was minted for.
func (w wholeSave) covers(root *os.Root, runID string) error {
	stated, present, err := statedOnDisk(root, runID)
	if err != nil || !present || stated == SchemaVersion || (w.runID == runID && w.stated == stated) {
		return err
	}
	id := termsafe.Sanitize(runID)
	return leftAt(id, stated, "this save was not judged as its driver's",
		"this is a fault in abcd, so report it; the run's driver steps it on with `abcd implement step --run "+id+"`")
}

// saveRun is every whole save's frame: check the tier, take the lock, read the state and
// the version its file states, judge the save (mayWholeSave), let fn change the state,
// and write it only when fn says it changed. A run left is left whole: fn runs only
// once the save is judged, so nothing fn would have done is done.
func saveRun(repoRoot, runID string, o Options, fn func(root *os.Root, st *State) (bool, error)) error {
	return withRun(repoRoot, runID, func(root *os.Root) error {
		st, stated, err := readStatedIn(root, runID)
		if err != nil {
			return err
		}
		right, err := mayWholeSave(root, runID, stated, o)
		if err != nil {
			return err
		}
		changed, err := fn(root, &st)
		if err != nil || !changed {
			return err
		}
		return storeState(root, st, right)
	})
}

// saveFrame is a whole save's frame: saveRun as a session saves (Options.saver), or with a
// claim made first for a drain over a run it takes up (Options.laneSave).
type saveFrame func(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error

// saver is saveRun as a frame.
func (o Options) saver() saveFrame {
	return func(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error {
		return saveRun(repoRoot, runID, o, fn)
	}
}

// laneSave is the frame for a drain's save over a run: one it takes up (claim) is
// claimed first where its file states another version (claimToSave), and every save
// is judged as saveRun judges it.
func (o Options) laneSave(claim bool) saveFrame {
	return func(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error {
		if claim {
			if err := claimToSave(repoRoot, runID, o); err != nil {
				if _, refused := AsRefusal(err); !refused {
					return err
				}
			}
		}
		return saveRun(repoRoot, runID, o, fn)
	}
}

// passingSave is the frame for a drain's save over a parked run that is not one of its
// lanes: a person's build, or an earlier drain's lane. It is claimed first, as laneSave
// claims, and a claim this call made is let go once the save is done (claimInPassing), so
// the run is the drain's for that save alone and a step from the person's own session
// is not refused for as long as the drain's session lives.
func (o Options) passingSave() saveFrame {
	return func(repoRoot, runID string, fn func(root *os.Root, st *State) (bool, error)) error {
		letGo, err := claimInPassing(repoRoot, runID, o)
		if err != nil {
			if _, refused := AsRefusal(err); !refused {
				return err
			}
		}
		defer letGo()
		return saveRun(repoRoot, runID, o, fn)
	}
}

// claimInPassing is claimToSave for a run this session saves once and does not drive on:
// letGo lets go a claim this call recorded, and does nothing where it recorded none (the
// run at this abcd's version, already this session's, or another live session's). Letting
// it go leaves the control record naming no driver, which the run's record says, so the
// next session to step it claims it. A let-go that fails leaves the claim, which then
// stands until this session ends, as any driver's does.
func claimInPassing(repoRoot, runID string, o Options) (letGo func(), err error) {
	letGo = func() {}
	before, _ := runControl(repoRoot, runID)
	if err := claimToSave(repoRoot, runID, o); err != nil {
		return letGo, err
	}
	after, err := runControl(repoRoot, runID)
	if err != nil || after.Driver == nil || (before.Driver != nil && sameClaim(*before.Driver, *after.Driver)) {
		return letGo, nil
	}
	made := *after.Driver
	return func() { _ = letClaimGo(repoRoot, runID, made, o) }, nil
}

// sameClaim reports whether a and b are one claim: one process, claimed at one instant.
func sameClaim(a, b LiveDriver) bool {
	return a.PID == b.PID && a.StartUnix == b.StartUnix && a.At.Equal(b.At)
}

// letClaimGo takes the claim made off the run's control record where it still stands
// there, with a line in the run's record; the file keeps the version it states
// (saveRecord).
func letClaimGo(repoRoot, runID string, made LiveDriver, o Options) error {
	return saveRecord(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil || c.Driver == nil || !sameClaim(*c.Driver, made) {
			return false, err
		}
		c.Driver = nil
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		now := o.now()
		st.Record = append(st.Record, Entry{At: now, Stage: StageDriver,
			Note: fmt.Sprintf("%s let the run go once the drain's save was made: it is not one of the drain's lanes, so the next session to step it drives it", driverName(made))})
		st.UpdatedAt = now
		return true, nil
	})
}

// claimToSave makes this session the driver of a run whose file states another version,
// as a step claims a run (ClaimDriver), so the whole save that follows is its driver's: a
// gone driver is named in the run's record and replaced, one that names none is
// claimed, and one another live session drives is not, the claim refusing at exit 3. A
// run at this abcd's version, or whose file cannot be read, is not claimed: its save is
// judged, or refused by the read, as before.
func claimToSave(repoRoot, runID string, o Options) error {
	if stated, err := statedAt(repoRoot, runID); err != nil || stated == SchemaVersion {
		return nil
	}
	return ClaimDriver(repoRoot, runID, o)
}

// statedAt is the version the run's state file states (readListedStated).
func statedAt(repoRoot, runID string) (int, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return 0, err
	}
	defer root.Close()
	_, stated, err := readListedStated(root, runID)
	return stated, err
}

// wholeSaveLeft is mayWholeSave read outside the lock, for a route that asks before it
// does anything: nil where the run may be saved whole, or where its file cannot be read,
// which the save that follows refuses as it always has.
func wholeSaveLeft(repoRoot, runID string, o Options) *Refusal {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil
	}
	defer root.Close()
	_, stated, err := readListedStated(root, runID)
	if err != nil {
		return nil
	}
	if _, err := mayWholeSave(root, runID, stated, o); err != nil {
		if r, ok := AsRefusal(err); ok {
			return r
		}
	}
	return nil
}

// leftAtVersion is the refusal mayWholeSave gave in err, if it gave one.
func leftAtVersion(err error) (*Refusal, bool) {
	r, ok := AsRefusal(err)
	return r, ok && r.leftAtVersion
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
