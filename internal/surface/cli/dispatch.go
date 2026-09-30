package cli

// dispatch.go sends a delegated step to the provider its route resolved to
// (spc-2609251028149555 AC 3, itd-2609081951381895 criterion 1): the agent's
// own prompt as the instructions, the request the verb emitted as the input,
// and the answer judged by the verb's own ingest. The receipt names the
// provider as the connection used and carries the call's record, so a host
// reading a receipt whose connection_used is not the harness knows the step
// already ran and must not dispatch the agent itself.
//
// Which agents a paid provider may take is ruling DR5 of 2026-09-29, applied
// by oracle.(*APIConfig).Admitted before anything is written and again by
// Dispatch before any call. A provider that could not be reached at all
// leaves the step to the harness with one line on stderr; any other failure
// exits 2 before anything is written.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// dispatchCredentials is the credential source a dispatch resolves the key
// through: the person's own store. A variable so a test can count reads.
var dispatchCredentials = credential.UserStore

// maxAgentPromptBytes bounds an agent's prompt file read for a dispatch.
const maxAgentPromptBytes = 1 << 20

// dispatched is one step a provider ran: the payload its contract admitted and
// the receipt Dispatch returned.
type dispatched struct {
	payload []byte
	receipt oracle.ReceiptRoute
}

// admit refuses, at exit 2, a step on a provider that Dispatch would refuse,
// so a verb calls it before it writes anything. A step the host runs passes.
func (rf *routeFlag) admit(verb string, r *oracle.Route) error {
	if r == nil || !r.OnProvider() {
		return nil
	}
	if rf.api == nil {
		return &exitError{Code: 2, Msg: verb + ": the route names provider " + termsafe.Sanitize(r.ConnectionUsed) +
			" and no provider configuration was read"}
	}
	if _, err := rf.api.Admitted(*r); err != nil {
		return &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return nil
}

// dispatch sends the step when r is on a provider and returns what it ran.
// It returns a nil *dispatched when the host runs the step: r is on the
// harness, or its provider could not be reached (nothing was sent), in which
// case the fallback line is printed on stderr and the returned route is r
// moved to the harness.
func (rf *routeFlag) dispatch(cmd *cobra.Command, verb string, r *oracle.Route, brief openaiapi.Brief,
	contract func([]byte) error) (*dispatched, *oracle.Route, error) {
	if r == nil || !r.OnProvider() {
		return nil, r, nil
	}
	if err := rf.admit(verb, r); err != nil {
		return nil, r, err
	}
	payload, rc, err := rf.api.Dispatch(cmd.Context(), dispatchCredentials(), *r, brief, contract)
	if err != nil {
		if fb, ok := r.FellBack(err); ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s: %s\n", verb, termsafe.Sanitize(fsutil.RedactHome(fb.Fallback)))
			return nil, &fb, nil
		}
		return nil, r, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return &dispatched{payload: payload, receipt: rc}, r, nil
}

// hostPayloadOnProvider refuses an ingest handed a payload the host produced
// while the route sends the agent to a provider: abcd sends such a step
// itself, and a receipt naming the provider for work the host did would be
// false. --route <agent>=host-decides keeps one run on the harness.
func hostPayloadOnProvider(verb string, r *oracle.Route, instead string) error {
	if r == nil || !r.OnProvider() {
		return nil
	}
	agent := termsafe.Sanitize(r.Agent)
	return &exitError{Code: 2, Msg: fmt.Sprintf("%s: %s is routed to provider %s, so abcd sends the step there itself, "+
		"and a payload the host produced is refused rather than recorded as the provider's: %s, or pass --route %s=%s "+
		"to keep this run on the %s", verb, agent, termsafe.Sanitize(r.ConnectionUsed), instead, agent, oracle.HostDecides, oracle.Harness)}
}

// withDispatchReceipt joins the receipt Dispatch returned to a verb's result.
func withDispatchReceipt(v any, d *dispatched) any {
	return withMember{v: v, key: "route", val: d.receipt}
}

// renderDispatchLine is a dispatched step's receipt line.
func renderDispatchLine(w interface{ Write([]byte) (int, error) }, d *dispatched) {
	rc := d.receipt
	line := fmt.Sprintf("route: %s ran on %s", rc.Agent, rc.ConnectionUsed)
	if rc.ProviderCall != nil {
		line += fmt.Sprintf(", model asked %s, model reported %s", rc.ProviderCall.ModelAsked, rc.ProviderCall.ModelReported)
	}
	fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(line))
}

// agentPrompt is the prompt the host sub-agent gets for agent: its file under
// the plugin root's agents/, read through a guarded open.
func agentPrompt(agent string) (string, error) {
	root, ok := ahoy.ResolvePluginRoot()
	if !ok {
		return "", fmt.Errorf("the plugin root that holds agents/%s.md could not be resolved, so the agent's prompt cannot be sent", agent)
	}
	return readAgentFile(filepath.Join(root, "agents"), agent+".md")
}

// readAgentFile reads name inside dir through an os.Root, so a symlinked leaf
// cannot aim the read elsewhere.
func readAgentFile(dir, name string) (string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", fmt.Errorf("opening the agents directory: %w", err)
	}
	defer root.Close()
	raw, err := fsutil.ReadGuardedInRoot(root, name, maxAgentPromptBytes)
	if err != nil {
		return "", fmt.Errorf("reading the agent prompt %s: %w", name, err)
	}
	return string(raw), nil
}

// sendRequest dispatches agent with the files an emit wrote as its input:
// each read through a guarded open under repoRoot, joined in order. It
// returns a nil *dispatched when the host runs the step (see dispatch).
func (rf *routeFlag) sendRequest(cmd *cobra.Command, verb string, route *oracle.Route, agent, repoRoot string,
	files ...string) (*dispatched, *oracle.Route, error) {
	prompt, err := agentPrompt(agent)
	if err != nil {
		return nil, route, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	var input strings.Builder
	for _, rel := range files {
		body, err := readAgentFile(filepath.Join(repoRoot, filepath.Dir(filepath.FromSlash(rel))), filepath.Base(rel))
		if err != nil {
			return nil, route, &exitError{Code: 2, Msg: verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
		}
		if input.Len() > 0 {
			input.WriteString("\n\n")
		}
		fmt.Fprintf(&input, "=== %s ===\n%s", rel, body)
	}
	return rf.dispatch(cmd, verb, route, openaiapi.Brief{Instructions: prompt, Input: input.String()}, jsonContract)
}

// dispatchAudit sends the review request an audit emit wrote to the provider
// the auditor is routed to, ingests the verdict it returns, and renders the
// ingest's result with the dispatch's receipt. A provider that could not be
// reached leaves the written request to the host, as an emit on the harness
// does, and the request block names the harness.
func dispatchAudit(cmd *cobra.Command, rf *routeFlag, route *oracle.Route, repoRoot string, res intent.AuditEmitResult, asJSON bool) error {
	const verb = "abcd intent audit"
	d, fell, err := rf.sendRequest(cmd, verb, route, auditAgent, repoRoot, res.RequestPath)
	if err != nil {
		return err
	}
	if d == nil {
		return render(cmd.OutOrStdout(), asJSON, withRequest(res, fell), func(w io.Writer) {
			fmt.Fprintf(w, "abcd intent audit — %s %s (receipt %s)\n  request: %s\n", res.IntentID, res.Status, res.ReceiptID, res.RequestPath)
			renderRequestLine(w, fell)
		})
	}
	ing, err := intent.IngestVerdictBytes(repoRoot, d.payload)
	if err != nil {
		return &exitError{Code: 2, Msg: verb + ": the answer from " + termsafe.Sanitize(d.receipt.ConnectionUsed) + ": " +
			termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return render(cmd.OutOrStdout(), asJSON, withDispatchReceipt(ing, d), func(w io.Writer) {
		fmt.Fprintf(w, "abcd intent audit — %s (receipt %s, intent %s)\n", ing.Status, ing.ReceiptID, ing.IntentID)
		renderAuditOwed(w, ing)
		renderDispatchLine(w, d)
	})
}

// jsonContract admits an answer that is one JSON document; the verb's own
// ingest then judges it whole.
func jsonContract(b []byte) error {
	if !json.Valid(b) {
		return errors.New("the answer is not one JSON document")
	}
	return nil
}

// disembarkNoDispatch is what a disembark ingest routed to a provider tells
// the person: the four lifeboat agents read the packed lifeboat, which is
// files, and no verb emits a request that carries them, so none is sent to a
// provider; the host runs them.
const disembarkNoDispatch = "the lifeboat agents read the packed lifeboat's files, and abcd builds no request carrying them, " +
	"so it sends none of them to a provider: remove the oracle.roles entry for the agent from ~/.abcd/config.json"
