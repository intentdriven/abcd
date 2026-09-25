// Package oracle is the model-tier routing of itd-2609170822093401: which tier
// of model each host-delegated agent deserves, how many agents a step may run
// at once, and, at run time, whether a configured provider takes the step or
// the harness does.
//
// The table has one row per agent (a tier from a closed set, a fan-out bound,
// provider settings) and four layers, resolved through the shared layered
// configuration resolver (internal/core/layered): an invocation --route over
// the repository's .abcd/config/oracle-routing.json over the machine's
// ~/.abcd/oracle-routing.json over the bundled proposal. Nothing is applied
// until a table is accepted: with neither file present every agent resolves to
// the harness at host-decides, and the bundled proposal is what an accepted
// table falls back to for an agent it has no row for.
//
// The resolver never writes, never reaches a network and never prints. It
// takes the machine's connections as a value (Connections), so a test hands it
// a provider that is "reachable" without a socket. Until the provider adapter
// intent (itd-2609081951381895) implements Connections, NoConnections is the
// only implementation, so every row resolves to the harness.
//
// Staged, loudly (the loud-staging rule): spc-2609180535002478 part 1 lands the
// types, the proposal and its roster test, the store readers, the --route
// parser and Resolve. The --route flag on the delegating verbs, the
// request-block and receipt fields, the ahoy consent step and the board lines
// are the spec's steps 3 to 6, listed in the spec's Progress note; no verb
// calls this package until they land.
package oracle

import (
	"fmt"
	"strings"
)

// Tier is a model tier: a class of model that survives model churn, not a
// model name.
type Tier string

const (
	// Local: a model on this machine, for material that must not leave it.
	Local Tier = "local"
	// Economy: a cheap cloud model, for plumbing no human reads as a verdict.
	Economy Tier = "economy"
	// Frontier: the strongest model, for verdicts a human reads and acts on.
	Frontier Tier = "frontier"
	// HostDecides: no tier asked; the harness picks. The floor every step
	// stands on when nothing is accepted.
	HostDecides Tier = "host-decides"
)

// tiers is the closed vocabulary, in the order every refusal renders it.
var tiers = []Tier{Local, Economy, Frontier, HostDecides}

// Tiers returns the closed tier vocabulary.
func Tiers() []Tier { return append([]Tier(nil), tiers...) }

// ParseTier admits exactly a member of the vocabulary, spelled as it is.
func ParseTier(s string) (Tier, error) {
	for _, t := range tiers {
		if s == string(t) {
			return t, nil
		}
	}
	return "", fmt.Errorf("tier %q is not one of %s", s, tierList())
}

func tierList() string {
	out := make([]string, len(tiers))
	for i, t := range tiers {
		out[i] = string(t)
	}
	return strings.Join(out, ", ")
}
