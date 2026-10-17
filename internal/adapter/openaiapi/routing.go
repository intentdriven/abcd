package openaiapi

// routing.go is OpenRouter's half of iss-2610161734413178: the provider
// routing object a pinned call carries, and the generation record a caller
// that opted in reads back after the call.
//
// The routing object is OpenRouter's documented one
// (openrouter.ai/docs/features/provider-routing, read 2026-10-16 for the
// model-matrix SOTA note): order, allow_fallbacks, require_parameters,
// data_collection and quantizations. Which values a pin takes is
// internal/core/oracle's to decide; the adapter sends what it is given.
//
// The generation record is OpenRouter's GET /api/v1/generation?id=
// (openrouter.ai/docs/api-reference/get-a-generation): the served model and
// provider, each provider attempted (provider_responses), the native token
// counts and the total cost. Reading it is a request of its own, so it is
// made only when the caller asks (Generation), on the same pinned client as
// the call: the key only as the bearer token, no redirect followed, the body
// bounded, and the whole readback bounded in time, every ask and every wait
// between asks included. OpenRouter records a generation a moment after its
// answer ends, so a not-found is asked again after a short wait, doubling,
// until the bound; it is never waited on past it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// ProviderRouting is OpenRouter's provider routing object, as a request
// carries it under ProviderRoutingField.
type ProviderRouting struct {
	Order             []string `json:"order"`
	AllowFallbacks    bool     `json:"allow_fallbacks"`
	RequireParameters bool     `json:"require_parameters"`
	DataCollection    string   `json:"data_collection"`
	Quantizations     []string `json:"quantizations"`
}

// ProviderRoutingField is the request member the routing object rides in.
const ProviderRoutingField = "provider"

const (
	// GenerationTimeout bounds one readback end to end: every ask, every
	// wait between asks and every body read.
	GenerationTimeout = 20 * time.Second
	// generationBackoff is the first wait after a not-found, doubled after
	// each, up to maxGenerationBackoff.
	generationBackoff    = 250 * time.Millisecond
	maxGenerationBackoff = 4 * time.Second
	// MaxAttempts bounds the provider attempts a record keeps.
	MaxAttempts = 16
)

// generationIDRe is an id OpenRouter issues (gen-<digits>-<letters>): an id
// of another shape is never put into a request.
var generationIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// errGenerationTimeout is the cause GenerationTimeout ends a readback with.
var errGenerationTimeout = errors.New("generation readback timeout")

// Attempt is one provider OpenRouter tried for a generation: its name and
// the status it answered, each cleaned and bounded like the model.
type Attempt struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

// Generation is what OpenRouter's record of one generation says served it.
// Every string is the provider's own text, scrubbed of the key, cleaned and
// bounded; a field it did not carry readably is empty, or nil in Usage.
type Generation struct {
	Model        string
	Provider     string
	FinishReason string
	// Attempts is every provider the record names as tried, in its order,
	// at most MaxAttempts; empty when it names none.
	Attempts []Attempt
	// Usage holds the native prompt, completion, reasoning and cached
	// counts and the total cost; nil when none of them reads.
	Usage *Usage
}

// Generation reads OpenRouter's record of the generation id names, bounded
// by the client's readback limit (GenerationTimeout). It is a request of its
// own: a caller makes it only when the run it belongs to asked for it. Every
// failure is an error in abcd's words, scrubbed of the key, and quotes no
// part of the provider's body.
func (c *Client) Generation(ctx context.Context, id string) (Generation, error) {
	if !generationIDRe.MatchString(id) || c.carriesKey(id) {
		return Generation{}, c.fail("the generation id is not one OpenRouter issues, so its record is not asked for")
	}
	gctx, cancel := context.WithTimeoutCause(ctx, c.genWait, errGenerationTimeout)
	defer cancel()
	addr := c.generation + "?" + url.Values{"id": {id}}.Encode()
	wait := c.genBackoff
	for {
		raw, code, err := c.fetchGeneration(ctx, gctx, addr)
		switch {
		case err != nil:
			return Generation{}, err
		case code == http.StatusOK:
			return c.decodeGeneration(raw)
		case code != http.StatusNotFound:
			return Generation{}, c.fail(fmt.Sprintf("%s answered HTTP %d when asked for the generation's record", c.host, code))
		}
		t := time.NewTimer(wait)
		select {
		case <-gctx.Done():
			t.Stop()
			return Generation{}, c.generationEnded(ctx)
		case <-t.C:
		}
		wait = min(2*wait, maxGenerationBackoff)
	}
}

// fetchGeneration is one ask: the body of a 200, unread for any other
// status, which is returned alone.
func (c *Client) fetchGeneration(parent, gctx context.Context, addr string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(gctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, 0, c.fail("the generation's record could not be asked for: the request could not be built")
	}
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		if gctx.Err() != nil {
			return nil, 0, c.generationEnded(parent)
		}
		if errors.Is(err, errRedirect) {
			return nil, 0, c.fail(c.host + " answered the generation's record with a redirect, which abcd never follows")
		}
		return nil, 0, c.fail("the generation's record could not be read: " + c.host + " could not be reached")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	switch {
	case err != nil && gctx.Err() != nil:
		return nil, 0, c.generationEnded(parent)
	case err != nil:
		return nil, 0, c.fail("the generation's record from " + c.host + " was cut off")
	case len(raw) > MaxResponseBytes:
		return nil, 0, c.fail(fmt.Sprintf("%s answered with a generation record larger than %d bytes, so it is refused unread", c.host, MaxResponseBytes))
	}
	return raw, http.StatusOK, nil
}

// generationEnded is a readback its bound or its caller ended.
func (c *Client) generationEnded(parent context.Context) error {
	if parent.Err() != nil {
		return c.fail("the generation's record was not waited for: the call it belongs to ended")
	}
	return c.fail(fmt.Sprintf("%s had no record of the generation within %s", c.host, c.genWait))
}

// decodeGeneration reads a record field by field: a field of another type
// than OpenRouter's, a negative count or a cost that is not readable
// (ReadableCost) is left unread, never a failure; only a body that is not a
// record at all is one.
func (c *Client) decodeGeneration(raw []byte) (Generation, error) {
	var env struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Data == nil {
		return Generation{}, c.fail(c.host + " answered with something that is not a generation record")
	}
	d := env.Data
	clean := func(raw json.RawMessage) string { return cleanModel(c.scrub(lenientString(raw))) }
	g := Generation{Model: clean(d["model"]), Provider: clean(d["provider_name"]), FinishReason: clean(d["finish_reason"])}
	var tried []map[string]json.RawMessage
	if json.Unmarshal(d["provider_responses"], &tried) == nil {
		for _, a := range tried {
			if len(g.Attempts) == MaxAttempts {
				break
			}
			g.Attempts = append(g.Attempts, Attempt{Provider: clean(a["provider_name"]), Status: c.status(a["status"])})
		}
	}
	u := &Usage{PromptTokens: readCount(d["native_tokens_prompt"]), CompletionTokens: readCount(d["native_tokens_completion"]),
		ReasoningTokens: readCount(d["native_tokens_reasoning"]), CachedTokens: readCount(d["native_tokens_cached"])}
	var cost float64
	if r := d["total_cost"]; len(r) > 0 && !isNull(r) && json.Unmarshal(r, &cost) == nil && ReadableCost(cost) {
		u.Cost = &cost
	}
	if u.readable() > 0 {
		g.Usage = u
	}
	return g, nil
}

// status is an attempt's status, a number or a string, cleaned; "" when it
// is neither.
func (c *Client) status(raw json.RawMessage) string {
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return strconv.Itoa(n)
	}
	return cleanModel(c.scrub(lenientString(raw)))
}

// readCount is raw as a count: a whole number not below zero, nil otherwise.
// A null is nil, never zero: encoding/json leaves the number as it was, with
// no error, when it reads a null, and OpenRouter answers null for a count the
// model has none of.
func readCount(raw json.RawMessage) *int {
	var n int
	if len(raw) == 0 || isNull(raw) || json.Unmarshal(raw, &n) != nil || n < 0 {
		return nil
	}
	return &n
}

// isNull reports whether raw is JSON's null.
func isNull(raw json.RawMessage) bool { return string(bytes.TrimSpace(raw)) == "null" }
