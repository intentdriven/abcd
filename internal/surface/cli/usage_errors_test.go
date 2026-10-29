package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

// usageMarker is a distinctive operand built at run time: lowercase letters
// only, so it reads as a verb-shaped word wherever a verb is expected (the
// update verb refuses such a word as an unknown sub-command), and it appears
// in no command name, flag, help text or vocabulary.
func usageMarker() string {
	n := time.Now().UnixNano()
	var b strings.Builder
	b.WriteString("zq")
	for range 12 {
		b.WriteByte(byte('a' + n%26))
		n /= 26
	}
	return b.String()
}

// runUsage runs the front door with a scratch home, returning the exit code
// and both streams.
func runUsage(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	var out, errOut strings.Builder
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// repeats reports whether text carries the marker in any letter case: a verb
// that upper-cases its operand (rules domains) must not repeat it either.
func repeats(text, marker string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(marker))
}

// usageCase is one invocation whose refusal must not repeat the marker it
// carries, while still naming what the caller needs to correct the call.
type usageCase struct {
	name  string
	args  func(m string) []string
	code  int
	names []string // each must appear in the refusal
}

// checkUsageCase runs c plainly and under --json: in both forms the exit code
// is c.code, the marker reaches neither stream, and the refusal names each of
// c.names. Under --json the refusal is the usual error object on stdout.
func checkUsageCase(t *testing.T, c usageCase) {
	t.Helper()
	m := usageMarker()
	args := c.args(m)

	code, out, errOut := runUsage(t, args...)
	if code != c.code {
		t.Errorf("%s: exit %d, want %d\nstderr: %s", c.name, code, c.code, errOut)
	}
	if repeats(out, m) || repeats(errOut, m) {
		t.Errorf("%s: a usage error repeated what was passed\nstdout: %s\nstderr: %s", c.name, out, errOut)
	}
	for _, want := range c.names {
		if !strings.Contains(errOut, want) {
			t.Errorf("%s: the refusal does not name %q\nstderr: %s", c.name, want, errOut)
		}
	}

	code, out, errOut = runUsage(t, append([]string{"--json"}, args...)...)
	if code != c.code {
		t.Errorf("%s --json: exit %d, want %d", c.name, code, c.code)
	}
	if repeats(out, m) || repeats(errOut, m) {
		t.Errorf("%s --json: a usage error repeated what was passed\nstdout: %s\nstderr: %s", c.name, out, errOut)
	}
	var env struct {
		Abcd     string `json:"abcd"`
		Error    string `json:"error"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Errorf("%s --json: stdout is not the error object: %v\n%s", c.name, err, out)
		return
	}
	if env.Abcd != "error" || env.ExitCode != c.code {
		t.Errorf("%s --json: object = %+v, want abcd=error exit_code=%d", c.name, env, c.code)
	}
	for _, want := range c.names {
		if !strings.Contains(env.Error, want) {
			t.Errorf("%s --json: the error field does not name %q: %s", c.name, want, env.Error)
		}
	}
}

// TestUsageErrorUnknownSubcommandNamesTheVerbNotTheOperand: a word that names
// no sub-command is refused naming the verb and listing its sub-commands, and
// never repeats what was passed.
func TestUsageErrorUnknownSubcommandNamesTheVerbNotTheOperand(t *testing.T) {
	for _, c := range []usageCase{
		{"root", func(m string) []string { return []string{m} }, 2,
			[]string{`unknown command for "abcd"`, "first argument", "capture", "intent"}},
		{"docs", func(m string) []string { return []string{"docs", m} }, 2,
			[]string{`unknown command for "abcd docs"`, "first argument", "fidelity"}},
		{"implement", func(m string) []string { return []string{"implement", m} }, 2,
			[]string{`unknown command for "abcd implement"`, "first argument", "mode"}},
		{"update", func(m string) []string { return []string{"update", m} }, 2,
			[]string{`unknown command for "abcd update"`, "first argument", "news"}},
		{"capture list", func(m string) []string { return []string{"capture", "list", m} }, 2,
			[]string{`unknown command for "abcd capture list"`, "first argument", "takes no sub-command"}},
	} {
		checkUsageCase(t, c)
	}
}

// TestUsageErrorFlagNamesTheFlagNotTheValue: a flag refusal names the flag
// (and, for a value it could not read, the type it takes), never the value or
// the cluster it came in.
func TestUsageErrorFlagNamesTheFlagNotTheValue(t *testing.T) {
	for _, c := range []usageCase{
		{"invalid int value", func(m string) []string {
			return []string{"implement", "mode", "single", "--session", "s1", "--window", m}
		}, 2, []string{`"--window"`, "int"}},
		{"invalid bool value", func(m string) []string {
			return []string{"capture", "list", "--json=" + m}
		}, 2, []string{`"--json"`, "bool"}},
		{"unknown shorthand cluster", func(m string) []string {
			return []string{"docs", "-Z" + m}
		}, 2, []string{"unknown shorthand flag: 'Z'"}},
		{"bad flag syntax", func(m string) []string {
			return []string{"docs", "---" + m}
		}, 2, []string{"bad flag syntax"}},
	} {
		checkUsageCase(t, c)
	}
}

// TestUsageErrorFlagDescriptionsFromTheParser feeds describeFlagError the
// parser's own errors, including the two shapes no abcd verb can provoke today
// because none declares a shorthand that takes a value.
func TestUsageErrorFlagDescriptionsFromTheParser(t *testing.T) {
	m := usageMarker()
	parse := func(args ...string) error {
		fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
		fs.SetOutput(new(strings.Builder))
		fs.BoolP("all", "a", false, "")
		fs.IntP("count", "c", 0, "")
		fs.StringP("name", "n", "", "")
		err := fs.Parse(args)
		if err == nil {
			t.Fatalf("parse %v: no error", args)
		}
		return err
	}
	for _, c := range []struct {
		args  []string
		hide  string
		names []string
	}{
		{[]string{"-c" + m}, m, []string{`"-c, --count"`, "int"}},
		{[]string{"--count=" + m}, m, []string{`"-c, --count"`, "int"}},
		{[]string{"-a" + m}, m, []string{"unknown shorthand flag: 'z'"}},
		{[]string{"-aan"}, "in -", []string{"flag needs an argument: 'n'", "--name"}},
		{[]string{"--=" + m}, m, []string{"bad flag syntax"}},
	} {
		got := describeFlagError(parse(c.args...)).Error()
		if repeats(got, c.hide) {
			t.Errorf("%v: %q repeats what was passed", c.args, got)
		}
		for _, want := range c.names {
			if !strings.Contains(got, want) {
				t.Errorf("%v: %q does not name %q", c.args, got, want)
			}
		}
	}
	// An unknown long flag keeps the parser's words: they name the flag, and
	// the retired-flag, help and stale-binary answers read them.
	if got := describeFlagError(parse("--nosuch")).Error(); got != "unknown flag: --nosuch" {
		t.Errorf("unknown long flag = %q, want the parser's line unchanged", got)
	}
}

// TestUsageErrorSiblingRefusalsNameTheVocabularyNotTheOperand: abcd's own
// refusals of a closed-set operand list what is accepted and never repeat what
// was passed.
func TestUsageErrorSiblingRefusalsNameTheVocabularyNotTheOperand(t *testing.T) {
	for _, c := range []usageCase{
		{"rules domain", func(m string) []string { return []string{"rules", m} }, 2,
			[]string{"abcd rules", "COMMITTING"}},
		{"implement mode", func(m string) []string { return []string{"implement", "mode", m, "--session", "s1"} }, 2,
			[]string{"single", "split-roles"}},
		{"implement role", func(m string) []string { return []string{"implement", "join", "--session", "s1", "--role", m} }, 2,
			[]string{"first", "second"}},
	} {
		checkUsageCase(t, c)
	}

	m := usageMarker()
	if _, err := parseRecurs("rdi-2608300000000004," + m); err == nil || strings.Contains(err.Error(), m) ||
		!strings.Contains(err.Error(), "rdi-N") {
		t.Errorf("parseRecurs: %v; want a refusal naming the shape, not the token passed", err)
	}
}
