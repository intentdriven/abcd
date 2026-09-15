//go:build smoke

package evals

// The self-discovering smoke harness: it walks the Cobra command tree
// in-process (so a command added tomorrow is covered with no edit here) and
// exercises each one against the binary harness_test.go builds. Gated behind
// the `smoke` build tag so it does not slow the unit-test lane.
//
// v1 smokes structure only (help renders, no panic, flags parse, read-only verbs
// run). Fixture-driven per-command scenarios (evals/data/) are future work — see
// intent itd-75.

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/surface/cli"
	"github.com/spf13/cobra"
)

// commandArgs walks the Cobra tree from the real root command and returns each
// command as the arg slice needed to invoke it (root program name excluded).
// Includes hidden and auto-generated (help/completion) commands — their --help
// must render too.
func commandArgs() [][]string {
	var paths [][]string
	var walk func(c *cobra.Command, prefix []string)
	walk = func(c *cobra.Command, prefix []string) {
		for _, sub := range c.Commands() {
			p := append(append([]string(nil), prefix...), sub.Name())
			paths = append(paths, p)
			walk(sub, p)
		}
	}
	walk(cli.NewRootCommand(), nil)
	return paths
}

// panicked reports whether output carries a Go runtime panic/stack trace — the
// failure mode this harness exists to catch (a command that compiles but crashes
// when actually invoked).
func panicked(out string) bool {
	return strings.Contains(out, "panic:") || strings.Contains(out, "goroutine ")
}

// TestEveryCommandHelpRenders is the core smoke: for every discovered command,
// `--help` must exit 0, produce output, and never panic. --help short-circuits
// before arg/flag validation in Cobra, so this is safe for commands with required
// args, yet still proves the command is wired and its help text builds.
func TestEveryCommandHelpRenders(t *testing.T) {
	cmds := commandArgs()
	if len(cmds) == 0 {
		t.Fatal("discovered zero commands — the command tree walk is broken")
	}
	for _, p := range cmds {
		p := p
		t.Run(strings.Join(p, "/"), func(t *testing.T) {
			args := append(append([]string(nil), p...), "--help")
			out, code := run(t, args...)
			if panicked(out) {
				t.Fatalf("`abcd %s --help` panicked:\n%s", strings.Join(p, " "), out)
			}
			if code != 0 {
				t.Errorf("`abcd %s --help` exit=%d\n%s", strings.Join(p, " "), code, out)
			}
			if strings.TrimSpace(out) == "" {
				t.Errorf("`abcd %s --help` produced no output", strings.Join(p, " "))
			}
		})
	}
}

// TestReadOnlyVerbsRun executes the safe, no-argument, read-only verbs for real
// (not just --help), asserting they run to a graceful exit without panicking.
func TestReadOnlyVerbsRun(t *testing.T) {
	cases := []struct {
		args     []string
		wantZero bool // version/help must be 0; the bare status board may report non-zero
	}{
		{[]string{"--help"}, true},
		{[]string{"version"}, true},
		{[]string{}, false}, // bare status board: no panic, any exit
	}
	for _, tc := range cases {
		out, code := run(t, tc.args...)
		label := "abcd " + strings.Join(tc.args, " ")
		if panicked(out) {
			t.Errorf("`%s` panicked:\n%s", label, out)
		}
		if tc.wantZero && code != 0 {
			t.Errorf("`%s` exit=%d, want 0\n%s", label, code, out)
		}
	}
}

// TestUnknownFlagIsGraceful proves an unknown flag is a clean non-zero error, not
// a panic — flag parsing must degrade gracefully on bad input.
func TestUnknownFlagIsGraceful(t *testing.T) {
	out, code := run(t, "version", "--definitely-not-a-real-flag")
	if panicked(out) {
		t.Fatalf("unknown flag panicked:\n%s", out)
	}
	if code == 0 {
		t.Errorf("unknown flag unexpectedly succeeded (exit 0):\n%s", out)
	}
}
