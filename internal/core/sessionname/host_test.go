package sessionname

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// readAt is the fixed moment every fixture table was read at.
var readAt = time.Unix(1_800_000_000, 0)

// proc is one fixture process that has existed for age.
func proc(pid, ppid int, name string, age time.Duration) machineload.Proc {
	return machineload.Proc{PID: pid, PPID: ppid, Name: name, Age: age}
}

// hookChain is the live ancestry seen on 2026-10-15 (findings.md, probe 9):
// the host under a terminal multiplexer, the hook's sh, and abcd itself, with
// a second shell between them as a wrapper might add.
func hookChain() Table {
	return Table{ReadAt: readAt, Procs: []machineload.Proc{
		proc(1, 0, "launchd", 100*time.Hour),
		proc(50, 1, "tmux", 10*time.Hour),
		proc(100, 50, "claude", time.Hour),
		proc(200, 100, "sh", time.Second),
		proc(210, 200, "-zsh", time.Second),
		proc(300, 210, "abcd-darwin-arm64", time.Second),
	}}
}

// TestFindHostSkipsShellsAndAbcd is the spec's host process: the first
// ancestor of the hook that is neither a shell nor abcd, and its start time
// as the read time minus its age.
func TestFindHostSkipsShellsAndAbcd(t *testing.T) {
	for _, from := range []int{300, 210, 200} {
		h, err := FindHost(hookChain(), from)
		if err != nil {
			t.Fatalf("FindHost from %d: %v", from, err)
		}
		if h.PID != 100 || h.StartUnix != readAt.Add(-time.Hour).Unix() {
			t.Fatalf("FindHost from %d = %+v, want pid 100 started an hour before the read", from, h)
		}
	}
}

// TestFindHostRefusesAWalkThatFindsNoHost: a walk that leaves the table, that
// reaches the machine's first process, or that loops names no host, so no
// register entry can be keyed on launchd or on a process abcd cannot see.
func TestFindHostRefusesAWalkThatFindsNoHost(t *testing.T) {
	for name, tab := range map[string]Table{
		"absent": {ReadAt: readAt, Procs: []machineload.Proc{proc(5, 1, "claude", time.Hour)}},
		"only shells to init": {ReadAt: readAt, Procs: []machineload.Proc{
			proc(1, 0, "launchd", time.Hour), proc(300, 1, "sh", time.Second)}},
		"a loop": {ReadAt: readAt, Procs: []machineload.Proc{
			proc(300, 301, "sh", time.Second), proc(301, 300, "bash", time.Second)}},
	} {
		if h, err := FindHost(tab, 300); err == nil {
			t.Errorf("%s: FindHost = %+v, want an error", name, h)
		}
	}
}

// TestAliveTellsAReusedProcessIDFromTheSession is C9: an entry is live only
// while its process id is in the table AND that process started when the
// entry says, within two seconds. A pid reused by a later process, or gone,
// has ended.
func TestAliveTellsAReusedProcessIDFromTheSession(t *testing.T) {
	tab := hookChain()
	start := readAt.Add(-time.Hour).Unix()
	for name, c := range map[string]struct {
		h    Host
		want bool
	}{
		"same process":            {Host{PID: 100, StartUnix: start}, true},
		"within the tolerance":    {Host{PID: 100, StartUnix: start + 2}, true},
		"within it, earlier":      {Host{PID: 100, StartUnix: start - 2}, true},
		"a reused pid, later":     {Host{PID: 100, StartUnix: start - 600}, false},
		"just past the tolerance": {Host{PID: 100, StartUnix: start + 3}, false},
		"gone":                    {Host{PID: 4242, StartUnix: start}, false},
		"no pid":                  {Host{PID: 0, StartUnix: start}, false},
	} {
		if got := tab.Alive(c.h); got != c.want {
			t.Errorf("%s: Alive(%+v) = %v, want %v", name, c.h, got, c.want)
		}
	}
}

// TestAliveWidensForTheReadThatRecordedTheHost: a start time read from a slow
// table is early by up to the read's length, so an entry recorded from a
// three-second read is alive when a fast later reading puts its start three
// seconds later. Without the read's length (an entry written before it was
// recorded) the tolerance is unchanged, and the same start is a different
// process. The widening is capped, so a recorded length past the cap cannot
// keep a reused id alive.
func TestAliveWidensForTheReadThatRecordedTheHost(t *testing.T) {
	tab := hookChain()
	start := readAt.Add(-time.Hour).Unix()
	for name, c := range map[string]struct {
		h    Host
		want bool
	}{
		"recorded 3s early by a 3s read":     {Host{PID: 100, StartUnix: start - 3, Slack: 3 * time.Second}, true},
		"recorded 3s early, no read length":  {Host{PID: 100, StartUnix: start - 3}, false},
		"recorded 5s early by a 3s read":     {Host{PID: 100, StartUnix: start - 5, Slack: 3 * time.Second}, true},
		"recorded 6s early by a 3s read":     {Host{PID: 100, StartUnix: start - 6, Slack: 3 * time.Second}, false},
		"later than the reading, 3s read":    {Host{PID: 100, StartUnix: start + 3, Slack: 3 * time.Second}, false},
		"a reused pid, an hour-long read":    {Host{PID: 100, StartUnix: start - 600, Slack: time.Hour}, false},
		"just inside the cap, past it early": {Host{PID: 100, StartUnix: start - 2 - int64(maxReadSlack/time.Second), Slack: time.Hour}, true},
	} {
		if got := tab.Alive(c.h); got != c.want {
			t.Errorf("%s: Alive(%+v) = %v, want %v", name, c.h, got, c.want)
		}
	}
}
