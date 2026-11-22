package guard

import "testing"

// checkOptionValues runs the lines through the default registry: each of
// allowed must not block, and each key of blocked must block under its
// entry.
func checkOptionValues(t *testing.T, allowed []string, blocked map[string]string) {
	t.Helper()
	r := Defaults()
	for _, line := range allowed {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
	for line, id := range blocked {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
}

// TestGuardReadsAQuotedPositionalAsOneOptionValue: a quoted positional
// parameter, `"$1"` to `"$9"` or `"${1}"`, is one word in bash and zsh, so as
// the value of one of env's or sudo's options it is that option's value. A
// positional with an operator, a subscript or a nesting is still read as
// possibly several words.
func TestGuardReadsAQuotedPositionalAsOneOptionValue(t *testing.T) {
	checkOptionValues(t, []string{
		`env -u "$1" opencode run x`,
		`sudo -u "$9" opencode run x`,
		`sudo -u "${1}" opencode run x`,
		`env -C "${12}" claude -p x`,
		`env -C "$1/sub" claude -p x`,
	}, map[string]string{
		`env -u "$@" opencode run x`:        "agent-launch-opencode-config-env",
		`env -u "${1+"$@"}" opencode run x`: "agent-launch-opencode-config-env",
		`env -u "${1:-x}" opencode run x`:   "agent-launch-opencode-config-env",
		`env -u "$1[@]" opencode run x`:     "agent-launch-opencode-config-env",
		`sudo -u "${a[@]}" opencode run x`:  "agent-launch-opencode-config-env",
		`env -u $1 opencode run x`:          "agent-launch-opencode-config-env",
		`env -u ${1} opencode run x`:        "agent-launch-opencode-config-env",
	})
}

// TestGuardReadsAQuotedSubstitutionAsOneOptionValue: a double-quoted command
// substitution or arithmetic expansion, `"$(…)"`, its backtick spelling or
// `"$((…))"`, is one word in bash and zsh, whatever it prints, so as the
// value of one of env's or sudo's options it is that option's value. An
// unquoted one, or a quoted one beside an expansion that can yield several
// words, is still read as possibly several.
func TestGuardReadsAQuotedSubstitutionAsOneOptionValue(t *testing.T) {
	checkOptionValues(t, []string{
		`sudo -u "$(id -un)" opencode run x`,
		"sudo -u \"`id -un`\" opencode run x",
		`env -u "$(cat v)" opencode run x`,
		`env -C "$(pwd)/sub" claude -p x`,
		`env -C "$((n+1))" claude -p x`,
	}, map[string]string{
		`env -u $(cat v) opencode run x`:             "agent-launch-opencode-config-env",
		"env -u `cat v` opencode run x":              "agent-launch-opencode-config-env",
		`env -u $((n)) opencode run x`:               "agent-launch-opencode-config-env",
		`env -u "$(cat v)"$@ opencode run x`:         "agent-launch-opencode-config-env",
		`env -u "$(cat v)$@" opencode run x`:         "agent-launch-opencode-config-env",
		`env -u "${(f)$(cat v)}" opencode run x`:     "agent-launch-opencode-config-env",
		`sudo -u "$(id -un)" "$PAIR" opencode run x`: "agent-launch-opencode-config-env",
	})
}
