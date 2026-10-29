package oracle

import (
	"strings"
	"testing"
)

// TestEscalateMovesOneStepUpTheLadder is spc-2609251028149555's escalation
// ladder (AC 10, the intent's Decision 1): local moves to economy, economy to
// frontier, and the two fixed points, frontier and host-decides, do not move.
func TestEscalateMovesOneStepUpTheLadder(t *testing.T) {
	for _, c := range []struct{ from, want Tier }{
		{Local, Economy},
		{Economy, Frontier},
		{Frontier, Frontier},
		{HostDecides, HostDecides},
	} {
		if got := Escalate(c.from); got != c.want {
			t.Errorf("Escalate(%s) = %s, want %s", c.from, got, c.want)
		}
	}
	// Every member of the vocabulary has a row above, so a tier added to it
	// without a place on the ladder fails here.
	if len(Tiers()) != 4 {
		t.Fatalf("the vocabulary is %v; give a new tier its place on the ladder and in this table", Tiers())
	}
	// One step, never two: escalating twice from local is frontier, and
	// escalating once is not.
	if Escalate(Escalate(Local)) != Frontier || Escalate(Local) == Frontier {
		t.Fatal("Escalate moves one step per call")
	}
}

// TestResolveAtReResolvesTheRowAtTheEscalatedTier: the winning row is resolved
// again with its tier moved, through the same legs and the same allowlist, so
// an escalated tier no connection serves goes to the harness with that tier
// named, and one a connection serves is held to its allowlist (AC 11).
func TestResolveAtReResolvesTheRowAtTheEscalatedTier(t *testing.T) {
	f := newFx(t)
	f.machine(`{"scribe":{"tier":"economy","fan_out":1}}`)
	l := f.load()
	r, err := ResolveAt("scribe", l, NoConnections{}, Frontier)
	if err != nil {
		t.Fatal(err)
	}
	if r.Row.Tier != Frontier || r.ConnectionUsed != Harness || r.Source.String() != "machine" || r.Override != "" {
		t.Fatalf("re-resolved at frontier = %+v", r)
	}
	if rr := r.Receipt(""); rr.TierAsked != Frontier {
		t.Fatalf("the receipt names %s as the tier asked, want the escalated frontier", rr.TierAsked)
	}
	// A connection serving the escalated tier whose allowlist lists no model
	// refuses the route rather than sending it, naming the provider.
	s := &spy{serves: map[Tier]Connection{Frontier: {Name: "big-cloud"}}}
	if _, err := ResolveAt("scribe", l, s, Frontier); err == nil || !strings.Contains(err.Error(), "big-cloud") ||
		!strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("an escalated tier off the allowlist is refused naming the provider: %v", err)
	}
	if _, err := ResolveAt("scribe", l, NoConnections{}, "cheap"); err == nil {
		t.Fatal("a tier outside the vocabulary is refused")
	}
}
