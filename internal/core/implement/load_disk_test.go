package implement

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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
	ReadDisk = func(_ string, floor uint64) DiskCheck {
		return DiskCheck{Status: DiskOK, FloorBytes: floor, Volumes: []DiskVolume{}}
	}
}

// productionReadDisk is ReadDisk as the binary runs it, kept before init
// replaces it (package variables are set before any init runs).
var productionReadDisk = ReadDisk

// withDisk makes ReadDisk judge p, at the floor the gate asks for, for the
// test's length.
func withDisk(t *testing.T, p diskProbe) {
	t.Helper()
	withReadDisk(t, func(home string, floor uint64) DiskCheck {
		q := p
		q.floor = floor
		return checkDisk(home, q)
	})
}

// withReadDisk makes ReadDisk read for the test's length.
func withReadDisk(t *testing.T, read func(string, uint64) DiskCheck) {
	t.Helper()
	prev := ReadDisk
	ReadDisk = read
	t.Cleanup(func() { ReadDisk = prev })
}

// fakeDisk is a probe for a home whose Go build cache and worktree store sit
// on volumes as vols lays them out: by path prefix, the volume and its free
// GiB, the longest prefix winning. The cache holds cacheGiB, estimated from 12
// of 256 subdirectories, and is a directory with the Go build cache's README.
func fakeDisk(home string, cacheGiB uint64, vols map[string]machineload.Volume) (diskProbe, *[]string) {
	var measured []string
	cache := filepath.Join(home, "Library", "Caches", "go-build")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "README"), []byte(goCacheREADMEText), 0o644); err != nil {
		panic(err)
	}
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
		floor: DiskFloorFor(1),
	}, &measured
}

// goCacheREADMEText is the README the go command writes at the top of a build
// cache.
const goCacheREADMEText = "This directory holds cached build artifacts from the Go build system.\n" +
	"Run \"go clean -cache\" if the directory is getting too large.\n"

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
		"the floor is 8.0 GiB",
		"The floor is 4.0 GiB for each preflight the gate admits to its test stages at once (load.preflights",
		"about 177.0 GiB (estimated from 12 of its 256 subdirectories)",
		"find -H ~/Library/Caches/go-build -type f ! -path ~/'Library/Caches/go-build/fuzz/*' -mmin +2880 -delete",
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
	p, measured := fakeDisk(home, 177, dataVolume(home, DiskFloorFor(1)))
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

// TestWaitRefusesWhenTheDiskFillsWhileItWaits: the disk is read again just
// before the gate opens, after the jitter, and a volume that fell below the
// floor while the waiter waited is refused there, with no hold taken.
func TestWaitRefusesWhenTheDiskFillsWhileItWaits(t *testing.T) {
	home := t.TempDir()
	roomy, _ := fakeDisk(home, 177, dataVolume(home, 200*gib))
	low, _ := fakeDisk(home, 177, dataVolume(home, 3*gib))
	reads := 0
	withReadDisk(t, func(h string, floor uint64) DiskCheck {
		reads++
		p := roomy
		if reads > 1 {
			p = low
		}
		p.floor = floor
		return checkDisk(h, p)
	})
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 51000, clock, poll{load: 1}))
	if err == nil {
		t.Fatalf("a disk that filled while the waiter waited was admitted: %+v", res)
	}
	if reads != 2 {
		t.Fatalf("the disk was read %d times, want 2 (at the start and before opening)", reads)
	}
	if !strings.Contains(err.Error(), "fell below the floor while this waited") || !strings.Contains(err.Error(), "3.0 GiB free") {
		t.Fatalf("the refusal does not say the disk filled during the wait:\n%v", err)
	}
	if res.Hold != nil || res.Status == WaitOpen {
		t.Fatalf("the refusal opened or held: %+v", res)
	}
	if _, err := os.Stat(abcdhome.Path(home, WaitHoldFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a hold was left behind: %v", err)
	}
}

// TestDiskPartlyReadIsReported: a volume that cannot be read beside one with
// room refuses nothing, but the load check reports it and the waiter carries
// a note naming it.
func TestDiskPartlyReadIsReported(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 1, dataVolume(home, 200*gib))
	inner := p.volume
	p.volume = func(path string) (machineload.Volume, error) {
		if strings.Contains(path, "go-build") {
			return machineload.Volume{}, errors.New("statfs " + path + ": stale NFS file handle")
		}
		return inner(path)
	}
	d := checkDisk(home, p)
	if d.Status != DiskOK || d.Refusal() != nil {
		t.Fatalf("check = %+v, refusal %v", d, d.Refusal())
	}
	r := d.Report()
	if r == nil {
		t.Fatalf("a volume that could not be read went unreported: %+v", d)
	}
	text := r.WarningText(SitePreflight)
	for _, want := range []string{"disk check (preflight): the disk was checked only in part", "~/Library/Caches/go-build could not be read", "stale NFS file handle"} {
		if !strings.Contains(text, want) {
			t.Errorf("the report does not say %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, home) {
		t.Errorf("the report carries the home's path:\n%s", text)
	}
}

// TestWaitNotesAnUncheckedDisk: a disk the waiter cannot read is not
// refused, and the result says it was not checked, and why.
func TestWaitNotesAnUncheckedDisk(t *testing.T) {
	home := t.TempDir()
	p, _ := fakeDisk(home, 1, nil)
	p.volume = func(string) (machineload.Volume, error) { return machineload.Volume{}, machineload.ErrUnsupported }
	withDisk(t, p)
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(waitReq(home, 51000, clock, poll{load: 1}))
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("an unread disk was refused: res %+v, err %v", res, err)
	}
	if !strings.Contains(res.DiskNote, "the disk was not checked: abcd reads free space on macOS and Linux only") {
		t.Fatalf("DiskNote = %q, want it to say the disk was not checked and why", res.DiskNote)
	}
}

// TestDiskVolumesWithoutAnIDAreApart: two volumes reporting no identity
// (machineload's "" for a zero filesystem ID) are judged one by one, so a
// low one beside a roomy one is still found.
func TestDiskVolumesWithoutAnIDAreApart(t *testing.T) {
	home := t.TempDir()
	vols := map[string]machineload.Volume{
		filepath.Join(home, "Library"): {Free: 400 * gib},
		home:                           {Free: 3 * gib},
	}
	p, _ := fakeDisk(home, 1, vols)
	d := checkDisk(home, p)
	if d.Status != DiskLow || len(d.Volumes) != 2 {
		t.Fatalf("check = %+v, want two volumes, one low", d)
	}
}

// TestDiskFloorFollowsTheSlots: the floor is 4 GiB for each preflight the gate
// admits at once, and 8 GiB at least; the waiter judges at its own slots.
func TestDiskFloorFollowsTheSlots(t *testing.T) {
	for slots, want := range map[int]uint64{1: 8 * gib, 2: 8 * gib, 3: 12 * gib, 4: 16 * gib} {
		if got := DiskFloorFor(slots); got != want {
			t.Errorf("DiskFloorFor(%d) = %s, want %s", slots, bytesText(got), bytesText(want))
		}
	}
	for env, want := range map[string]uint64{"": 8 * gib, "4": 16 * gib} {
		home := t.TempDir()
		var floors []uint64
		withReadDisk(t, func(_ string, floor uint64) DiskCheck {
			floors = append(floors, floor)
			return DiskCheck{Status: DiskOK, FloorBytes: floor, Volumes: []DiskVolume{}}
		})
		clock := &fakeClock{now: waitT0}
		req := waitReq(home, 51000, clock, poll{load: 1})
		req.Getenv = func(k string) string {
			if k == PreflightSlotsEnv {
				return env
			}
			return ""
		}
		if _, err := WaitLoad(req); err != nil {
			t.Fatal(err)
		}
		if len(floors) == 0 || slices.ContainsFunc(floors, func(f uint64) bool { return f != want }) {
			t.Errorf("%s=%q: the gate judged against %v, want %s", PreflightSlotsEnv, env, floors, bytesText(want))
		}
	}

	// At the default of one, 10 GiB free is room enough.
	home := t.TempDir()
	p, _ := fakeDisk(home, 1, dataVolume(home, 10*gib))
	p.floor = DiskFloorFor(1)
	if d := checkDisk(home, p); d.Status != DiskOK {
		t.Fatalf("10 GiB at one slot = %+v", d)
	}
}

// TestRealReadDiskCarriesTheFloor: the binary's own ReadDisk judges against
// the floor it is given, and reads the worktree store's volume and the Go
// build cache's, on this machine's real disk.
func TestRealReadDiskCarriesTheFloor(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("free space is read on macOS and Linux only")
	}
	home := t.TempDir()
	floor := DiskFloorFor(4)
	d := productionReadDisk(home, floor)
	if d.FloorBytes != floor {
		t.Fatalf("FloorBytes = %d, want %d", d.FloorBytes, floor)
	}
	if d.Status != DiskOK && d.Status != DiskLow {
		t.Fatalf("check = %+v, want a reading", d)
	}
	var holds []string
	for _, v := range d.Volumes {
		holds = append(holds, v.Holds...)
		if v.Low != (v.FreeBytes < floor) {
			t.Errorf("volume %+v judged against a floor other than %d", v, floor)
		}
	}
	if !slices.Contains(holds, DiskHoldsWorktreeStore) {
		t.Errorf("the worktree store's volume was not read: %+v", d)
	}
	if machineload.GoCacheDir() != "" && !slices.Contains(holds, DiskHoldsGoCache) {
		t.Errorf("the Go build cache's volume was not read: %+v", d)
	}
}

// TestDiskTrimKeepsFuzzAndFollowsALink runs the printed trim as a person
// pastes it, against a cache reached through a symlink: it deletes the build
// entries unused for two days and keeps the fuzz corpora and newer entries.
func TestDiskTrimKeepsFuzzAndFollowsALink(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the trim is a POSIX find line")
	}
	home := t.TempDir()
	target := filepath.Join(t.TempDir(), "go-build")
	files := map[string]bool{ // path under the cache: deleted by the trim
		"ab/old-a":            true,
		"ab/new-a":            false,
		"fuzz/example/corpus": false,
	}
	old := time.Now().Add(-72 * time.Hour)
	for rel, gone := range files {
		path := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if gone || strings.HasPrefix(rel, "fuzz/") {
			if err := os.Chtimes(path, old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(target, "README"), []byte(goCacheREADMEText), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "Library", "Caches", "go-build")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	p, _ := fakeDisk(home, 1, dataVolume(home, gib))
	msg := checkDisk(home, p).Refusal().Error()
	var line string
	for _, l := range strings.Split(msg, "\n") {
		if strings.Contains(l, "find ") {
			line = strings.TrimSpace(l)
		}
	}
	if line == "" {
		t.Fatalf("no trim line:\n%s", msg)
	}
	cmd := exec.Command("/bin/sh", "-c", line)
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", line, err, out)
	}
	for rel, gone := range files {
		_, err := os.Stat(filepath.Join(target, rel))
		if gone != errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: deleted %v, want %v (by %s)", rel, !gone, gone, line)
		}
	}
}

// TestDiskTrimOnlyForACache: a GOCACHE that does not have a Go build cache's
// shape, or that is the home itself, is offered only go clean -cache, never a
// find line deleting files in it.
func TestDiskTrimOnlyForACache(t *testing.T) {
	home := t.TempDir()
	notACache := filepath.Join(home, "project")
	if err := os.MkdirAll(filepath.Join(notACache, "00"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The home holds a README in the cache's words, which must not make it one.
	if err := os.WriteFile(filepath.Join(home, "README"), []byte(goCacheREADMEText), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{"not a cache": notACache, "the home": home} {
		p, _ := fakeDisk(home, 1, dataVolume(home, gib))
		p.goCache = func() string { return dir }
		msg := checkDisk(home, p).Refusal().Error()
		if strings.Contains(msg, "find ") || strings.Contains(msg, "-delete") {
			t.Errorf("%s: a find line was offered:\n%s", name, msg)
		}
		if !strings.Contains(msg, "go clean -cache") {
			t.Errorf("%s: go clean -cache was not offered:\n%s", name, msg)
		}
	}
}
