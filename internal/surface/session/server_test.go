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
	if len(replay) != 1 || !bytes.Equal(replay[0], []byte{core.TagUpdate, 1, 2, 3}) {
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
			if !bytes.Equal(b, []byte{core.TagUpdate, 4}) {
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

func TestDisallowedOriginIsRefused(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.Origins = []string{"https://brief.example.org"} })
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
	for _, o := range []string{"https://abcdesign.app", "https://brief.example.org", "http://127.0.0.1:8000", "http://[::1]:3000", "http://localhost"} {
		f.dial(t, o)
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
	if got := clientAddress(r, false); got != "10.0.0.1" {
		t.Errorf("not behind a proxy: %q, want the socket's address", got)
	}
	if got := clientAddress(r, true); got != "198.51.100.4" {
		t.Errorf("behind a proxy: %q, want the entry the proxy added", got)
	}
	r.Header.Del("X-Forwarded-For")
	if got := clientAddress(r, true); got != "10.0.0.1" {
		t.Errorf("behind a proxy with no header: %q, want the socket's address", got)
	}
}

func TestOriginsAreParsedStrictly(t *testing.T) {
	for _, o := range []string{"https://ok.example", "http://ok.example:8080"} {
		if _, err := New(Config{Origins: []string{o}}); err != nil {
			t.Errorf("%s refused: %v", o, err)
		}
	}
	for _, o := range []string{"*", "ok.example", "https://ok.example/path", "ftp://ok.example", "https://*.example"} {
		if _, err := New(Config{Origins: []string{o}}); err == nil {
			t.Errorf("%s accepted", o)
		}
	}
}
