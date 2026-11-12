package lint

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// ruleIntentLayout is iss-2611121000338677: an intent whose text cannot be read
// whole — a fence or HTML comment nothing closes, or fences or comments whose
// extent turns on the layout around them (a fence indented four columns or
// more under a list item among them) — is not ready at the build's
// open_questions row, which reads it through spec.LayoutStatements. This rule
// names each such statement where the intent is written, on its line, through
// the same reader, so the two never disagree about what the layout says.
// spec_unsettled reports the same statements for an open spec.
const ruleIntentLayout = "intent_layout"

// checkIntentLayout names each statement spec.LayoutStatements finds in a
// draft or planned intent, one finding on its line, at the configured
// severity. It reads the text the intent-tree scan already read (bounded by
// maxRepoFileBytes), never the file a second time.
//
// WHY drafts and planned: the open_questions row judges every bucket but the
// three that are never built, and a draft meets it when it is planned
// (autoplan) and when `abcd intent ready` judges it; shipped, superseded and
// disciplines are exempt there and here. WHY warn in the shipped config: the
// row already refuses the intent at the start of a run or a plan; this rule
// moves the report to the change that writes the layout without making a
// record-lint run fail on it.
func checkIntentLayout(tree intentTree, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range tree.records {
		if r.bucket != "planned" && r.bucket != "drafts" {
			continue
		}
		for _, st := range spec.LayoutStatements(strings.Join(r.lines, "\n")) {
			out = append(out, Finding{
				File: r.rel, Line: st.Line, RuleID: ruleIntentLayout, Severity: cfg.Severity,
				Message: "intent's text is not read whole, so the build's open_questions check refuses it: " +
					st.Describe() + " — " + spec.LayoutRemedy,
			})
		}
	}
	return out
}
