package oracle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// Harness is the connection name of the harness leg: the host's own sub-agent
// dispatch, which owns model choice, credentials and execution (adr-25).
const Harness = "harness"

// Connection is one provider connection configured on this machine, with the
// sampling settings it sends by default.
type Connection struct {
	Name     string
	Defaults Settings
	// Models is the provider's allowlist (adr-2609221009491186): the only
	// models it may serve, every one already cleared against oracle.denylist
	// when the configuration was read. nil on a connection no
	// provider block backs.
	Models []string
	// Accepts is the settings the connection's adapter accepts; a setting
	// outside it is refused before a step runs, never dropped
	// (spc-2609251028149555, AC 8). nil on a connection no adapter backs.
	Accepts []string
	// Roles is the model each agent whose oracle.roles.<agent> points at this
	// connection asks it for, keyed by agent: a provider is reached by a role
	// pointed at <provider>/<model> (itd-2609081951381895 Decision 9), so the
	// role is where a provider leg's model is chosen. nil when no role points
	// here.
	Roles map[string]string
	// Keyed reports that the provider holds a key: its block names a
	// credential, so a call through it spends the person's key. A keyed leg
	// takes no setting from the repository's routing row: only the person's
	// own machine configuration shapes a call that spends their key.
	Keyed bool
}

// Admits reports whether model is on the connection's allowlist.
func (c Connection) Admits(model string) bool {
	for _, m := range c.Models {
		if m == model {
			return true
		}
	}
	return false
}

// Accepted reports whether the connection's adapter accepts setting key.
func (c Connection) Accepted(key string) bool {
	for _, k := range c.Accepts {
		if k == key {
			return true
		}
	}
	return false
}

// Connections is the machine's configured provider connections. The provider
// adapter intent (itd-2609081951381895) implements it; this package only
// consumes it, so it never configures, probes or reaches a provider itself.
type Connections interface {
	// Serves returns a configured connection that is reachable and can serve
	// the tier, if there is one.
	Serves(t Tier) (Connection, bool)
	// Named returns the configured connection of that name, if there is one.
	Named(name string) (Connection, bool)
	// Pointed returns the connection agent's role is pointed at, if it is.
	Pointed(agent string) (Connection, bool)
}

// NoConnections is a machine with no provider configured, and the only
// implementation until the adapter intent lands: every step goes to the
// harness, whatever its row says.
type NoConnections struct{}

// Serves reports that nothing serves any tier.
func (NoConnections) Serves(Tier) (Connection, bool) { return Connection{}, false }

// Named reports that no connection of any name is configured.
func (NoConnections) Named(string) (Connection, bool) { return Connection{}, false }

// Pointed reports that no role is pointed at any connection.
func (NoConnections) Pointed(string) (Connection, bool) { return Connection{}, false }

// Route is one agent's resolved routing for one step: what the request block
// carries and what the receipt records.
type Route struct {
	Agent string
	// Row is the winning row, the flag's overrides applied and the fan-out
	// clamped to the contract ceiling.
	Row Row
	// Source is the layer the row came from: flag, repo, machine, bundled, or
	// none when nothing is accepted.
	Source layered.Layer
	// Origin names that layer's file, the flag text, "bundled" or "none".
	Origin string
	// Override is the --route text verbatim when a flag governed the step, so
	// a measurement run is never mistaken for accepted routing.
	Override string
	// ConnectionTried is the provider connection the step was offered to; ""
	// when none was (host-decides, or no connection serves the tier).
	ConnectionTried string
	// ConnectionUsed is the provider connection that takes the step, or
	// Harness.
	ConnectionUsed string
	// Fallback is why a step whose tier asked for a provider went to the
	// harness; "" when it did not fall back. The front door prints it on
	// stderr before the step runs.
	Fallback string
	// SettingsSent is the provider leg's settings: the connection's defaults,
	// then the row's, then the flag's. nil on the harness leg.
	SettingsSent Settings
}

// Resolve returns agent's route: the flag's row over the repository's over the
// machine's over the bundled proposal (the last only once a table is
// accepted), resolved against conns. The leg is the connection a --route
// names; else, with no --route, the connection agent's role is pointed at;
// else the harness for host-decides, and a connection serving the tier for
// any other. It contacts no provider: Pointed is a lookup in the machine's
// own configuration, and conns is asked to serve only a tier that asks for a
// provider or a connection a --route named.
func Resolve(agent string, l *Layered, conns Connections) (Route, error) {
	if !inRoster(agent) {
		return Route{}, notInRoster(agent)
	}
	ceiling, _ := Ceiling(agent)
	found, err := l.stack.Lookup("agents." + agent)
	if err != nil {
		return Route{}, err
	}

	r := Route{Agent: agent}
	var flag *flagRow
	var flagText string
	var base *Row
	// baseWhere names where the base row came from, for a refusal of one of
	// its settings, and baseLayer its layer; r.Origin and r.Source are
	// overwritten when a flag governs the step.
	var baseWhere string
	var baseLayer layered.Layer
	for _, fd := range found {
		if fd.Layer == layered.Flag {
			fr, err := layered.Decode[flagRow](fd.Raw)
			if err != nil {
				return Route{}, fmt.Errorf("oracle routing: %s: %w", fd.Origin, err)
			}
			flag, flagText = &fr, fd.Origin
			continue
		}
		if base == nil {
			row, _, err := decodeRow(agent, fd.Raw)
			if err != nil {
				return Route{}, fmt.Errorf("oracle routing: %s (%s layer): agents.%s: %w", fd.Origin, fd.Layer, agent, err)
			}
			base, r.Source, r.Origin = &row, fd.Layer, fd.Origin
			baseWhere, baseLayer = fmt.Sprintf("%s (%s layer)", fd.Origin, fd.Layer), fd.Layer
		}
	}
	if base == nil {
		if l.accepted {
			row := Proposal()[agent]
			base, r.Source, r.Origin = &row, layered.Bundled, "bundled"
		} else {
			base, r.Source, r.Origin = &Row{Tier: HostDecides, FanOut: ceiling}, layered.None, "none"
		}
	}
	row := *base
	row.Settings = merge(nil, base.Settings)
	if flag != nil {
		tier, err := ParseTier(flag.Tier)
		if err != nil {
			return Route{}, fmt.Errorf("oracle routing: %s: %w", flagText, err)
		}
		row.Tier = tier
		row.Settings = merge(row.Settings, flag.Settings)
		r.Source, r.Origin, r.Override = layered.Flag, flagText, flagText
	}
	if row.FanOut <= 0 || row.FanOut > ceiling {
		row.FanOut = ceiling
	}
	if len(row.Settings) == 0 {
		row.Settings = nil
	}
	r.Row = row

	leg := providerLeg{agent: agent, flagText: flagText, baseWhere: baseWhere, baseLayer: baseLayer, base: base.Settings}
	if flag != nil {
		leg.flag = flag.Settings
	}
	switch {
	case flag != nil && flag.Connection != "":
		c, ok := conns.Named(flag.Connection)
		if !ok {
			return Route{}, fmt.Errorf("oracle routing: %s names connection %q, which is not configured on this machine", flagText, layered.BoundKey(flag.Connection))
		}
		leg.via = "named by --route " + flagText
		if err := leg.take(&r, c, row.Settings); err != nil {
			return Route{}, err
		}
	case flag == nil && pointedAt(conns, agent, &leg):
		if err := leg.take(&r, leg.conn, row.Settings); err != nil {
			return Route{}, err
		}
	case row.Tier == HostDecides:
		r.ConnectionUsed = Harness
	default:
		if c, ok := conns.Serves(row.Tier); ok {
			leg.via = fmt.Sprintf("serving tier %s", row.Tier)
			if flagText != "" {
				leg.via += " under --route " + flagText
			}
			if err := leg.take(&r, c, row.Settings); err != nil {
				return Route{}, err
			}
		} else {
			r.ConnectionUsed = Harness
			r.Fallback = fmt.Sprintf("no configured connection reachable from this machine serves tier %s, "+
				"so %s runs through the harness with tier %s named in its request", row.Tier, agent, row.Tier)
		}
	}
	return r, nil
}

// pointedAt reports whether agent's role is pointed at a configured
// connection, and if so gives leg that connection and how it was reached. A
// provider claims no tier (itd-2609081951381895 Decision 9): it is reached by
// the role pointed at it, so a pointed role takes the step whatever tier the
// routing tables name, host-decides included. Only a --route governs the step
// over it, for that run alone.
func pointedAt(conns Connections, agent string, leg *providerLeg) bool {
	c, ok := conns.Pointed(agent)
	if !ok {
		return false
	}
	leg.conn, leg.via = c, "pointed at by oracle.roles."+agent
	return true
}

// providerLeg is what a refusal on a provider leg names: the agent, how the
// leg was reached, and where each layer's settings came from.
type providerLeg struct {
	conn      Connection
	agent     string
	via       string
	flagText  string
	flag      Settings
	baseWhere string
	baseLayer layered.Layer
	base      Settings
}

// take gives r the provider leg through c, or refuses it before the step runs
// (spc-2609251028149555). The allowlist is consulted first, before any
// settings merge: a provider serves only the models it lists
// (adr-2609221009491186), so a connection that lists none admits no route, and
// the model the agent's role points at on c must be one it lists (AC 11). A
// leg to a connection the agent's role does not point at names no model, so
// it is refused rather than guessed, naming the oracle.roles.<agent> setting
// to add (AC 11, the product thinker's ruling of 2026-09-29). The merged
// settings are then held to the set c's adapter accepts: a setting outside it
// is refused, never dropped (AC 8), and a connection no adapter backs accepts
// none. On a keyed connection, a setting the repository's routing row names is
// refused too, never dropped: a paid key is spent only through the person's
// own machine configuration (ruling AA(b) of 2026-09-29), so the repository
// may not size or shape the call; the refusal names each setting, the
// repository file and the machine file to move it to.
func (p providerLeg) take(r *Route, c Connection, rowSettings Settings) error {
	if len(c.Models) == 0 {
		return fmt.Errorf("oracle routing: %s resolves to connection %s (%s), whose allowlist lists no model; "+
			"a provider serves only the models it lists (adr-2609221009491186), so the step is refused rather than sent: "+
			"list the models %s may serve in its provider block, or route %s to the harness with tier %s",
			p.agent, c.Name, p.via, c.Name, p.agent, HostDecides)
	}
	model, pointed := c.Roles[p.agent]
	if !pointed {
		return fmt.Errorf("oracle routing: %s resolves to connection %s (%s), but oracle.roles.%s does not point at %s, "+
			"so the route names no model for %s to serve, and a route that names no model is refused rather than sent: "+
			"point oracle.roles.%s at %s/<model> with a model its allowlist lists (%s), "+
			"or route %s to the harness with tier %s",
			p.agent, c.Name, p.via, p.agent, c.Name, c.Name, p.agent, c.Name, listNames(c.Models), p.agent, HostDecides)
	}
	if !c.Admits(model) {
		return fmt.Errorf("oracle routing: %s resolves to connection %s (%s), where oracle.roles.%s points at model %s, "+
			"which is not on %s's allowlist (%s); a provider serves only the models it lists (adr-2609221009491186), "+
			"so the step is refused rather than sent: add %s to %s's models in its provider block, point oracle.roles.%s "+
			"at a model the list holds, or route %s to the harness with tier %s",
			p.agent, c.Name, p.via, p.agent, layered.BoundKey(model), c.Name, listNames(c.Models),
			layered.BoundKey(model), c.Name, p.agent, p.agent, HostDecides)
	}
	if c.Keyed && p.baseLayer == layered.Repo && len(p.base) > 0 {
		machine := layered.OracleRouting.MachineOrigin()
		return fmt.Errorf("oracle routing: %s resolves to connection %s (%s), a provider that holds a key, and the repository's "+
			"routing row sets %s (from %s); only the person's own machine configuration may shape a call that spends their key, "+
			"so the step is refused rather than sent with those settings or without them: move agents.%s.settings to %s, "+
			"or remove it from %s",
			p.agent, c.Name, p.via, strings.Join(sortedKeys(p.base), ", "), p.baseWhere, p.agent, machine,
			layered.OracleRouting.RepoOrigin())
	}
	sent := merge(merge(nil, c.Defaults), rowSettings)
	var refused []string
	for _, k := range sortedKeys(sent) {
		if !c.Accepted(k) {
			refused = append(refused, fmt.Sprintf("%s (from %s)", k, p.where(k, c.Name)))
		}
	}
	if len(refused) > 0 {
		accepts := "it accepts no setting"
		if len(c.Accepts) > 0 {
			sorted := append([]string(nil), c.Accepts...)
			sort.Strings(sorted)
			accepts = "it accepts " + strings.Join(sorted, ", ")
		}
		return fmt.Errorf("oracle routing: %s resolves to connection %s (%s), whose adapter does not accept %s; %s. "+
			"A setting is refused rather than dropped, so the step does not run: remove each one from where it is set",
			p.agent, c.Name, p.via, strings.Join(refused, ", "), accepts)
	}
	r.ConnectionTried, r.ConnectionUsed = c.Name, c.Name
	r.SettingsSent = sent
	return nil
}

// where names the layer whose value of setting k is the one sent: the
// --route's over the row's over the connection's own defaults.
func (p providerLeg) where(k, conn string) string {
	if _, ok := p.flag[k]; ok {
		return "--route " + p.flagText
	}
	if _, ok := p.base[k]; ok {
		return p.baseWhere
	}
	return conn + "'s defaults"
}

func sortedKeys(s Settings) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// merge returns dst with src's keys laid over it, allocating when dst is nil.
func merge(dst, src Settings) Settings {
	if dst == nil {
		dst = Settings{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
