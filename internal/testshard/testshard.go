// Package testshard lets one package's tests run as several test processes, so
// a package too long for go test's per-binary -timeout as one process fits it
// as several (iss-2610251004378066).
//
// A package opts in from its TestMain, after any branch that re-executes the
// test binary as something else and before the set-up its tests need:
//
//	if code, ok := testshard.Shard(); ok {
//		os.Exit(code)
//	}
//
// Shard does nothing unless the environment asks for shards: ShardsEnv set to a
// count above one, and go test running the whole package with nothing but the
// flags Run knows to be safe to repeat (-test.timeout, -test.count, -test.v,
// -test.short, -test.failfast, -test.paniconexit0, and go test's own
// -test.testlogfile, which each shard writes apart and Shard merges, so go
// test's cache reads the inputs of every shard). Then it lists the
// package's tests, deals them round-robin into that many shares, runs that
// many copies of its own binary at once, each over one share under -test.run
// and each with its own -test.timeout, and writes each copy's output whole, in
// order, once all have ended. The result is a failure when any copy failed.
// Any other flag (a -test.run of the caller's own, a profile, -json's
// test2json stream, a fuzz or bench run) runs the package as one process, as
// without the variable. Each shard runs the TestMain again, as a shard, and
// so makes the set-up its share needs.
//
// make preflight exports the variable; CI does not, so its runs are unchanged.
// go test's cache is unaffected: the variable is an environment read, which
// go test records like any other.
package testshard

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// ShardsEnv is the number of test processes a package that opts in runs as.
const ShardsEnv = "ABCD_TEST_SHARDS"

// childEnv marks a process Shard started, which runs its share as one.
const childEnv = "ABCD_TEST_SHARD_CHILD"

// Shard runs the test binary's tests as several processes when the
// environment asks for shards (see the package comment), returning their exit
// code and true; otherwise it returns false and the caller runs its tests.
func Shard() (int, bool) {
	n, ok := shards(os.Getenv, os.Args[1:])
	if !ok {
		return 0, false
	}
	return runShards(os.Args[0], os.Args[1:], n, os.Stdout, os.Stderr), true
}

// shards reports how many processes to run as, and whether to shard at all:
// only for a count above one, outside a shard, over the whole package and
// with only the flags safe to repeat in every process.
func shards(getenv func(string) string, args []string) (int, bool) {
	if getenv(childEnv) != "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(getenv(ShardsEnv)))
	if err != nil || n < 2 {
		return 0, false
	}
	for _, a := range args {
		if !repeatable(a) {
			return 0, false
		}
	}
	return n, true
}

// repeatable reports whether a test binary flag means the same in every
// shard. -test.v=test2json is left out: -json reads one framed stream.
func repeatable(arg string) bool {
	name, value, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
	switch name {
	case "test.timeout", "test.count", "test.short", "test.failfast", "test.paniconexit0", "test.testlogfile":
		return true
	case "test.v":
		return value != "test2json"
	}
	return false
}

// deal splits names round-robin into at most n non-empty shares, each a
// -test.run pattern matching exactly its names.
func deal(names []string, n int) []string {
	if n > len(names) {
		n = len(names)
	}
	shares := make([][]string, n)
	for i, name := range names {
		shares[i%n] = append(shares[i%n], regexp.QuoteMeta(name))
	}
	patterns := make([]string, n)
	for i, s := range shares {
		patterns[i] = "^(" + strings.Join(s, "|") + ")$"
	}
	return patterns
}

// runShards lists bin's tests, runs n copies of it at once over a share each,
// and writes their output in order once every copy has ended.
func runShards(bin string, args []string, n int, stdout, stderr io.Writer) int {
	env := append(os.Environ(), childEnv+"=1")
	list := exec.Command(bin, "-test.list", ".")
	list.Env = env
	out, err := list.Output()
	if err != nil {
		fmt.Fprintf(stderr, "testshard: listing the tests failed (%v); running them as one process\n", err)
		return single(bin, args, env, stdout, stderr)
	}
	var names []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "ok ") {
			names = append(names, l)
		}
	}
	if len(names) == 0 {
		return single(bin, args, env, stdout, stderr)
	}
	patterns := deal(names, n)
	logfile, rest := testlogArg(args)

	cmds := make([]*exec.Cmd, len(patterns))
	outs := make([]bytes.Buffer, len(patterns))
	for i, p := range patterns {
		a := append(append([]string{}, rest...), "-test.run="+p)
		if logfile != "" {
			a = append(a, "-test.testlogfile="+shardLog(logfile, i))
		}
		c := exec.Command(bin, a...)
		c.Env = env
		c.Stdout, c.Stderr = &outs[i], &outs[i]
		cmds[i] = c
	}
	// A signal to this process, go test's own timeout among them, reaches
	// every shard, which then dumps its goroutines as one process would.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sig)
	codes := make([]int, len(cmds))
	var wg sync.WaitGroup
	for i, c := range cmds {
		if err := c.Start(); err != nil {
			fmt.Fprintf(&outs[i], "testshard: shard %d did not start: %v\n", i+1, err)
			codes[i] = 1
			continue
		}
		wg.Add(1)
		go func(i int, c *exec.Cmd) {
			defer wg.Done()
			codes[i] = exitCode(c.Wait())
		}(i, c)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for waiting := true; waiting; {
		select {
		case s := <-sig:
			for _, c := range cmds {
				if c.Process != nil {
					_ = c.Process.Signal(s)
				}
			}
		case <-done:
			waiting = false
		}
	}

	if logfile != "" {
		if err := mergeLogs(logfile, len(cmds)); err != nil {
			fmt.Fprintf(stderr, "testshard: %v\n", err)
		}
	}
	code := 0
	var failed []string
	for i := range cmds {
		fmt.Fprintf(stdout, "=== testshard: shard %d of %d\n", i+1, len(cmds))
		_, _ = stdout.Write(outs[i].Bytes())
		if codes[i] != 0 {
			code = 1
			failed = append(failed, strconv.Itoa(i+1))
		}
	}
	if code != 0 {
		fmt.Fprintf(stdout, "testshard: shard %s of %d failed\n", strings.Join(failed, ", "), len(cmds))
	}
	return code
}

// testlogArg splits go test's -test.testlogfile out of args, returning its
// file and the rest.
func testlogArg(args []string) (string, []string) {
	var file string
	var rest []string
	for _, a := range args {
		if v, ok := strings.CutPrefix(strings.TrimLeft(a, "-"), "test.testlogfile="); ok {
			file = v
			continue
		}
		rest = append(rest, a)
	}
	return file, rest
}

// shardLog is the test log shard i writes, beside go test's own.
func shardLog(file string, i int) string { return file + ".shard" + strconv.Itoa(i+1) }

// testlogMagic opens every test log (testing/internal/testdeps, read by
// cmd/go/internal/test).
const testlogMagic = "# test log\n"

// mergeLogs writes the n shards' test logs into file as one log, each
// shard's inputs after a chdir back to the package directory every shard
// started in, so the relative paths in each resolve as they did, and removes
// the shards' own. A shard that wrote no log adds nothing: it failed, and go
// test caches no failure.
func mergeLogs(file string, n int) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("the test log was not merged: %w", err)
	}
	var b strings.Builder
	b.WriteString(testlogMagic)
	for i := range n {
		data, err := os.ReadFile(shardLog(file, i))
		if err != nil {
			continue
		}
		_ = os.Remove(shardLog(file, i))
		b.WriteString("chdir " + dir + "\n")
		b.WriteString(strings.TrimPrefix(string(data), testlogMagic))
	}
	if err := os.WriteFile(file, []byte(b.String()), 0o666); err != nil {
		return fmt.Errorf("the test log was not merged: %w", err)
	}
	return nil
}

// single runs bin's tests as one process.
func single(bin string, args, env []string, stdout, stderr io.Writer) int {
	c := exec.Command(bin, args...)
	c.Env = env
	c.Stdout, c.Stderr = stdout, stderr
	return exitCode(c.Run())
}

// exitCode is a finished process's exit status: 0 for none, 1 for an error
// that carries no status.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() > 0 {
		return ee.ExitCode()
	}
	return 1
}
