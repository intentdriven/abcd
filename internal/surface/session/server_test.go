package session

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	core "github.com/intentdriven/abcd/internal/core/session"
)

// testSecret is the sign-in app secret the tests configure. It is a test
// value, never a real one.
const testSecret = "test-client-secret-0123456789abcdef"

// fakeGitHub is GitHub's API and sign-in endpoint as the relay sees them.
type fakeGitHub struct {
	mu        sync.Mutex
	exchanges []url.Values
}

func (g *fakeGitHub) handler() http.Handler {
	logins := map[string]string{"tok-ada": "ada", "tok-bea": "bea", "tok-cy": "cy"}
	push := map[string]bool{"ada": true, "bea": true}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		login, ok := logins[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
		if !ok {
			http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"login": login})
	})
	mux.HandleFunc("GET /repos/{owner}/{name}", func(w http.ResponseWriter, r *http.Request) {
		login := logins[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
		if r.PathValue("owner") != "acme" || r.PathValue("name") != "brief" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"full_name": "acme/brief", "permissions": map[string]bool{"pull": true, "push": push[login]}})
	})
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		g.mu.Lock()
		g.exchanges = append(g.exchanges, r.PostForm)
		g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("code") != "good-code" {
			// A hostile or careless upstream echoing the secret back.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "bad_verification_code",
				"error_description": "the code is wrong; client_secret=" + r.PostForm.Get("client_secret"),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gho_session_token", "token_type": "bearer", "scope": ""})
	})
	return mux
}

type fixture struct {
	srv    *Server
	http   *httptest.Server
	github *fakeGitHub
	wsURL  string
}

func newFixture(t *testing.T, mod func(*Config)) *fixture {
	t.Helper()
	gh := &fakeGitHub{}
	ghSrv := httptest.NewServer(gh.handler())
	t.Cleanup(ghSrv.Close)
	cfg := Config{
		ClientID:     "Iv1.testclientid",
		ClientSecret: testSecret,
		GitHubAPI:    ghSrv.URL,
		GitHubWeb:    ghSrv.URL,
		HTTPClient:   ghSrv.Client(),
		Repos:        []string{"acme/brief"},
	}
	if mod != nil {
		mod(&cfg)
	}
	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	hs := httptest.NewServer(srv)
	t.Cleanup(hs.Close)
	return &fixture{srv: srv, http: hs, github: gh, wsURL: "ws" + strings.TrimPrefix(hs.URL, "http") + "/session"}
}

// client is one browser's connection as a test drives it.
type client struct {
	t  *testing.T
	ws *websocket.Conn
}

func (f *fixture) dial(t *testing.T, origin string) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {origin}}})
	if err != nil {
		t.Fatalf("dial from %q: %v", origin, err)
	}
	t.Cleanup(func() { _ = ws.CloseNow() })
	return &client{t: t, ws: ws}
}

// dialFrom dials as Tailscale Serve would proxy a visitor: with the visitor's
// tailnet address in X-Forwarded-For.
func (f *fixture) dialFrom(t *testing.T, origin, forwarded string) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h := http.Header{"Origin": {origin}, "X-Forwarded-For": {forwarded}}
	ws, _, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: h})
	if err != nil {
		t.Fatalf("dial from %q as %s: %v", origin, forwarded, err)
	}
	t.Cleanup(func() { _ = ws.CloseNow() })
	return &client{t: t, ws: ws}
}

func (c *client) sendJSON(v any) {
	c.t.Helper()
	data, _ := json.Marshal(v)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

func (c *client) sendBinary(b []byte) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageBinary, b); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

// next reads one message; text messages decode to a map, binary come back raw.
func (c *client) next() (map[string]any, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	typ, data, err := c.ws.Read(ctx)
	if err != nil {
		return nil, nil, err
	}
	if typ == websocket.MessageBinary {
		return nil, data, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		c.t.Fatalf("text frame is not JSON: %q", data)
	}
	return m, nil, nil
}

// until reads until a text message of type typ arrives, returning it and the
// binary frames read on the way.
func (c *client) until(typ string) (map[string]any, [][]byte) {
	c.t.Helper()
	var bins [][]byte
	for {
		m, b, err := c.next()
		if err != nil {
			c.t.Fatalf("waiting for %s: %v", typ, err)
		}
		if b != nil {
			bins = append(bins, b)
			continue
		}
		if m["type"] == typ {
			return m, bins
		}
	}
}

const origin = "https://abcdesign.app"

// hostNumber is the host's participant number from its hosted answer: the
// last byte of the four an attributed update carries, in a test's small
// relay.
func hostNumber(t *testing.T, hosted map[string]any) byte {
	t.Helper()
	id, _ := hosted["id"].(string)
	if len(id) != 2 || id[0] != 'p' || id[1] < '1' || id[1] > '9' {
		t.Fatalf("hosted id %q is not a small participant id", id)
	}
	return id[1] - '0'
}

func TestHostAndGuestOverTheWire(t *testing.T) {
	f := newFixture(t, nil)
	host := f.dial(t, origin)
	host.sendJSON(map[string]string{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	if v, _ := host.until("verified"); v["member"] != true {
		t.Fatalf("verified = %v", v)
	}
	host.sendJSON(map[string]string{"type": "host", "repo": "acme/brief"})
	hosted, _ := host.until("hosted")
	code := hosted["code"].(string)
	host.sendBinary([]byte{core.TagUpdate, 1, 2, 3})

	guest := f.dial(t, "http://localhost:5173")
	guest.sendJSON(map[string]string{"type": "join", "code": code, "name": "Gus"})
	if j, _ := guest.until("joined"); j["role"] != "guest" {
		t.Fatalf("joined = %v", j)
	}
	_, replay := guest.until("synced")
	hostNum := hostNumber(t, hosted)
	if len(replay) != 1 || !bytes.Equal(replay[0], append([]byte{core.TagAttributedUpdate, 0, 0, 0, hostNum}, 1, 2, 3)) {
		t.Fatalf("replay = %v, want the host's update", replay)
	}

	guest.sendBinary([]byte{core.TagUpdate, 9})
	if e, _ := guest.until("error"); e["code"] != "read-only" {
		t.Fatalf("a guest's update: %v, want read-only", e)
	}
	host.sendBinary([]byte{core.TagUpdate, 4})
	for {
		_, b, err := guest.next()
		if err != nil {
			t.Fatal(err)
		}
		if b != nil {
			if !bytes.Equal(b, []byte{core.TagAttributedUpdate, 0, 0, 0, hostNum, 4}) {
				t.Fatalf("the guest received %v, want the host's second update", b)
			}
			break
		}
	}

	guest.sendJSON(map[string]any{"type": "suggest", "body": map[string]string{"text": "x"}})
	if s, _ := host.until("suggest"); s["from"] != "Gus" {
		t.Fatalf("suggest = %v", s)
	}

	host.sendJSON(map[string]string{"type": "close"})
	guest.until("closed")
	if _, _, err := guest.next(); websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("after close the guest read %v, want a normal closure", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for f.srv.Relay().Stats() != (core.Stats{}) {
		if time.Now().After(deadline) {
			t.Fatalf("the relay still holds %+v", f.srv.Relay().Stats())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestDisallowedOriginIsRefused: by default the relay admits
// https://abcdesign.app and loopback pages; given origins replace that list,
// so then only they are admitted.
func TestDisallowedOriginIsRefused(t *testing.T) {
	refused := func(f *fixture, o string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h := http.Header{}
		if o != "" {
			h.Set("Origin", o)
		}
		_, resp, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: h})
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %q: err=%v resp=%v, want 403", o, err, resp)
		}
	}
	def := newFixture(t, nil)
	for _, o := range []string{"https://abcdesign.app", "http://127.0.0.1:8000", "http://[::1]:3000", "http://localhost"} {
		def.dial(t, o)
	}
	for _, o := range []string{"https://brief.example.org", "https://evil.example", "", "null"} {
		refused(def, o)
	}
	f := newFixture(t, func(c *Config) { c.Origins = []string{"https://brief.example.org", "http://localhost:5173"} })
	for _, o := range []string{"https://abcdesign.app", "http://127.0.0.1:8000", "http://localhost", "http://localhost:5174"} {
		refused(f, o)
	}
	for _, o := range []string{"https://brief.example.org", "http://localhost:5173"} {
		f.dial(t, o)
	}
	for _, o := range []string{"https://evil.example", "", "null", "https://abcdesign.app.evil.example", "http://abcdesign.app"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		h := http.Header{}
		if o != "" {
			h.Set("Origin", o)
		}
		_, resp, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: h})
		cancel()
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %q: err=%v resp=%v, want 403", o, err, resp)
		}
	}
}

func TestOversizedFrameEndsTheConnection(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.Limits.MaxMessageBytes = 1024 })
	c := f.dial(t, origin)
	c.sendBinary(make([]byte, 2048))
	_, _, err := c.next()
	if websocket.CloseStatus(err) != websocket.StatusMessageTooBig {
		t.Fatalf("read after an oversized frame: %v, want close 1009", err)
	}
}

func post(t *testing.T, url, origin, body string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func TestAuthTokenExchangesWithoutEverEchoingTheSecret(t *testing.T) {
	f := newFixture(t, nil)
	url := f.http.URL + "/auth/token"

	resp, body := post(t, url, origin, `{"code":"good-code","code_verifier":"`+verifier+`","redirect_uri":"https://abcdesign.app/acme/brief/brief/"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"access_token":"gho_session_token"`) {
		t.Fatalf("exchange: %d %s", resp.StatusCode, body)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("CORS origin = %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
	f.github.mu.Lock()
	got := f.github.exchanges[0]
	f.github.mu.Unlock()
	if got.Get("client_secret") != testSecret || got.Get("client_id") != "Iv1.testclientid" || got.Get("code_verifier") != verifier {
		t.Fatalf("GitHub received %v, want the secret added and the verifier passed through", got)
	}

	// GitHub's error description echoing the secret never reaches the browser.
	resp, body = post(t, url, origin, `{"code":"bad-code","code_verifier":"`+verifier+`"}`)
	if resp.StatusCode != http.StatusBadRequest || strings.Contains(body, testSecret) || strings.Contains(body, "client_secret") {
		t.Fatalf("failed exchange: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "bad_verification_code") {
		t.Fatalf("failed exchange body %s, want GitHub's error code", body)
	}

	for _, c := range []struct{ origin, body string }{
		{"https://evil.example", `{"code":"good-code","code_verifier":"` + verifier + `"}`},
		{"", `{"code":"good-code","code_verifier":"` + verifier + `"}`},
		{origin, `{"code":"good-code","code_verifier":"short"}`},
		{origin, `{"code":"good-code","code_verifier":"` + verifier + `","redirect_uri":"https://evil.example/cb"}`},
		{origin, `not json`},
	} {
		resp, body := post(t, url, c.origin, c.body)
		if resp.StatusCode < 400 || strings.Contains(body, testSecret) || strings.Contains(body, "gho_") {
			t.Errorf("origin %q body %s: %d %s, want a refusal", c.origin, c.body, resp.StatusCode, body)
		}
	}

	// The configuration names the public client id and never the secret.
	cfgResp, err := http.Get(f.http.URL + "/config")
	if err != nil {
		t.Fatal(err)
	}
	cfgBody, _ := io.ReadAll(cfgResp.Body)
	cfgResp.Body.Close()
	if !strings.Contains(string(cfgBody), "Iv1.testclientid") || strings.Contains(string(cfgBody), testSecret) {
		t.Fatalf("/config = %s", cfgBody)
	}
}

func TestAuthTokenWithoutASecretIsNotConfigured(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.ClientSecret = "" })
	resp, body := post(t, f.http.URL+"/auth/token", origin, `{"code":"good-code","code_verifier":"`+verifier+`"}`)
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "sign-in-not-configured") {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if n := len(f.github.exchanges); n != 0 {
		t.Fatalf("GitHub was asked %d times", n)
	}
}

func TestAuthTokenPreflight(t *testing.T) {
	f := newFixture(t, nil)
	req, _ := http.NewRequest(http.MethodOptions, f.http.URL+"/auth/token", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != origin ||
		!strings.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("preflight: %d %v", resp.StatusCode, resp.Header)
	}
}

func TestHealthz(t *testing.T) {
	f := newFixture(t, nil)
	resp, err := http.Get(f.http.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d", resp.StatusCode)
	}
	resp, err = http.Get(f.http.URL + "/anything-else")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("an unknown path = %d, want 404", resp.StatusCode)
	}
}

func TestGitHubVerifier(t *testing.T) {
	gh := httptest.NewServer((&fakeGitHub{}).handler())
	defer gh.Close()
	v := GitHub{API: gh.URL, Client: gh.Client()}
	ctx := context.Background()
	for _, c := range []struct {
		token, repo, login string
		push, err          bool
	}{
		{"tok-ada", "acme/brief", "ada", true, false},
		{"tok-cy", "acme/brief", "cy", false, false},
		{"tok-ada", "acme/other", "ada", false, false}, // GitHub's 404: no access
		{"tok-nobody", "acme/brief", "", false, true},
	} {
		login, push, err := v.Verify(ctx, c.token, c.repo)
		if login != c.login || push != c.push || (err != nil) != c.err {
			t.Errorf("%s on %s: login=%q push=%v err=%v", c.token, c.repo, login, push, err)
		}
		if err != nil && strings.Contains(err.Error(), c.token) {
			t.Errorf("the error carries the token: %v", err)
		}
	}
	if _, _, err := v.Verify(ctx, "tok-ada", "../../user"); err == nil {
		t.Error("a repository that is not owner/name reached a request")
	}
}

func TestClientAddressTrustsForwardedOnlyBehindAProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/session", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 198.51.100.4")
	if got := clientAddress(r, false, false); got != "10.0.0.1" {
		t.Errorf("not behind a proxy: %q, want the socket's address", got)
	}
	if got := clientAddress(r, true, false); got != "198.51.100.4" {
		t.Errorf("behind a proxy: %q, want the entry the proxy added", got)
	}
	r.Header.Del("X-Forwarded-For")
	if got := clientAddress(r, true, false); got != "10.0.0.1" {
		t.Errorf("behind a proxy with no header: %q, want the socket's address", got)
	}
}

// TestClientAddressOnTheTailnetRoute: on the host's own machine the relay
// listens on loopback and Tailscale Serve proxies to it, so every visitor's
// socket address is loopback. Only then is the address Serve forwards the one
// counted; a header from anywhere else, or one Serve would not send, is not.
func TestClientAddressOnTheTailnetRoute(t *testing.T) {
	at := func(remote string, xff ...string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/session", nil)
		r.RemoteAddr = remote
		for _, v := range xff {
			r.Header.Add("X-Forwarded-For", v)
		}
		return r
	}
	for _, c := range []struct {
		name     string
		r        *http.Request
		loopback bool
		want     string
	}{
		{"tailnet IPv4 through Serve", at("127.0.0.1:5555", "100.101.102.103"), true, "100.101.102.103"},
		{"tailnet IPv6 through Serve", at("[::1]:5555", "fd7a:115c:a1e0::a1"), true, "fd7a:115c:a1e0::a1"},
		{"a forged header from a non-loopback peer", at("192.0.2.7:5555", "100.64.0.1"), true, "192.0.2.7"},
		{"a relay not bound to loopback alone", at("127.0.0.1:5555", "100.64.0.1"), false, "127.0.0.1"},
		{"no header", at("127.0.0.1:5555"), true, "127.0.0.1"},
		{"an empty header", at("127.0.0.1:5555", ""), true, "127.0.0.1"},
		{"not an address", at("127.0.0.1:5555", "not-an-ip"), true, "127.0.0.1"},
		{"an address with a port", at("127.0.0.1:5555", "100.64.0.1:443"), true, "127.0.0.1"},
		{"a list Serve never sends", at("127.0.0.1:5555", "100.64.0.1, 100.64.0.2"), true, "127.0.0.1"},
		{"two header lines", at("127.0.0.1:5555", "100.64.0.1", "100.64.0.2"), true, "127.0.0.1"},
		{"an address outside the tailnet", at("127.0.0.1:5555", "203.0.113.9"), true, "127.0.0.1"},
	} {
		if got := clientAddress(c.r, false, c.loopback); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// TestTailnetVisitorsAreLimitedSeparately: one tailnet visitor's wrong codes
// shut out that visitor alone, not everyone Serve proxies.
func TestTailnetVisitorsAreLimitedSeparately(t *testing.T) {
	f := newFixture(t, func(c *Config) {
		c.LoopbackOnly = true
		c.Limits.MaxFailedJoins = 2
	})
	host := f.dial(t, origin)
	host.sendJSON(map[string]string{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	host.until("verified")
	host.sendJSON(map[string]string{"type": "host", "repo": "acme/brief"})
	hosted, _ := host.until("hosted")
	code := hosted["code"].(string)

	for _, pair := range [][2]string{{"100.101.102.103", "100.64.0.9"}, {"fd7a:115c:a1e0::a1", "fd7a:115c:a1e0::b2"}} {
		mal := f.dialFrom(t, origin, pair[0])
		for i := 0; i < 2; i++ {
			mal.sendJSON(map[string]string{"type": "join", "code": "ZZZZZZ", "name": "Mal"})
			if e, _ := mal.until("error"); e["code"] != "bad-code" {
				t.Fatalf("%s try %d: %v, want bad-code", pair[0], i, e)
			}
		}
		again := f.dialFrom(t, origin, pair[0])
		again.sendJSON(map[string]string{"type": "join", "code": code, "name": "Mal"})
		if e, _ := again.until("error"); e["code"] != "too-many-tries" {
			t.Fatalf("%s past its limit: %v, want too-many-tries", pair[0], e)
		}
		other := f.dialFrom(t, origin, pair[1])
		other.sendJSON(map[string]string{"type": "join", "code": code, "name": "Gus"})
		if j, _ := other.until("joined"); j["role"] != "guest" {
			t.Fatalf("%s was held up by %s's wrong codes: %v", pair[1], pair[0], j)
		}
	}
}

func TestOriginsAreParsedStrictly(t *testing.T) {
	for _, o := range []string{"https://ok.example", "http://ok.example:8080"} {
		if _, err := New(Config{Origins: []string{o}, LoopbackOnly: true}); err != nil {
			t.Errorf("%s refused: %v", o, err)
		}
	}
	for _, o := range []string{"*", "ok.example", "https://ok.example/path", "ftp://ok.example", "https://*.example"} {
		if _, err := New(Config{Origins: []string{o}, LoopbackOnly: true}); err == nil {
			t.Errorf("%s accepted", o)
		}
	}
}

// hostSession verifies a fresh connection as ada and opens a session,
// returning the connection and its join code.
func (f *fixture) hostSession(t *testing.T) (*client, string) {
	t.Helper()
	host := f.dial(t, origin)
	host.sendJSON(map[string]string{"type": "verify", "token": "tok-ada", "repo": "acme/brief"})
	host.until("verified")
	host.sendJSON(map[string]string{"type": "host", "repo": "acme/brief"})
	hosted, _ := host.until("hosted")
	return host, hosted["code"].(string)
}

// TestReplayLongerThanTheSendQueue: a log longer than a connection's send
// queue reaches a newcomer whole, followed by synced, and does not drop it.
func TestReplayLongerThanTheSendQueue(t *testing.T) {
	f := newFixture(t, func(c *Config) {
		c.Limits.UpdatesPerSecond = 1 << 20
		c.Limits.UpdateBurst = 1 << 20
	})
	host, _ := f.hostSession(t)
	n := sendQueue + 300
	for i := 0; i < n; i++ {
		host.sendBinary([]byte{core.TagUpdate, byte(i), byte(i >> 8)})
	}
	// The rotation's answer comes after every update was handled.
	host.sendJSON(map[string]string{"type": "rotate"})
	got, _ := host.until("code")
	code := got["code"].(string)

	guest := f.dial(t, origin)
	guest.sendJSON(map[string]string{"type": "join", "code": code, "name": "Gus"})
	guest.until("joined")
	_, replay := guest.until("synced")
	if len(replay) != n {
		t.Fatalf("the newcomer received %d updates before synced, want all %d", len(replay), n)
	}
	for i, b := range replay {
		body := b[len(b)-2:]
		if int(body[0])|int(body[1])<<8 != i&0xffff {
			t.Fatalf("update %d arrived out of order: %v", i, b)
		}
	}
}

// dialStatus tries the WebSocket upgrade and returns the HTTP status of a
// refusal, or 0 when the upgrade succeeded.
func (f *fixture) dialStatus(t *testing.T) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, resp, err := websocket.Dial(ctx, f.wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {origin}}})
	if err == nil {
		t.Cleanup(func() { _ = ws.CloseNow() })
		return 0
	}
	if resp == nil {
		t.Fatalf("dial: %v", err)
	}
	return resp.StatusCode
}

// TestConnectionCapsRefuseTheUpgrade: past the relay-wide cap or the
// per-address cap, the upgrade is refused with 503, and the connections
// already held carry on.
func TestConnectionCapsRefuseTheUpgrade(t *testing.T) {
	for _, c := range []struct {
		name string
		mod  func(*Config)
	}{
		{"relay-wide", func(c *Config) { c.Limits.MaxConnections = 2 }},
		{"per address", func(c *Config) { c.Limits.MaxConnsPerAddr = 2 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, c.mod)
			host, code := f.hostSession(t)
			guest := f.dial(t, origin)
			if got := f.dialStatus(t); got != http.StatusServiceUnavailable {
				t.Fatalf("a third connection: status %d, want 503", got)
			}
			guest.sendJSON(map[string]string{"type": "join", "code": code, "name": "Gus"})
			guest.until("joined")
			host.sendJSON(map[string]string{"type": "rotate"})
			host.until("code")
			if s := f.srv.Relay().Stats(); s.Connections != 2 {
				t.Fatalf("connections = %d, want the two held", s.Connections)
			}
		})
	}
}

// TestAConnectionThatJoinsNothingIsClosed: a connection that has not hosted,
// joined or rejoined within the grace period is closed normally; one that
// has stays open past it.
func TestAConnectionThatJoinsNothingIsClosed(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.Limits.AdmitWithin = 200 * time.Millisecond })
	host, _ := f.hostSession(t)
	idle := f.dial(t, origin)
	if e, _ := idle.until("error"); e["code"] != "join-timeout" {
		t.Fatalf("idle connection: %v, want join-timeout", e)
	}
	if _, _, err := idle.next(); websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("idle connection ended with %v, want a normal closure", err)
	}
	time.Sleep(300 * time.Millisecond)
	host.sendJSON(map[string]string{"type": "rotate"})
	host.until("code")
}

// TestTheSendQueueGrowsOnEnteringASession: a connection outside a session
// may have few frames waiting; one inside, many.
func TestTheSendQueueGrowsOnEnteringASession(t *testing.T) {
	p := newPeer(func() {}, preRoomQueue)
	for i := 0; i < preRoomQueue; i++ {
		if !p.Send(core.Frame{Data: []byte("{}")}) {
			t.Fatalf("frame %d refused below the pre-session bound", i)
		}
	}
	if p.Send(core.Frame{Data: []byte("{}")}) {
		t.Fatalf("frame %d accepted past the pre-session bound", preRoomQueue)
	}
	p.Entered()
	for i := preRoomQueue; i < sendQueue; i++ {
		if !p.Send(core.Frame{Data: []byte("{}")}) {
			t.Fatalf("frame %d refused below the session bound", i)
		}
	}
	if p.Send(core.Frame{Data: []byte("{}")}) {
		t.Fatalf("frame %d accepted past the session bound", sendQueue)
	}
	if preRoomQueue > 64 {
		t.Fatalf("preRoomQueue = %d, want it small", preRoomQueue)
	}
}

// TestARelayBeyondLoopbackNeedsItsRepositories: a relay not bound to
// loopback alone serves only the repositories it is given, and refuses to
// start without any; a malformed entry is refused too.
func TestARelayBeyondLoopbackNeedsItsRepositories(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("a relay beyond loopback started with no repositories")
	}
	if _, err := New(Config{LoopbackOnly: true}); err != nil {
		t.Fatalf("a loopback relay with no list: %v", err)
	}
	for _, c := range []Config{
		{Repos: []string{"acme/brief"}},
		{Owners: []string{"acme"}},
	} {
		if _, err := New(c); err != nil {
			t.Errorf("%+v refused: %v", c, err)
		}
	}
	for _, c := range []Config{
		{Repos: []string{"acme"}},
		{Repos: []string{"acme/brief/x"}},
		{Owners: []string{"acme/brief"}},
		{Owners: []string{"-acme"}},
		{Repos: []string{"acme/brief"}, Owners: []string{""}},
	} {
		if _, err := New(c); err == nil {
			t.Errorf("%+v accepted", c)
		}
	}
}

// TestVerifyAgainstAnUnservedRepositoryIsRefused: over the wire, a token
// presented for a repository the relay does not serve is refused before
// GitHub is asked.
func TestVerifyAgainstAnUnservedRepositoryIsRefused(t *testing.T) {
	f := newFixture(t, nil)
	c := f.dial(t, origin)
	c.sendJSON(map[string]string{"type": "verify", "token": "tok-ada", "repo": "mal/own"})
	if e, _ := c.until("error"); e["code"] != "repo-not-served" {
		t.Fatalf("verify: %v, want repo-not-served", e)
	}
	c.sendJSON(map[string]string{"type": "host", "repo": "mal/own"})
	if e, _ := c.until("error"); e["code"] != "repo-not-served" {
		t.Fatalf("host: %v, want repo-not-served", e)
	}
}

// TestConfigNamesTheSignInAppType: /config says whether the sign-in app is
// a GitHub App (the default) or an OAuth app, and its authorize URL carries
// no scope, which a GitHub App does not use; an unknown type is refused.
func TestConfigNamesTheSignInAppType(t *testing.T) {
	get := func(f *fixture) map[string]any {
		resp, err := http.Get(f.http.URL + "/config")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	m := get(newFixture(t, nil))
	if m["app_type"] != "github-app" {
		t.Fatalf("default app_type = %v, want github-app", m["app_type"])
	}
	if u, _ := m["authorize_url"].(string); strings.Contains(u, "scope") || strings.Contains(u, "?") {
		t.Fatalf("authorize_url = %q, want no scope", u)
	}
	if m := get(newFixture(t, func(c *Config) { c.AppType = "oauth-app" })); m["app_type"] != "oauth-app" {
		t.Fatalf("app_type = %v, want oauth-app", m["app_type"])
	}
	if _, err := New(Config{LoopbackOnly: true, AppType: "token"}); err == nil {
		t.Fatal("an unknown app type was accepted")
	}
}
