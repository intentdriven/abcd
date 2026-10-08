package openaiapi

// size.go judges a request against the size a server serves the model at,
// before the request is sent (iss-2610030956156354). A request a local
// service ran about 0.5% over was refused by the service with HTTP 400 after
// it had been sent; where the service publishes the model's served size, the
// adapter now refuses such a request itself, naming both numbers and a listed
// model that fits.
//
// The grounds are the guided-connect state-of-the-art report of 2026-10-03:
// only the server can judge a request's size exactly, and the figures servers
// list differ in meaning (the size a model was trained at, the size a router
// advertises, the size a server actually serves it at). So the check reads
// only a figure the server labels as served, names where it came from, and
// never stands a trained or advertised figure in for one: where no served
// figure is published, the request is sent as before and the server's own
// refusal reaches the caller as the server wrote it.

import (
	"context"
	"encoding/json"
	"fmt"

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
	// bytesPerToken is the estimate's rate: one token for every three bytes
	// of the brief. The report's cautious estimate: a tokenizer gives English
	// prose about four bytes a token, and the estimate errs toward refusing
	// rather than sending a request the server will refuse.
	bytesPerToken = 3
	// maxServedTokens bounds a served figure the check believes; a larger one
	// is not a size and is read as no figure.
	maxServedTokens = 1 << 30
)

// served is one listed model's served size and the field it came from.
type served struct {
	id     string
	tokens int
	source string
}

// estimate is the request's estimated size in tokens: its brief at
// bytesPerToken, plus what max_tokens reserves for the answer when the
// request sets it, because the server holds the prompt and the answer to the
// one size.
func estimate(req Request) (tokens, briefBytes, reserved int) {
	briefBytes = len(req.Brief.Instructions) + len(req.Brief.Input)
	if raw, ok := req.Settings["max_tokens"]; ok {
		// A JSON integer only: render sends the setting as written, and a
		// server reads no other form as a count.
		var v int64
		if json.Unmarshal(raw, &v) == nil && v > 0 {
			reserved = int(min(v, maxServedTokens))
		}
	}
	return (briefBytes+bytesPerToken-1)/bytesPerToken + reserved, briefBytes, reserved
}

// servedSizes is every listed model the server publishes a served size for,
// in the service's order. A list that cannot be read, or that publishes no
// served figure, gives none: the check then has nothing to judge by.
func (c *Client) servedSizes(ctx context.Context) []served {
	raw, err := c.fetchList(ctx)
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
			out = append(out, served{id: id, tokens: int(t), source: f.source})
			break
		}
	}
	return out
}

// checkSize refuses req before it is sent when the server lists the model it
// asks for at a served size and the request's estimate is over it. The
// refusal names the estimate and how it was made, the served size and the
// field it came from, and the first listed model served at the estimate or
// more, or says that none is. Without a served figure for the model it
// refuses nothing.
//
// The listing is waited for no longer than the call itself would wait for
// its first byte (or its total cap, or ListTimeout, whichever is shortest),
// so a call with a short bound of its own, such as the setup's verification,
// keeps it however the listing behaves.
func (c *Client) checkSize(ctx context.Context, req Request) error {
	lctx, cancel := context.WithTimeout(ctx, min(c.listWait, c.firstByte, c.timeout))
	defer cancel()
	sizes := c.servedSizes(lctx)
	var asked *served
	for i := range sizes {
		if sizes[i].id == req.Model {
			asked = &sizes[i]
			break
		}
	}
	if asked == nil {
		return nil
	}
	est, briefBytes, reserved := estimate(req)
	if est <= asked.tokens {
		return nil
	}
	fits := fmt.Sprintf("no model %s lists is served at that size", c.host)
	for _, s := range sizes {
		if s.id != req.Model && s.tokens >= est {
			fits = fmt.Sprintf("%s lists %s at %d tokens (%s), which fits", c.host, s.id, s.tokens, s.source)
			break
		}
	}
	return c.fail(fmt.Sprintf("the request is refused before it is sent: it is estimated at %d tokens "+
		"(%d bytes of brief at one token per %d bytes, plus the %d max_tokens reserves for the answer), "+
		"over the %d tokens %s serves %s at (%s in its model list); %s",
		est, briefBytes, bytesPerToken, reserved, asked.tokens, c.host,
		termsafe.Sanitize(bound(req.Model)), asked.source, fits))
}
