package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// naming_show_operand_test.go: the naming rule in brief
// 02-constraints/04-naming.md forbids a sub-verb that only renames the bare
// render. It once forbade `<verb> show` by name, while `inbox show`,
// `history show` and `checklist show` each take the one id they render, which
// the rule's own earn-your-place test admits. dcn-2610271112330536 ruled the
// ban covers only an operand-less render (iss-2610262031019028), so the rule
// says so, and every registered show, stats or view sub-verb takes an operand.

// renderNames are the sub-verb names the rule forbids when they take no operand.
var renderNames = map[string]bool{"show": true, "stats": true, "view": true}

// TestShowSubVerbsTakeAnOperandAndTheRuleSaysSo fails when a registered
// show, stats or view sub-verb accepts a bare invocation, or when the naming
// rule's ban names `<verb> show` without confining it to an operand-less one.
func TestShowSubVerbsTakeAnOperandAndTheRuleSaysSo(t *testing.T) {
	var found int
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			// A sub-verb sits below a top-level verb, so its parent has one.
			if c.HasParent() && renderNames[sub.Name()] {
				found++
				if sub.Args == nil {
					t.Errorf("`%s` takes no operand check: a show, stats or view sub-verb must take the operand it renders", sub.CommandPath())
				} else if err := sub.Args(sub, nil); err == nil {
					t.Errorf("`%s` accepts a bare invocation: that renames the bare render, which the naming rule forbids", sub.CommandPath())
				}
			}
			walk(sub)
		}
	}
	walk(NewRootCommand())
	if found == 0 {
		t.Fatal("found no show, stats or view sub-verb: the walk reads nothing, so it proves nothing")
	}

	root, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "02-constraints", "04-naming.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(data)), " ")
	ban := regexp.MustCompile(`[^.]*\*\*forbidden\*\*[^.]*\.`).FindString(text)
	if ban == "" {
		t.Fatal("04-naming.md carries no sentence with **forbidden** in it: the ban moved, so this test must follow it")
	}
	if !strings.Contains(ban, "`<verb> show`") {
		t.Fatalf("the ban no longer names `<verb> show`:\n%s", ban)
	}
	if !strings.Contains(ban, "no operand") {
		t.Errorf("the ban forbids `<verb> show` by name, but `show <id>` takes an operand and conforms (dcn-2610271112330536); confine it to a sub-verb that takes no operand:\n%s", ban)
	}
}
