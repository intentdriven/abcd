// Command check-tests runs one of make preflight's test stages from the
// repository's fixed check directory, reusing every result whose inputs cannot
// have changed and reporting how many it reused (internal/testreuse,
// itd-2611011325421304).
//
// Usage: check-tests [-fresh] [-cached-timeout <d>] [-budget <d>] [-lock-wait <d>] <stage> go test <flags> <packages>
//
// The recipe's go test command is written out after the stage, whole, so the
// recipe still says what the stage runs. It exits with go test's result, 2 for a
// recipe mistake and 3 when another check held the directory past -lock-wait or it
// was cancelled waiting, having run nothing. A stage that waited takes its -p and
// -timeout again from -budget, as cmd/test-parallelism prints them, since the
// recipe derived them before the wait, and a race stage goes back through the race
// gate, since another may have started meanwhile. It is developer tooling, never
// part of a release.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/testreuse"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	code := run(ctx, os.Args[1:], wd, home, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, wd, home string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check-tests", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fresh := fs.Bool("fresh", false, "reuse nothing, as a release cut does")
	cachedTimeout := fs.String("cached-timeout", "", "the -timeout of a line whose results are to be reused")
	lockWait := fs.Duration("lock-wait", 4*time.Hour, "how long to wait for another check of this directory")
	budget := fs.Duration("budget", 0, "the stage's base package budget, to scale again after a wait")
	if err := fs.Parse(args); err != nil {
		return testreuse.ExitUsage
	}
	rest := fs.Args()
	if len(rest) < 3 || !slices.Equal(rest[1:3], []string{"go", "test"}) {
		fmt.Fprintln(stderr, "usage: check-tests [-fresh] [-cached-timeout <d>] [-budget <d>] [-lock-wait <d>] <stage> go test <flags> <packages>")
		return testreuse.ExitUsage
	}
	var afterWait func(bool) (string, string, error)
	if *budget > 0 {
		afterWait = func(race bool) (string, string, error) {
			if race {
				gate := exec.CommandContext(ctx, "go", "run", "./cmd/abcd", "implement", "load", "--race-gate", "--timeout", "2h")
				gate.Dir, gate.Stdout, gate.Stderr = wd, stdout, stderr
				if err := gate.Run(); err != nil {
					return "", "", fmt.Errorf("the race gate: %w", err)
				}
			}
			slots, _, _, err := implement.PreflightSlots(os.Getenv, home)
			if err != nil {
				slots = 1
			}
			p := implement.PreflightParallelism(os.Getenv, machineload.Read, slots)
			return strconv.Itoa(p), implement.PreflightTimeout(os.Getenv, machineload.Read, *budget).String(), nil
		}
	}
	return testreuse.Run(ctx, testreuse.Config{
		Stage:         rest[0],
		Args:          rest[3:],
		Worktree:      wd,
		Home:          home,
		Stdout:        stdout,
		Stderr:        stderr,
		Fresh:         *fresh,
		CachedTimeout: *cachedTimeout,
		LockWait:      *lockWait,
		AfterWait:     afterWait,
		Uid:           os.Geteuid(),
	})
}
