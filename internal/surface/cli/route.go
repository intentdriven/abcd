package cli

// route.go is the --route flag every delegating verb carries, and the two
// places a resolved route leaves a verb: the request block a verb hands the
// host before a step runs, and the receipt its ingest returns after
// (itd-2609170822093401, spc-2609180535002478 steps 3 and 4).
//
// A delegating verb is one whose step is run by an agent in the roster under
// agents/: `intent audit` (and its ingest), `launch ship`, `disembark review`,
// `principles`, `press-release` and `graveyard`, and `reading ingest`. Each
// registers the flag through addRouteFlag, naming the agents it can dispatch,
// and TestEveryDelegatingVerbCarriesRoute holds the command tree to that list,
// so a verb cannot gain delegation without gaining the flag.
//
// The route is resolved before anything else the verb does, so every refusal
// (a malformed routing table, a --route naming an agent this invocation does
// not dispatch, a tier outside the vocabulary, a connection this machine has
// not configured) exits 2 before any step runs and before anything is
// written. A fallback to the harness is announced on stderr, one line.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// machineConnections is the machine's configured provider connections. The
// provider adapter intent (itd-2609081951381895) supplies a real one; until it
// lands no provider is configured, so every step resolves to the harness. It
// is a variable so a test can hand the verbs a reachable provider without a
// socket.
var machineConnections = func() oracle.Connections { return oracle.NoConnections{} }

// routeFlag is one delegating verb's --route values and the agents the verb
// can dispatch.
type routeFlag struct {
	texts  []string
	agents []string
}

// The agents the delegating verbs dispatch, by the name each carries in
// agents/ (the roster oracle.Roster holds the proposal to).
const (
	auditAgent        = "intent-auditor"
	changelogAgent    = "release-changelog-composer"
	reviewAgent       = "lifeboat-reviewer"
	principlesAgent   = "principle-distiller"
	pressReleaseAgent = "press-release-composer"
	graveyardAgent    = "graveyard-interpreter"
	// readingAgentPrefix + a position names a cold-reading agent.
	readingAgentPrefix = "cold-reading-"
)

// delegatedAgent is the agent a dual-mode verb dispatches: agent when its
// payload flag is given, none when it runs its deterministic mode.
func delegatedAgent(payloadFlag, agent string) string {
	if payloadFlag == "" {
		return ""
	}
	return agent
}

// routeFlagName is the flag every delegating verb carries.
const routeFlagName = "route"

// addRouteFlag registers --route on a delegating verb that can dispatch
// agents. It is the one registration, so the flag's grammar, help and
// refusals are spelled once.
func addRouteFlag(cmd *cobra.Command, agents ...string) *routeFlag {
	rf := &routeFlag{agents: agents}
	cmd.Flags().StringArrayVar(&rf.texts, routeFlagName, nil,
		"route one agent for this run: "+oracle.RouteSyntax+", tier one of "+tierHelp()+
			" (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)")
	return rf
}

func tierHelp() string {
	s := ""
	for i, t := range oracle.Tiers() {
		if i > 0 {
			s += " | "
		}
		s += string(t)
	}
	return s
}

// resolve returns the route for agent, the one agent this invocation
// dispatches, or a refusal at exit 2. agent "" means this invocation
// dispatches none (a deterministic mode), where a --route is refused and no
// table is read, so a verb's non-delegating modes are untouched by routing.
func (rf *routeFlag) resolve(cmd *cobra.Command, verb, agent string) (*oracle.Route, error) {
	stderr := cmd.ErrOrStderr()
	if agent == "" {
		if len(rf.texts) > 0 {
			return nil, &exitError{Code: 2, Msg: verb + ": --route routes an agent this step dispatches, and this " +
				"invocation dispatches none: without its payload flag the verb runs its deterministic mode, which no agent runs"}
		}
		return nil, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	roots, notes := layered.RootsFor(cwd)
	for _, n := range notes {
		fmt.Fprintf(stderr, "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	conns := machineConnections()
	l, err := oracle.Load(roots)
	if err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) +
			"; the routing table decides which model this step asks for, so it is refused rather than guessed past — fix or remove the file"}
	}
	for _, d := range l.Diagnostics {
		fmt.Fprintf(stderr, "abcd %s: %s\n", verb, termsafe.Sanitize(d))
	}
	routes, err := oracle.ParseRoutes(rf.texts, []string{agent}, conns)
	if err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(err.Error())}
	}
	if err := l.Apply(routes); err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(err.Error())}
	}
	r, err := oracle.Resolve(agent, l, conns)
	if err != nil {
		return nil, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	if r.Fallback != "" {
		fmt.Fprintf(stderr, "abcd %s: %s\n", verb, termsafe.Sanitize(r.Fallback))
	}
	return &r, nil
}

// withMember is a verb's result with one member added after its own: the
// request block ("routing") or the receipt ("route"). The verb's core result
// type stays ignorant of routing, and its own members keep their order.
type withMember struct {
	v   any
	key string
	val any
}

func (w withMember) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(w.v)
	if err != nil {
		return nil, err
	}
	base = bytes.TrimSpace(base)
	if len(base) < 2 || base[0] != '{' || base[len(base)-1] != '}' {
		return nil, fmt.Errorf("a %s member needs a JSON object to join, got %.20s", w.key, base)
	}
	// The member is written after the verb's own, so a result already carrying
	// the key would hold it twice, which a reader resolves last-wins, silently.
	var own map[string]json.RawMessage
	if err := json.Unmarshal(base, &own); err != nil {
		return nil, err
	}
	if _, dup := own[w.key]; dup {
		return nil, fmt.Errorf("the verb's result already carries a %q member, so the %s block cannot join it", w.key, w.key)
	}
	add, err := json.Marshal(w.val)
	if err != nil {
		return nil, err
	}
	key, _ := json.Marshal(w.key)
	var out bytes.Buffer
	out.WriteByte('{')
	if body := bytes.TrimSpace(base[1 : len(base)-1]); len(body) > 0 {
		out.Write(body)
		out.WriteByte(',')
	}
	out.Write(key)
	out.WriteByte(':')
	out.Write(add)
	out.WriteByte('}')
	return out.Bytes(), nil
}

// withRequest joins the request block to a verb's result; with no route (no
// agent dispatched) the result is returned as it is.
func withRequest(v any, r *oracle.Route) any {
	if r == nil {
		return v
	}
	return withMember{v: v, key: "routing", val: r.Request()}
}

// withReceipt joins the receipt's route block to an ingest's result, with the
// model the payload reported.
func withReceipt(v any, r *oracle.Route, payload []byte) any {
	if r == nil {
		return v
	}
	return withMember{v: v, key: "route", val: r.Receipt(oracle.ModelReported(payload))}
}

// renderRequestLine is the request block's line in a verb's human rendering.
func renderRequestLine(w io.Writer, r *oracle.Route) {
	if r == nil {
		return
	}
	rr := r.Request()
	line := fmt.Sprintf("routing: %s at tier %s, fan-out %d, decided by %s (%s), via %s",
		rr.Agent, rr.Tier, rr.FanOut, rr.Source, rr.Origin, rr.Connection)
	if rr.Override != "" {
		line += ", override " + rr.Override
	}
	fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(line))
}

// renderReceiptLine is the receipt's route line in an ingest's human rendering.
func renderReceiptLine(w io.Writer, r *oracle.Route, payload []byte) {
	if r == nil {
		return
	}
	rc := r.Receipt(oracle.ModelReported(payload))
	model := rc.ModelReported
	if model == "" {
		model = "not reported"
	}
	line := fmt.Sprintf("route: %s asked tier %s, used %s", rc.Agent, rc.TierAsked, rc.ConnectionUsed)
	if rc.ConnectionTried != "" {
		line += ", tried " + rc.ConnectionTried
	}
	line += ", model reported " + model
	if rc.Override != "" {
		line += ", override " + rc.Override
	}
	fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(line))
}

// peekPayload reads an ingest's payload file for the receipt's model and a
// reading's position, bounded; any fault returns nil, and the verb's own
// guarded read then refuses the payload with its own reason.
func peekPayload(path string, limit int64) []byte {
	if path == "" || path == "-" {
		return nil
	}
	raw, err := readGuardedOperand(path, limit)
	if err != nil {
		return nil
	}
	return raw
}

// routeCloseRequest gives the fidelity-review request a spec close emitted the
// same `## Routing` section `intent audit <itd-N>` writes, so a host that reads
// the close's request directly runs the auditor at the resolved tier. The core
// close emits the request without one (the route is a front-door fact), so the
// request is re-emitted here with the section once the route resolves.
//
// Only a request still owed is rewritten. The close is a record move whose
// emit is report-only, so a route that cannot be resolved does not undo or
// refuse it: the request keeps no routing section and one stderr line names the
// fault and the re-emit that adds the section once the table reads.
func routeCloseRequest(cmd *cobra.Command, repoRoot string, res intent.ReconcileResult) {
	if res.AuditEmitError != "" || (res.ReceiptStatus != "owed" && res.ReceiptStatus != "already_owed") {
		return
	}
	stderr := cmd.ErrOrStderr()
	id := termsafe.Sanitize(res.Intent.ID)
	route, err := (&routeFlag{}).resolve(cmd, "abcd spec close", auditAgent)
	if err == nil {
		_, err = intent.ReEmitAuditWith(repoRoot, res.Intent.ID,
			intent.AuditEmitOptions{RoutingSection: oracle.RenderRequestSection(route.Request())})
	}
	if err != nil {
		fmt.Fprintf(stderr, "WARNING: abcd spec close — the fidelity-review request for %s carries no routing section (the close stands): %s; "+
			"once that is fixed, `abcd intent audit %s` re-emits the request with one\n",
			id, termsafe.Sanitize(fsutil.RedactHome(strings.TrimPrefix(err.Error(), "abcd spec close: "))), id)
	}
}
