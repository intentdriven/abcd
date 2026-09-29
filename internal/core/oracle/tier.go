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
// a provider that is "reachable" without a socket. The provider adapter
// (itd-2609081951381895, config.go) implements Connections from the machine's
// provider blocks, each connection carrying its allowlist, the settings its
// adapter accepts and the model each role pointed at it asks for; the
// delegating verbs still hand every resolution
// NoConnections, so every row resolves to the harness until provider dispatch
// lands (spc-2609251028149555).
//
// Staged, loudly (the loud-staging rule): spc-2609180535002478 lands the types,
// the proposal and its roster test, the store readers, the --route parser,
// Resolve, the bare board's oracle lines, the request block and receipt every
// delegating verb carries (Route.Request, Route.Receipt), and the ahoy consent
// step that writes an accepted table. spc-2609251028149555 adds the refusals
// Resolve makes on a provider leg before the step runs: a connection whose
// allowlist lists no model admits no route; the model the agent's
// oracle.roles.<agent> points at on the connection must be on its allowlist;
// a leg to a connection the agent's role does not point at names no model and
// is refused, because the record does not yet decide which model it asks for;
// and a merged setting outside the set the connection's adapter accepts is
// refused, never dropped. Escalating a tier after a failed fix round and dispatching a
// step to a provider are still that spec's; until dispatch lands every
// delegating verb resolves against NoConnections, so every step resolves to
// the harness and no front door reaches a provider-leg refusal.
package oracle

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
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
	return "", fmt.Errorf("tier %q is not one of %s", layered.BoundKey(s), tierList())
}

func tierList() string {
	out := make([]string, len(tiers))
	for i, t := range tiers {
		out[i] = string(t)
	}
	return strings.Join(out, ", ")
}
