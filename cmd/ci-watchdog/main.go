//go:build unix

// Command ci-watchdog runs one CI step under the job's clock, so a hang fails
// with a goroutine dump before the job's cap cancels it with none.
//
// go test's -timeout counts from each test binary's own start, so a package
// that starts late in a long step meets the job's timeout-minutes first, and a
// cancellation at the cap carries nothing but the cancellation. The check job
// in ci.yml therefore runs every step that runs go test with
//
//	shell: go run ./cmd/ci-watchdog bash --noprofile --norc -eo pipefail {0}
//
// It starts the command as a process group of its own and, at the job's start
// plus its cap less its margin, sends SIGQUIT to the test binaries: the
// processes in that group whose parent is go test, a child of the shell. On SIGQUIT each test binary prints every goroutine's stack and exits,
// and go test prints that output with the package's FAIL line. go test itself
// is never sent it: it takes SIGQUIT for an interrupt, stops scheduling work,
// and drops the output it holds for each package of a run that names several,
// so a SIGQUIT to the whole group loses the dumps. Each target is chosen from
// ps by the group's id and its parent, never by its own name, and its group and
// parent are read again just before its signal. After a grace period the watchdog sends the whole
// group SIGKILL through the step's own handle, checked again just before: the
// group's leader must still be this watchdog's child, not yet waited for, and
// still lead its group. A SIGINT or SIGTERM sent to the watchdog itself is
// passed on to the group the same way. Started through go run, as ci.yml
// starts it, the runner's cancellation does not reach it: go run ignores
// SIGINT and dies on SIGTERM without passing either on, so the step's group is
// left to the runner's clean-up of the job's orphan processes when the job
// ends. If no child of the shell is go at the
// deadline (a step whose shell ran its last command in its own place, say),
// there is nothing to send SIGQUIT to: the watchdog says what it found instead
// and sends the group SIGKILL at once, so that step fails loudly rather than
// with no dump and no word why.
//
// A target's pid is signalled on its own, not through a handle: between the
// last read of its group and parent and its signal, the process can exit and
// its pid be reused. The read just before narrows that window to a new process
// that took the same pid and joined the step's own group under the same parent,
// which only the step's own processes can do.
//
// It reads the job's clock from the environment the check job sets:
// ABCD_CI_LEG (the leg it names), ABCD_CI_JOB_STARTED (the job's start in Unix
// seconds, exported by the "Record the job's start" step),
// ABCD_CI_JOB_CAP_MINUTES (the leg's timeout-minutes) and
// ABCD_CI_WATCHDOG_MARGIN_MINUTES (the one place the margin is declared).
// TestCheckJobTestStepsRunUnderTheJobClockWatchdog holds the workflow to them.
//
// It is developer tooling, never part of a release. A step that ends before
// the deadline exits with the step's own status (128 plus the signal for one a
// signal ended). A step stopped at the deadline exits 124, as timeout(1) does,
// with one line on standard error naming the leg, the deadline and the time on
// the job's clock. Started through go run, as ci.yml starts it, a status other
// than 0 reaches the runner as go run's own 1, after go run prints the status
// it got ("exit status 124"). A setting it cannot read, or a command it cannot start,
// exits 2 before anything runs.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// grace is how long the group has, after SIGQUIT, to print its dumps and exit
// before SIGKILL.
const grace = 30 * time.Second

// exitDeadline is the exit status of a step stopped at the deadline.
const exitDeadline = 124

func main() {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	os.Exit(run(os.Args[1:], os.Getenv, time.Now, time.After, sigs, os.Stdout, os.Stderr, grace))
}

// clock is the job's clock as the check job declares it.
type clock struct {
	leg     string
	started time.Time
	cap     time.Duration
	margin  time.Duration
}

// deadline is how far into the job the step is stopped.
func (c clock) deadline() time.Duration { return c.cap - c.margin }

// readClock reads the job's clock from getenv, refusing a setting that is
// missing or does not read.
func readClock(getenv func(string) string) (clock, error) {
	minutes := func(name string, min int) (time.Duration, error) {
		raw := getenv(name)
		n, err := strconv.Atoi(raw)
		if raw == "" || err != nil || n < min {
			return 0, fmt.Errorf("%s is %q, not a whole number of minutes from %d", name, raw, min)
		}
		return time.Duration(n) * time.Minute, nil
	}
	c := clock{leg: getenv("ABCD_CI_LEG")}
	if c.leg == "" {
		return clock{}, errors.New("ABCD_CI_LEG is not set, so the step's leg cannot be named")
	}
	raw := getenv("ABCD_CI_JOB_STARTED")
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || secs <= 0 {
		return clock{}, fmt.Errorf("ABCD_CI_JOB_STARTED is %q, not the job's start in Unix seconds "+
			"(the check job's `Record the job's start` step exports it)", raw)
	}
	c.started = time.Unix(secs, 0)
	if c.cap, err = minutes("ABCD_CI_JOB_CAP_MINUTES", 1); err != nil {
		return clock{}, err
	}
	if c.margin, err = minutes("ABCD_CI_WATCHDOG_MARGIN_MINUTES", 0); err != nil {
		return clock{}, err
	}
	if c.margin >= c.cap {
		return clock{}, fmt.Errorf("ABCD_CI_WATCHDOG_MARGIN_MINUTES (%s) leaves no deadline inside the %s cap", c.margin, c.cap)
	}
	return c, nil
}

// group is the step's process group, reached only through its leader's handle.
type group struct{ leader *os.Process }

// leads reports whether the handle still names the group: the leader must not
// have been waited for (the handle then refuses it, so its pid, and the group
// id it names, cannot have been reused) and must still lead its own group.
func (g group) leads() bool {
	if g.leader.Signal(syscall.Signal(0)) != nil {
		return false
	}
	pgid, err := syscall.Getpgid(g.leader.Pid)
	return err == nil && pgid == g.leader.Pid
}

// signal sends sig to the group through its handle, and reports whether it
// was sent.
func (g group) signal(sig syscall.Signal) bool {
	return g.leads() && syscall.Kill(-g.leader.Pid, sig) == nil
}

// trace sends SIGQUIT to the test binaries, the group's processes whose parent
// is go test, a child of the leader, and returns how many it sent it to. Each
// target's group and parent are read again just before its signal. It sends
// nothing, and says why in its error, when the handle no longer names the
// group, when ps cannot be read, or when no child of the leader is go (the
// error then names the children it found).
func (g group) trace() (int, error) {
	if !g.leads() {
		return 0, errors.New("the step's shell no longer leads its process group")
	}
	leader := g.leader.Pid
	procs, err := processes("-A")
	if err != nil {
		return 0, fmt.Errorf("ps could not list the processes (%v)", err)
	}
	gotest := map[int]bool{}
	var found []string
	for pid, p := range procs {
		if p.ppid != leader || p.pgid != leader {
			continue
		}
		found = append(found, p.name)
		if p.name == "go" {
			gotest[pid] = true
		}
	}
	if len(gotest) == 0 {
		sort.Strings(found)
		children := "none"
		if len(found) > 0 {
			children = strings.Join(found, ", ")
		}
		return 0, fmt.Errorf("found no go test under the step's shell (its children: %s)", children)
	}
	sent := 0
	for pid, p := range procs {
		if p.pgid != leader || !gotest[p.ppid] {
			continue
		}
		if now, err := processes("-p", strconv.Itoa(pid)); err == nil && now[pid] == p && syscall.Kill(pid, syscall.SIGQUIT) == nil {
			sent++
		}
	}
	return sent, nil
}

// proc is a process's parent, group and command name, as ps reports them.
type proc struct {
	ppid, pgid int
	name       string
}

// processes reads the processes ps selects with sel ("-A" for every one, or
// "-p" and a pid), by pid.
func processes(sel ...string) (map[int]proc, error) {
	out, err := exec.Command("ps", append(sel, "-o", "pid=", "-o", "ppid=", "-o", "pgid=", "-o", "comm=")...).Output()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	procs := map[int]proc{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		pgid, err3 := strconv.Atoi(f[2])
		if err1 == nil && err2 == nil && err3 == nil {
			// macOS reports the command's path, Linux its name.
			procs[pid] = proc{ppid: ppid, pgid: pgid, name: filepath.Base(strings.Join(f[3:], " "))}
		}
	}
	return procs, nil
}

// run starts args as a process group, stops it at the job clock's deadline
// (after says when a duration from now has passed), passes on the signals sigs
// carries, and returns the exit status.
func run(args []string, getenv func(string) string, now func() time.Time, after func(time.Duration) <-chan time.Time,
	sigs <-chan os.Signal, stdout, stderr io.Writer, grace time.Duration) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ci-watchdog <command> [<argument>...]")
		return 2
	}
	c, err := readClock(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "ci-watchdog: %v; nothing was run\n", err)
		return 2
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(stderr, "ci-watchdog: %v; nothing was run\n", err)
		return 2
	}
	g := group{leader: cmd.Process}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	deadline := after(c.started.Add(c.deadline()).Sub(now()))
wait:
	for {
		select {
		case err := <-done:
			return status(err)
		case sig := <-sigs:
			if s, ok := sig.(syscall.Signal); ok {
				g.signal(s)
			}
		case <-deadline:
			break wait
		}
	}

	at := now().Sub(c.started).Round(time.Second)
	passed := fmt.Sprintf("ci-watchdog: %s passed its deadline of %s on the job clock (the %s cap less a %s margin) at %s",
		c.leg, c.deadline(), c.cap, c.margin, at)
	traced, err := g.trace()
	if err != nil {
		g.signal(syscall.SIGKILL)
		<-done
		fmt.Fprintf(stderr, "%s; %v, so no test binary was sent SIGQUIT for a dump; sent the step's process group SIGKILL\n",
			passed, err)
		return exitDeadline
	}
	killed := false
	select {
	case <-done:
	case <-time.After(grace):
		killed = g.signal(syscall.SIGKILL)
		<-done
	}
	then := ""
	if killed {
		then = fmt.Sprintf(", then the step's process group SIGKILL after %s", grace)
	}
	fmt.Fprintf(stderr, "%s; sent SIGQUIT to %d test process(es) under go test for the goroutine dumps above%s\n",
		passed, traced, then)
	return exitDeadline
}

// status is the exit status of a step that ended with err from Wait.
func status(err error) int {
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return exit.ExitCode()
	default:
		return 2
	}
}
