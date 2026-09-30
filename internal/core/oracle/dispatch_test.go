package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// Provider dispatch (spc-2609251028149555 AC 3, the dispatch half, and
// itd-2609081951381895 criterion 1): a step whose route resolves to a
// configured provider is sent through that provider's adapter, with the brief
// the host sub-agent would get and the key resolved by name, and the receipt
// names the provider as the connection used. No test reaches a network: the
// provider is an httptest fake.

// dispatchKey is built at run time, so no key-shaped literal sits in the tree.
var dispatchKey = "dk-" + strings.Repeat("7c", 16) + "-not-a-real-key"

// dispatchBrief is the step as the host sub-agent would get it.
var dispatchBrief = openaiapi.Brief{Instructions: "the scribe's own prompt", Input: "the request the verb emitted"}

// pointed writes a machine configuration with a provider at base pointing
// scribe at its one listed model, stores key under the provider's credential
// name, and returns the fixture and the configuration read.
func pointed(t *testing.T, base string) (*fx, *APIConfig) {
	t.Helper()
	return configured(t, base, dispatchKey)
}

// TestAPointedRoleResolvesToItsProvider: a provider claims no tier
// (Decision 9), so a step reaches it through the role its configuration
// points there. With no --route, the role's provider is the leg, whatever
// tier the routing tables name, and the receipt names it as tried and used.
func TestAPointedRoleResolvesToItsProvider(t *testing.T) {
	f, c := pointed(t, "https://provider.example.com/v1")
	f.machine(`{"scribe":{"tier":"economy","settings":{"temperature":0}}}`)
	r, err := Resolve("scribe", f.load(), c.Connections())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.ConnectionUsed != "openrouter" || r.ConnectionTried != "openrouter" || r.Fallback != "" {
		t.Fatalf("route = used %q, tried %q, fallback %q; want the pointed provider", r.ConnectionUsed, r.ConnectionTried, r.Fallback)
	}
	if !r.OnProvider() {
		t.Fatal("OnProvider = false for a provider leg")
	}
	if string(r.SettingsSent["temperature"]) != "0" {
		t.Fatalf("settings sent = %v, want the row's temperature", r.SettingsSent)
	}
	// Nothing accepted and nothing pointed: the harness, as before.
	r, err = Resolve("intent-auditor", f.load(), c.Connections())
	if err != nil || r.ConnectionUsed != Harness || r.OnProvider() {
		t.Fatalf("an unpointed agent resolved to %q, %v; want the harness", r.ConnectionUsed, err)
	}
}

// TestARouteWithoutAConnectionKeepsItsOwnLeg: a --route governs the step for
// this run alone. Without @<connection> it names a tier, and a provider claims
// none, so a pointed agent routed that way runs through the harness: that is
// how a person sends one run of a pointed agent to the host.
func TestARouteWithoutAConnectionKeepsItsOwnLeg(t *testing.T) {
	f, c := pointed(t, "https://provider.example.com/v1")
	l := f.load()
	conns := c.Connections()
	routes, err := ParseRoutes([]string{"scribe=host-decides"}, []string{"scribe"}, conns)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(routes); err != nil {
		t.Fatal(err)
	}
	r, err := Resolve("scribe", l, conns)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.ConnectionUsed != Harness || r.ConnectionTried != "" || r.Override != "scribe=host-decides" {
		t.Fatalf("route = used %q, tried %q, override %q; want the harness under the override", r.ConnectionUsed, r.ConnectionTried, r.Override)
	}
}

// TestDispatchSendsTheStepThroughThePointedProvider is AC 3's dispatch half
// and criterion 1: the brief, the settings as sent and the key by name reach
// the provider, the answer the contract admits is the payload, and the
// receipt names the provider as used, with the call's record beside it.
func TestDispatchSendsTheStepThroughThePointedProvider(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13-20260915", `{"verdict":"keep","model":"typesafe/jev-1.13"}`))
	f, c := pointed(t, p.base())
	f.machine(`{"scribe":{"tier":"economy","settings":{"temperature":0}}}`)
	r, err := Resolve("scribe", f.load(), c.Connections())
	if err != nil {
		t.Fatal(err)
	}
	payload, rc, err := c.Dispatch(context.Background(), credential.Machine(f.roots.Home), r, dispatchBrief, verdictContract)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if string(payload) != `{"verdict":"keep","model":"typesafe/jev-1.13"}` {
		t.Fatalf("payload = %s", payload)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if p.auth.Load() != "Bearer "+dispatchKey {
		t.Fatal("the key resolved by name was not the one sent")
	}
	body := p.body.Load().(string)
	for _, want := range []string{dispatchBrief.Instructions, dispatchBrief.Input, `"temperature":0`, `"model":"typesafe/jev-1.13"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not carry %s", body, want)
		}
	}
	if rc.ConnectionUsed != "openrouter" || rc.ConnectionTried != "openrouter" || rc.FallbackReason != "" {
		t.Fatalf("receipt = %+v; want openrouter tried and used", rc)
	}
	if rc.ProviderCall == nil || *rc.ProviderCall != (CallRecord{Provider: "openrouter", ModelAsked: "typesafe/jev-1.13",
		ModelReported: "typesafe/jev-1.13-20260915", Credential: "openrouter"}) {
		t.Fatalf("provider call = %+v", rc.ProviderCall)
	}
	if rc.ModelReported != "typesafe/jev-1.13" {
		t.Fatalf("model reported = %q, want the payload's own", rc.ModelReported)
	}
	enc, _ := json.Marshal(rc)
	if strings.Contains(string(enc), dispatchKey) {
		t.Fatal("the receipt carries the key")
	}
}

// TestARepositoryRouteToAKeylessProviderDispatches: a repository may point a
// role at a provider that holds no key (a local server), and the step is sent
// there with no Authorization header.
func TestARepositoryRouteToAKeylessProviderDispatches(t *testing.T) {
	p := newProvFake(t, 200, chat("local-model", `{"verdict":"keep"}`))
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"local":{"base_url":"` + p.base() + `","models":["local-model"]}}}}`)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"local/local-model"}}}`)
	c := f.loadAPI()
	r, err := Resolve("scribe", f.load(), c.Connections())
	if err != nil {
		t.Fatal(err)
	}
	if _, rc, err := c.Dispatch(context.Background(), credential.Machine(f.roots.Home), r, dispatchBrief, verdictContract); err != nil || rc.ConnectionUsed != "local" {
		t.Fatalf("Dispatch = %+v, %v; want the local server used", rc, err)
	}
	if a := p.auth.Load(); a != "" {
		t.Fatalf("Authorization = %v, want none", a)
	}
}

// TestDispatchRefusesBeforeAnyCall: every refusal is made before the
// provider is contacted, and names the setting to change.
func TestDispatchRefusesBeforeAnyCall(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", `{"verdict":"keep"}`))
	f, c := pointed(t, p.base())
	creds := credential.Machine(f.roots.Home)
	ctx := context.Background()

	t.Run("a harness route", func(t *testing.T) {
		r, err := Resolve("intent-auditor", f.load(), c.Connections())
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = c.Dispatch(ctx, creds, r, dispatchBrief, verdictContract)
		wantAll(t, err, "intent-auditor", "harness", "oracle.roles.intent-auditor")
	})

	t.Run("a route resolved against another machine's connections", func(t *testing.T) {
		// The route names a connection this configuration's roles do not
		// point scribe at, so no model can be taken from it.
		r := Route{Agent: "scribe", Row: Row{Tier: Economy, FanOut: 1}, ConnectionTried: "elsewhere", ConnectionUsed: "elsewhere"}
		_, _, err := c.Dispatch(ctx, creds, r, dispatchBrief, verdictContract)
		wantAll(t, err, "scribe", "elsewhere", "oracle.roles.scribe", "~/.abcd/config.json")
	})

	t.Run("a keyed provider's route from anywhere but the machine", func(t *testing.T) {
		// The read refuses it (keyRoutes); a route that reached dispatch any
		// other way is refused again, so only the person's own machine route
		// spends their key.
		r, err := Resolve("scribe", f.load(), c.Connections())
		if err != nil {
			t.Fatal(err)
		}
		tgt := c.roles["scribe"]
		tgt.Origin = ".abcd/config.json"
		c.roles["scribe"] = tgt
		defer func() { tgt.Origin = "~/.abcd/config.json"; c.roles["scribe"] = tgt }()
		_, _, err = c.Dispatch(ctx, creds, r, dispatchBrief, verdictContract)
		wantAll(t, err, "oracle.roles.scribe", "~/.abcd/config.json", "openrouter")
	})

	t.Run("a key that is not set", func(t *testing.T) {
		g := newFx(t)
		g.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"` + p.base() + `","key":"openrouter",
			"models":["typesafe/jev-1.13"]}},"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`)
		gc := g.loadAPI()
		r, err := Resolve("scribe", g.load(), gc.Connections())
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = gc.Dispatch(ctx, credential.Machine(g.roots.Home), r, dispatchBrief, verdictContract)
		wantAll(t, err, `"openrouter"`, credential.Walkthrough("openrouter"))
	})

	if n := p.calls.Load(); n != 0 {
		t.Fatalf("a refused dispatch reached the provider %d time(s)", n)
	}
}

// TestDispatchNeverCarriesTheKey: whatever the provider answers, the key
// reaches no error and no receipt; a provider echoing it is scrubbed.
func TestDispatchNeverCarriesTheKey(t *testing.T) {
	for _, tc := range []struct {
		name  string
		code  int
		reply string
	}{
		{"a refusal echoing the key", 401, `{"error":{"message":"bad key ` + dispatchKey + `"}}`},
		{"a server error echoing the key", 500, `upstream said ` + dispatchKey},
		{"an answer the contract refuses, echoing the key", 200, chat("typesafe/jev-1.13", `not a verdict `+dispatchKey)},
		{"a reported model carrying the key", 200, chat(dispatchKey, `{"verdict":"keep"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProvFake(t, tc.code, tc.reply)
			f, c := pointed(t, p.base())
			r, err := Resolve("scribe", f.load(), c.Connections())
			if err != nil {
				t.Fatal(err)
			}
			payload, rc, err := c.Dispatch(context.Background(), credential.Machine(f.roots.Home), r, dispatchBrief, verdictContract)
			if err != nil && strings.Contains(err.Error(), dispatchKey) {
				t.Fatalf("the error carries the key: %v", err)
			}
			enc, _ := json.Marshal(rc)
			if strings.Contains(string(enc), dispatchKey) || strings.Contains(string(payload), dispatchKey) {
				t.Fatalf("the receipt or payload carries the key: %s", enc)
			}
		})
	}
}

// TestAnUnreachableProviderFallsBackToTheHarness is AC 4 at dispatch: a
// provider that cannot be reached before anything is sent leaves the step to
// the harness, and the route says which connection was tried and why.
func TestAnUnreachableProviderFallsBackToTheHarness(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", `{"verdict":"keep"}`))
	base := p.base()
	p.srv.Close()
	f, c := pointed(t, base)
	r, err := Resolve("scribe", f.load(), c.Connections())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.Dispatch(context.Background(), credential.Machine(f.roots.Home), r, dispatchBrief, verdictContract)
	if !errors.Is(err, openaiapi.ErrUnreachable) {
		t.Fatalf("err = %v, want one that is openaiapi.ErrUnreachable", err)
	}
	h, ok := r.FellBack(err)
	if !ok {
		t.Fatal("FellBack refused an unreachable provider")
	}
	if _, ok := r.FellBack(errors.New("openaiapi: example.com answered HTTP 500")); ok {
		t.Fatal("FellBack moved a step whose provider answered")
	}
	if h.ConnectionUsed != Harness || h.ConnectionTried != "openrouter" || h.OnProvider() || h.SettingsSent != nil {
		t.Fatalf("fallback route = %+v", h)
	}
	for _, want := range []string{"openrouter", "could not be reached", "scribe", "harness"} {
		if !strings.Contains(h.Fallback, want) {
			t.Fatalf("fallback %q does not name %q", h.Fallback, want)
		}
	}
	rc := h.Receipt("")
	if rc.ConnectionUsed != Harness || rc.ConnectionTried != "openrouter" || rc.FallbackReason != h.Fallback || rc.ProviderCall != nil {
		t.Fatalf("receipt = %+v", rc)
	}
	if strings.Contains(h.Fallback, dispatchKey) {
		t.Fatal("the fallback carries the key")
	}
}
