package site

// The GitHub forge over an in-process fake of the forge's API
// (iss-2609260927217168): every list the environment check depends on is read
// page by page until it is whole, and a page that cannot be read fails the read
// rather than shortening it. Nothing here reaches a network.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// fakeGHAPI answers `gh api` the way the forge does: list endpoints page at
// per_page (default 30, at most 100) and report total_count, and a write is
// recorded rather than applied.
type fakeGHAPI struct {
	repo string
	// lists maps an endpoint path (no query) to every entry it holds.
	lists map[string][]map[string]any
	// objects maps an endpoint path to the one object it returns.
	objects map[string]map[string]any
	// failPage makes one page of one list fail.
	failPage map[string]int
	// shortPage makes one page of one list come back empty, as a list that
	// shrank between two page reads would.
	shortPage map[string]int
	// fail makes every read of a path fail.
	fail   map[string]error
	writes []string
	reads  []string
}

func newFakeGHAPI() *fakeGHAPI {
	return &fakeGHAPI{
		repo: "example-owner/example-site", lists: map[string][]map[string]any{}, objects: map[string]map[string]any{},
		failPage: map[string]int{}, shortPage: map[string]int{}, fail: map[string]error{},
	}
}

// listKey is the field a list endpoint's entries arrive under.
func listKey(path string) string {
	switch {
	case strings.HasSuffix(path, "/deployment-branch-policies"):
		return "branch_policies"
	case strings.HasSuffix(path, "/secrets"):
		return "secrets"
	case strings.HasSuffix(path, "/environments"):
		return "environments"
	}
	return ""
}

func (f *fakeGHAPI) gh(_ string, stdin []byte, args ...string) ([]byte, error) {
	method, target := "GET", ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--method" && i+1 < len(args):
			method = args[i+1]
			i++
		case strings.HasPrefix(args[i], "repos/"):
			target = args[i]
		}
	}
	u, err := url.Parse(target)
	if err != nil || target == "" {
		return nil, fmt.Errorf("fake gh: no endpoint in %q", args)
	}
	path := u.Path
	if method != "GET" {
		f.writes = append(f.writes, method+" "+path+" "+string(stdin))
		return []byte("{}"), nil
	}
	f.reads = append(f.reads, target)
	if err := f.fail[path]; err != nil {
		return nil, err
	}
	if obj, ok := f.objects[path]; ok {
		return json.Marshal(obj)
	}
	key := listKey(path)
	if key == "" {
		return nil, fmt.Errorf("fake gh: HTTP 404 for %s", path)
	}
	q := u.Query()
	per, page := 30, 1
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil {
		per = min(v, 100)
	}
	if v, err := strconv.Atoi(q.Get("page")); err == nil {
		page = v
	}
	if f.failPage[path] == page {
		return nil, fmt.Errorf("fake gh: HTTP 502 for page %d of %s", page, path)
	}
	all := f.lists[path]
	var slice []map[string]any
	if lo := (page - 1) * per; lo < len(all) && f.shortPage[path] != page {
		slice = all[lo:min(lo+per, len(all))]
	}
	if slice == nil {
		slice = []map[string]any{}
	}
	return json.Marshal(map[string]any{"total_count": len(all), key: slice})
}

func (f *fakeGHAPI) forge() *ghForge {
	return &ghForge{dir: "", repo: f.repo, gh: f.gh}
}

func (f *fakeGHAPI) envs() string { return "repos/" + f.repo + "/environments" }

func (f *fakeGHAPI) env(name, rest string) string {
	return "repos/" + f.repo + "/environments/" + name + rest
}

// crowded fills the fake with more than a page of everything, the entries
// setup looks for placed past the first hundred.
func crowded() *fakeGHAPI {
	f := newFakeGHAPI()
	f.objects["repos/"+f.repo] = map[string]any{"default_branch": "main"}
	for i := 0; i < 120; i++ {
		f.lists[f.envs()] = append(f.lists[f.envs()], map[string]any{"name": fmt.Sprintf("preview-%03d", i)})
	}
	for _, env := range []string{EnvRender, EnvDeploy} {
		f.lists[f.envs()] = append(f.lists[f.envs()], map[string]any{
			"name": env, "deployment_branch_policy": map[string]bool{"protected_branches": false, "custom_branch_policies": true},
		})
		f.lists[f.env(env, "/deployment-branch-policies")] = []map[string]any{
			{"name": "main", "type": "branch"}, {"name": "v*", "type": "tag"},
		}
	}
	for i := 0; i < 130; i++ {
		f.lists[f.env(EnvDeploy, "/secrets")] = append(f.lists[f.env(EnvDeploy, "/secrets")], map[string]any{"name": fmt.Sprintf("OTHER_%03d", i)})
	}
	f.lists[f.env(EnvDeploy, "/secrets")] = append(f.lists[f.env(EnvDeploy, "/secrets")], map[string]any{"name": "CLOUDFLARE_API_TOKEN"})
	return f
}

func TestTheForgeReadsEveryPageOfEachList(t *testing.T) {
	f := crowded()
	var policies []map[string]any
	for i := 0; i < 150; i++ {
		policies = append(policies, map[string]any{"name": fmt.Sprintf("release/%03d", i), "type": "branch"})
	}
	policies = append(policies, map[string]any{"name": "*", "type": "branch"})
	f.lists[f.env(EnvDeploy, "/deployment-branch-policies")] = policies
	g := f.forge()

	envs, err := g.Environments(t.Context())
	if err != nil {
		t.Fatalf("environments: %v", err)
	}
	if len(envs) != 122 {
		t.Errorf("read %d environments, the forge holds 122", len(envs))
	}
	for _, env := range []string{EnvRender, EnvDeploy} {
		if st, ok := envs[env]; !ok || !st.CustomBranchPolicies {
			t.Errorf("environment %s past the first page reads as %+v, %v", env, st, ok)
		}
	}

	got, err := g.Policies(t.Context(), EnvDeploy)
	if err != nil {
		t.Fatalf("policies: %v", err)
	}
	if len(got) != 151 || got[150] != (BranchPolicy{Name: "*", Type: "branch"}) {
		t.Errorf("read %d policies (the forge holds 151, the last branch *)", len(got))
	}

	names, err := g.SecretNames(t.Context(), EnvDeploy)
	if err != nil {
		t.Fatalf("secrets: %v", err)
	}
	if len(names) != 131 || names[130] != "CLOUDFLARE_API_TOKEN" {
		t.Errorf("read %d secret names (the forge holds 131, CLOUDFLARE_API_TOKEN among them)", len(names))
	}
}

// TestAListThatCannotBeReadWholeIsAnError is the fail-closed half: a page the
// forge will not serve, or a page that comes back short of the total the forge
// reported, fails the read. A shortened list would read an environment as
// absent and have setup rewrite it.
func TestAListThatCannotBeReadWholeIsAnError(t *testing.T) {
	reads := map[string]func(g *ghForge) error{
		"environments": func(g *ghForge) error { _, err := g.Environments(t.Context()); return err },
		"policies": func(g *ghForge) error {
			_, err := g.Policies(t.Context(), EnvDeploy)
			return err
		},
		"secrets": func(g *ghForge) error { _, err := g.SecretNames(t.Context(), EnvDeploy); return err },
	}
	paths := map[string]func(f *fakeGHAPI) string{
		"environments": func(f *fakeGHAPI) string { return f.envs() },
		"policies":     func(f *fakeGHAPI) string { return f.env(EnvDeploy, "/deployment-branch-policies") },
		"secrets":      func(f *fakeGHAPI) string { return f.env(EnvDeploy, "/secrets") },
	}
	for name, read := range reads {
		t.Run(name+" page unreadable", func(t *testing.T) {
			f := crowded()
			p := paths[name](f)
			for len(f.lists[p]) <= 100 {
				f.lists[p] = append(f.lists[p], map[string]any{"name": fmt.Sprintf("pad-%d", len(f.lists[p])), "type": "branch"})
			}
			f.failPage[p] = 2
			if err := read(f.forge()); err == nil {
				t.Fatal("a list whose second page failed read as whole")
			}
		})
		t.Run(name+" page short", func(t *testing.T) {
			f := crowded()
			p := paths[name](f)
			for len(f.lists[p]) <= 100 {
				f.lists[p] = append(f.lists[p], map[string]any{"name": fmt.Sprintf("pad-%d", len(f.lists[p])), "type": "branch"})
			}
			f.shortPage[p] = 2
			if err := read(f.forge()); err == nil {
				t.Fatal("a list that came back short of its total read as whole")
			}
		})
	}
}

// TestSetupNeverRewritesAnEnvironmentPastTheFirstPage is the consequence the
// unpaginated read had: an existing environment past the hundredth read as
// absent, so setup would have written it through the endpoint that replaces its
// whole protection set.
func TestSetupNeverRewritesAnEnvironmentPastTheFirstPage(t *testing.T) {
	h := newHarness(t)
	f := crowded()
	res := h.run(t, func(r *SetupRequest) { r.Forge = f.forge() })
	for _, w := range f.writes {
		if strings.HasPrefix(w, "PUT ") {
			t.Errorf("setup rewrote an existing environment: %s", w)
		}
	}
	for _, e := range res.Environments {
		if e.Status != RemoteCurrent {
			t.Errorf("environment %s is %s, want %s: %v", e.Name, e.Status, RemoteCurrent, e.Changes)
		}
	}
	if strings.Contains(strings.Join(res.Remaining, "\n"), "gh secret set CLOUDFLARE_API_TOKEN") {
		t.Errorf("the secret past the first page is named as missing: %v", res.Remaining)
	}
}

// TestAnEmptyListIsWholeAtOnePage: a repository with no environments is one
// read, and the forge may leave the entries field out when there are none.
func TestAnEmptyListIsWholeAtOnePage(t *testing.T) {
	f := newFakeGHAPI()
	envs, err := f.forge().Environments(t.Context())
	if err != nil || len(envs) != 0 || len(f.reads) != 1 {
		t.Fatalf("an empty environment list read as %v, %v after %d reads", envs, err, len(f.reads))
	}
	g := &ghForge{repo: f.repo, gh: func(string, []byte, ...string) ([]byte, error) { return []byte(`{"total_count":0}`), nil }}
	if names, err := g.SecretNames(t.Context(), EnvDeploy); err != nil || len(names) != 0 {
		t.Fatalf("a list with no entries field read as %v, %v", names, err)
	}
	g.gh = func(string, []byte, ...string) ([]byte, error) { return []byte(`{"total_count":3}`), nil }
	if _, err := g.SecretNames(t.Context(), EnvDeploy); err == nil {
		t.Fatal("a list reporting entries it did not serve read as whole")
	}
}

// TestTheForgeNamesTheDefaultBranch is iss-2609260927214634's forge half: the
// default branch is the repository object's default_branch, and a response
// that names none is an error, never an empty branch.
func TestTheForgeNamesTheDefaultBranch(t *testing.T) {
	f := newFakeGHAPI()
	f.objects["repos/"+f.repo] = map[string]any{"default_branch": "trunk"}
	if b, err := f.forge().DefaultBranch(t.Context()); err != nil || b != "trunk" {
		t.Fatalf("default branch read as %q, %v; want trunk", b, err)
	}
	f.objects["repos/"+f.repo] = map[string]any{"name": "example-site"}
	if b, err := f.forge().DefaultBranch(t.Context()); err == nil {
		t.Fatalf("a repository naming no default branch read as %q", b)
	}
}
