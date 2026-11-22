package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// No test in this package reads the real disk or the real memory pressure, or
// trims a real cache: every load check and wait sees a disk with room, macOS
// at its normal memory level and a trim at the floor that frees nothing, so a
// nearly full or memory-pressed development machine or CI runner does not turn
// the load verb's tests into disk or memory tests (implement.ReadDisk,
// implement.ReadMemory, implement.TrimAtFloor). It is the one init among this
// package's tests: the plain-only guard scan (internal/core/lint) refuses a
// function name a package declares twice.
func init() {
	implement.ReadDisk = func(string, uint64) implement.DiskCheck { return implement.DiskCheck{Status: implement.DiskOK} }
	implement.ReadMemory = func() (machineload.Memory, error) {
		return machineload.Memory{Platform: machineload.MemoryDarwin, Level: machineload.MemoryNormal, Raw: "1"}, nil
	}
	implement.TrimAtFloor = func(string, string, time.Time, func(string)) worktree.FloorTrim {
		return worktree.FloorTrim{None: "no cache is trimmed in this package's tests"}
	}
}

// TestLoadCheckRendersTheLowDiskWarning: the load check prints the low-disk
// finding beside the load's own line, and prints nothing more without one
// (iss-2610281502581386).
func TestLoadCheckRendersTheLowDiskWarning(t *testing.T) {
	res := implement.LoadResult{Status: implement.LoadOK, Site: implement.SitePreflight, Load: &implement.LoadAverages{One: 3}, Cores: 16,
		Disk: &implement.DiskCheck{Status: implement.DiskLow, FloorBytes: implement.DiskFloorMin, Volumes: []implement.DiskVolume{{
			Holds: []string{implement.DiskHoldsWorktreeStore}, Paths: []string{"~/x"}, FreeBytes: 1 << 30, Low: true}}}}
	var b bytes.Buffer
	renderLoad(&b, res)
	if !strings.Contains(b.String(), "LOW DISK WARNING (preflight)") || !strings.Contains(b.String(), "1.0 GiB free") {
		t.Fatalf("no low-disk warning:\n%s", b.String())
	}
	res.Disk = nil
	b.Reset()
	renderLoad(&b, res)
	if strings.Contains(b.String(), "DISK") {
		t.Fatalf("a warning without a finding:\n%s", b.String())
	}
}

// TestImplementLoadWaitSaysTheDiskWasNotChecked: a waiter that cannot read
// the disk opens as before, and prints one line on stderr saying the disk was
// not checked, and why.
func TestImplementLoadWaitSaysTheDiskWasNotChecked(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	prev := implement.ReadDisk
	implement.ReadDisk = func(string, uint64) implement.DiskCheck {
		return implement.DiskCheck{Status: implement.DiskUnknown, Volumes: []implement.DiskVolume{}, Reason: "abcd reads free space on macOS and Linux only"}
	}
	t.Cleanup(func() { implement.ReadDisk = prev })
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	want := "abcd implement load --wait: the disk was not checked: abcd reads free space on macOS and Linux only"
	if strings.Count(errOut, want) != 1 {
		t.Fatalf("stderr does not say once %q:\n%s", want, errOut)
	}
}

// TestImplementLoadWaitSaysTheTrimAtTheFloor: a waiter that finds the Go build
// cache's volume under the floor says on stderr, as it goes, that it is
// trimming, what the trim freed and the free space after, and opens once the
// trim cleared the floor (iss-2611220942266387).
func TestImplementLoadWaitSaysTheTrimAtTheFloor(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	free := uint64(3 << 30)
	prevDisk, prevTrim := implement.ReadDisk, implement.TrimAtFloor
	implement.ReadDisk = func(_ string, floor uint64) implement.DiskCheck {
		low := free < floor
		status := implement.DiskOK
		if low {
			status = implement.DiskLow
		}
		return implement.DiskCheck{Status: status, FloorBytes: floor, Volumes: []implement.DiskVolume{{
			Holds: []string{implement.DiskHoldsGoCache}, Paths: []string{"~/Library/Caches/go-build"}, FreeBytes: free, Low: low}}}
	}
	implement.TrimAtFloor = func(_, _ string, _ time.Time, before func(string)) worktree.FloorTrim {
		before("the Go build cache")
		free = 40 << 30
		return worktree.FloorTrim{Steps: []worktree.ToolchainTrim{{Toolchain: "go", Cache: "the Go build cache",
			Trim: &worktree.CacheTrim{Path: "~/Library/Caches/go-build", EntriesRemoved: 5, BytesFreed: 37 << 30, LeftBytes: 100 << 30, LimitBytes: 100 << 30, KeepHours: 3}}}}
	}
	t.Cleanup(func() { implement.ReadDisk, implement.TrimAtFloor = prevDisk, prevTrim })
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--timeout", "1m")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"abcd implement load --wait: 3.0 GiB free on the volume holding the Go build cache, under the floor of 8.0 GiB: trimming",
		"abcd implement load --wait: trimming the Go build cache now",
		"abcd implement load --wait: the Go build cache ~/Library/Caches/go-build: removed 5 entries, 37.0 GiB freed",
		"abcd implement load --wait: 40.0 GiB free after the trim, over the floor of 8.0 GiB",
	} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr does not say %q:\n%s", want, errOut)
		}
	}
}
