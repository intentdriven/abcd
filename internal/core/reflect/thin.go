package reflect

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/record/match"
)

// The declared floor (spec scope 3). An answer is thin when it is blank, when it
// only restates its section heading, or when it holds fewer than MinClauses
// substantive clauses. A clause is substantive when it carries at least
// MinClauseWords words. The criterion's own example, "it worked", is one clause
// of two words, so it is thin twice over.
//
// The floor is a heuristic and a cheap one to be wrong about: a thin answer
// costs one follow-up question, never a refusal of the answer the follow-up
// brings.
const (
	MinClauses     = 2
	MinClauseWords = 3
)

// clauseBreakRe splits an answer into clauses: at sentence and clause
// punctuation, at a line break (so each bullet is its own clause), at a spaced
// dash, and before a conjunction that opens a new clause.
var clauseBreakRe = regexp.MustCompile(`(?i)[.;:!?,\n]|\s[-–—]\s|\b(?:and|but|because|so|which|while|although|though|since|whereas|yet)\b`)

// bulletMarkRe is a list marker at the start of a clause.
var bulletMarkRe = regexp.MustCompile(`^\s*(?:[-*+]|[0-9]+[.)])\s+`)

// underFloor reports whether an answer to section falls under the floor, and why.
func underFloor(s Section, text string) (bool, string) {
	if strings.TrimSpace(text) == "" {
		return true, "the answer is empty"
	}
	heading := map[string]bool{}
	for _, t := range match.Terms(s.Heading()) {
		heading[t] = true
	}
	restates := true
	for _, t := range match.Terms(text) {
		if !heading[t] {
			restates = false
			break
		}
	}
	if restates {
		return true, "the answer restates the section heading"
	}
	clauses := 0
	for _, c := range clauseBreakRe.Split(text, -1) {
		if len(strings.Fields(bulletMarkRe.ReplaceAllString(c, ""))) >= MinClauseWords {
			clauses++
		}
	}
	if clauses < MinClauses {
		return true, "the answer is a single clause"
	}
	return false, ""
}
