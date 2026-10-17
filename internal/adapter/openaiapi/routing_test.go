package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The tests in this file are iss-2610161734413178's adapter half: a call can
// carry OpenRouter's provider routing object, and a caller that opts in can
// read OpenRouter's generation record for the call, bounded in time and
// cleaned like every other string a provider supplies. No test reaches a
// network: OpenRouter is an httptest fake answering recorded shapes.

// pinned is the routing object a pinned call sends.
func pinned() *ProviderRouting {
	return &ProviderRouting{Order: []string{"deepinfra"}, AllowFallbacks: false, RequireParameters: true,
		DataCollection: "deny", Quantizations: []string{"fp8"}}
}

// TestAPinnedRequestCarriesTheProviderRoutingObject: the request's provider
// member is OpenRouter's documented routing object, every field present, and
// a request with no pin carries no provider member at all.
func TestAPinnedRequestCarriesTheProviderRoutingObject(t *testing.T) {
	f := newFake(t, ok("m", `{"verdict":"yes"}`))
	req := request()
	req.Provider = pinned()
	if _, err := mustClient(t, f.base(), testKey).Complete(context.Background(), req, jsonObject); err != nil {
		t.Fatal(err)
	}
	want := `{"order":["deepinfra"],"allow_fallbacks":false,"require_parameters":true,"data_collection":"deny","quantizations":["fp8"]}`
	if got := string(f.last.Load().body["provider"]); got != want {
		t.Fatalf("provider = %s, want %s", got, want)
	}
	if _, err := mustClient(t, f.base(), testKey).Complete(context.Background(), request(), jsonObject); err != nil {
		t.Fatal(err)
	}
	if got, present := f.last.Load().body["provider"]; present {
		t.Fatalf("an unpinned request carries provider = %s", got)
	}
}

// generationRecord is OpenRouter's generation record as GET
// /api/v1/generation?id= answers it, with a fallback attempt before the
// provider that served.
const generationRecord = `{"data":{"id":"gen-1760000000-abcdef","model":"meta-llama/llama-3.1-8b-instruct",` +
	`"provider_name":"DeepInfra","finish_reason":"stop","total_cost":0.00051,` +
	`"native_tokens_prompt":130,"native_tokens_completion":31,"native_tokens_reasoning":7,"native_tokens_cached":90,` +
	`"provider_responses":[{"provider_name":"Together","status":503},{"provider_name":"DeepInfra","status":200}]}}`

// genFake answers the generation record after notFound 404s, counting every
// request and keeping the last one's query and authorisation.
type genFake struct {
	*fake
	asked atomic.Int32
	query atomic.Value
	auth  atomic.Value
}

func newGenFake(t *testing.T, notFound int32, record string) *genFake {
	t.Helper()
	g := &genFake{}
	g.fake = newFake(t, func(w http.ResponseWriter, r *http.Request, _ map[string]json.RawMessage) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/generation" {
			http.NotFound(w, r)
			return
		}
		n := g.asked.Add(1)
		g.query.Store(r.URL.RawQuery)
		g.auth.Store(r.Header.Get("Authorization"))
		if n <= notFound {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, record)
	})
	return g
}

// quick is a client whose readback waits are short, so a test of the bound
// takes milliseconds.
func quick(t *testing.T, base string, wait time.Duration) *Client {
	c := mustClient(t, base, testKey)
	c.genWait, c.genBackoff = wait, time.Millisecond
	return c
}

// TestTheGenerationRecordFillsTheServedSide: the record's served provider,
// every attempt in order, the native token counts and the cost are read, the
// request names the generation and carries the key as its bearer token.
func TestTheGenerationRecordFillsTheServedSide(t *testing.T) {
	g := newGenFake(t, 0, generationRecord)
	gen, err := quick(t, g.base(), time.Second).Generation(context.Background(), "gen-1760000000-abcdef")
	if err != nil {
		t.Fatalf("Generation: %v", err)
	}
	if q := g.query.Load(); q != "id=gen-1760000000-abcdef" {
		t.Fatalf("query = %v", q)
	}
	if a := g.auth.Load(); a != "Bearer "+testKey {
		t.Fatalf("authorization = %v", a)
	}
	if gen.Provider != "DeepInfra" || gen.Model != "meta-llama/llama-3.1-8b-instruct" || gen.FinishReason != "stop" {
		t.Fatalf("generation = %+v", gen)
	}
	want := []Attempt{{Provider: "Together", Status: "503"}, {Provider: "DeepInfra", Status: "200"}}
	if len(gen.Attempts) != 2 || gen.Attempts[0] != want[0] || gen.Attempts[1] != want[1] {
		t.Fatalf("attempts = %+v, want %+v", gen.Attempts, want)
	}
	u := gen.Usage
	is := func(p *int, n int) bool { return p != nil && *p == n }
	if u == nil || !is(u.PromptTokens, 130) || !is(u.CompletionTokens, 31) || !is(u.ReasoningTokens, 7) ||
		!is(u.CachedTokens, 90) || u.Cost == nil || *u.Cost != 0.00051 {
		t.Fatalf("usage = %+v", u)
	}
}

// TestTheGenerationRecordIsWaitedForWithinItsBound: OpenRouter records a
// generation a moment after its answer ends, so a not-found is asked again,
// and the wait is bounded: a record that never appears ends in a failure
// within the bound, never a wait without end.
func TestTheGenerationRecordIsWaitedForWithinItsBound(t *testing.T) {
	g := newGenFake(t, 2, generationRecord)
	if _, err := quick(t, g.base(), time.Second).Generation(context.Background(), "gen-1"); err != nil {
		t.Fatalf("a record found on the third ask: %v", err)
	}
	if n := g.asked.Load(); n != 3 {
		t.Fatalf("asked %d times, want 3", n)
	}
	never := newGenFake(t, 1<<30, generationRecord)
	start := time.Now()
	_, err := quick(t, never.base(), 50*time.Millisecond).Generation(context.Background(), "gen-1")
	if err == nil || !strings.Contains(err.Error(), "no record of the generation") {
		t.Fatalf("err = %v, want a bounded failure", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("the readback waited %s past a 50ms bound", took)
	}
}

// TestAFailedReadbackNeverCarriesTheKey: a refusal that echoes the key, and a
// record whose strings carry it or terminal escapes, reach neither an error
// nor the generation read.
func TestAFailedReadbackNeverCarriesTheKey(t *testing.T) {
	refusing := newFake(t, status(http.StatusUnauthorized, `{"error":{"message":"bad key `+testKey+`"}}`))
	_, err := quick(t, refusing.base(), time.Second).Generation(context.Background(), "gen-1")
	if err == nil {
		t.Fatal("a refused readback succeeded")
	}
	assertNoKey(t, err)
	hostile := newGenFake(t, 0, `{"data":{"model":"m\u001b[2J","provider_name":"`+testKey+`",`+
		`"provider_responses":[{"provider_name":"x‮","status":"oops"}],"native_tokens_prompt":-4,"total_cost":"free"}}`)
	gen, err := quick(t, hostile.base(), time.Second).Generation(context.Background(), "gen-1")
	if err != nil {
		t.Fatalf("Generation: %v", err)
	}
	if strings.Contains(gen.Provider, testKey) || strings.ContainsAny(gen.Model, "\x1b") || strings.ContainsRune(gen.Attempts[0].Provider, '‮') {
		t.Fatalf("generation carries raw provider text: %+v", gen)
	}
	if gen.Usage != nil {
		t.Fatalf("usage = %+v, want none read from a negative count and a cost as text", gen.Usage)
	}
}

// TestAReadbackRefusesAnIDItCannotHaveIssued: an id that is not OpenRouter's
// shape is never put into a request.
func TestAReadbackRefusesAnIDItCannotHaveIssued(t *testing.T) {
	g := newGenFake(t, 0, generationRecord)
	for _, id := range []string{"", "gen-1&id=other", "gen%0a1", strings.Repeat("g", 200)} {
		if _, err := quick(t, g.base(), time.Second).Generation(context.Background(), id); err == nil {
			t.Fatalf("id %q was asked for", id)
		}
	}
	if n := g.asked.Load(); n != 0 {
		t.Fatalf("%d requests made for refused ids", n)
	}
}

// TestAReadbackThatIsNotARecordFails: a body that is not a generation record
// is a failure, and a record is never read from an error status.
func TestAReadbackThatIsNotARecordFails(t *testing.T) {
	g := newGenFake(t, 0, `[1,2,3]`)
	if _, err := quick(t, g.base(), time.Second).Generation(context.Background(), "gen-1"); err == nil {
		t.Fatal("a list was read as a generation record")
	}
	broken := newFake(t, status(http.StatusInternalServerError, generationRecord))
	_, err := quick(t, broken.base(), time.Second).Generation(context.Background(), "gen-1")
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the status refused", err)
	}
}

// TestANullInTheGenerationRecordIsNotReported: OpenRouter answers null for a
// native count the model has none of (reasoning, on a model that does not
// reason) and may for the cost; a null is a count not reported, never a
// count of zero, so it never reads as the record carrying it.
func TestANullInTheGenerationRecordIsNotReported(t *testing.T) {
	g := newGenFake(t, 0, `{"data":{"id":"gen-1","provider_name":"DeepInfra","total_cost":null,`+
		`"native_tokens_prompt":130,"native_tokens_completion":31,"native_tokens_reasoning":null,"native_tokens_cached":90}}`)
	gen, err := quick(t, g.base(), time.Second).Generation(context.Background(), "gen-1")
	if err != nil {
		t.Fatalf("Generation: %v", err)
	}
	u := gen.Usage
	if u == nil || u.PromptTokens == nil || *u.PromptTokens != 130 || u.CachedTokens == nil || *u.CachedTokens != 90 {
		t.Fatalf("usage = %+v, want the counts the record carries", u)
	}
	if u.ReasoningTokens != nil {
		t.Errorf("a null reasoning count read as %d", *u.ReasoningTokens)
	}
	if u.Cost != nil {
		t.Errorf("a null cost read as %v", *u.Cost)
	}
}

// TestANullAttemptStatusIsNotAStatusOfZero: an attempt whose status is null
// carries no status, never the status "0".
func TestANullAttemptStatusIsNotAStatusOfZero(t *testing.T) {
	g := newGenFake(t, 0, `{"data":{"id":"gen-1","provider_name":"DeepInfra",`+
		`"provider_responses":[{"provider_name":"Together","status":null},{"provider_name":"DeepInfra","status":200}]}}`)
	gen, err := quick(t, g.base(), time.Second).Generation(context.Background(), "gen-1")
	if err != nil {
		t.Fatalf("Generation: %v", err)
	}
	want := []Attempt{{Provider: "Together", Status: ""}, {Provider: "DeepInfra", Status: "200"}}
	if len(gen.Attempts) != 2 || gen.Attempts[0] != want[0] || gen.Attempts[1] != want[1] {
		t.Fatalf("attempts = %+v, want %+v", gen.Attempts, want)
	}
}
