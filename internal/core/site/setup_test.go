package site

// `abcd site setup` end to end (itd-2609061543533170, spc-2609212141407459):
// the writes against a fixture managed repository, the forge against an
// in-process fake, the provider against cloudflaretest's fake API. Nothing here
// reaches a network.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare/cloudflaretest"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/gittest"
)

// --- the fakes -------------------------------------------------------------

type fakeForge struct {
	envs     map[string]EnvironmentState
	policies map[string][]BranchPolicy
	secrets  map[string][]string
	fail     map[string]error
	writes   []string
	// protection is each environment's protection rules other than its
	// deployment policy (required reviewers, a wait timer). The forge's
	// environment write replaces the whole set, so PutEnvironment clears it,
	// as the real endpoint would.
	protection map[string][]string
}

func newFakeForge() *fakeForge {
	return &fakeForge{
		envs: map[string]EnvironmentState{}, policies: map[string][]BranchPolicy{},
		secrets: map[string][]string{}, fail: map[string]error{}, protection: map[string][]string{},
	}
}

func (f *fakeForge) Repo() string { return "example-owner/example-site" }

func (f *fakeForge) Environments(context.Context) (map[string]EnvironmentState, error) {
	if err := f.fail["Environments"]; err != nil {
		return nil, err
	}
	out := map[string]EnvironmentState{}
	for k, v := range f.envs {
		out[k] = v
	}
	return out, nil
}

func (f *fakeForge) Policies(_ context.Context, env string) ([]BranchPolicy, error) {
	if err := f.fail["Policies"]; err != nil {
		return nil, err
	}
	return append([]BranchPolicy(nil), f.policies[env]...), nil
}

func (f *fakeForge) PutEnvironment(_ context.Context, env string) error {
	if err := f.fail["PutEnvironment"]; err != nil {
		return err
	}
	f.writes = append(f.writes, "put "+env)
	f.envs[env] = EnvironmentState{CustomBranchPolicies: true}
	delete(f.protection, env)
	return nil
}

func (f *fakeForge) AddPolicy(_ context.Context, env string, p BranchPolicy) error {
	if err := f.fail["AddPolicy"]; err != nil {
		return err
	}
	f.writes = append(f.writes, "policy "+env+" "+p.Type+" "+p.Name)
	f.policies[env] = append(f.policies[env], p)
	return nil
}

func (f *fakeForge) SecretNames(_ context.Context, env string) ([]string, error) {
	if err := f.fail["SecretNames"]; err != nil {
		return nil, err
	}
	return f.secrets[env], nil
}

type answer bool

func (a answer) Confirm(string) bool { return bool(a) }

type fixedCredential struct {
	value string
	err   error
}

func (c fixedCredential) Resolve(string) (string, error) { return c.value, c.err }

// --- the managed repository --------------------------------------------------

// newManagedRepo builds a small repository abcd manages: the marker block, an
// identity block and its pointer, one documentation page, a record with a
// glossary, and a github.com origin.
func newManagedRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("AGENTS.md", "# Example\n\n<!-- BEGIN ABCD -->\nmanaged\n<!-- END ABCD -->\n")
	r.Write(".abcd/positioning.json", `{
  "schema_version": 1,
  "block": {"file": ".abcd/development/IDENTITY.md", "heading": "Identity (canonical)"},
  "severity": "warn",
  "surfaces": []
}
`)
	r.Write(".abcd/development/IDENTITY.md", "# Identity\n\n## Identity (canonical)\n\n"+
		"- **Title:** Example Site\n- **Tagline:** An example repository.\n- **Pitch:** It exists to be rendered.\n")
	r.Write("docs/README.md", "# Example Site\n\nThe example repository's documentation.\n\n## Why\n\nBecause a test needs one.\n")
	r.Write(".abcd/record-lint.json", `{
  "roots": [".abcd/development"],
  "banned_tokens": [],
  "rules": {"record_schema": {"enabled": true, "severity": "blocker", "record_stores": {
    "adr": ".abcd/development/decisions/adrs"
  }}}
}
`)
	r.Write(".abcd/development/decisions/adrs/0001-a-decision.md", `---
id: adr-1
slug: a-decision
status: accepted
date: 2026-01-02
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: []
---

# ADR-1: A decision

The decision's body names a phase.
`)
	r.Write(".abcd/development/brief/glossary/README.md", "# Glossary\n\nThe example glossary.\n")
	r.Write(".abcd/development/brief/glossary/core/README.md", "# core\n\nThe core context.\n")
	r.Write(".abcd/development/brief/glossary/core/phase.md", glossaryTermFile(
		"phase", "core", "An ordered stretch of work.", `[]`, "A phase is a stretch of work.\n"))
	r.Commit("the example repository")
	r.Git("remote", "add", "origin", "https://github.com/example-owner/example-site.git")
	return r
}

type harness struct {
	repo  *gittest.Repo
	forge *fakeForge
	host  *cloudflaretest.Server
	cred  credential.Source
	ask   Asker
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return &harness{
		repo:  newManagedRepo(t),
		forge: newFakeForge(),
		host:  cloudflaretest.New(t),
		cred:  fixedCredential{err: credential.ErrNotSet},
		ask:   answer(true),
	}
}

func (h *harness) withCredential() *harness {
	h.cred = fixedCredential{value: cloudflaretest.Token}
	return h
}

func (h *harness) run(t *testing.T, mutate ...func(*SetupRequest)) SetupResult {
	t.Helper()
	req := SetupRequest{
		RepoRoot:    h.repo.Root(),
		Asker:       h.ask,
		Forge:       h.forge,
		Credentials: h.cred,
		Adapter:     cloudflare.Adapter{BaseURL: h.host.URL},
		Context:     context.Background(),
	}
	for _, m := range mutate {
		m(&req)
	}
	res, err := Setup(req)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	return res
}

func withDomain(d string) func(*SetupRequest) { return func(r *SetupRequest) { r.Domain = d } }

// setupFiles is the whole repository half, in the order the verb reports it.
var setupFiles = []string{
	ManifestRelPath, "site-src/ui.json", "site-src/site.css", "site-src/site.js", "site-src/record.js",
	"site-src/headers", SiteWorkflowRelPath, "wrangler.jsonc",
}

func fileStatuses(res SetupResult) map[string]string {
	out := map[string]string{}
	for _, f := range res.Files {
		out[f.Path] = string(f.Status)
	}
	return out
}

// --- criterion 1: no credential ------------------------------------------------

func TestSetupWithoutACredentialWritesTheRepositoryHalfAndSaysWhatRemains(t *testing.T) {
	h := newHarness(t)
	res := h.run(t)

	got := fileStatuses(res)
	for _, p := range setupFiles {
		if got[p] != "written" {
			t.Errorf("%s: %q, want written", p, got[p])
		}
		if _, err := os.Stat(filepath.Join(h.repo.Root(), p)); err != nil {
			t.Errorf("%s reported written and is not on disk: %v", p, err)
		}
	}
	for _, env := range []string{EnvRender, EnvDeploy} {
		pol := h.forge.policies[env]
		want := []BranchPolicy{{Name: "main", Type: "branch"}, {Name: "v*", Type: "tag"}}
		if !reflect.DeepEqual(pol, want) {
			t.Errorf("environment %s policies = %v, want %v", env, pol, want)
		}
		if !h.forge.envs[env].CustomBranchPolicies {
			t.Errorf("environment %s is not restricted to custom policies", env)
		}
	}
	if res.Host.Status != HostNoCredential {
		t.Fatalf("host status = %q, want %q", res.Host.Status, HostNoCredential)
	}
	if n := len(h.host.CallLog()); n != 0 {
		t.Fatalf("with no credential the provider was called %d times: %v", n, h.host.CallLog())
	}
	remaining := strings.Join(res.Remaining, "\n")
	for _, want := range []string{
		cloudflare.CredentialName, credential.StorePath,
		"gh secret set CLOUDFLARE_API_TOKEN --env site --repo example-owner/example-site",
		"gh secret set CLOUDFLARE_ACCOUNT_ID --env site --repo example-owner/example-site",
		"git add",
	} {
		if !strings.Contains(remaining, want) {
			t.Errorf("the remaining steps do not say %q:\n%s", want, remaining)
		}
	}
	if res.Status != StatusChanged {
		t.Fatalf("status = %q, want %q", res.Status, StatusChanged)
	}
}

// --- criterion 2: with a credential ---------------------------------------------

func TestSetupWithACredentialCreatesRoutesAndReportsTheHost(t *testing.T) {
	h := newHarness(t).withCredential()
	before := h.repo.Git("status", "--porcelain")
	if strings.TrimSpace(before) != "" {
		t.Fatalf("precondition: the fixture is dirty:\n%s", before)
	}
	res := h.run(t, withDomain("docs.example.com"))

	if res.Host.Status != HostWritten || res.Host.Address != "https://docs.example.com" {
		t.Fatalf("host = %+v, want written at https://docs.example.com", res.Host)
	}
	h.host.Mu.Lock()
	worker, routed := h.host.Workers["example-site"], h.host.Domains["docs.example.com"]
	auth := append([]string(nil), h.host.Auth...)
	h.host.Mu.Unlock()
	if !worker || routed != "example-site" {
		t.Fatalf("the host holds worker=%v domain→%q", worker, routed)
	}
	for _, a := range auth {
		if a != "Bearer "+cloudflaretest.Token {
			t.Fatalf("a request carried an unexpected Authorization header")
		}
	}

	// Nothing is written into the repository but the files above, and none of
	// them carries the credential.
	var changed []string
	for _, line := range strings.Split(strings.TrimRight(h.repo.Git("status", "--porcelain", "-uall"), "\n"), "\n") {
		if line == "" {
			continue
		}
		changed = append(changed, strings.TrimSpace(line[2:]))
	}
	sort.Strings(changed)
	want := append([]string(nil), setupFiles...)
	sort.Strings(want)
	if !reflect.DeepEqual(changed, want) {
		t.Fatalf("the repository changed in\n  %v\nwant exactly\n  %v", changed, want)
	}
	for _, p := range setupFiles {
		raw, err := os.ReadFile(filepath.Join(h.repo.Root(), p))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), cloudflaretest.Token) {
			t.Fatalf("%s carries the credential", p)
		}
	}
	for _, line := range append(append([]string{}, res.Remaining...), res.Notes...) {
		if strings.Contains(line, cloudflaretest.Token) {
			t.Fatal("the report carries the credential")
		}
	}
	if !strings.Contains(string(mustRead(t, h.repo.Root(), ManifestRelPath)), `"domain": "docs.example.com"`) {
		t.Fatal("the composition does not record the domain the host was routed for")
	}
}

func mustRead(t *testing.T, root, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// --- criterion 3: the seam -------------------------------------------------------

func TestTheProviderListHasOneAdapterBehindTheSeam(t *testing.T) {
	if got := Providers(); !reflect.DeepEqual(got, []string{"cloudflare"}) {
		t.Fatalf("providers = %v, want exactly [cloudflare]", got)
	}
	a, ok := adapterNamed("cloudflare")
	if !ok {
		t.Fatal("the listed provider does not resolve")
	}
	var _ hosting.Adapter = a
}

// --- criterion 4: the page set for any managed repository ------------------------

func TestSetupGivesAManagedRepositoryTheWholePageSet(t *testing.T) {
	h := newHarness(t)
	h.run(t)
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: h.repo.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("the composition setup wrote does not build: %v", err)
	}
	pages := htmlPages(t, out)
	for _, want := range []string{
		"index.html", "record/index.html", "record/adr/adr-1/index.html", "record/graph/index.html",
		"record/glossary/index.html", "record/health/index.html",
	} {
		if _, ok := pages[want]; !ok {
			t.Errorf("the managed repository's site lacks %s", want)
		}
	}
	if !strings.Contains(pages["record/index.html"], `class="panel tl"`) {
		t.Error("the managed repository's dashboard lacks the timeline")
	}
	if !strings.Contains(pages["index.html"], "The example repository") {
		t.Error("the landing page is not composed from the repository's own text")
	}
}

// --- criterion 5: re-runnable ------------------------------------------------------

func TestASecondRunWritesNothingAndSaysSo(t *testing.T) {
	h := newHarness(t).withCredential()
	h.run(t, withDomain("docs.example.com"))
	forgeWrites, hostWrites := len(h.forge.writes), h.host.Writes()

	res := h.run(t)
	if res.Status != StatusNoChange {
		t.Fatalf("second run status = %q, want %q", res.Status, StatusNoChange)
	}
	for _, f := range res.Files {
		if f.Status != "current" && f.Status != "kept" {
			t.Errorf("second run: %s is %q", f.Path, f.Status)
		}
	}
	if len(h.forge.writes) != forgeWrites {
		t.Errorf("second run wrote to the forge: %v", h.forge.writes[forgeWrites:])
	}
	if h.host.Writes() != hostWrites {
		t.Errorf("second run wrote to the host: %v", h.host.CallLog())
	}
	if res.Host.Status != HostCurrent || res.Host.Address != "https://docs.example.com" {
		t.Errorf("second run host = %+v", res.Host)
	}
}

// --- the confirmation and the failures -------------------------------------------

func TestADeclinedRunWritesNothingRemote(t *testing.T) {
	h := newHarness(t).withCredential()
	h.ask = answer(false)
	res := h.run(t)
	if len(h.forge.writes) != 0 || h.host.Writes() != 0 {
		t.Fatalf("a declined run wrote remotely: forge %v host %v", h.forge.writes, h.host.CallLog())
	}
	if res.Status != StatusDeclined || res.Host.Status != HostDeclined {
		t.Fatalf("status %q host %q, want declined", res.Status, res.Host.Status)
	}
	for _, e := range res.Environments {
		if e.Status != RemoteDeclined {
			t.Errorf("environment %s is %q, want declined", e.Name, e.Status)
		}
	}
}

// TestAnExistingEnvironmentIsNeverRewritten: the forge's environment write
// replaces the environment's whole protection set, so an environment that
// already exists is never written through it. One that admits more than the
// default branch and release tags is left as it is and named as a remaining
// step, ahead of any secret step — whether it admits more through its
// protection mode or through a custom rule of its own; one already on custom
// policies and nothing broader only gains the policies it lacks. An absent
// environment is still created.
func TestAnExistingEnvironmentIsNeverRewritten(t *testing.T) {
	cases := []struct {
		name     string
		state    EnvironmentState
		policies []BranchPolicy
		names    string
	}{
		{"admits every ref", EnvironmentState{}, nil, ""},
		{"admits protected branches", EnvironmentState{ProtectedBranches: true}, nil, ""},
		{"admits every branch through a custom rule", EnvironmentState{CustomBranchPolicies: true},
			[]BranchPolicy{{Name: "main", Type: "branch"}, {Name: "*", Type: "branch"}}, "branch *"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			h.forge.envs[EnvDeploy] = c.state
			h.forge.policies[EnvDeploy] = c.policies
			h.forge.protection[EnvDeploy] = []string{"required reviewers: example-reviewer"}
			res := h.run(t)

			for _, w := range h.forge.writes {
				if strings.HasSuffix(w, " "+EnvDeploy) || strings.Contains(w, " "+EnvDeploy+" ") {
					t.Errorf("the existing environment %s was written: %q (all writes %v)", EnvDeploy, w, h.forge.writes)
				}
			}
			if got := h.forge.protection[EnvDeploy]; len(got) != 1 {
				t.Errorf("the existing environment's reviewers were lost: %v", got)
			}
			if !reflect.DeepEqual(h.forge.writes, []string{
				"put " + EnvRender, "policy " + EnvRender + " branch main", "policy " + EnvRender + " tag v*",
			}) {
				t.Errorf("forge writes = %v, want only the absent %s created", h.forge.writes, EnvRender)
			}
			var deploy EnvironmentOutcome
			for _, e := range res.Environments {
				if e.Name == EnvDeploy {
					deploy = e
				}
			}
			if deploy.Status != RemoteUnrestricted {
				t.Errorf("%s status = %q, want %q", EnvDeploy, deploy.Status, RemoteUnrestricted)
			}
			restrict, secret := -1, -1
			for i, r := range res.Remaining {
				if restrict < 0 && strings.Contains(r, "restrict the existing environment "+EnvDeploy) &&
					strings.Contains(r, "branch main and tags v*") {
					restrict = i
				}
				if secret < 0 && strings.Contains(r, "gh secret set") {
					secret = i
				}
			}
			if restrict < 0 || secret < 0 || restrict > secret {
				t.Errorf("the restriction step is missing or follows a secret step (restrict %d, secret %d):\n%s",
					restrict, secret, strings.Join(res.Remaining, "\n"))
			} else if c.names != "" && !strings.Contains(res.Remaining[restrict], c.names) {
				t.Errorf("the restriction step does not name the rule %q it has to remove:\n%s", c.names, res.Remaining[restrict])
			}
		})
	}
	t.Run("already on custom policies", func(t *testing.T) {
		h := newHarness(t)
		h.forge.envs[EnvDeploy] = EnvironmentState{CustomBranchPolicies: true}
		h.forge.policies[EnvDeploy] = []BranchPolicy{{Name: "main", Type: "branch"}}
		h.forge.protection[EnvDeploy] = []string{"required reviewers: example-reviewer"}
		h.run(t)
		for _, w := range h.forge.writes {
			if w == "put "+EnvDeploy {
				t.Errorf("the existing environment %s was rewritten (writes %v)", EnvDeploy, h.forge.writes)
			}
		}
		if !reflect.DeepEqual(h.forge.policies[EnvDeploy], []BranchPolicy{{Name: "main", Type: "branch"}, {Name: "v*", Type: "tag"}}) {
			t.Errorf("%s policies = %v, want the missing tag policy added", EnvDeploy, h.forge.policies[EnvDeploy])
		}
		if len(h.forge.protection[EnvDeploy]) != 1 {
			t.Errorf("the existing environment's reviewers were lost: %v", h.forge.protection[EnvDeploy])
		}
	})
}

func TestEveryForgeFailureIsReportedAndStopsTheRemoteWrites(t *testing.T) {
	for _, call := range []string{"Environments", "Policies", "PutEnvironment", "AddPolicy"} {
		t.Run(call, func(t *testing.T) {
			h := newHarness(t).withCredential()
			if call == "Policies" {
				h.forge.envs[EnvRender] = EnvironmentState{CustomBranchPolicies: true}
			}
			h.forge.fail[call] = errors.New("forge said no")
			res := h.run(t)
			if res.Status != StatusRefused {
				t.Fatalf("status = %q, want refused", res.Status)
			}
			if !strings.Contains(strings.Join(res.Notes, "\n"), "forge said no") {
				t.Fatalf("the failure is not in the notes: %v", res.Notes)
			}
			if h.host.Writes() != 0 {
				t.Fatalf("the host was written after the forge failed: %v", h.host.CallLog())
			}
			// The secret steps are still printed, so the step that protects
			// the environment they land in comes first: without it the
			// workflow's first run creates site with no deployment policy.
			create, secret := -1, -1
			for i, r := range res.Remaining {
				if create < 0 && strings.Contains(r, "create the forge environments "+EnvRender+" and "+EnvDeploy) {
					create = i
				}
				if secret < 0 && strings.Contains(r, "gh secret set") {
					secret = i
				}
			}
			if create < 0 || secret < 0 || create > secret {
				t.Fatalf("a refused forge stage must name the environment step before any secret step (create %d, secret %d):\n%s",
					create, secret, strings.Join(res.Remaining, "\n"))
			}
		})
	}
	t.Run("SecretNames", func(t *testing.T) {
		h := newHarness(t)
		h.forge.fail["SecretNames"] = errors.New("forge said no")
		res := h.run(t)
		if !strings.Contains(strings.Join(res.Remaining, "\n"), "gh secret set CLOUDFLARE_API_TOKEN") {
			t.Fatalf("an unreadable secret list must still name the secret step: %v", res.Remaining)
		}
		if !strings.Contains(strings.Join(res.Notes, "\n"), "forge said no") {
			t.Fatalf("the failure is not in the notes: %v", res.Notes)
		}
	})
}

func TestEveryProviderFailureIsReported(t *testing.T) {
	for _, route := range []string{
		cloudflaretest.ListAccounts, cloudflaretest.ListWorkers, cloudflaretest.ListDomains,
		cloudflaretest.CreateWorker, cloudflaretest.AttachDomain,
	} {
		t.Run(route, func(t *testing.T) {
			h := newHarness(t).withCredential()
			h.host.Fail[route] = 500
			res := h.run(t, withDomain("docs.example.com"))
			if res.Status != StatusRefused || res.Host.Status != HostRefused {
				t.Fatalf("status %q host %q, want refused", res.Status, res.Host.Status)
			}
			if !strings.Contains(res.Host.Detail, "500") {
				t.Fatalf("the host's failure is not reported: %q", res.Host.Detail)
			}
			if strings.Contains(res.Host.Detail, cloudflaretest.Token) {
				t.Fatal("the failure carries the credential")
			}
		})
	}
	t.Run(cloudflaretest.GetSubdomain, func(t *testing.T) {
		h := newHarness(t).withCredential()
		h.host.Fail[cloudflaretest.GetSubdomain] = 500
		res := h.run(t)
		if res.Status != StatusRefused || !strings.Contains(res.Host.Detail, "500") {
			t.Fatalf("status %q host %+v, want the address failure refused", res.Status, res.Host)
		}
	})
}

func TestAnUnreadableCredentialIsRefusedNotSkipped(t *testing.T) {
	h := newHarness(t)
	h.cred = fixedCredential{err: errors.New("credential: the store is readable by others")}
	res := h.run(t)
	if res.Host.Status != HostRefused || res.Status != StatusRefused {
		t.Fatalf("status %q host %q, want refused", res.Status, res.Host.Status)
	}
	if len(h.host.CallLog()) != 0 {
		t.Fatalf("the provider was called without a credential: %v", h.host.CallLog())
	}
}

func TestSetupRefusesAFolderAbcdDoesNotManage(t *testing.T) {
	h := newHarness(t)
	h.repo.Write("AGENTS.md", "# Example\n")
	_, err := Setup(SetupRequest{RepoRoot: h.repo.Root(), Forge: h.forge, Credentials: h.cred, Context: context.Background()})
	if !errors.Is(err, ErrNotManaged) {
		t.Fatalf("err = %v, want ErrNotManaged", err)
	}
}

func TestDriftedMachineryRefusesTheWholeRun(t *testing.T) {
	h := newHarness(t).withCredential()
	h.repo.Write(SiteWorkflowRelPath, "name: site\n# hand-made\n")
	res := h.run(t)
	if res.Status != StatusRefused {
		t.Fatalf("status = %q, want refused", res.Status)
	}
	if _, err := os.Stat(filepath.Join(h.repo.Root(), ManifestRelPath)); !os.IsNotExist(err) {
		t.Fatal("a refused run wrote the composition")
	}
	if len(h.forge.writes) != 0 || len(h.host.CallLog()) != 0 {
		t.Fatalf("a refused run reached the forge %v or the host %v", h.forge.writes, h.host.CallLog())
	}
	res = h.run(t, func(r *SetupRequest) { r.Confirm = true })
	if fileStatuses(res)[SiteWorkflowRelPath] != "written" {
		t.Fatalf("--confirm did not replace the drifted workflow: %v", res.Files)
	}
}

func TestSetupRefusesWithoutAnIdentityOrAPage(t *testing.T) {
	h := newHarness(t)
	h.repo.Remove(".abcd/positioning.json")
	if _, err := Setup(SetupRequest{RepoRoot: h.repo.Root(), Forge: h.forge, Credentials: h.cred, Context: context.Background()}); err == nil ||
		!strings.Contains(err.Error(), "abcd identity init") {
		t.Fatalf("no identity block: err = %v, want one naming `abcd identity init`", err)
	}
	h = newHarness(t)
	h.repo.Remove("docs/README.md")
	if _, err := Setup(SetupRequest{RepoRoot: h.repo.Root(), Forge: h.forge, Credentials: h.cred, Context: context.Background()}); err == nil ||
		!strings.Contains(err.Error(), "docs/README.md") {
		t.Fatalf("no page: err = %v, want one naming docs/README.md", err)
	}
}

func TestAnUnsafeNameOrDomainIsRefused(t *testing.T) {
	h := newHarness(t)
	for _, m := range []func(*SetupRequest){
		func(r *SetupRequest) { r.Name = "Not_A_Name" },
		func(r *SetupRequest) { r.Domain = "https://example.com/x" },
	} {
		req := SetupRequest{RepoRoot: h.repo.Root(), Forge: h.forge, Credentials: h.cred, Context: context.Background()}
		m(&req)
		if _, err := Setup(req); err == nil {
			t.Fatalf("setup accepted %+v", req)
		}
	}
}

// --- the workflow ------------------------------------------------------------------

// TestTheWorkflowInterpolatesNothingIntoAShell: a `${{ … }}` expression inside a
// run block is spliced into the script before the shell parses it, which is
// the injection shape; every value reaches the shell through env instead.
func TestTheWorkflowInterpolatesNothingIntoAShell(t *testing.T) {
	wf, err := renderSiteWorkflow("main", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	inRun, runIndent := false, 0
	for i, line := range strings.Split(string(wf), "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if inRun && trimmed != "" && indent <= runIndent {
			inRun = false
		}
		if strings.HasPrefix(trimmed, "run:") || strings.HasPrefix(trimmed, "- run:") {
			if strings.Contains(trimmed, "${{") {
				t.Errorf("line %d interpolates into a shell: %s", i+1, trimmed)
			}
			inRun, runIndent = true, indent
			continue
		}
		if inRun && strings.Contains(line, "${{") {
			t.Errorf("line %d interpolates into a shell: %s", i+1, trimmed)
		}
	}
	for _, want := range []string{"environment: site-render", "environment: site\n", "persist-credentials: false"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("the workflow lacks %q", want)
		}
	}
}

// TestTheWorkflowPinsFollowAbcdsOwn: every action the setup workflow uses is
// pinned to the commit abcd's own site workflow uses, so a pin bump there is a
// failing test here rather than a quietly stale copy in every managed repo.
func TestTheWorkflowPinsFollowAbcdsOwn(t *testing.T) {
	own, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", "site.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf, err := renderSiteWorkflow("main", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	usesRe := regexp.MustCompile(`uses: ([^@\s]+)@([0-9a-f]{40})`)
	pins := map[string]string{}
	for _, m := range usesRe.FindAllStringSubmatch(string(own), -1) {
		pins[m[1]] = m[2]
	}
	found := usesRe.FindAllStringSubmatch(string(wf), -1)
	if len(found) == 0 {
		t.Fatal("the setup workflow pins no action")
	}
	for _, m := range found {
		if pins[m[1]] != m[2] {
			t.Errorf("%s is pinned to %s here and %q in abcd's own site workflow", m[1], m[2], pins[m[1]])
		}
	}
	if v := regexp.MustCompile(`wranglerVersion: '([^']+)'`).FindStringSubmatch(string(own)); v == nil ||
		!strings.Contains(string(wf), "wranglerVersion: '"+v[1]+"'") {
		t.Errorf("the wrangler version differs from abcd's own site workflow")
	}
}

// TestTheSeedSourcesAreAbcdsOwn: the static inputs setup seeds a managed
// repository with are byte-identical to the ones abcd's own site renders from.
func TestTheSeedSourcesAreAbcdsOwn(t *testing.T) {
	for _, name := range seedNames {
		own, err := os.ReadFile(filepath.Join("..", "..", "..", "site-src", name))
		if err != nil {
			t.Fatal(err)
		}
		seed, err := setupSources.ReadFile("setupsrc/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if string(own) != string(seed) {
			t.Errorf("setupsrc/%s differs from site-src/%s; copy it across", name, name)
		}
	}
}

// TestAWorkflowRunFromAForkNeverRenders: the workflow_run entry admits only a
// successful run of this repository's own release workflow, never a pull
// request's and never a fork's.
func TestAWorkflowRunFromAForkNeverRenders(t *testing.T) {
	wf, err := renderSiteWorkflow("main", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"github.event.workflow_run.conclusion == 'success'",
		"github.event.workflow_run.event != 'pull_request'",
		"github.event.workflow_run.head_repository.full_name == github.repository",
	} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("the render job's gate lacks %q", want)
		}
	}
}

// TestTheWorkflowFiresOnEveryReleasePath: on a repository `launch scaffold`
// laid out, auto-release runs release.yml by workflow_call, so no run named
// `release` exists for it; a hand-pushed tag's `release` run has the TAG as its
// head branch; and a release made with the workflow's own token fires no
// `release: published`. So the workflow_run entry names both release
// workflows, filters no branch in the trigger (a branch filter drops the tag),
// and admits the default branch or a v-tag inside the render job's gate.
func TestTheWorkflowFiresOnEveryReleasePath(t *testing.T) {
	wf, err := renderSiteWorkflow("trunk", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(wf), "\n")
	block := func(start string, end func(string) bool) string {
		var out []string
		in := false
		for _, l := range lines {
			if !in {
				in = strings.TrimRight(l, " ") == start
				continue
			}
			if end(l) {
				break
			}
			out = append(out, l)
		}
		if !in {
			t.Fatalf("the workflow has no %q", start)
		}
		return strings.Join(out, "\n")
	}
	on := block("on:", func(l string) bool { return l != "" && l[0] != ' ' && l[0] != '#' })
	wantNames := []string{"release", "auto-release"}
	if !strings.Contains(on, "    workflows: ["+strings.Join(wantNames, ", ")+"]") {
		t.Errorf("the workflow_run entry does not name %v:\n%s", wantNames, on)
	}
	if strings.Contains(on, "branches:") {
		t.Errorf("the trigger filters branches, which drops a tag-push release run:\n%s", on)
	}
	for _, name := range wantNames {
		src, err := os.ReadFile(filepath.Join("..", "launch", "scaffold", "templates", name+".yml.tmpl"))
		if err != nil {
			t.Fatal(err)
		}
		if first, _, _ := strings.Cut(string(src), "\n"); first != "name: "+name {
			t.Errorf("the scaffolded %s workflow is named %q, not the %q the site workflow listens for", name, first, name)
		}
	}
	gate := block("    if: >-", func(l string) bool { return strings.HasPrefix(l, "    runs-on:") })
	for _, want := range []string{
		"github.event.workflow_run.head_branch == 'trunk'",
		"startsWith(github.event.workflow_run.head_branch, 'v')",
		"github.event.workflow_run.head_repository.full_name == github.repository",
	} {
		if !strings.Contains(gate, want) {
			t.Errorf("the render job's gate lacks %q:\n%s", want, gate)
		}
	}
}
