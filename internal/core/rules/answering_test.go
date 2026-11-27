package rules

import (
	"slices"
	"strings"
	"testing"
)

// answeringDomain is the bundled domain that has a question about the
// project's state answered from the record read in that turn
// (iss-2611260623136583).
const answeringDomain = "ANSWERING"

// TestAnsweringDomainRecallsQuestionsAboutTheProject pins iss-2611260623136583:
// a question about where the project stands, whether something is possible,
// planned or already built, or who is working on what, recalls the domain, and
// its rule has the answer read from the record that turn, each claim citing
// what it rests on, and a claim the record does not settle said to be
// unsettled. Before it, such a prompt injected nothing, and an agent answered
// from memory.
func TestAnsweringDomainRecallsQuestionsAboutTheProject(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"where are we in terms of running a session entirely through the CLI?",
		"Where are we with the drain?",
		"is that already possible?",
		"is it planned, or built already?",
		"has this been built yet?",
		"is anyone working on the release?",
		"who is building the dashboard?",
		"what's the plan forward?",
		"status? what's being built?",
		"what's the status of the drain?",
		"what is the state of the reading work?",
	} {
		if got := rs.Match(prompt); !has(got, answeringDomain) {
			t.Errorf("%q does not recall %s: %v", prompt, answeringDomain, names(got))
		}
	}
	d, ok := rs.Domains[answeringDomain]
	if !ok {
		t.Fatalf("no bundled %s domain", answeringDomain)
	}
	if len(d.Rules) == 0 {
		t.Fatalf("%s has no rules", answeringDomain)
	}
	rule := d.Rules[0]
	for _, want := range []string{
		"read in this turn",
		"never from memory",
		"abcd status",
		"abcd <id>",
		"abcd peers",
		"Cite the record id",
		"unsettled",
	} {
		if !strings.Contains(rule, want) {
			t.Errorf("the %s rule misses %q:\n%s", answeringDomain, want, rule)
		}
	}
	res := Inject(rs, "where are we with the drain?", SessionState{}, 0, BracketUnknown)
	if !slices.Contains(res.Injected, answeringDomain) {
		t.Fatalf("a question about the project injected %v, want %s in it", res.Injected, answeringDomain)
	}
}

// TestAnsweringDomainIgnoresWorkPrompts holds recall to questions about the
// project: an instruction to do work recalls nothing of it.
func TestAnsweringDomainIgnoresWorkPrompts(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"fix the flaky test",
		"run the tests for the parser",
		"rename hook_away.go",
		"add a flag to the capture verb",
		"commit and push",
		"keep working on the parser",
		"run abcd status",
		"save the state of the run",
	} {
		if got := rs.Match(prompt); has(got, answeringDomain) {
			t.Errorf("%q recalls %s", prompt, answeringDomain)
		}
	}
}
