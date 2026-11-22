package implement

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// withTrimAtFloor makes the gate's trim at the floor trim for the test's
// length, so no test here walks a real cache.
func withTrimAtFloor(t *testing.T, trim func(home, dir string, now time.Time, before func(string)) worktree.FloorTrim) {
	t.Helper()
	prev := TrimAtFloor
	TrimAtFloor = trim
	t.Cleanup(func() { TrimAtFloor = prev })
}

// trimmingDisk is a home whose one volume holds the Go build cache and the
// worktree store with *free bytes free, read afresh at every look.
func trimmingDisk(t *testing.T, home string, free *uint64) {
	t.Helper()
	withReadDisk(t, func(h string, floor uint64) DiskCheck {
		p, _ := fakeDisk(h, 177, dataVolume(h, *free))
		p.floor = floor
		return checkDisk(h, p)
	})
}

// goStepTrim is a floor trim whose one step, Go's, ran and freed freed bytes.
func goStepTrim(freed uint64) worktree.FloorTrim {
	return worktree.FloorTrim{Steps: []worktree.ToolchainTrim{{
		Toolchain: "go", Cache: "the Go build cache",
		Trim: &worktree.CacheTrim{Path: "~/Library/Caches/go-build", EntriesRemoved: 1200, BytesFreed: freed,
			SizeBytes: 160 * gib, LeftBytes: 160*gib - freed, LimitBytes: 100 * gib, KeepHours: 3, OverLimit: 160*gib-freed > 100*gib},
	}}}
}

// TestWaitTrimsTheCacheBeforeRefusingAtTheFloor is the detector for
// iss-2611220942266387: a waiter that finds the volume holding the Go build
// cache under the floor runs the paced trim once, for the checkout it runs in,
// says on its way that it is trimming and then what the trim freed, reads the
// disk again, and is admitted when the trim brought the volume over the floor.
func TestWaitTrimsTheCacheBeforeRefusingAtTheFloor(t *testing.T) {
	home := t.TempDir()
	free := 3 * gib
	trimmingDisk(t, home, &free)
	var dirs []string
	withTrimAtFloor(t, func(h, dir string, _ time.Time, before func(string)) worktree.FloorTrim {
		dirs = append(dirs, dir)
		before("the Go build cache")
		free = 63 * gib
		return goStepTrim(60 * gib)
	})
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 51000, clock, poll{load: 1})
	req.Dir = "/checkouts/repo"
	var said []string
	req.OnDiskTrim = func(line string) { said = append(said, line) }

	res, err := WaitLoad(req)

	if err != nil || res.Status != WaitOpen {
		t.Fatalf("the trim cleared the floor, so the gate opens: %+v, %v", res, err)
	}
	if len(dirs) != 1 || dirs[0] != "/checkouts/repo" {
		t.Fatalf("the trim runs once, for the checkout the waiter runs in: %q", dirs)
	}
	all := strings.Join(said, "\n")
	for _, want := range []string{
		"3.0 GiB free", "under the floor of 8.0 GiB", "trimming the Go build cache",
		"removed 1200 entries, 60.0 GiB freed", "63.0 GiB free after the trim",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("stderr does not say %q:\n%s", want, all)
		}
	}
	if len(said) < 2 || !strings.Contains(said[0], "trimming") {
		t.Fatalf("it says it is trimming before the trim's answer: %q", said)
	}
	if !strings.Contains(res.DiskTrim, "60.0 GiB freed") {
		t.Fatalf("the report carries what the trim did: %q", res.DiskTrim)
	}
}

// TestWaitRefusesWhenTheTrimLeavesTheDiskLow: a trim that frees too little is
// said, and the refusal that follows names what it did beside the low volume.
func TestWaitRefusesWhenTheTrimLeavesTheDiskLow(t *testing.T) {
	home := t.TempDir()
	free := 3 * gib
	reads := 0
	withReadDisk(t, func(h string, floor uint64) DiskCheck {
		reads++
		p, _ := fakeDisk(h, 177, dataVolume(h, free))
		p.floor = floor
		return checkDisk(h, p)
	})
	withTrimAtFloor(t, func(string, string, time.Time, func(string)) worktree.FloorTrim {
		free = 5 * gib
		return goStepTrim(2 * gib)
	})
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 51000, clock, poll{load: 1})
	var said []string
	req.OnDiskTrim = func(line string) { said = append(said, line) }

	res, err := WaitLoad(req)

	if err == nil || res.Status == WaitOpen || res.Hold != nil {
		t.Fatalf("still under the floor, so refused with no hold: %+v, %v", res, err)
	}
	if reads != 2 {
		t.Fatalf("the disk was read %d times, want 2 (before and after the trim)", reads)
	}
	msg := err.Error()
	for _, want := range []string{"5.0 GiB free", "nothing was waited for", "2.0 GiB freed", "still over the limit"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, msg)
		}
	}
	if all := strings.Join(said, "\n"); !strings.Contains(all, "2.0 GiB freed") || !strings.Contains(all, "still under the floor") {
		t.Fatalf("stderr says what the trim freed and that the floor still holds: %q", said)
	}
}

// TestWaitSaysATrimThatDidNotRun: a trim paced out, one whose pacing record
// could not be claimed, one that skipped the cache, and a repository with no
// known toolchain each refuse at the floor, saying why nothing was freed, on
// stderr and in the refusal.
func TestWaitSaysATrimThatDidNotRun(t *testing.T) {
	go1 := func(s worktree.ToolchainTrim) worktree.FloorTrim {
		s.Toolchain, s.Cache = "go", "the Go build cache"
		return worktree.FloorTrim{Steps: []worktree.ToolchainTrim{s}}
	}
	for name, c := range map[string]struct {
		trim worktree.FloorTrim
		want []string
	}{
		"paced, still running": {go1(worktree.ToolchainTrim{PacedSince: waitT0.Add(-10 * time.Minute)}),
			[]string{"the Go build cache was not trimmed", "17:50:00Z", "has not finished", "at most once an hour"}},
		"paced, finished": {go1(worktree.ToolchainTrim{PacedSince: waitT0.Add(-10 * time.Minute), Last: &worktree.CacheTrim{Path: "~/c", EntriesRemoved: 9, BytesFreed: 4 * gib, LeftBytes: 90 * gib, LimitBytes: 100 * gib}}),
			[]string{"the Go build cache was not trimmed", "at most once an hour", "removed 9 entries, 4.0 GiB freed"}},
		"unclaimed": {go1(worktree.ToolchainTrim{Problem: "its pacing record could not be claimed: locked"}),
			[]string{"the Go build cache was not trimmed", "its pacing record could not be claimed: locked"}},
		"skipped": {go1(worktree.ToolchainTrim{Trim: &worktree.CacheTrim{Skipped: "no go command to say where the cache is"}}),
			[]string{"the Go build cache was not trimmed", "no go command to say where the cache is"}},
		"no toolchain": {worktree.FloorTrim{None: "the repository uses no toolchain abcd trims a cache for"},
			[]string{"no cache was trimmed", "the repository uses no toolchain abcd trims a cache for"}},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			free := 3 * gib
			trimmingDisk(t, home, &free)
			withTrimAtFloor(t, func(string, string, time.Time, func(string)) worktree.FloorTrim { return c.trim })
			clock := &fakeClock{now: waitT0}
			req := waitReq(home, 51000, clock, poll{load: 1})
			var said []string
			req.OnDiskTrim = func(line string) { said = append(said, line) }

			_, err := WaitLoad(req)

			if err == nil {
				t.Fatal("nothing was freed, so the floor refuses")
			}
			all := strings.Join(said, "\n")
			for _, want := range c.want {
				if !strings.Contains(all, want) {
					t.Errorf("stderr does not say %q:\n%s", want, all)
				}
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not say %q:\n%v", want, err)
				}
			}
		})
	}
}

// TestWaitTrimsOnlyForTheCachesVolume: when only the worktree store's volume is
// low, the cache's trim could free nothing there, so it is not run, and the
// refusal says no trim was run.
func TestWaitTrimsOnlyForTheCachesVolume(t *testing.T) {
	home := t.TempDir()
	vols := map[string]machineload.Volume{
		home:                {Free: 200 * gib, ID: "1:2", Mount: "/System/Volumes/Data"},
		abcdhome.Path(home): {Free: 1 * gib, ID: "1:3", Mount: "/Volumes/Store"},
	}
	p, _ := fakeDisk(home, 1, vols)
	withDisk(t, p)
	ran := false
	withTrimAtFloor(t, func(string, string, time.Time, func(string)) worktree.FloorTrim {
		ran = true
		return worktree.FloorTrim{}
	})
	clock := &fakeClock{now: waitT0}
	_, err := WaitLoad(waitReq(home, 51000, clock, poll{load: 1}))
	if err == nil || ran {
		t.Fatalf("refused without a trim: err %v, ran %v", err, ran)
	}
	if !strings.Contains(err.Error(), "No cache was trimmed: no low volume holds the Go build cache") {
		t.Fatalf("the refusal says why no trim ran:\n%v", err)
	}
}

// TestWaitTrimsOnceAWait: a waiter whose trim at its first look cleared the
// floor, and whose disk then fell below it while it waited, is refused at the
// open look without trimming again.
func TestWaitTrimsOnceAWait(t *testing.T) {
	home := t.TempDir()
	free := 3 * gib
	trimmingDisk(t, home, &free)
	runs := 0
	withTrimAtFloor(t, func(string, string, time.Time, func(string)) worktree.FloorTrim {
		runs++
		free = 63 * gib
		return goStepTrim(60 * gib)
	})
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 51000, clock, poll{load: 1})
	req.Sleep = func(d time.Duration) { clock.Sleep(d); free = 2 * gib }

	_, err := WaitLoad(req)

	if err == nil || runs != 1 {
		t.Fatalf("refused at the open look after one trim: err %v, runs %d", err, runs)
	}
	if !strings.Contains(err.Error(), "fell below the floor while this waited") || !strings.Contains(err.Error(), "already trimmed once") {
		t.Fatalf("the refusal says the disk fell and that this wait trimmed once:\n%v", err)
	}
}
