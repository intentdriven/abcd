package openaiapi

// size.go judges a request against the size a server serves the model at,
// before the chat call (iss-2610030956156354). A request a local service ran
// about 0.5% over was refused by the service with HTTP 400 after it had been
// sent; where the service publishes the model's served size and counts the
// request exactly, the adapter now refuses such a request itself, naming both
// numbers and a listed model that fits.
//
// The grounds are the guided-connect state-of-the-art report of 2026-10-03:
// only the server can judge a request's size exactly, and the figures servers
// list differ in meaning (the size a model was trained at, the size a router
// advertises, the size a server actually serves it at). So the check reads
// only a figure the server labels as served, and refuses only on the
// server's own count of the request, never on an estimate: an estimate errs
// one way or the other, and erring toward refusing turns away a request the
// server would take (abcd's own material runs at about 3.85 bytes a token,
// spc-68). Where either is missing (no served figure, no count, a count that
// fails or does not arrive in time) the request is sent as before and the
// server's own refusal reaches the caller as the server wrote it.
//
// The one server that publishes both is vLLM: max_model_len in its model
// list, and an exact count at /tokenize, which takes the chat call's own
// messages, applies the same chat template, and answers {"count": N, ...}
// (vllm/entrypoints/serve/tokenize/protocol.py, TokenizeChatRequest and
// TokenizeResponse). vLLM refuses a chat call whose prompt tokens plus
// max_tokens exceed max_model_len (vllm/renderers/params.py,
// _token_len_check), and the check refuses on exactly that sum. /tokenize is
// served beside /v1, not under it, so it is asked only when the base URL's
// path ends in a /v1 segment.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// servedField is a field of a model-list entry that a server documents as the
// size it serves that model at, in tokens.
type servedField struct {
	name   string // the field in a data[] entry
	source string // whose field it is, as a refusal names it
}

// servedFields is every model-list field the check reads. It carries only
// fields documented as the served size. Left out on purpose, because each
// means something else: llama.cpp's meta.n_ctx_train (the trained size),
// OpenRouter's context_length and top_provider.context_length (advertised),
// and LM Studio's max_context_length (trained). Served figures a server
// publishes somewhere other than its model list (llama.cpp's /props, LM
// Studio's loaded instances, Ollama's /api/ps) are not asked for: those
// services' requests are sent as before.
var servedFields = []servedField{
	{name: "max_model_len", source: "vLLM's max_model_len"},
}

const (
	// maxServedTokens bounds a served figure, a count and a max_tokens the
	// check believes; a larger one is not a size and is read as none. Two of
	// them sum inside an int64 on every platform.
	maxServedTokens = 1 << 30
	// maxCandidates bounds how many other listed models are counted when a
	// request is refused, to name one that fits.
	maxCandidates = 3
)

// served is one listed model's served size and the field it came from.
type served struct {
	id     string
	tokens int64
	source string
}

// tokenizeURL is the exact count's address for a base URL whose path ends in
// a /v1 segment (the server's root, then /tokenize), or empty for any other
// base URL, which gives no address the count is known to be at. Only the
// path is cut, never the URL as a string: a base whose host is v1 (https://v1)
// has no /v1 path, so it gives no address rather than one on another host.
func tokenizeURL(u *url.URL) string {
	if root, ok := strings.CutSuffix(basePath(u), "/v1"); ok {
		return under(u, root+"/tokenize")
	}
	return ""
}

// reserved is what max_tokens reserves for the answer when the request sets
// it, because the server holds the prompt and the answer to the one size.
func reserved(req Request) int64 {
	raw, ok := req.Settings["max_tokens"]
	if !ok {
		return 0
	}
	// A JSON integer only: render sends the setting as written, and a server
	// reads no other form as a count.
	var v int64
	if json.Unmarshal(raw, &v) != nil || v <= 0 {
		return 0
	}
	return min(v, maxServedTokens)
}

// servedSizes is every listed model the server publishes a served size for,
// in the service's order. A list that cannot be read, or that publishes no
// served figure, gives none: the check then has nothing to judge by.
func (c *Client) servedSizes(ctx context.Context, wait time.Duration) []served {
	lctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	raw, err := c.fetchList(lctx)
	if err != nil {
		return nil
	}
	var env struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return nil
	}
	var out []served
	for _, entry := range env.Data {
		var id string
		if json.Unmarshal(entry["id"], &id) != nil || !c.usableID(id) {
			continue
		}
		for _, f := range servedFields {
			v, ok := entry[f.name]
			if !ok {
				continue
			}
			// A JSON integer only: a figure written as a string or a
			// fraction is not the field as documented, so it is no figure.
			var t int64
			if json.Unmarshal(v, &t) != nil || t <= 0 || t > maxServedTokens {
				continue
			}
			out = append(out, served{id: id, tokens: t, source: f.source})
			break
		}
	}
	return out
}

// count is the server's exact count of the brief's prompt tokens for model,
// asked of /tokenize in the chat form the call itself sends, and false when
// there is none: no address, a failure, an answer that is not a count, or no
// answer within wait. Nothing of the answer but the count is read, and
// nothing of it is quoted.
func (c *Client) count(ctx context.Context, wait time.Duration, model string, b Brief) (int64, bool) {
	if c.tokenize == "" {
		return 0, false
	}
	body, err := json.Marshal(map[string]any{"model": model, "messages": messages(b)})
	if err != nil {
		return 0, false
	}
	cctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, c.tokenize, bytes.NewReader(body))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil || len(raw) > MaxResponseBytes {
		return 0, false
	}
	// A JSON integer only, as TokenizeResponse declares it.
	var v struct {
		Count *int64 `json:"count"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Count == nil || *v.Count <= 0 || *v.Count > maxServedTokens {
		return 0, false
	}
	return *v.Count, true
}

// checkSize refuses req before the chat call when the server lists the model
// it asks for at a served size and counts the request, with the max_tokens it
// reserves, over it. The refusal names the count and where it came from, the
// served size and the field it came from, and the first listed model the
// server serves at that size and counts the request within, or says that
// none was found. Without a served figure for the model and a count of the
// request it refuses nothing.
//
// The listing and each count are waited for no longer than the call itself
// would wait for its first byte (or its total cap, or ListTimeout, whichever
// is shortest), and ctx is the call's own, so they are spent inside its total
// cap. reached reports whether any of them connected to the provider.
func (c *Client) checkSize(ctx context.Context, req Request) (reached bool, err error) {
	var conn atomic.Bool
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { conn.Store(true) },
	})
	wait := min(c.listWait, c.firstByte, c.timeout)
	sizes := c.servedSizes(ctx, wait)
	var asked *served
	for i := range sizes {
		if sizes[i].id == req.Model {
			asked = &sizes[i]
			break
		}
	}
	if asked == nil {
		return conn.Load(), nil
	}
	counted, ok := c.count(ctx, wait, req.Model, req.Brief)
	if !ok {
		return conn.Load(), nil
	}
	answer := reserved(req)
	need := counted + answer
	if need <= asked.tokens {
		return conn.Load(), nil
	}
	// A listed model is named only on the server's own count for it: its
	// tokenizer may count the same brief differently.
	fits, tried := "", 0
	for _, s := range sizes {
		if s.id == req.Model || s.tokens < need {
			continue
		}
		if tried == maxCandidates {
			break
		}
		tried++
		if n, ok := c.count(ctx, wait, s.id, req.Brief); ok && n+answer <= s.tokens {
			fits = fmt.Sprintf("%s lists %s at %d tokens (%s) and counts the request at %d tokens for it, which fits",
				c.host, s.id, s.tokens, s.source, n+answer)
			break
		}
	}
	switch {
	case fits != "":
	case tried > 0:
		fits = fmt.Sprintf("no other model %s lists was counted within the size it is served at", c.host)
	default:
		fits = fmt.Sprintf("no model %s lists is served at that size", c.host)
	}
	return conn.Load(), c.fail(fmt.Sprintf("the request is refused before the chat call: it comes to %d tokens "+
		"(%s counts the brief at %d tokens at its /tokenize, plus the %d max_tokens reserves for the answer), "+
		"over the %d tokens %s serves %s at (%s in its model list); %s",
		need, c.host, counted, answer, asked.tokens, c.host,
		termsafe.Sanitize(bound(req.Model)), asked.source, fits))
}
