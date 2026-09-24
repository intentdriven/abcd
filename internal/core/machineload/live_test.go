//go:build darwin || linux

package machineload

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// busyHelperEnv marks a re-execution of this test binary as one busy child.
const busyHelperEnv = "ABCD_MACHINELOAD_BUSY_CHILD"

// busyFor is how long each child holds a core.
const busyFor = 3 * time.Second

// TestBusyChild is not a test: it is the body a re-executed child runs. Without
// the marker it returns at once.
func TestBusyChild(t *testing.T) {
	if os.Getenv(busyHelperEnv) != "1" {
		return
	}
	end := time.Now().Add(busyFor)
	for n := 0; time.Now().Before(end); n++ {
		_ = n * n
	}
	os.Exit(0)
}

// TestEightConcurrentShortLivedProcessesDoNotWarn starts short-lived busy
// children as ONE owned process group, reads the real machine, and classifies
// with the default stray limit: none of them is a stray. It then advances the
// same snapshot's children past the limit and asserts every one is named, so
// the test can fail. The machine's real load is not the subject, so the extreme
// limit is set above it.
//
// The LOAD rule governs this: one owned group, killed as a group only after the
// handle is re-checked, and a cap below the core count. The spec asks for eight
// children; on a machine with fewer than nine cores the count is the core count
// less one. The re-check is structural: the group leader is this process's
// unreaped child, and an unreaped child's pid cannot be reused, so the group id
// still names what was started when it is signalled.
func TestEightConcurrentShortLivedProcessesDoNotWarn(t *testing.T) {
	if testing.Short() {
		t.Skip("starts busy children")
	}
	n := min(8, runtime.NumCPU()-1)
	if n < 1 {
		t.Skip("a single-core machine has no core to spare below the LOAD rule's cap")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var children []*exec.Cmd
	pgid := 0
	t.Cleanup(func() {
		if pgid > 0 && children[0].ProcessState == nil {
			if got, err := syscall.Getpgid(pgid); err == nil && got == pgid {
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
					t.Errorf("stopping the group %d: %v", pgid, err)
				}
			}
		}
		for _, c := range children {
			_ = c.Wait()
		}
		// Proven gone by what is running, never by the list of what was started.
		snap, err := Read()
		if err != nil {
			t.Errorf("re-reading after the cleanup: %v", err)
			return
		}
		for _, p := range snap.Procs {
			if pgid > 0 && p.PGID == pgid {
				t.Errorf("pid %d of the busy group %d is still running", p.PID, pgid)
			}
		}
	})
	for i := 0; i < n; i++ {
		c := exec.Command(self, "-test.run=^TestBusyChild$")
		c.Env = append(os.Environ(), busyHelperEnv+"=1")
		// The first child leads a new group; the rest join it.
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pgid}
		if err := c.Start(); err != nil {
			t.Fatalf("starting child %d: %v", i, err)
		}
		children = append(children, c)
		if i == 0 {
			pgid = c.Process.Pid
		}
	}
	time.Sleep(500 * time.Millisecond)

	snap, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	pids := map[int]bool{}
	for _, c := range children {
		pids[c.Process.Pid] = true
	}
	seen := 0
	for _, p := range snap.Procs {
		if pids[p.PID] {
			seen++
			if p.PGID != pgid {
				t.Fatalf("child %d is in group %d, not the owned group %d", p.PID, p.PGID, pgid)
			}
		}
	}
	if seen != n {
		t.Fatalf("the snapshot holds %d of the %d children", seen, n)
	}

	lim := DefaultLimits(snap.Cores)
	lim.ExtremeLoad = snap.Load1 + 1000
	me := Self{PID: os.Getpid(), UID: uint32(os.Geteuid())}
	for _, s := range Classify(snap, lim, me).Own {
		if pids[s.PID] {
			t.Fatalf("a %v-old child was named a stray: %+v", busyFor, s)
		}
	}

	// The same snapshot with the children run past the limit at a full core.
	past := 31 * time.Minute
	aged := snap
	aged.Procs = append([]Proc(nil), snap.Procs...)
	for i, p := range aged.Procs {
		if pids[p.PID] {
			aged.Procs[i].Age += past
			aged.Procs[i].CPU += past
		}
	}
	named := 0
	for _, s := range Classify(aged, lim, me).Own {
		if pids[s.PID] {
			named++
		}
	}
	if named != n {
		t.Fatalf("%d of the %d aged children were named strays", named, n)
	}
}
