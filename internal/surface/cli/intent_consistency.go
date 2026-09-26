package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newIntentConsistencyCommand builds `abcd intent consistency`, Role 2 of the
// intent-auditor (itd-48): the cross-document consistency pass over the brief
// and every intent. The bare form (or one intent id) emits the request and the
// assembled corpus under the local tier; `ingest --findings-json` validates the
// host's findings and writes the dated report on the reviews shelf and one
// capture per finding. Both are front doors onto internal/core; every refusal
// exits 2 with nothing written.
func newIntentConsistencyCommand(asJSON *bool) *cobra.Command {
	var emitRoute, ingestRoute *routeFlag
	cmd := &cobra.Command{
		Use:  "consistency [<itd-N>]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := intentStoreRoot(cmd)
			if err != nil {
				return err
			}
			route, err := emitRoute.resolve(cmd, "abcd intent consistency", auditAgent)
			if err != nil {
				return err
			}
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			res, err := intent.EmitConsistency(repoRoot, id,
				intent.ConsistencyEmitOptions{RoutingSection: oracle.RenderRequestSection(route.Request())})
			if err != nil {
				e := &exitError{Code: 2, Msg: "abcd intent consistency: " + fsutil.RedactHome(err.Error())}
				if id != "" {
					return peerHeldRefusal(repoRoot, "abcd intent consistency: ", id, e)
				}
				return e
			}
			return render(cmd.OutOrStdout(), *asJSON, withRequest(res, route), func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent consistency — %s %s (receipt %s)\n", res.Scope, res.Status, res.ReceiptID)
				fmt.Fprintf(w, "  read: %d documents (%d brief pages, %d intents) at %s\n",
					res.Documents, res.BriefDocuments, res.IntentDocuments, res.ReviewOfCommit)
				fmt.Fprintf(w, "  request: %s\n  corpus: %s\n", res.RequestPath, res.CorpusPath)
				renderRequestLine(w, route)
			})
		},
	}
	emitRoute = addRouteFlag(cmd, auditAgent)

	var findingsJSON string
	ingestCmd := &cobra.Command{
		Use:  "ingest --findings-json <path>",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := intentStoreRoot(cmd)
			if err != nil {
				return err
			}
			if findingsJSON == "" {
				return &exitError{Code: 2, Msg: "abcd intent consistency ingest: --findings-json <path> is required"}
			}
			route, err := ingestRoute.resolve(cmd, "abcd intent consistency ingest", auditAgent)
			if err != nil {
				return err
			}
			// Read once: the ingest validates these bytes and the receipt's
			// model_reported is read from them.
			payload, err := intent.ReadConsistencyFindings(findingsJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent consistency ingest: " + fsutil.RedactHome(err.Error())}
			}
			res, err := capture.IngestConsistency(repoRoot, payload, "")
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent consistency ingest: " + fsutil.RedactHome(err.Error())}
			}
			return render(cmd.OutOrStdout(), *asJSON, withReceipt(res, route, payload), func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent consistency ingest — %s (receipt %s, scope %s)\n", res.Status, res.ReceiptID, res.Scope)
				fmt.Fprintf(w, "  report: %s (read %s)\n", res.ReportPath, res.ReviewOfCommit)
				if res.Status == "ingested" {
					fmt.Fprintf(w, "  findings %d: filed %d · linked to an open record %d\n", res.Findings, len(res.Filed), len(res.Linked))
					for _, r := range res.Rows {
						how := "filed"
						if r.Linked {
							how = "already open"
						}
						fmt.Fprintf(w, "  %d. %s (%s) — %s %s: %s\n", r.Number, r.ClassLabel(), r.Severity, r.IssueID, how,
							termsafe.Sanitize(r.Summary))
					}
				}
				renderReceiptLine(w, route, payload)
			})
		},
	}
	ingestCmd.Flags().StringVar(&findingsJSON, "findings-json", "", "path to the consistency findings JSON the intent-auditor returned")
	ingestRoute = addRouteFlag(ingestCmd, auditAgent)
	cmd.AddCommand(ingestCmd)
	return cmd
}
