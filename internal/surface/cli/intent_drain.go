package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// runOwedDrain is `abcd intent audit --owed [--max <n>]` (itd-53,
// spc-2609211930059886): the owed reviews from the one reader, ordered oldest
// shipped first and capped in internal/core/intent, with the oldest's request
// emitted through the single audit's own emit and its path printed, so a host
// without the plugin page drives the drain by hand: audit the request, ingest
// the verdict, run this again. It runs no reviewer, so every entry stays owed
// until its verdict is ingested. The shipped day comes from the site package's
// one history walk; a history that cannot be read leaves every day unknown and
// the queue in mint order, said on stderr, rather than refusing the drain.
// The head is emitted the way `audit <itd-N>` emits it: the route is resolved
// first (so a refusal writes nothing), its section lands in the request, and
// the JSON next carries the routing member.
func runOwedDrain(cmd *cobra.Command, asJSON bool, max int, auditRoute *routeFlag) error {
	repoRoot, err := intentStoreRoot(cmd)
	if err != nil {
		return err
	}
	route, err := auditRoute.resolve(cmd, "abcd intent audit --owed", auditAgent)
	if err != nil {
		return err
	}
	var shippedOn intent.ShippedOn
	if h, herr := site.LoadHistory(repoRoot); herr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd intent audit --owed: the shipped days are unknown (%s); the queue falls to mint order\n",
			termsafe.Sanitize(herr.Error()))
	} else {
		shippedOn = h.EnteredBucket
	}
	step, err := intent.NextOwedAudit(repoRoot, max, shippedOn,
		intent.AuditEmitOptions{RoutingSection: oracle.RenderRequestSection(route.Request())})
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd intent audit --owed: " + err.Error()}
	}
	view := drainView{ReviewQueue: step.ReviewQueue}
	if step.Next != nil {
		view.Next = withRequest(*step.Next, route)
	}
	return render(cmd.OutOrStdout(), asJSON, view, func(w io.Writer) {
		if step.Owed == 0 {
			fmt.Fprintln(w, "abcd intent audit --owed — 0 owed; nothing to drain")
			return
		}
		fmt.Fprintf(w, "abcd intent audit --owed — %d owed, oldest shipped first", step.Owed)
		if step.Remaining > 0 {
			fmt.Fprintf(w, "; %d listed (--max %d), %d remain", len(step.Queue), step.Max, step.Remaining)
		}
		fmt.Fprintln(w)
		for i, e := range step.Queue {
			day := "shipped " + e.Shipped
			if e.Shipped == "" {
				day = "shipped (not yet committed)"
			}
			rcp := "receipt " + e.ReceiptID
			if e.ReceiptID == "" {
				rcp = "no receipt (one is minted on re-emit)"
			}
			fmt.Fprintf(w, "  %d. %s  %s  %s\n", i+1, e.IntentID, day, rcp)
		}
		if step.Next != nil {
			fmt.Fprintf(w, "next: %s (receipt %s) — request: %s\n", step.Next.IntentID, step.Next.ReceiptID, step.Next.RequestPath)
			renderRequestLine(w, route)
			fmt.Fprintln(w, "  hand the whole request to the intent-auditor agent, ingest its verdict with "+
				"`abcd intent audit ingest --verdict-json <file>`, then run this again for the next")
			fmt.Fprintln(w, "  a NOT_MET verdict is captured (`abcd capture`, naming the receipt), never fixed by the drain")
		}
		fmt.Fprintln(w, "this command runs no reviewer: every entry stays owed until its verdict is ingested")
	})
}

// drainView is the drain step as the front door renders it: the queue, and the
// head's emit joined with its routing member, as `audit <itd-N>` joins it.
type drainView struct {
	intent.ReviewQueue
	Next any `json:"next,omitempty"`
}
