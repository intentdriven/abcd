package oracle

import (
	"strings"
	"testing"
)

// A judgement type's route carries a mode (itd-2609221009495079,
// spc-2609221011151661 scope 3): `host` keeps the type on the host,
// `shadow:<provider>/<model>` runs the adapter beside the host, which still
// decides, and a bare `<provider>/<model>` turns the type over to the adapter.
// The allowlist is judged for a shadow route exactly as for one turned on, when
// the configuration is read and before any call (criterion 4).

func TestAJudgementRouteCarriesItsMode(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},
		"judgements":{"duplicate-match":"shadow:openrouter/typesafe/jev-1.13",
			"remedy-read":"openrouter/typesafe/jev-latest",
			"memory-check":"host"}}}`)
	c := f.loadAPI()
	if len(c.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %q, want none", c.Diagnostics)
	}
	tgt, ok := c.Judgement("duplicate-match")
	if !ok || tgt.Provider != "openrouter" || tgt.Model != "typesafe/jev-1.13" || tgt.Mode != JudgementShadow {
		t.Fatalf("shadow judgement = %+v, %v", tgt, ok)
	}
	if got := tgt.Route(); got != "shadow:openrouter/typesafe/jev-1.13" {
		t.Fatalf("Route() = %q", got)
	}
	tgt, ok = c.Judgement("remedy-read")
	if !ok || tgt.Model != "typesafe/jev-latest" || tgt.Mode != JudgementOn {
		t.Fatalf("judgement turned on = %+v, %v", tgt, ok)
	}
	if got := tgt.Route(); got != "openrouter/typesafe/jev-latest" {
		t.Fatalf("Route() = %q", got)
	}
	if tgt, ok := c.Judgement("memory-check"); ok {
		t.Fatalf("a host judgement is pointed at %+v; host means no route", tgt)
	}
	// A role carries no mode: the shadow prefix is the judgements' alone.
	for _, r := range c.Routes() {
		if r.Kind == "role" && r.Target.Mode != "" {
			t.Fatalf("role %s carries mode %q", r.Name, r.Target.Mode)
		}
	}
}

func TestAShadowRouteToAnUnlistedModelIsRefusedBeforeAnyCall(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},
		"judgements":{"duplicate-match":"shadow:openrouter/mistral/small"}}}`)
	err := f.loadAPIErr()
	for _, want := range []string{"not on openrouter's list", "typesafe/jev-1.13, typesafe/jev-latest", "refused before any call"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
}

func TestADenylistedShadowRouteIsRefused(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"denylist":["typesafe/*"],
		"judgements":{"duplicate-match":"shadow:openrouter/typesafe/jev-1.13"}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "typesafe/*") {
		t.Fatalf("refusal = %v, want the denylist entry named", err)
	}
}

func TestARepositoryShadowRouteToAKeyedProviderIsSkipped(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `}}}`)
	f.repoConfig(`{"oracle":{"judgements":{"duplicate-match":"shadow:openrouter/typesafe/jev-1.13"}}}`)
	c := f.loadAPI()
	if tgt, ok := c.Judgement("duplicate-match"); ok {
		t.Fatalf("a repository's shadow route spends the person's key: %+v", tgt)
	}
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "holds a key") {
		t.Fatalf("diagnostics = %q, want the keyed-route skip", c.Diagnostics)
	}
}

func TestARepositoryHostJudgementOverridesTheMachineRoute(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + localBlock + `},"judgements":{"duplicate-match":"local/qwen/qwen3-8b"}}}`)
	f.repoConfig(`{"oracle":{"judgements":{"duplicate-match":"host"}}}`)
	c := f.loadAPI()
	if tgt, ok := c.Judgement("duplicate-match"); ok {
		t.Fatalf("judgement = %+v; the repository's host keeps it on the host", tgt)
	}
	if len(c.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %q, want none: host is not a malformed route", c.Diagnostics)
	}
}

func TestAMalformedShadowRouteIsRefusedOnTheMachine(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"judgements":{"duplicate-match":"shadow:openrouter"}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "<provider>/<model>") {
		t.Fatalf("refusal = %v", err)
	}
	// A role takes no mode: shadow: on a role names a provider called
	// "shadow:openrouter", which is not a plain name and not configured.
	f = newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"scribe":"shadow:openrouter/typesafe/jev-1.13"}}}`)
	c := f.loadAPI()
	if tgt, ok := c.Role("scribe"); ok {
		t.Fatalf("a role took the shadow mode: %+v", tgt)
	}
}
