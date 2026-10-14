package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/layered"
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
//
// The emit measures the corpus against consistency.max_request_bytes
// (itd-2609291923559186, A4). A corpus over it is emitted as chunks: on a
// provider route they are sent one after another and ingested together; on the
// harness each chunk's request is listed for the host, and the ingest takes
// --findings-json once per chunk.
func newIntentConsistencyCommand(asJSON *bool) *cobra.Command {
	var emitRoute, ingestRoute *routeFlag
	var maxChunks int
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
			if err := emitRoute.admit("abcd intent consistency", route); err != nil {
				return err
			}
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			roots, notes := layered.RootsFor(repoRoot)
			for _, n := range notes {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd intent consistency: %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
			}
			cfg, err := intent.LoadConsistencyConfig(roots)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent consistency: the request limit is refused: " +
					termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
			}
			if maxChunks < 1 {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd intent consistency: --max-chunks %d is not a chunk count (want 1 or more)", maxChunks)}
			}
			opts := intent.ConsistencyEmitOptions{
				RoutingSection:  oracle.RenderRequestSection(route.Request()),
				MaxRequestBytes: cfg.MaxRequestBytes,
			}
			// Every chunk a provider route sends is a request paid for on its
			// own, so a plan of more than --max-chunks is refused there before
			// anything is written or sent. The host runs a harness emit's
			// chunks itself, and decides for itself how many.
			if route != nil && route.OnProvider() {
				opts.MaxChunks = maxChunks
			}
			res, err := intent.EmitConsistency(repoRoot, id, opts)
			var capErr *intent.ConsistencySendCapError
			if errors.As(err, &capErr) {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd intent consistency: the corpus overflows into %d chunks, %d bytes of requests and corpus files in all, "+
					"and the route sends each to %s as a request of its own, more than the %d --max-chunks allows; nothing was written or sent. "+
					"Run again with --max-chunks %d to send them all, raise consistency.max_request_bytes for fewer chunks, or route the auditor to the host",
					capErr.Chunks, capErr.Bytes, termsafe.Sanitize(route.ConnectionUsed), capErr.Cap, capErr.Chunks)}
			}
			if err != nil {
				e := &exitError{Code: 2, Msg: "abcd intent consistency: " + fsutil.RedactHome(err.Error())}
				if id != "" {
					return peerHeldRefusal(repoRoot, "abcd intent consistency: ", id, e)
				}
				return e
			}
			// The ingest that follows a provider's answers warns in its turn,
			// so this emit warns only when the host is handed the chunks.
			warnEmit := func() { warnUnreviewed(cmd.ErrOrStderr(), "abcd intent consistency", "reviews", res.Overflow) }
			if route != nil && route.OnProvider() {
				ds, fell, err := sendConsistency(cmd, emitRoute, route, repoRoot, res)
				if err != nil {
					return err
				}
				if ds != nil {
					payloads := make([][]byte, len(ds))
					for i, d := range ds {
						payloads[i] = d.payload
					}
					return ingestConsistency(cmd, repoRoot, payloads, func(v any) any { return withDispatchReceipts(v, ds) },
						func(w io.Writer) {
							for _, d := range ds {
								renderDispatchLine(w, d)
							}
						}, *asJSON)
				}
				route = fell
			}
			warnEmit()
			return render(cmd.OutOrStdout(), *asJSON, withRequest(res, route), func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent consistency — %s %s (receipt %s)\n", res.Scope, res.Status, res.ReceiptID)
				fmt.Fprintf(w, "  read: %d documents (%d brief pages, %d intents) at %s\n",
					res.Documents, res.BriefDocuments, res.IntentDocuments, res.ReviewOfCommit)
				if res.Dirty {
					fmt.Fprintf(w, "  dirty: %d corpus path(s) differ from that commit, and the report will say so: %s\n",
						len(res.DirtyPaths), termsafe.Sanitize(strings.Join(res.DirtyPaths, ", ")))
				}
				if res.Overflow == nil {
					fmt.Fprintf(w, "  request: %s\n  corpus: %s\n", res.RequestPath, res.CorpusPath)
				} else {
					renderConsistencyOverflow(w, res.Overflow)
					for _, ch := range res.Chunks {
						fmt.Fprintf(w, "  chunk %d of %d: request %s, corpus %s (%d documents)\n",
							ch.Index, len(res.Chunks), ch.RequestPath, ch.CorpusPath, ch.Documents)
					}
					fmt.Fprintf(w, "  run every chunk, then ingest them together: abcd intent consistency ingest "+
						"--findings-json <chunk 1> ... --findings-json <chunk %d>\n", len(res.Chunks))
				}
				renderRequestLine(w, route)
			})
		},
	}
	emitRoute = addRouteFlag(cmd, auditAgent)
	cmd.Flags().IntVar(&maxChunks, "max-chunks", defaultConsistencySendChunks,
		"on a provider route, the most chunks of an overflowing corpus sent in one run; a plan of more is refused, naming its count and bytes")

	var findingsJSON []string
	ingestCmd := &cobra.Command{
		Use:  "ingest --findings-json <path> [--findings-json <path>...]",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := intentStoreRoot(cmd)
			if err != nil {
				return err
			}
			if len(findingsJSON) == 0 {
				return &exitError{Code: 2, Msg: "abcd intent consistency ingest: --findings-json <path> is required"}
			}
			route, err := ingestRoute.resolve(cmd, "abcd intent consistency ingest", auditAgent)
			if err != nil {
				return err
			}
			if err := hostPayloadOnProvider("abcd intent consistency ingest", route,
				"run `abcd intent consistency`, which sends the pass there and ingests the answer"); err != nil {
				return err
			}
			// Read once: the ingest validates these bytes and the receipt's
			// model_reported is read from them (the first chunk's, for a
			// chunk set).
			payloads := make([][]byte, 0, len(findingsJSON))
			for _, p := range findingsJSON {
				payload, err := intent.ReadConsistencyFindings(p)
				if err != nil {
					return &exitError{Code: 2, Msg: "abcd intent consistency ingest: " + fsutil.RedactHome(err.Error())}
				}
				payloads = append(payloads, payload)
			}
			return ingestConsistency(cmd, repoRoot, payloads, func(v any) any { return withReceipt(v, route, payloads[0]) },
				func(w io.Writer) { renderReceiptLine(w, route, payloads[0]) }, *asJSON)
		},
	}
	ingestCmd.Flags().StringArrayVar(&findingsJSON, "findings-json", nil,
		"path to the consistency findings JSON the intent-auditor returned; give it once per chunk of an overflowing corpus")
	ingestRoute = addRouteFlag(ingestCmd, auditAgent)
	cmd.AddCommand(ingestCmd)
	return cmd
}

// defaultConsistencySendChunks is --max-chunks unset: the most chunks a
// provider route sends without the person naming the count. At the bundled
// 512 KiB limit eight chunks are about 4 MiB of requests; past that, the run
// states the count and the bytes and waits for --max-chunks to name them.
const defaultConsistencySendChunks = 8

// sendConsistency sends an emit's requests to the provider the route names:
// the one request of a corpus that fits, or every chunk of one that overflows,
// one after another. It returns one dispatch per request, or nil when the host
// runs the pass: the route is on the harness, or a send fell back to it, in
// which case every chunk is left to the host and the returned route is the
// harness.
//
// Each chunk's answer is kept under the local tier as it arrives
// (intent.KeepConsistencyAnswer), and a chunk whose answer an earlier run kept
// is not sent again, so a fallback part-way through a chunk set loses no
// answer already paid for: the host ingest takes the kept files beside the
// chunks it runs, and the next provider run sends only the chunks not yet
// answered. An answer is ingested only with the whole set, because the report
// covers every chunk or none.
func sendConsistency(cmd *cobra.Command, rf *routeFlag, route *oracle.Route, repoRoot string, res intent.ConsistencyEmitResult) ([]*dispatched, *oracle.Route, error) {
	const verb = "abcd intent consistency"
	stderr := cmd.ErrOrStderr()
	if res.Overflow == nil {
		d, fell, err := rf.sendRequest(cmd, verb, route, auditAgent, repoRoot, res.RequestPath, res.CorpusPath)
		if err != nil || d == nil {
			return nil, fell, err
		}
		return []*dispatched{d}, fell, nil
	}
	ds := make([]*dispatched, len(res.Chunks))
	kept := newKeptChunks(rf, route, repoRoot)
	reused, sendBytes := 0, 0
	for i, ch := range res.Chunks {
		if d, why := kept.reuse(ch.ReceiptID); d != nil {
			ds[i] = d
			reused++
			continue
		} else if why != "" {
			fmt.Fprintf(stderr, "%s: the answer kept for chunk %d of %d is not reused, so the chunk is sent again: %s\n",
				verb, ch.Index, len(res.Chunks), why)
		}
		sendBytes += ch.Bytes
	}
	fmt.Fprintf(stderr, "%s: sending %d chunks, %d bytes of requests and corpus files in all, one after another to %s",
		verb, len(res.Chunks)-reused, sendBytes, termsafe.Sanitize(route.ConnectionUsed))
	if reused > 0 {
		fmt.Fprintf(stderr, " (%d answered on an earlier run, whose kept answers are reused)", reused)
	}
	fmt.Fprintln(stderr)
	for i, ch := range res.Chunks {
		if ds[i] != nil {
			continue
		}
		d, fell, err := rf.sendRequest(cmd, verb, route, auditAgent, repoRoot, ch.RequestPath, ch.CorpusPath)
		if err != nil {
			return nil, fell, err
		}
		if d == nil {
			var answered []string
			for j, a := range ds {
				if a != nil {
					answered = append(answered, intent.ConsistencyAnswerPath(res.Chunks[j].ReceiptID))
				}
			}
			fmt.Fprintf(stderr, "%s: chunk %d of %d was not sent, so all %d chunks are left to the host: "+
				"run each chunk's request and ingest every chunk's findings together", verb, ch.Index, len(res.Chunks), len(res.Chunks))
			if len(answered) > 0 {
				fmt.Fprintf(stderr, "; %d chunk(s) are answered already, kept at %s: give those files to the ingest for their chunks, "+
					"or run again to send only the chunks not yet answered", len(answered), strings.Join(answered, ", "))
			}
			fmt.Fprintln(stderr)
			return nil, fell, nil
		}
		rcpt, err := json.Marshal(d.receipt)
		if err == nil {
			_, err = intent.KeepConsistencyAnswer(repoRoot, ch.ReceiptID, d.payload, rcpt)
		}
		if err != nil {
			fmt.Fprintf(stderr, "%s: warning: chunk %d of %d's answer was not kept, so a later fallback would ask for it again: %s\n",
				verb, ch.Index, len(res.Chunks), termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		}
		ds[i] = d
	}
	return ds, route, nil
}

// keptChunks judges, once per chunk, whether the answer an earlier run kept
// for it is reused on this route: only when it still validates and the call
// that produced it went to the connection and the model the route sends to
// now. An answer from another provider or model is the answer to another
// question, so its chunk is sent again.
type keptChunks struct {
	kept   *intent.KeptConsistencyAnswers
	conn   string
	target oracle.Target
	judged map[string]keptJudgement
}

type keptJudgement struct {
	d   *dispatched
	why string // why a kept answer is not reused; "" when none is kept
}

func newKeptChunks(rf *routeFlag, route *oracle.Route, repoRoot string) *keptChunks {
	k := &keptChunks{kept: intent.NewKeptConsistencyAnswers(repoRoot), conn: route.ConnectionUsed, judged: map[string]keptJudgement{}}
	if rf.api != nil {
		if t, err := rf.api.Admitted(*route); err == nil {
			k.target = t
		}
	}
	return k
}

// reuse is the kept answer for chunk receipt rcp as a dispatch, or nil with
// the reason it is not reused ("" when nothing is kept).
func (k *keptChunks) reuse(rcp string) (*dispatched, string) {
	j, ok := k.judged[rcp]
	if !ok {
		j = k.judge(rcp)
		k.judged[rcp] = j
	}
	return j.d, j.why
}

func (k *keptChunks) judge(rcp string) keptJudgement {
	payload, raw, ok, err := k.kept.Get(rcp)
	if err != nil {
		return keptJudgement{why: termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	if !ok {
		return keptJudgement{}
	}
	var rr oracle.ReceiptRoute
	if err := json.Unmarshal(raw, &rr); err != nil {
		return keptJudgement{why: "the record of the call that produced it is not readable"}
	}
	asked := ""
	if rr.ProviderCall != nil {
		asked = rr.ProviderCall.ModelAsked
	}
	if k.target.Model == "" || rr.ProviderCall == nil || rr.ConnectionUsed != k.conn ||
		rr.ProviderCall.Provider != k.target.Provider || asked != k.target.Model {
		return keptJudgement{why: termsafe.Sanitize(fmt.Sprintf("it was answered through %s by %s, and the route now sends to %s with %s",
			orUnknown(rr.ConnectionUsed), orUnknown(asked), k.conn, orUnknown(k.target.Model)))}
	}
	return keptJudgement{d: &dispatched{payload: payload, receipt: rr}}
}

func orUnknown(s string) string {
	if s == "" {
		return "an unrecorded one"
	}
	return s
}

// withDispatchReceipts joins the first send's receipt as the verb's route, as
// one send's is joined, and, for a chunked pass, every chunk's receipt in
// chunk order beside it.
func withDispatchReceipts(v any, ds []*dispatched) any {
	out := withDispatchReceipt(v, ds[0])
	if len(ds) == 1 {
		return out
	}
	all := make([]oracle.ReceiptRoute, len(ds))
	for i, d := range ds {
		all[i] = d.receipt
	}
	return withMember{v: out, key: "chunk_routes", val: all}
}

// renderConsistencyOverflow is an overflow's line in the emit's and the
// ingest's text: the corpus's size against the limit, the plan, and the
// documents no chunk could hold.
func renderConsistencyOverflow(w io.Writer, ov *intent.ConsistencyOverflow) {
	unreviewed := "none"
	if len(ov.Unreviewed) > 0 {
		parts := make([]string, len(ov.Unreviewed))
		for i, u := range ov.Unreviewed {
			parts[i] = fmt.Sprintf("%s (%d bytes)", termsafe.Sanitize(u.Path), u.Bytes)
		}
		unreviewed = strings.Join(parts, ", ")
	}
	fmt.Fprintf(w, "  overflow: the corpus is %d bytes, over the request limit of %d bytes: %d groups, %d chunks; unreviewed: %s\n",
		ov.CorpusBytes, ov.LimitBytes, ov.Groups, ov.Chunks, unreviewed)
}

// warnUnreviewed warns on stderr, naming each with its framed size, of every
// document an overflowing plan put in no chunk: the pass reads none of them,
// and a run that still succeeds would otherwise say so only in its report.
// reviews is the verb's tense ("reviews" before the pass, "reviewed" after).
func warnUnreviewed(w io.Writer, verb, reviews string, ov *intent.ConsistencyOverflow) {
	if ov == nil || len(ov.Unreviewed) == 0 {
		return
	}
	parts := make([]string, len(ov.Unreviewed))
	for i, u := range ov.Unreviewed {
		parts[i] = fmt.Sprintf("%s (%d bytes)", termsafe.Sanitize(u.Path), u.Bytes)
	}
	what, it := "1 document is", "it"
	if len(parts) > 1 {
		what, it = fmt.Sprintf("%d documents are", len(parts)), "them"
	}
	fmt.Fprintf(w, "%s: warning: %s too large for any chunk, so no chunk %s %s: %s; raise consistency.max_request_bytes to read %s\n",
		verb, what, reviews, it, strings.Join(parts, ", "), it)
}

// ingestConsistency validates a consistency pass's findings, files them, and
// renders the result with receipt joined to it and receiptLine closing its
// text, for the ingest verb and for a pass a provider ran. payloads is one
// findings payload, or one per chunk of an overflowing emit.
func ingestConsistency(cmd *cobra.Command, repoRoot string, payloads [][]byte, receipt func(any) any, receiptLine func(io.Writer), asJSON bool) error {
	// The filing-time match (itd-2609212137116617) on every record the
	// pass files, configured as the capture verb's is and never a
	// refusal: a refused configuration files unmatched and says why.
	mc, matchRefused := resolveMatch(cmd.ErrOrStderr(), "intent consistency ingest", repoRoot)
	var res intent.ConsistencyIngestResult
	var err error
	if len(payloads) == 1 {
		res, err = capture.IngestConsistency(repoRoot, payloads[0], "", mc)
	} else {
		res, err = capture.IngestConsistencyChunks(repoRoot, payloads, "", mc)
	}
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd intent consistency ingest: " + fsutil.RedactHome(err.Error())}
	}
	warnUnreviewed(cmd.ErrOrStderr(), "abcd intent consistency ingest", "reviewed", res.Overflow)
	// The set is in the ledger (or was already): the answers kept for its
	// chunks have done their work, and a later run asks the model again.
	if len(res.Chunks) > 0 {
		rcps := make([]string, len(res.Chunks))
		for i, ch := range res.Chunks {
			rcps[i] = ch.ReceiptID
		}
		if err := intent.ForgetConsistencyAnswers(repoRoot, rcps); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd intent consistency ingest: warning: the answers kept for this set were not all removed, "+
				"so a later provider run on the same model may reuse them: %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		}
	}
	for i := range res.Rows {
		if res.Rows[i].Match == nil && res.Rows[i].State == intent.ConsistencyStateFiled {
			res.Rows[i].Match = matchRefused
		}
	}
	return render(cmd.OutOrStdout(), asJSON, receipt(res), func(w io.Writer) {
		fmt.Fprintf(w, "abcd intent consistency ingest — %s (receipt %s, scope %s)\n", res.Status, res.ReceiptID, res.Scope)
		dirty := ""
		if res.Dirty {
			dirty = ", dirty"
		}
		fmt.Fprintf(w, "  report: %s (read %s%s)\n", res.ReportPath, res.ReviewOfCommit, dirty)
		if res.Overflow != nil {
			renderConsistencyOverflow(w, res.Overflow)
		}
		if res.Status == "ingested" {
			fmt.Fprintf(w, "  findings %d: filed %d · linked to an open record %d · addressed by a closed record %d\n",
				res.Findings, len(res.Filed), len(res.Linked), len(res.Addressed))
			if res.SetAside > 0 {
				fmt.Fprintf(w, "  set aside: %d finding(s) a chunk reported on a pair another chunk owns\n", res.SetAside)
			}
			for _, r := range res.Rows {
				fmt.Fprintf(w, "  %d. %s (%s) — %s, key %s: %s\n", r.Number, r.ClassLabel(), r.Severity, r.LedgerCell(), r.Key,
					termsafe.Sanitize(r.Summary))
				renderMatch(w, r.Match)
			}
		}
		receiptLine(w)
	})
}
