package interview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestProductChipTasksSayWhatToDoWithInternals (iss-2611252121330106): a
// question under a Product chip is held to the product thinker's knowledge
// floor, so both AI-written interviews' tasks, and the planning interviewer's
// page, say what a Product chip does with material in internals: restate it in
// plain words, dropping nothing an answer turns on; introduce a word that
// cannot be avoided; and put text that is mechanism alone to the facilitator
// under a Tech chip.
func TestProductChipTasksSayWhatToDoWithInternals(t *testing.T) {
	for name, task := range map[string]string{"planning": PlanningTask, "retrospective": RetrospectiveTask} {
		if !strings.Contains(task, ProductChipInternals) {
			t.Errorf("the %s task does not carry the Product chip's rule %q", name, ProductChipInternals)
		}
	}
	for _, want := range []string{"restate it in plain words", "dropping nothing an answer turns on", "introduce it in one sentence", "under a Tech chip"} {
		if !strings.Contains(ProductChipInternals, want) {
			t.Errorf("the Product chip's rule does not say %q: %s", want, ProductChipInternals)
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "agents", "planning-interviewer.md"))
	if err != nil {
		t.Fatal(err)
	}
	if page := strings.Join(strings.Fields(string(raw)), " "); !strings.Contains(page, ProductChipInternals) {
		t.Errorf("agents/planning-interviewer.md does not carry the Product chip's rule %q", ProductChipInternals)
	}
	if m := sampleQuestion(1).Material[0].Text; !strings.Contains(m, "in the person's words") {
		t.Errorf("the brief's sample question does not quote the material in the person's words: %q", m)
	}
}

// TestBriefListsTheWordsBelowTheFloor: a runner in a plain Terminal reads no
// plugin page, so the brief carries the words below the product thinker's
// floor itself, each with its other forms and its plain words.
func TestBriefListsTheWordsBelowTheFloor(t *testing.T) {
	brief := newWrittenRun(t, "").w.brief(1, 0, nil, carried{})
	if !strings.Contains(brief, "## The words below the product thinker's floor") {
		t.Fatalf("the brief carries no list of the words below the floor:\n%s", brief)
	}
	for _, w := range question.BelowFloor() {
		line := "- " + w.Word
		if !strings.Contains(brief, line) || !strings.Contains(brief, w.Plain) {
			t.Errorf("the brief does not list %q with its plain words %q", w.Word, w.Plain)
		}
		for _, f := range append(w.Also, w.Short...) {
			if !strings.Contains(brief, f) {
				t.Errorf("the brief does not list %q's form %q", w.Word, f)
			}
		}
	}
}
