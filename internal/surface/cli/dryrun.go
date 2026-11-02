package cli

// dryrun.go is the front doors' half of the practice run (itd-2610301428390406,
// spc-2610301431131636): every command that reads a filled-in input (a flag
// ending -json) and writes from it takes --dry-run, which checks the input
// through the same validator the real write uses and reports what would be
// written, writing nothing. The core's Plan stage does the checking; this file
// only renders what it planned, so a practice run and the real write cannot
// disagree about the input.

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// dryRunFlagUsage is --dry-run's help line on every command that takes it.
const dryRunFlagUsage = "check the input and report what would be written; write nothing"

// dryRunRecordLines is how many opening lines of a single-record write the
// text render shows: the frontmatter and the first lines after it.
const dryRunRecordLines = 14

// dryRunResult is a practice run's --json: the verb's own result fields as the
// real run would fill them, then dry_run, would_write and would_do.
func dryRunResult(v any, p dryrun.Plan) any {
	p = p.Normalise()
	return withMember{v: withMember{v: withMember{v: v, key: "dry_run", val: true},
		key: "would_write", val: p.Writes}, key: "would_do", val: p.Effects}
}

// renderDryRun renders a practice run. The text form is a "practice run"
// heading naming the verb, one line per planned write and effect, the
// verb's own lines (detail), and, for a run that writes one record, that
// record's opening lines.
func renderDryRun(w io.Writer, asJSON bool, verb string, v any, p dryrun.Plan, detail func(io.Writer)) error {
	return render(w, asJSON, dryRunResult(v, p), func(w io.Writer) {
		fmt.Fprintf(w, "%s — practice run: the input is checked and nothing is written\n", verb)
		if len(p.Writes) == 0 && len(p.Effects) == 0 {
			fmt.Fprintln(w, "  would write nothing")
		}
		for _, wr := range p.Writes {
			fmt.Fprintf(w, "  %s\n", dryRunWriteLine(wr))
		}
		for _, e := range p.Effects {
			fmt.Fprintf(w, "  would %s: %s\n", termsafe.Sanitize(e.Effect), termsafe.Sanitize(e.Target))
		}
		if detail != nil {
			detail(w)
		}
		if rec, ok := singleRecord(p); ok {
			fmt.Fprintf(w, "  %s would open:\n", termsafe.Sanitize(rec.Path))
			for _, ln := range openingLines(string(rec.Content), dryRunRecordLines) {
				fmt.Fprintf(w, "    | %s\n", termsafe.Sanitize(ln))
			}
		}
	})
}

// dryRunWriteLine is one planned write as the text render says it:
// `would create .abcd/…/x.md (2.1 kB)`, sized as humanBytes sizes.
func dryRunWriteLine(wr dryrun.Write) string {
	var parts []string
	if wr.Bytes > 0 {
		parts = append(parts, humanBytes(wr.Bytes))
	}
	if wr.Pending != "" {
		parts = append(parts, wr.Pending)
	}
	line := "would " + string(wr.Action) + " " + termsafe.Sanitize(wr.Path)
	if wr.From != "" {
		line += " from " + termsafe.Sanitize(wr.From)
	}
	if len(parts) > 0 {
		line += " (" + strings.Join(parts, "; ") + ")"
	}
	return line
}

// singleRecord is the one planned write whose bytes the plan composed, when
// exactly one has them: the record a single-record write would produce. A
// write another writer composes (a decision, an issue) carries none.
func singleRecord(p dryrun.Plan) (dryrun.Write, bool) {
	var rec dryrun.Write
	n := 0
	for _, wr := range p.Writes {
		if len(wr.Content) > 0 {
			rec, n = wr, n+1
		}
	}
	return rec, n == 1
}

// openingLines is the first n lines of text.
func openingLines(text string, n int) []string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > n {
		lines = append(lines[:n], "…")
	}
	return lines
}
