package oracle

// dispatch.go sends one delegated step through the provider connection its
// route resolved to (spc-2609251028149555 AC 3, the dispatch half, and
// itd-2609081951381895 criterion 1): the brief the host sub-agent would get,
// the settings as Resolve merged and checked them, the key resolved by name,
// and the answer judged by the same output contract. The receipt names the
// provider as the connection used and carries the call's record.
//
// It is the core entry a delegating verb calls once Resolve has returned a
// provider leg (Route.OnProvider). It never prints and never writes; the
// front door prints the fallback line, formats a refusal and exits 2.

import (
	"context"
	"errors"
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// OnProvider reports whether r's step goes to a provider connection rather
// than the harness.
func (r Route) OnProvider() bool { return r.ConnectionUsed != "" && r.ConnectionUsed != Harness }

// Dispatch sends one step through r's provider connection and returns the
// payload the contract admitted and the receipt's route block, which names
// the provider as tried and used and carries the call's record.
//
// Everything that decides where the step goes and with which key is taken
// again from this configuration, never from r alone: the model is the one
// oracle.roles.<agent> points at on r's connection, and a provider that holds
// a key is reached only through a route set on this machine, so only a route
// the person set up on their own machine spends their key (ruling AA(b) of
// 2026-09-29, itd-2609081951381895 Decision 8). A provider that holds a key
// takes only a self-contained agent, or one the person's override admits
// (ruling DR5 of 2026-09-29, admitAgent). Call then admits the target
// against the allowlist and oracle.denylist again and resolves the key by
// name. Every refusal is made before the provider is contacted and names the
// setting to change. An error that is openaiapi.ErrUnreachable means nothing
// was sent, and FellBack gives the route that leaves the step to the harness.
func (c *APIConfig) Dispatch(ctx context.Context, creds credential.Source, r Route, brief openaiapi.Brief,
	contract func([]byte) error, opts ...openaiapi.Option) ([]byte, ReceiptRoute, error) {
	agent := termsafe.Sanitize(layered.BoundKey(r.Agent))
	if !r.OnProvider() {
		return nil, ReceiptRoute{}, fmt.Errorf("oracle dispatch: %s resolves to the %s, which the host runs, so there is no provider "+
			"to send it to; hand the step to the host, or point oracle.roles.%s at <provider>/<model> in %s to send it to a provider",
			agent, Harness, agent, layered.Config.MachineOrigin())
	}
	conn := termsafe.Sanitize(layered.BoundKey(r.ConnectionUsed))
	t, ok := c.roles[r.Agent]
	if !ok || t.Provider != r.ConnectionUsed {
		return nil, ReceiptRoute{}, fmt.Errorf("oracle dispatch: %s resolves to connection %s, but this machine's configuration "+
			"does not point oracle.roles.%s at %s, so the step names no model there and is refused rather than sent: "+
			"point oracle.roles.%s at %s/<model> in %s, or route %s to the harness with tier %s",
			agent, conn, agent, conn, agent, conn, layered.Config.MachineOrigin(), agent, HostDecides)
	}
	if p := c.providers[t.Provider]; keyed(p) && t.Origin != layered.Config.MachineOrigin() {
		return nil, ReceiptRoute{}, fmt.Errorf("oracle dispatch: oracle.roles.%s points at %s, a provider that holds a key "+
			"(its block names the credential %s), and the route comes from %s; only a route set on this machine may spend "+
			"that key, so the step is refused before any call: set oracle.roles.%s in %s",
			agent, conn, p.Key, termsafe.Sanitize(t.Origin), agent, layered.Config.MachineOrigin())
	}
	if err := c.admitAgent(r.Agent, c.providers[t.Provider]); err != nil {
		return nil, ReceiptRoute{}, fmt.Errorf("oracle dispatch: %s through %s: %w", agent, conn, err)
	}
	payload, rec, err := c.Call(ctx, creds, CallRequest{Target: t, Brief: brief, Settings: r.SettingsSent, Contract: contract}, opts...)
	if err != nil {
		return nil, ReceiptRoute{}, fmt.Errorf("oracle dispatch: %s through %s: %w", agent, conn, err)
	}
	return payload, r.Receipt(ModelReported(payload)).WithCall(rec), nil
}

// FellBack returns r moved to the harness when err says its provider could
// not be reached (openaiapi.ErrUnreachable: nothing was sent), and false for
// any other error, which refuses the step instead. The step then runs through
// the harness with its tier named in the request, the receipt records the
// connection tried and the harness used, and Fallback is the one line the
// front door prints on stderr before the step runs (itd-2609170822093401, the
// fourth criterion, at dispatch).
func (r Route) FellBack(err error) (Route, bool) {
	if !r.OnProvider() || !errors.Is(err, openaiapi.ErrUnreachable) {
		return r, false
	}
	r.Fallback = fmt.Sprintf("connection %s could not be reached, so %s runs through the %s with tier %s named in its request (%s)",
		r.ConnectionUsed, r.Agent, Harness, r.Row.Tier, termsafe.Sanitize(err.Error()))
	r.ConnectionTried, r.ConnectionUsed = r.ConnectionUsed, Harness
	r.SettingsSent = nil
	return r, true
}
