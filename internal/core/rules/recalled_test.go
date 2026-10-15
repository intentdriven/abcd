package rules

import (
	"slices"
	"testing"
)

// TestRecalledLedgerSurvivesTheRulesPrune: Inject prunes every rules-ledger
// name that is not an active domain, so the memory recall's ledger is a map
// of its own, carried through untouched on every turn — a turn that matches
// nothing included (spc-2610111627213138, "Dedup").
func TestRecalledLedgerSurvivesTheRulesPrune(t *testing.T) {
	rs := Defaults()
	prev := SessionState{
		Count:    1,
		Ledger:   map[string]string{"GONE": "0000000000000001"},
		Recalled: map[string]string{"mem-2610111200001234": "00000000000000aa"},
	}
	res := Inject(rs, "paint a landscape in oils", prev, 0)
	if _, ok := res.State.Ledger["GONE"]; ok {
		t.Fatalf("the rules ledger kept a name that is not an active domain: %v", res.State.Ledger)
	}
	if got := res.State.Recalled["mem-2610111200001234"]; got != "00000000000000aa" {
		t.Fatalf("the rules prune forgot a recalled item: recalled = %v", res.State.Recalled)
	}
	// The returned ledger is a copy: a later write to it never reaches the
	// caller's previous state.
	res.State.Recalled["mem-2610111200009999"] = "x"
	if _, ok := prev.Recalled["mem-2610111200009999"]; ok {
		t.Fatal("Inject handed back the caller's recalled map, not a copy")
	}
}

// TestResetClearsBothLedgers: the event-driven reset (SessionStart,
// PreCompact) and the fixed backstop each clear the rules ledger and the
// recalled ledger together, so a recalled item comes back after either (P3).
func TestResetClearsBothLedgers(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	const sid = "reset-both"
	st := SessionState{
		Count:    3,
		Ledger:   map[string]string{"COMMITTING": "deadbeef"},
		Recalled: map[string]string{"mem-2610111200001234": "cafe"},
	}
	if err := SaveState(sid, st); err != nil {
		t.Fatal(err)
	}
	if got := LoadState(sid); got.Recalled["mem-2610111200001234"] != "cafe" {
		t.Fatalf("the recalled ledger did not round-trip: %+v", got)
	}
	if err := ResetState(sid); err != nil {
		t.Fatal(err)
	}
	if got := LoadState(sid); len(got.Ledger) != 0 || len(got.Recalled) != 0 {
		t.Fatalf("the reset left a ledger behind: %+v", got)
	}

	// The backstop: the prompt whose count is a multiple of it clears both.
	prev := SessionState{Count: 4, Ledger: map[string]string{"COMMITTING": "x"}, Recalled: map[string]string{"mem-2610111200001234": "cafe"}}
	res := Inject(Defaults(), "paint a landscape in oils", prev, 5)
	if len(res.State.Recalled) != 0 {
		t.Fatalf("the backstop prompt kept the recalled ledger: %v", res.State.Recalled)
	}
	if slices.Contains(res.Injected, "COMMITTING") {
		t.Fatalf("a prompt about painting injected COMMITTING: %v", res.Injected)
	}
}

// TestTermMatcherIsTheDomainMatch: the exported term rule is the one Match
// runs — a term hits a prompt exactly when a domain recalling on that term
// alone matches it.
func TestTermMatcherIsTheDomainMatch(t *testing.T) {
	prompts := []string{"the tests are failing", "commit and push", "open pull requests", "attestation", "merging now", ""}
	terms := []string{"test", "push", "pull request", "commit", "merge", "attest"}
	for _, p := range prompts {
		hits := TermMatcher(p)
		for _, term := range terms {
			rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{"PROBE": {Recall: []string{term}, Rules: []string{"r"}}}}
			want := len(rs.Match(p)) == 1
			if got := hits(term); got != want {
				t.Errorf("TermMatcher(%q)(%q) = %v, but a domain recalling on it matches = %v", p, term, got, want)
			}
		}
	}
}

// TestHitsNamesTheWordsThatMatched: the diagnostic names which recall words
// and aliases hit, and a star activation as *NAME.
func TestHitsNamesTheWordsThatMatched(t *testing.T) {
	d := ResolvedDomain{Name: "WIDGETS", Domain: Domain{Recall: []string{"widget", "gadget"}, Aliases: []string{"sprocket set"}, Rules: []string{"r"}}}
	if got := Hits(d, "count the widgets and the sprocket sets"); !slices.Equal(got, []string{"widget", "sprocket set"}) {
		t.Errorf("Hits = %v", got)
	}
	if got := Hits(d, "*WIDGETS now"); !slices.Equal(got, []string{"*WIDGETS"}) {
		t.Errorf("Hits on a star activation = %v", got)
	}
}
