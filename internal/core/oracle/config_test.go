package oracle

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
)

// The provider configuration of itd-2609081951381895: oracle.api.<provider>
// blocks on the machine, the bundled vendor denylist the repository and the
// machine may extend, and the roles and judgement types pointed at
// <provider>/<model>, every one validated when the configuration is read.

func (f *fx) machineConfig(body string) {
	f.put(filepath.Join(f.roots.Home, ".abcd", "config.json"), body)
}

func (f *fx) repoConfig(body string) {
	f.put(filepath.Join(f.roots.Repo, ".abcd", "config.json"), body)
}

func (f *fx) loadAPI() *APIConfig {
	f.t.Helper()
	c, err := LoadAPI(f.roots)
	if err != nil {
		f.t.Fatalf("LoadAPI: %v", err)
	}
	return c
}

func (f *fx) loadAPIErr() error {
	f.t.Helper()
	_, err := LoadAPI(f.roots)
	if err == nil {
		f.t.Fatal("LoadAPI succeeded; want a refusal")
	}
	return err
}

const openrouterBlock = `"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter","models":["typesafe/jev-1.13","typesafe/jev-latest"]}`

// TestUnconfiguredChangesNothing is criterion 1's second half (adr-25's
// default): with no provider block nothing is pointed anywhere, the machine's
// connections serve nothing, and every step stays on the host.
func TestUnconfiguredChangesNothing(t *testing.T) {
	f := newFx(t)
	f.repoConfig(`{"oracle":{"backend":"host-delegated"}}`)
	c := f.loadAPI()
	if len(c.Providers()) != 0 {
		t.Fatalf("providers = %v, want none", c.Providers())
	}
	if _, ok := c.Role("scribe"); ok {
		t.Fatal("a role is pointed at a provider with nothing configured")
	}
	conns := c.Connections()
	if _, ok := conns.Named("openrouter"); ok {
		t.Fatal("a connection is named with nothing configured")
	}
	for _, tier := range Tiers() {
		if _, ok := conns.Serves(tier); ok {
			t.Fatalf("tier %s is served with nothing configured", tier)
		}
	}
	if got := c.Denylist(); len(got) != 1 || got[0].Pattern != "anthropic/*" || got[0].Origin != "bundled" {
		t.Fatalf("denylist = %+v, want the bundled anthropic/*", got)
	}
}

// TestAProviderBlockAndItsRoutesLoad: a block on the machine, a role and a
// judgement type pointed at listed models, and the connection carrying the
// allowlist and the accepted-settings declaration spc-2609251028149555 reads.
func TestAProviderBlockAndItsRoutesLoad(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},
		"roles":{"scribe":"openrouter/typesafe/jev-1.13"},
		"judgements":{"duplicate-match":"openrouter/typesafe/jev-latest"}}}`)
	c := f.loadAPI()
	p, ok := c.Provider("openrouter")
	if !ok || p.BaseURL != "https://openrouter.ai/api/v1" || p.Key != "openrouter" ||
		!reflect.DeepEqual(p.Models, []string{"typesafe/jev-1.13", "typesafe/jev-latest"}) || p.Origin != "~/.abcd/config.json" {
		t.Fatalf("provider = %+v, %v", p, ok)
	}
	tgt, ok := c.Role("scribe")
	if !ok || tgt.Provider != "openrouter" || tgt.Model != "typesafe/jev-1.13" {
		t.Fatalf("role = %+v, %v", tgt, ok)
	}
	if tgt, ok := c.Judgement("duplicate-match"); !ok || tgt.Model != "typesafe/jev-latest" {
		t.Fatalf("judgement = %+v, %v", tgt, ok)
	}
	conn, ok := c.Connections().Named("openrouter")
	if !ok || conn.Name != "openrouter" {
		t.Fatalf("Named = %+v, %v", conn, ok)
	}
	if !reflect.DeepEqual(conn.Models, p.Models) || !conn.Admits("typesafe/jev-1.13") || conn.Admits("typesafe/other") {
		t.Fatalf("connection allowlist = %v", conn.Models)
	}
	if !reflect.DeepEqual(conn.Accepts, openaiapi.AcceptedSettings()) || !conn.Accepted("temperature") || conn.Accepted("model") {
		t.Fatalf("connection accepts = %v", conn.Accepts)
	}
	// A provider claims no tier: it is reached by a route pointed at it.
	if _, ok := c.Connections().Serves(Economy); ok {
		t.Fatal("a provider claimed a tier")
	}
}

// TestAnUnlistedModelIsRefusedWhenTheConfigurationIsRead is criterion 2: a role
// or a judgement type pointed at a model its provider does not list is refused
// before any call, and the refusal names the list.
func TestAnUnlistedModelIsRefusedWhenTheConfigurationIsRead(t *testing.T) {
	for _, route := range []string{
		`"roles":{"scribe":"openrouter/typesafe/jev-2"}`,
		`"judgements":{"duplicate-match":"openrouter/mistral/small"}`,
	} {
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},` + route + `}}`)
		err := f.loadAPIErr()
		for _, want := range []string{"not on openrouter's list", "typesafe/jev-1.13, typesafe/jev-latest", "~/.abcd/config.json"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: refusal %q does not name %q", route, err, want)
			}
		}
	}
	// A route in the repository is held to the machine's list the same way.
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `}}}`)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"openrouter/openai/gpt-5"}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), ".abcd/config.json (repo layer)") || !strings.Contains(err.Error(), "typesafe/jev-1.13") {
		t.Fatalf("repo route refusal = %v", err)
	}
}

// TestTheVendorDenylistWinsOverEveryListing is criterion 3: a listed model the
// denylist matches is refused the same way, and no allowlist entry overrides
// it, however it is spelt.
func TestTheVendorDenylistWinsOverEveryListing(t *testing.T) {
	for _, model := range []string{
		"anthropic/claude-opus-4",
		"Anthropic/Claude-Sonnet",
		"~anthropic/claude-opus-latest",
		"anthropic/claude-3.5-haiku:beta",
	} {
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",
			"models":["typesafe/jev-1.13","` + model + `"]}}}}`)
		err := f.loadAPIErr()
		for _, want := range []string{"anthropic/*", "vendor denylist", "no allowlist entry overrides"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: refusal %q does not name %q", model, err, want)
			}
		}
	}
}

// TestTheDenylistIsExtendedNeverShortened: the repository and the machine add
// entries; neither can remove the bundled one.
func TestTheDenylistIsExtendedNeverShortened(t *testing.T) {
	f := newFx(t)
	f.repoConfig(`{"oracle":{"denylist":["openai/*"]}}`)
	f.machineConfig(`{"oracle":{"denylist":[],"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["openai/gpt-5"]}}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "(openai/*, from .abcd/config.json)") {
		t.Fatalf("refusal = %v, want the repo's openai/* named", err)
	}

	f = newFx(t)
	f.machineConfig(`{"oracle":{"denylist":["google/gemini-3-pro"],"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["typesafe/jev-1.13"]}}}}`)
	c := f.loadAPI()
	got := c.Denylist()
	if len(got) != 2 || got[0].Pattern != "anthropic/*" || got[1].Pattern != "google/gemini-3-pro" || got[1].Origin != "~/.abcd/config.json" {
		t.Fatalf("denylist = %+v", got)
	}
	if err := c.Admit("openrouter", "google/gemini-3-pro:free"); err == nil {
		t.Fatal("an exact denylist entry did not refuse its variant")
	}
	if err := c.Admit("openrouter", "anthropic/claude-opus-4"); err == nil || !strings.Contains(err.Error(), "anthropic/*") {
		t.Fatalf("Admit(anthropic) = %v", err)
	}
	if err := c.Admit("openrouter", "typesafe/jev-1.13"); err != nil {
		t.Fatalf("Admit(listed) = %v", err)
	}
	if err := c.Admit("elsewhere", "typesafe/jev-1.13"); err == nil {
		t.Fatal("Admit on an unconfigured provider passed")
	}
}

// TestAProviderBlockInTheRepositoryIsRefused: a provider block names the
// address a key is sent to, so a checkout may not declare one; a hostile
// repository could otherwise aim the person's key at its own server.
func TestAProviderBlockInTheRepositoryIsRefused(t *testing.T) {
	f := newFx(t)
	f.repoConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://attacker.example/v1","key":"openrouter","models":["typesafe/jev-1.13"]}}}}`)
	err := f.loadAPIErr()
	for _, want := range []string{".abcd/config.json (repo layer)", "oracle.api", "~/.abcd/config.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
}

// TestARouteToAnUnconfiguredProviderStaysOnTheHost: a committed route naming a
// provider this machine has not configured is a diagnostic, not a refusal; the
// step runs on the host, as it would with nothing configured.
func TestARouteToAnUnconfiguredProviderStaysOnTheHost(t *testing.T) {
	f := newFx(t)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`)
	c := f.loadAPI()
	if _, ok := c.Role("scribe"); ok {
		t.Fatal("a route to an unconfigured provider resolved")
	}
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "not configured on this machine") ||
		!strings.Contains(c.Diagnostics[0], "host") {
		t.Fatalf("diagnostics = %v", c.Diagnostics)
	}
}

// TestARoleOutsideTheRosterIsNamedAndSkipped mirrors the routing table's
// orphan rows.
func TestARoleOutsideTheRosterIsNamedAndSkipped(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"no-such-agent":"openrouter/typesafe/jev-1.13"}}}`)
	c := f.loadAPI()
	if len(c.Diagnostics) != 1 || !strings.Contains(c.Diagnostics[0], "no-such-agent") || !strings.Contains(c.Diagnostics[0], "roster") {
		t.Fatalf("diagnostics = %v", c.Diagnostics)
	}
}

// TestAMalformedProviderBlockIsRefused: every field is checked where it is
// read, and a fault is an error naming the file, never a default.
func TestAMalformedProviderBlockIsRefused(t *testing.T) {
	cases := map[string]string{
		"plain http elsewhere":  `"p":{"base_url":"http://api.example.com/v1","models":["m/x"]}`,
		"credentials in url":    `"p":{"base_url":"https://u:secret@api.example.com/v1","models":["m/x"]}`,
		"no models":             `"p":{"base_url":"https://api.example.com/v1","models":[]}`,
		"models absent":         `"p":{"base_url":"https://api.example.com/v1"}`,
		"duplicate model":       `"p":{"base_url":"https://api.example.com/v1","models":["m/x","m/x"]}`,
		"model with a space":    `"p":{"base_url":"https://api.example.com/v1","models":["m x"]}`,
		"key not a plain name":  `"p":{"base_url":"https://api.example.com/v1","key":"../../etc/passwd","models":["m/x"]}`,
		"key empty":             `"p":{"base_url":"https://api.example.com/v1","key":"","models":["m/x"]}`,
		"unknown field":         `"p":{"base_url":"https://api.example.com/v1","models":["m/x"],"api_key":"sk-live"}`,
		"reserved name harness": `"harness":{"base_url":"https://api.example.com/v1","models":["m/x"]}`,
		"name not plain":        `"Open Router":{"base_url":"https://api.example.com/v1","models":["m/x"]}`,
	}
	for name, block := range cases {
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{` + block + `}}}`)
		err := f.loadAPIErr()
		if !strings.Contains(err.Error(), "~/.abcd/config.json") {
			t.Errorf("%s: refusal %q does not name the file", name, err)
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "sk-live") {
			t.Errorf("%s: refusal %q echoes a secret-shaped value", name, err)
		}
	}
}

// TestAMalformedRouteIsRefused: a route is <provider>/<model>, and a judgement
// type is a plain name.
func TestAMalformedRouteIsRefused(t *testing.T) {
	for _, route := range []string{
		`"roles":{"scribe":"jev"}`,
		`"roles":{"scribe":"/typesafe/jev"}`,
		`"roles":{"scribe":"openrouter/"}`,
		`"roles":{"scribe":7}`,
		`"judgements":{"Bad Type":"openrouter/typesafe/jev-1.13"}`,
		`"denylist":["anthropic/"]`,
		`"denylist":["anthropic/* "]`,
		`"denylist":"anthropic/*"`,
	} {
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},` + route + `}}`)
		if _, err := LoadAPI(f.roots); err == nil {
			t.Errorf("%s: LoadAPI succeeded; want a refusal", route)
		}
	}
}
