package rules

import (
	"slices"
	"testing"
)

// withWidgets is the bundled defaults plus one custom domain the tests can
// delete, rename or silence between turns, the way a mid-session edit to
// .abcd/rules.json does.
func withWidgets(name string, d Domain) RuleSet {
	return Merge(Defaults(), RuleSet{Domains: map[string]Domain{name: d}})
}

var widgets = Domain{Recall: []string{"widget"}, Rules: []string{"Widgets are counted twice."}}

// TestInjectActiveNamesTheFullSetEveryTurn pins ruling J15
// (iss-2608261550580260): the router names the FULL active-domain set on
// every prompt, not only the domains whose text it injected, so a turn that
// injects nothing still tells a snapshotting client what stays in force.
func TestInjectActiveNamesTheFullSetEveryTurn(t *testing.T) {
	rs := withWidgets("WIDGETS", widgets)
	var want []string
	for _, d := range rs.Active() {
		want = append(want, d.Name)
	}

	first := Inject(rs, "count the widget", SessionState{}, 0)
	if !slices.Equal(first.Injected, []string{"WIDGETS"}) {
		t.Fatalf("turn 1 injected %v, want [WIDGETS]", first.Injected)
	}
	if !slices.Equal(first.Active, want) {
		t.Fatalf("turn 1 active = %v, want the full active set %v", first.Active, want)
	}

	// A no-match turn injects no text (the zero-token promise) and still
	// names the whole set.
	second := Inject(rs, "paint a landscape", first.State, 0)
	if second.Text != "" {
		t.Fatalf("a no-match turn must render nothing, got %q", second.Text)
	}
	if !slices.Equal(second.Active, want) {
		t.Fatalf("turn 2 active = %v, want %v", second.Active, want)
	}
}

// TestInjectActiveLetsAClientSeeADomainStop is the client's view: it saw
// WIDGETS on turn 1, and from turn 2's output alone it can tell WIDGETS
// stopped, whichever way it stopped.
func TestInjectActiveLetsAClientSeeADomainStop(t *testing.T) {
	cases := map[string]RuleSet{
		"deleted": Defaults(),
		"renamed": withWidgets("GADGETS", widgets),
		"dormant": withWidgets("WIDGETS", Domain{State: StateDormant, Recall: widgets.Recall, Rules: widgets.Rules}),
		"killed":  func() RuleSet { rs := withWidgets("WIDGETS", widgets); rs.Disabled = true; return rs }(),
	}
	for how, next := range cases {
		t.Run(how, func(t *testing.T) {
			first := Inject(withWidgets("WIDGETS", widgets), "count the widget", SessionState{}, 0)
			if !slices.Contains(first.Active, "WIDGETS") {
				t.Fatalf("turn 1 active = %v, want WIDGETS in it", first.Active)
			}
			second := Inject(next, "count the widget", first.State, 0)
			if slices.Contains(second.Active, "WIDGETS") {
				t.Fatalf("turn 2 active = %v still names WIDGETS after it was %s", second.Active, how)
			}
			if how == "renamed" && !slices.Contains(second.Active, "GADGETS") {
				t.Fatalf("turn 2 active = %v, want the new name GADGETS", second.Active)
			}
			if how == "killed" && (second.Active == nil || len(second.Active) != 0) {
				t.Fatalf("kill switch: active = %#v, want an empty, non-nil set", second.Active)
			}
		})
	}
}

// TestInjectReturningDomainReinjects: a client that pruned a stopped domain
// must get its text again when it comes back, so the router forgets a domain
// the moment it leaves the active set rather than deduping its return
// against a signature the client has already thrown away.
func TestInjectReturningDomainReinjects(t *testing.T) {
	on := withWidgets("WIDGETS", widgets)
	off := withWidgets("WIDGETS", Domain{State: StateDormant, Recall: widgets.Recall, Rules: widgets.Rules})

	first := Inject(on, "count the widget", SessionState{}, 0)
	second := Inject(off, "count the widget", first.State, 0)
	third := Inject(on, "count the widget", second.State, 0)
	if !slices.Equal(third.Injected, []string{"WIDGETS"}) {
		t.Fatalf("a domain back in force was deduped against its old signature: injected %v", third.Injected)
	}
}

// TestInjectActiveCarriesAnExplicitlyActivatedDormantDomain: `*NAME` injects a
// dormant domain for the prompt that names it, so that turn's set names it
// too — a set that omitted a domain injected on the same turn would tell the
// client to prune what it was just handed.
func TestInjectActiveCarriesAnExplicitlyActivatedDormantDomain(t *testing.T) {
	rs := withWidgets("WIDGETS", Domain{State: StateDormant, Recall: widgets.Recall, Rules: widgets.Rules})
	res := Inject(rs, "*WIDGETS please", SessionState{}, 0)
	if !slices.Equal(res.Injected, []string{"WIDGETS"}) {
		t.Fatalf("*WIDGETS injected %v", res.Injected)
	}
	if !slices.Contains(res.Active, "WIDGETS") {
		t.Fatalf("active = %v omits the domain injected this turn", res.Active)
	}
	next := Inject(rs, "carry on", res.State, 0)
	if slices.Contains(next.Active, "WIDGETS") {
		t.Fatalf("active = %v still names a dormant domain no prompt activated", next.Active)
	}
}
