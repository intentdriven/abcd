package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/tailscale"
)

// childEnv marks a run of this test binary that must act as the dashboard
// server start launches, with loopback standing in for the tailnet.
const childEnv = "ABCD_DASHBOARD_TEST_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(childEnv) == "env" && len(os.Args) == 3 && os.Args[1] == "dashboard" && os.Args[2] == "serve" {
		// Report the names in this server's environment as its refusal.
		var names []string
		for _, kv := range os.Environ() {
			names = append(names, strings.SplitN(kv, "=", 2)[0])
		}
		data, _ := json.Marshal(readyMessage{Error: "environment: " + strings.Join(names, ",")})
		_, _ = os.NewFile(readyFD, "ready").Write(append(data, '\n'))
		os.Exit(2)
	}
	if v := os.Getenv(childEnv); (v == "1" || v == "dual") && len(os.Args) == 3 && os.Args[1] == "dashboard" && os.Args[2] == "serve" {
		setTailnetForTest(loopbackPrefixes)
		if v == "dual" {
			setTailnetForTest(dualLoopbackPrefixes)
		}
		if err := Serve(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "abcd dashboard serve:", err)
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// fakeTailscale is a `tailscale` command answering from files beside it, and
// logging every call: status.json, whois-<address>.json, serve.json.
type fakeTailscale struct {
	dir  string
	path string
}

const fakeScript = `#!/bin/sh
d=$(dirname "$0")
printf '%s\n' "$*" >> "$d/calls.log"
case "$1" in
status) cat "$d/status.json" ;;
whois)
  f="$d/whois-$3.json"
  if [ -f "$f" ]; then cat "$f"; else echo "no match for IP:port" >&2; exit 1; fi ;;
serve)
  if [ -f "$d/serve.json" ]; then cat "$d/serve.json"; else echo '{}'; fi ;;
*) echo "the fake refuses: $*" >&2; exit 2 ;;
esac
`

func newFakeTailscale(t *testing.T) *fakeTailscale {
	t.Helper()
	dir := t.TempDir()
	f := &fakeTailscale{dir: dir, path: filepath.Join(dir, "tailscale")}
	f.write(t, "tailscale", fakeScript)
	if err := os.Chmod(f.path, 0o755); err != nil {
		t.Fatal(err)
	}
	f.write(t, "status.json", `{"Version":"1.102.4","BackendState":"Running","Self":{"DNSName":"`+testName+`.","TailscaleIPs":["127.0.0.1"]}}`)
	f.write(t, "whois-127.0.0.1.json", `{"Node":{"StableID":"nThis","ComputedName":"dash"},"UserProfile":{"LoginName":"pt@example.com","DisplayName":"Product Thinker"}}`)
	return f
}

func (f *fakeTailscale) write(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeTailscale) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, "calls.log"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return strings.Fields(strings.ReplaceAll(string(data), " ", "_"))
}

// startOpts is a start against the fake, launching this test binary as the
// server, with loopback standing in for the tailnet in this process too.
func startOpts(t *testing.T, f *fakeTailscale, home string) StartOptions {
	t.Helper()
	restore := setTailnetForTest(loopbackPrefixes)
	t.Cleanup(restore)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return StartOptions{
		Home:      home,
		Root:      t.TempDir(),
		Port:      0,
		Tailscale: tailscale.New(f.path),
		Launch:    Launcher{Path: exe, Args: []string{"dashboard", "serve"}, Env: []string{childEnv + "=1"}},
	}
}

// startForTest starts a dashboard and stops it when the test ends.
func startForTest(t *testing.T, opts StartOptions) StartResult {
	t.Helper()
	res, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if run, ok, _ := readRun(opts.Home); ok {
			if alive, _ := sameProcess(run); alive {
				_ = syscall.Kill(run.PID, syscall.SIGKILL)
			}
		}
	})
	return res
}

func answers(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func TestStartLineNamesAddressAndWhoCan(t *testing.T) {
	f := newFakeTailscale(t)
	home := t.TempDir()
	res := startForTest(t, startOpts(t, f, home))
	if len(res.Addrs) != 1 || !strings.HasPrefix(res.Addrs[0], "127.0.0.1:") {
		t.Fatalf("addresses = %v, want the one tailnet address", res.Addrs)
	}
	for _, want := range []string{
		"http://" + testName + ":" + fmt.Sprint(res.Port),
		"on a device on your Tailscale network",
		"anyone on that network can open it",
		res.Addrs[0],
		"`abcd dashboard stop` stops it",
	} {
		if !strings.Contains(res.Line, want) {
			t.Errorf("the start line does not say %q:\n%s", want, res.Line)
		}
	}
	if strings.Contains(res.Line, "\n") {
		t.Errorf("the start line is more than one line:\n%s", res.Line)
	}
	if !answers(res.Addrs[0]) {
		t.Errorf("nothing answers at %s after start", res.Addrs[0])
	}
	for _, c := range f.calls(t) {
		if strings.HasPrefix(c, "cert") || strings.HasPrefix(c, "serve_") && c != "serve_status_--json" || strings.HasPrefix(c, "funnel") || strings.HasPrefix(c, "up") || strings.HasPrefix(c, "set") {
			t.Errorf("start ran `tailscale %s`, which changes Tailscale or asks for a certificate", strings.ReplaceAll(c, "_", " "))
		}
	}
	// One dashboard per computer (D5).
	if _, err := Start(context.Background(), startOpts(t, f, home)); !isRefusal(err) || !strings.Contains(err.Error(), "already runs") {
		t.Errorf("a second start = %v, want the already-running refusal", err)
	}
}

func isRefusal(err error) bool {
	var r *Refusal
	return errors.As(err, &r)
}

func TestStartChecksItCanReachItself(t *testing.T) {
	f := newFakeTailscale(t)
	// The lookup does not know this computer's address, so the gate refuses
	// the self-fetch exactly as it would a stranger's.
	if err := os.Remove(filepath.Join(f.dir, "whois-127.0.0.1.json")); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	_, err := Start(context.Background(), startOpts(t, f, home))
	if err == nil || !strings.Contains(err.Error(), "could not be reached from this computer") {
		t.Fatalf("Start = %v, want the could-not-reach failure", err)
	}
	if _, ok, _ := readRun(home); ok {
		t.Error("a start that could not reach itself left a run file")
	}
	calls := strings.Join(f.calls(t), " ")
	if !strings.Contains(calls, "whois") {
		t.Errorf("the self-fetch never reached the gate's lookup: %s", calls)
	}
}

// dualLoopbackPrefixes stands both loopback addresses in for the tailnet's
// IPv4 and IPv6 addresses.
var dualLoopbackPrefixes = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")}

// TestStartDropsAnAddressItCannotConnectTo holds start to the addresses that
// answer its self-check (iss-2610090841317423): on a Mac whose Tailscale
// cannot connect to its own IPv6 address while its IPv4 address answers,
// start keeps the IPv4 address, stops listening on the IPv6 one, and names
// it, and why, in its line and in its JSON, rather than stopping the
// dashboard.
func TestStartDropsAnAddressItCannotConnectTo(t *testing.T) {
	probe, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("this machine cannot listen on ::1: %v", err)
	}
	probe.Close()
	f := newFakeTailscale(t)
	f.write(t, "status.json", `{"Version":"1.102.4","BackendState":"Running","Self":{"DNSName":"`+testName+`.","TailscaleIPs":["127.0.0.1","::1"]}}`)
	home := t.TempDir()
	opts := startOpts(t, f, home)
	opts.Launch.Env = []string{childEnv + "=dual"}
	restoreTailnet := setTailnetForTest(dualLoopbackPrefixes)
	defer restoreTailnet()
	var dialled []string
	restoreDial := setDialSelfForTest(func(d *net.Dialer, ctx context.Context, network, address string) (net.Conn, error) {
		dialled = append(dialled, address)
		if strings.HasPrefix(address, "[::1]:") {
			return nil, &net.OpError{Op: "dial", Net: network, Addr: net.TCPAddrFromAddrPort(netip.MustParseAddrPort(address)), Err: os.ErrDeadlineExceeded}
		}
		return d.DialContext(ctx, network, address)
	})
	defer restoreDial()

	res := startForTest(t, opts)
	v4 := fmt.Sprintf("127.0.0.1:%d", res.Port)
	v6 := fmt.Sprintf("[::1]:%d", res.Port)
	if len(dialled) != 2 {
		t.Errorf("the self-check dialled %v, want both addresses", dialled)
	}
	if len(res.Addrs) != 1 || res.Addrs[0] != v4 {
		t.Errorf("addresses = %v, want only %s, the one that answered", res.Addrs, v4)
	}
	for _, want := range []string{v4, v6, "could not connect", "i/o timeout", "`abcd dashboard stop` stops it"} {
		if !strings.Contains(res.Line, want) {
			t.Errorf("the start line does not say %q:\n%s", want, res.Line)
		}
	}
	if strings.Contains(res.Line, "\n") {
		t.Errorf("the start line is more than one line:\n%s", res.Line)
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Addrs   []string `json:"addresses"`
		Dropped []struct {
			Addr   string `json:"address"`
			Reason string `json:"reason"`
		} `json:"dropped"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Dropped) != 1 || payload.Dropped[0].Addr != v6 || !strings.Contains(payload.Dropped[0].Reason, "i/o timeout") {
		t.Errorf("--json dropped = %+v in %s, want %s and why", payload.Dropped, data, v6)
	}
	// The dropped address is no longer listened on; the kept one answers.
	if !answers(v4) {
		t.Errorf("nothing answers at %s, the address start kept", v4)
	}
	if answers(v6) {
		t.Errorf("%s still answers: a dropped address must stop being listened on", v6)
	}
	run, ok, err := readRun(home)
	if err != nil || !ok {
		t.Fatalf("readRun = %v, %v", ok, err)
	}
	if len(run.Addrs) != 1 || run.Addrs[0] != v4 {
		t.Errorf("the run file names %v, want only %s", run.Addrs, v4)
	}
}

// TestStartNeverSignalsAServerThatAlreadyExited holds start's clean-up to
// the server it launched: a server that has exited was reaped, so its
// process group's number may by now name another program's processes, and
// start must not kill that group. A server still running when start gives up
// is killed through its group.
func TestStartNeverSignalsAServerThatAlreadyExited(t *testing.T) {
	var killed []int
	restore := setKillGroupForTest(func(pid int) error {
		killed = append(killed, pid)
		return syscall.Kill(-pid, syscall.SIGKILL)
	})
	defer restore()
	f := newFakeTailscale(t)
	for _, c := range []struct {
		name   string
		script string
	}{
		{"exits at once", "exit 0"},
		{"refuses and exits", `printf '%s\n' '{"ok":false,"error":"refused by the test"}' >&3`},
	} {
		killed = nil
		opts := startOpts(t, f, t.TempDir())
		opts.Launch = Launcher{Path: "/bin/sh", Args: []string{"-c", c.script}}
		if _, err := Start(context.Background(), opts); err == nil {
			t.Fatalf("%s: Start succeeded with a server that never listened", c.name)
		}
		if len(killed) != 0 {
			t.Errorf("%s: start killed process group %v of a server that had already exited", c.name, killed)
		}
	}

	// A server that reported it listens, on a port where nothing answers,
	// is still running when the self-fetch fails: start kills its group.
	killed = nil
	opts := startOpts(t, f, t.TempDir())
	opts.Launch = Launcher{Path: "/bin/sh", Args: []string{"-c", `printf '%s\n' '{"ok":true,"port":1}' >&3; exec sleep 30`}}
	if _, err := Start(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "could not be reached") {
		t.Fatalf("Start = %v, want the could-not-reach failure", err)
	}
	if len(killed) != 1 {
		t.Errorf("start killed %v, want the one running server's group", killed)
	}
}

// TestStartGivesAServerThatClosedItsConfigurationTimeToExit holds start to
// the grace it gives a server on its way out when the way out shows as the
// configuration write failing: a server that closed its configuration pipe
// (or exited) before start wrote to it is exiting, not stuck, so start waits
// exitGrace for it rather than killing its group at once, and a server that
// then exits is never signalled. On Linux a server that exits at once makes
// the write fail this way; here the server closes the pipe and lingers, so
// the order is fixed rather than raced.
func TestStartGivesAServerThatClosedItsConfigurationTimeToExit(t *testing.T) {
	var killed []int
	restore := setKillGroupForTest(func(pid int) error {
		killed = append(killed, pid)
		return syscall.Kill(-pid, syscall.SIGKILL)
	})
	defer restore()
	mark := filepath.Join(t.TempDir(), "config-closed")
	restoreWrite := setBeforeConfigWriteForTest(func() {
		deadline := time.Now().Add(10 * time.Second)
		for {
			if _, err := os.Stat(mark); err == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Error("the server never closed its configuration pipe")
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	})
	defer restoreWrite()
	f := newFakeTailscale(t)
	opts := startOpts(t, f, t.TempDir())
	opts.Launch = Launcher{Path: "/bin/sh", Args: []string{"-c", `exec 4<&-; : > "$1"; sleep 0.5`, "sh", mark}}
	_, err := Start(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "handing the dashboard server its configuration") {
		t.Fatalf("Start = %v, want the failed configuration write", err)
	}
	if len(killed) != 0 {
		t.Errorf("start killed process group %v of a server that was exiting within %s", killed, exitGrace)
	}
}

// TestServerInheritsOnlyWhatItNeeds holds the long-lived, network-facing
// server to the environment it needs (the path and home its Tailscale
// lookups run with, and start's marker): a token or credential in the
// environment start was run from does not live on in it.
func TestServerInheritsOnlyWhatItNeeds(t *testing.T) {
	t.Setenv("ABCD_TEST_SECRET_TOKEN", "a value the server must not hold")
	f := newFakeTailscale(t)
	opts := startOpts(t, f, t.TempDir())
	opts.Launch.Env = []string{childEnv + "=env"}
	_, err := Start(context.Background(), opts)
	if err == nil {
		t.Fatal("Start succeeded with a server that reports its environment and exits")
	}
	_, list, ok := strings.Cut(err.Error(), "environment: ")
	if !ok {
		t.Fatalf("Start = %v, want the server's report of its environment", err)
	}
	got := map[string]bool{}
	for _, name := range strings.Split(list, ",") {
		got[name] = true
	}
	want := map[string]bool{serveMarkerEnv: true, childEnv: true}
	for _, name := range []string{"PATH", "HOME"} {
		if _, set := os.LookupEnv(name); set {
			want[name] = true
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("the server inherited %s from start's environment", name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("the server's environment lacks %s", name)
		}
	}
}

func TestStartRefusesAServedPort(t *testing.T) {
	f := newFakeTailscale(t)
	f.write(t, "serve.json", `{"TCP":{"8080":{"HTTP":true}},"AllowFunnel":{"`+testName+`:8443":true}}`)
	home := t.TempDir()
	for _, port := range []int{8080, 8443} {
		opts := startOpts(t, f, home)
		opts.Port = port
		opts.Launch = Launcher{Path: filepath.Join(t.TempDir(), "never-run")}
		_, err := Start(context.Background(), opts)
		if !isRefusal(err) || !strings.Contains(err.Error(), fmt.Sprintf("port %d", port)) {
			t.Errorf("port %d: Start = %v, want the served-port refusal naming it", port, err)
		}
	}
	if _, ok, _ := readRun(home); ok {
		t.Error("a refused start left a run file")
	}
}

func TestStartRefusesWhenTailscaleIsNotRunning(t *testing.T) {
	f := newFakeTailscale(t)
	f.write(t, "status.json", `{"Version":"1.102.4","BackendState":"Stopped"}`)
	opts := startOpts(t, f, t.TempDir())
	opts.Launch = Launcher{Path: filepath.Join(t.TempDir(), "never-run")}
	if _, err := Start(context.Background(), opts); !isRefusal(err) || !strings.Contains(err.Error(), "not running") {
		t.Errorf("Start = %v, want the not-running refusal", err)
	}
}

func TestStopLeavesNothingListening(t *testing.T) {
	f := newFakeTailscale(t)
	home := t.TempDir()
	res := startForTest(t, startOpts(t, f, home))
	if !answers(res.Addrs[0]) {
		t.Fatalf("nothing answers at %s before stop", res.Addrs[0])
	}
	got, err := Stop(home)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !got.Stopped || got.Stale {
		t.Errorf("Stop = %+v, want stopped", got)
	}
	if answers(res.Addrs[0]) {
		t.Errorf("%s still answers after stop", res.Addrs[0])
	}
	if _, ok, _ := readRun(home); ok {
		t.Error("stop left the run file")
	}
	st, err := ReadStatus(home)
	if err != nil || st.Running {
		t.Errorf("status after stop = %+v, %v; want not running", st, err)
	}
	// A second stop finds nothing to stop and says so.
	if got, err := Stop(home); err != nil || got.Stopped || got.Stale {
		t.Errorf("a second stop = %+v, %v; want nothing done", got, err)
	}
}

func TestStopChecksTheProcessBeforeSignalling(t *testing.T) {
	home := t.TempDir()
	// A live process that is not the server: the run file names its pid, but
	// with another start time, as after a pid is reused.
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Process.Kill(); _, _ = other.Process.Wait() }()
	if err := RecordRunForTest(home, other.Process.Pid, testName, 8080, []string{"127.0.0.1:8080"}); err != nil {
		t.Fatal(err)
	}
	path := abcdhome.Path(home, stateDir, runFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var run RunFile
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	run.Process.Start = "a start time that is not this process's"
	data, _ = json.Marshal(run)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Stop(home)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !got.Stale || got.Stopped {
		t.Errorf("Stop = %+v, want the stale run file reported and nothing stopped", got)
	}
	if err := other.Process.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("stop signalled a process that was not the server: %v", err)
	}
	if _, ok, _ := readRun(home); ok {
		t.Error("a stale run file was kept")
	}

	// The same for an executable that differs.
	if err := RecordRunForTest(home, other.Process.Pid, testName, 8080, []string{"127.0.0.1:8080"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	_ = json.Unmarshal(data, &run)
	run.Process.Exe = "/not/the/server"
	data, _ = json.Marshal(run)
	_ = os.WriteFile(path, data, 0o600)
	if got, err := Stop(home); err != nil || !got.Stale {
		t.Errorf("Stop with another executable = %+v, %v; want stale", got, err)
	}
	if err := other.Process.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("stop signalled a process running another executable: %v", err)
	}
}

func TestOnlyStartStartsTheServer(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Without start's marker, and with the marker but without start's pipes,
	// the server refuses and exits.
	for _, env := range [][]string{{childEnv + "=1"}, {childEnv + "=1", serveMarkerEnv + "=1"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, exe, "dashboard", "serve")
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		cancel()
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("serve with %v and no pipes = %v, want it to refuse and exit", env, err)
		}
		if !strings.Contains(string(out), "started only by `abcd dashboard start`") {
			t.Errorf("with %v the refusal does not say only start starts it:\n%s", env, out)
		}
	}
	// Pipes at 3 and 4 without start's marker are not start's either: the
	// server refuses without reading them.
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "dashboard", "serve")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	cmd.ExtraFiles = []*os.File{w1, r2}
	out, err := cmd.CombinedOutput()
	w1.Close()
	r2.Close()
	r1.Close()
	w2.Close()
	if !strings.Contains(string(out), "started only by `abcd dashboard start`") {
		t.Errorf("pipes without start's marker = %v:\n%s; want the refusal", err, out)
	}

	// Called in this process, whose descriptors 3 and 4 are not start's
	// pipes, it refuses the same way and touches neither.
	if err := Serve(context.Background()); !errors.Is(err, error(errNotFromStart)) {
		t.Errorf("Serve in-process = %v, want the refusal", err)
	}
}

// TestNothingInstallsALoginItem holds "never starts by itself": no code in
// the dashboard or its front door names a login item, a launch agent, a
// service unit or a scheduled job.
func TestNothingInstallsALoginItem(t *testing.T) {
	root := repoRoot(t)
	needles := []string{"LaunchAgents", "LaunchDaemons", "launchctl", "systemctl", "systemd", "crontab", "SMAppService", "LoginItems"}
	for _, dir := range []string{"internal/surface/dashboard", "internal/adapter/tailscale"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filepath.Join(root, dir, e.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					for _, needle := range needles {
						if strings.Contains(lit.Value, needle) {
							t.Errorf("%s/%s:%d names %q: the dashboard never starts by itself", dir, e.Name(), fset.Position(lit.Pos()).Line, needle)
						}
					}
				}
				return true
			})
		}
	}
}

func TestStatusReportsTheRunAndItsDevices(t *testing.T) {
	f := newFakeTailscale(t)
	home := t.TempDir()
	res := startForTest(t, startOpts(t, f, home))
	st, err := ReadStatus(home)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.URL != res.URL || st.Since == nil || len(st.Devices) != 0 {
		t.Errorf("status = %+v; want running at %s with no device yet (the self-check is not a device)", st, res.URL)
	}
	// This computer is refused once start's self-check has answered: a page
	// asked for from it gets nothing, and it is not listed as a device.
	c, err := net.Dial("tcp", res.Addrs[0])
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", testName)
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	got, _ := io.ReadAll(c)
	c.Close()
	if len(got) != 0 {
		t.Errorf("a page asked for from this computer after start got %d bytes: %q", len(got), got)
	}
	st, err = ReadStatus(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Devices) != 0 {
		t.Errorf("devices = %+v, want none: this computer is not a device that opened it", st.Devices)
	}
	// A device the server records opening a page is listed. Connections
	// from another device cannot be made from this one, so the server's
	// recorder stands in; TestTailscaleHeadersAreIgnored holds that a page
	// opened records the lookup's device.
	newSeenRecorder(home).record(tailscale.Identity{Node: "nPhone", Device: "phone", Login: "pt@example.com", Person: "Product Thinker"})
	st, err = ReadStatus(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Devices) != 1 || st.Devices[0].Login != "pt@example.com" || st.Devices[0].Device != "phone" {
		t.Errorf("devices = %+v, want the one device that opened a page", st.Devices)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test")
		}
		dir = parent
	}
}

// A self-check whose dial fails because start was cancelled reports the
// cancellation, never a *noConnectionError: a cancelled start must stop the
// server, not drop the address and report success.
func TestFetchSelfReportsCancellationNotNoConnection(t *testing.T) {
	restore := setDialSelfForTest(func(_ *net.Dialer, ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: ctx.Err()}
	})
	defer restore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := fetchSelf(ctx, "localhost", netip.MustParseAddr("127.0.0.1"), 1, "value")
	var nc *noConnectionError
	if errors.As(err, &nc) {
		t.Fatalf("fetchSelf on a cancelled start = %v, a *noConnectionError; want the cancellation", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("fetchSelf on a cancelled start = %v, want context.Canceled", err)
	}
}
