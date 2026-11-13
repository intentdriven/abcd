package gitutil_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// hookedRunners are the runners of gitutil that start a git with the
// repository's hooks, merge drivers or content filters on, or that reach a
// remote, each running `git -C <dir> <args>`.
var hookedRunners = map[string]func(dir string, args ...string) error{
	"Net": func(dir string, args ...string) error { _, err := gitutil.Net(dir, args...); return err },
	"NetEach": func(dir string, args ...string) error {
		return gitutil.NetEach(dir, 64, func(string) {}, args...)
	},
	"NetInput": func(dir string, args ...string) error { _, err := gitutil.NetInput(dir, "x\n", args...); return err },
	"NetContext": func(dir string, args ...string) error {
		_, err := gitutil.NetContext(context.Background(), dir, args...)
		return err
	},
	"LocalGit": func(dir string, args ...string) error { _, err := gitutil.LocalGit(dir, args...); return err },
	"RunCommitCommandBounded": func(dir string, args ...string) error {
		_, err := gitutil.RunCommitCommandBounded(dir, time.Minute, errors.New("stalled"), args...)
		return err
	},
	"RunCommitCommandBoundedAsStored": func(dir string, args ...string) error {
		_, err := gitutil.RunCommitCommandBoundedAsStored(dir, time.Minute, errors.New("stalled"), args...)
		return err
	},
	"WorkingTreeCommand": func(dir string, args ...string) error {
		return gitutil.WorkingTreeCommand(dir, args...).Run()
	},
	"WorkingTreeCommandContext": func(dir string, args ...string) error {
		return gitutil.WorkingTreeCommandContext(context.Background(), dir, args...).Run()
	},
	"RunLimitedNetContext": func(dir string, args ...string) error {
		_, err := gitutil.RunLimitedNetContext(context.Background(), dir, 4096, args...)
		return err
	},
}

// TestEveryHookedRunnerAsksTheCheckBeforeGitRuns: with a check set that
// refuses, each runner gives the check the command's arguments, returns the
// check's error as a NotRunError naming the subcommand, and
// git makes nothing (its `init` of a new directory leaves none); with no
// check set each runs git as it always has, which is what holds outside a
// drive, where nothing sets one.
func TestEveryHookedRunnerAsksTheCheckBeforeGitRuns(t *testing.T) {
	dir := gittest.NewRepo(t).Root()
	refused := errors.New("held setting changed")
	for name, run := range hookedRunners {
		t.Run(name, func(t *testing.T) {
			asked := 0
			var saw []string
			unset, err := gitutil.SetBeforeHookedGit(func(args []string) error { asked++; saw = args; return refused })
			if err != nil {
				t.Fatal(err)
			}
			made := filepath.Join(dir, "made-"+name)
			err = run(dir, "init", "-q", made)
			unset()
			if !errors.Is(err, refused) || !strings.Contains(err.Error(), "git init was not run") || asked != 1 {
				t.Fatalf("the runner asks the check once and does not run git: asked %d, %v", asked, err)
			}
			var notRun *gitutil.NotRunError
			if !errors.As(err, &notRun) || notRun.Verb != "init" || !strings.HasSuffix(strings.Join(saw, " "), " init -q "+made) && strings.Join(saw, " ") != "init -q "+made {
				t.Fatalf("the check is given the command's arguments and its refusal names the subcommand: %q %v", saw, err)
			}
			if _, serr := os.Stat(made); !errors.Is(serr, os.ErrNotExist) {
				t.Fatalf("git ran under a refusing check: %v", serr)
			}
			if err := run(dir, "init", "-q", made); err != nil {
				t.Fatalf("with no check set the runner runs git: %v", err)
			}
			if _, serr := os.Stat(made); serr != nil {
				t.Fatalf("with no check set git made what it was asked to: %v", serr)
			}
		})
	}
}

// TestOneCheckIsSetAtATime: a second check is refused while one is set, so
// one drive's check is never replaced by another's, and a clear removes only
// the check it was returned with.
func TestOneCheckIsSetAtATime(t *testing.T) {
	first := errors.New("first")
	clearFirst, err := gitutil.SetBeforeHookedGit(func([]string) error { return first })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitutil.SetBeforeHookedGit(func([]string) error { return nil }); !errors.Is(err, gitutil.ErrBeforeHookedGitSet) {
		t.Fatalf("a second check is refused while one is set: %v", err)
	}
	dir := gittest.NewRepo(t).Root()
	if _, err := gitutil.LocalGit(dir, "status"); !errors.Is(err, first) {
		t.Fatalf("the first check stands after the refused set: %v", err)
	}
	clearFirst()
	second := errors.New("second")
	clearSecond, err := gitutil.SetBeforeHookedGit(func([]string) error { return second })
	if err != nil {
		t.Fatalf("once cleared, a check can be set: %v", err)
	}
	defer clearSecond()
	clearFirst()
	if _, err := gitutil.LocalGit(dir, "status"); !errors.Is(err, second) {
		t.Fatalf("the first check's clear does not remove the second: %v", err)
	}
}
