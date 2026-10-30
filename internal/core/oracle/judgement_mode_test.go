package oracle

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
	// The refusal quotes the value as the file spells it, and names every
	// form a judgement type's route takes.
	err := f.loadAPIErr()
	for _, want := range []string{`"shadow:openrouter"`, "host, shadow:<provider>/<model> or <provider>/<model>"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
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

// A judgement type is turned over to an adapter only from the person's own
// configuration: the machine's file, or their own flag for one run, the same
// rule runner.personal applies to a role handed to a runner. A route the
// repository's file turns on is skipped with one diagnostic, and the
// machine's route to the type applies in its place, its mode with it, or the
// host when the machine sets none. The repository may still keep a type on
// the host, or run the adapter beside the host in shadow.
func TestOnlyThePersonsOwnConfigurationTurnsAJudgementTypeOn(t *testing.T) {
	const repoOn = `{"oracle":{"judgements":{"eligibility-residual":"local/qwen/qwen3-8b"}}}`
	for name, tc := range map[string]struct {
		machine  string
		wantOK   bool
		wantMode string
	}{
		"no machine route, so the host": {machine: ``},
		"the machine's shadow stays shadow": {machine: `,"judgements":{"eligibility-residual":"shadow:local/qwen/qwen3-8b"}`,
			wantOK: true, wantMode: JudgementShadow},
		"the machine's own on applies": {machine: `,"judgements":{"eligibility-residual":"local/qwen/qwen3-8b"}`,
			wantOK: true, wantMode: JudgementOn},
		"the machine's host stays host": {machine: `,"judgements":{"eligibility-residual":"host"}`},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + localBlock + `}` + tc.machine + `}}`)
			f.repoConfig(repoOn)
			c := f.loadAPI()
			tgt, ok := c.Judgement("eligibility-residual")
			if ok != tc.wantOK {
				t.Fatalf("judgement = %+v, %v; want routed %v", tgt, ok, tc.wantOK)
			}
			if ok && (tgt.Mode != tc.wantMode || tgt.Origin != abcdhome.Display("config.json")) {
				t.Fatalf("judgement = %+v; want the machine's route in mode %q", tgt, tc.wantMode)
			}
			if len(c.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %q, want exactly one naming the skipped route", c.Diagnostics)
			}
			for _, want := range []string{".abcd/config.json (repo layer)", "oracle.judgements.eligibility-residual",
				"person's own configuration", "skipped", "shadow:local/qwen/qwen3-8b"} {
				if !strings.Contains(c.Diagnostics[0], want) {
					t.Errorf("diagnostic %q does not name %q", c.Diagnostics[0], want)
				}
			}
		})
	}
}

// The repository's own shadow and host stand: neither hands the decision over.
func TestARepositoryMayKeepAJudgementTypeInShadow(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + localBlock + `}}}`)
	f.repoConfig(`{"oracle":{"judgements":{"eligibility-residual":"shadow:local/qwen/qwen3-8b"}}}`)
	c := f.loadAPI()
	tgt, ok := c.Judgement("eligibility-residual")
	if !ok || tgt.Mode != JudgementShadow || tgt.Origin != ".abcd/config.json" {
		t.Fatalf("judgement = %+v, %v; want the repository's shadow route", tgt, ok)
	}
	if len(c.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %q, want none", c.Diagnostics)
	}
}
