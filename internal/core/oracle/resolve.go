package oracle

import (
	"fmt"

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
	// models it may serve, every one already cleared against the vendor
	// denylist when the configuration was read. nil on a connection no
	// provider block backs.
	Models []string
	// Accepts is the settings the connection's adapter accepts; a setting
	// outside it is refused before a step runs, never dropped
	// (spc-2609251028149555, AC 8). nil on a connection no adapter backs.
	Accepts []string
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
}

// NoConnections is a machine with no provider configured, and the only
// implementation until the adapter intent lands: every step goes to the
// harness, whatever its row says.
type NoConnections struct{}

// Serves reports that nothing serves any tier.
func (NoConnections) Serves(Tier) (Connection, bool) { return Connection{}, false }

// Named reports that no connection of any name is configured.
func (NoConnections) Named(string) (Connection, bool) { return Connection{}, false }

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
// accepted), resolved against conns. It consults conns only for a tier that
// asks for a provider or a connection a --route named.
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

	switch {
	case flag != nil && flag.Connection != "":
		c, ok := conns.Named(flag.Connection)
		if !ok {
			return Route{}, fmt.Errorf("oracle routing: %s names connection %q, which is not configured on this machine", flagText, layered.BoundKey(flag.Connection))
		}
		r.ConnectionTried, r.ConnectionUsed = c.Name, c.Name
		r.SettingsSent = merge(merge(nil, c.Defaults), row.Settings)
	case row.Tier == HostDecides:
		r.ConnectionUsed = Harness
	default:
		if c, ok := conns.Serves(row.Tier); ok {
			r.ConnectionTried, r.ConnectionUsed = c.Name, c.Name
			r.SettingsSent = merge(merge(nil, c.Defaults), row.Settings)
		} else {
			r.ConnectionUsed = Harness
			r.Fallback = fmt.Sprintf("no configured connection reachable from this machine serves tier %s, "+
				"so %s runs through the harness with tier %s named in its request", row.Tier, agent, row.Tier)
		}
	}
	return r, nil
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
