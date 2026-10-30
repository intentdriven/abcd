package oracle

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// RouteSyntax is the --route flag's grammar, as every refusal names it.
const RouteSyntax = "<agent>=<tier>[@<connection>][?k=v,...]"

// RunnerRouteSyntax is the --route flag's runner rung (iss-2610161801358637):
// the role, the runner that runs it for this invocation, and, after @, the
// <provider>/<model> route the runner asks for in place of its machine model.
// It is the one per-invocation override, extended: a verb that launches a
// runner for its roles takes this rung, a verb that sends its agent to the
// harness or a provider takes the tier rung, and each refuses the other.
const RunnerRouteSyntax = "<role>=runner:<name>[@<provider>/<model>]"

// runnerRung marks a --route's runner rung; no tier is spelt with a colon, so
// every route the tier rung admits reads as it did.
const runnerRung = "runner:"

// FlagRoute is one parsed --route: a row for one agent for this invocation
// alone, the connection it names (if any), and the flag text verbatim. A
// runner-rung route carries the runner and the model instead of a row.
type FlagRoute struct {
	Agent      string
	Row        Row
	Connection string
	// Runner is the runner a runner-rung route hands the role to; "" on a
	// tier route.
	Runner string
	// Model is the <provider>/<model> route a runner-rung route asks for; ""
	// keeps the runner's machine model.
	Model string
	Text  string
}

// ParseRoutes parses every --route value and refuses, before any step runs, a
// value that is malformed, names an agent this invocation does not dispatch
// (one outside the roster included), names a tier outside the vocabulary,
// names a connection not configured on this machine, carries a malformed
// setting, or repeats an agent. dispatched is the set of agents this
// invocation dispatches, which for every delegating verb is one agent, so a
// second --route naming another agent is refused rather than merged (the
// 2026-09-25 ruling on AC 7, dcn-2609250000000009).
func ParseRoutes(texts, dispatched []string, conns Connections) ([]FlagRoute, error) {
	parse := func(text string) (FlagRoute, error) {
		if _, rest, ok := strings.Cut(text, "="); ok && strings.HasPrefix(rest, runnerRung) {
			// Refused whole, malformed or not: the rung is not this verb's,
			// so its grammar is never the one a refusal teaches here.
			return FlagRoute{}, fmt.Errorf("names a runner, and this invocation sends %s to the harness or a provider "+
				"connection, never through a runner; a runner route is for a role abcd launches a runner for, "+
				"in `abcd implement step`; here, want %s", dispatchList(dispatched), RouteSyntax)
		}
		return parseRoute(text)
	}
	return parseEach(texts, dispatched, parse, func(fr FlagRoute, shown string) error {
		if !inRoster(fr.Agent) {
			return notDispatched(shown, dispatched, fr.Agent)
		}
		if fr.Connection != "" {
			if _, ok := conns.Named(fr.Connection); !ok {
				return fmt.Errorf("--route %s: its connection (%s) is not configured on this machine", shown, termsafe.DescribeRefused(fr.Connection))
			}
		}
		return nil
	})
}

// ParseRunnerRoutes parses every --route value of an invocation that runs its
// roles on the host or a runner (the build loop's step), and refuses, before
// any step runs, a value that is malformed, names a role this invocation does
// not dispatch, repeats a role, or is a tier route, which no routing table
// turns into a runner. Every malformed value is refused naming the runner
// grammar alone. Which runners the machine enables and which models their
// providers list are the runner configuration's to check
// (runner.Config.Override).
func ParseRunnerRoutes(texts, dispatched []string) ([]FlagRoute, error) {
	return parseEach(texts, dispatched, parseRunnerRoute, func(FlagRoute, string) error { return nil })
}

// notDispatched is the refusal of a --route naming an agent the invocation
// does not dispatch.
func notDispatched(shown string, dispatched []string, agent string) error {
	return fmt.Errorf("--route %s: this invocation dispatches %s, not %s; a --route naming any other agent is refused, never merged or ignored",
		shown, dispatchList(dispatched), namedAgent(agent))
}

// namedAgent is how a refusal names the agent a --route gave: quoted when it
// is an agent of the roster, so the caller reads which one, and otherwise by
// its length alone, since it is only what the caller typed.
func namedAgent(agent string) string {
	if inRoster(agent) {
		return strconv.Quote(agent)
	}
	return "the agent given (" + termsafe.DescribeRefused(agent) + ")"
}

// parseEach parses every --route value, holds each to the bound, the
// invocation's dispatched set and one route per agent, and hands it to check
// for what the invocation's own dispatch admits. Each refusal names the value
// by its position among the --route values, never by its text: the build
// loop's step carries it into its --json refusal document, which the error
// boundary's pass does not read.
func parseEach(texts, dispatched []string, parse func(string) (FlagRoute, error), check func(fr FlagRoute, shown string) error) ([]FlagRoute, error) {
	can := map[string]bool{}
	for _, a := range dispatched {
		can[a] = true
	}
	seen := map[string]bool{}
	var out []FlagRoute
	for i, text := range texts {
		shown := RoutePosition(i)
		if len(text) > MaxRouteBytes {
			return nil, fmt.Errorf("--route %s: it is %d bytes; a --route is at most %d, because a receipt carries it verbatim",
				shown, len(text), MaxRouteBytes)
		}
		fr, err := parse(text)
		if err != nil {
			return nil, fmt.Errorf("--route %s: %w", shown, err)
		}
		if !can[fr.Agent] {
			return nil, notDispatched(shown, dispatched, fr.Agent)
		}
		if seen[fr.Agent] {
			return nil, fmt.Errorf("--route %s: %s is routed more than once in this invocation", shown, fr.Agent)
		}
		seen[fr.Agent] = true
		if err := check(fr, shown); err != nil {
			return nil, err
		}
		out = append(out, fr)
	}
	return out, nil
}

// RoutePosition is how a refusal names the --route value at index i of an
// invocation's --route values: "number <i+1>", read after "--route".
func RoutePosition(i int) string { return "number " + strconv.Itoa(i+1) }

// parseRoute reads one tier route, <agent>=<tier>[@<connection>][?k=v,...].
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
		// ParseTier names a routing file's tier as written; a typed one is
		// named by its length.
		return FlagRoute{}, fmt.Errorf("its tier (%s) is not one of %s", termsafe.DescribeRefused(tierText), tierList())
	}
	fr := FlagRoute{Agent: agent, Row: Row{Tier: tier}, Connection: conn, Text: text}
	if hasQuery {
		raw := map[string]json.RawMessage{}
		for _, pair := range strings.Split(query, ",") {
			k, v, ok := strings.Cut(pair, "=")
			if !ok || k == "" || v == "" {
				return FlagRoute{}, fmt.Errorf("a setting (%s) is not k=v; want %s", termsafe.DescribeRefused(pair), RouteSyntax)
			}
			if _, dup := raw[k]; dup {
				return FlagRoute{}, fmt.Errorf("a setting's key (%s) is given more than once", termsafe.DescribeRefused(k))
			}
			raw[k] = settingValue(v)
		}
		s, err := checkSettings(raw, true)
		if err != nil {
			return FlagRoute{}, err
		}
		fr.Row.Settings = s
	}
	return fr, nil
}

// parseRunnerRoute reads one runner route, <role>=runner:<name>[@<provider>/<model>].
// A value that is not one is refused naming that grammar: a tier route as a
// tier, which this invocation does not choose by, and anything else as the
// runner rung left out, never as a tier it did not ask for.
func parseRunnerRoute(text string) (FlagRoute, error) {
	agent, rest, ok := strings.Cut(text, "=")
	if !ok || agent == "" || rest == "" {
		return FlagRoute{}, fmt.Errorf("want %s", RunnerRouteSyntax)
	}
	rung, isRunner := strings.CutPrefix(rest, runnerRung)
	if !isRunner {
		if _, err := parseRoute(text); err == nil {
			return FlagRoute{}, fmt.Errorf("names a tier, and this invocation runs its roles on the host or a runner, which "+
				"a tier does not choose; want %s", RunnerRouteSyntax)
		}
		return FlagRoute{}, fmt.Errorf("names no runner: the runner is named after %s, as in <role>=%sopencode; want %s",
			runnerRung, runnerRung, RunnerRouteSyntax)
	}
	if strings.Contains(rung, "?") {
		return FlagRoute{}, fmt.Errorf("a runner route takes no settings: the runner's harness sets its model's own; want %s", RunnerRouteSyntax)
	}
	return parseRunnerRung(agent, rung, text)
}

// parseRunnerRung reads the runner rung after runner:, <name>[@<provider>/<model>].
func parseRunnerRung(role, rung, text string) (FlagRoute, error) {
	name, model, hasModel := strings.Cut(rung, "@")
	if name == "" {
		return FlagRoute{}, fmt.Errorf("no runner after %s; want %s", runnerRung, RunnerRouteSyntax)
	}
	if hasModel {
		if model == "" {
			return FlagRoute{}, fmt.Errorf("an empty model after @; want %s", RunnerRouteSyntax)
		}
		provider, id, ok := strings.Cut(model, "/")
		if !ok || provider == "" || id == "" {
			return FlagRoute{}, fmt.Errorf("its model (%s) is not <provider>/<model>; want %s", termsafe.DescribeRefused(model), RunnerRouteSyntax)
		}
	}
	return FlagRoute{Agent: role, Runner: name, Model: model, Text: text}, nil
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

// dispatchList names the agents a verb dispatches, for a refusal.
func dispatchList(dispatched []string) string {
	if len(dispatched) == 0 {
		return "no agent in the roster"
	}
	return strings.Join(dispatched, ", ")
}
