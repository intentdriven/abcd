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
	n, src, err := PreflightSlots(noEnv, t.TempDir())
	if err != nil || n != 1 || src != SlotsDefault {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 1 from the default", n, src, err)
	}
}

// TestPreflightSlotsFromTheMachineConfig: load.preflights in the person's own
// config.json sets it.
func TestPreflightSlotsFromTheMachineConfig(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 2}}`)
	n, src, err := PreflightSlots(noEnv, home)
	if err != nil || n != 2 || src != SlotsMachine {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 2 from the machine's config.json", n, src, err)
	}
}

// TestPreflightSlotsFromTheEnvironment: ABCD_LOAD_PREFLIGHTS sets it for a
// trial, over the machine's config.json.
func TestPreflightSlotsFromTheEnvironment(t *testing.T) {
	home := t.TempDir()
	writeMachineConfig(t, home, `{"load": {"preflights": 1}}`)
	n, src, err := PreflightSlots(slotsEnv("2"), home)
	if err != nil || n != 2 || src != SlotsEnv {
		t.Fatalf("PreflightSlots = %d, %q, %v; want 2 from %s", n, src, err, PreflightSlotsEnv)
	}
}

// TestPreflightSlotsRefusesABadValue: a value that is not a whole number from
// one to MaxPreflightSlots, from either place, is refused naming where it was
// set, never replaced by the default; so is an unknown key under load.
func TestPreflightSlotsRefusesABadValue(t *testing.T) {
	for _, v := range []string{"0", "5", "two", "1.5", "-1"} {
		if _, _, err := PreflightSlots(slotsEnv(v), t.TempDir()); err == nil || !strings.Contains(err.Error(), PreflightSlotsEnv) {
			t.Errorf("%s=%q: err = %v, want a refusal naming the variable", PreflightSlotsEnv, v, err)
		}
	}
	for _, body := range []string{
		`{"load": {"preflights": 0}}`,
		`{"load": {"preflights": 5}}`,
		`{"load": {"preflights": "2"}}`,
		`{"load": {"preflights": 1.5}}`,
		`{"load": {"preflight": 2}}`,
	} {
		home := t.TempDir()
		writeMachineConfig(t, home, body)
		if _, _, err := PreflightSlots(noEnv, home); err == nil || !strings.Contains(err.Error(), "config.json") {
			t.Errorf("%s: err = %v, want a refusal naming config.json", body, err)
		}
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
	n, src, err := PreflightSlots(noEnv, t.TempDir())
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

// TestWaitRaceGateRefusesWhileARaceRunIsLive: the race gate stays shut while
// another go test -race run is live, and opens once it exits.
func TestWaitRaceGateRefusesWhileARaceRunIsLive(t *testing.T) {
	race := poll{load: 3, procs: []machineload.Proc{raceRun(70000, 1)}}
	clock := &fakeClock{now: waitT0}
	req := waitReq(t.TempDir(), 60000, clock, race, race, poll{load: 3})
	req.Race = true
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || !reflect.DeepEqual(res.WaitedOn, []string{WaitOnRaceRun}) || res.Gate != GateRace {
		t.Fatalf("result = %+v, want the race gate shut on [race-run], then open", res)
	}
}

// TestWaitRaceGateAdmitsBesideAUnitStage: the race gate is not shut by a
// preflight in its unit stage, a first gate's waiter or the slot count; only
// a race run, a race waiter ahead, the load and the hold shut it.
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
	req.Race = true
	res, err := WaitLoad(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != WaitOpen || len(res.WaitedOn) != 0 {
		t.Fatalf("result = %+v, want the race gate open beside a unit stage and a first gate's waiter", res)
	}

	// An earlier race waiter is ahead of it.
	args[50000] = []string{"run", "./cmd/abcd", "implement", "load", "--wait", "--race"}
	args[50001] = []string{"implement", "load", "--wait", "--race"}
	clock = &fakeClock{now: waitT0}
	req = waitReq(t.TempDir(), 60000, clock, poll{load: 3, procs: early, args: args})
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
