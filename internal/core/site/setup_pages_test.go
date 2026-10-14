package site

// `abcd site setup --provider github-pages` (itd-2610091010348892): the switch
// asked at setup, the release workflow added and nothing else, public
// repositories only, and a switched-off site that publishes nothing. The forge
// is the in-process fake, the Pages API githubpagestest's. Nothing here reaches
// a network.

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
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages/githubpagestest"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
)

// asked answers every question the same way and records each one.
type asked struct {
	yes       bool
	questions []string
}

func (a *asked) Confirm(q string) bool { a.questions = append(a.questions, q); return a.yes }

// noCredential fails the test if anything resolves a credential: the Pages
// provider acts through the forge and reads none.
type noCredential struct{ t *testing.T }

func (c noCredential) Resolve(name string) (string, error) {
	c.t.Errorf("the credential %q was resolved; the github-pages provider reads none", name)
	return "", errors.New("no")
}

type pagesHarness struct {
	*harness
	pages *githubpagestest.Server
	ask   *asked
}

func newPagesHarness(t *testing.T) *pagesHarness {
	t.Helper()
	h := &pagesHarness{harness: newHarness(t), pages: githubpagestest.New(), ask: &asked{yes: true}}
	h.pages.Repos = map[string]*githubpagestest.Repo{"example-owner/example-site": {}}
	return h
}

func (h *pagesHarness) run(t *testing.T, mutate ...func(*SetupRequest)) SetupResult {
	t.Helper()
	return h.harness.run(t, append([]func(*SetupRequest){func(r *SetupRequest) {
		r.Provider = "github-pages"
		r.Adapter = githubpages.Adapter{API: h.pages.API}
		r.Credentials = noCredential{t}
		r.Asker = h.ask
	}}, mutate...)...)
}

// pagesFiles is the whole repository half for the github-pages provider: the
// project's abcd site settings (the composition holding the switch, and the
// site's inputs setup writes for every provider) and the release workflow.
var pagesFiles = []string{
	ManifestRelPath, "site-src/ui.json", "site-src/site.css", "site-src/site.js", "site-src/record.js",
	"site-src/headers", SiteWorkflowRelPath,
}

func changedPaths(t *testing.T, h *harness) []string {
	t.Helper()
	var changed []string
	for _, line := range strings.Split(strings.TrimRight(h.repo.Git("status", "--porcelain", "-uall"), "\n"), "\n") {
		if line != "" {
			changed = append(changed, strings.TrimSpace(line[2:]))
		}
	}
	sort.Strings(changed)
	return changed
}

func TestSwitchingOnAProjectSiteSetsTheSwitchAndAddsTheWorkflowOnly(t *testing.T) {
	h := newPagesHarness(t)
	res := h.run(t)

	if len(h.ask.questions) == 0 || !strings.Contains(h.ask.questions[0], "github-pages") ||
		!strings.Contains(h.ask.questions[0], "https://example-owner.github.io/example-site/") {
		t.Fatalf("the first question is not the switch naming the address: %q", h.ask.questions)
	}
	want := append([]string(nil), pagesFiles...)
	sort.Strings(want)
	if got := changedPaths(t, h.harness); !reflect.DeepEqual(got, want) {
		t.Fatalf("the repository changed in\n  %v\nwant exactly\n  %v", got, want)
	}
	m, err := LoadManifest(h.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if m.Hosting == nil || m.Hosting.Provider != "github-pages" || !m.Hosting.On() {
		t.Fatalf("hosting = %+v, want github-pages switched on", m.Hosting)
	}
	if !strings.Contains(string(mustRead(t, h.repo.Root(), ManifestRelPath)), `"publish": true`) {
		t.Fatal("the composition does not carry the switch")
	}

	wf := string(mustRead(t, h.repo.Root(), SiteWorkflowRelPath))
	for _, s := range []string{"environment: github-pages\n", "      pages: write\n", "      id-token: write\n",
		"--base-path /example-site/", "jq -e '.hosting.publish == true' .abcd/site.json"} {
		if !strings.Contains(wf, s) {
			t.Errorf("the workflow lacks %q", s)
		}
	}

	for _, env := range []string{EnvRender, githubpages.Environment} {
		pol := h.forge.policies[env]
		if !reflect.DeepEqual(pol, []BranchPolicy{{Name: "main", Type: "branch"}, {Name: "v*", Type: "tag"}}) {
			t.Errorf("environment %s policies = %v", env, pol)
		}
	}
	if _, ok := h.forge.envs[EnvDeploy]; ok {
		t.Errorf("the cloudflare deploy environment %q was created for a Pages site", EnvDeploy)
	}
	if res.Host.Status != HostWritten || res.Host.Address != "https://example-owner.github.io/example-site/" {
		t.Fatalf("host = %+v", res.Host)
	}
	if p := h.pages.Repos["example-owner/example-site"].Pages; p == nil || p.BuildType != "workflow" {
		t.Fatalf("the Pages site = %+v, want one built by the workflow", p)
	}
	remaining := strings.Join(res.Remaining, "\n")
	if strings.Contains(remaining, "gh secret set") || !strings.Contains(remaining, "git add") {
		t.Errorf("remaining steps:\n%s", remaining)
	}
	if res.Status != StatusChanged {
		t.Fatalf("status = %q", res.Status)
	}

	// The site the composition declares builds below the project's path.
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: h.repo.Root(), OutDir: out, Stamp: fixtureStamp, BasePath: "/example-site/"}); err != nil {
		t.Fatalf("the switched-on composition does not build: %v", err)
	}
	if res, err := Check(CheckRequest{RepoRoot: h.repo.Root(), OutDir: out, BasePath: "/example-site/"}); err != nil {
		t.Fatal(err)
	} else {
		for _, f := range res.Findings {
			if f.Check == CheckBasePath {
				t.Errorf("the Pages build links outside its path: %+v", f)
			}
		}
	}
}

func TestDecliningTheSwitchChangesNothing(t *testing.T) {
	h := newPagesHarness(t)
	h.ask.yes = false
	res := h.run(t)
	if res.Status != StatusDeclined {
		t.Fatalf("status = %q, want declined", res.Status)
	}
	if got := changedPaths(t, h.harness); len(got) != 0 {
		t.Fatalf("a declined switch changed %v", got)
	}
	// The host is read before the question (the seam's one call before a
	// confirmation); nothing is written.
	if len(h.forge.writes) != 0 || len(pagesWrites(h)) != 0 {
		t.Fatalf("a declined switch wrote to the forge: %v %v", h.forge.writes, pagesWrites(h))
	}
	if len(h.ask.questions) != 1 {
		t.Fatalf("questions = %q, want the switch alone", h.ask.questions)
	}
	if !strings.Contains(strings.Join(res.Remaining, "\n"), "--provider github-pages") {
		t.Fatalf("the remaining steps do not say how to switch it on: %v", res.Remaining)
	}
}

func TestAPrivateRepositoryIsRefusedBeforeAnything(t *testing.T) {
	h := newPagesHarness(t)
	h.forge.private = true
	_, err := Setup(SetupRequest{
		RepoRoot: h.repo.Root(), Provider: "github-pages", Asker: h.ask, Forge: h.forge,
		Adapter: githubpages.Adapter{API: h.pages.API}, Credentials: noCredential{t}, Context: context.Background(),
	})
	if err == nil || !strings.Contains(err.Error(), "private") || !strings.Contains(err.Error(), "public repositories only") ||
		strings.Contains(err.Error(), "\n") {
		t.Fatalf("setup = %v, want a one-line refusal of a private repository", err)
	}
	if len(h.ask.questions) != 0 || len(changedPaths(t, h.harness)) != 0 || len(h.pages.CallLog()) != 0 {
		t.Fatalf("a private repository was asked %q, changed %v, called %v", h.ask.questions, changedPaths(t, h.harness), h.pages.CallLog())
	}
}

func TestAVisibilityTheForgeCannotReadIsRefused(t *testing.T) {
	h := newPagesHarness(t)
	h.forge.fail["Private"] = errors.New("HTTP 502")
	_, err := Setup(SetupRequest{
		RepoRoot: h.repo.Root(), Provider: "github-pages", Asker: h.ask, Forge: h.forge,
		Adapter: githubpages.Adapter{API: h.pages.API}, Credentials: noCredential{t}, Context: context.Background(),
	})
	if err == nil || !strings.Contains(err.Error(), "public") {
		t.Fatalf("setup = %v, want a refusal", err)
	}
	if len(changedPaths(t, h.harness)) != 0 {
		t.Fatal("an unreadable visibility still wrote files")
	}
}

// switchOff commits the switched-on files and turns the switch off.
func switchOff(t *testing.T, h *pagesHarness) {
	t.Helper()
	h.repo.Git("add", "-A")
	h.repo.Git("commit", "-q", "-m", "switch the project site on")
	raw := strings.Replace(string(mustRead(t, h.repo.Root(), ManifestRelPath)), `"publish": true`, `"publish": false`, 1)
	if err := os.WriteFile(filepath.Join(h.repo.Root(), ManifestRelPath), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestASwitchedOffSiteRemovesTheWorkflowAndPublishesNothing(t *testing.T) {
	h := newPagesHarness(t)
	h.run(t)
	switchOff(t, h)
	forgeWrites, pagesCalls := len(h.forge.writes), len(h.pages.CallLog())
	h.ask.questions = nil

	res := h.run(t)
	if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); !os.IsNotExist(err) {
		t.Fatalf("the workflow is still there (%v)", err)
	}
	if got := fileStatuses(res)[SiteWorkflowRelPath]; got != string(FileRemoved) {
		t.Fatalf("workflow outcome = %q, want removed", got)
	}
	if len(h.ask.questions) != 1 || !strings.Contains(h.ask.questions[0], SiteWorkflowRelPath) {
		t.Fatalf("questions = %q, want the removal alone", h.ask.questions)
	}
	if len(h.forge.writes) != forgeWrites || len(h.pages.CallLog()) != pagesCalls {
		t.Fatalf("a switched-off run reached the forge: %v %v", h.forge.writes[forgeWrites:], h.pages.CallLog()[pagesCalls:])
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "publishes nothing") {
		t.Fatalf("notes do not say the next release publishes nothing: %v", res.Notes)
	}
	if res.Status != StatusChanged {
		t.Fatalf("status = %q", res.Status)
	}
	// Nothing else in the repository moved.
	if got := changedPaths(t, h.harness); !reflect.DeepEqual(got, []string{ManifestRelPath, SiteWorkflowRelPath}) {
		t.Fatalf("changed %v, want the switch and the removed workflow", got)
	}

	// A third run has nothing left to do.
	h.ask.questions = nil
	if res = h.run(t); res.Status != StatusNoChange || len(h.ask.questions) != 0 {
		t.Fatalf("third run = %q, asked %q", res.Status, h.ask.questions)
	}
}

func TestASwitchedOffSiteKeepsAWorkflowEditedByHand(t *testing.T) {
	h := newPagesHarness(t)
	h.run(t)
	switchOff(t, h)
	wf := filepath.Join(h.repo.Root(), SiteWorkflowRelPath)
	if err := os.WriteFile(wf, append(mustRead(t, h.repo.Root(), SiteWorkflowRelPath), "# mine\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	h.ask.questions = nil
	res := h.run(t)
	if _, err := os.Stat(wf); err != nil {
		t.Fatal("a hand-edited workflow was removed")
	}
	if got := fileStatuses(res)[SiteWorkflowRelPath]; got != string(scaffold.StatusKept) {
		t.Fatalf("workflow outcome = %q, want kept", got)
	}
	if len(h.ask.questions) != 0 {
		t.Fatalf("asked %q about a workflow it will not remove", h.ask.questions)
	}
}

func TestDecliningTheRemovalKeepsTheWorkflow(t *testing.T) {
	h := newPagesHarness(t)
	h.run(t)
	switchOff(t, h)
	h.ask.yes = false
	res := h.run(t)
	if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
		t.Fatal("a declined removal removed the workflow")
	}
	if res.Status != StatusDeclined {
		t.Fatalf("status = %q, want declined", res.Status)
	}
}

func TestThePagesWorkflowHoldsNoSecretAndInterpolatesNothingIntoAShell(t *testing.T) {
	wf, err := renderSiteWorkflow("main", githubpages.Adapter{}, hosting.Site{Name: "widgets", Repo: "acme/widgets"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(wf)
	if strings.Contains(s, "{{ secrets.") {
		t.Error("the Pages workflow reads a secret")
	}
	if strings.Contains(s, "for the host configuration") {
		t.Error("the Pages deploy job checks out the repository, which it reads nothing from")
	}
	if !regexp.MustCompile(`(?m)^  deploy:\n    needs: render\n    if: needs.render.outputs.publish == 'true'\n`).MatchString(s) {
		t.Error("the deploy job is not gated on the switch")
	}
	if got := strings.Count(s, "        if: steps.switch.outputs.on == 'true'\n"); got != 5 {
		t.Errorf("%d render steps are gated on the switch, want the five after it", got)
	}
	if !strings.Contains(s, "    permissions:\n      contents: read\n      pages: write\n      id-token: write\n    steps:\n") {
		t.Error("the deploy job's permissions are not exactly the Pages three")
	}
	inRun, runIndent := false, 0
	for i, line := range strings.Split(s, "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if inRun && trimmed != "" && indent <= runIndent {
			inRun = false
		}
		if strings.HasPrefix(trimmed, "run:") {
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
}

func TestThePublishSwitchBelongsToASwitchedProvider(t *testing.T) {
	on := true
	bad := func(format string, args ...any) error { return errors.New(format) }
	if err := (&Hosting{Provider: "cloudflare", Name: "x", Publish: &on}).validate(bad); err == nil {
		t.Fatal("a cloudflare hosting block accepted the publish switch, which its workflow never reads")
	}
	if err := (&Hosting{Provider: "github-pages", Name: "x", Publish: &on}).validate(bad); err != nil {
		t.Fatalf("github-pages refused its own switch: %v", err)
	}
}

// pagesRequest is the setup request the Pages harness makes, for a test that
// expects Setup itself to refuse.
func pagesRequest(t *testing.T, h *pagesHarness) SetupRequest {
	return SetupRequest{
		RepoRoot: h.repo.Root(), Provider: "github-pages", Asker: h.ask, Forge: h.forge,
		Adapter: githubpages.Adapter{API: h.pages.API}, Credentials: noCredential{t}, Context: context.Background(),
	}
}

// pagesWrites are the calls that changed the Pages fake: everything but a read.
func pagesWrites(h *pagesHarness) []string {
	var out []string
	for _, c := range h.pages.CallLog() {
		if !strings.HasPrefix(c, "GET ") {
			out = append(out, c)
		}
	}
	return out
}

// refusedBeforeAnything asserts a refusal changed nothing anywhere.
func refusedBeforeAnything(t *testing.T, h *pagesHarness, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "\n") {
		t.Fatalf("setup = %v, want a one-line refusal naming %q", err, want)
	}
	if got := changedPaths(t, h.harness); len(got) != 0 {
		t.Fatalf("the refused run changed %v", got)
	}
	if len(h.forge.writes) != 0 || len(pagesWrites(h)) != 0 {
		t.Fatalf("the refused run reached the forge: %v %v", h.forge.writes, pagesWrites(h))
	}
	if len(h.ask.questions) != 0 {
		t.Fatalf("the refused run asked %q", h.ask.questions)
	}
}

// A repository that already serves Pages at a custom domain, set up without
// --domain, would get a workflow built for /<repo>/ while GitHub serves the
// site at the domain's root, so every asset would 404. Setup refuses before
// it asks or writes anything, and says which flag to pass.
func TestALiveDomainWithNoneAskedForIsRefusedBeforeAnything(t *testing.T) {
	h := newPagesHarness(t)
	h.pages.Repos["example-owner/example-site"].Pages = &githubpagestest.Pages{BuildType: "workflow", CNAME: "docs.example.com"}
	_, err := Setup(pagesRequest(t, h))
	refusedBeforeAnything(t, h, err, "pass --domain docs.example.com")
}

// A Pages site GitHub serves below a path other than the one the workflow
// builds for is refused before anything is written.
func TestASiteServedBelowAnotherPathIsRefusedBeforeAnything(t *testing.T) {
	h := newPagesHarness(t)
	h.pages.Repos["example-owner/example-site"].Pages = &githubpagestest.Pages{BuildType: "workflow",
		HTMLURL: "https://example-owner.github.io/elsewhere/"}
	_, err := Setup(pagesRequest(t, h))
	refusedBeforeAnything(t, h, err, "/elsewhere/")
}

// The host stage holds the same line on its own: an address served below a
// path other than the profile's base path is refused, never reported live.
func TestTheHostStageRefusesASiteServedBelowAnotherPath(t *testing.T) {
	f := githubpagestest.New()
	f.Repos["acme/widgets"].Pages = &githubpagestest.Pages{BuildType: "workflow", HTMLURL: "https://acme.github.io/elsewhere/"}
	out, declined, refused := setupHost(context.Background(), githubpages.Adapter{API: f.API},
		hosting.Site{Name: "widgets", Repo: "acme/widgets"}, true, SetupRequest{Asker: answer(true)})
	if !refused || declined || out.Status != HostRefused {
		t.Fatalf("host = %+v (declined %v, refused %v), want refused", out, declined, refused)
	}
	if out.Address != "" || !strings.Contains(out.Detail, "/elsewhere/") || !strings.Contains(out.Detail, "/widgets/") {
		t.Fatalf("host = %+v, want no address and a detail naming both paths", out)
	}
}
