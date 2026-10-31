package source

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestSourceQuestionPassesTheLimitsAtTheFloors holds the class question to the
// floor of every configurable key (itd-2610301914350541 decision 3): its
// two-sentence descriptions are the floor of description sentences, and it is
// admitted whole there as at the defaults. It ran 26 rows until its lead and
// its change-later line were shortened to fit the rows floor, 24. Its
// "Library" chip is not in abcd's chip grammar, so the hook leaves the
// question unchecked, and the header rule is the one it is not held to.
func TestSourceQuestionPassesTheLimitsAtTheFloors(t *testing.T) {
	a := question.Ask{Questions: []question.Question{ClassQuestion()}}
	for name, l := range map[string]question.Limits{"defaults": question.Default, "floors": question.Floors(question.Default)} {
		for _, f := range question.CheckLimits(a.Fields(), l, question.Addressee{}) {
			if f.Rule != question.RuleHeader {
				t.Errorf("%s: %s", name, f)
			}
		}
	}
}
