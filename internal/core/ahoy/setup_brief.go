package ahoy

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// The one-line form of every setup question (itd-2610240823055889, criterion
// 6, and the amendment of brief invariant 1 for setup): a question is shown
// as its one line and its answers' labels, and answering ? shows the whole
// question (the material, what each answer does, what holds now and where to
// change it later) and asks again. The plan screen keeps its plan, which is
// the screen; only what each answer does waits for ?.

// ExplainHint closes the one line of every setup question.
const ExplainHint = "(? explains)"

// briefValueAsks is the one line of each value question whose full form asks
// the generic setupAsk.
var briefValueAsks = map[string]string{
	"visibility":     "Commit abcd's records with your code (private), or keep them out of git (public)?",
	"docs_target":    "Add abcd's block to AGENTS.md (agents_md), or leave AGENTS.md alone (skip)?",
	"oracle_backend": "Which AI reviewer does abcd use?",
	"scan_deep":      "Record that this repository wants the trufflehog deep secret scan?",
	artefactKindKey:  "What does this repository release?",
	emDashPromptKey:  "Does an em dash in a list item block the docs check, or only warn?",
}

// BriefQuestion is q in its one-line form: the question's one line, ending on
// ExplainHint, and the answers' labels, with the material, the meanings and
// the Now and change-later lines left for ?. The plan screen keeps its
// material, which is the plan itself.
func BriefQuestion(q question.Question) question.Question {
	b := q
	if q.ID != PlanQuestionID {
		b.Material = nil
	}
	b.Ask = briefAsk(q) + " " + ExplainHint
	b.Options = make([]question.Option, len(q.Options))
	for i, o := range q.Options {
		o.Meaning = ""
		b.Options[i] = o
	}
	if q.List != nil {
		l := *q.List
		l.Choices = make([]question.Option, len(q.List.Choices))
		for i, o := range q.List.Choices {
			o.Meaning = ""
			l.Choices[i] = o
		}
		b.List = &l
	}
	b.Later.Meaning = ""
	b.Now, b.ChangeLater = "", ""
	return b
}

// briefAsk is the one line q is asked by in its one-line form.
func briefAsk(q question.Question) string {
	if q.Ask == setupAsk {
		if a, ok := briefValueAsks[q.ID]; ok {
			return a
		}
		if el, ok := strings.CutPrefix(q.ID, elementPromptPrefix); ok {
			if shows, known := statusLineElementAbout[statusline.ElementKey(el)]; known {
				return "Show " + shows + " on the status line?"
			}
		}
		return "Choose " + q.ID + "?"
	}
	// An approval of a kind of change points at its list, which the plan
	// already showed and ? shows again.
	return strings.Replace(q.Ask, "listed above", "on the plan", 1)
}
