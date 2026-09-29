package oracle

import (
	"strings"
	"testing"
)

// The refusals Resolve makes on a provider leg (spc-2609251028149555, AC 8's
// adapter half and the allowlist half of AC 11 that no model choice decides):
// each is an error before the step runs, never a silent drop, and each names
// what it refuses, where that came from and the remedy.

// wantAll fails unless err is non-nil and names every one of parts.
func wantAll(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Resolve returned no error; want a refusal naming %q", parts)
	}
	for _, p := range parts {
		if !strings.Contains(err.Error(), p) {
			t.Fatalf("refusal %q does not name %q", err, p)
		}
	}
}

// TestResolveRefusesASettingTheAdapterDoesNotAccept is AC 8's adapter half.
//
// Given a provider connection whose adapter accepts temperature and seed,
// when a step resolves to it carrying a setting outside that set, from any of
// the three places a setting comes from,
// then Resolve refuses before the step runs, naming the setting, the
// connection, where the setting was set and what the adapter accepts.
func TestResolveRefusesASettingTheAdapterDoesNotAccept(t *testing.T) {
	accepts := []string{"seed", "temperature"}
	models := []string{"example/model-1"}

	t.Run("from the routing row", func(t *testing.T) {
		// Given a repository row setting top_k, which the adapter does not accept
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"economy","settings":{"temperature":0.2,"top_k":40}}}`)
		l := f.load()
		c := Connection{Name: "openrouter", Models: models, Accepts: accepts}
		s := &spy{named: map[string]Connection{"openrouter": c}}
		routes, err := ParseRoutes([]string{"scribe=economy@openrouter"}, []string{"scribe"}, s)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		// When the step resolves
		_, err = Resolve("scribe", l, s)
		// Then it is refused naming the setting, its file, the connection and the accepted set
		wantAll(t, err, "top_k", "oracle-routing.json", "openrouter", "seed, temperature", "remove")
		if strings.Contains(err.Error(), "temperature (from") {
			t.Fatalf("refusal %q names an accepted setting as refused", err)
		}
	})

	t.Run("from the --route", func(t *testing.T) {
		f := newFx(t)
		l := f.load()
		c := Connection{Name: "openrouter", Models: models, Accepts: accepts}
		s := &spy{named: map[string]Connection{"openrouter": c}}
		routes, err := ParseRoutes([]string{"scribe=economy@openrouter?top_k=40"}, []string{"scribe"}, s)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		_, err = Resolve("scribe", l, s)
		wantAll(t, err, "top_k", "--route scribe=economy@openrouter?top_k=40", "openrouter", "seed, temperature")
	})

	t.Run("from the connection's defaults", func(t *testing.T) {
		f := newFx(t)
		f.machine(`{"scribe":{"tier":"local"}}`)
		l := f.load()
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Defaults: Settings{"mirostat": raw("2")}}
		// When the tier itself proposes the provider leg
		s := &spy{serves: map[Tier]Connection{Local: c}}
		_, err := Resolve("scribe", l, s)
		wantAll(t, err, "mirostat", "desk's defaults", "tier local")
	})

	t.Run("a connection no adapter backs accepts no setting", func(t *testing.T) {
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"local","settings":{"seed":1}}}`)
		l := f.load()
		s := &spy{serves: map[Tier]Connection{Local: {Name: "desk", Models: models}}}
		_, err := Resolve("scribe", l, s)
		wantAll(t, err, "seed", "desk", "accepts no setting")
	})

	t.Run("every accepted setting passes", func(t *testing.T) {
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"local","settings":{"seed":1}}}`)
		l := f.load()
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Defaults: Settings{"temperature": raw("0")}}
		r, err := Resolve("scribe", l, &spy{serves: map[Tier]Connection{Local: c}})
		if err != nil {
			t.Fatal(err)
		}
		if r.ConnectionUsed != "desk" || len(r.SettingsSent) != 2 {
			t.Fatalf("%+v", r)
		}
	})

	t.Run("the harness leg sends no setting and refuses none", func(t *testing.T) {
		// A row's settings on a step that goes to the harness are not sent to
		// any adapter, so no adapter's set judges them.
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"local","settings":{"top_k":40}}}`)
		r, err := Resolve("scribe", f.load(), NoConnections{})
		if err != nil || r.ConnectionUsed != Harness || r.SettingsSent != nil {
			t.Fatalf("%+v, %v", r, err)
		}
	})
}

// TestResolveRefusesAProviderLegWhoseAllowlistListsNothing is the half of AC
// 11 that does not wait on how a route names its model: a provider serves only
// the models it lists (adr-2609221009491186), so a connection that lists none
// admits no route at all.
//
// Given a provider connection with an empty allowlist,
// when a --route names it or the tier proposes it,
// then Resolve refuses before any settings merge, naming the connection, the
// route and the remedy, instead of returning a provider leg.
func TestResolveRefusesAProviderLegWhoseAllowlistListsNothing(t *testing.T) {
	t.Run("named by --route", func(t *testing.T) {
		f := newFx(t)
		l := f.load()
		c := Connection{Name: "desk", Accepts: []string{"seed"}}
		s := &spy{named: map[string]Connection{"desk": c}}
		routes, err := ParseRoutes([]string{"scribe=local@desk?seed=1"}, []string{"scribe"}, s)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		_, err = Resolve("scribe", l, s)
		wantAll(t, err, "desk", "--route scribe=local@desk?seed=1", "lists no model", "host-decides")
	})

	t.Run("proposed by the tier", func(t *testing.T) {
		f := newFx(t)
		f.machine(`{"scribe":{"tier":"local","settings":{"top_k":40}}}`)
		l := f.load()
		s := &spy{serves: map[Tier]Connection{Local: {Name: "desk"}}}
		_, err := Resolve("scribe", l, s)
		// The allowlist is checked before any settings merge, so the refusal
		// is the allowlist's, not the unaccepted top_k's.
		wantAll(t, err, "desk", "tier local", "lists no model")
		if strings.Contains(err.Error(), "top_k") {
			t.Fatalf("refusal %q judged settings before the allowlist", err)
		}
	})
}

// TestResolveHoldsTheConfiguredAdapterToItsAcceptedSet runs AC 8 through the
// machine's real provider configuration rather than a test double.
//
// Given a provider block read from the machine's configuration,
// when a --route names it with a setting the OpenAI-compatible adapter does
// not accept (model among them, so a setting can never choose a model past
// the allowlist),
// then Resolve refuses naming the setting; with only accepted settings it
// returns the provider leg with the settings as sent.
func TestResolveHoldsTheConfiguredAdapterToItsAcceptedSet(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `}}}`)
	conns := f.loadAPI().Connections()

	resolve := func(route string) (Route, error) {
		t.Helper()
		l := f.load()
		routes, err := ParseRoutes([]string{route}, []string{"scribe"}, conns)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		return Resolve("scribe", l, conns)
	}

	for _, bad := range []string{"top_k", "model"} {
		_, err := resolve("scribe=economy@openrouter?" + bad + "=x")
		wantAll(t, err, bad+" (from --route", "openrouter", "temperature")
	}
	r, err := resolve("scribe=economy@openrouter?temperature=0.2,seed=1")
	if err != nil {
		t.Fatal(err)
	}
	if r.ConnectionUsed != "openrouter" || dump(r.SettingsSent) != `{"seed":1,"temperature":0.2}` {
		t.Fatalf("%+v settings %s", r, dump(r.SettingsSent))
	}
}
