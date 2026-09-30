package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// The call through the adapter (criteria 1, 4 and 5): a role or judgement type
// pointed at a listed model is sent the host's brief with the key resolved by
// name, and the record names the provider, the model asked for and the model
// reported. No test here reaches a network: the provider is an httptest fake.

const callKey = "sk-or-v1-fedcba9876543210-not-a-real-key"

type provFake struct {
	srv   *httptest.Server
	calls atomic.Int32
	auth  atomic.Value
	body  atomic.Value
}

func newProvFake(t *testing.T, code int, reply string) *provFake {
	t.Helper()
	p := &provFake{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.calls.Add(1)
		p.auth.Store(r.Header.Get("Authorization"))
		raw, _ := io.ReadAll(r.Body)
		p.body.Store(string(raw))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *provFake) base() string { return p.srv.URL + "/v1" }

func chat(model, content string) string {
	b, _ := json.Marshal(map[string]any{"model": model,
		"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}})
	return string(b)
}

// configured writes a machine config pointing scribe and a self-contained
// agent (DR5) at a fake provider, and
// stores the key under its name when key is non-empty.
func configured(t *testing.T, base, key string) (*fx, *APIConfig) {
	t.Helper()
	f := newFx(t)
	keyField := ""
	if key != "" {
		keyField = `"key":"openrouter",`
		if _, err := credential.SetMachine(f.roots.Home, "openrouter", key); err != nil {
			t.Fatal(err)
		}
	}
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"` + base + `",` + keyField +
		`"models":["typesafe/jev-1.13"]}},"roles":{"scribe":"openrouter/typesafe/jev-1.13","cold-reading-detection":"openrouter/typesafe/jev-1.13"}}}`)
	return f, f.loadAPI()
}

func verdictContract(b []byte) error {
	var v struct {
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(b, &v); err != nil || v.Verdict == "" {
		return errors.New("not a verdict")
	}
	return nil
}

func TestCallSendsTheBriefToThePointedModelWithTheKeyByName(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13-20260915", `{"verdict":"keep"}`))
	f, c := configured(t, p.base(), callKey)
	tgt, ok := c.Role("scribe")
	if !ok {
		t.Fatal("scribe is not pointed")
	}
	payload, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target:   tgt,
		Brief:    openaiapi.Brief{Instructions: "the scribe's contract", Input: "the request document"},
		Settings: Settings{"temperature": json.RawMessage(`0`)},
		Contract: verdictContract,
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if string(payload) != `{"verdict":"keep"}` {
		t.Fatalf("payload = %q", payload)
	}
	want := CallRecord{Provider: "openrouter", ModelAsked: "typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-20260915", Credential: "openrouter"}
	if rec != want {
		t.Fatalf("record = %+v, want %+v", rec, want)
	}
	if p.auth.Load() != "Bearer "+callKey {
		t.Fatal("the key resolved by name was not the one sent")
	}
	body := p.body.Load().(string)
	if !strings.Contains(body, "the scribe's contract") || !strings.Contains(body, "the request document") {
		t.Fatalf("the brief did not reach the provider: %s", body)
	}
	enc, _ := json.Marshal(rec)
	if strings.Contains(string(enc), callKey) {
		t.Fatal("the record carries the key")
	}
}

// TestCallRefusesAnUnsetKeyWithoutACall is criterion 4's refusal: a named key
// that resolves to nothing refuses, naming the setup, and nothing is sent.
func TestCallRefusesAnUnsetKeyWithoutACall(t *testing.T) {
	p := newProvFake(t, 200, chat("m", `{"verdict":"keep"}`))
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"` + p.base() + `","key":"openrouter","models":["typesafe/jev-1.13"]}}}}`)
	c := f.loadAPI()
	_, _, err := c.Call(context.Background(), credential.Store(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: "typesafe/jev-1.13"}, Contract: verdictContract})
	if err == nil || !strings.Contains(err.Error(), `"openrouter"`) || !strings.Contains(err.Error(), credential.Walkthrough("openrouter")) {
		t.Fatalf("err = %v, want a refusal naming the credential and the setup", err)
	}
	if n := p.calls.Load(); n != 0 {
		t.Fatalf("an unauthenticated call was made %d time(s)", n)
	}
}

// TestCallRefusesAnUnadmittedTargetWithoutACall: a target that did not come
// from the read (a model its provider does not list) is refused again at the
// call.
func TestCallRefusesAnUnadmittedTargetWithoutACall(t *testing.T) {
	p := newProvFake(t, 200, chat("m", `{"verdict":"keep"}`))
	f, c := configured(t, p.base(), callKey)
	for _, model := range []string{"typesafe/jev-2", "anthropic/claude-opus-4"} {
		_, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
			Target: Target{Provider: "openrouter", Model: model}, Contract: verdictContract})
		if err == nil {
			t.Fatalf("%s: Call succeeded", model)
		}
	}
	if n := p.calls.Load(); n != 0 {
		t.Fatalf("a refused target reached the provider %d time(s)", n)
	}
}

// TestCallServesAListedModelOfAnyVendor: with no vendor denylist bundled
// (adr-2609300107513982), a model the provider lists is called and its answer
// used, whichever vendor made it; the record names what was asked and what
// answered.
func TestCallServesAListedModelOfAnyVendor(t *testing.T) {
	p := newProvFake(t, 200, chat("anthropic/claude-opus-4", `{"verdict":"keep"}`))
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "openrouter", callKey); err != nil {
		t.Fatal(err)
	}
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"` + p.base() + `","key":"openrouter",
		"models":["anthropic/claude-opus-4"]}},"roles":{"scribe":"openrouter/anthropic/claude-opus-4"}}}`)
	c := f.loadAPI()
	payload, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: "anthropic/claude-opus-4"}, Contract: verdictContract})
	if err != nil || string(payload) != `{"verdict":"keep"}` {
		t.Fatalf("Call = %q, %v; want the listed model's answer", payload, err)
	}
	if rec.ModelAsked != "anthropic/claude-opus-4" || rec.ModelReported != "anthropic/claude-opus-4" {
		t.Fatalf("record = %+v", rec)
	}
}

// TestCallRefusesADeniedReportedModel: an aggregator that answers with a model
// the configuration's own denylist refuses has substituted it; the answer is
// not used, and the refusal names what it reported and the entry.
func TestCallRefusesADeniedReportedModel(t *testing.T) {
	p := newProvFake(t, 200, chat("anthropic/claude-opus-4", `{"verdict":"keep"}`))
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "openrouter", callKey); err != nil {
		t.Fatal(err)
	}
	f.machineConfig(`{"oracle":{"denylist":["anthropic/*"],"api":{"openrouter":{"base_url":"` + p.base() + `","key":"openrouter",
		"models":["typesafe/jev-1.13"]}}}}`)
	c := f.loadAPI()
	payload, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: "typesafe/jev-1.13"}, Contract: verdictContract})
	if err == nil || payload != nil {
		t.Fatalf("Call = %q, %v; want a refusal", payload, err)
	}
	if !strings.Contains(err.Error(), "anthropic/claude-opus-4") || !strings.Contains(err.Error(), "(anthropic/*, from ~/.abcd/config.json)") {
		t.Fatalf("err = %v", err)
	}
}

// TestCallWithNoKeySendsNone: a local server's block names no key, and the
// call carries no Authorization header.
func TestCallWithNoKeySendsNone(t *testing.T) {
	p := newProvFake(t, 200, chat("local", `{"verdict":"keep"}`))
	f, c := configured(t, p.base(), "")
	if _, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: "typesafe/jev-1.13"}, Contract: verdictContract}); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if a := p.auth.Load(); a != "" {
		t.Fatalf("Authorization = %v, want none", a)
	}
}

// TestCallErrorsNeverCarryTheKey: a provider echoing the key in its refusal
// does not put it in the error.
func TestCallErrorsNeverCarryTheKey(t *testing.T) {
	p := newProvFake(t, 401, `{"error":{"message":"bad key `+callKey+`"}}`)
	f, c := configured(t, p.base(), callKey)
	_, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "openrouter", Model: "typesafe/jev-1.13"}, Contract: verdictContract})
	if err == nil || strings.Contains(err.Error(), callKey) {
		t.Fatalf("err = %v", err)
	}
}

// TestTheReceiptCarriesTheProviderCall is criterion 5's record: a step's
// receipt names the provider, the model asked for and the model reported; on
// the harness leg the member is present and null, so "no call" is never
// mistaken for "not recorded".
func TestTheReceiptCarriesTheProviderCall(t *testing.T) {
	l := newFx(t).load()
	r, err := Resolve("scribe", l, NoConnections{})
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := json.Marshal(r.Receipt(""))
	if !strings.Contains(string(enc), `"provider_call":null`) {
		t.Fatalf("harness receipt = %s", enc)
	}
	rec := CallRecord{Provider: "openrouter", ModelAsked: "typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-20260915", Credential: "openrouter"}
	enc, _ = json.Marshal(r.Receipt("").WithCall(rec))
	// The run's record names the credential the call used, and never a value
	// (itd-2609221017023290 criterion 5).
	for _, want := range []string{`"provider_call":{"provider":"openrouter","model_asked":"typesafe/jev-1.13","model_reported":"typesafe/jev-1.13-20260915","credential":"openrouter"}`} {
		if !strings.Contains(string(enc), want) {
			t.Fatalf("receipt = %s, want %s", enc, want)
		}
	}
}
