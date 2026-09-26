package main

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// decisionsAppendUsage is the mode's invocation. Both refs are positional and
// both are required: the base may be the empty string — a CI event that names
// no base passes one through — and the mode then SAYS it skipped, but an absent
// argument is a malformed invocation, refused rather than read as "nothing to
// check".
const decisionsAppendUsage = "usage: record-lint decisions-append <base-ref> <head-ref>"

// runDecisionsAppend is record-lint's decisions-append mode: DA001–DA004 over
// every commit in base..head (internal/core/lint, CheckDecisionsAppend). It
// prints each finding as `file:line: [BLOCKER DAnnn] message` on stdout and the
// verdict last, and returns the exit code: 0 clean or an announced skip, 1 a
// rule violation, 2 the gate could not answer (usage, a repository git cannot
// read, a shallow checkout, a ref that resolves to nothing, an unanchorable
// merge base). The three are kept apart because "could not answer" must never
// collapse into either a pass or a violation.
func runDecisionsAppend(args []string, root string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, decisionsAppendUsage)
		return 2
	}
	rep, err := lint.CheckDecisionsAppend(root, args[0], args[1])
	if err != nil {
		fmt.Fprintln(stderr, "decisions-append:", termsafe.Sanitize(scrubPaths(err, root)))
		return 2
	}
	if rep.Skipped != "" {
		fmt.Fprintln(stdout, "decisions-append:", rep.Skipped)
		return 0
	}
	rng := termsafe.Sanitize(rep.Base + ".." + rep.Head)
	if rep.Checked == 0 {
		fmt.Fprintf(stdout, "decisions-append: no commits in %s — nothing to check\n", rng)
	}
	for _, f := range rep.Findings {
		fmt.Fprintln(stdout, renderFinding(f, root))
	}
	if rep.Checked > 0 {
		fmt.Fprintf(stdout, "decisions-append: DA001-DA004 checked %d commit(s) in %s\n", rep.Checked, rng)
	}
	if n := len(rep.Findings); n > 0 {
		fmt.Fprintf(stderr, "decisions-append: FAILED — %d violation(s)\n", n)
		return 1
	}
	fmt.Fprintln(stdout, "decisions-append: OK")
	return 0
}
