// Command test-parallelism prints the -p `make preflight` gives its test and
// race stages, so one run cannot saturate a machine other sessions share
// (iss-2610102020442624): twice the online cores less the one-minute load,
// never more than the cores and never fewer than a quarter of them
// (implement.PreflightParallelism, machineload.GoTestParallelism). Each stage
// runs it as it starts, so the number reflects the load at that moment. On a CI
// runner, or where the machine cannot be read, it prints go's own default,
// GOMAXPROCS.
//
// Usage: go test -p "$(go run ./cmd/test-parallelism)" ./...
//
// It is developer tooling, never part of a release: it prints one number and a
// newline, reads the machine once, writes nothing and exits 0.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

func main() {
	run(os.Stdout, os.Getenv, machineload.Read)
}

// run prints the parallelism derived from getenv and read.
func run(w io.Writer, getenv func(string) string, read func() (machineload.Snapshot, error)) {
	fmt.Fprintln(w, implement.PreflightParallelism(getenv, read))
}
