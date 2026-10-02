package oracle

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
)

// The provider configuration of itd-2609081951381895: oracle.api.<provider>
// blocks on the machine, whose allowlists alone decide which models a provider
// serves (adr-2609300107513982), the optional denylist the repository and the
// machine may write, and the roles and judgement types pointed at
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
	if got := c.Denylist(); len(got) != 0 {
		t.Fatalf("denylist = %+v, want none: abcd bundles no vendor denylist", got)
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
	// The machine's own route to a keyless provider is refused the same way: the
	// person's file is never dropped silently. A repository's route to an
	// unlisted model is skipped instead
	// (TestARepositoryRouteToAnUnlistedModelIsSkipped).
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + localBlock + `},"roles":{"scribe":"local/openai/gpt-5"}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "~/.abcd/config.json (machine layer)") || !strings.Contains(err.Error(), "qwen/qwen3-8b") {
		t.Fatalf("machine route refusal = %v", err)
	}
}

// TestARepositoryRouteToAnUnlistedModelIsSkipped is the technical
// facilitator's ruling CD3 of 2026-10-02: a repository's route naming a model
// that a configured provider holding no key does not list is skipped with one
// diagnostic naming the repository's file, the route and the list, rather
// than refusing the whole configuration. The machine's own route to the name,
// if it sets one, applies in its place; every other route still loads.
func TestARepositoryRouteToAnUnlistedModelIsSkipped(t *testing.T) {
	for name, tc := range map[string]struct {
		repo, machine, setting string
		family, route          string
		machineModel           string
	}{
		"role": {repo: `"roles":{"scribe":"local/openai/gpt-5"}`, setting: "oracle.roles.scribe",
			family: rolesKey, route: "scribe"},
		"judgement type": {repo: `"judgements":{"duplicate-match":"local/openai/gpt-5"}`,
			setting: "oracle.judgements.duplicate-match", family: judgementsKey, route: "duplicate-match"},
		"role over a machine route": {repo: `"roles":{"scribe":"local/openai/gpt-5"}`,
			machine: `,"roles":{"scribe":"local/qwen/qwen3-8b"}`, setting: "oracle.roles.scribe",
			family: rolesKey, route: "scribe", machineModel: "qwen/qwen3-8b"},
		"judgement type over a machine route": {repo: `"judgements":{"duplicate-match":"local/openai/gpt-5"}`,
			machine: `,"judgements":{"duplicate-match":"local/qwen/qwen3-8b"}`, setting: "oracle.judgements.duplicate-match",
			family: judgementsKey, route: "duplicate-match", machineModel: "qwen/qwen3-8b"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + localBlock + `}` + tc.machine + `}}`)
			other := `"roles":{"scribe":"local/qwen/qwen3-8b"}`
			if tc.family == rolesKey {
				other = `"judgements":{"duplicate-match":"local/qwen/qwen3-8b"}`
			}
			f.repoConfig(`{"oracle":{` + tc.repo + `,` + other + `}}`)
			c, err := LoadAPI(f.roots)
			if err != nil {
				t.Fatalf("LoadAPI refused the whole configuration over one repository route: %v", err)
			}
			if len(c.Diagnostics) != 1 {
				t.Fatalf("diagnostics %q, want exactly one naming the skipped route", c.Diagnostics)
			}
			for _, want := range []string{".abcd/config.json (repo layer)", tc.setting, `"local/openai/gpt-5"`,
				"not on local's list (qwen/qwen3-8b)", "skipped"} {
				if !strings.Contains(c.Diagnostics[0], want) {
					t.Errorf("diagnostic %q does not name %q", c.Diagnostics[0], want)
				}
			}
			var got Target
			var ok bool
			if tc.family == rolesKey {
				got, ok = c.Role(tc.route)
				if tgt, on := c.Judgement("duplicate-match"); !on || tgt.Provider != "local" {
					t.Errorf("the other route = %+v, %v; want it loaded", tgt, on)
				}
			} else {
				got, ok = c.Judgement(tc.route)
				if tgt, on := c.Role("scribe"); !on || tgt.Provider != "local" {
					t.Errorf("the other route = %+v, %v; want it loaded", tgt, on)
				}
			}
			switch {
			case tc.machineModel != "":
				if !ok || got.Model != tc.machineModel || got.Origin != "~/.abcd/config.json" {
					t.Errorf("%s = %+v, %v; want the machine's own route", tc.route, got, ok)
				}
			case ok:
				t.Errorf("%s = %+v; a repository route to an unlisted model must never load", tc.route, got)
			}
		})
	}
}

// TestASkippedUnlistedRepositoryRouteLeavesTheMachineRouteJudged: skipping a
// repository's route under ruling CD3 hands the name to the machine's own
// route, which is judged exactly as it would be alone: a machine route to an
// unlisted model still refuses, naming the machine's file.
func TestASkippedUnlistedRepositoryRouteLeavesTheMachineRouteJudged(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + localBlock + `},"roles":{"scribe":"local/mistral/small"}}}`)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"local/openai/gpt-5"}}}`)
	err := f.loadAPIErr()
	for _, want := range []string{"~/.abcd/config.json (machine layer)", "oracle.roles.scribe", "local/mistral/small", "not on local's list"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
}

// TestADenylistedKeylessRepositoryRouteIsStillRefused: ruling CD3 skips a
// repository's route to a model a keyless provider does not list, but never
// one the denylist matches: no layer softens the denylist, whichever layer
// wrote the entry.
func TestADenylistedKeylessRepositoryRouteIsStillRefused(t *testing.T) {
	for label, tc := range map[string]struct{ machine, repo string }{
		"the machine's entry":    {machine: `"denylist":["openai/*"],`, repo: `{"oracle":{"roles":{"scribe":"local/openai/gpt-5"}}}`},
		"the repository's entry": {repo: `{"oracle":{"denylist":["openai/gpt-5"],"roles":{"scribe":"local/openai/gpt-5"}}}`},
	} {
		t.Run(label, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{` + tc.machine + `"api":{` + localBlock + `},"roles":{"scribe":"local/qwen/qwen3-8b"}}}`)
			f.repoConfig(tc.repo)
			err := f.loadAPIErr()
			for _, want := range []string{"oracle.denylist", "oracle.roles.scribe", ".abcd/config.json (repo layer)"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal %q does not name %q", err, want)
				}
			}
		})
	}
}

// localBlock is a provider that holds no key: a server on this machine, its
// block naming no credential.
const localBlock = `"local":{"base_url":"http://localhost:11434/v1","models":["qwen/qwen3-8b"]}`

// TestARepositoryRouteToAKeyedProviderIsSkipped is the product thinker's
// ruling AA(b) of 2026-09-29 as ruling CD2 of the same day shapes it: only a
// route the person set up on their own machine may spend their paid key, so a
// role or a judgement type the repository's configuration points at a provider
// that holds a key never reaches it. The route is skipped, with one diagnostic
// naming the route, the provider and the machine's file as where the route is
// set, and the rest of the configuration loads: every other route keeps
// working, and a machine route to the same name is the one that applies. A
// provider holds a key when its block names one; the credential store is never
// consulted, so no secret is read to decide it.
func TestARepositoryRouteToAKeyedProviderIsSkipped(t *testing.T) {
	for name, tc := range map[string]struct {
		repo, machine, setting string
		family, route          string
		machineModel           string
	}{
		"role": {repo: `"roles":{"scribe":"openrouter/typesafe/jev-1.13"}`, setting: "oracle.roles.scribe",
			family: rolesKey, route: "scribe"},
		"judgement type": {repo: `"judgements":{"duplicate-match":"openrouter/typesafe/jev-latest"}`,
			setting: "oracle.judgements.duplicate-match", family: judgementsKey, route: "duplicate-match"},
		"role over a machine route": {repo: `"roles":{"scribe":"openrouter/typesafe/jev-latest"}`,
			machine: `,"roles":{"scribe":"openrouter/typesafe/jev-1.13"}`, setting: "oracle.roles.scribe",
			family: rolesKey, route: "scribe", machineModel: "typesafe/jev-1.13"},
		"unlisted model": {repo: `"roles":{"scribe":"openrouter/openai/gpt-5"}`, setting: "oracle.roles.scribe",
			family: rolesKey, route: "scribe"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `,` + localBlock + `}` + tc.machine + `}}`)
			// A keyless route beside the refused one: the skip costs nothing else.
			other := `"roles":{"scribe":"local/qwen/qwen3-8b"}`
			if tc.family == rolesKey {
				other = `"judgements":{"duplicate-match":"local/qwen/qwen3-8b"}`
			}
			f.repoConfig(`{"oracle":{` + tc.repo + `,` + other + `}}`)
			c, err := LoadAPI(f.roots)
			if err != nil {
				t.Fatalf("LoadAPI refused the whole configuration over one repository route: %v", err)
			}
			var hits []string
			for _, d := range c.Diagnostics {
				if strings.Contains(d, "holds a key") {
					hits = append(hits, d)
				}
			}
			if len(hits) != 1 {
				t.Fatalf("diagnostics %q, want exactly one naming the skipped keyed route", c.Diagnostics)
			}
			for _, want := range []string{".abcd/config.json (repo layer)", tc.setting, "openrouter", "holds a key", "skipped",
				"set " + tc.setting + " in ~/.abcd/config.json and remove it from .abcd/config.json,"} {
				if !strings.Contains(hits[0], want) {
					t.Errorf("diagnostic %q does not name %q", hits[0], want)
				}
			}
			var got Target
			var ok bool
			if tc.family == rolesKey {
				got, ok = c.Role(tc.route)
				if tgt, on := c.Judgement("duplicate-match"); !on || tgt.Provider != "local" {
					t.Errorf("the other route = %+v, %v; want it loaded", tgt, on)
				}
			} else {
				got, ok = c.Judgement(tc.route)
				if tgt, on := c.Role("scribe"); !on || tgt.Provider != "local" {
					t.Errorf("the other route = %+v, %v; want it loaded", tgt, on)
				}
			}
			switch {
			case tc.machineModel != "":
				if !ok || got.Model != tc.machineModel || got.Origin != "~/.abcd/config.json" {
					t.Errorf("%s = %+v, %v; want the machine's own route", tc.route, got, ok)
				}
			case ok:
				t.Errorf("%s = %+v; a repository route to a keyed provider must never load", tc.route, got)
			}
		})
	}
}

// TestADenylistedKeyedRepositoryRouteIsStillRefused: skipping a repository's
// route to a keyed provider never softens the denylist. A route the denylist
// matches refuses the configuration from any layer, keyed provider or not.
// No denylist is bundled (adr-2609300107513982), so the entry is the one the
// person's configuration writes.
func TestADenylistedKeyedRepositoryRouteIsStillRefused(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"denylist":["anthropic/*"],"api":{` + openrouterBlock + `}}}`)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"openrouter/anthropic/claude-opus-4"}}}`)
	err := f.loadAPIErr()
	for _, want := range []string{"anthropic/*", "oracle.denylist", "oracle.roles.scribe"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
}

// TestARepositoryRouteWithAMalformedNameIsSkipped is ruling CD2's "other
// commands keep working" for a route's NAME: a repository route whose name is
// not a plain lower-case name (the review probe's Cyrillic U+0456 in "scribe",
// a judgement type with a space, a name carrying a bidi override and an escape)
// is skipped with one diagnostic naming the file and the rejected name, and
// the rest of the configuration loads. The name is repository-authored bytes,
// so the diagnostic carries no terminal-attack rune and spells a lookalike
// letter as an escape the reader can see.
func TestARepositoryRouteWithAMalformedNameIsSkipped(t *testing.T) {
	for label, tc := range map[string]struct {
		route, setting, shown string
	}{
		"cyrillic lookalike": {route: `"roles":{"scr\u0456be":"local/qwen/qwen3-8b"}`,
			setting: "oracle.roles", shown: `"scr\u0456be"`},
		"judgement with a space": {route: `"judgements":{"Bad Type":"local/qwen/qwen3-8b"}`,
			setting: "oracle.judgements", shown: `"Bad Type"`},
		"bidi and escape": {route: `"roles":{"scribe\u202e\u001b[2J":"local/qwen/qwen3-8b"}`,
			setting: "oracle.roles", shown: `"scribe??[2J"`},
	} {
		t.Run(label, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `,` + localBlock + `}}}`)
			f.repoConfig(`{"oracle":{` + tc.route + `,"denylist":["openai/*"]}}`)
			c, err := LoadAPI(f.roots)
			if err != nil {
				t.Fatalf("LoadAPI refused the whole configuration over one repository route's name: %v", err)
			}
			if len(c.Diagnostics) != 1 {
				t.Fatalf("diagnostics %q, want exactly one naming the skipped route", c.Diagnostics)
			}
			d := c.Diagnostics[0]
			for _, want := range []string{".abcd/config.json (repo layer)", tc.setting, tc.shown, "skipped"} {
				if !strings.Contains(d, want) {
					t.Errorf("diagnostic %q does not name %q", d, want)
				}
			}
			for _, r := range d {
				if r < 0x20 || r == 0x7f || r > 0x7e {
					t.Errorf("diagnostic %q carries the rune %U; a repository name reaches the terminal sanitised", d, r)
				}
			}
			if dl := c.Denylist(); len(dl) != 1 || dl[0].Pattern != "openai/*" {
				t.Errorf("denylist = %+v; the rest of the repository's configuration must still load", dl)
			}
			if len(c.Providers()) != 2 {
				t.Errorf("providers = %+v; the machine's blocks must still load", c.Providers())
			}
		})
	}
}

// TestARepositoryRouteWithAMalformedValueIsSkipped: a repository route that is
// not <provider>/<model> names no provider, so no trust question arises, and
// under ruling CD2 it is skipped with one diagnostic rather than taking every
// command that reads the configuration down; the machine's own route to the
// name applies in its place. The machine's malformed route is still refused
// (TestAMalformedRouteIsRefused).
func TestARepositoryRouteWithAMalformedValueIsSkipped(t *testing.T) {
	for route, shown := range map[string]string{
		`"jev"`:           `"jev"`,
		`"/typesafe/jev"`: `"/typesafe/jev"`,
		`"openrouter/"`:   `"openrouter/"`,
		`"lo\u202ecal"`:   `"lo?cal"`,
		`7`:               "not a string",
	} {
		t.Run(route, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + localBlock + `},"roles":{"scribe":"local/qwen/qwen3-8b"}}}`)
			f.repoConfig(`{"oracle":{"roles":{"scribe":` + route + `},"denylist":["openai/*"]}}`)
			c, err := LoadAPI(f.roots)
			if err != nil {
				t.Fatalf("LoadAPI refused the whole configuration over one repository route's value: %v", err)
			}
			if len(c.Diagnostics) != 1 {
				t.Fatalf("diagnostics %q, want exactly one naming the skipped route", c.Diagnostics)
			}
			d := c.Diagnostics[0]
			for _, want := range []string{".abcd/config.json (repo layer)", "oracle.roles.scribe", shown, "skipped"} {
				if !strings.Contains(d, want) {
					t.Errorf("diagnostic %q does not name %q", d, want)
				}
			}
			for _, r := range d {
				if r < 0x20 || r == 0x7f || r > 0x7e {
					t.Errorf("diagnostic %q carries the rune %U; a repository value reaches the terminal sanitised", d, r)
				}
			}
			if tgt, ok := c.Role("scribe"); !ok || tgt.Origin != "~/.abcd/config.json" {
				t.Errorf("scribe = %+v, %v; want the machine's own route in its place", tgt, ok)
			}
		})
	}
}

// TestAMachineRouteWithAMalformedNameIsRefused: the machine's file is the
// person's own, so a malformed route name there refuses the configuration
// naming that file, as every machine fault does; the skip is for a repository.
// A name both layers spell alike is the machine's too.
func TestAMachineRouteWithAMalformedNameIsRefused(t *testing.T) {
	for label, repo := range map[string]string{
		"machine only": `{"oracle":{"denylist":["openai/*"]}}`,
		"both layers":  `{"oracle":{"roles":{"scr\u0456be":"local/qwen/qwen3-8b"}}}`,
	} {
		t.Run(label, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + localBlock + `},"roles":{"scr\u0456be":"local/qwen/qwen3-8b"}}}`)
			f.repoConfig(repo)
			err := f.loadAPIErr()
			for _, want := range []string{"~/.abcd/config.json (machine layer)", "oracle.roles", "not a plain lower-case name"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal %q does not name %q", err, want)
				}
			}
		})
	}
}

// TestAMachineRouteToAKeyedProviderIsAdmitted: the machine's own route to a
// provider that holds a key is the person's, and loads as it always did.
func TestAMachineRouteToAKeyedProviderIsAdmitted(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"scribe":"openrouter/typesafe/jev-1.13"},
		"judgements":{"duplicate-match":"openrouter/typesafe/jev-latest"}}}`)
	f.repoConfig(`{"oracle":{"denylist":["openai/*"]}}`)
	c := f.loadAPI()
	if tgt, ok := c.Role("scribe"); !ok || tgt.Provider != "openrouter" || tgt.Origin != "~/.abcd/config.json" {
		t.Fatalf("role = %+v, %v", tgt, ok)
	}
	if tgt, ok := c.Judgement("duplicate-match"); !ok || tgt.Origin != "~/.abcd/config.json" {
		t.Fatalf("judgement = %+v, %v", tgt, ok)
	}
}

// TestARepositoryRouteToAKeylessProviderIsAdmitted: a provider whose block
// names no key spends no key, so the repository may point at it, and its route
// wins over the machine's as every repository route does.
func TestARepositoryRouteToAKeylessProviderIsAdmitted(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `,` + localBlock + `},"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`)
	f.repoConfig(`{"oracle":{"roles":{"scribe":"local/qwen/qwen3-8b"},"judgements":{"duplicate-match":"local/qwen/qwen3-8b"}}}`)
	c := f.loadAPI()
	if tgt, ok := c.Role("scribe"); !ok || tgt.Provider != "local" || tgt.Origin != ".abcd/config.json" {
		t.Fatalf("role = %+v, %v", tgt, ok)
	}
	if tgt, ok := c.Judgement("duplicate-match"); !ok || tgt.Provider != "local" {
		t.Fatalf("judgement = %+v, %v", tgt, ok)
	}
}

// TestATypedRouteToAKeyedProviderIsAdmitted: a --route the person types names a
// keyed connection and resolves to it, the model coming from the machine's
// role; the repository's refusal does not reach a flag.
func TestATypedRouteToAKeyedProviderIsAdmitted(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`)
	conns := f.loadAPI().Connections()
	l := f.load()
	routes, err := ParseRoutes([]string{"scribe=economy@openrouter"}, []string{"scribe"}, conns)
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
	if r.ConnectionUsed != "openrouter" || r.Override != "scribe=economy@openrouter" {
		t.Fatalf("route = %+v", r)
	}
}

// TestTheAllowlistAloneDecides is criterion 3 as ruling H9 of 2026-09-29
// left it (adr-2609300107513982): no vendor denylist is bundled, so a model
// the provider lists is served whichever vendor made it, however it is spelt,
// and a model it does not list is refused naming the list.
func TestTheAllowlistAloneDecides(t *testing.T) {
	for _, model := range []string{
		"anthropic/claude-opus-4",
		"Anthropic/Claude-Sonnet",
		"~anthropic/claude-opus-latest",
		"anthropic/claude-3.5-haiku:beta",
	} {
		f := newFx(t)
		f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",
			"models":["typesafe/jev-1.13","` + model + `"]}},"roles":{"scribe":"openrouter/` + model + `"}}}`)
		c, err := LoadAPI(f.roots)
		if err != nil {
			t.Fatalf("%s: a listed model is refused: %v", model, err)
		}
		if tgt, ok := c.Role("scribe"); !ok || tgt.Model != model {
			t.Fatalf("%s: role = %+v, %v; want the listed model", model, tgt, ok)
		}
		if err := c.Admit("openrouter", model); err != nil {
			t.Fatalf("%s: Admit = %v", model, err)
		}
		if err := c.Admit("openrouter", "anthropic/claude-sonnet-4"); err == nil ||
			!strings.Contains(err.Error(), "not on openrouter's list (typesafe/jev-1.13, "+model+")") {
			t.Fatalf("%s: Admit(unlisted) = %v, want the list named", model, err)
		}
	}
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"roles":{"scribe":"openrouter/anthropic/claude-opus-4"}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "not on openrouter's list (typesafe/jev-1.13, typesafe/jev-latest)") {
		t.Fatalf("refusal = %v, want the unlisted model refused naming the list", err)
	}
}

// TestADenylistTheConfigurationWritesStillRefuses: oracle.denylist is the
// person's and the repository's own, optional. Every layer's entries apply, a
// machine's empty list removes none of the repository's, a listed model an
// entry matches is refused naming the entry, and an exact entry refuses its
// variant.
func TestADenylistTheConfigurationWritesStillRefuses(t *testing.T) {
	f := newFx(t)
	f.repoConfig(`{"oracle":{"denylist":["openai/*"]}}`)
	f.machineConfig(`{"oracle":{"denylist":[],"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["openai/gpt-5"]}}}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "(openai/*, from .abcd/config.json)") ||
		!strings.Contains(err.Error(), "oracle.denylist") {
		t.Fatalf("refusal = %v, want the repo's openai/* named", err)
	}

	f = newFx(t)
	f.machineConfig(`{"oracle":{"denylist":["google/gemini-3-pro"],"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["typesafe/jev-1.13"]}}}}`)
	c := f.loadAPI()
	got := c.Denylist()
	if len(got) != 1 || got[0].Pattern != "google/gemini-3-pro" || got[0].Origin != "~/.abcd/config.json" {
		t.Fatalf("denylist = %+v", got)
	}
	if err := c.Admit("openrouter", "google/gemini-3-pro:free"); err == nil {
		t.Fatal("an exact denylist entry did not refuse its variant")
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

// TestARepositoryRouteToAnUnconfiguredProviderYieldsToTheMachineRoute is the
// technical facilitator's ruling CD4 of 2026-10-02: a repository's route
// naming a provider this machine has not configured never displaces the
// machine's own route to the same name. The repository's route is skipped
// with one diagnostic naming it, and the owner's setting applies. With no
// machine route to the name, the step stays on the host
// (TestARouteToAnUnconfiguredProviderStaysOnTheHost).
func TestARepositoryRouteToAnUnconfiguredProviderYieldsToTheMachineRoute(t *testing.T) {
	for name, tc := range map[string]struct {
		family, route, setting string
	}{
		"role":           {family: rolesKey, route: "scribe", setting: "oracle.roles.scribe"},
		"judgement type": {family: judgementsKey, route: "duplicate-match", setting: "oracle.judgements.duplicate-match"},
	} {
		t.Run(name, func(t *testing.T) {
			fam := strings.TrimPrefix(tc.family, "oracle.")
			f := newFx(t)
			f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"` + fam + `":{"` + tc.route + `":"openrouter/typesafe/jev-1.13"}}}`)
			f.repoConfig(`{"oracle":{"` + fam + `":{"` + tc.route + `":"elsewhere/typesafe/jev-1.13"}}}`)
			c := f.loadAPI()
			var got Target
			var ok bool
			if tc.family == rolesKey {
				got, ok = c.Role(tc.route)
			} else {
				got, ok = c.Judgement(tc.route)
			}
			if !ok || got.Provider != "openrouter" || got.Origin != "~/.abcd/config.json" {
				t.Fatalf("%s = %+v, %v; want the machine's own route", tc.route, got, ok)
			}
			if len(c.Diagnostics) != 1 {
				t.Fatalf("diagnostics %q, want exactly one naming the skipped route", c.Diagnostics)
			}
			for _, want := range []string{".abcd/config.json (repo layer)", tc.setting, `"elsewhere"`,
				"not configured on this machine", "skipped", "~/.abcd/config.json"} {
				if !strings.Contains(c.Diagnostics[0], want) {
					t.Errorf("diagnostic %q does not name %q", c.Diagnostics[0], want)
				}
			}
			if strings.Contains(c.Diagnostics[0], "runs on the host") {
				t.Errorf("diagnostic %q says the step runs on the host; the machine's route applies", c.Diagnostics[0])
			}
		})
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
