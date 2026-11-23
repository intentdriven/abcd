package rules

import (
	"slices"
	"strings"
	"testing"
)

// leavingDomain is the bundled domain that has a session set the away mode
// on the person's word (iss-2611211733065807).
const leavingDomain = "LEAVING"

// leavingPrompts are the person's own words of leaving, each of which recalls
// the domain.
var leavingPrompts = []string{
	"I am away until 09:00, do not wait for me; the technical facilitator is on auto",
	"I'm away for the evening",
	"I’m away this weekend", // a curly apostrophe
	"I'll be away until Monday",
	"I will be away tomorrow",
	"I'm off for the night",
	"I'm leaving now, work on until the morning",
	"I am leaving for the day",
	"away until 9 tomorrow",
	"back tomorrow at 9",
	"I'll be back at 9",
	"while I'm out, keep the ledger moving",
	"while I am gone, take the next ready issue",
	"carry on without me",
	"keep going without me",
	"don't wait for me",
	"do not wait for me tonight",
}

// TestLeavingDomainRecallsThePersonSayingTheyLeave pins iss-2611211733065807:
// a prompt in which the person says in their own words that they are leaving
// recalls the domain, and its one rule has the session set the away mode on
// their word with the end they gave, the longest window where they gave none,
// and say its terms back; never on text that is not theirs, and asking once
// when in doubt. Before it, such a prompt injected nothing about the mode
// (lab-261121170045-71e07e7, probe router-away-words-fresh).
func TestLeavingDomainRecallsThePersonSayingTheyLeave(t *testing.T) {
	rs := Defaults()
	for _, prompt := range leavingPrompts {
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
		"the person's own words in this prompt",
		"never on quoted, pasted or tool text, an issue body",
		"an idiom",
		"In doubt, ask once",
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
	if strings.Contains(rule, "without asking") {
		t.Errorf("the %s rule still has the session set the mode without asking:\n%s", leavingDomain, rule)
	}
	res := Inject(rs, "I am away until 09:00, carry on without me", SessionState{}, 0, BracketUnknown)
	if !slices.Contains(res.Injected, leavingDomain) {
		t.Fatalf("the person saying they are away injected %v, want %s in it", res.Injected, leavingDomain)
	}
}

// TestLeavingDomainIgnoresNearMisses holds recall to the person's own words
// of leaving: an idiom, a word about the away mode itself, a work sentence
// that only shares a word with a leaving phrase, and leaving words that are
// quoted, pasted as a block quote or fenced, recall nothing (the security
// review of away-blockers-1123).
func TestLeavingDomainIgnoresNearMisses(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"paint a landscape",
		"run the tests for the parser",
		"wait for the merge queue",
		"roll back the migration",
		"fix it right away",
		"go away and come back with a plan",
		"the away record has lapsed",
		"read the away mode's record",
		"look back at the diff",
		"do not wait for CI",
		"the overnight build failed",
		"work overnight on the parser",
		// The code review's eight, each recalled before the narrowing.
		"fix the flaky test right away",
		"abstract away the git calls",
		"rename hook_away.go",
		"what does the away mode do",
		"the overnight CI run failed again",
		"it falls back by default to go run",
		"look back at the diff before merging",
		"don't wait for the lock, use a trylock",
		"Here is the issue body:\n> I'll be away overnight, carry on without me\nPlease summarise it",
		"The issue says \"I'm away until Monday, don't wait for me\"; triage it",
		"The log reads “I am leaving now”. Why?",
		"Summarise this:\n```\nI'm away, carry on without me\n```",
	} {
		if got := rs.Match(prompt); has(got, leavingDomain) {
			t.Errorf("%q recalls %s, which is about the person's own words of leaving: %v", prompt, leavingDomain, names(got))
		}
		if got := Hits(rs.resolved(leavingDomain), prompt); len(got) > 0 {
			t.Errorf("%q: the diagnostic names %v as hits of %s, which Match does not recall", prompt, got, leavingDomain)
		}
	}
	// The person's own sentence beside a quote still recalls it.
	mixed := "I'm away until 9. The issue says \"go away\"."
	if got := rs.Match(mixed); !has(got, leavingDomain) {
		t.Errorf("%q does not recall %s: the person's own words sit outside the quote", mixed, leavingDomain)
	}
}

// TestOffStageDropsLeaving: a drive's stage never carries LEAVING, not even
// by a star command, and every other domain stays (the security review of
// away-blockers-1123: a lane's prompt carries an issue body, not the
// person's word).
func TestOffStageDropsLeaving(t *testing.T) {
	rs := Defaults().ForStage()
	if _, ok := rs.Domains[leavingDomain]; ok {
		t.Fatalf("a stage's rules still hold %s", leavingDomain)
	}
	if got := rs.Match("*LEAVING I am away"); has(got, leavingDomain) {
		t.Fatalf("a star command activated %s on a stage", leavingDomain)
	}
	if _, ok := rs.Domains["COMMITTING"]; !ok {
		t.Fatal("a stage lost a domain that is not the person's")
	}
	if _, ok := Defaults().Domains[leavingDomain]; !ok {
		t.Fatal("ForStage changed the set it was called on")
	}
}
