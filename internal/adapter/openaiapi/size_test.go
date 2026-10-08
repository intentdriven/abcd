package openaiapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// sizedService is an OpenAI-compatible server whose model list carries
// entries (each one a data[] member as given) and whose chat endpoint answers
// with chat; posts counts the requests that reached the chat endpoint, so a
// test can prove a refusal sent nothing.
type sizedService struct {
	*fake
	posts atomic.Int32
}

func newSizedService(t *testing.T, entries []map[string]any,
	chat func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)) *sizedService {
	t.Helper()
	list, err := json.Marshal(map[string]any{"object": "list", "data": entries})
	if err != nil {
		t.Fatal(err)
	}
	s := &sizedService{}
	s.fake = newFake(t, func(w http.ResponseWriter, r *http.Request, body map[string]json.RawMessage) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, string(list))
			return
		}
		s.posts.Add(1)
		chat(w, r, body)
	})
	return s
}

// sizedRequest is a request whose brief is n bytes, asking for model.
func sizedRequest(model string, n int) Request {
	return Request{Model: model, Brief: Brief{Instructions: "Emit one JSON object.",
		Input: strings.Repeat("a", n-len("Emit one JSON object."))}}
}

// vllmContextError is the refusal a vLLM server answers an oversized request
// with, the server's own text the adapter must keep.
const vllmContextError = `{"object":"error","message":"This model's maximum context length is 1000 tokens. However, you requested 1500 tokens (1500 in the messages, 0 in the completion). Please reduce the length of the messages or completion.","type":"BadRequestError","param":null,"code":400}`

// TestARequestOverTheServedSizeIsRefusedBeforeItIsSent is
// iss-2610030956156354: a model list that labels a model's served size
// (vLLM's max_model_len) is read before the request is sent, and a request
// estimated over it is refused there, the refusal naming both numbers, where
// the served figure came from, and a listed model that fits; nothing reaches
// the chat endpoint.
func TestARequestOverTheServedSizeIsRefusedBeforeItIsSent(t *testing.T) {
	s := newSizedService(t, []map[string]any{
		{"id": "local/small", "object": "model", "max_model_len": 1000},
		{"id": "local/medium", "object": "model", "max_model_len": 1500},
		{"id": "local/large", "object": "model", "max_model_len": 8192},
	}, status(http.StatusBadRequest, vllmContextError))
	// 6000 bytes at one token per 3 bytes is 2000 tokens, plus the 100
	// max_tokens reserves for the answer: 2100.
	req := sizedRequest("local/small", 6000)
	req.Settings = map[string]json.RawMessage{"max_tokens": json.RawMessage(`100`)}
	_, err := mustClient(t, s.base(), testKey).Complete(context.Background(), req, nil)
	if err == nil {
		t.Fatal("Complete sent a request estimated over the model's served size")
	}
	if n := s.posts.Load(); n != 0 {
		t.Fatalf("the chat endpoint was called %d time(s); a refusal before sending sends nothing", n)
	}
	msg := err.Error()
	for _, want := range []string{"2100 tokens", "1000 tokens", "local/small", "max_model_len", "local/large", "8192"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not name %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "local/medium") {
		t.Errorf("the refusal names local/medium, which does not fit 2100 tokens: %s", msg)
	}
	assertNoKey(t, err)
}

// TestNoListedModelFitsIsSaid: when no listed model is served at the
// estimated size, the refusal says so rather than naming one.
func TestNoListedModelFitsIsSaid(t *testing.T) {
	s := newSizedService(t, []map[string]any{
		{"id": "local/small", "max_model_len": 1000},
		{"id": "local/medium", "max_model_len": 1500},
	}, ok("local/small", "{}"))
	_, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 6000), nil)
	if err == nil || s.posts.Load() != 0 {
		t.Fatalf("err = %v, posts = %d: want a refusal before sending", err, s.posts.Load())
	}
	if !strings.Contains(err.Error(), "no model") {
		t.Errorf("the refusal does not say no listed model fits: %v", err)
	}
}

// TestARequestWithinTheServedSizeIsSent: a request estimated within the
// served size is sent as before.
func TestARequestWithinTheServedSizeIsSent(t *testing.T) {
	s := newSizedService(t, []map[string]any{{"id": "local/small", "max_model_len": 1000}},
		ok("local/small", `{"verdict":"yes"}`))
	res, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 2400), jsonObject)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if s.posts.Load() != 1 || string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("posts = %d, content = %q: want the request sent and answered", s.posts.Load(), res.Content)
	}
}

// TestWithoutAServedFigureTheRequestIsSentAndTheServersRefusalKept: a list
// that publishes no figure labelled as served (none at all, a trained size,
// an advertised one, a figure for another model, or no list) is no ground to
// refuse: the request is sent as before, and the server's own refusal text
// reaches the caller intact.
func TestWithoutAServedFigureTheRequestIsSentAndTheServersRefusalKept(t *testing.T) {
	for name, entries := range map[string][]map[string]any{
		"no figure":           {{"id": "local/small", "object": "model"}},
		"trained (llama.cpp)": {{"id": "local/small", "meta": map[string]any{"n_ctx_train": 1000}}},
		"advertised":          {{"id": "local/small", "context_length": 1000, "top_provider": map[string]any{"context_length": 1000}}},
		"another model":       {{"id": "local/other", "max_model_len": 1000}},
		"not a number":        {{"id": "local/small", "max_model_len": "1000"}},
		"no list":             nil,
	} {
		t.Run(name, func(t *testing.T) {
			var s *sizedService
			if entries != nil {
				s = newSizedService(t, entries, status(http.StatusBadRequest, vllmContextError))
			} else {
				s = &sizedService{}
				s.fake = newFake(t, func(w http.ResponseWriter, r *http.Request, body map[string]json.RawMessage) {
					if r.Method == http.MethodGet {
						http.NotFound(w, r)
						return
					}
					s.posts.Add(1)
					status(http.StatusBadRequest, vllmContextError)(w, r, body)
				})
			}
			_, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 6000), nil)
			if s.posts.Load() != 1 {
				t.Fatalf("posts = %d: without a served figure the request is sent as before", s.posts.Load())
			}
			if err == nil || !strings.Contains(err.Error(), "This model's maximum context length is 1000 tokens. However, you requested 1500 tokens") {
				t.Fatalf("the server's own refusal did not reach the caller intact: %v", err)
			}
		})
	}
}

// TestAStalledListingHoldsTheCallNoLongerThanItsFirstByteLimit: the listing
// the check reads is bounded by the call's own first-byte limit, so a service
// whose list never arrives costs a call with a short bound no more than that
// bound before the request is sent as before.
func TestAStalledListingHoldsTheCallNoLongerThanItsFirstByteLimit(t *testing.T) {
	var posts atomic.Int32
	f := newFake(t, func(w http.ResponseWriter, r *http.Request, body map[string]json.RawMessage) {
		if r.Method == http.MethodGet {
			<-r.Context().Done()
			return
		}
		posts.Add(1)
		ok("local/small", `{"verdict":"yes"}`)(w, r, body)
	})
	c := mustClient(t, f.base(), "", WithFirstByteTimeout(200*time.Millisecond))
	_, err, d := completeWithin(t, 10*time.Second, context.Background(), c, jsonObject)
	if err != nil || posts.Load() != 1 {
		t.Fatalf("Complete after %s: %v (posts %d); a listing that never arrives is no figure, so the request is sent", d, err, posts.Load())
	}
	if d > 5*time.Second {
		t.Fatalf("the stalled listing held the call for %s, past its first-byte limit of 200ms", d)
	}
}
