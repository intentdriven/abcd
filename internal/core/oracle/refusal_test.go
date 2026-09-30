package oracle

import (
	"strings"
	"testing"
)

// The refusals Resolve makes on a provider leg (spc-2609251028149555, AC 8's
// adapter half and AC 11):
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

// wantNone fails unless err is non-nil and names none of parts.
func wantNone(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Resolve returned no error; want a refusal naming none of %q", parts)
	}
	for _, p := range parts {
		if strings.Contains(err.Error(), p) {
			t.Fatalf("refusal %q still names %q", err, p)
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
	roles := map[string]string{"scribe": "example/model-1"}

	t.Run("from the routing row", func(t *testing.T) {
		// Given a repository row setting top_k, which the adapter does not accept
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"economy","settings":{"temperature":0.2,"top_k":40}}}`)
		l := f.load()
		c := Connection{Name: "openrouter", Models: models, Accepts: accepts, Roles: roles}
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
		c := Connection{Name: "openrouter", Models: models, Accepts: accepts, Roles: roles}
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
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Roles: roles, Defaults: Settings{"mirostat": raw("2")}}
		// When the tier itself proposes the provider leg
		s := &spy{serves: map[Tier]Connection{Local: c}}
		_, err := Resolve("scribe", l, s)
		wantAll(t, err, "mirostat", "desk's defaults", "tier local")
	})

	t.Run("a connection no adapter backs accepts no setting", func(t *testing.T) {
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"local","settings":{"seed":1}}}`)
		l := f.load()
		s := &spy{serves: map[Tier]Connection{Local: {Name: "desk", Models: models, Roles: roles}}}
		_, err := Resolve("scribe", l, s)
		wantAll(t, err, "seed", "desk", "accepts no setting")
	})

	t.Run("every accepted setting passes", func(t *testing.T) {
		f := newFx(t)
		f.repo(`{"scribe":{"tier":"local","settings":{"seed":1}}}`)
		l := f.load()
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Roles: roles, Defaults: Settings{"temperature": raw("0")}}
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

// TestResolveRefusesAProviderLegWhoseAllowlistListsNothing is the part of AC
// 11 that needs no model to judge: a provider serves only
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
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`)
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

// TestResolveRefusesARoleModelTheAllowlistDoesNotAdmit is AC 11's model half
// for a role-pointed leg: a provider is reached by a role pointed at
// <provider>/<model> (itd-2609081951381895 Decision 9), so the model that role
// names is the one the leg asks for, and it must be on the allowlist
// (adr-2609221009491186).
//
// Given a proposed route through a provider adapter, the agent's role pointed
// at a model on that connection,
// when a --route names the connection or the tier proposes it,
// then it is one the provider's allowlist admits, or Resolve names the refusal
// (agent, connection, model, allowlist, remedy) instead of returning the leg.
func TestResolveRefusesARoleModelTheAllowlistDoesNotAdmit(t *testing.T) {
	models := []string{"example/model-1", "example/model-2"}
	accepts := []string{"seed"}

	t.Run("named by --route, model unlisted", func(t *testing.T) {
		// Given scribe's role pointed at a model the connection does not list
		f := newFx(t)
		l := f.load()
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Roles: map[string]string{"scribe": "example/model-9"}}
		s := &spy{named: map[string]Connection{"desk": c}}
		routes, err := ParseRoutes([]string{"scribe=local@desk?top_k=1"}, []string{"scribe"}, s)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		// When the step resolves
		_, err = Resolve("scribe", l, s)
		// Then it is refused naming the agent, the connection, the model, the
		// allowlist and the remedy, before any settings are judged
		wantAll(t, err, "scribe", "desk", "--route scribe=local@desk?top_k=1", "example/model-9",
			"example/model-1, example/model-2", "oracle.roles.scribe", "host-decides")
		if strings.Contains(err.Error(), "does not accept") {
			t.Fatalf("refusal %q judged settings before the allowlist", err)
		}
	})

	t.Run("proposed by the tier, model unlisted", func(t *testing.T) {
		f := newFx(t)
		f.machine(`{"scribe":{"tier":"local"}}`)
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Roles: map[string]string{"scribe": "example/model-9"}}
		_, err := Resolve("scribe", f.load(), &spy{serves: map[Tier]Connection{Local: c}})
		wantAll(t, err, "scribe", "desk", "tier local", "example/model-9", "example/model-1, example/model-2")
	})

	t.Run("model listed", func(t *testing.T) {
		// Given the role pointed at a model the connection lists, the leg is returned
		f := newFx(t)
		f.machine(`{"scribe":{"tier":"local"}}`)
		c := Connection{Name: "desk", Models: models, Accepts: accepts, Roles: map[string]string{"scribe": "example/model-2"}}
		r, err := Resolve("scribe", f.load(), &spy{serves: map[Tier]Connection{Local: c}})
		if err != nil {
			t.Fatal(err)
		}
		if r.ConnectionUsed != "desk" {
			t.Fatalf("%+v", r)
		}
	})
}

// TestResolveRefusesALegTheAgentsRoleDoesNotPointAt is AC 11's unpointed
// route: a route to a provider connection for an agent whose
// oracle.roles.<agent> does not point at that connection names no model, so
// Resolve refuses it rather than choose one, naming the role setting to add
// (the product thinker's ruling BR1 of 2026-09-29). The refusal is the decided
// behaviour, so it never tells the person a ruling is still to come.
//
// Given a connection that lists models and an agent whose role points
// elsewhere or nowhere,
// when a --route or the tier sends the agent there,
// then Resolve refuses, naming the agent, the connection, the reason and the
// remedy.
func TestResolveRefusesALegTheAgentsRoleDoesNotPointAt(t *testing.T) {
	models := []string{"example/model-1"}

	t.Run("no role at all, named by --route", func(t *testing.T) {
		f := newFx(t)
		l := f.load()
		c := Connection{Name: "desk", Models: models, Accepts: []string{"seed"}}
		s := &spy{named: map[string]Connection{"desk": c}}
		routes, err := ParseRoutes([]string{"scribe=economy@desk"}, []string{"scribe"}, s)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Apply(routes); err != nil {
			t.Fatal(err)
		}
		_, err = Resolve("scribe", l, s)
		wantAll(t, err, "scribe", "desk", "--route scribe=economy@desk", "oracle.roles.scribe",
			"a route that names no model is refused", "point oracle.roles.scribe at desk/<model>", "example/model-1", "host-decides")
		wantNone(t, err, "not yet decided", "undecided")
	})

	t.Run("another agent's role points here, proposed by the tier", func(t *testing.T) {
		f := newFx(t)
		f.machine(`{"scribe":{"tier":"local"}}`)
		c := Connection{Name: "desk", Models: models, Roles: map[string]string{"intent-auditor": "example/model-1"}}
		_, err := Resolve("scribe", f.load(), &spy{serves: map[Tier]Connection{Local: c}})
		wantAll(t, err, "scribe", "desk", "tier local", "oracle.roles.scribe", "a route that names no model is refused")
		wantNone(t, err, "not yet decided", "undecided")
	})

	t.Run("through the machine's configuration", func(t *testing.T) {
		// Given a provider block and scribe's role pointed at another provider
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `,` +
			`"desk":{"base_url":"http://127.0.0.1:11434/v1","models":["example/model-1"]}},` +
			`"roles":{"scribe":"desk/example/model-1","intent-auditor":"openrouter/typesafe/jev-1.13"}}}`)
		conns := f.loadAPI().Connections()
		resolve := func(agent, route string) error {
			t.Helper()
			l := f.load()
			routes, err := ParseRoutes([]string{route}, []string{agent}, conns)
			if err != nil {
				t.Fatal(err)
			}
			if err := l.Apply(routes); err != nil {
				t.Fatal(err)
			}
			_, err = Resolve(agent, l, conns)
			return err
		}
		// When --route sends scribe to openrouter, which its role does not point at
		// Then it is refused, naming the role setting to add
		err := resolve("scribe", "scribe=economy@openrouter")
		wantAll(t, err, "scribe", "openrouter", "a route that names no model is refused", "point oracle.roles.scribe at openrouter/<model>")
		wantNone(t, err, "not yet decided", "undecided")
		// And the agents whose roles point at the named connection resolve
		if err := resolve("scribe", "scribe=local@desk"); err != nil {
			t.Fatal(err)
		}
		if err := resolve("intent-auditor", "intent-auditor=economy@openrouter"); err != nil {
			t.Fatal(err)
		}
	})
}
