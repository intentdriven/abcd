package recall

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// headerLine heads the recall's block, appended after the rules block.
const headerLine = "## MEMORY\n"

// The words a recalled note's head carries for its layer, so whose words
// these are is never invisible, as an override's rule heading says "(repo
// override)" (GHSA-22f8-qf5r-gjgq). A note in the project's file is whatever
// the repository's authors committed, so a cloned or compromised repository
// controls it; a note in the machine's file was written by an agent on this
// account through the write gate.
const (
	labelProject = "project file, committed in this repository"
	labelMachine = "personal note, agent-written"
)

// LayerLabel is the provenance an item's head carries: "project file,
// committed in this repository" for the project's file, "personal note,
// agent-written" for the machine's. An unknown layer is named as itself, so
// a new layer is never rendered as one of these by default.
func LayerLabel(layer string) string {
	switch layer {
	case LayerProject:
		return labelProject
	case LayerMachine:
		return labelMachine
	}
	return layer
}

// renderItem renders one item in full: a "- " bullet naming its key, its
// layer and its title, and the body under it inside a fence. The body is
// another party's text — the project's file is the repository's — so it
// goes through the rules package's one fence for such text
// (rules.FenceRuleBody), as an override's rule body does: no markdown
// construct inside it (a heading at any indent, a setext underline, a list
// item, a run of tildes or backticks) is live to a CommonMark reader, and
// every line sits at the bullet's content column. The head line, a one-line
// title, takes the rule body's mask (rules.SanitizeRuleBody).
func renderItem(it Item) string {
	head := rules.SanitizeRuleBody(it.Key + " (" + LayerLabel(it.Layer) + "): " + it.Title)
	var b strings.Builder
	b.WriteString("- ")
	b.WriteString(head)
	b.WriteString("\n")
	if body := rules.FenceRuleBody(it.Body); body != "" {
		b.WriteString("  ")
		b.WriteString(body)
		b.WriteString("\n")
	}
	return b.String()
}
