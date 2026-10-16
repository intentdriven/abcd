package session

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	coresession "github.com/intentdriven/abcd/internal/core/session"
)

// fakeManifestGitHub answers the manifest conversion as GitHub does, secrets and
// all, and the owner lookup.
func fakeManifestGitHub(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/app-manifests/code123/conversions":
			if r.Header.Get("Authorization") != "" {
				t.Errorf("the conversion sent credentials")
			}
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"id": 7, "slug": "acme-brief", "client_id": "Iv23liAbC", "client_secret": "SECRET-ONE",
				"webhook_secret": "SECRET-TWO", "pem": "SECRET-THREE", "owner": {"login": "acme", "type": "Organization"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/users/acme":
			io.WriteString(w, `{"login": "acme", "type": "Organization"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/users/octo":
			io.WriteString(w, `{"login": "octo", "type": "User"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

func newAppSetup(t *testing.T, api string) (*AppSetup, net.Listener) {
	t.Helper()
	m, err := coresession.NewAppManifest("acme/brief", "both", "")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return &AppSetup{
		Manifest: m,
		FormURL:  func(state string) string { return coresession.ManifestFormURL("acme", true, state) },
		API:      api,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}, ln
}

var (
	formActionRe = regexp.MustCompile(`<form method="post" action="([^"]+)"`)
	manifestRe   = regexp.MustCompile(`name="manifest" value="([^"]+)"`)
)

// TestAppSetupRunsTheManifestFlow: the page posts the pre-filled manifest to
// GitHub's new-app page, carrying a state; GitHub's redirect back, with that
// state and a code, converts the code, and only the public fields come back.
// Neither page shows a secret.
func TestAppSetupRunsTheManifestFlow(t *testing.T) {
	api, asked := fakeManifestGitHub(t)
	s, ln := newAppSetup(t, api.URL)
	base := "http://" + ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type outcome struct {
		app coresession.CreatedApp
		err error
	}
	done := make(chan outcome, 1)
	var opened string
	go func() {
		app, err := s.Run(ctx, ln, func(u string) { opened = u })
		done <- outcome{app, err}
	}()

	page := getPage(t, base+"/", http.StatusOK)
	m := formActionRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("no form on the page:\n%s", page)
	}
	action, err := url.Parse(html.UnescapeString(m[1]))
	if err != nil || action.Host != "github.com" || action.Path != "/organizations/acme/settings/apps/new" {
		t.Fatalf("the form posts to %q", m[1])
	}
	state := action.Query().Get("state")
	if len(state) < 32 {
		t.Fatalf("state %q is guessable", state)
	}
	mm := manifestRe.FindStringSubmatch(page)
	if mm == nil {
		t.Fatalf("no manifest on the page:\n%s", page)
	}
	var manifest coresession.AppManifest
	if err := json.Unmarshal([]byte(html.UnescapeString(mm[1])), &manifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if manifest.RedirectURL != base+"/created" || manifest.URL != "https://github.com/acme/brief" {
		t.Errorf("manifest = %+v", manifest)
	}

	// A wrong state is refused, and converts nothing.
	getPage(t, base+"/created?code=code123&state=wrong", http.StatusBadRequest)
	for _, a := range *asked {
		if strings.Contains(a, "conversions") {
			t.Fatal("a wrong state converted the code")
		}
	}
	done2 := getPage(t, base+"/created?code=code123&state="+url.QueryEscape(state), http.StatusOK)
	if strings.Contains(done2, "SECRET") || !strings.Contains(done2, "terminal") {
		t.Errorf("the done page:\n%s", done2)
	}
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.app.ClientID != "Iv23liAbC" || got.app.Slug != "acme-brief" || got.app.OwnerType != "Organization" {
		t.Errorf("app = %+v", got.app)
	}
	if opened != base+"/" {
		t.Errorf("opened %q", opened)
	}
	if dump, _ := json.Marshal(got.app); strings.Contains(string(dump), "SECRET") {
		t.Errorf("a secret came back: %s", dump)
	}
}

// TestAppSetupRefusesAnotherHost: the loopback page answers only its own
// address, as a host, so a page on another name cannot drive it.
func TestAppSetupRefusesAnotherHost(t *testing.T) {
	api, _ := fakeManifestGitHub(t)
	s, ln := newAppSetup(t, api.URL)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, ln, func(string) {})
		done <- err
	}()
	req, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/", nil)
	req.Host = "evil.example"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMisdirectedRequest {
		t.Errorf("another host got %d", resp.StatusCode)
	}
	cancel()
	if err := <-done; err == nil {
		t.Error("a cancelled setup reported success")
	}
}

func TestOwnerIsOrganisation(t *testing.T) {
	api, _ := fakeManifestGitHub(t)
	c := &http.Client{Timeout: 5 * time.Second}
	if org, err := OwnerIsOrganisation(context.Background(), c, api.URL, "acme"); err != nil || !org {
		t.Errorf("acme: %v %v", org, err)
	}
	if org, err := OwnerIsOrganisation(context.Background(), c, api.URL, "octo"); err != nil || org {
		t.Errorf("octo: %v %v", org, err)
	}
	if _, err := OwnerIsOrganisation(context.Background(), c, api.URL, "nobody"); err == nil {
		t.Error("an unknown owner answered")
	}
}

func getPage(t *testing.T, u string, want int) string {
	t.Helper()
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("GET %s = %d, want %d\n%s", u, resp.StatusCode, want, body)
	}
	return string(body)
}
