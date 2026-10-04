package ahoy

import (
	"sort"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// heldQuestion is the value question h as setup builds it (valueQuestion, the
// builder every front door uses), offering every choice the help names. A
// question a flag answers is held with no default, as the install asks it;
// any other with its last choice standing in for the default, so the
// decide-later meaning that names a default is held too.
func heldQuestion(n int, h PromptHelp) question.Question {
	choices := make([]string, len(h.Choices))
	for i, c := range h.Choices {
		choices[i] = c.Value
	}
	def := ""
	if h.Flag == "" && len(choices) > 0 {
		def = choices[len(choices)-1]
	}
	return valueQuestion(n, h, choices, def)
}

// confirmQuestion is the approval whose text ends on its question, tail, as
// setup builds it (SetupConfirmQuestion): the text before the tail is the
// material (an indented line an item of a list), the tail the question, its
// id the one given, and the answers yes, no and decide later.
func confirmQuestion(t *testing.T, n int, id, text, tail string) question.Question {
	t.Helper()
	q := SetupConfirmQuestion(n, text)
	if q.Ask != tail {
		t.Fatalf("%s: the question is %q, not its tail %q", id, q.Ask, tail)
	}
	if q.ID != id {
		t.Fatalf("%s: setup ids the question %q", id, q.ID)
	}
	return q
}

// setupLimitsOwed is every limit a setup question breaks today, as
// "<question id> <rule>". It is empty: iss-2610031236155833's questions all fit
// since itd-2610030814013772 retired claude_md and both (docs_target) and the
// 2026-10-03 fit (artefact_kind, visibility, the machine routing offer).
// oracle_backend is not asked while one answer has an adapter (the 2026-10-03
// ruling), so it owes nothing while it stays unasked. The list may only shrink: a
// question that newly breaks a limit fails, and so does a line here that no
// longer breaks, so the fix deletes its line.
var setupLimitsOwed = map[string]bool{}

// TestEverySetupQuestionPassesTheLimits holds every fixed question the install
// builds, the value questions (every PromptHelp, the status line's elements
// included) and the routing confirmation's two offers, to question.CheckLimits
// through their field view (spc-2610030944505997 step 4: "every fixed question
// abcd builds (setup, routing) passing CheckLimits"). The words are core's, so
// a change to one that breaks a limit fails here, not in front of a person.
// What breaks a limit today is owed by name in setupLimitsOwed. A question the
// install does not ask is not held to them.
func TestEverySetupQuestionPassesTheLimits(t *testing.T) {
	var helps []PromptHelp
	for _, h := range promptHelp {
		if h.Key == "oracle_backend" && !oracleBackendAsked() {
			continue // not asked while one answer has an adapter; it returns with a second
		}
		helps = append(helps, h)
	}
	for k := range statusLineElementAbout {
		h, ok := helpFor(elementPromptPrefix + string(k))
		if !ok {
			t.Fatalf("no help for the status-line element %s", k)
		}
		helps = append(helps, h)
	}
	sort.Slice(helps, func(i, j int) bool { return helps[i].Key < helps[j].Key })
	// The visibility question is held twice: as most repositories see it, and
	// as a repository whose .abcd/ holds tracked records sees it, with public's
	// caveat, its tallest form, labelled so a finding names which form broke.
	tracked := visibilityHelp(true)
	tracked.Key = "visibility (tracked)"
	helps = append(helps, tracked)

	var qs []question.Question
	for i, h := range helps {
		qs = append(qs, heldQuestion(i+1, h))
	}
	qs = append(qs,
		confirmQuestion(t, len(qs)+1, OracleRoutingMachineGapID, machineRoutingQuestion(), oracleRoutingMachineQuestionTail),
		confirmQuestion(t, len(qs)+2, OracleRoutingRepoGapID, repoRoutingQuestion(), oracleRoutingRepoQuestionTail))

	broken := map[string]bool{}
	for _, q := range qs {
		a := question.Ask{Questions: []question.Question{q}}
		for _, f := range question.Check(a) {
			t.Errorf("%s: structural: %s", q.ID, f)
		}
		for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
			key := q.ID + " " + string(f.Rule)
			broken[key] = true
			if !setupLimitsOwed[key] {
				t.Errorf("%s: %s", q.ID, f)
			}
		}
	}
	for key := range setupLimitsOwed {
		if !broken[key] {
			t.Errorf("%q is owed in setupLimitsOwed but the question no longer breaks it; delete its line", key)
		}
	}
}
