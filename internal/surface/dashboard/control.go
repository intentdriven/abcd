package dashboard

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/tailscale"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The dashboard's state under abcd's home (D5): the run file naming the one
// server this computer runs, the devices that opened it in this run, and the
// lock start and stop take. They name no project content.
const (
	stateDir = "dashboard"
	runFile  = "run.json"
	seenFile = "seen.json"
	lockFile = "control.lock"
)

// The control's bounds.
const (
	lockTimeout     = 5 * time.Second
	readyTimeout    = 15 * time.Second
	selfCheckWait   = 5 * time.Second
	stopWait        = 10 * time.Second
	maxStateBytes   = 256 << 10
	childConfigWire = 1
)

// errProcessGone is readProcess's answer for a pid with no live process.
var errProcessGone = errors.New("no such process")

// processID is what the kernel says about a process: when it started and what
// it runs. Together with the pid it names one process, not a reused number.
type processID struct {
	Start string `json:"started"`
	Exe   string `json:"executable"`
}

// Refusal is a start, stop or serve that did nothing because something it
// checks first says no. The front door exits 2 on it.
type Refusal struct{ msg string }

func (r *Refusal) Error() string { return r.msg }

func refuse(format string, a ...any) error { return &Refusal{msg: fmt.Sprintf(format, a...)} }

// RunFile is what start records about the server it launched.
type RunFile struct {
	PID     int       `json:"pid"`
	Process processID `json:"process"`
	Name    string    `json:"name"`
	Port    int       `json:"port"`
	Addrs   []string  `json:"addresses"`
	Since   time.Time `json:"since"`
}

// URL is the address a device on the tailnet opens.
func (r RunFile) URL() string { return "http://" + r.Name + ":" + strconv.Itoa(r.Port) }

// Launcher is how start runs the server process: this binary's hidden serve
// in production, the test binary in a test.
type Launcher struct {
	Path string
	Args []string
	Env  []string
}

// DefaultLauncher runs this binary's hidden `dashboard serve`.
func DefaultLauncher() (Launcher, error) {
	exe, err := os.Executable()
	if err != nil {
		return Launcher{}, err
	}
	return Launcher{Path: exe, Args: []string{"dashboard", "serve"}}, nil
}

// StartOptions is what start needs.
type StartOptions struct {
	// Home is the person's home directory, holding abcd's home.
	Home string
	// Root is the checkout the dashboard serves; the server runs in it.
	Root string
	// Port is the port to listen on.
	Port int
	// Tailscale is the resolved command.
	Tailscale *tailscale.Client
	// Launch starts the server process.
	Launch Launcher
}

// StartResult is what start reports.
type StartResult struct {
	URL   string   `json:"url"`
	Name  string   `json:"name"`
	Port  int      `json:"port"`
	Addrs []string `json:"addresses"`
	PID   int      `json:"pid"`
	Line  string   `json:"line"`
}

// childConfig is what start hands the server over its config pipe, never on
// its command line or in its environment, where another local process could
// read the one-time value.
type childConfig struct {
	Wire      int      `json:"wire"`
	Home      string   `json:"home"`
	Tailscale string   `json:"tailscale"`
	Name      string   `json:"name"`
	Port      int      `json:"port"`
	Addrs     []string `json:"addresses"`
	SelfCheck string   `json:"self_check"`
}

// readyMessage is the server's one line back over the readiness pipe.
type readyMessage struct {
	OK    bool   `json:"ok"`
	Port  int    `json:"port,omitempty"`
	Error string `json:"error,omitempty"`
}

// Start launches the dashboard and returns once a fetch of its own address
// through the tailnet has answered. It refuses (a *Refusal) when Tailscale is
// not running or names no tailnet address for this computer, when the port is
// one Tailscale's own Serve or Funnel configuration names, and when a
// dashboard already runs on this computer.
func Start(ctx context.Context, opts StartOptions) (StartResult, error) {
	if opts.Port < 0 || opts.Port > 65535 {
		return StartResult{}, refuse("port %d is not a port", opts.Port)
	}
	self, err := opts.Tailscale.Status(ctx)
	if err != nil {
		return StartResult{}, refuse("%v; start Tailscale and run this again", err)
	}
	var addrs []netip.Addr
	for _, a := range self.Addrs {
		if inTailnet(a, tailnetPrefixes) {
			addrs = append(addrs, a)
		}
	}
	if len(addrs) == 0 {
		return StartResult{}, refuse("Tailscale names no Tailscale address for this computer, so there is nothing to listen on")
	}
	served, err := opts.Tailscale.ServedPorts(ctx)
	if err != nil {
		return StartResult{}, refuse("could not read Tailscale's Serve and Funnel configuration (%v), so the port cannot be shown to be free of it", err)
	}
	if opts.Port != 0 && slices.Contains(served, uint16(opts.Port)) {
		return StartResult{}, refuse("port %d is one Tailscale's own Serve or Funnel configuration uses; choose another port", opts.Port)
	}

	dir, err := fsutil.EnsureHomeScope(opts.Home, abcdhome.Rel(stateDir), abcdhome.DirMode)
	if err != nil {
		return StartResult{}, fmt.Errorf("preparing %s: %w", abcdhome.Display(stateDir), err)
	}
	defer dir.Close()

	var res StartResult
	err = fsutil.WithFileLock(abcdhome.Path(opts.Home, stateDir, lockFile), lockTimeout, func() error {
		run, running, err := readRunning(opts.Home)
		if err != nil {
			return err
		}
		if running {
			return refuse("a dashboard already runs on this computer at %s; `abcd dashboard stop` stops it", run.URL())
		}
		if err := fsutil.WriteFileAtomicInRoot(dir, seenFile, []byte("{\"devices\":[]}\n"), abcdhome.FileMode); err != nil {
			return err
		}
		res, err = launch(ctx, opts, self.Name, addrs, dir)
		return err
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return StartResult{}, refuse("another start or stop of the dashboard holds its lock; try again")
	}
	return res, err
}

// launch runs the server, waits for it to report it is listening, fetches
// its own address through the tailnet, and only then records the run. Any
// failure after the server started stops it before returning.
func launch(ctx context.Context, opts StartOptions, name string, addrs []netip.Addr, dir *os.Root) (StartResult, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return StartResult{}, err
	}
	selfCheck := hex.EncodeToString(nonce)
	cfg := childConfig{Wire: childConfigWire, Home: opts.Home, Tailscale: opts.Tailscale.Path, Name: name, Port: opts.Port, SelfCheck: selfCheck}
	for _, a := range addrs {
		cfg.Addrs = append(cfg.Addrs, a.String())
	}

	readyR, readyW, err := os.Pipe()
	if err != nil {
		return StartResult{}, err
	}
	defer readyR.Close()
	cfgR, cfgW, err := os.Pipe()
	if err != nil {
		readyW.Close()
		return StartResult{}, err
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		readyW.Close()
		cfgR.Close()
		cfgW.Close()
		return StartResult{}, err
	}
	cmd := exec.Command(opts.Launch.Path, opts.Launch.Args...)
	cmd.Env = append(os.Environ(), opts.Launch.Env...)
	cmd.Dir = opts.Root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	cmd.ExtraFiles = []*os.File{readyW, cfgR} // fd 3 and fd 4
	// Its own session, so closing the Terminal that started it does not
	// stop it (decision 12), and its own process group.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	err = cmd.Start()
	readyW.Close()
	cfgR.Close()
	devnull.Close()
	if err != nil {
		cfgW.Close()
		return StartResult{}, fmt.Errorf("starting the dashboard server: %w", err)
	}
	// Reap the server if it exits while this process still runs; once start
	// returns and exits, the server belongs to the system.
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	fail := func(err error) (StartResult, error) {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-exited
		return StartResult{}, err
	}

	data, _ := json.Marshal(cfg)
	_, werr := cfgW.Write(append(data, '\n'))
	cfgW.Close()
	if werr != nil {
		return fail(fmt.Errorf("handing the dashboard server its configuration: %w", werr))
	}
	msg, err := readReady(readyR)
	if err != nil {
		return fail(err)
	}
	if !msg.OK {
		return fail(refuse("the dashboard server refused to start: %s", msg.Error))
	}
	port := msg.Port
	for _, a := range addrs {
		if err := fetchSelf(ctx, name, a, port, selfCheck); err != nil {
			return fail(fmt.Errorf("the dashboard started, but %s could not be reached from this computer (%v), so it is stopped; check that nothing on this computer blocks the port", netip.AddrPortFrom(a, uint16(port)), err))
		}
	}
	pid := cmd.Process.Pid
	proc, err := readProcess(pid)
	if err != nil {
		return fail(fmt.Errorf("reading the dashboard server's process: %w", err))
	}
	run := RunFile{PID: pid, Process: proc, Name: name, Port: port, Since: time.Now().UTC().Truncate(time.Second)}
	for _, a := range addrs {
		run.Addrs = append(run.Addrs, netip.AddrPortFrom(a, uint16(port)).String())
	}
	data, _ = json.MarshalIndent(run, "", "  ")
	if err := fsutil.WriteFileAtomicInRoot(dir, runFile, append(data, '\n'), abcdhome.FileMode); err != nil {
		return fail(fmt.Errorf("recording the run in %s: %w", abcdhome.Display(stateDir, runFile), err))
	}
	res := StartResult{URL: run.URL(), Name: name, Port: port, Addrs: run.Addrs, PID: pid}
	res.Line = StartLine(res)
	return res, nil
}

// readReady reads the server's one line from the readiness pipe, bounded in
// size and time. The pipe closing first means the server exited.
func readReady(r *os.File) (readyMessage, error) {
	type result struct {
		msg readyMessage
		err error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := bufio.NewReader(io.LimitReader(r, 4096)).ReadBytes('\n')
		if err != nil && len(line) == 0 {
			ch <- result{err: errors.New("the dashboard server exited before it was listening")}
			return
		}
		var m readyMessage
		if err := json.Unmarshal(line, &m); err != nil {
			ch <- result{err: errors.New("the dashboard server's readiness line is unreadable")}
			return
		}
		ch <- result{msg: m}
	}()
	select {
	case res := <-ch:
		return res.msg, res.err
	case <-time.After(readyTimeout):
		return readyMessage{}, fmt.Errorf("the dashboard server did not report it was listening within %s", readyTimeout)
	}
}

// fetchSelf fetches the self-check from this computer's own address a, so the
// gate judges this computer as it judges any device, with the one-time value
// only this run knows.
func fetchSelf(ctx context.Context, name string, a netip.Addr, port int, value string) error {
	target := netip.AddrPortFrom(a, uint16(port)).String()
	d := &net.Dialer{Timeout: selfCheckWait, LocalAddr: &net.TCPAddr{IP: a.AsSlice()}}
	client := &http.Client{
		Timeout: selfCheckWait,
		Transport: &http.Transport{
			Proxy:             nil,
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return d.DialContext(ctx, network, target)
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort(name, strconv.Itoa(port))+"/self-check", nil)
	if err != nil {
		return err
	}
	req.Header.Set(selfCheckHeader, value)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("it answered %d", resp.StatusCode)
	}
	return nil
}

// StartLine is the one line start prints: where to open it, who can, every
// address it listens on, and how to stop it.
func StartLine(r StartResult) string {
	return "abcd dashboard: open " + r.URL + " on a device on your Tailscale network; anyone on that network can open it. " +
		"It listens on " + strings.Join(r.Addrs, " and ") + " only. `abcd dashboard stop` stops it."
}

// readRun reads the run file, or reports none. A run file abcd did not write
// in the shape it writes is an error, never a guess.
func readRun(home string) (RunFile, bool, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(stateDir, runFile), maxStateBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return RunFile{}, false, nil
	default:
		return RunFile{}, false, fmt.Errorf("%s cannot be read safely: %v", abcdhome.Display(stateDir, runFile), err)
	}
	var run RunFile
	if err := json.Unmarshal(data, &run); err != nil || run.PID <= 0 || run.Process.Start == "" {
		return RunFile{}, false, fmt.Errorf("%s is not a run file this abcd wrote; remove it if no dashboard runs", abcdhome.Display(stateDir, runFile))
	}
	return run, true, nil
}

// readRunning reads the run file and whether its process is the one start
// launched: the same pid, start time and executable, read now.
func readRunning(home string) (RunFile, bool, error) {
	run, ok, err := readRun(home)
	if err != nil || !ok {
		return run, false, err
	}
	alive, err := sameProcess(run)
	return run, alive, err
}

// sameProcess reports whether run's pid is still the process start launched.
func sameProcess(run RunFile) (bool, error) {
	cur, err := readProcess(run.PID)
	if errors.Is(err, errProcessGone) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return cur == run.Process, nil
}

// StopResult is what stop reports.
type StopResult struct {
	// Stopped is set when stop signalled the server and it is gone.
	Stopped bool `json:"stopped"`
	// Stale is set when a run file named a process that is gone or is no
	// longer the server; it was removed and nothing was signalled.
	Stale bool   `json:"stale"`
	URL   string `json:"url,omitempty"`
}

// Stop stops the server start launched. It reads the run file, checks that
// its process is the one start launched (pid, start time and executable,
// re-read just before the signal), sends it SIGTERM, waits for it to exit and
// for its addresses to answer nothing, and removes the run file. It never
// signals by name or pattern, and never signals a process that is not the
// one recorded.
func Stop(home string) (StopResult, error) {
	dir, err := fsutil.OpenHomeScope(home, abcdhome.Rel(stateDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StopResult{}, nil
		}
		return StopResult{}, err
	}
	defer dir.Close()
	var res StopResult
	err = fsutil.WithFileLock(abcdhome.Path(home, stateDir, lockFile), lockTimeout, func() error {
		run, ok, err := readRun(home)
		if err != nil || !ok {
			return err
		}
		res.URL = run.URL()
		alive, err := sameProcess(run)
		if err != nil {
			return err
		}
		if !alive {
			res.Stale = true
			return removeRun(dir)
		}
		if err := syscall.Kill(run.PID, syscall.SIGTERM); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				res.Stale = true
				return removeRun(dir)
			}
			return fmt.Errorf("signalling the dashboard server: %w", err)
		}
		deadline := time.Now().Add(stopWait)
		for {
			alive, err := sameProcess(run)
			if err != nil {
				return err
			}
			if !alive {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("the dashboard server (process %d) did not stop within %s; %s is kept", run.PID, stopWait, abcdhome.Display(stateDir, runFile))
			}
			time.Sleep(50 * time.Millisecond)
		}
		for _, a := range run.Addrs {
			if c, err := net.DialTimeout("tcp", a, time.Second); err == nil {
				c.Close()
				return fmt.Errorf("the dashboard server stopped, but something still answers at %s", a)
			}
		}
		res.Stopped = true
		return removeRun(dir)
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return StopResult{}, refuse("another start or stop of the dashboard holds its lock; try again")
	}
	return res, err
}

func removeRun(dir *os.Root) error {
	if err := dir.Remove(runFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Status is what the status verb reports.
type Status struct {
	Running bool         `json:"running"`
	URL     string       `json:"url,omitempty"`
	Addrs   []string     `json:"addresses,omitempty"`
	Since   *time.Time   `json:"since,omitempty"`
	PID     int          `json:"pid,omitempty"`
	Stale   bool         `json:"stale,omitempty"`
	Devices []SeenDevice `json:"devices"`
}

// ReadStatus reports whether the dashboard runs, where, since when, and the
// devices that opened it in this run. It writes nothing: a stale run file is
// reported, and left for stop or the next start.
func ReadStatus(home string) (Status, error) {
	st := Status{Devices: []SeenDevice{}}
	run, ok, err := readRun(home)
	if err != nil || !ok {
		return st, err
	}
	alive, err := sameProcess(run)
	if err != nil {
		return st, err
	}
	if !alive {
		st.Stale = true
		return st, nil
	}
	since := run.Since
	st.Running, st.URL, st.Addrs, st.Since, st.PID = true, run.URL(), run.Addrs, &since, run.PID
	devices, err := readSeen(home)
	if err != nil {
		return st, err
	}
	st.Devices = devices
	return st, nil
}

// RecordRunForTest writes a run file naming process pid as it stands now, so
// a front door's test can stand a live process in for the server.
func RecordRunForTest(home string, pid int, name string, port int, addrs []string) error {
	proc, err := readProcess(pid)
	if err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(stateDir), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	run := RunFile{PID: pid, Process: proc, Name: name, Port: port, Addrs: addrs, Since: time.Now().UTC().Truncate(time.Second)}
	data, _ := json.Marshal(run)
	return fsutil.WriteFileAtomicInRoot(dir, runFile, data, abcdhome.FileMode)
}
