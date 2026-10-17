package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// The tests in this file are itd-166's adapter half: a call records what
// answered as the response names it (the generation id, the upstream
// provider, the finish reason, the token counts and the cost), asks for usage
// in the stream unless the connection turns that off, and adds no request to
// the ones the call already makes.

// openRouterUsage is the usage member OpenRouter sends: its cost in credits
// and the token details beside the counts.
const openRouterUsage = `{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,` +
	`"prompt_tokens_details":{"cached_tokens":100,"cache_write_tokens":20},` +
	`"completion_tokens_details":{"reasoning_tokens":12},"cost":0.00042}`

// openRouterChunk is one OpenRouter-shaped stream event: the generation id
// and the upstream provider ride on every chunk.
func openRouterChunk(content, finish, usage string) string {
	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}
	var fr any
	if finish != "" {
		fr = finish
	}
	m := map[string]any{
		"id": "gen-1760000000-abcdef", "provider": "Fireworks", "object": "chat.completion.chunk", "model": "typesafe/jev-1.13-20260915",
		"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": fr}},
	}
	if usage != "" {
		m["choices"] = []any{}
		m["usage"] = json.RawMessage(usage)
	}
	b, _ := json.Marshal(m)
	return "data: " + string(b)
}

// assertServed holds a result to the OpenRouter-shaped answer's served side.
func assertServed(t *testing.T, res Result) {
	t.Helper()
	if res.GenerationID != "gen-1760000000-abcdef" || res.Provider != "Fireworks" || res.FinishReason != "stop" {
		t.Fatalf("served: id %q, provider %q, finish %q", res.GenerationID, res.Provider, res.FinishReason)
	}
	u := res.Usage
	if u == nil || u.PromptTokens != 120 || u.CompletionTokens != 30 || u.TotalTokens != 150 {
		t.Fatalf("usage = %+v", u)
	}
	if u.CachedTokens == nil || *u.CachedTokens != 100 || u.CacheWriteTokens == nil || *u.CacheWriteTokens != 20 {
		t.Fatalf("cache details = %v / %v", u.CachedTokens, u.CacheWriteTokens)
	}
	if u.ReasoningTokens == nil || *u.ReasoningTokens != 12 {
		t.Fatalf("reasoning = %v", u.ReasoningTokens)
	}
	if u.Cost == nil || *u.Cost != 0.00042 {
		t.Fatalf("cost = %v", u.Cost)
	}
}

// TestAnOpenRouterStreamFillsTheServedSide: a streamed answer whose usage
// arrives in a chunk after the finish reason is read to [DONE], and every
// served field the stream names is on the result.
func TestAnOpenRouterStreamFillsTheServedSide(t *testing.T) {
	f := newFake(t, streams(
		openRouterChunk(`{"verdict":"yes"}`, "", ""), "",
		openRouterChunk("", "stop", ""), "",
		openRouterChunk("", "", openRouterUsage), "",
		"data: [DONE]", "",
	))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	assertServed(t, res)
}

// TestAnOpenRouterPlainBodyFillsTheServedSide: the same answer as one
// chat-completion body fills the same fields.
func TestAnOpenRouterPlainBodyFillsTheServedSide(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"gen-1760000000-abcdef","provider":"Fireworks","model":"typesafe/jev-1.13-20260915",`+
			`"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"{\"verdict\":\"yes\"}"}}],`+
			`"usage":`+openRouterUsage+`}`)
	})
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	assertServed(t, res)
}

// TestAnAnswerNamingNoServedDetailLeavesThemUnset: an answer with no id, no
// provider and no usage leaves each unset, never zero standing in for a value.
func TestAnAnswerNamingNoServedDetailLeavesThemUnset(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"m","choices":[{"index":0,"finish_reason":"stop","message":{"content":"{\"verdict\":\"yes\"}"}}],`+
			`"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
	})
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if res.GenerationID != "" || res.Provider != "" {
		t.Fatalf("id %q, provider %q, want both empty", res.GenerationID, res.Provider)
	}
	if u := res.Usage; u == nil || u.CachedTokens != nil || u.CacheWriteTokens != nil || u.ReasoningTokens != nil || u.Cost != nil {
		t.Fatalf("usage = %+v, want counts alone", res.Usage)
	}
}

// TestAMalformedUsageMemberIsNotAFailure: a usage member whose fields are of
// another type (a cost as text, details as a list) is read for what it holds
// and never turns an answer into a refusal.
func TestAMalformedUsageMemberIsNotAFailure(t *testing.T) {
	f := newFake(t, streams(
		chunkLine("m", `{"verdict":"yes"}`, "stop"), "",
		`data: {"id":7,"provider":{"name":"x"},"model":"m","choices":[],"usage":{"prompt_tokens":4,"completion_tokens":"2","cost":"0.1","prompt_tokens_details":[1]}}`, "",
		"data: [DONE]", "",
	))
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatalf("Complete refused an answer over a malformed usage member: %v", err)
	}
	if res.Usage == nil || res.Usage.PromptTokens != 4 || res.Usage.Cost != nil || res.Usage.CachedTokens != nil {
		t.Fatalf("usage = %+v", res.Usage)
	}
	if res.Provider != "" || res.GenerationID != "gen-1" {
		t.Fatalf("provider %q, id %q", res.Provider, res.GenerationID)
	}
}

// TestTheRequestAsksForUsageInTheStreamUnlessTurnedOff is decision 4: the
// request carries stream_options.include_usage, and a connection that turns
// it off sends a request without it.
func TestTheRequestAsksForUsageInTheStreamUnlessTurnedOff(t *testing.T) {
	f := newFake(t, ok("m", `{"verdict":"yes"}`))
	if _, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject); err != nil {
		t.Fatal(err)
	}
	if got := string(f.last.Load().body["stream_options"]); got != `{"include_usage":true}` {
		t.Fatalf("stream_options = %s, want include_usage", got)
	}
	req := request()
	req.NoStreamUsage = true
	if _, err := mustClient(t, f.base(), testKey).Complete(context.Background(), req, jsonObject); err != nil {
		t.Fatal(err)
	}
	if got, present := f.last.Load().body["stream_options"]; present {
		t.Fatalf("stream_options = %s, want it absent", got)
	}
}

// TestARefusalOfTheUsageRequestNamesIt: a server that refuses the request
// with HTTP 400 while it asked for usage in the stream is told apart
// (ErrStreamUsage) and the refusal names the field; the same refusal of a
// request that did not ask is not.
func TestARefusalOfTheUsageRequestNamesIt(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"Unrecognized request argument supplied: stream_options"}}`)
	})
	_, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if !errors.Is(err, ErrStreamUsage) || !strings.Contains(err.Error(), "stream_options.include_usage") {
		t.Fatalf("err = %v, want ErrStreamUsage naming stream_options.include_usage", err)
	}
	req := request()
	req.NoStreamUsage = true
	_, err = mustClient(t, f.base(), testKey).Complete(context.Background(), req, jsonObject)
	if err == nil || errors.Is(err, ErrStreamUsage) {
		t.Fatalf("err = %v, want a refusal that is not ErrStreamUsage", err)
	}
}

// TestRecordingTheServedSideAddsNoRequest is criterion 8: on a server the
// size check reaches (vLLM's listing and /tokenize), a call makes the same
// requests whether it asks for usage or not and whether the answer carries
// the served detail or not.
func TestRecordingTheServedSideAddsNoRequest(t *testing.T) {
	count := func(noUsage bool, chat func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)) int32 {
		s := newSizedService(t, []map[string]any{{"id": "typesafe/jev-1.13", "max_model_len": 100000}}, chat)
		s.tokenize = counts(map[string]int{"typesafe/jev-1.13": 50})
		req := request()
		req.NoStreamUsage = noUsage
		if _, err := mustClient(t, s.base(), "").Complete(context.Background(), req, jsonObject); err != nil {
			t.Fatal(err)
		}
		return s.calls.Load()
	}
	bare := count(true, ok("typesafe/jev-1.13", `{"verdict":"yes"}`))
	recorded := count(false, streams(
		openRouterChunk(`{"verdict":"yes"}`, "stop", ""), "",
		openRouterChunk("", "", openRouterUsage), "",
		"data: [DONE]", "",
	))
	if bare != recorded || bare < 2 {
		t.Fatalf("requests: %d without the record, %d with it; want the same (and the size check's among them)", bare, recorded)
	}
}

// TestThePromptHashIsStableAndSensitive is decision 5: the rendered messages
// are recorded by their SHA-256, the same for the same brief and different
// for a brief one byte apart.
func TestThePromptHashIsStableAndSensitive(t *testing.T) {
	b := request().Brief
	h := PromptSHA256(b)
	if len(h) != 64 || strings.Trim(h, "0123456789abcdef") != "" {
		t.Fatalf("hash = %q, want 64 hex digits", h)
	}
	if PromptSHA256(b) != h {
		t.Fatal("the same brief hashed twice differs")
	}
	b.Input += "."
	if PromptSHA256(b) == h {
		t.Fatal("a brief one byte longer hashes the same")
	}
	b = request().Brief
	b.Instructions, b.Input = b.Instructions+b.Input, ""
	if PromptSHA256(b) == h {
		t.Fatal("moving text between the messages hashes the same")
	}
}

// TestServedStringsAreScrubbedCleanedAndBounded: the provider, the
// generation id and the finish reason are the provider's own text, so each
// is scrubbed of the key, has its hidden and control runes encoded and is
// bounded, as the reported model is.
func TestServedStringsAreScrubbedCleanedAndBounded(t *testing.T) {
	long := strings.Repeat("p", 300)
	f := newFake(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{
			"id": "gen-\x1b[31m-‮-" + testKey, "provider": long, "model": "m\x07",
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop​", "message": map[string]any{"content": `{"verdict":"yes"}`}}},
		})
		_, _ = w.Write(b)
	})
	res, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject)
	if err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]string{"id": res.GenerationID, "provider": res.Provider, "model": res.ModelReported, "finish": res.FinishReason} {
		if strings.Contains(v, testKey) {
			t.Errorf("%s carries the key: %q", name, v)
		}
		for _, r := range v {
			if r < 0x20 || r == 0x7f || r == '‮' || r == '​' {
				t.Errorf("%s carries rune %U: %q", name, r, v)
			}
		}
		if len(v) > MaxModelBytes+len("...") {
			t.Errorf("%s is %d bytes, over the bound", name, len(v))
		}
	}
}

// TestIsLocalNamesThisMachine: a base URL on this machine is a local route,
// whose weights and runtime no pin names yet (decision 7).
func TestIsLocalNamesThisMachine(t *testing.T) {
	for base, want := range map[string]bool{
		"http://localhost:8000/v1": true, "http://127.0.0.1:11434/v1": true, "http://[::1]:8080/v1": true,
		"https://openrouter.ai/api/v1": false, "not a url": false,
	} {
		if got := IsLocal(base); got != want {
			t.Errorf("IsLocal(%q) = %v, want %v", base, got, want)
		}
	}
}
