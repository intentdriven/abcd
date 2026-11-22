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

// TestGuardReadsAQuotedNumberParameterAsOneOptionValue: `"$!"` and
// `"${#NAME}"` print a number, or nothing for a `$!` with no job, and are one
// word in double quotes, so as the value of one of env's or sudo's options
// they are that option's value. An unquoted `$!`, and a length with a
// subscript or an operator, are still read as possibly several words.
func TestGuardReadsAQuotedNumberParameterAsOneOptionValue(t *testing.T) {
	checkOptionValues(t, []string{
		`env -u "$!" opencode run x`,
		`sudo -u "${#u}" opencode run x`,
		`sudo -u "${#1}" opencode run x`,
	}, map[string]string{
		`env -u $! opencode run x`:             "agent-launch-opencode-config-env",
		`env -u "${#a[@]}" opencode run x`:     "agent-launch-opencode-config-env",
		`env -u "${#@}" opencode run x`:        "agent-launch-opencode-config-env",
		`env -u "${#v:-x}" opencode run x`:     "agent-launch-opencode-config-env",
		`env -u "$!$@" opencode run x`:         "agent-launch-opencode-config-env",
		`env -u "${#u}${a[@]}" opencode run x`: "agent-launch-opencode-config-env",
	})
}

// TestGuardReadsAColonAfterAQuotedNameAsText: a `:` after a quoted `"$NAME"`
// is text unless a zsh modifier letter follows it (zshexpn(1), Modifiers), so
// `"$USER: "` is one word and, as the value of one of env's or sudo's
// options, that option's value. A modifier is still read as possibly several
// words, as the guard does not work out what each prints.
func TestGuardReadsAColonAfterAQuotedNameAsText(t *testing.T) {
	checkOptionValues(t, []string{
		`sudo -p "Password for $USER: " claude -p x`,
		`env -C "$d:/x" claude -p x`,
		`env -u "$v:" opencode run x`,
		`env -u "$v::" opencode run x`,
		`env -u "$v:1" opencode run x`,
		`env -u "$1: x" opencode run x`,
	}, map[string]string{
		`env -u "$v:h" opencode run x`:            "agent-launch-opencode-config-env",
		`env -u "$v:t" opencode run x`:            "agent-launch-opencode-config-env",
		`env -u "$v:A" opencode run x`:            "agent-launch-opencode-config-env",
		`env -u "$v:Q" opencode run x`:            "agent-launch-opencode-config-env",
		`env -u "$v:s/a/b c/" opencode run x`:     "agent-launch-opencode-config-env",
		`env -u "$v:gs/a/b/" opencode run x`:      "agent-launch-opencode-config-env",
		`env -u "$v:&" opencode run x`:            "agent-launch-opencode-config-env",
		`env -u "$v:W: :u" opencode run x`:        "agent-launch-opencode-config-env",
		`env -u "$1:h" opencode run x`:            "agent-launch-opencode-config-env",
		`sudo -p "$USER: "$@ claude -p x`:         "agent-launch-claude-config-env",
		`sudo -p "Password for $@: " claude -p x`: "agent-launch-claude-config-env",
	})
}
