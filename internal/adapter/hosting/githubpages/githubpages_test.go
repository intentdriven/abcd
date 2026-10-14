package githubpages

// The github-pages provider over githubpagestest's in-process fake of the
// GitHub REST API. Nothing here reaches a network.

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages/githubpagestest"
)

func connect(f *githubpagestest.Server) hosting.Provider { return Adapter{API: f.API}.Connect("") }

var widgets = hosting.Site{Name: "widgets", Repo: "acme/widgets"}

func TestTheRepositoryHalfHoldsNoSecret(t *testing.T) {
	a := Adapter{}
	if a.Name() != "github-pages" {
		t.Fatalf("name = %q", a.Name())
	}
	if a.CredentialName() != "" {
		t.Fatalf("credential = %q; the provider acts through the forge as the person and resolves none", a.CredentialName())
	}
	if len(a.Secrets()) != 0 {
		t.Fatalf("secrets = %v, want none", a.Secrets())
	}
	if rel, data := a.HostConfig(widgets); rel != "" || data != nil {
		t.Fatalf("host config = %q, want none", rel)
	}
	step := a.DeployStep()
	pinRe := regexp.MustCompile(`uses: actions/(upload-pages-artifact|deploy-pages)@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+\n`)
	if got := len(pinRe.FindAllString(step, -1)); got != 2 {
		t.Fatalf("the deploy step pins %d of the two Pages actions to a full commit:\n%s", got, step)
	}
	for _, want := range []string{"path: site\n", "id: deployment\n"} {
		if !strings.Contains(step, want) {
			t.Errorf("the deploy step lacks %q:\n%s", want, step)
		}
	}
	if strings.Contains(step, "secrets.") || strings.Contains(step, "${{") {
		t.Fatalf("the deploy step reads a secret or an expression:\n%s", step)
	}
}

func TestTheProfileIsTheRunsOwnTokenOnly(t *testing.T) {
	p := Adapter{}.Profile(widgets)
	want := hosting.Profile{
		Environment: "github-pages",
		Permissions: []string{"contents: read", "pages: write", "id-token: write"},
		Switched:    true, PublicOnly: true, BasePath: "/widgets/", Address: "https://acme.github.io/widgets/",
	}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("profile = %+v, want %+v", p, want)
	}
	for _, c := range []struct {
		site hosting.Site
		base string
	}{
		{hosting.Site{Name: "w", Repo: "acme/widgets", Domain: "docs.example.com"}, "/"},
		{hosting.Site{Name: "w", Repo: "Acme/acme.github.io"}, "/"},
		{hosting.Site{Name: "w", Repo: "acme/acme.github.io.old"}, "/acme.github.io.old/"},
		{hosting.Site{Name: "w"}, "/"},
	} {
		if got := (Adapter{}).Profile(c.site).BasePath; got != c.base {
			t.Errorf("base path for %+v = %q, want %q", c.site, got, c.base)
		}
	}
}

func TestInspectCreateAndAddressAProjectSite(t *testing.T) {
	f := githubpagestest.New()
	p := connect(f)
	st, err := p.Inspect(context.Background(), widgets)
	if err != nil || st.Exists || !st.Routed {
		t.Fatalf("inspect = %+v, %v; want absent and vacuously routed", st, err)
	}
	if err := p.Create(context.Background(), widgets); err != nil {
		t.Fatal(err)
	}
	if got := f.Repos["acme/widgets"].Pages; got == nil || got.BuildType != "workflow" {
		t.Fatalf("pages = %+v, want a site built by the workflow", got)
	}
	if st, err = p.Inspect(context.Background(), widgets); err != nil || !st.Exists {
		t.Fatalf("inspect after create = %+v, %v", st, err)
	}
	addr, err := p.Address(context.Background(), widgets)
	if err != nil || addr != "https://acme.github.io/widgets/" {
		t.Fatalf("address = %q, %v", addr, err)
	}
	if err := p.Route(context.Background(), widgets); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.CallLog() {
		if strings.HasPrefix(c, githubpagestest.UpdatePages) {
			t.Fatalf("a site with no domain set one: %v", f.CallLog())
		}
	}
}

func TestRouteSetsTheCustomDomain(t *testing.T) {
	f := githubpagestest.New()
	f.Repos["acme/widgets"].Pages = &githubpagestest.Pages{BuildType: "workflow"}
	s := widgets
	s.Domain = "docs.example.com"
	p := connect(f)
	st, err := p.Inspect(context.Background(), s)
	if err != nil || !st.Exists || st.Routed {
		t.Fatalf("inspect = %+v, %v; want existing and unrouted", st, err)
	}
	if err := p.Route(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if got := f.Repos["acme/widgets"].Pages.CNAME; got != "docs.example.com" {
		t.Fatalf("cname = %q", got)
	}
	if st, _ = p.Inspect(context.Background(), s); !st.Routed {
		t.Fatal("the domain does not read as routed after Route")
	}
	if addr, err := p.Address(context.Background(), s); err != nil || addr != "https://docs.example.com/" {
		t.Fatalf("address = %q, %v", addr, err)
	}
}

func TestInspectRefuses(t *testing.T) {
	for name, c := range map[string]struct {
		repo githubpagestest.Repo
		site hosting.Site
		want string
	}{
		"a private repository": {githubpagestest.Repo{Private: true}, widgets, "public repositories only"},
		"a site built from a branch": {githubpagestest.Repo{Pages: &githubpagestest.Pages{BuildType: "legacy"}}, widgets,
			"from a branch"},
		"a domain routed elsewhere": {githubpagestest.Repo{Pages: &githubpagestest.Pages{BuildType: "workflow", CNAME: "other.example.com"}},
			hosting.Site{Name: "widgets", Repo: "acme/widgets", Domain: "docs.example.com"}, "other.example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			f := githubpagestest.New()
			r := c.repo
			f.Repos["acme/widgets"] = &r
			_, err := connect(f).Inspect(context.Background(), c.site)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("inspect = %v, want a refusal naming %q", err, c.want)
			}
		})
	}
}

func TestEveryForgeFailureIsReported(t *testing.T) {
	boom := errors.New("exit status 1 (gh: Server Error (HTTP 500))")
	for _, route := range []string{githubpagestest.GetRepo, githubpagestest.GetPages, githubpagestest.CreatePages, githubpagestest.UpdatePages} {
		t.Run(route, func(t *testing.T) {
			f := githubpagestest.New()
			f.Repos["acme/widgets"].Pages = &githubpagestest.Pages{BuildType: "workflow"}
			if route == githubpagestest.CreatePages {
				f.Repos["acme/widgets"].Pages = nil
			}
			f.Fail[route] = boom
			s := widgets
			s.Domain = "docs.example.com"
			p := connect(f)
			var err error
			switch route {
			case githubpagestest.GetRepo, githubpagestest.GetPages:
				_, err = p.Inspect(context.Background(), s)
			case githubpagestest.CreatePages:
				err = p.Create(context.Background(), s)
			case githubpagestest.UpdatePages:
				err = p.Route(context.Background(), s)
			}
			if err == nil || !strings.Contains(err.Error(), "HTTP 500") || !strings.HasPrefix(err.Error(), "github-pages: ") {
				t.Fatalf("%s failure = %v", route, err)
			}
		})
	}
}

func TestAnUnsafeSiteIsRefusedBeforeACall(t *testing.T) {
	for _, s := range []hosting.Site{
		{Name: "w"},
		{Name: "w", Repo: "acme"},
		{Name: "w", Repo: "acme/widgets/extra"},
		{Name: "w", Repo: "acme/../widgets"},
		{Name: "w", Repo: "acme/wid gets"},
		{Name: "w", Repo: "acme/widgets", Domain: "https://x.example.com"},
	} {
		f := githubpagestest.New()
		p := connect(f)
		if _, err := p.Inspect(context.Background(), s); err == nil {
			t.Errorf("inspect accepted %+v", s)
		}
		if err := p.Create(context.Background(), s); err == nil {
			t.Errorf("create accepted %+v", s)
		}
		if n := len(f.CallLog()); n != 0 {
			t.Errorf("%+v reached the forge %d times", s, n)
		}
	}
}

func TestAnAddressThatIsNotAPlainHTTPSURLIsRefused(t *testing.T) {
	p := Adapter{API: func(context.Context, string, string, []byte) ([]byte, error) {
		return []byte(`{"build_type":"workflow","html_url":"javascript:alert(1)"}`), nil
	}}.Connect("")
	if addr, err := p.Address(context.Background(), widgets); err == nil {
		t.Fatalf("address = %q, want a refusal", addr)
	}
}

func TestWithoutTheForgeNothingIsCalled(t *testing.T) {
	if _, err := (Adapter{}).Connect("").Inspect(context.Background(), widgets); err == nil {
		t.Fatal("a provider with no forge inspected the site")
	}
}
