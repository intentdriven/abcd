package matrix

import (
	"strings"
	"testing"
)

// Each reviewer is scored on its own, never by vote, against copies whose
// truth is known (adr-2610161733400551): copies the hidden tests pass, copies
// they fail and copies with a defect seeded on purpose, so recall and the
// true-negative rate both have a denominator. Its guess at each copy's author
// is scored as the measured leak, and verdicts on a copy its own family wrote
// are reported apart.

func reviewPins() []CellPin {
	return []CellPin{
		{Cell: "gpt", Role: RoleBuilder, Model: "openai/gpt-5.1"},
		{Cell: "sonnet", Role: RoleBuilder, Model: "claude-sonnet-5"},
		{Cell: "opus-rev", Role: RoleReviewer, Model: "claude-opus-5-5"},
		{Cell: "qwen-rev", Role: RoleReviewer, Model: "qwen/qwen3-coder"},
	}
}

func reviewCopies() []ReviewedCopy {
	key := func(cell string, n int) TrialKey { return TrialKey{Task: "widget", Cell: cell, Repeat: n} }
	return []ReviewedCopy{
		{Label: "copy-00000001", Key: key("gpt", 1), Truth: TruthPassing},
		{Label: "copy-00000002", Key: key("gpt", 2), Truth: TruthPassing},
		{Label: "copy-00000003", Key: key("gpt", 3), Truth: TruthFailing},
		{Label: "copy-00000004", Key: key("gpt", 1), Truth: TruthSeeded},
		{Label: "copy-00000005", Key: key("sonnet", 1), Truth: TruthPassing},
		{Label: "copy-00000006", Key: key("sonnet", 2), Truth: TruthFailing},
		{Label: "copy-00000007", Key: key("sonnet", 1), Truth: TruthSeeded},
	}
}

func verdict(reviewer, label string, defect bool, guess string) ReviewVerdict {
	return ReviewVerdict{Reviewer: reviewer, Label: label, Defective: defect, AuthorGuess: guess}
}

func scoreOf(t *testing.T, scores []ReviewerScore, cell string) ReviewerScore {
	t.Helper()
	for _, s := range scores {
		if s.Reviewer == cell {
			return s
		}
	}
	t.Fatalf("no score for %s in %+v", cell, scores)
	return ReviewerScore{}
}

// TestRecallAndTrueNegativeRateOnKnownGoodAndSeededDefectCopies: the step's
// second test.
func TestRecallAndTrueNegativeRateOnKnownGoodAndSeededDefectCopies(t *testing.T) {
	scores, err := scoreReviewers(reviewCopies(), reviewPins(), []ReviewVerdict{
		// qwen reviews only gpt's copies here, a family not its own.
		verdict("qwen-rev", "copy-00000001", true, ""),  // a false alarm on a passing copy
		verdict("qwen-rev", "copy-00000002", false, ""), // a passing copy cleared
		verdict("qwen-rev", "copy-00000003", true, ""),  // a failing copy caught
		verdict("qwen-rev", "copy-00000004", false, ""), // a seeded defect missed
	})
	if err != nil {
		t.Fatal(err)
	}
	q := scoreOf(t, scores, "qwen-rev")
	if q.Model != "qwen/qwen3-coder" {
		t.Fatalf("the score names the reviewer's model: %q", q.Model)
	}
	o := q.OtherFamily
	if o.Recall != (Rate{Hits: 1, Of: 2}) || o.RecallFailing != (Rate{Hits: 1, Of: 1}) ||
		o.RecallSeeded != (Rate{Hits: 0, Of: 1}) || o.TrueNegative != (Rate{Hits: 1, Of: 2}) {
		t.Fatalf("qwen's score on other families' copies: %+v", o)
	}
	if q.Unanswered != 3 {
		t.Fatalf("the three copies qwen gave no verdict on are counted apart, not as misses: %d", q.Unanswered)
	}
	// A reviewer with no verdict has rates with no denominator, never zero.
	opus := scoreOf(t, scores, "opus-rev")
	if opus.OtherFamily.Recall.Estimable() || opus.OtherFamily.TrueNegative.Estimable() {
		t.Fatalf("a rate with no copy under it is not estimable: %+v", opus.OtherFamily)
	}
	if !o.Recall.Estimable() {
		t.Fatal("a rate with copies under it is estimable")
	}
}

// TestSameFamilyVerdictsAreReportedApart: the step's fourth test. A model's
// family is its developer, so an Anthropic reviewer's verdicts on an
// Anthropic builder's copies are kept out of its main score.
func TestSameFamilyVerdictsAreReportedApart(t *testing.T) {
	var vs []ReviewVerdict
	for _, c := range reviewCopies() {
		vs = append(vs, verdict("opus-rev", c.Label, c.Truth != TruthPassing, ""))
	}
	scores, err := scoreReviewers(reviewCopies(), reviewPins(), vs)
	if err != nil {
		t.Fatal(err)
	}
	opus := scoreOf(t, scores, "opus-rev")
	if opus.OtherFamily.Recall != (Rate{Hits: 2, Of: 2}) || opus.OtherFamily.TrueNegative != (Rate{Hits: 2, Of: 2}) {
		t.Fatalf("opus's verdicts on gpt's copies: %+v", opus.OtherFamily)
	}
	if opus.SameFamily.Recall != (Rate{Hits: 2, Of: 2}) || opus.SameFamily.TrueNegative != (Rate{Hits: 1, Of: 1}) {
		t.Fatalf("opus's verdicts on sonnet's copies are its same-family score: %+v", opus.SameFamily)
	}
	if opus.UnknownFamily.Recall.Of+opus.UnknownFamily.TrueNegative.Of != 0 {
		t.Fatalf("no copy here has an unknown family: %+v", opus.UnknownFamily)
	}
}

// A copy whose builder's family, or the reviewer's, cannot be read is neither
// the same family nor another: it is reported apart again, so an unread name
// never dilutes either score.
func TestAnUnreadFamilyIsReportedApartFromBoth(t *testing.T) {
	pins := append(reviewPins(), CellPin{Cell: "mystery", Role: RoleBuilder, Model: "default"})
	copies := []ReviewedCopy{{Label: "copy-0000000a", Key: TrialKey{Task: "widget", Cell: "mystery", Repeat: 1}, Truth: TruthFailing}}
	scores, err := scoreReviewers(copies, pins, []ReviewVerdict{verdict("opus-rev", "copy-0000000a", true, "")})
	if err != nil {
		t.Fatal(err)
	}
	opus := scoreOf(t, scores, "opus-rev")
	if opus.UnknownFamily.Recall != (Rate{Hits: 1, Of: 1}) || opus.OtherFamily.Recall.Of != 0 || opus.SameFamily.Recall.Of != 0 {
		t.Fatalf("an unread family: %+v", opus)
	}
}

// A guess at the author of a copy whose builder's name cannot be read, or
// names no version, can never be a hit, so counting it would understate the
// leak: it is kept out of the rate it cannot be scored in and counted apart.
func TestAGuessOnAnUnreadBuilderIsKeptOutOfTheLeakRates(t *testing.T) {
	pins := append(reviewPins(),
		CellPin{Cell: "mystery", Role: RoleBuilder, Model: "default"},
		CellPin{Cell: "unversioned", Role: RoleBuilder, Model: "claude-opus"})
	key := func(cell string) TrialKey { return TrialKey{Task: "widget", Cell: cell, Repeat: 1} }
	copies := []ReviewedCopy{
		{Label: "copy-00000001", Key: key("gpt"), Truth: TruthPassing},
		{Label: "copy-0000000a", Key: key("mystery"), Truth: TruthFailing},
		{Label: "copy-0000000b", Key: key("mystery"), Truth: TruthPassing},
		{Label: "copy-0000000c", Key: key("unversioned"), Truth: TruthPassing},
	}
	scores, err := scoreReviewers(copies, pins, []ReviewVerdict{
		verdict("qwen-rev", "copy-00000001", false, "gpt-5.1"),         // the model and the developer
		verdict("qwen-rev", "copy-0000000a", true, "claude-opus-5-5"),  // no builder name to score against
		verdict("qwen-rev", "copy-0000000b", false, "claude-opus-5-5"), // nor here
		verdict("qwen-rev", "copy-0000000c", false, "claude-opus-5-5"), // the developer; the builder has no version
	})
	if err != nil {
		t.Fatal(err)
	}
	g := scoreOf(t, scores, "qwen-rev").AuthorGuess
	if g.Asked != 4 || g.Guessed != 4 {
		t.Fatalf("four asked, four guessed: %+v", g)
	}
	if g.Model != (Rate{Hits: 1, Of: 1}) || g.ModelUnscorable != 3 {
		t.Fatalf("only the gpt copy's guess can name the model; three are counted apart: %+v", g)
	}
	if g.Developer != (Rate{Hits: 2, Of: 2}) || g.DeveloperUnscorable != 2 {
		t.Fatalf("the mystery copies' guesses are counted apart from the developer rate: %+v", g)
	}
}

// TestTheAuthorGuessIsScored: the step's third test. The guess is the measured
// leak: how often the reviewer named the builder's model, how often its
// family, and how often it declined.
func TestTheAuthorGuessIsScored(t *testing.T) {
	scores, err := scoreReviewers(reviewCopies(), reviewPins(), []ReviewVerdict{
		verdict("qwen-rev", "copy-00000001", false, "gpt-5.1"),         // the model
		verdict("qwen-rev", "copy-00000002", false, "openai/gpt-5"),    // the family, another version
		verdict("qwen-rev", "copy-00000003", true, "o3"),               // the developer, another family
		verdict("qwen-rev", "copy-00000005", false, "claude-opus-5-5"), // the developer, not the model
		verdict("qwen-rev", "copy-00000006", true, "gemini-3"),         // wrong
		verdict("qwen-rev", "copy-00000007", true, ""),                 // declined
	})
	if err != nil {
		t.Fatal(err)
	}
	g := scoreOf(t, scores, "qwen-rev").AuthorGuess
	if g.Asked != 6 || g.Guessed != 5 {
		t.Fatalf("six asked, five guessed: %+v", g)
	}
	if g.Model != (Rate{Hits: 1, Of: 5}) {
		t.Fatalf("one guess named the builder's model: %+v", g.Model)
	}
	if g.Developer != (Rate{Hits: 4, Of: 5}) {
		t.Fatalf("four guesses named the builder's developer: %+v", g.Developer)
	}
}

// Verdicts are a reviewer model's output, so they are untrusted: one naming
// a copy or a reviewer the run does not hold, or a second verdict on one
// copy, refuses the whole set.
func TestAVerdictSetThatDoesNotValidateIsRefusedWhole(t *testing.T) {
	for name, vs := range map[string][]ReviewVerdict{
		"unknown copy":     {verdict("qwen-rev", "copy-ffffffff", true, "")},
		"unknown reviewer": {verdict("gpt", "copy-00000001", true, "")},
		"twice":            {verdict("qwen-rev", "copy-00000001", true, ""), verdict("qwen-rev", "copy-00000001", false, "")},
		"long guess":       {verdict("qwen-rev", "copy-00000001", true, strings.Repeat("g", 300))},
		"control guess":    {verdict("qwen-rev", "copy-00000001", true, "gpt\x1b[31m")},
	} {
		if _, err := scoreReviewers(reviewCopies(), reviewPins(), vs); err == nil {
			t.Errorf("%s: the verdict set was scored", name)
		}
	}
	dup := append(reviewCopies(), ReviewedCopy{Label: "copy-00000001", Key: TrialKey{Task: "widget", Cell: "gpt", Repeat: 9}, Truth: TruthPassing})
	if _, err := scoreReviewers(dup, reviewPins(), nil); err == nil {
		t.Error("two copies under one label were scored")
	}
	unpinned := []ReviewedCopy{{Label: "copy-0000000b", Key: TrialKey{Task: "widget", Cell: "nobody", Repeat: 1}, Truth: TruthPassing}}
	if _, err := scoreReviewers(unpinned, reviewPins(), nil); err == nil {
		t.Error("a copy no builder cell made was scored")
	}
}

// TestAReviewersAnswerIsReadAsUntrustedInput: the answer a reviewer model
// returns is decoded strictly and bounded, and must answer the label it was
// asked about.
func TestAReviewersAnswerIsReadAsUntrustedInput(t *testing.T) {
	ok := `{"label":"copy-0a1b2c3d","verdict":"defect","findings":[{"item":"item-001","text":"off by one"}],"author_guess":"gpt-5"}`
	v, err := parseReviewAnswer([]byte(ok), "opus-rev", "copy-0a1b2c3d")
	if err != nil {
		t.Fatal(err)
	}
	if v != (ReviewVerdict{Reviewer: "opus-rev", Label: "copy-0a1b2c3d", Defective: true, AuthorGuess: "gpt-5"}) {
		t.Fatalf("verdict: %+v", v)
	}
	if v, err := parseReviewAnswer([]byte(`{"label":"copy-0a1b2c3d","verdict":"clean","findings":[],"author_guess":""}`), "opus-rev", "copy-0a1b2c3d"); err != nil || v.Defective {
		t.Fatalf("a clean answer: %+v %v", v, err)
	}
	for name, bad := range map[string]string{
		"other label":   `{"label":"copy-ffffffff","verdict":"defect","findings":[],"author_guess":""}`,
		"unknown key":   `{"label":"copy-0a1b2c3d","verdict":"defect","findings":[],"author_guess":"","author":"x"}`,
		"bad verdict":   `{"label":"copy-0a1b2c3d","verdict":"maybe","findings":[],"author_guess":""}`,
		"trailing data": `{"label":"copy-0a1b2c3d","verdict":"defect","findings":[],"author_guess":""} {}`,
		"not json":      `defect`,
		"huge":          `{"label":"copy-0a1b2c3d","verdict":"defect","findings":[{"item":"item-001","text":"` + strings.Repeat("x", maxReviewAnswerBytes) + `"}],"author_guess":""}`,
	} {
		if _, err := parseReviewAnswer([]byte(bad), "opus-rev", "copy-0a1b2c3d"); err == nil {
			t.Errorf("%s: the answer was read", name)
		}
	}
}

// Every family the model-name reader knows has a developer, so a family
// added there without one is caught here rather than read as unknown.
func TestEveryModelFamilyHasADeveloper(t *testing.T) {
	for _, f := range append(append([]string(nil), modelFamilies...), oSeries) {
		if developers[f] == "" {
			t.Errorf("family %q has no developer", f)
		}
	}
}
