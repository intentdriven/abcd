package rules

import (
	"slices"
	"strings"
	"testing"
)

// leavingDomain is the bundled domain that has a session set the away mode
// on the person's word (iss-2611211733065807).
const leavingDomain = "LEAVING"

// TestLeavingDomainRecallsThePersonSayingTheyLeave pins iss-2611211733065807:
// a prompt in which the person says they are away, back at a time, gone
// overnight, that the session should not wait or should carry on without
// them, recalls the domain, and its one rule has the session set the away
// mode on their word with the end they gave, the longest window where they
// gave none, and say its terms back. Before it, such a prompt injected
// nothing about the mode (lab-261121170045-71e07e7, probe
// router-away-words-fresh).
func TestLeavingDomainRecallsThePersonSayingTheyLeave(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"I am away until 09:00, do not wait for me; the technical facilitator is on auto",
		"I'm away for the evening",
		"back at 9 tomorrow",
		"work through it overnight",
		"don't wait for me",
		"carry on without me",
	} {
		if got := rs.Match(prompt); !has(got, leavingDomain) {
			t.Errorf("%q does not recall %s: %v", prompt, leavingDomain, names(got))
		}
	}
	d, ok := rs.Domains[leavingDomain]
	if !ok {
		t.Fatalf("no bundled %s domain", leavingDomain)
	}
	if len(d.Rules) != 1 {
		t.Fatalf("%s has %d rules, want one", leavingDomain, len(d.Rules))
	}
	rule := d.Rules[0]
	for _, want := range []string{
		"on the person's word",
		"abcd mode autonomous --until",
		"longest window the mode allows",
		"say so",
		"terms",
		"grants nothing",
		"abcd mode back",
	} {
		if !strings.Contains(rule, want) {
			t.Errorf("the %s rule misses %q:\n%s", leavingDomain, want, rule)
		}
	}
	res := Inject(rs, "I am away until 09:00, carry on without me", SessionState{}, 0, BracketUnknown)
	if !slices.Contains(res.Injected, leavingDomain) {
		t.Fatalf("the person saying they are away injected %v, want %s in it", res.Injected, leavingDomain)
	}
}

// TestLeavingDomainIgnoresNearMisses holds recall to the person leaving: a
// prompt about the work alone does not inject the domain.
func TestLeavingDomainIgnoresNearMisses(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"paint a landscape",
		"run the tests for the parser",
		"wait for the merge queue",
		"roll back the migration",
	} {
		if got := rs.Match(prompt); has(got, leavingDomain) {
			t.Errorf("%q recalls %s, which is about the person leaving: %v", prompt, leavingDomain, names(got))
		}
	}
}
