package implement

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// No test in this package reads the real disk unless it asks: every wait and
// every check sees a disk with room, so a nearly full development machine or
// CI runner does not turn every gate test into a disk test.
func init() {
	ReadDisk = func(string) DiskCheck { return roomyDisk }
}

var roomyDisk = DiskCheck{Status: DiskOK, FloorBytes: DiskFloor, Volumes: []DiskVolume{}}

// withDisk makes ReadDisk judge p for the test's length.
func withDisk(t *testing.T, p diskProbe) {
	t.Helper()
	prev := ReadDisk
	ReadDisk = func(home string) DiskCheck { return checkDisk(home, p) }
	t.Cleanup(func() { ReadDisk = prev })
}

// fakeDisk is a probe for a home whose Go build cache and worktree store sit
// on volumes as vols lays them out: by path prefix, the volume and its free
// GiB, the longest prefix winning. The cache holds cacheGiB, estimated from 12 of 256 subdirectories.
func fakeDisk(home string, cacheGiB uint64, vols map[string]machineload.Volume) (diskProbe, *[]string) {
	var measured []string
	cache := filepath.Join(home, "Library", "Caches", "go-build")
	return diskProbe{
		goCache: func() string { return cache },
		volume: func(p string) (machineload.Volume, error) {
			best := ""
			for prefix := range vols {
				if strings.HasPrefix(p, prefix) && len(prefix) > len(best) {
					best = prefix
				}
			}
			if best == "" {
				return machineload.Volume{}, fmt.Errorf("no fixture volume for %s", p)
			}
			v := vols[best]
			v.Path = p
			return v, nil
		},
		cacheSize: func(dir string) (machineload.CacheSize, error) {
			measured = append(measured, dir)
			return machineload.CacheSize{Bytes: cacheGiB << 30, Sampled: 12, Subdirs: 256}, nil
		},
		floor: DiskFloor,
	}, &measured
}

const gib = uint64(1) << 30

// dataVolume is one volume holding the whole home with free bytes.
func dataVolume(home string, free uint64) map[string]machineload.Volume {
	return map[string]machineload.Volume{home: {Free: free, Total: 926 * gib, ID: "1:2", Mount: "/System/Volumes/Data"}}
}

// TestWaitRefusesAtOnceOnALowDisk is the detector for iss-2610281502581386: a
// free machine whose disk is nearly full is not admitted, the refusal comes at
// once (no look, no sleep), and it names the volume, its free space, the
// floor, the cache's path and size, and the trim.
func TestWaitRefusesAtOnceOnALowDisk(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 177, dataVolume(home, 658<<20))
	withDisk(t, p)
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 51000, clock, poll{load: 1})
	slept := false
	req.Sleep = func(time.Duration) { slept = true }
	res, err := WaitLoad(req)
	if err == nil {
		t.Fatalf("a nearly full disk was admitted: %+v", res)
	}
	if res.Polls != 0 || slept || res.Hold != nil {
		t.Fatalf("the refusal waited or held: polls %d, slept %v, hold %+v", res.Polls, slept, res.Hold)
	}
	msg := err.Error()
	for _, want := range []string{
		"658 MiB free on the volume mounted at /System/Volumes/Data",
		"the Go build cache (~/Library/Caches/go-build)",
		"the worktree store (~/" + abcdhome.Rel("worktrees") + ")",
		"the floor is 16.0 GiB",
		"about 177.0 GiB (estimated from 12 of its 256 subdirectories)",
		"find ~/Library/Caches/go-build -type f -mmin +2880 -delete",
		"go clean -cache",
		"nothing was waited for",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, home) {
		t.Errorf("the refusal carries the home's path:\n%s", msg)
	}

	// The race gate refuses too, even at one slot, where it otherwise opens
	// without a look.
	req.Race = true
	if _, err := WaitLoad(req); err == nil {
		t.Fatal("the race gate admitted a race stage on a nearly full disk")
	}
}

// TestWaitAdmitsAtTheFloor: a volume with the floor free, or more, is admitted
// as before.
func TestWaitAdmitsAtTheFloor(t *testing.T) {
	home := t.TempDir()
	p, measured := fakeDisk(home, 177, dataVolume(home, DiskFloor))
	withDisk(t, p)
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 51000, clock, poll{load: 1}))
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("res %+v, err %v", res, err)
	}
	if len(*measured) != 0 {
		t.Fatalf("the cache was walked on a disk with room: %v", *measured)
	}
}

// TestLoadCheckWarnsOnALowDisk: the load check reports the same finding as a
// warning beside the load's own status, which it does not change.
func TestLoadCheckWarnsOnALowDisk(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 177, dataVolume(home, 2*gib))
	withDisk(t, p)
	req := loadRequest(t, loadSnapshot(3), nil)
	req.Home = home
	res := CheckLoad(req)
	if res.Status != LoadOK {
		t.Fatalf("the load's status changed: %+v", res)
	}
	if res.Disk == nil || res.Disk.Status != DiskLow {
		t.Fatalf("no low-disk finding: %+v", res.Disk)
	}
	text := res.Disk.WarningText(SitePreflight)
	for _, want := range []string{"LOW DISK WARNING (preflight)", "2.0 GiB free", "-mmin +2880"} {
		if !strings.Contains(text, want) {
			t.Errorf("the warning does not say %q:\n%s", want, text)
		}
	}

	// With room, the report carries no disk finding at all.
	withDisk(t, func() diskProbe { p, _ := fakeDisk(home, 1, dataVolume(home, 500*gib)); return p }())
	if res := CheckLoad(req); res.Disk != nil {
		t.Fatalf("a disk with room was reported: %+v", res.Disk)
	}
}

// TestDiskVolumesApart: the cache and the store on different volumes are
// judged apart; only the low one is named, with the remedy that frees it.
func TestDiskVolumesApart(t *testing.T) {
	home := t.TempDir()
	vols := map[string]machineload.Volume{
		filepath.Join(home, "Library"): {Free: 400 * gib, ID: "1:1", Mount: "/"},
		home:                           {Free: 3 * gib, ID: "2:2", Mount: "/Volumes/Work"},
	}
	p, measured := fakeDisk(home, 50, vols)
	d := checkDisk(home, p)
	if d.Status != DiskLow || len(d.Volumes) != 2 {
		t.Fatalf("check = %+v", d)
	}
	if len(*measured) != 0 {
		t.Fatalf("the cache was measured though its volume has room: %v", *measured)
	}
	msg := d.Refusal().Error()
	if strings.Contains(msg, "Go build cache") || strings.Contains(msg, "-mmin") {
		t.Errorf("the refusal names the cache's volume, which has room:\n%s", msg)
	}
	for _, want := range []string{"3.0 GiB free on the volume mounted at /Volumes/Work", "git worktree remove"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, msg)
		}
	}
}

// TestDiskUnknownNeverRefuses: a platform whose free space cannot be read is
// unknown, and refuses nothing.
func TestDiskUnknownNeverRefuses(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 1, nil)
	p.volume = func(string) (machineload.Volume, error) { return machineload.Volume{}, machineload.ErrUnsupported }
	d := checkDisk(home, p)
	if d.Status != DiskUnknown || d.Refusal() != nil {
		t.Fatalf("check = %+v, refusal %v", d, d.Refusal())
	}
	if r := d.Report(); r == nil || !strings.Contains(r.WarningText(SitePreflight), "macOS and Linux only") {
		t.Fatalf("unknown is not reported: %+v", r)
	}

	// One volume read and the other failing still judges the one read.
	p, _ = fakeDisk(home, 1, dataVolume(home, gib))
	inner := p.volume
	p.volume = func(path string) (machineload.Volume, error) {
		if strings.Contains(path, "go-build") {
			return machineload.Volume{}, errors.New("statfs " + path + ": permission denied")
		}
		return inner(path)
	}
	d = checkDisk(home, p)
	if d.Status != DiskLow || !strings.Contains(d.Reason, "~/Library/Caches/go-build could not be read") || strings.Contains(d.Reason, home) {
		t.Fatalf("check = %+v", d)
	}
}

// TestDiskNoCacheReadsTheStore: with no Go build cache (GOCACHE=off, or none
// yet), only the store's volume is judged.
func TestDiskNoCacheReadsTheStore(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 1, dataVolume(home, gib))
	p.goCache = func() string { return "" }
	d := checkDisk(home, p)
	if d.Status != DiskLow || len(d.Volumes) != 1 || d.Volumes[0].Holds[0] != DiskHoldsWorktreeStore || d.GoCache != "" {
		t.Fatalf("check = %+v", d)
	}
}

// TestShellPathQuotes: the trim names the cache as a pasted command line takes
// it: the tilde bare, the rest quoted when it needs it.
func TestShellPathQuotes(t *testing.T) {
	home := "/Users/someone"
	for in, want := range map[string]string{
		"/Users/someone/Library/Caches/go-build":           "~/Library/Caches/go-build",
		"/Users/someone/Library/Application Support/cache": "~/'Library/Application Support/cache'",
		"/var/cache/go-build":                              "/var/cache/go-build",
		"/var/it's here":                                   `'/var/it'\''s here'`,
	} {
		if got := shellPath(home, in); got != want {
			t.Errorf("shellPath(%q) = %q, want %q", in, got, want)
		}
	}
}
