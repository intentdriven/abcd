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
// A page is labelled as a page of its scope: the project's .abcd/memory/ is
// committed like the project's file, and the machine's ~/.abcd.noindex/memory/
// is this account's store, which no agent writes through the note gate.
const (
	labelProject     = "project file, committed in this repository"
	labelMachine     = "personal note, agent-written"
	labelPageProject = "project page, committed in this repository"
	labelPageMachine = "machine page, this account's memory store"
)

// layerLabel is the provenance an item's head carries: for a note, "project
// file, committed in this repository" for the project's file and "personal
// note, agent-written" for the machine's; for a page, "project page,
// committed in this repository" or "machine page, this account's memory
// store". An unknown layer is named as itself, so a new layer is never
// rendered as one of these by default.
func layerLabel(kind, layer string) string {
	switch {
	case kind == KindPage && layer == LayerProject:
		return labelPageProject
	case kind == KindPage && layer == LayerMachine:
		return labelPageMachine
	case layer == LayerProject:
		return labelProject
	case layer == LayerMachine:
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
	head := rules.SanitizeRuleBody(it.Key + " (" + layerLabel(it.Kind, it.Layer) + "): " + it.Title)
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
