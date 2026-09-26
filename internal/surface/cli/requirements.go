package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

// requirements.go — every unmet requirement of a verb in one refusal
// (iss-2609100531051385).
//
// A verb's requirements were revealed one refusal at a time: the positional
// count refusal ("accepts 2 arg(s), received 1") named no flag, and a verb
// checking two required flags refused on the first and only then on the
// second, so a caller paid one round trip per requirement — every time, for an
// autonomous caller with no memory of the last session. The Use line is where
// each verb already declares its requirements: positionals, and the flags it
// writes outside square brackets (`resolve <iss-N> <note> --impact <…> [--grounds …]`).
// That declaration is read once, here, so the refusal cannot drift from it.
//
// For a verb whose Use line declares a required flag, the refusal is
// aggregated when the positionals are wrong, or when more than one requirement
// is unmet. A single missing flag with correct positionals is
// left to the verb's own refusal, which names that flag with what the verb
// knows about it (its allowed values, why it has no default). Nothing is
// loosened: every refusal that fired before fires now, only its message grows.

// usageRequirements returns the flags a Use line declares required: each flag
// written outside square, angle and round brackets, one group per flag, and one
// group holding every alternative of a `--a|--b` spelling (one of them is
// required). Round brackets hold a conditional requirement (`(--grounds <t>, or
// --exit-condition <t> when held)`), which no single call can be judged by.
// A Use line offering whole alternative forms (`audit [<itd-N>] | audit
// --issue-drift`) declares no single requirement set, so it yields none.
func usageRequirements(use string) [][]string {
	var (
		depth int
		top   strings.Builder
	)
	for i, r := range use {
		switch r {
		case '[', '<', '(':
			depth++
			continue
		case ']', '>', ')':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth != 0 {
			continue
		}
		if r == '|' && i > 0 && use[i-1] == ' ' {
			return nil // whole alternative forms
		}
		top.WriteRune(r)
	}
	var groups [][]string
	for _, tok := range strings.Fields(top.String()) {
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		var g []string
		for _, alt := range strings.Split(tok, "|") {
			if strings.HasPrefix(alt, "--") && len(alt) > 2 {
				g = append(g, alt)
			}
		}
		if len(g) > 0 {
			groups = append(groups, g)
		}
	}
	return groups
}

// aggregateUsageRequirements wraps every command's positional validator so a
// refusal names every unmet requirement the Use line declares at once. A
// validator that already chose its own refusal (an *exitError, such as the
// hook plane's fail-open one) is left alone.
func aggregateUsageRequirements(c *cobra.Command) {
	validate := c.Args
	reqs := usageRequirements(c.Use)
	// Only a verb whose Use line declares a required flag is wrapped: with none,
	// the positional refusal already names everything unmet, and cobra's own
	// lines (an unknown command above all) stay byte-for-byte. A command with no
	// validator of its own runs cobra's legacy check, which matters only for a
	// parent (unknown sub-verbs), so a parent is never wrapped without one.
	if len(reqs) > 0 && (validate != nil || !c.HasSubCommands()) {
		c.Args = func(cmd *cobra.Command, args []string) error {
			var unmet []string
			positionalFailed := false
			if validate != nil {
				if err := validate(cmd, args); err != nil {
					var coded *exitError
					if errors.As(err, &coded) {
						return err
					}
					unmet = append(unmet, err.Error())
					positionalFailed = true
				}
			}
			for _, g := range reqs {
				if !anyFlagChanged(cmd, g) {
					unmet = append(unmet, strings.Join(g, " or ")+" is not set")
				}
			}
			if len(unmet) == 0 || (len(unmet) == 1 && !positionalFailed) {
				return nil
			}
			return &exitError{Code: 2, Msg: strings.Join(unmet, "; ") +
				" — usage: " + cmd.UseLine() + " (nothing written)"}
		}
	}
	for _, sub := range c.Commands() {
		aggregateUsageRequirements(sub)
	}
}

func anyFlagChanged(cmd *cobra.Command, names []string) bool {
	for _, n := range names {
		if f := cmd.Flags().Lookup(strings.TrimPrefix(n, "--")); f != nil && f.Changed {
			return true
		}
	}
	return false
}
