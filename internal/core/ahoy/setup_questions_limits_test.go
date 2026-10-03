package ahoy

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// setupLater is the decide-later answer every setup question offers: it
// leaves the value unset, so the gap stays listed (spc-2610030911534855, "The
// interviews whose questions abcd fixes").
var setupLater = question.Option{
	Value:   "later",
	Label:   question.Default.LaterLabels[0],
	Meaning: "Leaves the value unset, so the install lists it again next time.",
}

// valueQuestion is the field view of one value question, built the way the
// plain-Terminal spec maps PromptHelp onto the shared question type: the key
// as the id, About as the material, each choice as an option whose label is
// the value and whose description is its meaning, the flag hint as the
// change-later line, and the decide-later answer last.
func valueQuestion(n int, h PromptHelp) question.Question {
	opts := make([]question.Option, len(h.Choices))
	for i, c := range h.Choices {
		opts[i] = question.Option{Value: c.Value, Label: c.Value, Meaning: c.Meaning}
	}
	change := h.FlagHint()
	if change == "" {
		change = question.Default.NotApplicable
	}
	return question.Question{
		ID:          h.Key,
		Chip:        fmt.Sprintf("Setup Q%d", n),
		Material:    []question.Block{{Kind: question.KindParagraph, Text: h.About}},
		Ask:         "Which answer does this repository take?",
		Options:     opts,
		Later:       setupLater,
		Now:         "not set",
		ChangeLater: change,
	}
}

// confirmQuestion is the field view of one approval whose text ends on its
// question, tail: the text before it is the material (an indented line an
// item of a list, as the routing table is), the tail the question, and the
// answers yes, no and decide later.
func confirmQuestion(t *testing.T, n int, id, text, tail string) question.Question {
	t.Helper()
	material, ok := strings.CutSuffix(text, tail)
	if !ok {
		t.Fatalf("%s: the question does not end on %q", id, tail)
	}
	ask := tail
	var blocks []question.Block
	var items []string
	flush := func() {
		if len(items) > 0 {
			blocks = append(blocks, question.Block{Kind: question.KindList, Items: items})
			items = nil
		}
	}
	for _, ln := range strings.Split(material, "\n") {
		if strings.HasPrefix(ln, "  ") {
			items = append(items, strings.TrimSpace(ln))
			continue
		}
		flush()
		if strings.TrimSpace(ln) != "" {
			blocks = append(blocks, question.Block{Kind: question.KindParagraph, Text: ln})
		}
	}
	flush()
	return question.Question{
		ID:       id,
		Chip:     fmt.Sprintf("Setup Q%d", n),
		Material: blocks,
		Ask:      ask,
		Options: []question.Option{
			{Value: "yes", Label: "Yes, make the change", Meaning: "Writes what the text above describes."},
			{Value: "no", Label: "No, leave it", Meaning: "Writes nothing, so the next install asks again."},
		},
		Later:       setupLater,
		Now:         question.Default.NotApplicable,
		ChangeLater: question.Default.NotApplicable,
	}
}

// setupLimitsOwed is every limit a setup question breaks today, as
// "<question id> <rule>", recorded in iss-2610031236155833 (docs_target fits
// since itd-2610030814013772 retired claude_md and both): each exceeds the rows
// under the host figures calibrated on 2026-10-03 (step 5), each still taller
// than its copy can be cut to without losing what an answer means.
// oracle_backend is not asked while one answer has an adapter (the 2026-10-03
// ruling), so it owes nothing while it stays unasked. The list may only shrink: a
// question that newly breaks a limit fails, and so does a line here that no
// longer breaks, so the fix deletes its line.
var setupLimitsOwed = map[string]bool{
	"artefact_kind rows":                  true,
	"oracle_routing.machine_offered rows": true,
	"visibility rows":                     true,
}

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
		h, ok := HelpFor(elementPromptPrefix + string(k))
		if !ok {
			t.Fatalf("no help for the status-line element %s", k)
		}
		helps = append(helps, h)
	}
	sort.Slice(helps, func(i, j int) bool { return helps[i].Key < helps[j].Key })

	var qs []question.Question
	for i, h := range helps {
		qs = append(qs, valueQuestion(i+1, h))
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
