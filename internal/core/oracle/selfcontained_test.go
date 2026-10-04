package oracle

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/credential"
)

// Ruling DR5 of 2026-09-29: a provider that holds a key (a paid one) takes
// only a self-contained agent by default, one whose emitted request carries
// all its input, and a file-reading agent is refused before any call with a
// reason. The person's override is a machine-only list,
// oracle.bundled_context_providers, naming providers that may take
// bundled-context requests for file-reading agents. No test reaches a network:
// every provider is an httptest fake, and every key is built at run time.

// pointAt writes a machine configuration whose keyed provider openrouter is
// the fake at base, stores dispatchKey under its credential name, points each
// agent at its one model, and adds extra (a JSON member list, or "") to the
// oracle block.
func pointAt(t *testing.T, base, extra string, agents ...string) (*fx, *APIConfig) {
	t.Helper()
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "openrouter", dispatchKey); err != nil {
		t.Fatal(err)
	}
	roles := make([]string, 0, len(agents))
	for _, a := range agents {
		roles = append(roles, `"`+a+`":"openrouter/typesafe/jev-1.13"`)
	}
	if extra != "" {
		extra = "," + extra
	}
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"` + base + `","key":"openrouter",` +
		`"models":["typesafe/jev-1.13"]}},"roles":{` + strings.Join(roles, ",") + `}` + extra + `}}`)
	return f, f.loadAPI()
}

// dispatchAs resolves agent against f's tables and c's connections and
// dispatches one step.
func dispatchAs(t *testing.T, f *fx, c *APIConfig, agent string) error {
	t.Helper()
	r, err := Resolve(agent, f.load(), c.Connections())
	if err != nil {
		t.Fatalf("Resolve(%s): %v", agent, err)
	}
	if !r.OnProvider() {
		t.Fatalf("%s resolved to %q; want the pointed provider", agent, r.ConnectionUsed)
	}
	_, _, err = c.Dispatch(context.Background(), credential.Machine(f.roots.Home), r, dispatchBrief, verdictContract)
	return err
}

// TestTheSelfContainedListIsTheFourReadingPositions: the list compiled into
// the binary starts with the four cold-reading positions and nothing else, and
// each is an agent in the roster, so a name on it can be pointed at all.
func TestTheSelfContainedListIsTheFourReadingPositions(t *testing.T) {
	want := []string{"cold-reading-comparative", "cold-reading-detection", "cold-reading-entailment", "cold-reading-widening"}
	if got := SelfContained(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SelfContained() = %v, want %v", got, want)
	}
	for _, a := range want {
		if !inRoster(a) {
			t.Fatalf("%s is on the self-contained list and not in the roster", a)
		}
	}
	// The list is a copy: a caller cannot widen it.
	SelfContained()[0] = "intent-auditor"
	if SelfContained()[0] != want[0] {
		t.Fatal("SelfContained returned the list itself, not a copy")
	}
}

// TestAFileReadingAgentIsRefusedOnAKeyedProvider: DR5's default. An agent off
// the self-contained list, pointed at a provider that holds a key, is refused
// before any call, and the reason names the rule and the override setting.
func TestAFileReadingAgentIsRefusedOnAKeyedProvider(t *testing.T) {
	for _, agent := range []string{"intent-auditor", "scribe", "release-changelog-composer", "lifeboat-reviewer"} {
		t.Run(agent, func(t *testing.T) {
			p := newProvFake(t, 200, chat("typesafe/jev-1.13", `{"verdict":"keep"}`))
			f, c := pointAt(t, p.base(), "", agent)
			err := dispatchAs(t, f, c, agent)
			if err == nil {
				t.Fatal("Dispatch admitted a file-reading agent on a keyed provider")
			}
			for _, want := range []string{agent, "DR5", "self-contained", "oracle.bundled_context_providers", "~/.abcd.noindex/config.json"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("refusal %q does not name %q", err, want)
				}
			}
			if n := p.calls.Load(); n != 0 {
				t.Fatalf("provider called %d times; the refusal comes before any call", n)
			}
		})
	}
}

// TestASelfContainedAgentIsAdmittedOnAKeyedProvider: a cold-reading position's
// request carries its whole bundle, so DR5 admits it to a paid provider.
func TestASelfContainedAgentIsAdmittedOnAKeyedProvider(t *testing.T) {
	for _, agent := range SelfContained() {
		t.Run(agent, func(t *testing.T) {
			p := newProvFake(t, 200, chat("typesafe/jev-1.13", `{"verdict":"keep"}`))
			f, c := pointAt(t, p.base(), "", agent)
			if err := dispatchAs(t, f, c, agent); err != nil {
				t.Fatalf("Dispatch: %v", err)
			}
			if n := p.calls.Load(); n != 1 {
				t.Fatalf("provider called %d times, want 1", n)
			}
		})
	}
}

// TestAKeylessProviderIsOutsideDR5: DR5 rules on a paid provider, one that
// holds a key. A local server whose block names no key spends nothing of the
// person's, so a file-reading agent pointed at it is dispatched.
func TestAKeylessProviderIsOutsideDR5(t *testing.T) {
	p := newProvFake(t, 200, chat("local-model", `{"verdict":"keep"}`))
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"local":{"base_url":"` + p.base() + `","models":["local-model"]}},` +
		`"roles":{"intent-auditor":"local/local-model"}}}`)
	c := f.loadAPI()
	if err := dispatchAs(t, f, c, "intent-auditor"); err != nil {
		t.Fatalf("Dispatch to a keyless provider: %v", err)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
}

// TestTheOverrideIsReadFromTheMachineAlone: the override list is the person's,
// so it is read from ~/.abcd.noindex/config.json, and a repository's
// .abcd/config.json declaring it is refused the way a repository's provider
// block is.
func TestTheOverrideIsReadFromTheMachineAlone(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `}}}`)
	f.repoConfig(`{"oracle":{"bundled_context_providers":["openrouter"]}}`)
	err := f.loadAPIErr()
	for _, want := range []string{"oracle.bundled_context_providers", "repo layer", "~/.abcd.noindex/config.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err, want)
		}
	}

	f = newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"bundled_context_providers":["openrouter"]}}`)
	if got := f.loadAPI().BundledContextProviders(); !reflect.DeepEqual(got, []string{"openrouter"}) {
		t.Fatalf("BundledContextProviders() = %v, want [openrouter]", got)
	}
}

// TestTheOverrideIsValidated: the list names providers; a malformed entry is
// refused, and a name this machine has not configured is a diagnostic that
// admits nothing.
func TestTheOverrideIsValidated(t *testing.T) {
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"bundled_context_providers":["Not A Name"]}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "oracle.bundled_context_providers") {
		t.Fatalf("refusal %q does not name the setting", err)
	}
	f = newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"bundled_context_providers":"openrouter"}}`)
	if err := f.loadAPIErr(); !strings.Contains(err.Error(), "oracle.bundled_context_providers") {
		t.Fatalf("refusal %q does not name the setting", err)
	}
	f = newFx(t)
	f.machineConfig(`{"oracle":{"api":{` + openrouterBlock + `},"bundled_context_providers":["elsewhere"]}}`)
	c := f.loadAPI()
	if len(c.BundledContextProviders()) != 0 {
		t.Fatalf("an unconfigured provider was admitted to the override: %v", c.BundledContextProviders())
	}
	if !strings.Contains(strings.Join(c.Diagnostics, "\n"), "elsewhere") {
		t.Fatalf("diagnostics %v do not name the unconfigured provider", c.Diagnostics)
	}
}

// TestTheOverrideAdmitsNoAgentWhoseBundleIsNotBuilt: the override lets a
// named provider take bundled-context requests, and a request is bundled only
// for an agent whose bundle abcd builds. None is built yet (which files
// travel, a size cap and a scan before sending are not ruled), so a
// file-reading agent stays refused even on a provider the override names, and
// the refusal says why.
func TestTheOverrideAdmitsNoAgentWhoseBundleIsNotBuilt(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", `{"verdict":"keep"}`))
	f, c := pointAt(t, p.base(), `"bundled_context_providers":["openrouter"]`, "intent-auditor")
	err := dispatchAs(t, f, c, "intent-auditor")
	if err == nil {
		t.Fatal("the override admitted an agent whose bundle is not built")
	}
	for _, want := range []string{"intent-auditor", "oracle.bundled_context_providers", "no bundle"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err, want)
		}
	}
	if n := p.calls.Load(); n != 0 {
		t.Fatalf("provider called %d times; the refusal comes before any call", n)
	}
}
