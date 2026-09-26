package oracle

// call.go is one call through the OpenAI-compatible API adapter
// (itd-2609081951381895 criteria 1, 4 and 5): a role or a judgement type
// pointed at a listed model is sent the brief the host sub-agent would get,
// with its key resolved by name through internal/core/credential, the answer
// judged by the same output contract, and the call recorded as the provider,
// the model asked for and the model the provider reported.
//
// No delegating verb dispatches through it yet: sending a step whose route
// names a provider through its connection, instead of handing it to the host,
// is spc-2609251028149555's (AC 3). Until then the setup's verification call
// is its one caller from a front door.

import (
	"context"
	"errors"
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// CallRecord is the per-call record the run record carries (criterion 5,
// adr-2609221009491186 Decision 5): the provider, the model asked for and the
// model the provider reported, side by side, so a substitution is visible. It
// never carries a key, a key's name or the brief.
type CallRecord struct {
	Provider      string `json:"provider"`
	ModelAsked    string `json:"model_asked"`
	ModelReported string `json:"model_reported"`
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
// answer whose reported model the denylist refuses is discarded: the provider
// substituted a frontier model, and the refusal names what it reported.
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
	return complete(ctx, p.Name, p.BaseURL, key, t.Model, req.Brief, req.Settings, req.Contract, c.denylist, opts...)
}

// resolveKey resolves a provider's key by name; a keyless block resolves to
// "" and sends none.
func resolveKey(creds credential.Source, p Provider) (string, error) {
	if p.Key == "" {
		return "", nil
	}
	if creds == nil {
		creds = credential.UserMachine()
	}
	key, err := creds.Resolve(p.Key)
	switch {
	case errors.Is(err, credential.ErrNotSet):
		return "", fmt.Errorf("oracle adapter: provider %s names credential %q, which is not set on this machine, so no call is made; "+
			"`abcd ahoy connect` stores one (`abcd ahoy --providers` explains where it can live)", p.Name, p.Key)
	case err != nil:
		return "", fmt.Errorf("oracle adapter: provider %s: %w", p.Name, err)
	}
	return key, nil
}

// complete is the one place a call is made: the client built on the pinned
// base URL, the answer read, and the reported model held to the denylist.
func complete(ctx context.Context, provider, baseURL, key, model string, brief openaiapi.Brief, settings Settings,
	contract func([]byte) error, denylist []DenyEntry, opts ...openaiapi.Option) ([]byte, CallRecord, error) {
	client, err := openaiapi.New(baseURL, key, opts...)
	if err != nil {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s: %w", provider, err)
	}
	res, err := client.Complete(ctx, openaiapi.Request{Model: model, Brief: brief, Settings: settings}, contract)
	if err != nil {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s, model %s: %w", provider, model, err)
	}
	if e, denied := Denied(denylist, res.ModelReported); denied {
		return nil, CallRecord{}, fmt.Errorf("oracle adapter: provider %s was asked for %s and reported answering with %s, "+
			"which the vendor denylist refuses (%s, from %s); the answer is discarded", provider, model, res.ModelReported, e.Pattern, e.Origin)
	}
	return res.Content, CallRecord{Provider: provider, ModelAsked: res.ModelAsked, ModelReported: res.ModelReported}, nil
}
