package oracle

// selfcontained.go is ruling DR5 of 2026-09-29 at dispatch: which agents a
// provider that holds a key (a paid one) may take, and the person's override.
//
// A provider call carries no tools, so an agent that reads files cannot read
// them there: whatever it needs must travel in the request. DR5 admits to a
// paid provider, by default, only an agent whose emitted request already
// carries all its input, named on a list compiled into the binary (default
// deny, as ruling AA(a) asked for an allow list rather than a deny list). The
// four cold-reading positions are that list: each is handed one assembled,
// manifest-hashed bundle and reads nothing else. Every other agent is refused
// before any call, with a reason naming the rule and the override.
//
// The override is oracle.bundled_context_providers in ~/.abcd.noindex/config.json: the
// providers the person lets take bundled-context requests for file-reading
// agents. It is read from the machine layer alone, and a repository declaring
// it is refused, as a repository's provider block is. It inherits the machine
// layer's trust in $HOME, pending iss-2609300012273350. A provider it names
// takes a file-reading agent only once abcd builds that agent's bundle, and
// none is built: which files travel, a size cap and a scan before sending are
// choices no ruling has made, so every file-reading agent stays refused.
//
// A provider whose block names no key (a local server) spends nothing of the
// person's and is outside DR5: any agent pointed at it is dispatched.

import (
	"fmt"
	"slices"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// bundledContextKey is the override's configuration key.
const bundledContextKey = "oracle.bundled_context_providers"

// selfContained is the agents whose emitted request carries all their input,
// sorted. An agent joins it only with a test proving its request reads no
// file.
var selfContained = []string{
	"cold-reading-comparative",
	"cold-reading-detection",
	"cold-reading-entailment",
	"cold-reading-widening",
}

// SelfContained returns a copy of the self-contained list.
func SelfContained() []string { return slices.Clone(selfContained) }

// bundleBuilt is the file-reading agents whose bundle abcd builds, so that a
// provider the override names may take them. None is built.
var bundleBuilt = map[string]bool{}

// BundledContextProviders returns the providers the override names, sorted.
func (c *APIConfig) BundledContextProviders() []string {
	out := make([]string, 0, len(c.bundled))
	for n := range c.bundled {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// readBundled reads oracle.bundled_context_providers from the machine layer.
// A repository declaring it is refused: the list decides what of a checkout
// may be sent under the person's key, so a checkout never writes it.
func (c *APIConfig) readBundled(s *layered.Stack) error {
	found, err := s.Lookup(bundledContextKey)
	if err != nil {
		return fmt.Errorf("oracle adapter: %w", err)
	}
	c.bundled = map[string]bool{}
	for _, fd := range found {
		if fd.Layer != layered.Machine {
			return fmt.Errorf("oracle adapter: %s (%s layer): %s is refused outside this machine's configuration: "+
				"it names the providers that may take a file-reading agent's files under the person's key, so only %s declares it",
				fd.Origin, fd.Layer, bundledContextKey, layered.Config.MachineOrigin())
		}
		names, err := layered.Decode[[]string](fd.Raw)
		if err != nil {
			return fmt.Errorf("oracle adapter: %s (machine layer): %s: %w", fd.Origin, bundledContextKey, err)
		}
		if len(names) > MaxProviders {
			return fmt.Errorf("oracle adapter: %s (machine layer): %s names %d providers; it names at most %d",
				fd.Origin, bundledContextKey, len(names), MaxProviders)
		}
		for _, n := range names {
			if !providerNameRe.MatchString(n) {
				return fmt.Errorf("oracle adapter: %s (machine layer): %s entry %q is not a provider's name",
					fd.Origin, bundledContextKey, layered.BoundKey(n))
			}
			if _, ok := c.providers[n]; !ok {
				c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (machine layer): %s names provider %q, "+
					"which is not configured on this machine; the entry admits nothing", fd.Origin, bundledContextKey, n))
				continue
			}
			c.bundled[n] = true
		}
	}
	return nil
}

// admitAgent is DR5: nil when agent may be sent to p. A provider that holds no
// key is outside the rule; a self-contained agent is admitted; any other agent
// is refused, naming the rule and the override, and stays refused on a
// provider the override names until its bundle is built.
func (c *APIConfig) admitAgent(agent string, p Provider) error {
	if !keyed(p) || slices.Contains(selfContained, agent) {
		return nil
	}
	a := termsafe.Sanitize(layered.BoundKey(agent))
	if c.bundled[p.Name] && bundleBuilt[agent] {
		return nil
	}
	if c.bundled[p.Name] {
		return fmt.Errorf("%s reads files, and %s in %s names %s, but abcd builds no bundle for %s: which files travel, "+
			"a size cap and a scan before sending are not yet decided, so it is refused before any call; "+
			"route it to the harness, or point it at a provider whose block names no key",
			a, bundledContextKey, layered.Config.MachineOrigin(), p.Name, a)
	}
	return fmt.Errorf("%s reads files, and a provider call carries none of them; ruling DR5 of 2026-09-29 admits only "+
		"self-contained agents (%s) to a provider that holds a key, so it is refused before any call; the person's "+
		"override is %s in %s, naming the providers that may take bundled-context requests for file-reading agents "+
		"whose bundle abcd builds; otherwise route it to the harness, or point it at a provider whose block names no key",
		a, listNames(selfContained), bundledContextKey, layered.Config.MachineOrigin())
}
