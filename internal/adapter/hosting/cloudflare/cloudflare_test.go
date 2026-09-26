package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare/cloudflaretest"
)

func connect(t *testing.T, base string, token string) hosting.Provider {
	t.Helper()
	return Adapter{BaseURL: base}.Connect(token)
}

var site = hosting.Site{Name: "example-site", Domain: "docs.example.com"}

// TestCreateRouteAndReportAgainstTheFake walks the whole host half once: the
// read says nothing exists, the two writes create and route, the read after says
// both hold, and the address is the domain.
func TestCreateRouteAndReportAgainstTheFake(t *testing.T) {
	fake := cloudflaretest.New(t)
	p := connect(t, fake.URL, cloudflaretest.Token)
	ctx := context.Background()

	st, err := p.Inspect(ctx, site)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if st.Exists || st.Routed {
		t.Fatalf("a fresh account reports %+v, want nothing", st)
	}
	if fake.Writes() != 0 {
		t.Fatalf("Inspect wrote: %v", fake.CallLog())
	}
	if err := p.Create(ctx, site); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Route(ctx, site); err != nil {
		t.Fatalf("route: %v", err)
	}
	st, err = p.Inspect(ctx, site)
	if err != nil {
		t.Fatalf("inspect after: %v", err)
	}
	if !st.Exists || !st.Routed {
		t.Fatalf("after create and route, state is %+v", st)
	}
	addr, err := p.Address(ctx, site)
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if addr != "https://docs.example.com" {
		t.Fatalf("address = %q", addr)
	}
	body := fake.Bodies[cloudflaretest.AttachDomain][0]
	if body["hostname"] != "docs.example.com" || body["service"] != "example-site" {
		t.Fatalf("route body = %v", body)
	}
	created := fake.Bodies[cloudflaretest.CreateWorker][0]
	if created["name"] != "example-site" {
		t.Fatalf("create body = %v", created)
	}
}

// TestAddressWithoutADomainIsTheWorkersDevName: a site with no custom domain
// is served from the account's workers.dev subdomain, and routing it is a no-op.
func TestAddressWithoutADomainIsTheWorkersDevName(t *testing.T) {
	fake := cloudflaretest.New(t)
	p := connect(t, fake.URL, cloudflaretest.Token)
	s := hosting.Site{Name: "example-site"}
	ctx := context.Background()
	st, err := p.Inspect(ctx, s)
	if err != nil || !st.Routed {
		t.Fatalf("no-domain site: state %+v err %v; want Routed vacuously true", st, err)
	}
	if err := p.Route(ctx, s); err != nil {
		t.Fatalf("route: %v", err)
	}
	if fake.Writes() != 0 {
		t.Fatalf("routing a site with no domain wrote: %v", fake.CallLog())
	}
	addr, err := p.Address(ctx, s)
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if want := "https://example-site." + cloudflaretest.Subdomain + ".workers.dev"; addr != want {
		t.Fatalf("address = %q, want %q", addr, want)
	}
}

// TestEveryCallFailsLoudly makes each route the adapter calls fail in turn, and
// requires the step that made the call to return an error naming the route's
// status, never a success and never the credential.
func TestEveryCallFailsLoudly(t *testing.T) {
	steps := []struct {
		route string
		run   func(p hosting.Provider) error
	}{
		{cloudflaretest.ListAccounts, func(p hosting.Provider) error { _, err := p.Inspect(context.Background(), site); return err }},
		{cloudflaretest.ListWorkers, func(p hosting.Provider) error { _, err := p.Inspect(context.Background(), site); return err }},
		{cloudflaretest.ListDomains, func(p hosting.Provider) error { _, err := p.Inspect(context.Background(), site); return err }},
		{cloudflaretest.CreateWorker, func(p hosting.Provider) error { return p.Create(context.Background(), site) }},
		{cloudflaretest.AttachDomain, func(p hosting.Provider) error {
			_ = p.Create(context.Background(), site)
			return p.Route(context.Background(), site)
		}},
		{cloudflaretest.GetSubdomain, func(p hosting.Provider) error {
			_, err := p.Address(context.Background(), hosting.Site{Name: "example-site"})
			return err
		}},
	}
	for _, st := range steps {
		t.Run(st.route, func(t *testing.T) {
			fake := cloudflaretest.New(t)
			fake.Fail[st.route] = http.StatusInternalServerError
			err := st.run(connect(t, fake.URL, cloudflaretest.Token))
			if err == nil {
				t.Fatalf("%s failed on the host and the adapter reported success", st.route)
			}
			if !strings.Contains(err.Error(), "500") {
				t.Fatalf("the error does not carry the host's status: %v", err)
			}
			if strings.Contains(err.Error(), cloudflaretest.Token) {
				t.Fatalf("the error carries the credential: %v", err)
			}
		})
	}
}

// TestTheCredentialNeverReachesAnError: a host that echoes the Authorization
// header back in its error message must not get the token into the error the
// adapter returns, because that error is printed.
func TestTheCredentialNeverReachesAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1,"message":"bad header: ` + r.Header.Get("Authorization") + `"}],"result":null}`))
	}))
	defer srv.Close()
	_, err := connect(t, srv.URL, cloudflaretest.Token).Inspect(context.Background(), site)
	if err == nil {
		t.Fatal("a 403 read as success")
	}
	if strings.Contains(err.Error(), cloudflaretest.Token) {
		t.Fatalf("the credential reached the error: %v", err)
	}
}

// TestAWrongCredentialIsRefused: the fake refuses any other bearer, and the
// adapter reports the refusal.
func TestAWrongCredentialIsRefused(t *testing.T) {
	fake := cloudflaretest.New(t)
	if _, err := connect(t, fake.URL, "not-the-token").Inspect(context.Background(), site); err == nil ||
		!strings.Contains(err.Error(), "403") {
		t.Fatalf("a wrong credential: err = %v, want a 403", err)
	}
}

// TestAccountMustBeUnambiguous: the adapter takes the account from the token's
// reach, so a token that reaches none, or more than one, is refused before any
// write rather than guessed at.
func TestAccountMustBeUnambiguous(t *testing.T) {
	for _, tc := range []struct {
		name     string
		accounts []string
		want     string
	}{
		{"none", nil, "no account"},
		{"two", []string{cloudflaretest.Account, "fedcba9876543210fedcba9876543210"}, "2 accounts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := cloudflaretest.New(t)
			fake.Accounts = tc.accounts
			_, err := connect(t, fake.URL, cloudflaretest.Token).Inspect(context.Background(), site)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one naming %q", err, tc.want)
			}
		})
	}
}

// TestADomainOnAnotherHostIsNotMoved: a domain already routed to a different
// Worker is somebody's live site, and the adapter refuses rather than
// re-pointing it.
func TestADomainOnAnotherHostIsNotMoved(t *testing.T) {
	fake := cloudflaretest.New(t)
	fake.Workers["other-site"] = true
	fake.Domains["docs.example.com"] = "other-site"
	_, err := connect(t, fake.URL, cloudflaretest.Token).Inspect(context.Background(), site)
	if err == nil || !strings.Contains(err.Error(), "other-site") {
		t.Fatalf("err = %v, want a refusal naming the Worker that holds the domain", err)
	}
}

// TestTheWorkerListIsPaged: a Worker on the second page of the listing exists.
func TestTheWorkerListIsPaged(t *testing.T) {
	fake := cloudflaretest.New(t)
	fake.PageSize = 1
	fake.Workers["a-first"] = true
	fake.Workers["example-site"] = true
	st, err := connect(t, fake.URL, cloudflaretest.Token).Inspect(context.Background(), hosting.Site{Name: "example-site"})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !st.Exists {
		t.Fatal("a Worker on the second page was reported absent")
	}
}

// TestARedirectIsNotFollowed: a redirect would carry the request somewhere the
// base address did not name.
func TestARedirectIsNotFollowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://192.0.2.1/elsewhere", http.StatusFound)
	}))
	defer srv.Close()
	if _, err := connect(t, srv.URL, cloudflaretest.Token).Inspect(context.Background(), site); err == nil {
		t.Fatal("a redirect was followed or read as success")
	}
}

// TestAnUnsafeNameNeverReachesARequest: the name becomes a path and a body
// field, so the adapter holds it to the host-name charset itself.
func TestAnUnsafeNameNeverReachesARequest(t *testing.T) {
	fake := cloudflaretest.New(t)
	p := connect(t, fake.URL, cloudflaretest.Token)
	for _, s := range []hosting.Site{{Name: "../accounts"}, {Name: "ok-name", Domain: "https://example.com"}} {
		if err := p.Create(context.Background(), s); err == nil {
			t.Fatalf("Create accepted %+v", s)
		}
	}
	if fake.Writes() != 0 {
		t.Fatalf("an unsafe name reached the host: %v", fake.CallLog())
	}
}

// TestRepositoryHalf pins the data the verb writes into the repository: the
// host configuration names the Worker and its route and carries no secret, and
// the deploy step reads its credential from the named secrets alone.
func TestRepositoryHalf(t *testing.T) {
	a := Adapter{}
	if a.Name() != "cloudflare" {
		t.Fatalf("name = %q", a.Name())
	}
	rel, data := a.HostConfig(site)
	if rel != "wrangler.jsonc" {
		t.Fatalf("host config path = %q", rel)
	}
	text := string(data)
	for _, want := range []string{`"name": "example-site"`, `"pattern": "docs.example.com"`, `"directory": "./site"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("host config lacks %s:\n%s", want, text)
		}
	}
	if _, bare := a.HostConfig(hosting.Site{Name: "example-site"}); strings.Contains(string(bare), `"routes":`) {
		t.Fatalf("a site with no domain declares routes:\n%s", bare)
	}
	step := a.DeployStep()
	for _, s := range a.Secrets() {
		if !strings.Contains(step, "${{ secrets."+s+" }}") {
			t.Fatalf("the deploy step does not read secret %s:\n%s", s, step)
		}
	}
	if strings.Contains(step, "run:") {
		t.Fatalf("the deploy step runs a shell; it should only call the pinned action:\n%s", step)
	}
}
