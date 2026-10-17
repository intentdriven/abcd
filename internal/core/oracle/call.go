package oracle

// call.go is one call through the OpenAI-compatible API adapter
// (itd-2609081951381895 criteria 1, 4 and 5): a role or a judgement type
// pointed at a listed model is sent the brief the host sub-agent would get,
// with its key resolved by name through internal/core/credential, the answer
// judged by the same output contract, and the call recorded as the provider,
// the model asked for and the model the provider reported.
//
// Dispatch (dispatch.go) sends a step whose route names a provider through
// it, instead of handing the step to the host (spc-2609251028149555, AC 3).
// The delegating verbs call Dispatch, and the setup's verification call
// reaches Call directly.

import (
	"context"
	"errors"
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// CallRecord is the per-call record the run record carries (criterion 5,
// adr-2609221009491186 Decision 5): the provider, the model asked for and the
// model the provider reported, side by side, so a substitution is visible, and
// the name of the credential the call used. It never carries a key or the
// brief.
//
// It carries both sides of the call (itd-166): the asked side beside the
// settings the receipt records as sent (the prompt's SHA-256 and the request
// fields the adapter added on its own), and the served side as the response
// named it (the upstream provider, the generation id, the finish reason, the
// token counts and the cost). Every string the provider supplied is scrubbed,
// cleaned and bounded by the adapter (openaiapi's admit). Unpinned lists each
// field of the fixed lists the call could not pin, with its reason.
type CallRecord struct {
	Provider      string `json:"provider"`
	ModelAsked    string `json:"model_asked"`
	ModelReported string `json:"model_reported"`
	// Credential is the name of the credential the call used, never its
	// value; empty for a provider that takes no key (itd-2609221017023290
	// criterion 5).
	Credential string `json:"credential,omitempty"`
	// PromptSHA256 is the SHA-256 of the rendered messages, never their text
	// (decision 5).
	PromptSHA256 string `json:"prompt_sha256"`
	// AdapterAdded names the request fields the adapter sent on its own
	// (openaiapi.StreamUsageField), which are not settings sent.
	AdapterAdded []string `json:"adapter_added"`
	// ServedProvider is the upstream provider an aggregator names as the one
	// that served the call; "" when the response names none.
	ServedProvider string `json:"served_provider"`
	// GenerationID is the response's own id; "" when it names none.
	GenerationID string `json:"generation_id"`
	// FinishReason is why the provider stopped; "" when it gave none.
	FinishReason string `json:"finish_reason"`
	// Usage is the token counts the response reported, nil when it reported
	// none; a count it did not report is listed in Unpinned.
	Usage *Usage `json:"usage"`
	// Cost is the usage member's cost, in credits; nil when not reported.
	Cost *Cost `json:"cost"`
	// Unpinned lists every field of the fixed lists the call could not pin.
	Unpinned []Unpinned `json:"unpinned"`
}

// CallRequest is one call: where it goes, the brief, the settings as sent and
// the output contract the answer is judged by.
type CallRequest struct {
	Target   Target
	Brief    openaiapi.Brief
	Settings Settings
	// Contract is the output contract the host sub-agent's payload is judged
	// by; nil admits any answer.
	Contract func([]byte) error
}

// Call sends req through its target's provider and returns the answer the
// contract admitted and the call's record. It refuses before any call a
// target the configuration does not admit (Admit, again, so a target built
// anywhere but the read is held to the same rule) and a named key that
// resolves to nothing, because an unauthenticated call is never made. An
// answer whose reported model an oracle.denylist entry refuses is discarded:
// the provider substituted a model the configuration refuses, and the refusal
// names what it reported.
func (c *APIConfig) Call(ctx context.Context, creds credential.Source, req CallRequest, opts ...openaiapi.Option) ([]byte, CallRecord, error) {
	t := req.Target
	if err := c.Admit(t.Provider, t.Model); err != nil {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: %s/%s is refused before any call: %w", t.Provider, t.Model, err)
	}
	p := c.providers[t.Provider]
	key, err := resolveKey(creds, p)
	if err != nil {
		return nil, CallRecord{}, err
	}
	payload, rec, err := complete(ctx, p, key, t.Model, req.Brief, req.Settings, req.Contract, c.denylist, opts...)
	if err == nil {
		rec.Credential = p.Key
	}
	return payload, rec, err
}

// resolveKey resolves a provider's key by name; a keyless block resolves to
// "" and sends none.
func resolveKey(creds credential.Source, p Provider) (string, error) {
	if p.Key == "" {
		return "", nil
	}
	if creds == nil {
		creds = credential.UserStore()
	}
	key, err := creds.Resolve(p.Key)
	switch {
	case errors.Is(err, credential.ErrNotSet):
		return "", fmt.Errorf("oracle adapter: provider %s names credential %q, which is not set on this machine, so no call is made; "+
			"`%s` explains where it can live and stores it", p.Name, p.Key, credential.Walkthrough(p.Key))
	case err != nil:
		return "", fmt.Errorf("oracle adapter: provider %s: %w", p.Name, err)
	}
	return key, nil
}

// complete is the one place a call is made: the client built on the pinned
// base URL, the answer read, and the reported model held to oracle.denylist.
// The request asks for usage in the stream unless p's block turns that off;
// a server that refuses such a request is refused naming the setting, and is
// never asked again without it, since a retry is a second request.
func complete(ctx context.Context, p Provider, key, model string, brief openaiapi.Brief, settings Settings,
	contract func([]byte) error, denylist []DenyEntry, opts ...openaiapi.Option) ([]byte, CallRecord, error) {
	provider := p.Name
	client, err := openaiapi.New(p.BaseURL, key, opts...)
	if err != nil {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s: %w", provider, err)
	}
	req := openaiapi.Request{Model: model, Brief: brief, Settings: settings, NoStreamUsage: !p.AsksStreamUsage()}
	res, err := client.Complete(ctx, req, contract)
	if errors.Is(err, openaiapi.ErrStreamUsage) {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s, model %s: %w; if this server does not take %s, "+
			"set %s.%s.%s to false in %s, and its calls are sent without it, their usage recorded as not reported",
			provider, model, err, openaiapi.StreamUsageField, apiKey, provider, streamUsageKey, layered.Config.MachineOrigin())
	}
	if err != nil {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s, model %s: %w", provider, model, err)
	}
	if e, denied := Denied(denylist, res.ModelReported); denied {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s was asked for %s and reported answering with %s, "+
			"which %s refuses (%s, from %s); the answer is discarded", provider, model, res.ModelReported, denylistKey, e.Pattern, e.Origin)
	}
	return res.Content, callRecord(p, brief, settings, res), nil
}

// callRecord is the call's both sides, built from the request this layer sent
// and the response the adapter read: nothing in it was asked of the model,
// and building it sends nothing (criterion 8).
func callRecord(p Provider, brief openaiapi.Brief, settings Settings, res openaiapi.Result) CallRecord {
	rec := CallRecord{Provider: p.Name, ModelAsked: res.ModelAsked, ModelReported: res.ModelReported,
		PromptSHA256: openaiapi.PromptSHA256(brief), AdapterAdded: []string{},
		ServedProvider: res.Provider, GenerationID: res.GenerationID, FinishReason: res.FinishReason}
	if p.AsksStreamUsage() {
		rec.AdapterAdded = append(rec.AdapterAdded, openaiapi.StreamUsageField)
	}
	var un Unpin
	// The asked side: the model and the prompt are always the call's own. No
	// provider pin exists yet (iss-2610161734413178), no setting names an
	// effort or a context variant, and a sampling setting not sent leaves the
	// server's default, whatever it is.
	un.Add(AskedProviderPin, NotConfigured)
	for _, s := range []struct {
		f    Field
		name string
	}{{AskedTemperature, "temperature"}, {AskedTopP, "top_p"}, {AskedSeed, "seed"}} {
		_, sent := settings[s.name]
		un.Unless(sent, s.f, NotConfigured)
	}
	un.Add(AskedEffort, NotConfigured)
	un.Add(AskedContext, NotConfigured)
	un.Add(AskedHarnessVersion, NotReported)
	if openaiapi.IsLocal(p.BaseURL) {
		for _, f := range LocalFields {
			un.Add(f, NotConfigured)
		}
	}
	// The served side: the adapter refuses an answer that names no model, so
	// the model is always the provider's own report.
	un.Unless(res.Provider != "", ServedProvider, NotReported)
	un.Unless(res.GenerationID != "", ServedGenerationID, NotReported)
	un.Unless(res.FinishReason != "", ServedFinishReason, NotReported)
	u := res.Usage
	un.Unless(u != nil, ServedPromptTokens, NotReported)
	un.Unless(u != nil, ServedCompletionTokens, NotReported)
	un.Unless(u != nil && u.ReasoningTokens != nil, ServedReasoningTokens, NotReported)
	un.Unless(u != nil && u.CachedTokens != nil, ServedCacheReadTokens, NotReported)
	un.Unless(u != nil && u.CacheWriteTokens != nil, ServedCacheWriteTokens, NotReported)
	un.Unless(u != nil && u.Cost != nil, ServedCost, NotReported)
	if u != nil {
		deref := func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		}
		rec.Usage = &Usage{PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens,
			ReasoningTokens: deref(u.ReasoningTokens), CacheReadTokens: deref(u.CachedTokens), CacheWriteTokens: deref(u.CacheWriteTokens)}
		if u.Cost != nil {
			rec.Cost = &Cost{Amount: *u.Cost, Unit: CostCredits}
		}
	}
	rec.Unpinned = un.List()
	return rec
}
