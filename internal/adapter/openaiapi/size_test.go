package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// sizedService is an OpenAI-compatible server whose model list carries
// entries (each one a data[] member as given), whose /tokenize endpoint
// answers with tokenize (vLLM's exact count; nil answers not found, a server
// without it) and whose chat endpoint answers with chat. posts counts the
// requests that reached the chat endpoint, so a test can prove a refusal
// sent nothing to it, and counted the requests that reached /tokenize.
type sizedService struct {
	*fake
	posts    atomic.Int32
	counted  atomic.Int32
	tokenize func(http.ResponseWriter, *http.Request, map[string]json.RawMessage)
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
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, string(list))
		case r.URL.Path == "/api/tokenize":
			s.counted.Add(1)
			if s.tokenize == nil {
				http.NotFound(w, r)
				return
			}
			s.tokenize(w, r, body)
		default:
			s.posts.Add(1)
			chat(w, r, body)
		}
	})
	return s
}

// counts answers /tokenize as vLLM does, with the count given for the model
// the request names, and refuses a request that is not the chat form of the
// call (the system and user messages the chat call sends) with HTTP 400, so a
// count of anything else is never read as the call's.
func counts(byModel map[string]int) func(http.ResponseWriter, *http.Request, map[string]json.RawMessage) {
	return func(w http.ResponseWriter, _ *http.Request, body map[string]json.RawMessage) {
		var model string
		var msgs []map[string]string
		_ = json.Unmarshal(body["model"], &model)
		_ = json.Unmarshal(body["messages"], &msgs)
		n, known := byModel[model]
		if !known || len(msgs) != 2 || msgs[0]["role"] != "system" || msgs[1]["role"] != "user" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"count":%d,"max_model_len":1000,"tokens":[]}`, n)
	}
}

// sizedRequest is a request whose brief is n bytes, asking for model.
func sizedRequest(model string, n int) Request {
	return Request{Model: model, Brief: Brief{Instructions: "Emit one JSON object.",
		Input: strings.Repeat("a", n-len("Emit one JSON object."))}}
}

// vllmContextError is the refusal a vLLM server answers an oversized request
// with, the server's own text the adapter must keep.
const vllmContextError = `{"object":"error","message":"This model's maximum context length is 1000 tokens. However, you requested 1500 tokens (1500 in the messages, 0 in the completion). Please reduce the length of the messages or completion.","type":"BadRequestError","param":null,"code":400}`

// TestARequestOverTheServedSizeByExactCountIsRefused is
// iss-2610030956156354: where the model list labels a model's served size
// (vLLM's max_model_len) and the server counts the request exactly
// (/tokenize), a request whose count and max_tokens are over the served size
// is refused before the chat call, the refusal naming both numbers, where
// each came from, and a listed model that fits by its own exact count;
// nothing reaches the chat endpoint. The brief is small, so no estimate from
// its bytes could be the ground: only the count is.
func TestARequestOverTheServedSizeByExactCountIsRefused(t *testing.T) {
	s := newSizedService(t, []map[string]any{
		{"id": "local/small", "object": "model", "max_model_len": 1000},
		{"id": "local/medium", "object": "model", "max_model_len": 1500},
		{"id": "local/large", "object": "model", "max_model_len": 8192},
	}, status(http.StatusBadRequest, vllmContextError))
	s.tokenize = counts(map[string]int{"local/small": 2000, "local/medium": 2000, "local/large": 2000})
	// The count, 2000, plus the 100 max_tokens reserves for the answer: 2100.
	req := sizedRequest("local/small", 600)
	req.Settings = map[string]json.RawMessage{"max_tokens": json.RawMessage(`100`)}
	_, err := mustClient(t, s.base(), testKey).Complete(context.Background(), req, nil)
	if err == nil {
		t.Fatal("Complete sent a request counted over the model's served size")
	}
	if n := s.posts.Load(); n != 0 {
		t.Fatalf("the chat endpoint was called %d time(s); a refusal before the chat call sends nothing to it", n)
	}
	msg := err.Error()
	for _, want := range []string{"2100 tokens", "2000", "/tokenize", "1000 tokens", "local/small", "max_model_len", "local/large", "8192"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not name %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "local/medium") {
		t.Errorf("the refusal names local/medium, which does not fit 2100 tokens: %s", msg)
	}
	if strings.Contains(msg, "estimat") {
		t.Errorf("the refusal speaks of an estimate; it rests on the server's count: %s", msg)
	}
	assertNoKey(t, err)
}

// TestARequestOnlyOverByEstimateIsSent: a request whose bytes would put it
// over the served size at one token per three bytes, but which the server
// counts at 90% of it, is sent: no refusal rests on an estimate.
func TestARequestOnlyOverByEstimateIsSent(t *testing.T) {
	s := newSizedService(t, []map[string]any{{"id": "local/small", "max_model_len": 1000}},
		ok("local/small", `{"verdict":"yes"}`))
	s.tokenize = counts(map[string]int{"local/small": 900})
	// 3600 bytes is 1200 tokens at one per three bytes, over 1000.
	res, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 3600), jsonObject)
	if err != nil {
		t.Fatalf("Complete refused a request the server counts at 900 of 1000 tokens: %v", err)
	}
	if s.posts.Load() != 1 || string(res.Content) != `{"verdict":"yes"}` {
		t.Fatalf("posts = %d, content = %q: want the request sent and answered", s.posts.Load(), res.Content)
	}
	if s.counted.Load() != 1 {
		t.Fatalf("/tokenize was asked %d time(s); want once", s.counted.Load())
	}
}

// TestAModelThatOnlyFitsByEstimateIsNotNamed: a listed model is named as
// fitting only when the server's own count for it fits; one it cannot count
// is not named.
func TestAModelThatOnlyFitsByEstimateIsNotNamed(t *testing.T) {
	s := newSizedService(t, []map[string]any{
		{"id": "local/small", "max_model_len": 1000},
		{"id": "local/medium", "max_model_len": 1500},
	}, ok("local/small", "{}"))
	s.tokenize = counts(map[string]int{"local/small": 1200})
	_, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 600), nil)
	if err == nil || s.posts.Load() != 0 {
		t.Fatalf("err = %v, posts = %d: want a refusal before the chat call", err, s.posts.Load())
	}
	if strings.Contains(err.Error(), "local/medium") {
		t.Errorf("the refusal names local/medium, which the server did not count: %v", err)
	}
}

// TestNoListedModelFitsIsSaid: when no listed model is served at the counted
// size, the refusal says so rather than naming one.
func TestNoListedModelFitsIsSaid(t *testing.T) {
	s := newSizedService(t, []map[string]any{
		{"id": "local/small", "max_model_len": 1000},
		{"id": "local/medium", "max_model_len": 1500},
	}, ok("local/small", "{}"))
	s.tokenize = counts(map[string]int{"local/small": 2000, "local/medium": 2000})
	_, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 600), nil)
	if err == nil || s.posts.Load() != 0 {
		t.Fatalf("err = %v, posts = %d: want a refusal before the chat call", err, s.posts.Load())
	}
	if !strings.Contains(err.Error(), "no model") {
		t.Errorf("the refusal does not say no listed model fits: %v", err)
	}
}

// TestARequestCountedAtTheServedSizeIsSent: a request whose count and
// max_tokens come to exactly the served size is within it, as vLLM judges.
func TestARequestCountedAtTheServedSizeIsSent(t *testing.T) {
	s := newSizedService(t, []map[string]any{{"id": "local/small", "max_model_len": 1000}},
		ok("local/small", `{"verdict":"yes"}`))
	s.tokenize = counts(map[string]int{"local/small": 900})
	req := sizedRequest("local/small", 600)
	req.Settings = map[string]json.RawMessage{"max_tokens": json.RawMessage(`100`)}
	if _, err := mustClient(t, s.base(), "").Complete(context.Background(), req, jsonObject); err != nil || s.posts.Load() != 1 {
		t.Fatalf("Complete: %v (posts %d): want the request sent", err, s.posts.Load())
	}
}

// TestWithoutAnExactCountTheRequestIsSentAndTheServersRefusalKept: a served
// figure is published, but the server gives no usable count (no /tokenize,
// an error, a body that is not a count, or no answer in time). Nothing
// exact is known, so the request is sent as before, however large its bytes,
// and the server's own refusal reaches the caller intact.
func TestWithoutAnExactCountTheRequestIsSentAndTheServersRefusalKept(t *testing.T) {
	for name, tokenize := range map[string]func(http.ResponseWriter, *http.Request, map[string]json.RawMessage){
		"no /tokenize":   nil,
		"an error":       status(http.StatusInternalServerError, `{"error":"boom"}`),
		"not a count":    status(http.StatusOK, `{"count":"2000"}`),
		"a zero count":   status(http.StatusOK, `{"count":0}`),
		"not JSON":       status(http.StatusOK, `count: 2000`),
		"no answer":      func(_ http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) { <-r.Context().Done() },
		"a fraction":     status(http.StatusOK, `{"count":2000.5}`),
		"a huge count":   status(http.StatusOK, `{"count":99999999999}`),
		"a redirect":     status(http.StatusFound, ``),
		"a negative one": status(http.StatusOK, `{"count":-5}`),
	} {
		t.Run(name, func(t *testing.T) {
			s := newSizedService(t, []map[string]any{{"id": "local/small", "max_model_len": 1000}},
				status(http.StatusBadRequest, vllmContextError))
			s.tokenize = tokenize
			c := mustClient(t, s.base(), "", WithFirstByteTimeout(300*time.Millisecond))
			// 6000 bytes: 2000 tokens at one per three bytes, twice the figure.
			_, err := c.Complete(context.Background(), sizedRequest("local/small", 6000), nil)
			if s.posts.Load() != 1 {
				t.Fatalf("posts = %d: without an exact count the request is sent as before", s.posts.Load())
			}
			if err == nil || !strings.Contains(err.Error(), "This model's maximum context length is 1000 tokens. However, you requested 1500 tokens") {
				t.Fatalf("the server's own refusal did not reach the caller intact: %v", err)
			}
		})
	}
}

// TestWithoutAServedFigureTheRequestIsSentAndTheServersRefusalKept: a list
// that publishes no figure labelled as served (none at all, a trained size,
// an advertised one, a figure for another model, or no list) is no ground to
// refuse: the request is sent as before, nothing is counted, and the
// server's own refusal text reaches the caller intact.
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
				s.tokenize = counts(map[string]int{"local/small": 2000, "local/other": 2000})
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
			if s.counted.Load() != 0 {
				t.Fatalf("/tokenize was asked %d time(s) without a served figure to judge by", s.counted.Load())
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

// TestThePreCallChecksSitInsideTheCallsTotalCap: the listing and the count
// are spent inside the call's total cap, never added to it, so a service
// that takes every request and answers none ends the call at its cap, and
// the refusal's "no answer within" is the whole wait.
func TestThePreCallChecksSitInsideTheCallsTotalCap(t *testing.T) {
	const capped = 2 * time.Second
	f := newFake(t, func(_ http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		<-r.Context().Done()
	})
	c := mustClient(t, f.base(), "", WithTimeout(capped), WithFirstByteTimeout(capped))
	_, err, d := completeWithin(t, 20*time.Second, context.Background(), c, nil)
	if err == nil || !strings.Contains(err.Error(), "no answer within 2s") {
		t.Fatalf("Complete after %s: %v", d, err)
	}
	if d > capped+capped*3/4 {
		t.Fatalf("the call took %s, past its total cap of %s: the pre-call checks were added to it", d, capped)
	}
}

// TestACountThatReachedTheServerIsNeverUnreachable: once the count has
// carried the brief to the server, a chat call that then cannot connect is a
// plain failure, never ErrUnreachable, which tells a caller nothing of the
// brief left this machine.
func TestACountThatReachedTheServerIsNeverUnreachable(t *testing.T) {
	s := newSizedService(t, []map[string]any{{"id": "local/small", "max_model_len": 1000}},
		ok("local/small", "{}"))
	s.tokenize = func(w http.ResponseWriter, r *http.Request, body map[string]json.RawMessage) {
		// The service goes away after counting: no new connection is
		// taken, and this one closes with the answer.
		_ = s.srv.Listener.Close()
		w.Header().Set("Connection", "close")
		counts(map[string]int{"local/small": 10})(w, r, body)
	}
	_, err := mustClient(t, s.base(), "").Complete(context.Background(), sizedRequest("local/small", 600), nil)
	if err == nil || s.posts.Load() != 0 {
		t.Fatalf("err = %v, posts = %d: want the chat call to fail to connect", err, s.posts.Load())
	}
	if errors.Is(err, ErrUnreachable) {
		t.Fatalf("the brief reached /tokenize, yet the failure says nothing left this machine: %v", err)
	}
}

// TestEveryDerivedAddressStaysOnTheBaseURLsHost proves the chat endpoint, the
// model listing and the exact count are each derived from the parsed base URL,
// never by string surgery on it: whatever the base's path, every address the
// key and the brief go to keeps the base's scheme and host (port included).
// A base whose host is literally v1 once gave a count address on the host
// "tokenize", so the brief and the key left for a host the block never named.
func TestEveryDerivedAddressStaysOnTheBaseURLsHost(t *testing.T) {
	cases := []struct {
		base, chat, models, tokenize string
	}{
		{"https://v1", "https://v1/chat/completions", "https://v1/models", ""},
		{"https://v1/", "https://v1/chat/completions", "https://v1/models", ""},
		{"https://v1:8000", "https://v1:8000/chat/completions", "https://v1:8000/models", ""},
		{"https://v1/v1", "https://v1/v1/chat/completions", "https://v1/v1/models", "https://v1/tokenize"},
		{"https://host/v1", "https://host/v1/chat/completions", "https://host/v1/models", "https://host/tokenize"},
		{"https://host/prefix/v1", "https://host/prefix/v1/chat/completions", "https://host/prefix/v1/models", "https://host/prefix/tokenize"},
		{"https://host/v1/", "https://host/v1/chat/completions", "https://host/v1/models", "https://host/tokenize"},
		{"https://host:8443/v1", "https://host:8443/v1/chat/completions", "https://host:8443/v1/models", "https://host:8443/tokenize"},
		{"https://host", "https://host/chat/completions", "https://host/models", ""},
		{"https://host/xv1", "https://host/xv1/chat/completions", "https://host/xv1/models", ""},
		{"https://host/a%2Fv1", "https://host/a%2Fv1/chat/completions", "https://host/a%2Fv1/models", ""},
		{"http://LOCALHOST:8000/v1", "http://localhost:8000/v1/chat/completions", "http://localhost:8000/v1/models", "http://localhost:8000/tokenize"},
	}
	for _, tc := range cases {
		t.Run(tc.base, func(t *testing.T) {
			c, err := New(tc.base, "")
			if err != nil {
				t.Fatalf("New(%q): %v", tc.base, err)
			}
			b, err := url.Parse(tc.base)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range []struct{ name, got, want string }{
				{"chat", c.endpoint, tc.chat},
				{"models", c.models, tc.models},
				{"tokenize", c.tokenize, tc.tokenize},
			} {
				if a.got != a.want {
					t.Errorf("%s address = %q, want %q", a.name, a.got, a.want)
				}
				if a.got == "" {
					continue
				}
				d, err := url.Parse(a.got)
				if err != nil {
					t.Fatalf("%s address %q does not parse: %v", a.name, a.got, err)
				}
				if d.Scheme != b.Scheme || !strings.EqualFold(d.Host, b.Host) {
					t.Errorf("%s address %q left the base's scheme and host %s://%s", a.name, a.got, b.Scheme, b.Host)
				}
			}
		})
	}
}
