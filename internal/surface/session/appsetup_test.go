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
	m, err := coresession.NewAppManifest("acme/brief", "both", "", "https://brief-relay.onrender.com")
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type outcome struct {
		app coresession.CreatedApp
		err error
	}
	done := make(chan outcome, 1)
	openedc := make(chan string, 1)
	go func() {
		app, err := s.Run(ctx, ln, func(u string) { openedc <- u })
		done <- outcome{app, err}
	}()
	opened := <-openedc
	base := strings.TrimSuffix(opened, "/")
	if !strings.HasPrefix(base, "http://"+ln.Addr().String()+"/") {
		t.Fatalf("opened %q", opened)
	}

	page := getPage(t, opened, http.StatusOK)
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

// runSetup starts s.Run and hands back the address it opened and the outcome.
func runSetup(t *testing.T, s *AppSetup, ln net.Listener) (string, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	opened := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, ln, func(u string) { opened <- u })
		done <- err
	}()
	select {
	case u := <-opened:
		return u, done
	case <-time.After(5 * time.Second):
		t.Fatal("setup never opened its page")
	}
	return "", done
}

// TestAppSetupServesItsPageOnlyAtThePrintedAddress: the page, and with it the
// state, is served only under the unguessable path the printed address
// carries, so a client that was not given the address learns no state and
// cannot bring a code of its own back.
func TestAppSetupServesItsPageOnlyAtThePrintedAddress(t *testing.T) {
	api, asked := fakeManifestGitHub(t)
	s, ln := newAppSetup(t, api.URL)
	base := "http://" + ln.Addr().String()
	page, done := runSetup(t, s, ln)
	u, err := url.Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	if tok := strings.Trim(u.Path, "/"); len(tok) < 32 || strings.Contains(tok, "/") {
		t.Fatalf("the printed address %q carries no unguessable path", page)
	}
	getPage(t, base+"/", http.StatusNotFound)
	body := getPage(t, page, http.StatusOK)
	m := formActionRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no form at the printed address:\n%s", body)
	}
	action, _ := url.Parse(html.UnescapeString(m[1]))
	state := action.Query().Get("state")
	getPage(t, base+"/created?code=code123&state="+url.QueryEscape(state), http.StatusNotFound)
	for _, a := range *asked {
		if strings.Contains(a, "conversions") {
			t.Fatal("a callback outside the printed address converted a code")
		}
	}
	getPage(t, strings.TrimSuffix(page, "/")+"/created?code=code123&state="+url.QueryEscape(state), http.StatusOK)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// TestAppSetupKeepsTheManifestCodeOutOfItsError: a conversion that fails
// before GitHub answers leaves the code unspent for an hour, so the error
// abcd prints must not carry it.
func TestAppSetupKeepsTheManifestCodeOutOfItsError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("cannot hijack")
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	t.Cleanup(api.Close)
	s, ln := newAppSetup(t, api.URL)
	page, done := runSetup(t, s, ln)
	body := getPage(t, page, http.StatusOK)
	m := formActionRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no form:\n%s", body)
	}
	action, _ := url.Parse(html.UnescapeString(m[1]))
	back := strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state"))
	for i := 0; i < maxConversionTries; i++ {
		if resp, gerr := http.Get(back); gerr == nil {
			resp.Body.Close()
		}
	}
	err := <-done
	if err == nil {
		t.Fatal("a dropped conversion reported success")
	}
	if !strings.Contains(err.Error(), "asking GitHub for the new app") {
		t.Fatalf("the run ended on something other than the conversion: %v", err)
	}
	if strings.Contains(err.Error(), "code123") || strings.Contains(err.Error(), "app-manifests") {
		t.Errorf("the error carries the manifest code: %v", err)
	}
}

// TestAppSetupKeepsServingAfterAFailedConversion: a conversion that fails,
// for a passing fault, leaves the page serving so the browser can come back
// with the same code, and a failure never overrides a success.
func TestAppSetupKeepsServingAfterAFailedConversion(t *testing.T) {
	var calls int
	var mu sync.Mutex
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"slug": "acme-brief", "client_id": "Iv23liAbC", "client_secret": "SECRET-ONE", "owner": {"login": "acme", "type": "Organization"}}`)
	}))
	t.Cleanup(api.Close)
	s, ln := newAppSetup(t, api.URL)
	page, done := runSetup(t, s, ln)
	body := getPage(t, page, http.StatusOK)
	m := formActionRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no form:\n%s", body)
	}
	action, _ := url.Parse(html.UnescapeString(m[1]))
	back := strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state"))
	getPage(t, back, http.StatusBadGateway)
	select {
	case err := <-done:
		t.Fatalf("one failed conversion ended the setup: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	getPage(t, back, http.StatusOK)
	if err := <-done; err != nil {
		t.Fatalf("the retried conversion did not record the app: %v", err)
	}
}

// TestAppSetupConvertsNothingOnceSettled: once the run has given up, a later
// callback converts nothing, so GitHub never spends the code on an answer
// abcd no longer waits for.
func TestAppSetupConvertsNothingOnceSettled(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(api.Close)
	s, ln := newAppSetup(t, api.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opened := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, ln, func(u string) { opened <- u })
		done <- err
	}()
	page := <-opened
	body := getPage(t, page, http.StatusOK)
	action, _ := url.Parse(html.UnescapeString(formActionRe.FindStringSubmatch(body)[1]))
	back := strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state"))
	for i := 0; i < maxConversionTries; i++ {
		getPage(t, back, http.StatusBadGateway)
	}
	// The run has settled; the server may still answer while it shuts down.
	if resp, err := http.Get(back); err == nil {
		resp.Body.Close()
	}
	<-done
	mu.Lock()
	defer mu.Unlock()
	if calls != maxConversionTries {
		t.Errorf("GitHub was asked to convert %d times, want %d", calls, maxConversionTries)
	}
}

// TestAppSetupConvertsOnceForTwoCallbacks: two callbacks arriving together
// make one conversion, and the run records that one app.
func TestAppSetupConvertsOnceForTwoCallbacks(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"slug": "acme-brief", "client_id": "Iv23liAbC", "owner": {"login": "acme", "type": "Organization"}}`)
	}))
	t.Cleanup(api.Close)
	s, ln := newAppSetup(t, api.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opened := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, ln, func(u string) { opened <- u })
		done <- err
	}()
	page := <-opened
	body := getPage(t, page, http.StatusOK)
	action, _ := url.Parse(html.UnescapeString(formActionRe.FindStringSubmatch(body)[1]))
	back := strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state"))
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if resp, err := http.Get(back); err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("two callbacks made %d conversions, want 1", calls)
	}
}

// TestAppSetupConvertsWhenTheBrowserLetsGo: the conversion runs on the
// setup's own context, so a browser that drops the request while GitHub is
// answering does not cancel an exchange GitHub completes.
func TestAppSetupConvertsWhenTheBrowserLetsGo(t *testing.T) {
	release := make(chan struct{})
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"slug": "acme-brief", "client_id": "Iv23liAbC", "owner": {"login": "acme", "type": "Organization"}}`)
	}))
	t.Cleanup(api.Close)
	s, ln := newAppSetup(t, api.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opened := make(chan string, 1)
	type outcome struct {
		app coresession.CreatedApp
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		app, err := s.Run(ctx, ln, func(u string) { opened <- u })
		done <- outcome{app, err}
	}()
	page := <-opened
	body := getPage(t, page, http.StatusOK)
	action, _ := url.Parse(html.UnescapeString(formActionRe.FindStringSubmatch(body)[1]))
	back := strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state"))
	impatient := &http.Client{Timeout: 150 * time.Millisecond}
	if resp, err := impatient.Get(back); err == nil {
		resp.Body.Close()
		t.Fatal("the browser's request finished before GitHub answered")
	}
	close(release)
	got := <-done
	if got.err != nil || got.app.ClientID != "Iv23liAbC" {
		t.Fatalf("a dropped browser request lost the app: %+v %v", got.app, got.err)
	}
}
