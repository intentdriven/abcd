package oracle

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RouteSyntax is the --route flag's grammar, as every refusal names it.
const RouteSyntax = "<agent>=<tier>[@<connection>][?k=v,...]"

// FlagRoute is one parsed --route: a row for one agent for this invocation
// alone, the connection it names (if any), and the flag text verbatim.
type FlagRoute struct {
	Agent      string
	Row        Row
	Connection string
	Text       string
}

// ParseRoutes parses every --route value and refuses, before any step runs, a
// value that is malformed, names an agent the verb does not dispatch (one
// outside the roster included), names a tier outside the vocabulary, names a
// connection not configured on this machine, carries a malformed setting, or
// repeats an agent. dispatched is the set of agents the verb can dispatch.
func ParseRoutes(texts, dispatched []string, conns Connections) ([]FlagRoute, error) {
	can := map[string]bool{}
	for _, a := range dispatched {
		can[a] = true
	}
	seen := map[string]bool{}
	var out []FlagRoute
	for _, text := range texts {
		fr, err := parseRoute(text)
		if err != nil {
			return nil, fmt.Errorf("--route %s: %w", text, err)
		}
		if !can[fr.Agent] || !inRoster(fr.Agent) {
			return nil, fmt.Errorf("--route %s: this verb does not dispatch %q; it dispatches %s",
				text, fr.Agent, strings.Join(dispatched, ", "))
		}
		if seen[fr.Agent] {
			return nil, fmt.Errorf("--route %s: %s is routed more than once in this invocation", text, fr.Agent)
		}
		seen[fr.Agent] = true
		if fr.Connection != "" {
			if _, ok := conns.Named(fr.Connection); !ok {
				return nil, fmt.Errorf("--route %s: connection %q is not configured on this machine", text, fr.Connection)
			}
		}
		out = append(out, fr)
	}
	return out, nil
}

func parseRoute(text string) (FlagRoute, error) {
	head, query, hasQuery := strings.Cut(text, "?")
	agent, rest, ok := strings.Cut(head, "=")
	if !ok || agent == "" || rest == "" {
		return FlagRoute{}, fmt.Errorf("want %s", RouteSyntax)
	}
	tierText, conn, hasConn := strings.Cut(rest, "@")
	if hasConn && conn == "" {
		return FlagRoute{}, fmt.Errorf("an empty connection after @; want %s", RouteSyntax)
	}
	tier, err := ParseTier(tierText)
	if err != nil {
		return FlagRoute{}, err
	}
	fr := FlagRoute{Agent: agent, Row: Row{Tier: tier}, Connection: conn, Text: text}
	if hasQuery {
		raw := map[string]json.RawMessage{}
		for _, pair := range strings.Split(query, ",") {
			k, v, ok := strings.Cut(pair, "=")
			if !ok || k == "" || v == "" {
				return FlagRoute{}, fmt.Errorf("setting %q is not k=v; want %s", pair, RouteSyntax)
			}
			if _, dup := raw[k]; dup {
				return FlagRoute{}, fmt.Errorf("setting %s is given more than once", k)
			}
			raw[k] = settingValue(v)
		}
		s, err := checkSettings(raw)
		if err != nil {
			return FlagRoute{}, err
		}
		fr.Row.Settings = s
	}
	return fr, nil
}

// settingValue keeps a number or a boolean as JSON and quotes anything else
// as a string, so seed=42 reaches a provider as 42 and stop=END as "END".
func settingValue(v string) json.RawMessage {
	b := []byte(v)
	if json.Valid(b) && scalar(b) {
		return b
	}
	enc, _ := json.Marshal(v)
	return enc
}

// Apply places parsed routes in the flag layer, where they win over every
// accepted table for this invocation.
func (l *Layered) Apply(routes []FlagRoute) error {
	for _, fr := range routes {
		v := flagRow{Tier: string(fr.Row.Tier), Settings: fr.Row.Settings, Connection: fr.Connection}
		if err := l.stack.SetFlag("agents."+fr.Agent, v, fr.Text); err != nil {
			return err
		}
	}
	return nil
}
