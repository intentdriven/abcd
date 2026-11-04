package cli

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// agentsScope says whose launches a listing holds.
type agentsScope struct {
	Session string `json:"session,omitempty"`
	Name    string `json:"name,omitempty"`
	Run     string `json:"run,omitempty"`
	All     bool   `json:"all,omitempty"`
}

// agentsOutput is the verb's --json: the scope and the ledger's listing.
type agentsOutput struct {
	Scope agentsScope `json:"scope"`
	runner.LedgerListing
}

// newAgentsCommand builds the `agents` verb, the front door onto the launch
// ledger (itd-2611031342011261): it lists and totals what the runner's
// RecordLaunch wrote, and writes nothing.
func newAgentsCommand(asJSON *bool) *cobra.Command {
	var run, session string
	var all bool
	cmd := &cobra.Command{
		Use: "agents",
		Long: "List the agents abcd started and add them up, from the launch ledger on this\n" +
			"computer (`" + abcdhome.Display("agents/") + "`): one line per launch naming the session or run that\n" +
			"asked for it, the work, the billing route, the model, the tokens, the cost, the API\n" +
			"time and how it ended.\n\n" +
			"Bare, it lists this session's launches; --run <id> lists one run's, --session <id>\n" +
			"one session's and --all every one. Money is counted for launches billed to a named\n" +
			"key alone: a subscription's or an unplaced launch's cost is shown as an estimate,\n" +
			"which is no money spent. Every cost is the harness's own estimate, and the\n" +
			"provider's invoice is the final figure. An entry whose process is gone with no end\n" +
			"written reads as lost. An entry it cannot read is named, never dropped.\n\n" +
			"It reads only. Exit 2 on a refusal: no session to list, or two scopes at once.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd agents: "
			scopes := 0
			for _, on := range []bool{run != "", session != "", all} {
				if on {
					scopes++
				}
			}
			if scopes > 1 {
				return &exitError{Code: 2, Msg: prefix + "--run, --session and --all each name a whole listing; pass one"}
			}
			roots, note := layered.HomeRoots()
			if note != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "abcd: "+note)
			}
			if roots.Home == "" {
				return &exitError{Code: 2, Msg: prefix + "no home to read " + abcdhome.Display("agents/") + " from; nothing was read"}
			}
			q := runner.LedgerQuery{Run: run, Session: session, All: all}
			scope := agentsScope{Run: run, Session: session, All: all}
			if scopes == 0 {
				o, ok := runner.CurrentSession(roots.Home)
				if !ok {
					return &exitError{Code: 2, Msg: prefix + "this command runs under no session abcd can name, so there is no " +
						"\"this session\" to list; pass --run <id>, --session <id> or --all"}
				}
				q.Session, scope.Session, scope.Name = o.ID, o.ID, o.Name
			}
			listing, err := runner.ReadLedger(roots.Home, q)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
			}
			out := agentsOutput{Scope: scope, LedgerListing: listing}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderAgents(w, out) })
		},
	}
	cmd.Flags().StringVar(&run, "run", "", "list the launches of this run")
	cmd.Flags().StringVar(&session, "session", "", "list the launches this session asked for")
	cmd.Flags().BoolVar(&all, "all", false, "list every launch the ledger holds")
	return cmd
}

// renderAgents is the listing as text. Every word an entry supplies is
// sanitised: a file in the store is data, whoever wrote it.
func renderAgents(w io.Writer, o agentsOutput) {
	safe := termsafe.Sanitize
	fmt.Fprintf(w, "abcd agents — %s\n", safe(agentsScopeText(o.Scope)))
	if len(o.Entries) == 0 {
		fmt.Fprintln(w, "no launches")
	}
	for _, e := range o.Entries {
		fmt.Fprintf(w, "%s  %s\n", safe(e.ID), safe(agentsLine(e)))
		if e.Reason != "" {
			fmt.Fprintf(w, "  %s\n", safe(e.Reason))
		}
		if e.RouteWhy != "" {
			fmt.Fprintf(w, "  route unknown: %s\n", safe(e.RouteWhy))
		}
	}
	t := o.Totals
	fmt.Fprintf(w, "\n%d launches: %d running, %d finished, %d failed, %d stopped, %d lost, %d handed to the host\n",
		t.Launches, t.Running, t.Finished, t.Failed, t.Stopped, t.Lost, t.Handed)
	for _, key := range slices.Sorted(maps.Keys(t.SpendByKey)) {
		fmt.Fprintf(w, "spend on %s: %s\n", safe(key), agentsUSD(t.SpendByKey[key]))
	}
	if t.EstimateUSD > 0 {
		fmt.Fprintf(w, "estimate, no money spent: %s\n", agentsUSD(t.EstimateUSD))
	}
	fmt.Fprintf(w, "%d tokens (%s)\n", t.Tokens, o.TokenBasis)
	fmt.Fprintf(w, "API time: %s\n", agentsDuration(t.APIMillis))
	fmt.Fprintln(w, o.CostNote)
	for _, r := range o.Repeats {
		fmt.Fprintf(w, "two launches for one await: %s %s %s: %s\n", safe(r.Run), safe(r.Lane), safe(r.Await), safe(strings.Join(r.Launches, ", ")))
	}
	for _, id := range o.Newer {
		fmt.Fprintf(w, "written by a newer abcd, read as far as this one can: %s\n", safe(id))
	}
	for _, u := range o.Unreadable {
		fmt.Fprintf(w, "could not read %s: %s\n", safe(u.File), safe(u.Why))
	}
	if o.LivenessUnread {
		fmt.Fprintln(w, "the process table could not be read, so a running launch stands as written")
	}
}

func agentsScopeText(s agentsScope) string {
	switch {
	case s.All:
		return "every launch"
	case s.Run != "":
		return "run " + s.Run
	case s.Name != "":
		return "session " + s.Session + " (" + s.Name + ")"
	default:
		return "session " + s.Session
	}
}

// agentsLine is one launch: who asked, the work, the route, the model, the
// figures and how it ended, each field left out when the harness reported none.
func agentsLine(e runner.LedgerEntry) string {
	parts := []string{e.Orchestrator.ID}
	if e.Orchestrator.Name != "" {
		parts[0] = e.Orchestrator.Name
	}
	work := e.Role
	for _, p := range []string{e.Lane, e.Run} {
		if p != "" {
			work += " " + p
		}
	}
	parts = append(parts, work, agentsRoute(e), cmp.Or(e.Model, e.ModelAsked, "model not reported"))
	if u := e.Usage; u != nil {
		parts = append(parts, strconv.Itoa(u.PromptTokens+u.CompletionTokens+u.CacheReadTokens+u.CacheWriteTokens)+" tokens")
	}
	if c := e.Cost; c != nil {
		text := strconv.FormatFloat(c.Amount, 'f', -1, 64) + " " + c.Unit
		if c.Unit == "usd" {
			text = agentsUSD(c.Amount)
		}
		parts = append(parts, text)
	}
	if e.APIMillis != nil {
		parts = append(parts, agentsDuration(*e.APIMillis)+" API")
	}
	return strings.Join(append(parts, e.Outcome), "  ")
}

func agentsRoute(e runner.LedgerEntry) string {
	if e.Route == runner.RouteKey && e.Key != "" {
		return "key " + e.Key
	}
	return e.Route
}

func agentsUSD(v float64) string { return fmt.Sprintf("$%.2f", v) }

func agentsDuration(ms int64) string {
	return (time.Duration(min(ms, 1<<62/int64(time.Millisecond))) * time.Millisecond).Round(time.Second).String()
}
