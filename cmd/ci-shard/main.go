// Command ci-shard prints the packages one shard of ci.yml's macOS suite tests,
// one import path per line, heaviest first (iss-2610181757540736):
//
//	pkgs=$(go run ./cmd/ci-shard -shard 2 -of 3)
//	go test -race -timeout 40m $pkgs
//
// The suite keeps its runner's three test slots full, so its time is its total
// work on that runner, and hosted macOS runners vary about twofold: the same
// pass took 32 minutes on one and more than 77 on another. Only running it on
// several runners shortens what one runner has to do.
//
// The module is read, never listed by hand: the assignment is computed from `go
// list ./...` in every shard, so a package added later is tested without anyone
// naming it, and a package cannot fall between two hand-kept lists. Each package
// takes the lightest shard so far, heaviest package first, ties broken by import
// path, which makes it deterministic for one tree: every shard computes it on
// its own runner and they all reach the same answer. TestShardsPartitionTheModule
// holds the union to the module's packages and the shards to disjoint, and
// lint's TestMacOSSuiteShardsCoverTheModule holds what the workflow's own matrix
// asks for to the same.
//
// The weights only balance the shards, so a stale one costs minutes, never
// coverage; a package the table does not name takes defaultWeight.
//
// It is developer tooling, never part of a release. It prints the shard on
// standard output and one line on standard error saying what it chose. Anything
// that would leave a shard with nothing to test (a number out of range, more
// shards than packages, a go list that failed or named nothing) prints nothing
// on standard output and exits 2, so the step fails there rather than running go
// test with no packages, which tests the working directory and reports success.
package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// modulePath is the import path every package of this module starts with, and
// which the weight table leaves out.
const modulePath = "github.com/intentdriven/abcd/"

// weights are the packages that decide how long a shard runs: go test's own
// seconds for each, under -race on the macOS runner, on the slowest passing run
// measured (merge-group run 41733393432, 2026-11-03, 76.6 minutes). They are
// every package at 100s or more, and together about seven eighths of the pass.
var weights = map[string]int{
	"internal/core/implement/loop": 1733,
	"internal/core/ahoy":           1097,
	"internal/surface/cli":         913,
	"internal/core/site":           831,
	"internal/core/claim":          778,
	"internal/core/capture":        730,
	"internal/core/reading":        719,
	"internal/core/launch":         634,
	"internal/core/embargo":        523,
	"internal/adapter/scanner":     518,
	"internal/core/guard":          430,
	"internal/cloudscripts":        336,
	"internal/core/memory":         269,
	"internal/core/decisions":      259,
	"internal/core/lint":           255,
	"internal/core/landing":        244,
	"internal/core/release":        175,
	"internal/core/intent":         173,
	"internal/core/lifeboat":       145,
	"internal/core/interview":      123,
	"internal/core/runner":         102,
}

// defaultWeight is the mean of the packages the table leaves out, on that run.
const defaultWeight = 15

// firstShardHead is the load the first shard starts with, in the same unit: it
// alone runs the build, vet and the plain-only guards (ci.yml's macos-suite job),
// about three minutes of wall clock on a slow runner, which at that runner's 2.7
// busy slots is worth this many seconds of package time.
const firstShardHead = 500

func main() {
	os.Exit(run(os.Args[1:], func() ([]string, error) { return goList(".") }, os.Stdout, os.Stderr))
}

// run prints the shard args name, from the packages list reads, and returns the
// exit code.
func run(args []string, list func() ([]string, error), w, errw io.Writer) int {
	refuse := func(format string, a ...any) int {
		fmt.Fprintf(errw, "ci-shard: "+format+"\n", a...)
		return 2
	}
	fs := flag.NewFlagSet("ci-shard", flag.ContinueOnError)
	fs.SetOutput(errw)
	shard := fs.Int("shard", 0, "the shard to print, from 1")
	of := fs.Int("of", 0, "how many shards the suite runs")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		return refuse("unexpected argument %q; usage: ci-shard -shard <n> -of <count>", fs.Arg(0))
	}
	if *of < 1 {
		return refuse("-of is %d; usage: ci-shard -shard <n> -of <count>", *of)
	}
	if *shard < 1 || *shard > *of {
		return refuse("-shard is %d, not from 1 to %d", *shard, *of)
	}
	pkgs, err := list()
	if err != nil {
		return refuse("reading the module's packages: %v", err)
	}
	if len(pkgs) == 0 {
		return refuse("go list named no packages")
	}
	shards := assign(pkgs, *of)
	for _, s := range shards {
		if len(s) == 0 {
			return refuse("%d shards over %d packages leaves one empty", *of, len(pkgs))
		}
	}
	got := shards[*shard-1]
	fmt.Fprintf(errw, "ci-shard: shard %d of %d: %d of the module's %d packages\n", *shard, *of, len(got), len(pkgs))
	for _, p := range got {
		fmt.Fprintln(w, p)
	}
	return 0
}

// assign deals pkgs into of shards, each package once whatever order pkgs names
// them in: heaviest first, each to the lightest shard so far, the first shard
// holding its head start. Each shard lists its packages heaviest first, the order
// go test starts them in.
func assign(pkgs []string, of int) [][]string {
	sorted := slices.Clone(pkgs)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	slices.SortStableFunc(sorted, func(a, b string) int { return cmp.Compare(weightOf(b), weightOf(a)) })

	shards := make([][]string, of)
	loads := make([]int, of)
	loads[0] = firstShardHead
	for _, p := range sorted {
		lightest := 0
		for i, l := range loads {
			if l < loads[lightest] {
				lightest = i
			}
		}
		shards[lightest] = append(shards[lightest], p)
		loads[lightest] += weightOf(p)
	}
	return shards
}

// weightOf is a package's measured seconds, or the default for one the table
// does not name.
func weightOf(pkg string) int {
	if w, ok := weights[strings.TrimPrefix(pkg, modulePath)]; ok {
		return w
	}
	return defaultWeight
}

// goList names the module's packages as `go test ./...` would resolve them, from
// dir.
func goList(dir string) ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Join(err, errors.New(strings.TrimSpace(stderr.String())))
	}
	return strings.Fields(string(out)), nil
}
