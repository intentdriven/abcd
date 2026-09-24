package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// loadSelf is the check's invocation in every fixture below.
var loadSelf = machineload.Self{PID: 50000, UID: 501}

// loadSnapshot is a 16-core machine at load1 with the invocation's own chain and
// the given processes.
func loadSnapshot(load1 float64, procs ...machineload.Proc) machineload.Snapshot {
	chain := []machineload.Proc{
		{PID: 1, PPID: 0, PGID: 1, UID: 0, Age: time.Hour, Name: "launchd"},
		{PID: 50000, PPID: 1, PGID: 50000, UID: 501, Age: time.Second, Name: "abcd"},
	}
	return machineload.Snapshot{Load1: load1, Load5: load1 - 1, Load15: load1 - 2, HasLoad: true, Cores: 16,
		Procs: append(chain, procs...), HasProcs: true}
}

// yesBurner is one of the caller's own burners: two days seven hours at 99%.
func yesBurner(pid int) machineload.Proc {
	age := 55 * time.Hour
	return machineload.Proc{PID: pid, PPID: 1, PGID: 41230, UID: 501, Age: age, CPU: time.Duration(float64(age) * 0.99), Name: "yes"}
}

// noEnv is an environment with no CI marker.
func noEnv(string) string { return "" }

// loadRequest is a request against a fixture snapshot, under the run's HOME.
func loadRequest(t *testing.T, snap machineload.Snapshot, readErr error) LoadRequest {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return LoadRequest{
		Site:     SitePreflight,
		Getenv:   noEnv,
		Read:     func() (machineload.Snapshot, error) { return snap, readErr },
		Home:     home,
		RootSHA:  testSHA,
		RepoRoot: t.TempDir(),
		Self:     loadSelf,
		Now:      func() time.Time { return time.Date(2026, 9, 24, 9, 12, 3, 0, time.UTC) },
	}
}

// loadLines returns the run log's load events.
func loadLines(t *testing.T, r *Run) []Event {
	t.Helper()
	evs, _, err := r.ReadLog()
	if err != nil {
		t.Fatal(err)
	}
	var out []Event
	for _, e := range evs {
		if e.Event == EventLoad {
			out = append(out, e)
		}
	}
	return out
}

// TestCheckLoadSkipsOnCIWithReason: on a CI runner nothing is read, and the
// result says why, naming the variable.
func TestCheckLoadSkipsOnCIWithReason(t *testing.T) {
	for _, env := range []map[string]string{{"GITHUB_ACTIONS": "true"}, {"CI": "true"}, {"CI": "1"}} {
		req := loadRequest(t, loadSnapshot(1), nil)
		req.Getenv = func(k string) string { return env[k] }
		req.Read = func() (machineload.Snapshot, error) {
			t.Fatal("the machine was read on a CI runner")
			return machineload.Snapshot{}, nil
		}
		res := CheckLoad(req)
		if res.Status != LoadSkipped {
			t.Fatalf("%v: status %q, want skipped", env, res.Status)
		}
		for k, v := range env {
			if !strings.Contains(res.Reason, k+"="+v) || !strings.Contains(res.Reason, "CI runner") {
				t.Fatalf("%v: reason %q does not name the variable", env, res.Reason)
			}
		}
	}
}

// TestCIValueFalseIsNotCI: an empty, "false" or "0" CI is not a CI runner, and
// GITHUB_ACTIONS counts only as "true".
func TestCIValueFalseIsNotCI(t *testing.T) {
	for _, env := range []map[string]string{{"CI": ""}, {"CI": "false"}, {"CI": "0"}, {"GITHUB_ACTIONS": "false"}} {
		req := loadRequest(t, loadSnapshot(1), nil)
		req.Getenv = func(k string) string { return env[k] }
		read := false
		req.Read = func() (machineload.Snapshot, error) { read = true; return loadSnapshot(1), nil }
		if res := CheckLoad(req); res.Status != LoadOK || !read {
			t.Fatalf("%v: status %q, read %v; want a real check", env, res.Status, read)
		}
	}
}

// TestLoadWarningIsLoggedInALiveRun: a warning inside a run with a joined first
// session writes one load line with the verdict's facts, attributed to the first
// session even when the second joined earlier.
func TestLoadWarningIsLoggedInALiveRun(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "beta", RoleSecond)
	c.t = c.t.Add(time.Minute)
	join(t, r, "alpha", RoleFirst)

	req := loadRequest(t, loadSnapshot(70, yesBurner(41233), yesBurner(41234)), nil)
	req.Getenv = func(k string) string {
		if k == "ABCD_LOAD_CHECKED" {
			return "preflight"
		}
		return ""
	}
	res := CheckLoad(req)
	if res.Status != LoadWarning || !slices.Equal(res.Triggers, []string{"stray", "extreme"}) {
		t.Fatalf("result = %+v", res)
	}
	if !res.RunLog.Logged || res.RunLog.Session != "alpha" || res.RunLog.Error != "" {
		t.Fatalf("run log = %+v", res.RunLog)
	}
	lines := loadLines(t, r)
	if len(lines) != 1 {
		t.Fatalf("%d load lines, want 1", len(lines))
	}
	e := lines[0]
	if e.Session != "alpha" || e.String("site") != "preflight" || e.String("within_preflight") != "true" {
		t.Fatalf("event = %+v", e.Fields)
	}
	var triggers []string
	if err := json.Unmarshal(e.Fields["triggers"], &triggers); err != nil || !slices.Equal(triggers, []string{"stray", "extreme"}) {
		t.Fatalf("triggers = %s", e.Fields["triggers"])
	}
	back, err := LoadResultFromEvent(e)
	if err != nil {
		t.Fatal(err)
	}
	// The event carries the verdict's facts: read back, it is the same warning.
	want := res
	want.RunLog, want.Reason, want.Limits.Malformed = LoadRunLog{}, "", ""
	back.RunLog = LoadRunLog{}
	if gotJ, wantJ := mustJSON(t, back), mustJSON(t, want); gotJ != wantJ {
		t.Fatalf("event reads back as\n%s\nwant\n%s", gotJ, wantJ)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestNoEventOutsideARun: with no joined session, a warning creates no directory
// and no file anywhere under the home.
func TestNoEventOutsideARun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	res := CheckLoad(loadRequest(t, loadSnapshot(1, yesBurner(41233)), nil))
	if res.Status != LoadWarning || res.RunLog.Logged || res.RunLog.Error != "" {
		t.Fatalf("result = %+v", res)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("the check created %v under HOME", entries)
	}

	// A run that exists with no joined session is not live either.
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	if _, err := r.Leave("alpha", "done"); err != nil {
		t.Fatal(err)
	}
	CheckLoad(loadRequest(t, loadSnapshot(1, yesBurner(41233)), nil))
	if n := len(loadLines(t, r)); n != 0 {
		t.Fatalf("%d load lines in a run nobody has joined", n)
	}
}

// TestNoEventWhenQuiet: ok, skipped and unchecked write nothing.
func TestNoEventWhenQuiet(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	quiet := loadRequest(t, loadSnapshot(3), nil)
	if res := CheckLoad(quiet); res.Status != LoadOK || res.RunLog.Logged {
		t.Fatalf("quiet = %+v", res)
	}
	unsupported := loadRequest(t, machineload.Snapshot{}, machineload.ErrUnsupported)
	if res := CheckLoad(unsupported); res.Status != LoadUnchecked || res.RunLog.Logged {
		t.Fatalf("unsupported = %+v", res)
	}
	if n := len(loadLines(t, r)); n != 0 {
		t.Fatalf("%d load lines for checks that did not warn", n)
	}
}

// TestLoadIsVerbOwned: `implement log load` cannot write an imitation.
func TestLoadIsVerbOwned(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	_, err := r.Log("alpha", EventLoad, map[string]string{"load1": "3"})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "written by the implement verbs themselves") {
		t.Fatalf("hand-written load event: %v, want the verb-owned refusal", err)
	}
	if slices.Contains(LoggableEvents(), EventLoad) {
		t.Fatal("load is listed as a loggable event")
	}
}

// TestRunLogFailureIsReportedAndTheWarningStands: a log that cannot be written
// is reported on the result; the warning is unchanged.
func TestRunLogFailureIsReportedAndTheWarningStands(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	req := loadRequest(t, loadSnapshot(1, yesBurner(41233)), nil)
	// Today's log file is a directory, so the append fails.
	if err := os.Mkdir(filepath.Join(r.Dir, "2026-09-24.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	res := CheckLoad(req)
	if res.Status != LoadWarning || res.RunLog.Logged || res.RunLog.Error == "" {
		t.Fatalf("result = %+v", res)
	}
}

// writeLimits writes the settings file under the run's HOME.
func writeLimits(t *testing.T, body string, mode os.FileMode) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".abcd", machineload.LimitsFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLimitsFromTheSettingsFile: the file's limits are used, and marked as the
// file's; without it the defaults derive from the online core count.
func TestLimitsFromTheSettingsFile(t *testing.T) {
	newRun(t)
	res := CheckLoad(loadRequest(t, loadSnapshot(50), nil))
	if res.Limits == nil || res.Limits.Source != LimitsDefault || res.Limits.ExtremeLoad != 64 || res.Limits.StrayMinutes != 30 {
		t.Fatalf("defaults = %+v", res.Limits)
	}
	writeLimits(t, "# mine\nextreme-load 40\n", 0o600)
	res = CheckLoad(loadRequest(t, loadSnapshot(50), nil))
	if res.Limits.Source != LimitsFile || res.Limits.ExtremeLoad != 40 || !res.Limits.ExtremeFromFile || res.Limits.StrayFromFile {
		t.Fatalf("file = %+v", res.Limits)
	}
	if res.Status != LoadWarning || !slices.Equal(res.Triggers, []string{"extreme"}) {
		t.Fatalf("load 50 over a limit of 40: %+v", res)
	}
}

// TestMalformedLimitsFileFallsBackWhole: every refusal of the guarded read, and
// every parse fault, yields both defaults and a report that never echoes a value.
func TestMalformedLimitsFileFallsBackWhole(t *testing.T) {
	newRun(t)
	cases := map[string]struct {
		setup func(t *testing.T)
		want  string
	}{
		"unknown key": {func(t *testing.T) { writeLimits(t, "stray-minutes 5\nstray-mins 7\n", 0o600) }, `line 2: unknown key "stray-mins"`},
		"oversize":    {func(t *testing.T) { writeLimits(t, "# "+strings.Repeat("x", 5000)+"\n", 0o600) }, "could not be read"},
		"group-writable": {func(t *testing.T) { writeLimits(t, "stray-minutes 5\n", 0o620) },
			"writable by group or other"},
		"symlink": {func(t *testing.T) {
			path := writeLimits(t, "stray-minutes 5\n", 0o600)
			target := path + ".real"
			if err := os.Rename(path, target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}, "not a regular file"},
		"another owner": {func(t *testing.T) {
			writeLimits(t, "stray-minutes 5\n", 0o600)
			restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil })
			t.Cleanup(restore)
		}, "not owned by you"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			c.setup(t)
			res := CheckLoad(loadRequest(t, loadSnapshot(3), nil))
			l := res.Limits
			if l.Source != LimitsDefaultAfterMalformed || l.StrayMinutes != 30 || l.ExtremeLoad != 64 || l.StrayFromFile || l.ExtremeFromFile {
				t.Fatalf("limits = %+v, want both defaults", l)
			}
			// The unknown key's value (7) and the file's other values (5) are never echoed.
			if !strings.Contains(l.Malformed, c.want) || strings.Contains(l.Malformed, " 7") || strings.Contains(l.Malformed, " 5") {
				t.Fatalf("malformed = %q, want it to name %q", l.Malformed, c.want)
			}
			if res.Status != LoadOK {
				t.Fatalf("status %q: an unusable file is reported, not a warning", res.Status)
			}
		})
	}
}

// TestCheckLoadCannotCheck: an unsupported platform, or a read that fails, is
// unchecked with the reason; a load average that read is still reported and the
// extreme trigger still applies.
func TestCheckLoadCannotCheck(t *testing.T) {
	newRun(t)
	req := loadRequest(t, machineload.Snapshot{}, machineload.ErrUnsupported)
	req.GOOS = "freebsd"
	res := CheckLoad(req)
	if res.Status != LoadUnchecked || !strings.Contains(res.Reason, "macOS and Linux only, and this is freebsd") || res.Load != nil {
		t.Fatalf("unsupported = %+v", res)
	}

	noLoad := loadRequest(t, machineload.Snapshot{}, errors.New("could not read the load average (boom)"))
	if res := CheckLoad(noLoad); res.Status != LoadUnchecked || !strings.Contains(res.Reason, "boom") {
		t.Fatalf("no load = %+v", res)
	}

	partial := loadSnapshot(3)
	partial.Procs, partial.HasProcs = nil, false
	procErr := errors.New("could not read the process table (/bin/ps: exit status 1)")
	res = CheckLoad(loadRequest(t, partial, procErr))
	if res.Status != LoadUnchecked || res.Load == nil || res.Load.One != 3 || !strings.Contains(res.Reason, "/bin/ps: exit status 1") {
		t.Fatalf("no process table = %+v", res)
	}
	partial.Load1 = 99
	res = CheckLoad(loadRequest(t, partial, procErr))
	if res.Status != LoadWarning || !slices.Equal(res.Triggers, []string{"extreme"}) || res.Reason == "" {
		t.Fatalf("no process table, extreme load = %+v", res)
	}
}

// TestOwnNamesPassThePrivateScrub: a name the private layer matches is masked in
// the result and in the logged event alike; a layer that cannot be read withholds
// every name.
func TestOwnNamesPassThePrivateScrub(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	secret := yesBurner(41233)
	secret.Name = "clientwidget"
	req := loadRequest(t, loadSnapshot(1, secret, yesBurner(41234)), nil)
	store := filepath.Join(req.RepoRoot, ".abcd", ".work.local", "private-names.txt")
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nclient CLIENTWIDGET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := CheckLoad(req)
	if len(res.OwnStrays) != 2 || res.OwnStrays[0].Name != PrivateNameMask || res.OwnStrays[1].Name != "yes" || res.NamesWithheld {
		t.Fatalf("own = %+v", res.OwnStrays)
	}
	lines := loadLines(t, r)
	if len(lines) != 1 || strings.Contains(string(lines[0].Fields["own_strays"]), "clientwidget") {
		t.Fatalf("the logged event carries the private name: %s", lines[0].Fields["own_strays"])
	}

	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nonly-a-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res = CheckLoad(req)
	if !res.NamesWithheld {
		t.Fatalf("an unreadable layer did not withhold: %+v", res)
	}
	for _, s := range res.OwnStrays {
		if s.Name != WithheldNameMask {
			t.Fatalf("name %q printed while the layer is unreadable", s.Name)
		}
	}
}
