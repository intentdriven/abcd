package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The memory floor (itd-2611012048006057, spc-2611020732419632). On
// 2026-11-01 a 128 GB, 16-core development machine ran out of application
// memory while several agent sessions drained, built and reviewed at once;
// its counters showed about 155 GiB written to swap over five days of
// uptime. The gate read the load, the process table and free disk, but never
// memory.
//
// So the waiter (WaitLoad) keeps a gate shut, at either gate, while the
// operating system reports memory pressure at or above the person's floor
// (memoryShut): macOS's own pressure level, or on Linux the share of the
// last ten seconds in which some task stalled waiting for memory
// (machineload.ReadMemory). It reads memory at every look and once more at
// the re-check before it opens, and before the one-slot race gate opens at
// once. It waits rather than refuses, as the disk floor does (load_disk.go):
// memory pressure falls as other work finishes, so the wait is bounded by the
// gate's own timeout and ends as a load wait ends, naming memory. The load
// check (CheckLoad) reports the same reading as a warning and never fails.
//
// A reading abcd cannot take (another platform, a Linux kernel without the
// pressure stall information, or a read that fails) shuts nothing: the gate
// admits as it did before the floor existed and says once that memory was not
// checked, and why. That deliberately differs from WaitOnUnreadable, which
// shuts the gate on an unreadable load or process table: the person ruled
// that an unread memory floor must never wedge a run.
//
// When the pressure falls, every waiter watching it would open at the one
// look that sees it fall. So, for WaitMemoryEpisode after any waiter last saw
// the pressure at or above the floor, the gate admits one run per
// WaitMemorySettle, through a record under ~/.abcd.noindex/
// (WaitMemoryAdmitFileName): the stage just admitted shows its own memory use
// before memory is read for the next. A machine never under pressure has no
// record and behaves exactly as before the floor.
//
// The floor is the person's own machine setting, read only from
// ~/.abcd.noindex/config.json, never from a repository: a repository's key is
// named as ignored and never used (memoryFloorAt).

// The machine configuration keys of the memory floor.
const (
	MemoryFloorMacOSKey = "load.memory_floor_macos"
	MemoryFloorLinuxKey = "load.memory_floor_linux"
)

// The floors that apply when the person sets none: macOS's warning level, and
// on Linux 10% of the last ten seconds stalled.
const (
	DefaultMemoryFloorMacOS = machineload.MemoryWarn
	DefaultMemoryFloorLinux = 10
)

// loadKeys are every key the machine configuration's `load` object takes, each
// claimed by every reader of it, so a key under `load` that nothing reads is
// still named and none of these is.
var loadKeys = []string{"preflights", "narrow_runs", "memory_floor_macos", "memory_floor_linux"}

// WaitOnMemory: the system reports memory pressure at or above the floor, or,
// after pressure was last seen, another waiter opened within the settle.
const WaitOnMemory = "memory"

// The memory admission's intervals. They are variables only so the front
// door can read them into its help; nothing reassigns them.
var (
	// WaitMemorySettle is how long, after a waiter opened inside an episode,
	// every other waiter stays shut on memory: the same as WaitSettle, long
	// enough for the stage admitted to show its own memory use.
	WaitMemorySettle = WaitSettle
	// WaitMemoryEpisode is how long after any waiter last saw the pressure at
	// or above the floor the gate admits one run per settle; past it the gate
	// admits as it does with no floor.
	WaitMemoryEpisode = 10 * time.Minute
)

// WaitMemoryAdmitFileName is the memory admission record under
// ~/.abcd.noindex/, and waitMemoryLockFileName the lock every write of it is
// taken under.
const (
	WaitMemoryAdmitFileName = "load-memory-admit.json"
	waitMemoryLockFileName  = "load-memory-admit.lock"
)

// maxMemoryRecordBytes caps the admission record.
const maxMemoryRecordBytes = 4 << 10

// ReadMemory reads the machine's memory pressure where a request names no
// reader of its own (WaitRequest.ReadMemory, LoadRequest.ReadMemory). It is a
// variable only so tests replace it: the core's and the front door's own
// tests by default, so none reads the real machine's pressure unless it asks.
var ReadMemory = machineload.ReadMemory

// MemoryFloor is the floor the gate judges memory against, and where each
// half came from (layered.Layer's names: "bundled" or "machine").
type MemoryFloor struct {
	// Level is the macOS floor: machineload.MemoryWarn or MemoryCritical.
	Level       string `json:"level"`
	LevelSource string `json:"level_source"`
	// StallPct is the Linux floor, a whole percentage from 1 to 100.
	StallPct       int    `json:"stall_pct"`
	StallPctSource string `json:"stall_pct_source"`
}

// defaultMemoryFloor is the floor with no key set.
func defaultMemoryFloor() MemoryFloor {
	return MemoryFloor{Level: DefaultMemoryFloorMacOS, LevelSource: layered.Bundled.String(),
		StallPct: DefaultMemoryFloorLinux, StallPctSource: layered.Bundled.String()}
}

// memoryShut is the one judgement of a reading against a floor, shared by the
// waiter and the check: on macOS the level is at or above the floor's level,
// on Linux the stalled share is at or above the floor's percentage.
func memoryShut(m machineload.Memory, f MemoryFloor) bool {
	switch m.Platform {
	case machineload.MemoryDarwin:
		return machineload.MemoryLevelRank(m.Level) >= machineload.MemoryLevelRank(f.Level)
	case machineload.MemoryLinux:
		return m.StallPct >= float64(f.StallPct)
	}
	return false
}

// memoryFloorAt resolves the memory floor for the account whose home is home,
// from the machine configuration alone (layered.MachineRoots), as
// PreflightSlots reads load.preflights. A value outside its set is an error
// naming the key, the file and the values it takes: the gate does not guess a
// floor. A machine configuration that cannot be read at all is read past,
// with the default floor and a note naming the file, as PreflightSlots reads
// past it. repo is the checkout the waiter runs in, "" for none: a key for
// the floor in its .abcd/config.json is never used, and the note says it is
// ignored and where the floor is set.
func memoryFloorAt(home, repo string) (MemoryFloor, string, error) {
	floor := defaultMemoryFloor()
	var notes []string
	if ignored := repoMemoryKeys(repo); len(ignored) > 0 {
		notes = append(notes, fmt.Sprintf("%s in %s %s ignored: the memory floor is a machine setting, read only from %s, never from a repository",
			strings.Join(ignored, " and "), layered.Config.RepoOrigin(), isAre(len(ignored)), layered.Config.MachineOrigin()))
	}
	note := func() string { return strings.Join(notes, "; ") }
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil || h == "" {
			return floor, note(), nil
		}
		home = h
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		// PreflightSlots names the refused home; the wait refuses it before
		// reading this (WaitLoad).
		return floor, note(), nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err == nil {
		_, err = stack.Lookup(MemoryFloorMacOSKey)
	}
	if err != nil {
		notes = append(notes, fmt.Sprintf("the machine configuration %s cannot be read for the memory floor (%v), so the default floor applies (%s on macOS, %d%% on Linux)",
			layered.Config.MachineOrigin(), err, DefaultMemoryFloorMacOS, DefaultMemoryFloorLinux))
		return floor, note(), nil
	}
	// An unknown key under load is named by PreflightSlots, which every gate
	// reads at the same look; the claim here lists every key so neither
	// reader names the other's.
	_ = stack.Claim("load", loadKeys...)
	level, err := layered.Get(stack, MemoryFloorMacOSKey, DefaultMemoryFloorMacOS, func(s string) error {
		if s != machineload.MemoryWarn && s != machineload.MemoryCritical {
			return fmt.Errorf("want %q or %q", machineload.MemoryWarn, machineload.MemoryCritical)
		}
		return nil
	})
	if err != nil {
		return MemoryFloor{}, note(), fmt.Errorf("the load gate stops rather than guess the memory floor: %w", err)
	}
	pct, err := layered.Get(stack, MemoryFloorLinuxKey, DefaultMemoryFloorLinux, func(n int) error {
		if n < 1 || n > 100 {
			return errors.New("want a whole percentage from 1 to 100")
		}
		return nil
	})
	if err != nil {
		return MemoryFloor{}, note(), fmt.Errorf("the load gate stops rather than guess the memory floor: %w", err)
	}
	floor.Level, floor.LevelSource = level.V, level.Layer.String()
	floor.StallPct, floor.StallPctSource = pct.V, pct.Layer.String()
	return floor, note(), nil
}

// repoMemoryKeys names the memory floor's keys a checkout's .abcd/config.json
// holds. Its file is read alone (no machine layer), and one it cannot read
// names nothing: whatever it holds, it sets no floor.
func repoMemoryKeys(repo string) []string {
	if repo == "" {
		return nil
	}
	stack, err := layered.Load(layered.Config, layered.Roots{Repo: repo, HomeRefused: true})
	if err != nil {
		return nil
	}
	var keys []string
	for _, k := range []string{MemoryFloorMacOSKey, MemoryFloorLinuxKey} {
		if found, err := stack.Lookup(k); err == nil && len(found) > 0 {
			keys = append(keys, k)
		}
	}
	return keys
}

// isAre agrees a verb with a count.
func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

// MemorySettle is a waiter held only by the settle: the waiter it follows,
// and how long ago that one opened.
type MemorySettle struct {
	PID  int     `json:"pid"`
	AgoS float64 `json:"ago_s"`
}

// MemoryCheck is one memory reading judged against the floor, as the waiter
// and the check report it.
type MemoryCheck struct {
	// Platform is machineload.MemoryDarwin or MemoryLinux.
	Platform string `json:"platform"`
	// Level and FloorLevel are macOS's reading and floor; StallPct and
	// FloorStallPct Linux's.
	Level         string  `json:"level,omitempty"`
	StallPct      float64 `json:"stall_pct"`
	Raw           string  `json:"raw"`
	FloorLevel    string  `json:"floor_level,omitempty"`
	FloorStallPct int     `json:"floor_stall_pct,omitempty"`
	// AtFloor is true when the reading is at or above the floor.
	AtFloor bool `json:"at_floor"`
	// Settle is the waiter this one follows, when only the settle held it.
	Settle *MemorySettle `json:"settle,omitempty"`
}

// judgeMemory is m judged against f.
func judgeMemory(m machineload.Memory, f MemoryFloor) MemoryCheck {
	c := MemoryCheck{Platform: m.Platform, Raw: termsafe.Sanitize(m.Raw), AtFloor: memoryShut(m, f)}
	if m.Platform == machineload.MemoryLinux {
		c.StallPct, c.FloorStallPct = m.StallPct, f.StallPct
	} else {
		c.Level, c.FloorLevel = termsafe.Sanitize(m.Level), f.Level
	}
	return c
}

// ReadingText is the reading and its floor as a waiting line names them:
// "memory pressure warn (floor warn)", or "memory stalled 14.2% over 10s
// (floor 10%)".
func (c MemoryCheck) ReadingText() string {
	if c.Platform == machineload.MemoryLinux {
		return fmt.Sprintf("memory stalled %s over 10s (floor %d%%)", pctText(c.StallPct), c.FloorStallPct)
	}
	return fmt.Sprintf("memory pressure %s (floor %s)", c.Level, c.FloorLevel)
}

// WaitText is what holds the gate shut on memory: the reading, or, when only
// the settle holds it, the waiter it follows beside the reading:
// "memory settling after pid 4242 opened 30s ago (pressure normal, floor warn)".
func (c MemoryCheck) WaitText() string {
	if c.AtFloor || c.Settle == nil {
		return c.ReadingText()
	}
	reading := fmt.Sprintf("pressure %s, floor %s", c.Level, c.FloorLevel)
	if c.Platform == machineload.MemoryLinux {
		reading = fmt.Sprintf("stalled %s over 10s, floor %d%%", pctText(c.StallPct), c.FloorStallPct)
	}
	ago := time.Duration(c.Settle.AgoS * float64(time.Second)).Round(time.Second)
	return fmt.Sprintf("memory settling after pid %d opened %s ago (%s)", c.Settle.PID, ago, reading)
}

// WarningText is the load check's warning when the reading is at or above the
// floor, "" otherwise. The check never fails for it.
func (c *MemoryCheck) WarningText(site string) string {
	if c == nil || !c.AtFloor {
		return ""
	}
	return fmt.Sprintf("MEMORY PRESSURE WARNING (%s): %s; the load gate holds every test stage until it falls below the floor (%s or %s in %s).\n",
		site, c.ReadingText(), MemoryFloorMacOSKey, MemoryFloorLinuxKey, layered.Config.MachineOrigin())
}

// pctText renders a stalled share to one decimal.
func pctText(p float64) string { return strconv.FormatFloat(p, 'f', 1, 64) + "%" }

// memoryNotChecked is the note for a reading that could not be taken.
func memoryNotChecked(err error) string {
	return "the memory floor was not checked: " + termsafe.Sanitize(err.Error()) + "; the gate does not wait on a memory reading it cannot take"
}

// checkMemory is the load check's memory reading: the reading judged against
// the floor, or nil and a note saying why there is none.
func checkMemory(read func() (machineload.Memory, error), home, repo string) (*MemoryCheck, string) {
	if read == nil {
		read = ReadMemory
	}
	floor, note, err := memoryFloorAt(home, repo)
	var notes []string
	if note != "" {
		notes = append(notes, note)
	}
	if err != nil {
		floor = defaultMemoryFloor()
		notes = append(notes, termsafe.Sanitize(err.Error())+"; the check judged by the default floor")
	}
	m, err := read()
	if err != nil {
		return nil, strings.Join(append(notes, memoryNotChecked(err)), "; ")
	}
	c := judgeMemory(m, floor)
	return &c, strings.Join(notes, "; ")
}

// memoryRecord is the admission record (WaitMemoryAdmitFileName): when any
// waiter last read memory at or above the floor, and the last waiter admitted
// while the episode lasted.
type memoryRecord struct {
	PressureAt        time.Time `json:"pressure_at"`
	AdmittedAt        time.Time `json:"admitted_at"`
	AdmittedPID       int       `json:"admitted_pid,omitempty"`
	AdmittedStartUnix int64     `json:"admitted_start_unix,omitempty"`
}

// memoryClockSlack is how far in the future a record's time may lie and
// still be read: a time further ahead holds nothing back, so a record
// written under a wrong clock cannot keep the gate shut.
const memoryClockSlack = time.Minute

// within reports whether x, read at t, is less than span old.
func within(x, t time.Time, span time.Duration) bool {
	age := t.Sub(x)
	return !x.IsZero() && age > -memoryClockSlack && age < span
}

// inEpisode reports whether the record's last pressure is within the episode
// at t.
func (r memoryRecord) inEpisode(t time.Time) bool { return within(r.PressureAt, t, WaitMemoryEpisode) }

// admitID is a waiter as the record names it: its pid, and its start time in
// Unix seconds, zero when the process table could not give it.
type admitID struct {
	pid   int
	start int64
}

// is reports whether the record's admitted waiter is w: the same pid, and,
// where both start times are known, the same start within holdStartSlack.
func (r memoryRecord) is(w admitID) bool {
	if r.AdmittedPID != w.pid {
		return false
	}
	if r.AdmittedStartUnix == 0 || w.start == 0 {
		return true
	}
	d := r.AdmittedStartUnix - w.start
	return d <= holdStartSlack && -d <= holdStartSlack
}

// readMemoryRecordIn reads the admission record through root. An absent
// record is the zero record; one that is not a regular file, too large or
// unreadable is an error, and the caller judges as if there were none.
func readMemoryRecordIn(root *os.Root) (memoryRecord, error) {
	fi, err := root.Lstat(WaitMemoryAdmitFileName)
	if errors.Is(err, os.ErrNotExist) {
		return memoryRecord{}, nil
	}
	if err != nil {
		return memoryRecord{}, err
	}
	if !fi.Mode().IsRegular() {
		return memoryRecord{}, errors.New("it is not a regular file")
	}
	f, err := root.OpenFile(WaitMemoryAdmitFileName, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return memoryRecord{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxMemoryRecordBytes+1))
	if err != nil {
		return memoryRecord{}, err
	}
	if len(raw) > maxMemoryRecordBytes {
		return memoryRecord{}, fmt.Errorf("it is larger than %d KiB", maxMemoryRecordBytes>>10)
	}
	var r memoryRecord
	if err := json.Unmarshal(raw, &r); err != nil {
		return memoryRecord{}, errors.New("it is not a valid record")
	}
	return r, nil
}

// updateMemoryRecordIn reads the record, lets change decide on it, and, when
// change says so, writes the result back atomically: a read-modify-write of
// both fields under one exclusive lock, so a waiter writing pressure_at never
// writes back an admitted_at older than one another waiter just wrote. A
// record it cannot read is replaced. A lock it cannot take within
// waitLockTimeout, inside one poll, is an error, as is a write that fails.
func updateMemoryRecordIn(root *os.Root, change func(r *memoryRecord, readErr error) (write bool)) error {
	return fsutil.WithFileLockIn(root, waitMemoryLockFileName, waitLockTimeout, func() error {
		r, readErr := readMemoryRecordIn(root)
		if !change(&r, readErr) {
			return nil
		}
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		return fsutil.WriteFileAtomicInRoot(root, WaitMemoryAdmitFileName, append(raw, '\n'), abcdhome.FileMode)
	})
}

// memoryRecordTrouble is the note for a record that could not be read or
// written.
func memoryRecordTrouble(what string, err error) string {
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		err = fmt.Errorf("its lock %s was held by another waiter past %s", abcdhome.Display(waitMemoryLockFileName), waitLockTimeout)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		err = fmt.Errorf("its lock %s is not a regular file", abcdhome.Display(waitMemoryLockFileName))
	}
	return fmt.Sprintf("the memory admission record %s could not be %s (%s); the gate judged as if there were none",
		abcdhome.Display(WaitMemoryAdmitFileName), what, termsafe.Sanitize(err.Error()))
}

// memoryGate is one wait's memory condition: its reader, the floor the last
// look read, and the notes said so far.
type memoryGate struct {
	read  func() (machineload.Memory, error)
	floor MemoryFloor
	notes []string
}

// note records s in res.MemoryNote, once.
func (g *memoryGate) note(res *WaitResult, s string) {
	if s == "" {
		return
	}
	for _, have := range g.notes {
		if have == s {
			return
		}
	}
	g.notes = append(g.notes, s)
	res.MemoryNote = strings.Join(g.notes, "; ")
}

// look reads memory at t for the waiter self and reports whether memory
// holds its gate shut: the reading is at or above the floor (and the record's
// pressure_at is written), or, inside an episode, another waiter opened
// within the settle. ancestor reports whether a pid is this waiter's
// ancestor, whose admission admitted the work this waiter is part of. A
// reading it cannot take shuts nothing, and nor does a record it cannot read
// or write; each is noted.
func (g *memoryGate) look(root *os.Root, res *WaitResult, t time.Time, self admitID, ancestor func(int) bool) bool {
	m, err := g.read()
	if err != nil {
		res.Memory = nil
		g.note(res, memoryNotChecked(err))
		return false
	}
	c := judgeMemory(m, g.floor)
	res.Memory = &c
	if c.AtFloor {
		err := updateMemoryRecordIn(root, func(r *memoryRecord, _ error) bool {
			if r.PressureAt.Before(t) || !within(r.PressureAt, t, WaitMemoryEpisode) {
				r.PressureAt = t
			}
			return true
		})
		if err != nil {
			g.note(res, memoryRecordTrouble("written", err))
		}
		return true
	}
	r, err := readMemoryRecordIn(root)
	if err != nil {
		g.note(res, memoryRecordTrouble("read", err))
		return false
	}
	if s := settling(r, t, self, ancestor); s != nil {
		c.Settle = s
		return true
	}
	return false
}

// settling is the settle that holds the waiter self shut at t, nil when none
// does: inside an episode, another waiter, not self and not its ancestor,
// opened less than WaitMemorySettle ago.
func settling(r memoryRecord, t time.Time, self admitID, ancestor func(int) bool) *MemorySettle {
	if !r.inEpisode(t) || !within(r.AdmittedAt, t, WaitMemorySettle) || r.is(self) || (ancestor != nil && ancestor(r.AdmittedPID)) {
		return nil
	}
	return &MemorySettle{PID: r.AdmittedPID, AgoS: max(t.Sub(r.AdmittedAt), 0).Seconds()}
}

// admit is the waiter's last step before it opens: under the record's lock it
// judges the settle again, and inside an episode writes itself as admitted,
// before it opens and before its queue entry or hold lets a later waiter look.
// It reports true when another waiter opened within the settle meanwhile, and
// the gate stays shut. Outside an episode it writes nothing, and a waiter
// whose ancestor is the one admitted does not take its place. A record it
// cannot read or write holds nothing back, and is noted.
func (g *memoryGate) admit(root *os.Root, res *WaitResult, t time.Time, self admitID, ancestor func(int) bool) bool {
	if res.Memory == nil {
		// The reading could not be taken: nothing is judged, and nothing
		// written, as with no floor.
		return false
	}
	blocked := false
	err := updateMemoryRecordIn(root, func(r *memoryRecord, readErr error) bool {
		if readErr != nil {
			g.note(res, memoryRecordTrouble("read", readErr))
			*r = memoryRecord{}
			return false
		}
		if !r.inEpisode(t) {
			return false
		}
		if s := settling(*r, t, self, ancestor); s != nil {
			res.Memory.Settle, blocked = s, true
			return false
		}
		// Asked last, since the shortcut's ancestor test reads the table.
		if within(r.AdmittedAt, t, WaitMemorySettle) && !r.is(self) && ancestor != nil && ancestor(r.AdmittedPID) {
			return false
		}
		r.AdmittedAt, r.AdmittedPID, r.AdmittedStartUnix = t, self.pid, self.start
		return true
	})
	if err != nil {
		g.note(res, memoryRecordTrouble("written", err))
		return false
	}
	return blocked
}

// shortcutIdentity is the waiter as the one-slot race shortcut judges its
// memory admission, which reads no process table of its own: its pid, and an
// ancestor test that reads the table once, the first time it is asked. A
// table it cannot read finds no ancestor, and a start it cannot read is
// zero, so the record's waiter is told from this one by pid alone.
func shortcutIdentity(req WaitRequest) (admitID, func(int) bool) {
	self := admitID{pid: req.Self.PID}
	var snap machineload.Snapshot
	read := false
	ancestor := func(pid int) bool {
		if !read {
			read = true
			snap, _ = req.Read()
		}
		return snap.HasProcs && machineload.Descends(snap.Procs, req.Self.PID, pid)
	}
	return self, ancestor
}
