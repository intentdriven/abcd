// Command test-parallelism prints the -p `make preflight` gives its test and
// race stages, so one run cannot saturate a machine other sessions share
// (iss-2610102020442624): twice the online cores less the one-minute load,
// never more than the cores divided among the preflights that may be in their
// test stages at once (load.preflights in ~/.abcd.noindex/config.json, or
// ABCD_LOAD_PREFLIGHTS; one by default) and never fewer than a quarter of them
// (implement.PreflightParallelism, machineload.GoTestParallelism). Each stage
// runs it as it starts, so the number reflects the load at that moment. On a
// CI runner, or where the machine cannot be read, it prints go's own default,
// GOMAXPROCS.
//
// Usage: go test -p "$(go run ./cmd/test-parallelism)" ./...
//
// It is developer tooling, never part of a release: it prints one number and a
// newline, reads the machine once, writes nothing and exits 0. A slot setting
// it cannot use is named on stderr and read as one; the load gate before the
// stage refuses that setting first.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

func main() {
	home, _ := os.UserHomeDir()
	run(os.Stdout, os.Stderr, os.Getenv, machineload.Read, home)
}

// run prints the parallelism derived from getenv, read and the slot setting
// under home.
func run(w, errw io.Writer, getenv func(string) string, read func() (machineload.Snapshot, error), home string) {
	slots, _, err := implement.PreflightSlots(getenv, home)
	if err != nil {
		fmt.Fprintf(errw, "test-parallelism: %v; taking one preflight at a time\n", err)
		slots = 1
	}
	fmt.Fprintln(w, implement.PreflightParallelism(getenv, read, slots))
}
