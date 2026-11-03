package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// No test in this package reads the real machine's memory pressure unless it
// asks: every wait and every check sees macOS at its normal level, so a
// development machine under pressure does not turn every gate test into a
// memory test.
func init() {
	ReadMemory = func() (machineload.Memory, error) { return macMemory(machineload.MemoryNormal), nil }
}

// macMemory is a macOS reading at level.
func macMemory(level string) machineload.Memory {
	return machineload.Memory{Platform: machineload.MemoryDarwin, Level: level, Raw: level}
}

// linuxMemory is a Linux reading of pct stalled.
func linuxMemory(pct float64) machineload.Memory {
	return machineload.Memory{Platform: machineload.MemoryLinux, StallPct: pct, Raw: "some"}
}

// memorySeq is a reader returning readings in turn, the last repeating, and
// counting its reads.
func memorySeq(readings ...machineload.Memory) (func() (machineload.Memory, error), func() int) {
	var mu sync.Mutex
	i := 0
	return func() (machineload.Memory, error) {
			mu.Lock()
			defer mu.Unlock()
			m := readings[min(i, len(readings)-1)]
			i++
			return m, nil
		}, func() int {
			mu.Lock()
			defer mu.Unlock()
			return i
		}
}

// writeMemoryRecord plants the admission record under home.
func writeMemoryRecord(t *testing.T, home string, r memoryRecord) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(dir, WaitMemoryAdmitFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// readMemoryRecord reads the admission record under home; ok is false when
// there is none.
func readMemoryRecord(t *testing.T, home string) (memoryRecord, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(abcdhome.Path(home), WaitMemoryAdmitFileName))
	if errors.Is(err, os.ErrNotExist) {
		return memoryRecord{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var r memoryRecord
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r, true
}

// TestMemoryFloorDefaults: with no key set the floor is macOS's warning level
// and, on Linux, 10%.
func TestMemoryFloorDefaults(t *testing.T) {
	f, note, err := memoryFloorAt(t.TempDir(), "")
	if err != nil || note != "" {
		t.Fatalf("memoryFloorAt = %+v, %q, %v", f, note, err)
	}
	if f.Level != machineload.MemoryWarn || f.StallPct != 10 || f.LevelSource != "bundled" || f.StallPctSource != "bundled" {
		t.Fatalf("floor = %+v, want warn and 10 from the defaults", f)
	}
}

// TestMemoryFloorMachineKey: the person's machine configuration sets either
// half; a value outside its set is refused naming the key, the file and the
// values it takes; and the keys are no key load.preflights' reader names as
// unknown.
func TestMemoryFloorMachineKey(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2, "memory_floor_macos": "critical", "memory_floor_linux": 25}}`)
	f, note, err := memoryFloorAt(home, "")
	if err != nil || note != "" || f.Level != machineload.MemoryCritical || f.StallPct != 25 || f.LevelSource != "machine" || f.StallPctSource != "machine" {
		t.Fatalf("memoryFloorAt = %+v, %q, %v; want critical and 25 from the machine", f, note, err)
	}
	if n, _, note, err := PreflightSlots(noEnv, home); err != nil || n != 2 || note != "" {
		t.Fatalf("PreflightSlots = %d, %q, %v; the floor's keys read as unknown beside load.preflights", n, note, err)
	}
	file := abcdhome.Display("config.json")
	for _, c := range []struct{ body, key, want string }{
		{`{"load": {"memory_floor_macos": "urgent"}}`, MemoryFloorMacOSKey, `want "warn" or "critical"`},
		{`{"load": {"memory_floor_macos": 2}}`, MemoryFloorMacOSKey, "does not decode"},
		{`{"load": {"memory_floor_linux": 0}}`, MemoryFloorLinuxKey, "want a whole percentage from 1 to 100"},
		{`{"load": {"memory_floor_linux": 101}}`, MemoryFloorLinuxKey, "want a whole percentage from 1 to 100"},
		{`{"load": {"memory_floor_linux": 12.5}}`, MemoryFloorLinuxKey, "does not decode"},
	} {
		writeMachineConfig(t, home, c.body)
		_, _, err := memoryFloorAt(home, "")
		if err == nil || !strings.Contains(err.Error(), c.key) || !strings.Contains(err.Error(), file) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v; want a refusal naming %s, %s and %q", c.body, err, c.key, file, c.want)
		}
		res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
		if err == nil || !strings.Contains(err.Error(), "nothing was waited for") || !strings.Contains(err.Error(), c.key) {
			t.Errorf("%s: the wait = %+v, %v; want a refusal before any wait naming %s", c.body, res.Status, err, c.key)
		}
	}
}

// TestMemoryFloorRepoKeyIgnored: a repository's .abcd/config.json cannot set
// the floor: its key changes nothing, and the note names it as ignored and
// where the floor is set, once.
func TestMemoryFloorRepoKeyIgnored(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(`{"load": {"memory_floor_macos": "critical", "memory_floor_linux": 90}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, note, err := memoryFloorAt(home, repo)
	if err != nil || f.Level != machineload.MemoryWarn || f.StallPct != 10 {
		t.Fatalf("memoryFloorAt = %+v, %v; a repository's key moved the floor", f, err)
	}
	for _, want := range []string{MemoryFloorMacOSKey, MemoryFloorLinuxKey, ".abcd/config.json are ignored", abcdhome.Display("config.json")} {
		if !strings.Contains(note, want) {
			t.Errorf("note lacks %q:\n%s", want, note)
		}
	}
	// The waiter at warn stays shut, as with no repository key, and says so
	// once however many looks it takes.
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Dir, req.Timeout = repo, time.Minute
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryWarn))
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnMemory}) {
		t.Fatalf("wait = %+v, %v; want a timeout on memory", res, err)
	}
	if strings.Count(res.MemoryNote, "ignored") != 1 {
		t.Fatalf("MemoryNote does not say once that the key was ignored:\n%s", res.MemoryNote)
	}
}

// TestWaitMemoryShutsAtFloor: a warn reading against the warn floor keeps the
// gate shut, and the waiting line's facts name the level and the floor; at a
// critical floor the same reading admits.
func TestWaitMemoryShutsAtFloor(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryWarn))
	var said *WaitResult
	req.OnWait = func(r WaitResult) { said = &r }
	req.Timeout = 0
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnMemory}) {
		t.Fatalf("wait = %+v, %v; want shut on memory", res, err)
	}
	if said == nil || said.Memory == nil || said.Memory.WaitText() != "memory pressure warn (floor warn)" {
		t.Fatalf("waiting line facts = %+v", said)
	}
	if res.Hold != nil {
		t.Fatal("a shut gate took the hold")
	}

	home = t.TempDir()
	writeMachineConfig(t, home, `{"load": {"memory_floor_macos": "critical"}}`)
	req = waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryWarn))
	res, err = WaitLoad(req)
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Memory == nil || res.Memory.AtFloor {
		t.Fatalf("wait at a critical floor = %+v, %v; want open at once", res, err)
	}
	// A level abcd does not know reads as critical, so it shuts even the
	// critical floor.
	req = waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(macMemory("mystery"))
	req.Timeout = 0
	if res, _ := WaitLoad(req); res.Status != WaitTimedOut || !slices.Contains(res.Still, WaitOnMemory) {
		t.Fatalf("an unknown level = %+v; want shut", res)
	}
}

// TestWaitMemoryRereadBeforeOpen: memory is read again at the re-check after
// the jitter, so a reading that rises meanwhile keeps the gate shut.
func TestWaitMemoryRereadBeforeOpen(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	read, reads := memorySeq(macMemory(machineload.MemoryNormal), macMemory(machineload.MemoryWarn), macMemory(machineload.MemoryNormal))
	req.ReadMemory = read
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("wait = %+v, %v", res, err)
	}
	if !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) || res.Polls != 4 || reads() != 4 {
		t.Fatalf("waited on %v over %d looks and %d memory reads; want memory, 4 and 4 (free, risen at the re-check, free, open)", res.WaitedOn, res.Polls, reads())
	}
}

// TestWaitMemoryOpensWhenItFalls: warn, then normal, opens as with no floor.
func TestWaitMemoryOpensWhenItFalls(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryWarn), macMemory(machineload.MemoryNormal))
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Hold == nil || len(res.Still) != 0 {
		t.Fatalf("wait = %+v, %v; want open with the hold", res, err)
	}
	if !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) || res.Memory == nil || res.Memory.AtFloor || res.Memory.Level != machineload.MemoryNormal {
		t.Fatalf("waited on %v, memory %+v", res.WaitedOn, res.Memory)
	}
}

// TestWaitMemoryTimeoutNamesMemory: pressure that stays at the floor until
// the timeout ends the wait as a load wait ends, naming memory.
func TestWaitMemoryTimeoutNamesMemory(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryCritical))
	req.Timeout = 2 * time.Minute
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnMemory}) || res.WaitedS < 120 {
		t.Fatalf("wait = %+v, %v; want a timeout after 2m still on memory", res, err)
	}
}

// TestWaitMemoryUnreadableAdmits: a reading that cannot be taken shuts
// nothing; the gate opens and the note says once that memory was not
// checked, and why.
func TestWaitMemoryUnreadableAdmits(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory = func() (machineload.Memory, error) {
		return machineload.Memory{}, &machineload.MemoryUnsupportedError{Why: "abcd reads memory pressure on macOS and Linux only, and this is plan9"}
	}
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Memory != nil {
		t.Fatalf("wait = %+v, %v; want open at once with no reading", res, err)
	}
	want := "the memory floor was not checked: abcd reads memory pressure on macOS and Linux only, and this is plan9"
	if strings.Count(res.MemoryNote, want) != 1 {
		t.Fatalf("MemoryNote does not say once %q:\n%s", want, res.MemoryNote)
	}
	if _, ok := readMemoryRecord(t, home); ok {
		t.Fatal("an unread memory floor wrote the admission record")
	}
}

// TestWaitMemoryLinuxStall: on Linux, 14.2% stalled against the 10% floor
// waits, naming the share and the floor; 9.9% opens.
func TestWaitMemoryLinuxStall(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.ReadMemory, _ = memorySeq(linuxMemory(14.2), linuxMemory(9.9))
	var said *WaitResult
	req.OnWait = func(r WaitResult) { said = &r }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) {
		t.Fatalf("wait = %+v, %v; want open after waiting on memory", res, err)
	}
	if said == nil || said.Memory == nil || said.Memory.WaitText() != "memory stalled 14.2% over 10s (floor 10%)" {
		t.Fatalf("waiting line facts = %+v", said.Memory)
	}
}

// TestWaitMemoryRaceShortcutReadsMemory: with one slot the race gate opens
// without a look, but memory is read once more before it opens, and pressure
// at the floor holds it until it falls.
func TestWaitMemoryRaceShortcutReadsMemory(t *testing.T) {
	home := t.TempDir()
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Race = true
	read, reads := memorySeq(macMemory(machineload.MemoryWarn), macMemory(machineload.MemoryNormal))
	req.ReadMemory = read
	var said *WaitResult
	req.OnWait = func(r WaitResult) { said = &r }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Gate != GateRace || !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) || len(res.Still) != 0 {
		t.Fatalf("wait = %+v, %v; want the race gate open after waiting on memory", res, err)
	}
	if reads() != 2 || said == nil || !slices.Equal(said.Still, []string{WaitOnMemory}) {
		t.Fatalf("%d memory reads, waiting line %+v", reads(), said)
	}
	if want := "one preflight runs at a time, and what it waited on (memory) no longer holds the race stage back"; res.Reason != want {
		t.Fatalf("reason = %q, want %q: a gate that waited did not open at once", res.Reason, want)
	}
}

// fourSlots is an environment admitting four preflights at once.
var fourSlots = slotsEnv("4")

// TestWaitMemorySettleSerialises: two waiters on memory at four slots; the
// pressure falls; the first opens and writes the record; the second stays
// shut on memory until the settle passes, then reads memory again and opens.
// The second waits at the race gate, which the first's hold never shuts, so
// memory alone holds it.
func TestWaitMemorySettleSerialises(t *testing.T) {
	home := t.TempDir()
	writeMemoryRecord(t, home, memoryRecord{PressureAt: waitT0.Add(-time.Minute)})
	clock := &fakeClock{now: waitT0}
	first := waitReq(home, 60000, clock, poll{load: 3})
	first.Getenv = fourSlots
	res, err := WaitLoad(first)
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("first wait = %+v, %v; want open at once", res, err)
	}
	openedAt := clock.Now()
	rec, ok := readMemoryRecord(t, home)
	if !ok || rec.AdmittedPID != 60000 || !rec.AdmittedAt.Equal(openedAt) {
		t.Fatalf("record = %+v; want the first waiter admitted at %s", rec, openedAt)
	}

	clock2 := &fakeClock{now: openedAt.Add(10 * time.Second)}
	second := waitReq(home, 61000, clock2, poll{load: 3})
	second.Getenv, second.Race = fourSlots, true
	read, reads := memorySeq(macMemory(machineload.MemoryNormal))
	second.ReadMemory = read
	var said *WaitResult
	second.OnWait = func(r WaitResult) { said = &r }
	res, err = WaitLoad(second)
	if err != nil || res.Status != WaitOpen || !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) {
		t.Fatalf("second wait = %+v, %v; want open after waiting on memory", res, err)
	}
	if said == nil || !slices.Equal(said.Still, []string{WaitOnMemory}) || said.Memory == nil ||
		said.Memory.WaitText() != "memory settling after pid 60000 opened 10s ago (pressure normal, floor warn)" {
		t.Fatalf("waiting line facts = %+v", said)
	}
	if opened := clock2.Now(); opened.Sub(openedAt) < WaitMemorySettle {
		t.Fatalf("the second opened %s after the first, inside the settle", opened.Sub(openedAt))
	}
	if reads() < 3 {
		t.Fatalf("memory read %d times; want it read again after the settle", reads())
	}
	if rec, _ := readMemoryRecord(t, home); rec.AdmittedPID != 61000 {
		t.Fatalf("record = %+v; want the second waiter admitted", rec)
	}
}

// TestWaitMemorySettleHoldsALatecomer: a waiter that arrives after the
// pressure fell, inside another's settle, waits for it too.
func TestWaitMemorySettleHoldsALatecomer(t *testing.T) {
	home := t.TempDir()
	writeMemoryRecord(t, home, memoryRecord{PressureAt: waitT0.Add(-2 * time.Minute), AdmittedAt: waitT0.Add(-30 * time.Second), AdmittedPID: 59000, AdmittedStartUnix: waitT0.Add(-time.Hour).Unix()})
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 60000, clock, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || !slices.Equal(res.WaitedOn, []string{WaitOnMemory}) {
		t.Fatalf("wait = %+v, %v; want open after the settle", res, err)
	}
	if waited := clock.Now().Sub(waitT0); waited < 90*time.Second {
		t.Fatalf("opened after %s, inside the other's settle", waited)
	}
}

// TestWaitMemoryEpisodeEnds: ten minutes after the pressure was last seen,
// an arrival opens without a settle, and writes nothing.
func TestWaitMemoryEpisodeEnds(t *testing.T) {
	home := t.TempDir()
	planted := memoryRecord{PressureAt: waitT0.Add(-11 * time.Minute), AdmittedAt: waitT0.Add(-30 * time.Second), AdmittedPID: 59000}
	writeMemoryRecord(t, home, planted)
	res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("wait = %+v, %v; want open at once", res, err)
	}
	if rec, _ := readMemoryRecord(t, home); rec.AdmittedPID != 59000 {
		t.Fatalf("record = %+v; a waiter outside the episode wrote itself admitted", rec)
	}
}

// TestWaitMemoryNoRecordWithoutPressure: a machine never under pressure has
// no record, and opens exactly as before the floor.
func TestWaitMemoryNoRecordWithoutPressure(t *testing.T) {
	home := t.TempDir()
	res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.MemoryNote != "" {
		t.Fatalf("wait = %+v, %v", res, err)
	}
	if _, ok := readMemoryRecord(t, home); ok {
		t.Fatal("a wait that never saw pressure wrote the admission record")
	}
}

// TestWaitMemoryAncestorAdmissionDoesNotHold: a waiter whose ancestor is the
// waiter last admitted (`--wait -- make preflight`, whose preflight gates
// itself) was admitted with it: the settle does not hold it, and it does not
// take its ancestor's place in the record.
func TestWaitMemoryAncestorAdmissionDoesNotHold(t *testing.T) {
	home := t.TempDir()
	writeMemoryRecord(t, home, memoryRecord{PressureAt: waitT0.Add(-time.Minute), AdmittedAt: waitT0.Add(-10 * time.Second), AdmittedPID: 59999})
	res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("wait = %+v, %v; want open at once", res, err)
	}
	if rec, _ := readMemoryRecord(t, home); rec.AdmittedPID != 59999 {
		t.Fatalf("record = %+v; the ancestor's admission was replaced", rec)
	}
}

// TestWaitMemoryUnusableRecordAdmits: a record that cannot be read is named
// in the note and judged as none, so it never wedges a run.
func TestWaitMemoryUnusableRecordAdmits(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(abcdhome.Path(home), WaitMemoryAdmitFileName), 0o700); err != nil {
		t.Fatal(err)
	}
	res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("wait = %+v, %v; want open at once", res, err)
	}
	if !strings.Contains(res.MemoryNote, "could not be read") || !strings.Contains(res.MemoryNote, "judged as if there were none") {
		t.Fatalf("MemoryNote = %q", res.MemoryNote)
	}
}

// TestCheckLoadMemoryWarns: the load check reports a reading at the floor as
// a warning beside the load's status, which it leaves unchanged; below the
// floor there is no warning; an unread one is noted.
func TestCheckLoadMemoryWarns(t *testing.T) {
	req := loadRequest(t, loadSnapshot(3), nil)
	req.Home = t.TempDir()
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryWarn))
	res := CheckLoad(req)
	if res.Status != LoadOK || res.Memory == nil || !res.Memory.AtFloor {
		t.Fatalf("check = %+v (memory %+v); want ok with memory at the floor", res.Status, res.Memory)
	}
	if w := res.Memory.WarningText(res.Site); !strings.Contains(w, "MEMORY PRESSURE WARNING (preflight): memory pressure warn (floor warn)") {
		t.Fatalf("warning = %q", w)
	}
	req.ReadMemory, _ = memorySeq(macMemory(machineload.MemoryNormal))
	res = CheckLoad(req)
	if res.Status != LoadOK || res.Memory == nil || res.Memory.AtFloor || res.Memory.WarningText(res.Site) != "" {
		t.Fatalf("check = %+v (memory %+v); want ok with no warning", res.Status, res.Memory)
	}
	req.ReadMemory = func() (machineload.Memory, error) { return machineload.Memory{}, errors.New("sysctl refused") }
	res = CheckLoad(req)
	if res.Status != LoadOK || res.Memory != nil || !strings.Contains(res.MemoryNote, "not checked: sysctl refused") {
		t.Fatalf("check = %+v, note %q", res.Status, res.MemoryNote)
	}
	if _, ok := readMemoryRecord(t, req.Home); ok {
		t.Fatal("the check wrote the admission record")
	}
}
