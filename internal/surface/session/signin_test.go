package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// deviceCode is the device code the fake GitHub hands out: it must never
// reach a browser.
const deviceCode = "3584d83530557fdd1f46af8289938c8ef79f9dc5"

// clock is a settable time for the relay's device sign-in.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// deviceFixture is a relay set up for a local session: the GitHub App's
// client id and no secret, so it offers the device flow.
func deviceFixture(t *testing.T, mod func(*Config)) (*fixture, *clock) {
	t.Helper()
	clk := &clock{now: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	f := newFixture(t, func(c *Config) {
		c.ClientSecret = ""
		c.Now = clk.Now
		if mod != nil {
			mod(c)
		}
	})
	return f, clk
}

func decode(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not JSON: %v: %s", err, body)
	}
	return m
}

// TestVerifyTakesOnlyAGitHubAppUserToken: nobody pastes a token. The relay
// takes only a GitHub App user access token, whose prefix GitHub documents as
// ghu_, and refuses any other before GitHub is asked (adr-2610161400542916).
func TestVerifyTakesOnlyAGitHubAppUserToken(t *testing.T) {
	gh := &fakeGitHub{}
	ghSrv := httptest.NewServer(gh.handler())
	t.Cleanup(ghSrv.Close)
	srv := newFixture(t, nil)
	v := GitHub{API: ghSrv.URL, Client: ghSrv.Client()}
	for _, token := range []string{"ghp_personal", "github_pat_11AAAA", "gho_oauthapp", "ghs_installation", "ada"} {
		if _, _, err := v.Verify(context.Background(), token, "acme/brief"); err == nil {
			t.Errorf("%s was accepted", token)
		}
	}
	gh.mu.Lock()
	asked := len(gh.users)
	gh.mu.Unlock()
	if asked != 0 {
		t.Fatalf("GitHub was asked about %d tokens that are not a GitHub App's", asked)
	}
	login, push, err := v.Verify(context.Background(), "ghu_ada", "acme/brief")
	if err != nil || login != "ada" || !push {
		t.Fatalf("a GitHub App user token: %q %v %v", login, push, err)
	}

	// Over the wire, too: a pasted personal access token is refused.
	c := srv.dial(t, origin)
	c.sendJSON(map[string]string{"type": "verify", "token": "ghp_personal", "repo": "acme/brief"})
	if e, _ := c.until("error"); e["code"] != "verify-failed" {
		t.Fatalf("verify with a personal access token: %v, want verify-failed", e)
	}
}

// TestConfigNamesTheFlow: a relay with the app's secret offers the web flow,
// one with the client id alone the device flow, and one with neither nothing.
func TestConfigNamesTheFlow(t *testing.T) {
	get := func(f *fixture) map[string]any {
		resp, err := http.Get(f.http.URL + "/config")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return decode(t, string(data))
	}
	if m := get(newFixture(t, nil)); m["flow"] != "web" || m["sign_in"] != true {
		t.Errorf("with a secret: %v, want the web flow", m)
	}
	f, _ := deviceFixture(t, nil)
	if m := get(f); m["flow"] != "device" || m["sign_in"] != true || m["app_type"] != "github-app" {
		t.Errorf("with a client id alone: %v, want the device flow", m)
	}
	if m := get(newFixture(t, func(c *Config) { c.ClientID, c.ClientSecret = "", "" })); m["sign_in"] != false {
		t.Errorf("with neither: %v, want no sign-in", m)
	}
}

// TestDeviceSignInKeepsTheDeviceCodeOnTheRelay walks a device sign-in: the
// relay asks GitHub for a device code and keeps it, the page gets a handle
// and the user code, and the relay polls GitHub no faster than its interval,
// slowing on slow_down, until the person confirms.
func TestDeviceSignInKeepsTheDeviceCodeOnTheRelay(t *testing.T) {
	f, clk := deviceFixture(t, nil)
	f.github.deviceAnswers = []map[string]any{
		{"error": "authorization_pending"},
		{"error": "slow_down", "interval": 10},
		{"access_token": "ghu_ada", "token_type": "bearer", "scope": ""},
	}
	resp, body := post(t, f.http.URL+"/auth/device", origin, `{}`)
	if resp.StatusCode != http.StatusOK || strings.Contains(body, deviceCode) {
		t.Fatalf("start: %d %s, want a handle and never the device code", resp.StatusCode, body)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("CORS origin = %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
	m := decode(t, body)
	handle, _ := m["handle"].(string)
	if len(handle) != 22 || m["user_code"] != "WDJB-MJHT" || m["verification_uri"] != f.srv.cfg.GitHubWeb+"/login/device" {
		t.Fatalf("start = %v", m)
	}
	f.github.mu.Lock()
	start := f.github.deviceStarts[0]
	f.github.mu.Unlock()
	if start.Get("client_id") != "Iv1.testclientid" || start.Has("client_secret") {
		t.Fatalf("GitHub received %v, want the client id and no secret", start)
	}

	poll := func() map[string]any {
		t.Helper()
		_, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"`+handle+`"}`)
		return decode(t, body)
	}
	polls := func() int {
		f.github.mu.Lock()
		defer f.github.mu.Unlock()
		return len(f.github.devicePolls)
	}
	// Too soon: the relay answers pending without asking GitHub.
	if got := poll(); got["error"] != "authorization_pending" || polls() != 0 {
		t.Fatalf("an early poll: %v, %d asked; want pending and GitHub not asked", got, polls())
	}
	clk.add(5 * time.Second)
	if got := poll(); got["error"] != "authorization_pending" || polls() != 1 {
		t.Fatalf("first poll: %v (%d asked)", got, polls())
	}
	f.github.mu.Lock()
	sent := f.github.devicePolls[0]
	f.github.mu.Unlock()
	if sent.Get("device_code") != deviceCode || sent.Get("client_id") != "Iv1.testclientid" || sent.Has("client_secret") {
		t.Fatalf("the poll sent %v", sent)
	}
	clk.add(5 * time.Second)
	if got := poll(); got["error"] != "slow_down" || polls() != 2 {
		t.Fatalf("second poll: %v (%d asked)", got, polls())
	}
	// Slowed to GitHub's new interval of ten seconds.
	clk.add(5 * time.Second)
	if got := poll(); got["error"] != "authorization_pending" || polls() != 2 {
		t.Fatalf("a poll inside the slowed interval: %v (%d asked)", got, polls())
	}
	clk.add(5 * time.Second)
	if got := poll(); got["access_token"] != "ghu_ada" || polls() != 3 {
		t.Fatalf("the confirmed poll: %v (%d asked)", got, polls())
	}
	// Done: the handle is gone.
	clk.add(10 * time.Second)
	if got := poll(); got["error"] != "expired_token" || polls() != 3 {
		t.Fatalf("a poll after the token: %v (%d asked)", got, polls())
	}
}

// TestDeviceSignInRefusals: the device routes check the page's origin, are
// offered only by a relay without a secret, count against the per-address
// exchange limit, refuse an unknown handle, and drop a sign-in GitHub ends.
func TestDeviceSignInRefusals(t *testing.T) {
	f, clk := deviceFixture(t, nil)
	if resp, body := post(t, f.http.URL+"/auth/device", "https://evil.example", `{}`); resp.StatusCode != http.StatusForbidden || strings.Contains(body, "handle") {
		t.Errorf("a foreign origin: %d %s", resp.StatusCode, body)
	}
	if resp, body := post(t, f.http.URL+"/auth/device/token", "https://evil.example", `{"handle":"AAAAAAAAAAAAAAAAAAAAAA"}`); resp.StatusCode != http.StatusForbidden {
		t.Errorf("a foreign origin's poll: %d %s", resp.StatusCode, body)
	}
	if resp, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"AAAAAAAAAAAAAAAAAAAAAA"}`); resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "expired_token") {
		t.Errorf("an unknown handle: %d %s", resp.StatusCode, body)
	}
	if resp, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"../x"}`); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("a malformed handle: %d %s", resp.StatusCode, body)
	}
	web := newFixture(t, nil)
	if resp, body := post(t, web.http.URL+"/auth/device", origin, `{}`); resp.StatusCode != http.StatusServiceUnavailable || len(web.github.deviceStarts) != 0 {
		t.Errorf("a relay with a secret: %d %s, want the device flow not offered", resp.StatusCode, body)
	}

	// GitHub ending the sign-in drops it.
	f.github.deviceAnswers = []map[string]any{{"error": "access_denied"}}
	_, body := post(t, f.http.URL+"/auth/device", origin, `{}`)
	handle, _ := decode(t, body)["handle"].(string)
	clk.add(5 * time.Second)
	if resp, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"`+handle+`"}`); resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "access_denied") {
		t.Errorf("a denied sign-in: %d %s", resp.StatusCode, body)
	}
	if _, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"`+handle+`"}`); !strings.Contains(body, "expired_token") {
		t.Errorf("the denied sign-in was kept: %s", body)
	}

	// A device code that expires is dropped without asking GitHub.
	f.github.deviceAnswers = []map[string]any{{"error": "authorization_pending"}}
	_, body = post(t, f.http.URL+"/auth/device", origin, `{}`)
	handle, _ = decode(t, body)["handle"].(string)
	clk.add(16 * time.Minute)
	f.github.mu.Lock()
	before := len(f.github.devicePolls)
	f.github.mu.Unlock()
	if _, body := post(t, f.http.URL+"/auth/device/token", origin, `{"handle":"`+handle+`"}`); !strings.Contains(body, "expired_token") {
		t.Errorf("an expired device code: %s", body)
	}
	f.github.mu.Lock()
	after := len(f.github.devicePolls)
	f.github.mu.Unlock()
	if after != before {
		t.Error("GitHub was asked about an expired device code")
	}

	// Starting a device sign-in counts against the exchange limit, ten in
	// ten minutes per address: the starts above are past the window now.
	for i := 0; i < 10; i++ {
		if resp, body := post(t, f.http.URL+"/auth/device", origin, `{}`); resp.StatusCode != http.StatusOK {
			t.Fatalf("start %d: %d %s", i+1, resp.StatusCode, body)
		}
	}
	if resp, body := post(t, f.http.URL+"/auth/device", origin, `{}`); resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Errorf("the eleventh start: %d %s, want 429", resp.StatusCode, body)
	}
}

// TestDeviceSignInsAreBounded: at most maxDeviceLogins wait at once, from
// however many addresses.
func TestDeviceSignInsAreBounded(t *testing.T) {
	f, _ := deviceFixture(t, func(c *Config) { c.BehindProxy = true })
	start := func(addr string) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, f.http.URL+"/auth/device", strings.NewReader(`{}`))
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Forwarded-For", addr)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	for i := 0; i < maxDeviceLogins; i++ {
		if resp := start("198.51.100." + itoa(i%200+1)); resp.StatusCode != http.StatusOK {
			t.Fatalf("start %d: %d", i+1, resp.StatusCode)
		}
	}
	if resp := start("203.0.113.9"); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("one past the bound: %d, want 503", resp.StatusCode)
	}
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
