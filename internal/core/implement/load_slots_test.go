package implement

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// How many preflights may be in their test stages at once is the person's own
// machine setting (iss-2610161639025702): load.preflights in
// ~/.abcd.noindex/config.json, or ABCD_LOAD_PREFLIGHTS for a trial, and one
// by default. No repository file is read for it, so a checkout cannot raise
// the machine's load.

// slotsEnv is an environment holding only ABCD_LOAD_PREFLIGHTS=v.
func slotsEnv(v string) func(string) string {
	return func(k string) string {
		if k == PreflightSlotsEnv {
			return v
		}
		return ""
	}
}

// writeMachineConfig writes ~/.abcd.noindex/config.json under home.
func writeMachineConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestPreflightSlotsDefaultsToOne: with nothing set, one preflight at a time.
func TestPreflightSlotsDefaultsToOne(t *testing.T) {
	n, src, _, err := PreflightSlots(noEnv, t.TempDir())
	if err != nil || n != 1 || src != SlotsDefault {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 1 from the default", n, src, err)
	}
}

// TestPreflightSlotsFromTheMachineConfig: load.preflights in the person's own
// config.json sets it.
func TestPreflightSlotsFromTheMachineConfig(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	n, src, _, err := PreflightSlots(noEnv, home)
	if err != nil || n != 2 || src != SlotsMachine {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 2 from the machine's config.json", n, src, err)
	}
}

// TestPreflightSlotsFromTheEnvironment: ABCD_LOAD_PREFLIGHTS sets it for a
// trial, over the machine's config.json.
func TestPreflightSlotsFromTheEnvironment(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 1}}`)
	n, src, _, err := PreflightSlots(slotsEnv("2"), home)
	if err != nil || n != 2 || src != SlotsEnv {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 2 from %s", n, src, err, PreflightSlotsEnv)
	}
}

// TestPreflightSlotsRefusesABadValue: a value that is not a whole number from
// one to MaxPreflightSlots, from either place, is refused naming where it was
// set, never replaced by the default.
func TestPreflightSlotsRefusesABadValue(t *testing.T) {
	for _, v := range []string{"0", "5", "two", "1.5", "-1"} {
		if _, _, _, err := PreflightSlots(slotsEnv(v), t.TempDir()); err == nil || !strings.Contains(err.Error(), PreflightSlotsEnv) {
			t.Errorf("%s=%q: err = %v, want a refusal naming the variable", PreflightSlotsEnv, v, err)
		}
	}
	for _, body := range []string{
		`{"load": {"preflights": 0}}`,
		`{"load": {"preflights": 5}}`,
		`{"load": {"preflights": "2"}}`,
		`{"load": {"preflights": 1.5}}`,
		`{"load": {"preflights": null}}`,
	} {
		home := t.TempDir()
		writeMachineConfig(t, home, body)
		if _, _, _, err := PreflightSlots(noEnv, home); err == nil || !strings.Contains(err.Error(), "config.json") {
			t.Errorf("%s: err = %v, want a refusal naming config.json", body, err)
		}
	}
}

// TestPreflightSlotsIsCappedAtTwoUntilMeasured: the product thinker approved
// a measured trial of two preflights at once (dcn-2610161814279204), so two is
// the most either place may set; three and four are refused, saying the limit
// rises once a measured trial supports it.
func TestPreflightSlotsIsCappedAtTwoUntilMeasured(t *testing.T) {
	if MaxPreflightSlots != 2 {
		t.Fatalf("MaxPreflightSlots = %d, want 2 until a measured trial supports more", MaxPreflightSlots)
	}
	const why = "the limit rises once a measured trial supports it"
	for _, v := range []string{"3", "4"} {
		_, _, _, err := PreflightSlots(slotsEnv(v), t.TempDir())
		if err == nil || !strings.Contains(err.Error(), PreflightSlotsEnv) || !strings.Contains(err.Error(), why) {
			t.Errorf("%s=%s: err = %v, want a refusal naming the variable and saying %q", PreflightSlotsEnv, v, err, why)
		}
		home := t.TempDir()
		writeMachineConfig(t, home, `{"load": {"preflights": `+v+`}}`)
		_, _, _, err = PreflightSlots(noEnv, home)
		if err == nil || !strings.Contains(err.Error(), "config.json") || !strings.Contains(err.Error(), why) {
			t.Errorf("load.preflights %s: err = %v, want a refusal naming config.json and saying %q", v, err, why)
		}
	}
}

// TestPreflightSlotsReadsPastAnUnreadableFile: a machine config.json that
// cannot be read for the count (group-writable, not valid JSON, a load that is
// not an object, or a key under load nothing reads) does not stop the gate,
// which never read that file before the setting existed: the count is the
// default, one, and a note names the file, the cause and the fix. Only a
// load.preflights that is present and unusable is refused.
func TestPreflightSlotsReadsPastAnUnreadableFile(t *testing.T) {
	file := abcdhome.Display("config.json")
	for _, c := range []struct {
		name, body string
		mode       os.FileMode
		cause      string
	}{
		{"group-writable", `{"load": {"preflights": 2}}`, 0o660, "writable by group or other"},
		{"invalid", `{"load": {"preflights": 2}`, 0o600, "not valid JSON"},
		{"load not an object", `{"load": 5}`, 0o600, "not an object"},
		{"unknown key", `{"load": {"preflight": 2}}`, 0o600, "unknown key load.preflight"},
	} {
		home := t.TempDir()
		writeMachineConfig(t, home, c.body)
		if err := os.Chmod(filepath.Join(abcdhome.Path(home), "config.json"), c.mode); err != nil {
			t.Fatal(err)
		}
		n, src, note, err := PreflightSlots(noEnv, home)
		if err != nil || n != 1 || src != SlotsDefault {
			t.Errorf("%s: PreflightSlots = %d, %q, %v; want 1 from the default", c.name, n, src, err)
			continue
		}
		for _, want := range []string{
			"the machine configuration " + file + " cannot be read for " + PreflightSlotsKey,
			c.cause,
			"so one preflight at a time applies",
			"keep " + file + " a regular file you own that nobody else can write (chmod go-w " + file + ")",
			`whose "load" names no key but "preflights"`,
		} {
			if !strings.Contains(note, want) {
				t.Errorf("%s: note lacks %q:\n%s", c.name, want, note)
			}
		}
		if strings.Contains(note, home) {
			t.Errorf("%s: note carries the home path:\n%s", c.name, note)
		}
	}

	// A key nothing reads beside a usable load.preflights leaves the count
	// as set, and the note names the key.
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2, "preflight": 1}}`)
	n, src, note, err := PreflightSlots(noEnv, home)
	if err != nil || n != 2 || src != SlotsMachine || !strings.Contains(note, "unknown key load.preflight") {
		t.Fatalf("PreflightSlots = %d, %q, %q, %v; want 2 from the machine with a note naming load.preflight", n, src, note, err)
	}

	// The environment is read first, and the file is then not read at all.
	n, src, note, err = PreflightSlots(slotsEnv("2"), home)
	if err != nil || n != 2 || src != SlotsEnv || note != "" {
		t.Fatalf("PreflightSlots = %d, %q, %q, %v; want 2 from the environment and no note", n, src, note, err)
	}
}

// TestWaitReadsPastAnUnreadableMachineConfig: the wait carries the note and
// opens, at either gate, rather than refusing.
func TestWaitReadsPastAnUnreadableMachineConfig(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}`)
	res, err := WaitLoad(waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3}))
	if err != nil || res.Status != WaitOpen || res.Slots != 1 || !strings.Contains(res.SlotsNote, "not valid JSON") {
		t.Fatalf("first gate = %+v, %v; want open with one slot and a note", res, err)
	}
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3})
	req.Race = true
	res, err = WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Slots != 1 || !strings.Contains(res.SlotsNote, "not valid JSON") {
		t.Fatalf("race gate = %+v, %v; want open at once with one slot and a note", res, err)
	}
}

// TestPreflightSlotsIgnoresTheRepository: a checkout's .abcd/config.json is
// never read for it, even when the process runs inside that checkout.
func TestPreflightSlotsIgnoresTheRepository(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(`{"load": {"preflights": 4}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	n, src, _, err := PreflightSlots(noEnv, t.TempDir())
	if err != nil || n != 1 || src != SlotsDefault {
		t.Fatalf("PreflightSlots in a checkout setting 4 = %d, %q, %v; want 1 from the default", n, src, err)
	}
}

// TestWaitAdmitsASecondPreflightWithTwoSlots: with two slots, a preflight's
// first gate opens while one other preflight is in its test stages, and a
// third stays shut while two are.
func TestWaitAdmitsASecondPreflightWithTwoSlots(t *testing.T) {
	one, oneArgs := otherAccountsPreflight(80000)
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: one, args: oneArgs})
	req.Getenv = slotsEnv("2")
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 || res.Slots != 2 || res.SlotsSource != SlotsEnv {
		t.Fatalf("result = %+v, want open at once with 2 slots from the environment", res)
	}

	two, twoArgs := otherAccountsPreflight(81000)
	for k, v := range oneArgs {
		twoArgs[k] = v
	}
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: append(one, two...), args: twoArgs})
	req.Getenv = slotsEnv("2")
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnPreflight}) || !reflect.DeepEqual(res.Preflights, []int{80000, 81000}) {
		t.Fatalf("result = %+v, want a timeout still on [preflight] naming both", res)
	}
}

// TestWaitWithTwoSlotsIsNotShutByARaceRunAlone: with two slots a live race
// run does not by itself shut the first gate (the race gate keeps race stages
// one at a time); the load limit still does.
func TestWaitWithTwoSlotsIsNotShutByARaceRunAlone(t *testing.T) {
	race := []machineload.Proc{raceRun(70000, 1)}
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: race})
	req.Getenv = slotsEnv("2")
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("result = %+v, want open: with two slots a race run alone does not shut the first gate", res)
	}

	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 40, procs: race})
	req.Getenv = slotsEnv("2")
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLoad}) {
		t.Fatalf("result = %+v, want a timeout still on [load]", res)
	}

	// With one slot the race run shuts it, as it always did.
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: race})
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnRaceRun}) || res.Slots != 1 || res.Gate != GateTests {
		t.Fatalf("result = %+v, want a timeout still on [race-run] at the tests gate with 1 slot", res)
	}
}

// TestWaitRefusesAnUnusableSlotSetting: a bad load.preflights refuses the
// wait before any look, naming the file, rather than guessing a count.
func TestWaitRefusesAnUnusableSlotSetting(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 9}}`)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3})
	if _, err := WaitLoad(req); err == nil || !strings.Contains(err.Error(), "nothing was waited for") || !strings.Contains(err.Error(), "load.preflights") {
		t.Fatalf("err = %v, want a refusal naming load.preflights before any wait", err)
	}
}

// TestWaitRaceGateRefusesWhileARaceRunIsLive: with two slots the race gate
// stays shut while another go test -race run is live, and opens once it exits.
func TestWaitRaceGateRefusesWhileARaceRunIsLive(t *testing.T) {
	race := poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}}
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, race, race, poll{load: 3})
	req.Getenv = slotsEnv("2")
	req.Race = true
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnRaceRun}) || res.Gate != GateRace {
		t.Fatalf("result = %+v, want the race gate shut on [race-run], then open", res)
	}
}

// TestWaitRaceGateAdmitsBesideAUnitStage: with two slots the race gate is not
// shut by a preflight in its unit stage or a first gate's waiter; only a race
// run, a race waiter ahead, the load and the hold shut it.
func TestWaitRaceGateAdmitsBesideAUnitStage(t *testing.T) {
	unit, args := otherAccountsPreflight(80000)
	early := []machineload.Proc{
		{PID: 50000, PPID: 1, PGID: 50000, UID: 502, Age: time.Minute, Name: "go"},
		{PID: 50001, PPID: 50000, PGID: 50000, UID: 502, Age: time.Minute, Name: "abcd"},
	}
	args[50000] = []string{"run", "./cmd/abcd", "implement", "load", "--wait"}
	args[50001] = []string{"implement", "load", "--wait"}
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: append(unit, early...), args: args})
	req.Getenv = slotsEnv("2")
	req.Race = true
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("result = %+v, want the race gate open beside a unit stage and a first gate's waiter", res)
	}

	// An earlier race waiter is ahead of it.
	args[50000] = []string{"run", "./cmd/abcd", "implement", "load", "--race-gate"}
	args[50001] = []string{"implement", "load", "--race-gate"}
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: early, args: args})
	req.Getenv = slotsEnv("2")
	req.Race = true
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnWaiter}) || !reflect.DeepEqual(res.Waiters, []int{50001}) {
		t.Fatalf("result = %+v, want a timeout still on [waiter] naming the earlier race waiter 50001", res)
	}

	// The load limit still shuts it.
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 40})
	req.Getenv = slotsEnv("2")
	req.Race = true
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnLoad}) {
		t.Fatalf("result = %+v, want a timeout still on [load]", res)
	}
}

// TestWaitRaceGateOpensAtOnceWithOneSlot: with one slot, the default, a
// preflight's tests run alone, so its race stage follows them at once, as it
// did before the race gate existed: the race gate opens without reading the
// machine, without judging or taking a hold (the preflight's own first gate's
// settle hold among them), and without the settle jitter, however loaded the
// machine and whatever race run is live.
func TestWaitRaceGateOpensAtOnceWithOneSlot(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	// The preflight's own first gate opens and takes its settle hold.
	first, err := WaitLoad(waitReq(home, 59000, clock, poll{load: 3}))
	if err != nil || first.Status != WaitOpen || first.Hold == nil {
		t.Fatalf("first gate = %+v, %v; want open with a hold", first, err)
	}
	holdPath := filepath.Join(abcdhome.Path(home), WaitHoldFileName)
	before, err := os.ReadFile(holdPath)
	if err != nil {
		t.Fatal(err)
	}

	clock = &fakeClock{now: waitT0.Add(30 * time.Second)}
	req := waitReq(home, 60000, clock, poll{load: 40, procs: []machineload.Proc{raceRun(70000, 1)}})
	req.Race = true
	req.Read = func() (machineload.Snapshot, error) {
		t.Fatal("the race gate read the machine with one slot")
		return machineload.Snapshot{}, nil
	}
	req.Sleep = func(d time.Duration) { t.Fatalf("the race gate slept %s with one slot", d) }
	req.Jitter = func() time.Duration { t.Fatal("the race gate drew a settle jitter with one slot"); return 0 }
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || res.Gate != GateRace || res.Slots != 1 || res.Hold != nil || res.Holder != nil ||
		res.Polls != 0 || res.WaitedS != 0 || len(res.WaitedOn) != 0 || res.Reason == "" {
		t.Fatalf("result = %+v, want the race gate open at once with one slot, no look, no hold, and a reason", res)
	}
	after, err := os.ReadFile(holdPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("the first gate's hold changed under the race gate: %v\nbefore %s\nafter  %s", err, before, after)
	}

	// With two slots the same race gate reads the machine and stays shut.
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}})
	req.Getenv = slotsEnv("2")
	req.Race = true
	req.Timeout = time.Minute
	res, err = WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitTimedOut || !reflect.DeepEqual(res.Still, []string{WaitOnRaceRun}) {
		t.Fatalf("result = %+v, want two slots' race gate still on [race-run]", res)
	}
}

// TestPreflightParallelismSharesTheMachine: with two slots each stage's -p is
// at most half the cores.
func TestPreflightParallelismSharesTheMachine(t *testing.T) {
	read := func() (machineload.Snapshot, error) {
		return machineload.Snapshot{Load1: 2, HasLoad: true, Cores: 16}, nil
	}
	if got := PreflightParallelism(noEnv, read, 1); got != 16 {
		t.Fatalf("one slot on an idle 16 cores: -p %d, want 16", got)
	}
	if got := PreflightParallelism(noEnv, read, 2); got != 8 {
		t.Fatalf("two slots on an idle 16 cores: -p %d, want 8", got)
	}
}

// TestWaitStandsBackAtTheFirstGateWithTwoSlots: with two slots the first
// gate's last look still applies (dcn-2610162257477571). Two waiters whose
// start and pid disagree, across accounts, open in one moment; the higher pid
// finds the lower opened beside it and stands back, once. Its next look counts
// the lower pid, running its command, as one unit in its test stages, and with
// a slot still free it opens beside it, long before that command exits; with
// one slot the same waiter waits for the command to exit, as on main.
func TestWaitStandsBackAtTheFirstGateWithTwoSlots(t *testing.T) {
	const (
		early, late = 96274, 50460 // early: higher pid; late: lower pid
		runFor      = 5 * time.Minute
	)
	uids := map[int]uint32{early: 502, late: 501}
	ages := map[int]time.Duration{early: 86 * time.Minute, late: 83 * time.Minute}
	at := waitT0.Add(testJitter)
	cmdAt := at.Add(WaitOpenPause)
	run := func(t *testing.T, slots string) (WaitResult, time.Time) {
		t.Helper()
		clock := &fakeClock{now: waitT0}
		m := &marks{clock: clock}
		req := waitReq(t.TempDir(), early, clock)
		req.Getenv = slotsEnv(slots)
		req.Read = openerRead(clock, early, late, uids, ages, opener{
			child: func(now time.Time) (string, time.Time, bool) {
				if now.Before(cmdAt) {
					return "sleep", at, now.After(at)
				}
				return "sh", at, true
			},
			gone: func(now time.Time) bool { return !now.Before(cmdAt.Add(runFor)) },
		})
		req.ReadArgs = openerArgs(uids, ages, early, late)
		req.Self = machineload.Self{PID: early, UID: uids[early]}
		req.Command = true
		req.Timeout = 30 * time.Minute
		req.Mark = m.mark
		res, err := WaitLoad(req)
		if err != nil {
			t.Fatal(err)
		}
		return res, clock.Now()
	}

	res, done := run(t, "2")
	if res.Status != WaitOpen || res.Backoffs != 1 || res.Slots != 2 {
		t.Fatalf("two slots: %+v; want open with 2 slots after standing back once from the lower pid %d", res, late)
	}
	if !done.Before(cmdAt.Add(runFor)) {
		t.Fatalf("two slots: done at %s, want open beside the lower pid's command, before it exits at %s", done, cmdAt.Add(runFor))
	}

	res, done = run(t, "1")
	if res.Status != WaitOpen || res.Backoffs != 1 || done.Before(cmdAt.Add(runFor)) {
		t.Fatalf("one slot: %+v, done at %s; want it to stand back once and open only once the lower pid's command exits at %s", res, done, cmdAt.Add(runFor))
	}
}

// TestWaitRaceGateNeitherMarksNorLooksWithOneSlot: with one slot the race
// gate opens before any look, so even in the command form it starts no mark,
// takes no last look and never stands back: a preflight's race stage follows
// its tests as on main, where the race stage had no gate.
func TestWaitRaceGateNeitherMarksNorLooksWithOneSlot(t *testing.T) {
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, poll{load: 3})
	req.Race, req.Command = true, true
	req.Read = func() (machineload.Snapshot, error) {
		t.Fatal("the race gate read the machine with one slot")
		return machineload.Snapshot{}, nil
	}
	req.Mark = func() (func(), error) {
		t.Fatal("the race gate marked itself with one slot")
		return nil, nil
	}
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Gate != GateRace || res.Backoffs != 0 || res.Unmark != nil || !clock.Now().Equal(waitT0) {
		t.Fatalf("%+v, err %v, done at %s; want the race gate open at once, unmarked, at %s", res, err, clock.Now(), waitT0)
	}
}

// TestWaitFirstGateNeverStandsBackFromARaceWaiter: a first gate's opener with
// one slot stands back only from a first gate's waiter that opened beside it,
// as on main, whose abcd reads no race waiter at all. A race waiter with the
// lower pid that started its race stage in that same moment passed its own
// first gate earlier, and the opener goes on; a first gate's waiter in its
// place still stands it back.
func TestWaitFirstGateNeverStandsBackFromARaceWaiter(t *testing.T) {
	stage := func(gate ...string) poll {
		words := append([]string{"implement", "load"}, gate...)
		return poll{load: 3, procs: []machineload.Proc{
			{PID: 40000, PPID: 1, PGID: 40000, UID: 502, Age: 9 * time.Minute, Name: "make"},
			{PID: 40001, PPID: 40000, PGID: 40000, UID: 502, Age: time.Minute, Name: "sh"},
			{PID: 40002, PPID: 40001, PGID: 40000, UID: 502, Age: time.Minute, Name: "go"},
			{PID: 40003, PPID: 40002, PGID: 40000, UID: 502, Age: time.Minute, Name: "abcd"},
			// Its command, started as this waiter's last look reads the table.
			{PID: 40004, PPID: 40003, PGID: 40000, UID: 502, Age: -(testJitter + WaitOpenPause), Name: "go"},
		}, args: map[int][]string{
			40000: {"preflight"},
			40002: append([]string{"run", "./cmd/abcd"}, words...),
			40003: words,
			40004: {"test", "-race", "./internal/..."},
		}}
	}
	for _, c := range []struct {
		name     string
		gate     []string
		backoffs int
	}{
		{"a race waiter", []string{"--race-gate", "--timeout", "2h"}, 0},
		{"a first gate's waiter", []string{"--wait", "--", "make", "preflight"}, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			clock := &fakeClock{now: waitT0}
			req := waitReq(t.TempDir(), 60000, clock, poll{load: 3}, poll{load: 3}, stage(c.gate...))
			req.Command = true
			res, err := WaitLoad(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.Backoffs != c.backoffs {
				t.Fatalf("%+v; want %d back-offs from %s with the lower pid that opened beside it", res, c.backoffs, c.name)
			}
			if c.backoffs == 0 && (res.Status != WaitOpen || res.Slots != 1) {
				t.Fatalf("%+v; want open with one slot", res)
			}
		})
	}
}

// TestPreflightSlotsIgnoresAHomeInsideAWorkingTree: load.preflights in a HOME
// that lies inside a git working tree is not read as the machine's layer, by
// the rule layered.RootsFor applies: one preflight at a time, and the note
// says why.
func TestPreflightSlotsIgnoresAHomeInsideAWorkingTree(t *testing.T) {
	tree := t.TempDir()
	if err := os.Mkdir(filepath.Join(tree, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tree, "home")
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	n, src, note, err := PreflightSlots(noEnv, home)
	if err != nil || n != 1 || src != SlotsDefault || !strings.Contains(note, "HOME lies inside a git working tree") {
		t.Fatalf("PreflightSlots = %d, %q, %q, %v; want 1 from the default with a note saying HOME lies inside a working tree", n, src, note, err)
	}
}
