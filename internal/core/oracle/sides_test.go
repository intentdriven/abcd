package oracle

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// The tests in this file are itd-166's oracle half: a call through the
// provider adapter records the side abcd asked for and the side that
// answered, and lists every field of the fixed lists it could not pin with
// its reason; the harness hand-off's receipt lists the payload's model as
// self-reported.

// openRouterBody is an OpenRouter-shaped chat completion: its generation id,
// the upstream provider, the finish reason and a usage member carrying the
// token details and the cost.
const openRouterBody = `{"id":"gen-1760000000-abcdef","provider":"Fireworks","model":"typesafe/jev-1.13-20260915",` +
	`"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"{\"verdict\":\"keep\"}"}}],` +
	`"usage":{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,` +
	`"prompt_tokens_details":{"cached_tokens":100,"cache_write_tokens":20},` +
	`"completion_tokens_details":{"reasoning_tokens":12},"cost":0.00042}}`

var sidesBrief = openaiapi.Brief{Instructions: "the scribe's contract", Input: "the request document"}

// assertListed holds every entry of a record's list to the fixed lists and
// the four reasons, each field at most once.
func assertListed(t *testing.T, us []Unpinned) {
	t.Helper()
	known := map[Field]bool{}
	for _, f := range append(append(append([]Field{}, AskedFields...), LocalFields...), ServedFields...) {
		known[f] = true
	}
	seen := map[Field]bool{}
	for _, u := range us {
		if !known[u.Field] {
			t.Errorf("listed field %q is not on the fixed lists", u.Field)
		}
		switch u.Reason {
		case NotReported, SelfReported, HarnessComputed, NotConfigured:
		default:
			t.Errorf("field %s has reason %q, outside the four", u.Field, u.Reason)
		}
		if seen[u.Field] {
			t.Errorf("field %s is listed twice", u.Field)
		}
		seen[u.Field] = true
	}
}

// TestAProviderCallRecordsBothSides is criterion 1: the record carries the
// asked side (the model, the prompt's SHA-256 and the field the adapter added)
// and the served side (the model, upstream provider, generation id, finish
// reason, every token count and the cost) the response names, and lists
// exactly the fixed fields neither carries.
func TestAProviderCallRecordsBothSides(t *testing.T) {
	p := Provider{Name: "openrouter", BaseURL: "https://openrouter.ai/api/v1"}
	res := openaiapi.Result{ModelAsked: "typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-20260915",
		FinishReason: "stop", GenerationID: "gen-1", Provider: "Fireworks"}
	n := func(v int) *int { return &v }
	cost := 0.00042
	res.Usage = &openaiapi.Usage{PromptTokens: 120, CompletionTokens: 30, TotalTokens: 150,
		CachedTokens: n(100), CacheWriteTokens: n(20), ReasoningTokens: n(12), Cost: &cost}
	rec := callRecord(p, sidesBrief, Settings{"temperature": raw("0")}, res)
	if rec.PromptSHA256 != openaiapi.PromptSHA256(sidesBrief) || rec.ServedProvider != "Fireworks" ||
		rec.GenerationID != "gen-1" || rec.FinishReason != "stop" {
		t.Fatalf("record = %+v", rec)
	}
	if rec.Usage == nil || *rec.Usage != (Usage{PromptTokens: 120, CompletionTokens: 30, ReasoningTokens: 12, CacheReadTokens: 100, CacheWriteTokens: 20}) {
		t.Fatalf("usage = %+v", rec.Usage)
	}
	if rec.Cost == nil || *rec.Cost != (Cost{Amount: 0.00042, Unit: CostCredits}) {
		t.Fatalf("cost = %+v", rec.Cost)
	}
	if !reflect.DeepEqual(rec.AdapterAdded, []string{openaiapi.StreamUsageField}) {
		t.Fatalf("adapter added = %v", rec.AdapterAdded)
	}
	want := []Unpinned{
		{AskedProviderPin, NotConfigured}, {AskedTopP, NotConfigured}, {AskedSeed, NotConfigured},
		{AskedEffort, NotConfigured}, {AskedContext, NotConfigured}, {AskedHarnessVersion, NotReported},
	}
	if !reflect.DeepEqual(rec.Unpinned, want) {
		t.Fatalf("unpinned = %v, want %v", rec.Unpinned, want)
	}
	assertListed(t, rec.Unpinned)
}

// TestACallNamingNoServedDetailListsEachMissing is criterion 4 on the
// adapter: an answer that names only its model leaves every other served
// field listed as not reported, and a local route lists its weights,
// quantisation and runtime as not configured.
func TestACallNamingNoServedDetailListsEachMissing(t *testing.T) {
	prov := newProvFake(t, 200, chat("typesafe/jev-1.13-20260915", `{"verdict":"keep"}`))
	f, c := configured(t, prov.base(), callKey)
	tgt, _ := c.Role("scribe")
	_, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{Target: tgt, Brief: sidesBrief})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Usage != nil || rec.Cost != nil {
		t.Fatalf("usage %+v, cost %+v: want neither", rec.Usage, rec.Cost)
	}
	got := map[Field]UnpinnedReason{}
	for _, u := range rec.Unpinned {
		got[u.Field] = u.Reason
	}
	for _, f := range ServedFields {
		if f == ServedModel {
			if _, listed := got[f]; listed {
				t.Errorf("the served model the provider reported is listed: %v", rec.Unpinned)
			}
			continue
		}
		if got[f] != NotReported {
			t.Errorf("%s = %q, want not_reported", f, got[f])
		}
	}
	for _, f := range LocalFields {
		if got[f] != NotConfigured {
			t.Errorf("%s on a local route = %q, want not_configured", f, got[f])
		}
	}
	for _, f := range []Field{AskedTemperature, AskedTopP, AskedSeed} {
		if got[f] != NotConfigured {
			t.Errorf("%s with no setting sent = %q, want not_configured", f, got[f])
		}
	}
	assertListed(t, rec.Unpinned)
}

// TestTheCallRecordTravelsInTheReceipt: a dispatched step's receipt carries
// the call's both sides, read from an OpenRouter-shaped answer.
func TestTheCallRecordTravelsInTheReceipt(t *testing.T) {
	prov := newProvFake(t, 200, openRouterBody)
	f, c := configured(t, prov.base(), callKey)
	tgt, _ := c.Role("scribe")
	_, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{Target: tgt, Brief: sidesBrief})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ServedProvider != "Fireworks" || rec.GenerationID != "gen-1760000000-abcdef" || rec.Usage == nil || rec.Cost == nil {
		t.Fatalf("record = %+v", rec)
	}
	if !strings.Contains(prov.body.Load().(string), `"stream_options":{"include_usage":true}`) {
		t.Fatalf("the request did not ask for usage in the stream: %s", prov.body.Load())
	}
	enc, _ := json.Marshal(ReceiptRoute{}.WithCall(rec))
	for _, k := range []string{`"prompt_sha256":"`, `"served_provider":"Fireworks"`, `"generation_id":"gen-1760000000-abcdef"`,
		`"finish_reason":"stop"`, `"cost":{"amount":0.00042,"unit":"credits"}`, `"reasoning_tokens":12`, `"adapter_added":["stream_options.include_usage"]`} {
		if !strings.Contains(string(enc), k) {
			t.Errorf("receipt lacks %s: %s", k, enc)
		}
	}
}

// TestAConnectionThatTurnsStreamUsageOffSendsWithoutIt is criterion 6: a
// provider block with stream_usage false sends no stream_options, the call
// succeeds with usage listed as not reported, and nothing is recorded as
// added by the adapter.
func TestAConnectionThatTurnsStreamUsageOffSendsWithoutIt(t *testing.T) {
	prov := newProvFake(t, 200, chat("typesafe/jev-1.13-20260915", `{"verdict":"keep"}`))
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"local":{"base_url":"` + prov.base() + `","stream_usage":false,` +
		`"models":["typesafe/jev-1.13"]}},"roles":{"scribe":"local/typesafe/jev-1.13"}}}`)
	c := f.loadAPI()
	tgt, _ := c.Role("scribe")
	_, rec, err := c.Call(context.Background(), nil, CallRequest{Target: tgt, Brief: sidesBrief})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if strings.Contains(prov.body.Load().(string), "stream_options") {
		t.Fatalf("a connection with stream_usage false asked for it: %s", prov.body.Load())
	}
	if len(rec.AdapterAdded) != 0 {
		t.Fatalf("adapter added = %v, want none", rec.AdapterAdded)
	}
	listed := false
	for _, u := range rec.Unpinned {
		listed = listed || (u.Field == ServedPromptTokens && u.Reason == NotReported)
	}
	if !listed {
		t.Fatalf("usage is not listed as not reported: %v", rec.Unpinned)
	}
}

// TestARefusedUsageRequestNamesTheSetting is criterion 6's refusal: a server
// that refuses the request while it asked for usage in the stream is refused
// with the setting that stops asking named.
func TestARefusedUsageRequestNamesTheSetting(t *testing.T) {
	prov := newProvFake(t, 400, `{"error":{"message":"Unrecognized request argument supplied: stream_options"}}`)
	f, c := configured(t, prov.base(), callKey)
	tgt, _ := c.Role("scribe")
	_, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{Target: tgt, Brief: sidesBrief})
	if err == nil || !strings.Contains(err.Error(), "oracle.api.openrouter.stream_usage") {
		t.Fatalf("err = %v, want it to name oracle.api.openrouter.stream_usage", err)
	}
	if prov.calls.Load() != 1 {
		t.Fatalf("the provider was called %d times; a refusal is never retried", prov.calls.Load())
	}
}

// TestStreamUsageMustBeABoolean: the block's stream_usage is read strictly.
func TestStreamUsageMustBeABoolean(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"local":{"base_url":"http://localhost:8000/v1","stream_usage":"no","models":["m"]}}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "stream_usage") {
		t.Fatalf("err = %v, want it to name stream_usage", err)
	}
}

// TestTheHostHandOffMarksTheServedModelSelfReported is criterion 5: on the
// harness leg the receipt's model is the payload's own declaration, listed
// as self-reported, and every other field of the fixed lists is listed with
// its reason; a payload naming no model lists it as not reported.
func TestTheHostHandOffMarksTheServedModelSelfReported(t *testing.T) {
	r, err := Resolve("scribe", newFx(t).load(), NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	rc := r.Receipt("claude-example-1")
	assertListed(t, rc.Unpinned)
	got := map[Field]UnpinnedReason{}
	for _, u := range rc.Unpinned {
		got[u.Field] = u.Reason
	}
	if got[ServedModel] != SelfReported {
		t.Fatalf("served model = %q, want self_reported: %v", got[ServedModel], rc.Unpinned)
	}
	if len(rc.Unpinned) != len(AskedFields)+len(ServedFields) {
		t.Fatalf("listed %d fields, want all %d of the fixed lists: %v", len(rc.Unpinned), len(AskedFields)+len(ServedFields), rc.Unpinned)
	}
	enc, _ := json.Marshal(rc)
	if !strings.Contains(string(enc), `{"field":"served.model","reason":"self_reported"}`) {
		t.Fatalf("receipt = %s", enc)
	}
	for _, u := range r.Receipt("").Unpinned {
		if u.Field == ServedModel && u.Reason != NotReported {
			t.Fatalf("a payload naming no model lists it as %q", u.Reason)
		}
	}
}
