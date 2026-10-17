package oracle

// pin.go holds an OpenRouter call to one named upstream provider
// (iss-2610161734413178, itd-2610161733316518 decisions 5, 6 and 12):
//
//   - oracle.api.<provider>.pins.<model> pins a model the block lists to one
//     provider (order, exactly one), one quantisation (quantizations, exactly
//     one) and its permanent slug (canonical_slug). The pin's other three
//     fields are not configurable: fallbacks are off (allow_fallbacks false,
//     decision 6: a fallback changes the cell), every parameter sent must be
//     honoured (require_parameters true) and the provider may not keep the
//     prompt (data_collection deny). It is read from the machine's file
//     alone, as the block it sits in is, and only on a block whose base URL
//     is OpenRouter's, since the routing object is OpenRouter's and a server
//     that ignored it would leave the record claiming a pin nothing held.
//   - A call to a pinned model sends the pin as OpenRouter's provider routing
//     object and records it on the asked side (CallRecord.ProviderPin); the
//     opencode runner writes the same object into a sealed per-run
//     configuration (internal/core/runner).
//   - The generation readback reads OpenRouter's record of the call into the
//     served side: the provider that served, every provider attempted, the
//     native token counts and the cost. It is a request of its own, so it is
//     made only when the caller asks (CallRequest.ReadGeneration), which a run
//     the person started and consented to does; it is off by default, made
//     with the key the call used (resolved through the one credential reader,
//     never again), bounded in time, and its failure never fails the call.
//
// Exactly one provider, because with fallbacks off OpenRouter still tries
// each provider order names, so a second entry is a fallback by another name;
// exactly one quantisation, because one provider can serve a model at more
// than one, which is a second endpoint behind the same name.

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// pinsKey is a provider block's member that pins its models.
const pinsKey = "pins"

// openRouterHost is the host a pin and a readback are OpenRouter's at. A
// variable so a test can point it at its fake on the loopback address.
var openRouterHost = "openrouter.ai"

// isOpenRouter reports whether baseURL reaches OpenRouter.
func isOpenRouter(baseURL string) bool {
	u, err := url.Parse(baseURL)
	return err == nil && strings.EqualFold(u.Hostname(), openRouterHost)
}

// quantisations are the values OpenRouter's routing object takes in
// quantizations.
var quantisations = []string{"int4", "int8", "fp4", "fp6", "fp8", "fp16", "bf16", "fp32", "unknown"}

// providerSlugRe is a provider as OpenRouter's order names one (deepinfra,
// google-vertex, deepinfra/turbo): no brace, so nothing in it is ever read as
// one of opencode's {env:} or {file:} substitutions.
var providerSlugRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$`)

// Pin is one model's pin as a record carries it: the routing object sent,
// and the model's permanent slug.
type Pin struct {
	Order             []string `json:"order"`
	AllowFallbacks    bool     `json:"allow_fallbacks"`
	RequireParameters bool     `json:"require_parameters"`
	DataCollection    string   `json:"data_collection"`
	Quantizations     []string `json:"quantizations"`
	CanonicalSlug     string   `json:"canonical_slug"`
}

// Routing is the pin as OpenRouter's provider routing object.
func (p Pin) Routing() openaiapi.ProviderRouting {
	return openaiapi.ProviderRouting{Order: slices.Clone(p.Order), AllowFallbacks: p.AllowFallbacks,
		RequireParameters: p.RequireParameters, DataCollection: p.DataCollection, Quantizations: slices.Clone(p.Quantizations)}
}

// pinFile is a pin as the file spells it, decoded strictly: the three
// settled fields are not among its members, so a block setting one is
// refused rather than obeyed.
type pinFile struct {
	Order         []string `json:"order"`
	Quantizations []string `json:"quantizations"`
	CanonicalSlug string   `json:"canonical_slug"`
}

// readPins checks a block's pins against the block: OpenRouter's, each
// pinned model on its list and none an alias.
func readPins(baseURL string, models []string, raw map[string]pinFile) (map[string]Pin, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if !isOpenRouter(baseURL) {
		return nil, fmt.Errorf("%s is OpenRouter's provider routing, and base_url does not reach OpenRouter (%s); "+
			"a pin sent anywhere else would be recorded as holding a call nothing held", pinsKey, openRouterHost)
	}
	names := make([]string, 0, len(raw))
	for m := range raw {
		names = append(names, m)
	}
	sort.Strings(names)
	pins := map[string]Pin{}
	for _, m := range names {
		where := fmt.Sprintf("%s.%s", pinsKey, layered.BoundKey(m))
		if !slices.Contains(models, m) {
			return nil, fmt.Errorf("%s: the model is not on its models, and a block pins only a model it serves", where)
		}
		pf := raw[m]
		switch {
		case strings.HasPrefix(m, "~"):
			return nil, fmt.Errorf("%s: a ~ alias moves to whatever model its vendor names latest, so it cannot be pinned; "+
				"list and pin the model it names", where)
		case len(pf.Order) != 1:
			return nil, fmt.Errorf("%s: order names exactly one provider: with fallbacks off OpenRouter still tries each "+
				"provider order names, so a second is a fallback by another name", where)
		case !providerSlugRe.MatchString(pf.Order[0]):
			return nil, fmt.Errorf("%s: order's %q is not a provider slug (letters, digits, '.', '_', '/' and '-')",
				where, layered.BoundKey(pf.Order[0]))
		case len(pf.Quantizations) != 1:
			return nil, fmt.Errorf("%s: quantizations names exactly one quantization: a provider serving a model at two is "+
				"two endpoints behind one name", where)
		case !slices.Contains(quantisations, pf.Quantizations[0]):
			return nil, fmt.Errorf("%s: quantizations' %q is not one OpenRouter routes by (%s)",
				where, layered.BoundKey(pf.Quantizations[0]), strings.Join(quantisations, ", "))
		case pf.CanonicalSlug == "" || !validModel(pf.CanonicalSlug) || strings.HasPrefix(pf.CanonicalSlug, "~"):
			return nil, fmt.Errorf("%s: canonical_slug names the model's permanent slug (OpenRouter's canonical_slug), "+
				"never an alias, so a re-run names the same model when the listed name moves", where)
		}
		pins[m] = Pin{Order: []string{pf.Order[0]}, AllowFallbacks: false, RequireParameters: true, DataCollection: "deny",
			Quantizations: []string{pf.Quantizations[0]}, CanonicalSlug: pf.CanonicalSlug}
	}
	return pins, nil
}

// PinFor returns the pin this machine sets for model on provider.
func (c *APIConfig) PinFor(provider, model string) (Pin, bool) {
	p, ok := c.providers[provider]
	if !ok {
		return Pin{}, false
	}
	return p.pin(model)
}

// pin returns the block's pin for model, a copy.
func (p Provider) pin(model string) (Pin, bool) {
	pin, ok := p.Pins[model]
	if !ok {
		return Pin{}, false
	}
	pin.Order, pin.Quantizations = slices.Clone(pin.Order), slices.Clone(pin.Quantizations)
	return pin, true
}

// What a call's record says of its generation readback.
const (
	// ReadbackNotAsked: the caller did not opt in, so no request was made.
	ReadbackNotAsked = "not_asked"
	// ReadbackRead: the record was read into the served side.
	ReadbackRead = "read"
	// ReadbackFailed: the caller opted in and no record was read; the
	// served side is the response's own, and the reason is recorded.
	ReadbackFailed = "failed"
)

// readback is one readback's outcome.
type readback struct {
	state  string
	reason string
	gen    *openaiapi.Generation
}

// readGeneration reads OpenRouter's record of res through client, the one
// the call used: never off OpenRouter, never without the id the response
// named. Its failure is the outcome's reason, never the call's.
func readGeneration(ctx context.Context, client *openaiapi.Client, p Provider, res openaiapi.Result) readback {
	switch {
	case !isOpenRouter(p.BaseURL):
		return readback{state: ReadbackFailed, reason: "the provider is not OpenRouter, which alone keeps a generation record"}
	case res.GenerationID == "":
		return readback{state: ReadbackFailed, reason: "the answer named no generation id to read the record of"}
	}
	g, err := client.Generation(ctx, res.GenerationID)
	if err != nil {
		return readback{state: ReadbackFailed, reason: err.Error()}
	}
	return readback{state: ReadbackRead, gen: &g}
}

// withGeneration is res with the served side g reads in its place: the
// provider that served, and each count and the cost the record carries; a
// field the record does not carry keeps the response's own.
func withGeneration(res openaiapi.Result, g openaiapi.Generation) openaiapi.Result {
	if g.Provider != "" {
		res.Provider = g.Provider
	}
	if g.Usage == nil {
		return res
	}
	u := openaiapi.Usage{}
	if res.Usage != nil {
		u = *res.Usage
	}
	for _, f := range []struct {
		dst **int
		src *int
	}{
		{&u.PromptTokens, g.Usage.PromptTokens}, {&u.CompletionTokens, g.Usage.CompletionTokens},
		{&u.ReasoningTokens, g.Usage.ReasoningTokens}, {&u.CachedTokens, g.Usage.CachedTokens},
	} {
		if f.src != nil {
			*f.dst = f.src
		}
	}
	if g.Usage.Cost != nil {
		u.Cost = g.Usage.Cost
	}
	res.Usage = &u
	return res
}
