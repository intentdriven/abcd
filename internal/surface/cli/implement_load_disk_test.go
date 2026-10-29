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
	implement.ReadDisk = func(string) implement.DiskCheck { return implement.DiskCheck{Status: implement.DiskOK} }
}

// TestLoadCheckRendersTheLowDiskWarning: the load check prints the low-disk
// finding beside the load's own line, and prints nothing more without one
// (iss-2610281502581386).
func TestLoadCheckRendersTheLowDiskWarning(t *testing.T) {
	res := implement.LoadResult{Status: implement.LoadOK, Site: implement.SitePreflight, Load: &implement.LoadAverages{One: 3}, Cores: 16,
		Disk: &implement.DiskCheck{Status: implement.DiskLow, FloorBytes: implement.DiskFloor, Volumes: []implement.DiskVolume{{
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
