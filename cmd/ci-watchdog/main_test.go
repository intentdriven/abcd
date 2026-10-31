//go:build unix

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// TestMain doubles as a step's long-running command: with
// CI_WATCHDOG_HELPER=sleep the test binary marks itself ready, so the
// deadline comes only once the process the step waits on exists, and sleeps.
func TestMain(m *testing.M) {
	if os.Getenv("CI_WATCHDOG_HELPER") == "sleep" {
		if err := os.WriteFile(os.Getenv("CI_WATCHDOG_READY"), nil, 0o644); err != nil {
			os.Exit(3)
		}
		time.Sleep(time.Hour)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// sleeper is the test binary under name, a command that marks itself ready
// at ready and sleeps.
func sleeper(t *testing.T, name, ready string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.Symlink(exe, path); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CI_WATCHDOG_HELPER", "sleep")
	t.Setenv("CI_WATCHDOG_READY", ready)
	return path
}

// jobClock is a check job's environment for the watchdog: a 10-minute cap less
// a 1-minute margin, on a job that started started ago.
func jobClock(started time.Duration) func(string) string {
	env := map[string]string{
		"ABCD_CI_LEG":                     "check (test-leg)",
		"ABCD_CI_JOB_STARTED":             strconv.FormatInt(time.Now().Add(-started).Unix(), 10),
		"ABCD_CI_JOB_CAP_MINUTES":         "10",
		"ABCD_CI_WATCHDOG_MARGIN_MINUTES": "1",
	}
	return func(k string) string { return env[k] }
}

// whenReady is an after whose deadline comes once the step has written ready,
// so the signal reaches a step that is running, however slowly it started. It
// stores the deadline's time in fired, which the step's own timing counts from.
func whenReady(t *testing.T, ready string, fired *atomic.Int64) func(time.Duration) <-chan time.Time {
	t.Helper()
	return func(time.Duration) <-chan time.Time {
		deadline := make(chan time.Time, 1)
		go func() {
			for start := time.Now(); time.Since(start) < 3*time.Minute; time.Sleep(20 * time.Millisecond) {
				if _, err := os.Stat(ready); err == nil {
					break
				}
			}
			now := time.Now()
			fired.Store(now.UnixNano())
			deadline <- now
		}()
		return deadline
	}
}

// sinceFired is how long ago the deadline fired.
func sinceFired(fired *atomic.Int64) time.Duration {
	return time.Since(time.Unix(0, fired.Load()))
}

// never is an after whose deadline does not come.
func never(time.Duration) <-chan time.Time { return nil }

// runStep runs args under the watchdog and returns its status and output.
func runStep(t *testing.T, args []string, getenv func(string) string, after func(time.Duration) <-chan time.Time,
	sigs <-chan os.Signal, grace time.Duration) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, getenv, time.Now, after, sigs, &stdout, &stderr, grace)
	return code, stdout.String(), stderr.String()
}

func TestAStepThatEndsBeforeTheDeadlineKeepsItsStatus(t *testing.T) {
	for script, want := range map[string]int{"exit 0": 0, "exit 3": 3, "kill -TERM $$": 128 + int(syscall.SIGTERM)} {
		code, _, stderr := runStep(t, []string{"sh", "-c", script}, jobClock(time.Minute), never, nil, time.Second)
		if code != want {
			t.Errorf("%q: exit %d, want %d (stderr %q)", script, code, want, stderr)
		}
		if stderr != "" {
			t.Errorf("%q: stderr %q, want nothing from the watchdog", script, stderr)
		}
	}
}

// A go test still running at the deadline fails with the hung test's
// goroutines in the step's output. The run names two packages, so go test
// holds each package's output until its binary exits: a SIGQUIT that reached
// go test itself would interrupt it and drop that output, so the dump appears
// only when the signal reaches the test binary alone. Every trial runs, because
// an interrupted go test still prints a package's output now and then.
func TestAHungGoTestFailsWithItsDumpAtTheDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a real go test")
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module zzhang\n\ngo 1.21\n",
		"hang/hang_test.go": "package hang\n\nimport (\n\t\"os\"\n\t\"testing\"\n\t\"time\"\n)\n\n" +
			"func TestHangs(t *testing.T) {\n\tos.WriteFile(os.Getenv(\"CI_WATCHDOG_READY\"), nil, 0o644)\n\ttime.Sleep(time.Hour)\n}\n",
		"pass/pass_test.go": "package pass\n\nimport \"testing\"\n\nfunc TestPasses(t *testing.T) {}\n",
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The step as the runner writes it: a script file bash runs, so go test is
	// the shell's child and the test binaries its children.
	step := filepath.Join(dir, "step.sh")
	script := "cd " + strconv.Quote(dir) + "\ngo test -count=1 -timeout 40m ./hang ./pass\n"
	if err := os.WriteFile(step, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	for trial := 1; trial <= 5; trial++ {
		ready := filepath.Join(dir, "ready-"+strconv.Itoa(trial))
		t.Setenv("CI_WATCHDOG_READY", ready)
		var fired atomic.Int64
		code, stdout, stderr := runStep(t, []string{"bash", "--noprofile", "--norc", "-eo", "pipefail", step},
			jobClock(2*time.Minute), whenReady(t, ready, &fired), nil, time.Minute)
		if code != exitDeadline {
			t.Errorf("trial %d: exit %d, want %d", trial, code, exitDeadline)
		}
		if !strings.Contains(stdout, "SIGQUIT: quit") || !strings.Contains(stdout, "goroutine ") ||
			!strings.Contains(stdout, "TestHangs") || !strings.Contains(stdout, "FAIL\tzzhang/hang") {
			t.Errorf("trial %d: the step's output carries no dump of the hung test:\n%s", trial, stdout)
		}
		if took := sinceFired(&fired); took > 30*time.Second {
			t.Errorf("trial %d: ended %s after the deadline: the dump ended the step, so it must not wait out the grace", trial, took)
		}
		lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
		last := lines[len(lines)-1]
		if !regexp.MustCompile(` at [0-9]+m[0-9]+s; `).MatchString(last) {
			t.Errorf("trial %d: last line %q does not give the time on the job clock", trial, last)
		}
		for _, want := range []string{"ci-watchdog: check (test-leg) passed its deadline of 9m0s on the job clock",
			"10m0s cap less a 1m0s margin", " test process(es) under go test"} {
			if !strings.Contains(last, want) {
				t.Errorf("trial %d: last line %q does not say %q", trial, last, want)
			}
		}
		// The passing package's binary, or its build, can still be running too.
		if !regexp.MustCompile(`sent SIGQUIT to [1-9][0-9]* test process`).MatchString(last) {
			t.Errorf("trial %d: last line %q does not say the hung binary was sent SIGQUIT", trial, last)
		}
		if strings.Contains(last, "SIGKILL") {
			t.Errorf("trial %d: last line %q names SIGKILL, but go test ended on the dump", trial, last)
		}
	}
}

// A go test that outlasts its SIGQUIT is killed with its group once the grace
// has passed. The step's go is a sleeper that has no test binary to signal.
func TestAGoTestThatOutlastsTheGraceIsKilled(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	gotest := sleeper(t, "go", ready)
	var fired atomic.Int64
	code, _, stderr := runStep(t, []string{"bash", "-c", `"$0"; true`, gotest},
		jobClock(time.Minute), whenReady(t, ready, &fired), nil, 500*time.Millisecond)
	if code != exitDeadline {
		t.Errorf("exit %d, want %d", code, exitDeadline)
	}
	if !strings.Contains(stderr, "sent SIGQUIT to 0 test process(es) under go test") ||
		!strings.Contains(stderr, "then the step's process group SIGKILL after 500ms") {
		t.Errorf("stderr %q does not say the group was killed after the grace", stderr)
	}
	if took := sinceFired(&fired); took > 20*time.Second {
		t.Errorf("ended %s after the deadline: the group was not killed at the end of the grace", took)
	}
}

// A step with no go test under its shell at the deadline says what it found
// and is killed at once, not after the grace: it has no test binary to dump.
func TestAStepWithNoGoTestIsKilledAtOnceAndSaysWhy(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	other := sleeper(t, "not-go", ready)
	var fired atomic.Int64
	code, _, stderr := runStep(t, []string{"bash", "-c", `"$0"; true`, other},
		jobClock(time.Minute), whenReady(t, ready, &fired), nil, time.Minute)
	if code != exitDeadline {
		t.Errorf("exit %d, want %d", code, exitDeadline)
	}
	if !strings.Contains(stderr, "found no go test under the step's shell (its children: not-go)") ||
		!strings.Contains(stderr, "sent the step's process group SIGKILL") {
		t.Errorf("stderr %q does not name what it found and the kill", stderr)
	}
	if took := sinceFired(&fired); took > 20*time.Second {
		t.Errorf("ended %s after the deadline: the group was not killed at once", took)
	}
}

// The runner's cancellation reaches the step's group, which the watchdog took
// out of the runner's own.
func TestTheRunnersCancellationReachesTheGroup(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	sigs <- syscall.SIGTERM
	code, _, stderr := runStep(t, []string{"sleep", "60"}, jobClock(time.Minute), never, sigs, time.Second)
	if want := 128 + int(syscall.SIGTERM); code != want {
		t.Errorf("exit %d, want %d (stderr %q)", code, want, stderr)
	}
}

func TestAClockThatDoesNotReadRunsNothing(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	good := jobClock(time.Minute)
	for name, bad := range map[string]string{
		"ABCD_CI_LEG":                     "",
		"ABCD_CI_JOB_STARTED":             "soon",
		"ABCD_CI_JOB_CAP_MINUTES":         "0",
		"ABCD_CI_WATCHDOG_MARGIN_MINUTES": "10",
	} {
		getenv := func(k string) string {
			if k == name {
				return bad
			}
			return good(k)
		}
		code, _, stderr := runStep(t, []string{"touch", marker}, getenv, never, nil, time.Second)
		if code != 2 || !strings.Contains(stderr, name) || !strings.Contains(stderr, "nothing was run") {
			t.Errorf("%s=%q: exit %d, stderr %q; want 2 naming the setting", name, bad, code, stderr)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("a step ran under a clock that does not read")
	}
}

// The handle refuses a leader the watchdog has already waited for, so a group
// id its pid could be reused as is never signalled.
func TestTheHandleRefusesALeaderItHasWaitedFor(t *testing.T) {
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if (group{leader: cmd.Process}).signal(syscall.SIGQUIT) {
		t.Error("signal reached a group whose leader was already waited for")
	}
}
