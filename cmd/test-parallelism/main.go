// Command test-parallelism prints the -p `make preflight` gives its test and
// race stages, so one run cannot saturate a machine other sessions share
// (iss-2610102020442624): twice the online cores less the one-minute load,
// never more than the cores divided among the preflights the first gate
// admits at once (load.preflights in ~/.abcd.noindex/config.json, or
// ABCD_LOAD_PREFLIGHTS; one by default) and
// never fewer than a quarter of them (implement.PreflightParallelism,
// machineload.GoTestParallelism). Each stage runs it as it starts, so the
// number reflects the load at that moment. On a CI runner, or where the
// machine cannot be read, it prints go's own default, GOMAXPROCS.
//
// Given -timeout and a stage's base package budget, it prints that budget
// instead, or twice it when the one-minute load passes one runnable per online
// core, so a package slowed by
// the load is not killed with no test hung (iss-2610241911052191,
// implement.PreflightTimeout, machineload.GoTestTimeout). On a CI runner, or
// where the machine cannot be read, it prints the base.
//
// Usage: go test -p "$(go run ./cmd/test-parallelism)" ./...
//
//	go test -timeout "$(go run ./cmd/test-parallelism -timeout 40m)" ./...
//
// It is developer tooling, never part of a release: it prints one value and a
// newline, reads the machine once, writes nothing and exits 0. A slot setting
// it cannot use is named on stderr and read as one; the load gate before the
// stage refuses that setting first. A machine config.json it cannot read is
// read as one without a word: the load gate before the stage names it. An
// argument it does not know, or a base that is not a positive duration, is a
// recipe mistake: it prints nothing on standard output and exits 2, so the
// stage fails there.
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

func main() {
	home, _ := os.UserHomeDir()
	os.Exit(runArgs(os.Args[1:], os.Stdout, os.Stderr, os.Getenv, machineload.Read, home))
}

// runArgs prints the -p with no argument and the scaled package budget with
// `-timeout <base>`, and returns the exit code.
func runArgs(args []string, w, errw io.Writer, getenv func(string) string, read func() (machineload.Snapshot, error), home string) int {
	switch {
	case len(args) == 0:
		run(w, errw, getenv, read, home)
		return 0
	case len(args) == 2 && args[0] == "-timeout":
		base, err := time.ParseDuration(args[1])
		if err != nil || base <= 0 {
			fmt.Fprintf(errw, "test-parallelism: -timeout %q is not a positive duration\n", args[1])
			return 2
		}
		fmt.Fprintln(w, implement.PreflightTimeout(getenv, read, base))
		return 0
	default:
		fmt.Fprintln(errw, "usage: test-parallelism [-timeout <base>]")
		return 2
	}
}

// run prints the parallelism derived from getenv, read and the slot setting
// under home.
func run(w, errw io.Writer, getenv func(string) string, read func() (machineload.Snapshot, error), home string) {
	slots, _, _, err := implement.PreflightSlots(getenv, home)
	if err != nil {
		fmt.Fprintf(errw, "test-parallelism: %v; taking one preflight at a time\n", err)
		slots = 1
	}
	fmt.Fprintln(w, implement.PreflightParallelism(getenv, read, slots))
}
