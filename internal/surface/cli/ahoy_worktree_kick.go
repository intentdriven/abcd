package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// A lane's start is where the Go build cache's limit is kept (iss-2611041338455722):
// a lane is what fills the cache, every build and drain makes them, and a drive
// that runs for weeks makes them all the way through. Prune is too rare to do it
// and the load gate only refuses. The walk took from 115 to 437 seconds on a very
// large cache, so `ahoy worktree trim-cache` runs detached and the lane does not
// wait; the marker holds it to once an hour (worktree.ClaimCacheTrim).
func init() {
	loop.LaneOpened = func() {
		home, err := abcdhome.MachineHome()
		if err != nil {
			fmt.Fprintf(os.Stderr, "abcd: the Go build cache was not trimmed at this lane's start: %s\n", shownCacheTrimText(err.Error()))
			return
		}
		kickCacheTrim(os.Stderr, home, time.Now(), spawnCacheTrim)
	}
}

// kickCacheTrim starts a trim when one is due, first saying on errw what the last
// one could not do, which had no terminal of its own to say it on. Nothing it
// meets stops the lane: every failure is one line on errw.
func kickCacheTrim(errw io.Writer, home string, now time.Time, spawn func(dir string) error) {
	last, due, err := worktree.ClaimCacheTrim(home, now)
	if err != nil {
		fmt.Fprintf(errw, "abcd: the Go build cache was not trimmed at this lane's start: %s\n", shownCacheTrimText(err.Error()))
		return
	}
	if !due {
		return
	}
	if said := lastCacheTrimProblem(last); said != "" {
		fmt.Fprintf(errw, "abcd: the last trim of the Go build cache: %s\n", shownCacheTrimText(said))
	}
	if err := spawn(home); err != nil {
		said := "the trim could not start: " + shownCacheTrimText(err.Error())
		fmt.Fprintf(errw, "abcd: the Go build cache's %s\n", said)
		// The claim stands, so the next lane an hour on says this rather than
		// that a run which never started did not finish.
		if serr := worktree.SaveCacheTrim(home, now, worktree.CacheTrim{Skipped: said}); serr != nil {
			fmt.Fprintf(errw, "abcd: the Go build cache's trim could not be recorded: %s\n", shownCacheTrimText(serr.Error()))
		}
	}
}

// lastCacheTrimProblem is what a run left for the next lane to say, "" for a clean
// run or none at all.
func lastCacheTrimProblem(r worktree.CacheTrimRun) string {
	switch {
	case r.StartedAt.IsZero():
		return ""
	case r.Trim == nil:
		return "the one started at " + r.StartedAt.UTC().Format(time.RFC3339) + " did not finish"
	case r.Trim.Skipped != "" || r.Trim.Failed > 0 || r.Trim.OverLimit:
		return cacheTrimLine(*r.Trim, false)
	}
	return ""
}

// spawnCacheTrim starts `ahoy worktree trim-cache` detached from the lane, in dir.
// Its own session, so it outlives the step that started it and the host's ending
// of that step does not reach it; nothing of the caller's held open, so a caller
// reading the step's output through a pipe is not kept waiting for its end, and it
// says what it found through the marker instead. dir is the home, so it holds no
// checkout a prune may remove.
var spawnCacheTrim = func(dir string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd := exec.Command(self, "ahoy", "worktree", "trim-cache")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
