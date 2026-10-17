package oracle

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// The tests in this file are iss-2610161734413178's oracle half: a provider
// block on the machine can pin a listed model to one OpenRouter provider
// (itd-2610161733316518 decisions 5 and 6), the adapter sends the pin and
// records it on the asked side, and a caller that opts in reads OpenRouter's
// generation record into the served side, which is off by default because it
// is a request of its own (decision 12). OpenRouter is an httptest fake.

const pinModel = "meta-llama/llama-3.1-8b-instruct"

// pinBlock is an OpenRouter block at base pinning pinModel with pins.
func pinBlock(base, pins string) string { return pinBlockReading(base, pins, false) }

// pinBlockReading is pinBlock, setting read_generation when read.
func pinBlockReading(base, pins string, read bool) string {
	extra := ""
	if read {
		extra = `,"read_generation":true`
	}
	return `{"oracle":{"api":{"openrouter":{"base_url":"` + base + `","key":"openrouter","models":["` + pinModel + `","~vendor/latest"],` +
		`"pins":` + pins + extra + `}},"roles":{"scribe":"openrouter/` + pinModel + `"}}}`
}

const goodPin = `{"` + pinModel + `":{"order":["deepinfra"],"quantizations":["fp8"],"canonical_slug":"meta-llama/llama-3.1-8b-instruct-20240723"}}`

// asOpenRouter makes the test's fake count as OpenRouter for the length of
// the test: a pin and a readback are OpenRouter's alone.
func asOpenRouter(t *testing.T) {
	t.Helper()
	was := openRouterHost
	openRouterHost = "127.0.0.1"
	t.Cleanup(func() { openRouterHost = was })
}

// TestAPinIsReadWithTheSettledFields: a pin names one provider, one
// quantisation and the canonical slug; fallbacks off, required parameters and
// data collection denied are the pin's own, never configurable.
func TestAPinIsReadWithTheSettledFields(t *testing.T) {
	f := newFx(t)
	f.machineConfig(pinBlock("https://openrouter.ai/api/v1", goodPin))
	c := f.loadAPI()
	pin, ok := c.PinFor("openrouter", pinModel)
	want := Pin{Order: []string{"deepinfra"}, AllowFallbacks: false, RequireParameters: true, DataCollection: "deny",
		Quantizations: []string{"fp8"}, CanonicalSlug: "meta-llama/llama-3.1-8b-instruct-20240723"}
	if !ok || !reflect.DeepEqual(pin, want) {
		t.Fatalf("PinFor = %+v, %v; want %+v", pin, ok, want)
	}
	if _, ok := c.PinFor("openrouter", "~vendor/latest"); ok {
		t.Fatal("an unpinned model has a pin")
	}
	r := pin.Routing()
	if r.AllowFallbacks || !r.RequireParameters || r.DataCollection != "deny" || r.Order[0] != "deepinfra" || r.Quantizations[0] != "fp8" {
		t.Fatalf("routing = %+v", r)
	}
}

// TestAPinThatCannotHoldIsRefused: each pin that would not hold a call to one
// provider and one endpoint, or that names what the block does not serve, is
// refused when the configuration is read, naming the fault.
func TestAPinThatCannotHoldIsRefused(t *testing.T) {
	slug := `"canonical_slug":"meta-llama/llama-3.1-8b-instruct-20240723"`
	for _, tc := range []struct{ name, base, pins, want string }{
		{"not OpenRouter", "https://api.example.com/v1", goodPin, "OpenRouter"},
		{"unlisted model", "https://openrouter.ai/api/v1", `{"other/model":{"order":["a"],"quantizations":["fp8"],` + slug + `}}`, "not on its models"},
		{"alias model", "https://openrouter.ai/api/v1", `{"~vendor/latest":{"order":["a"],"quantizations":["fp8"],` + slug + `}}`, "alias"},
		{"two providers", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a","b"],"quantizations":["fp8"],` + slug + `}}`, "exactly one provider"},
		{"no provider", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":[],"quantizations":["fp8"],` + slug + `}}`, "exactly one provider"},
		{"odd provider", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["{env:HOME}"],"quantizations":["fp8"],` + slug + `}}`, "provider slug"},
		{"two quantisations", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a"],"quantizations":["fp8","bf16"],` + slug + `}}`, "exactly one quantization"},
		{"unknown quantisation", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a"],"quantizations":["fp7"],` + slug + `}}`, "fp7"},
		{"no slug", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a"],"quantizations":["fp8"]}}`, "canonical_slug"},
		{"alias slug", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a"],"quantizations":["fp8"],"canonical_slug":"~a/b"}}`, "canonical_slug"},
		{"fallbacks set", "https://openrouter.ai/api/v1", `{"` + pinModel + `":{"order":["a"],"quantizations":["fp8"],"allow_fallbacks":true,` + slug + `}}`, "allow_fallbacks"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(pinBlock(tc.base, tc.pins))
			if err := f.loadAPIErr(); !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to name %q", err, tc.want)
			}
		})
	}
}

// orFake is OpenRouter: a chat endpoint answering body, and the generation
// record endpoint answering genStatus and genBody, each counting its calls.
type orFake struct {
	srv       *httptest.Server
	chats     atomic.Int32
	gens      atomic.Int32
	chatBody  atomic.Value
	genAuth   atomic.Value
	chatAuth  atomic.Value
	genStatus int
	genBody   string
}

func newORFake(t *testing.T, genStatus int, genBody string) *orFake {
	t.Helper()
	o := &orFake{genStatus: genStatus, genBody: genBody}
	o.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/generation"):
			o.gens.Add(1)
			o.genAuth.Store(r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(o.genStatus)
			_, _ = io.WriteString(w, o.genBody)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/chat/completions"):
			o.chats.Add(1)
			o.chatAuth.Store(r.Header.Get("Authorization"))
			raw, _ := io.ReadAll(r.Body)
			o.chatBody.Store(string(raw))
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, openRouterBody)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(o.srv.Close)
	return o
}

const genRecord = `{"data":{"id":"gen-1760000000-abcdef","model":"meta-llama/llama-3.1-8b-instruct","provider_name":"DeepInfra",` +
	`"total_cost":0.0007,"native_tokens_prompt":121,"native_tokens_completion":33,"native_tokens_reasoning":5,"native_tokens_cached":64,` +
	`"provider_responses":[{"provider_name":"DeepInfra","status":200}]}}`

// pinnedCall configures OpenRouter at o with goodPin and the key stored by
// name, its block setting read_generation when read, and makes one call to
// pinModel: the caller asks for nothing beyond the call itself.
func pinnedCall(t *testing.T, o *orFake, read bool) CallRecord {
	t.Helper()
	asOpenRouter(t)
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "openrouter", callKey); err != nil {
		t.Fatal(err)
	}
	f.machineConfig(pinBlockReading(o.srv.URL+"/api/v1", goodPin, read))
	c := f.loadAPI()
	_, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: pinModel}, Brief: sidesBrief})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	return rec
}

// listed reports whether rec lists field f, and with which reason.
func listed(rec CallRecord, f Field) (UnpinnedReason, bool) {
	for _, u := range rec.Unpinned {
		if u.Field == f {
			return u.Reason, true
		}
	}
	return "", false
}

// TestAPinnedCallSendsAndRecordsThePin: the request carries the routing
// object, and the record's asked side carries the pin, so the provider pin is
// no longer listed as not configured.
func TestAPinnedCallSendsAndRecordsThePin(t *testing.T) {
	o := newORFake(t, http.StatusOK, genRecord)
	rec := pinnedCall(t, o, false)
	var body map[string]json.RawMessage
	_ = json.Unmarshal([]byte(o.chatBody.Load().(string)), &body)
	want := `{"order":["deepinfra"],"allow_fallbacks":false,"require_parameters":true,"data_collection":"deny","quantizations":["fp8"]}`
	if got := string(body["provider"]); got != want {
		t.Fatalf("provider = %s, want %s", got, want)
	}
	if rec.ProviderPin == nil || rec.ProviderPin.CanonicalSlug != "meta-llama/llama-3.1-8b-instruct-20240723" || rec.ProviderPin.Order[0] != "deepinfra" {
		t.Fatalf("provider pin = %+v", rec.ProviderPin)
	}
	if r, ok := listed(rec, AskedProviderPin); ok {
		t.Fatalf("a pinned call lists the provider pin as %s", r)
	}
	assertListed(t, rec.Unpinned)
}

// TestTheReadbackIsOffByDefault is decision 12: a call on a block that does
// not set read_generation makes no request to the generation record, and
// says it did not ask.
func TestTheReadbackIsOffByDefault(t *testing.T) {
	o := newORFake(t, http.StatusOK, genRecord)
	rec := pinnedCall(t, o, false)
	if n := o.gens.Load(); n != 0 {
		t.Fatalf("%d generation requests on a call that did not opt in", n)
	}
	if rec.GenerationReadback != ReadbackNotAsked || rec.Attempts != nil {
		t.Fatalf("readback = %q, attempts %v", rec.GenerationReadback, rec.Attempts)
	}
	if rec.ServedProvider != "Fireworks" {
		t.Fatalf("served provider = %q, want the response's own", rec.ServedProvider)
	}
}

// TestAnOptedInReadbackFillsTheServedSide: on a block that sets
// read_generation, the record's served provider, the attempts, the native
// token counts and the cost replace what the response carried, with the key
// the call used, read by name.
func TestAnOptedInReadbackFillsTheServedSide(t *testing.T) {
	o := newORFake(t, http.StatusOK, genRecord)
	rec := pinnedCall(t, o, true)
	if n := o.gens.Load(); n != 1 {
		t.Fatalf("%d generation requests, want 1", n)
	}
	if a := o.genAuth.Load(); a != "Bearer "+callKey || a != o.chatAuth.Load() {
		t.Fatalf("readback authorization = %v", a)
	}
	if rec.GenerationReadback != ReadbackRead || rec.ServedProvider != "DeepInfra" {
		t.Fatalf("readback %q, served provider %q", rec.GenerationReadback, rec.ServedProvider)
	}
	if len(rec.Attempts) != 1 || rec.Attempts[0].Provider != "DeepInfra" || rec.Attempts[0].Status != "200" {
		t.Fatalf("attempts = %+v", rec.Attempts)
	}
	// The record carries its full set of native counts, so they replace the
	// response's all together; cache writes are not among them, and the
	// response's count, from the other tokenizer, is not kept beside them.
	if rec.Usage == nil || *rec.Usage != (Usage{PromptTokens: 121, CompletionTokens: 33, ReasoningTokens: 5, CacheReadTokens: 64}) {
		t.Fatalf("usage = %+v", rec.Usage)
	}
	if r, ok := listed(rec, ServedCacheWriteTokens); !ok || r != NotReported {
		t.Fatalf("cache writes listed as %q, %v; want not reported", r, ok)
	}
	if rec.Cost == nil || *rec.Cost != (Cost{Amount: 0.0007, Unit: CostCredits}) {
		t.Fatalf("cost = %+v", rec.Cost)
	}
	assertListed(t, rec.Unpinned)
}

// TestAFailedReadbackNeverFailsTheCall: a generation record the provider
// will not give leaves the call answered, says the readback failed, and keeps
// the response's own served side.
func TestAFailedReadbackNeverFailsTheCall(t *testing.T) {
	o := newORFake(t, http.StatusInternalServerError, `{"error":{"message":"down"}}`)
	rec := pinnedCall(t, o, true)
	if rec.GenerationReadback != ReadbackFailed || rec.GenerationReadbackError == "" {
		t.Fatalf("readback %q (%q), want failed with its reason", rec.GenerationReadback, rec.GenerationReadbackError)
	}
	if rec.ServedProvider != "Fireworks" || rec.Attempts != nil {
		t.Fatalf("served provider %q, attempts %v", rec.ServedProvider, rec.Attempts)
	}
}

// TestAReadbackOffOpenRouterIsRefused: a provider that is not OpenRouter has
// no generation record to read, so a block there that sets read_generation
// is refused when the configuration is read, as a pin is; false is the
// default and is read anywhere.
func TestAReadbackOffOpenRouterIsRefused(t *testing.T) {
	o := newORFake(t, http.StatusOK, genRecord)
	f := newFx(t)
	block := func(read string) string {
		return `{"oracle":{"api":{"local":{"base_url":"` + o.srv.URL + `/v1","models":["` + pinModel + `"],"read_generation":` + read + `}}}}`
	}
	f.machineConfig(block("true"))
	err := f.loadAPIErr()
	for _, want := range []string{"oracle.api.local", "read_generation", "OpenRouter"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	f.machineConfig(block("false"))
	if p := f.loadAPI().providers["local"]; p.ReadGeneration {
		t.Fatal("read_generation false reads as set")
	}
}

// TestReadGenerationIsTheMachinesAlone: like every member of a provider
// block, read_generation is refused from a repository's configuration, which
// cannot make a person's calls send a request of their own.
func TestReadGenerationIsTheMachinesAlone(t *testing.T) {
	asOpenRouter(t)
	o := newORFake(t, http.StatusOK, genRecord)
	f := newFx(t)
	f.machineConfig(pinBlock(o.srv.URL+"/api/v1", goodPin))
	f.repoConfig(`{"oracle":{"api":{"openrouter":{"read_generation":true}}}}`)
	err := f.loadAPIErr()
	for _, want := range []string{".abcd/config.json (repo layer)", "oracle.api", "refused"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
}

// TestAPartialGenerationRecordNeverMixesCounts: the response's counts and the
// generation record's native ones come from two tokenizers, so the record's
// replace the response's only all together, when it carries every count it
// has a field for; a record missing one leaves the response's counts as they
// were. Its cost is taken on its own, either way.
func TestAPartialGenerationRecordNeverMixesCounts(t *testing.T) {
	n := func(v int) *int { return &v }
	c := func(v float64) *float64 { return &v }
	resp := openaiapi.Result{Usage: &openaiapi.Usage{PromptTokens: n(120), CompletionTokens: n(30), TotalTokens: n(150),
		ReasoningTokens: n(12), CachedTokens: n(100), CacheWriteTokens: n(20), Cost: c(0.00042)}}
	partial := openaiapi.Generation{Usage: &openaiapi.Usage{PromptTokens: n(121), CompletionTokens: n(33), Cost: c(0.0007)}}
	got := withGeneration(resp, partial).Usage
	want := openaiapi.Usage{PromptTokens: n(120), CompletionTokens: n(30), TotalTokens: n(150),
		ReasoningTokens: n(12), CachedTokens: n(100), CacheWriteTokens: n(20), Cost: c(0.0007)}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("partial record: usage = %s, want %s", usageText(*got), usageText(want))
	}
	if *resp.Usage.Cost != 0.00042 {
		t.Fatal("the response's usage was changed underneath it")
	}
	full := openaiapi.Generation{Usage: &openaiapi.Usage{PromptTokens: n(121), CompletionTokens: n(33),
		ReasoningTokens: n(5), CachedTokens: n(64)}}
	got = withGeneration(resp, full).Usage
	want = openaiapi.Usage{PromptTokens: n(121), CompletionTokens: n(33), ReasoningTokens: n(5), CachedTokens: n(64), Cost: c(0.00042)}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("full record: usage = %s, want %s", usageText(*got), usageText(want))
	}
}

// usageText spells a usage's counts for a failure message.
func usageText(u openaiapi.Usage) string {
	raw, _ := json.Marshal(u)
	return string(raw)
}
