package machineload

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// The fixtures below are built from the recorded incidents and the run's own
// telemetry, as Go values, so no test leaves a burner running.

const (
	callerUID = 501
	otherUID  = 502
	rootUID   = 0
)

// caller is the check's own invocation: the check (pid 50000) run by make, run
// by the pre-push hook's shell, run by the agent host, run by launchd. The check
// shares make's process group; the agent host leads its own.
var caller = Self{PID: 50000, UID: callerUID}

func callerChain() []Proc {
	return []Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: rootUID, Age: 60 * time.Hour, CPU: 30 * time.Minute, Name: "launchd"},
		{PID: 49000, PPID: 1, PGID: 49000, UID: callerUID, Age: 50 * time.Hour, CPU: 50 * time.Hour, Name: "node"},
		{PID: 49990, PPID: 49000, PGID: 49990, UID: callerUID, Age: time.Minute, CPU: time.Second, Name: "zsh"},
		{PID: 49999, PPID: 49990, PGID: 49990, UID: callerUID, Age: 50 * time.Second, CPU: time.Second, Name: "make"},
		{PID: 50000, PPID: 49999, PGID: 49990, UID: callerUID, Age: time.Second, CPU: 100 * time.Millisecond, Name: "abcd"},
	}
}

func snapshot(load1 float64, procs ...Proc) Snapshot {
	return Snapshot{Load1: load1, Load5: load1, Load15: load1, HasLoad: true, Cores: 16, Procs: append(callerChain(), procs...), HasProcs: true}
}

// burner is a process that has run at the given share of a core for age.
func burner(pid, pgid int, uid uint32, name string, age time.Duration, share float64) Proc {
	return Proc{PID: pid, PPID: 1, PGID: pgid, UID: uid, Age: age, CPU: time.Duration(float64(age) * share), Name: name}
}

// TestMechanismFlagsIncidentOne: the 2026-09-21 incident, eight of the caller's
// `yes` burners in one group, two days seven hours old at 99%, whose launching
// shell is long gone. All eight are named with pid, group, age and share, and the
// group form of the remedy is offered, because every live member is named.
func TestMechanismFlagsIncidentOne(t *testing.T) {
	age := 55 * time.Hour
	var procs []Proc
	for pid := 41233; pid < 41241; pid++ {
		procs = append(procs, burner(pid, 41230, callerUID, "yes", age, 0.99))
	}
	v := Classify(snapshot(12.3, procs...), DefaultLimits(16), caller)

	if !slices.Equal(v.Triggers, []string{TriggerStray}) {
		t.Fatalf("triggers = %v, want [stray]", v.Triggers)
	}
	if len(v.Own) != 8 || v.OwnMore != 0 {
		t.Fatalf("own = %d (+%d more), want 8", len(v.Own), v.OwnMore)
	}
	for i, s := range v.Own {
		if s.Name != "yes" || s.PID != 41233+i || s.PGID != 41230 || s.Age != age || s.Share < 0.989 || s.Share > 0.991 {
			t.Fatalf("own[%d] = %+v", i, s)
		}
	}
	if v.Others != (OtherStrays{}) {
		t.Fatalf("others = %+v, want none", v.Others)
	}
	if len(v.Remedy) != 1 || v.Remedy[0].Form != RemedyGroup || v.Remedy[0].PGID != 41230 || len(v.Remedy[0].PIDs) != 8 {
		t.Fatalf("remedy = %+v, want one group form for 41230", v.Remedy)
	}
}

// TestMechanismFlagsIncidentTwo: the 2026-09-22/23 incident, another account's
// `zsh` loops left running since the Saturday. They are counted and their shares
// summed, and nothing else about them is carried.
//
// The fixture's second half is the same incident oversubscribed: at the
// incident's load of about 440 on 16 cores no loop can have held a near-full
// core, but each has held all it could get, about 16/440 of a core, so the stray
// rule still counts every one of them beside the extreme trigger.
func TestMechanismFlagsIncidentTwo(t *testing.T) {
	age := 4 * 24 * time.Hour
	var procs []Proc
	for i := 0; i < 38; i++ {
		procs = append(procs, burner(70000+i, 70000, otherUID, "zsh", age, 0.95))
	}
	v := Classify(snapshot(20, procs...), DefaultLimits(16), caller)
	if !slices.Equal(v.Triggers, []string{TriggerStray}) {
		t.Fatalf("triggers = %v, want [stray]", v.Triggers)
	}
	if len(v.Own) != 0 || len(v.Remedy) != 0 {
		t.Fatalf("another account's programs were named or given a remedy: %+v %+v", v.Own, v.Remedy)
	}
	if v.Others.Count != 38 || v.Others.Cores < 36.09 || v.Others.Cores > 36.11 {
		t.Fatalf("others = %+v, want 38 at 36.1 cores", v.Others)
	}

	// Oversubscribed: each loop has had about 16/440 of a core.
	procs = procs[:0]
	for i := 0; i < 38; i++ {
		procs = append(procs, burner(70000+i, 70000, otherUID, "zsh", age, 16.0/440))
	}
	v = Classify(snapshot(440.2, procs...), DefaultLimits(16), caller)
	if !slices.Equal(v.Triggers, []string{TriggerStray, TriggerExtreme}) || v.Others.Count != 38 {
		t.Fatalf("oversubscribed: triggers %v, others %+v; want both triggers and all 38 counted", v.Triggers, v.Others)
	}
}

// TestStrayRuleCoversTheOversubscribedBand: busy loops that have run for two
// days are strays at every load, because a stray is judged against its fair
// share of the machine as loaded, cores divided by the one-minute load, not
// against a fixed 0.9 of a core (the product thinker's ruling H1 of 2026-09-25,
// iss-2609231947544298). On 16 cores, n such loops at the load they cause (n)
// each hold 16/n of a core, which is all they can get; 18 to 64 loops, a load
// between 1.125 and 4 times the cores, are the band the fixed rule left silent.
// The caller's own loops and another account's are judged alike; only what the
// warning says about them differs.
func TestStrayRuleCoversTheOversubscribedBand(t *testing.T) {
	age := 48 * time.Hour
	cases := []struct {
		loops int
		want  []string
	}{
		{17, []string{TriggerStray}},
		{18, []string{TriggerStray}},
		{20, []string{TriggerStray}},
		{40, []string{TriggerStray}},
		{64, []string{TriggerStray}},
		{65, []string{TriggerStray, TriggerExtreme}},
		{440, []string{TriggerStray, TriggerExtreme}},
	}
	for _, c := range cases {
		for _, uid := range []uint32{otherUID, callerUID} {
			var procs []Proc
			for i := 0; i < c.loops; i++ {
				procs = append(procs, burner(80000+i, 80000, uid, "zsh", age, 16.0/float64(c.loops)))
			}
			v := Classify(snapshot(float64(c.loops), procs...), DefaultLimits(16), caller)
			if !slices.Equal(v.Triggers, c.want) {
				t.Errorf("uid %d, %d loops at load %d: triggers %v (others %+v), want %v", uid, c.loops, c.loops, v.Triggers, v.Others, c.want)
			}
			counted := v.Others.Count
			if uid == callerUID {
				counted = len(v.Own) + v.OwnMore
			}
			if counted != c.loops {
				t.Errorf("uid %d, %d loops at load %d: %d counted as strays, want all of them", uid, c.loops, c.loops, counted)
			}
		}
	}
}

// TestStrayShareIsRelativeToTheFairShare: under an oversubscribed load a
// process is a stray at 0.9 of its fair share or more, and under it is not; at a
// load no higher than the cores its fair share is one core, so the rule is the
// near-full core it was; with no load reading the fair share is one core too.
func TestStrayShareIsRelativeToTheFairShare(t *testing.T) {
	cases := []struct {
		load  float64
		share float64
		stray bool
	}{
		{32, 0.46, true}, // fair share 0.5 of a core; 0.9 of it is 0.45
		{32, 0.44, false},
		{64, 0.23, true}, // fair share 0.25
		{64, 0.22, false},
		{8, 0.89, false}, // under the cores: a full core is the fair share
		{8, 0.90, true},
		{16, 0.89, false},
	}
	for _, c := range cases {
		v := Classify(snapshot(c.load, burner(60000, 60000, callerUID, "spin", 2*time.Hour, c.share)), DefaultLimits(16), caller)
		if got := len(v.Own) == 1; got != c.stray {
			t.Errorf("load %v share %.2f: stray = %v, want %v", c.load, c.share, got, c.stray)
		}
	}
	s := snapshot(64, burner(60000, 60000, callerUID, "spin", 2*time.Hour, 0.5))
	s.HasLoad = false
	if v := Classify(s, DefaultLimits(16), caller); len(v.Own) != 0 {
		t.Fatalf("with no load reading a half-core process is a stray: %+v", v.Own)
	}
}

// TestFairShare: cores over the one-minute load, capped at one core, and one
// core when the load or the cores are unknown.
func TestFairShare(t *testing.T) {
	cases := []struct {
		load    float64
		hasLoad bool
		cores   int
		want    float64
	}{
		{40, true, 16, 0.4},
		{16, true, 16, 1},
		{3, true, 16, 1},
		{0, true, 16, 1},
		{440, false, 16, 1},
		{440, true, 0, 1},
	}
	for _, c := range cases {
		if got := FairShare(Snapshot{Load1: c.load, HasLoad: c.hasLoad, Cores: c.cores}); got != c.want {
			t.Errorf("FairShare(load %v, has %v, cores %d) = %v, want %v", c.load, c.hasLoad, c.cores, got, c.want)
		}
	}
}

// TestStrayBoundaries: over the limit and, on a machine loaded no higher than
// its cores, at a share of at least 0.9 of a core.
func TestStrayBoundaries(t *testing.T) {
	cases := []struct {
		age   time.Duration
		share float64
		stray bool
	}{
		{29 * time.Minute, 1.0, false},
		{31 * time.Minute, 1.0, true},
		{31 * time.Minute, 0.89, false},
		{31 * time.Minute, 0.90, true},
		{30 * time.Minute, 1.0, false}, // the limit itself is not over it
	}
	for _, c := range cases {
		v := Classify(snapshot(1, burner(60000, 60000, callerUID, "spin", c.age, c.share)), DefaultLimits(16), caller)
		if got := len(v.Own) == 1; got != c.stray {
			t.Errorf("age %v share %.2f: stray = %v, want %v", c.age, c.share, got, c.stray)
		}
	}
	// A limit from the settings file moves the boundary.
	lim := DefaultLimits(16)
	lim.StrayMinutes = 60
	if v := Classify(snapshot(1, burner(60000, 60000, callerUID, "spin", 45*time.Minute, 1)), lim, caller); len(v.Own) != 0 {
		t.Fatalf("a 45-minute process is a stray under a 60-minute limit: %+v", v.Own)
	}
}

// TestAncestryIsNeverAStray: the agent host in the check's parent chain has run
// two days at a full core, and is exempt: it is the run that asked.
func TestAncestryIsNeverAStray(t *testing.T) {
	v := Classify(snapshot(1), DefaultLimits(16), caller)
	if len(v.Triggers) != 0 || len(v.Own) != 0 {
		t.Fatalf("the caller's own ancestry was flagged: %+v", v)
	}
	// The same process outside the chain is a stray.
	self := Self{PID: 1234567, UID: callerUID}
	if v := Classify(snapshot(1), DefaultLimits(16), self); len(v.Own) != 1 || v.Own[0].PID != 49000 {
		t.Fatalf("outside the chain the agent host is not flagged: %+v", v.Own)
	}
}

// TestRemedyNeverGroupKillsAMixedGroup: a group that holds anything the warning
// does not name gets the process form for each stray.
func TestRemedyNeverGroupKillsAMixedGroup(t *testing.T) {
	v := Classify(snapshot(1,
		burner(701, 700, callerUID, "spin", 2*time.Hour, 1),
		burner(702, 700, callerUID, "editor", 2*time.Hour, 0.01),
	), DefaultLimits(16), caller)
	if len(v.Remedy) != 1 || v.Remedy[0].Form != RemedyProcess || v.Remedy[0].PID != 701 || v.Remedy[0].Why != WhyMixedGroup {
		t.Fatalf("remedy = %+v, want the process form for 701", v.Remedy)
	}
}

// TestRemedyNeverGroupKillsTheCallersGroup: a stray in the check's own group, or
// in any ancestor's group, gets the process form, so a group kill can never reach
// the caller's shell or agent host.
func TestRemedyNeverGroupKillsTheCallersGroup(t *testing.T) {
	for _, pgid := range []int{49990, 49000} {
		v := Classify(snapshot(1, burner(49500, pgid, callerUID, "spin", 2*time.Hour, 1)), DefaultLimits(16), caller)
		if len(v.Remedy) != 1 || v.Remedy[0].Form != RemedyProcess || v.Remedy[0].PID != 49500 || v.Remedy[0].Why != WhyCallersGroup {
			t.Fatalf("group %d: remedy = %+v, want the process form", pgid, v.Remedy)
		}
	}
}

// TestOwnStraysAreCappedAtTwenty: the list names twenty and counts the rest,
// and a group with an unnamed member is not offered the group form.
func TestOwnStraysAreCappedAtTwenty(t *testing.T) {
	var procs []Proc
	for i := 0; i < 25; i++ {
		procs = append(procs, burner(80000+i, 80000, callerUID, "spin", 2*time.Hour, 1))
	}
	v := Classify(snapshot(1, procs...), DefaultLimits(16), caller)
	if len(v.Own) != MaxOwnNamed || v.OwnMore != 5 {
		t.Fatalf("own = %d (+%d), want %d (+5)", len(v.Own), v.OwnMore, MaxOwnNamed)
	}
	for _, r := range v.Remedy {
		if r.Form == RemedyGroup {
			t.Fatalf("group form offered for a group with unnamed members: %+v", r)
		}
	}
}

// TestEightConcurrentPreflightsAreQuiet replays the run's busiest recorded sample
// (14:59:13Z on 2026-09-23: load 41.91 on 16 cores, eight preflights, twelve
// test binaries) as a process table: eight make, eight go, twelve test binaries
// and their compilers, all younger than 15 minutes at a full core, beside the
// machine's usual long-lived programs, none of which holds nearly all the 0.38
// of a core a program can get at that load.
func TestEightConcurrentPreflightsAreQuiet(t *testing.T) {
	var procs []Proc
	pid := 90000
	add := func(n int, name string, age time.Duration) {
		for i := 0; i < n; i++ {
			pid++
			procs = append(procs, Proc{PID: pid, PPID: 49990, PGID: 90000 + i, UID: callerUID, Age: age, CPU: age, Name: name})
		}
	}
	add(8, "make", 14*time.Minute)
	add(8, "go", 13*time.Minute)
	add(12, "core.test", 2*time.Minute)
	add(16, "compile", 20*time.Second)
	add(4, "link", 10*time.Second)
	add(2, "vet", 30*time.Second)
	// The machine's own long-lived programs: the highest share measured among
	// programs over 30 minutes old at load 18 to 26 was 0.19.
	procs = append(procs,
		burner(300, 300, 88, "WindowServer", 13*time.Hour, 0.19),
		burner(301, 301, rootUID, "mds_stores", 13*time.Hour, 0.05),
		burner(302, 302, callerUID, "Browser Helper", 10*time.Hour, 0.12),
	)
	snap := snapshot(41.91, procs...)
	snap.Load5, snap.Load15 = 30.2, 22.5
	v := Classify(snap, DefaultLimits(16), caller)
	if len(v.Triggers) != 0 || len(v.Own) != 0 || v.Others != (OtherStrays{}) {
		t.Fatalf("eight concurrent preflights warned: %+v", v)
	}
}

// TestExtremeTriggerIsStrictlyAbove: only the one-minute average decides, and
// only when strictly above the limit.
func TestExtremeTriggerIsStrictlyAbove(t *testing.T) {
	quiet := Classify(snapshot(64.0), DefaultLimits(16), caller)
	if len(quiet.Triggers) != 0 {
		t.Fatalf("64.0 on 16 cores warned: %v", quiet.Triggers)
	}
	loud := Classify(snapshot(64.1), DefaultLimits(16), caller)
	if !slices.Equal(loud.Triggers, []string{TriggerExtreme}) {
		t.Fatalf("64.1 on 16 cores: triggers %v, want [extreme]", loud.Triggers)
	}
	s := snapshot(10)
	s.Load5, s.Load15 = 200, 300
	if v := Classify(s, DefaultLimits(16), caller); len(v.Triggers) != 0 {
		t.Fatalf("a five- or fifteen-minute average alone warned: %v", v.Triggers)
	}
	// With no process table the extreme trigger still applies.
	s = snapshot(100)
	s.Procs, s.HasProcs = nil, false
	if v := Classify(s, DefaultLimits(16), caller); !slices.Equal(v.Triggers, []string{TriggerExtreme}) {
		t.Fatalf("no process table: triggers %v", v.Triggers)
	}
}

// TestOtherStraysTypeCarriesNoIdentity: another account's strays are a count and
// a CPU total by type, so no renderer can print what the verdict cannot carry.
// Adding a field fails this test on purpose.
func TestOtherStraysTypeCarriesNoIdentity(t *testing.T) {
	typ := reflect.TypeOf(OtherStrays{})
	var got []string
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, typ.Field(i).Name+" "+typ.Field(i).Type.String())
	}
	want := []string{"Count int", "Cores float64"}
	if !slices.Equal(got, want) {
		t.Fatalf("OtherStrays fields = %v, want exactly %v", got, want)
	}
}
