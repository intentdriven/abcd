package implement

import (
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// The trim at the floor (iss-2611220942266387). The waiter refused at the disk
// floor and left the cache to the trims a lane's start and a prune run, so a
// session that drove its own worktrees and preflights reached neither, and over
// a night of preflights a person cleared the disk by hand five times. So before
// it refuses, a waiter whose low volume holds the Go build cache runs the paced
// trim of each toolchain its checkout uses (worktree.TrimForFloor) once, says
// on stderr that it is trimming and then what the trim freed, or that it did
// not run and why, and reads the disk again. It refuses only when the volume is
// still under the floor, and the refusal names what the trim did. The trim's
// own guards hold (worktree/cachetrim.go): it deletes only Go's own entries in
// the directory Go names, down to the person's size limit and never by free
// space, so a cache already under its limit loses nothing here, and the
// person's lever is load.go_cache_limit_gib.

// DiskTrimMinLeft is the least of a waiter's --timeout that must be left for
// it to start the trim at the floor. The trim walks a large cache for minutes
// and cannot be stopped part way (worktree.TrimForFloor takes no deadline), so
// a waiter with less left starts none: it says so on stderr and in the
// report's disk_trim, and refuses at the floor at once, as it did before the
// trim existed. Five minutes keeps a lane's two-minute wait from ever running
// over its timeout, or its harness's ten-minute command cap, for a trim. A
// longer wait still has its trim bounded only by this much of its time.
const DiskTrimMinLeft = 5 * time.Minute

// TrimAtFloor runs, as of now, the paced cache trim of each toolchain the
// checkout whose root is dir uses, for the account whose home is home
// (worktree.TrimForFloor). It is a variable only so tests stand in for it: the
// package's own tests replace it by default, so none trims a real cache.
var TrimAtFloor = worktree.TrimForFloor

// floorTrim is one waiter's trim at the floor: run at most once a wait.
type floorTrim struct {
	home, dir string
	now       func() time.Time
	// deadline is when the wait's --timeout runs out: no trim starts with
	// less than DiskTrimMinLeft before it.
	deadline time.Time
	say       func(string)
	tried     bool
	// said is what the trim did, one sentence a step, for the report
	// (WaitResult.DiskTrim) and the refusal; "" until it ran.
	said string
}

// at judges disk, read against floor. When a low volume holds the Go build
// cache and this wait has not trimmed yet, it runs the trim, says what it did,
// and reads the disk again if the trim freed anything; it hands back the disk
// to judge, carrying for the refusal what the trim did or why none ran.
func (f *floorTrim) at(disk DiskCheck, floor uint64) DiskCheck {
	if disk.Status != DiskLow {
		return disk
	}
	vol, low := disk.cacheVolumeLow()
	switch {
	case !low:
		disk.trimNote = "No cache was trimmed: no low volume holds the Go build cache, the one cache a trim clears."
		return disk
	case f.tried:
		disk.trimNote = "No cache was trimmed again: this wait already trimmed once (" + f.said + "), and trims no more than once."
		return disk
	}
	f.tried = true
	if left := f.deadline.Sub(f.now()); left < DiskTrimMinLeft {
		f.said = fmt.Sprintf("no cache was trimmed: the wait had %s of its --timeout left, under the %s a trim needs to start, since a trim walks a large cache for minutes and cannot be stopped part way",
			max(left, 0).Round(time.Second), DiskTrimMinLeft)
		f.say(f.said)
		disk.trimNote = "N" + f.said[1:] + "."
		return disk
	}
	f.say(fmt.Sprintf("%s free on the volume holding the Go build cache, under the floor of %s: trimming the caches this checkout's toolchains fill before refusing",
		bytesText(vol.FreeBytes), bytesText(floor)))
	got := TrimAtFloor(f.home, f.dir, f.now(), func(cache string) {
		f.say("trimming " + cache + " now, paced to once an hour across the account; a large cache takes minutes")
	})
	var lines []string
	if got.None != "" {
		lines = append(lines, "no cache was trimmed: "+got.None)
	}
	for _, s := range got.Steps {
		lines = append(lines, trimStepLine(s))
	}
	for _, l := range lines {
		f.say(l)
	}
	after := disk
	var outcome string
	switch {
	case got.BytesFreed() == 0:
		outcome = "the trim freed nothing, so the volume is still under the floor"
	default:
		after = ReadDisk(f.home, floor)
		free := "the disk could not be read again"
		if v, ok := after.cacheVolume(); ok {
			free = bytesText(v.FreeBytes) + " free after the trim"
		}
		switch after.Status {
		case DiskLow:
			outcome = free + ", still under the floor of " + bytesText(floor) + " (a local snapshot can hold what a trim deleted, so free space can lag behind it)"
		default:
			outcome = free + ", over the floor of " + bytesText(floor)
		}
	}
	f.say(outcome)
	lines = append(lines, outcome)
	f.said = strings.Join(lines, "; ")
	after.trimNote = "Before refusing, the gate trimmed: " + f.said + "."
	return after
}

// trimStepLine is one sentence saying what one toolchain's trim did, or why it
// did not run.
func trimStepLine(s worktree.ToolchainTrim) string {
	switch {
	case s.Trim != nil && s.Trim.Skipped != "":
		return s.Cache + " was not trimmed: " + s.Trim.Skipped
	case s.Trim != nil:
		line := s.Cache + " " + cacheTrimText(*s.Trim)
		if s.Problem != "" {
			line += "; " + s.Problem
		}
		return line
	case s.Problem != "":
		return s.Cache + " was not trimmed: " + s.Problem
	}
	line := s.Cache + " was not trimmed: a trim of it started at " + s.PacedSince.UTC().Format(time.RFC3339) +
		", and it runs at most once an hour across the account (" + abcdhome.Display("go-cache-trim.json") + ")"
	if s.Last == nil {
		return line + "; that trim has not finished"
	}
	if s.Last.Skipped != "" {
		return line + "; that trim was skipped: " + s.Last.Skipped
	}
	return line + "; that trim, of " + cacheTrimText(*s.Last)
}

// cacheTrimText is what a trim that walked its cache did: the cache, what it
// removed and freed, and what it left of the limit.
func cacheTrimText(t worktree.CacheTrim) string {
	entries := fmt.Sprintf("%d entries", t.EntriesRemoved)
	if t.EntriesRemoved == 1 {
		entries = "1 entry"
	}
	line := fmt.Sprintf("%s: removed %s, %s freed, %s left of a limit of %s",
		t.Path, entries, bytesText(t.BytesFreed), bytesText(t.LeftBytes), bytesText(t.LimitBytes))
	if t.OverLimit {
		line += fmt.Sprintf(", still over the limit, since entries used in the last %d hours stay", t.KeepHours)
	}
	if t.Failed > 0 {
		line += fmt.Sprintf(", %d could not be read or removed", t.Failed)
	}
	return line
}

// cacheVolume is the volume holding the Go build cache, when it was read.
func (d DiskCheck) cacheVolume() (DiskVolume, bool) {
	for _, v := range d.Volumes {
		for _, h := range v.Holds {
			if h == DiskHoldsGoCache {
				return v, true
			}
		}
	}
	return DiskVolume{}, false
}

// cacheVolumeLow is the volume holding the Go build cache, and whether it is
// under the floor.
func (d DiskCheck) cacheVolumeLow() (DiskVolume, bool) {
	v, ok := d.cacheVolume()
	return v, ok && v.Low
}
