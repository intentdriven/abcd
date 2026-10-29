package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// No test in this package reads the real disk: every load check and wait sees
// a disk with room, so a nearly full development machine or CI runner does not
// turn the load verb's tests into disk tests (implement.ReadDisk).
func init() {
	implement.ReadDisk = func(string, uint64) implement.DiskCheck { return implement.DiskCheck{Status: implement.DiskOK} }
}

// TestLoadCheckRendersTheLowDiskWarning: the load check prints the low-disk
// finding beside the load's own line, and prints nothing more without one
// (iss-2610281502581386).
func TestLoadCheckRendersTheLowDiskWarning(t *testing.T) {
	res := implement.LoadResult{Status: implement.LoadOK, Site: implement.SitePreflight, Load: &implement.LoadAverages{One: 3}, Cores: 16,
		Disk: &implement.DiskCheck{Status: implement.DiskLow, FloorBytes: implement.DiskFloorFor(1), Volumes: []implement.DiskVolume{{
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
