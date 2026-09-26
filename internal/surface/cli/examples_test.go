package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// examples_test.go — every verb that takes a required input carries one worked
// example in its own --help (iss-2609100508565741), and each example is held to
// the command it names, so it cannot drift into an invocation the verb refuses
// on its face.

// declaresRequiredInput reports whether a Use line, once every square-bracketed
// (optional) part is removed, still names an operand or a flag after the verb.
func declaresRequiredInput(use string) bool {
	var b strings.Builder
	depth := 0
	for _, r := range use {
		switch {
		case r == '[':
			depth++
		case r == ']' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	_, rest, _ := strings.Cut(strings.TrimSpace(b.String()), " ")
	return strings.ContainsAny(rest, `<"`) || strings.Contains(rest, "--")
}

// exampleTokens splits an example the way a POSIX shell would for these
// simple lines: whitespace-separated, with single or double quotes grouping.
func exampleTokens(s string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	in := false
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote == 0 && (r == '\'' || r == '"'):
			quote, in = r, true
		case quote == 0 && r == ' ':
			if in {
				out = append(out, cur.String())
				cur.Reset()
				in = false
			}
		default:
			cur.WriteRune(r)
			in = true
		}
	}
	if in {
		out = append(out, cur.String())
	}
	return out
}

// visibleCommands (sentences_test.go) is every command a reader can see.

func TestEveryVerbWithARequiredInputCarriesAWorkedExample(t *testing.T) {
	root := NewRootCommand()
	for _, c := range visibleCommands(root) {
		if !declaresRequiredInput(c.Use) {
			continue
		}
		want, ok := surface.ExampleFor(c.CommandPath())
		if !ok {
			t.Errorf("`%s` (%s) takes a required input and has no worked example in internal/core/surface/examples.go", c.CommandPath(), c.Use)
			continue
		}
		if !strings.Contains(c.Example, want) {
			t.Errorf("`%s --help` does not render its worked example %q (Example = %q)", c.CommandPath(), want, c.Example)
		}
		help := string(runCLI(t, append(strings.Fields(strings.TrimPrefix(c.CommandPath(), "abcd ")), "--help")...))
		if !strings.Contains(help, want) {
			t.Errorf("`%s --help` output lacks the example:\n%s", c.CommandPath(), help)
		}
	}
}

func TestEachWorkedExampleIsShapedLikeItsCommand(t *testing.T) {
	root := NewRootCommand()
	byPath := map[string]*cobra.Command{}
	for _, c := range visibleCommands(root) {
		byPath[c.CommandPath()] = c
	}
	for _, path := range surface.ExamplePaths() {
		c, ok := byPath[path]
		if !ok {
			t.Errorf("examples.go names %q, which is not a visible command", path)
			continue
		}
		ex, _ := surface.ExampleFor(path)
		if !strings.HasPrefix(ex, path+" ") && ex != path {
			t.Errorf("%q's example does not begin with the command: %q", path, ex)
			continue
		}
		toks := exampleTokens(ex)
		given := map[string]bool{}
		for _, tok := range toks {
			if !strings.HasPrefix(tok, "--") {
				continue
			}
			name, _, _ := strings.Cut(strings.TrimPrefix(tok, "--"), "=")
			given["--"+name] = true
			if c.Flags().Lookup(name) == nil && c.InheritedFlags().Lookup(name) == nil {
				t.Errorf("%q's example passes --%s, which the command does not register", path, name)
			}
		}
		for _, group := range usageRequirements(c.Use) {
			met := false
			for _, f := range group {
				met = met || given[f]
			}
			if !met {
				t.Errorf("%q's example omits the required %s its Use line declares", path, strings.Join(group, " or "))
			}
		}
	}
}
