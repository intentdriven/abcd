// Package session is the brief session relay's front door
// (itd-2610091016262335, spc-2610141709430146 step 1): the HTTP handler
// `abcd session serve` listens with. It serves four routes and nothing else:
//
//   - /session, the WebSocket every participant's page holds, handed frame by
//     frame to the transport-agnostic relay in internal/core/session;
//   - /auth/token, the sign-in exchange: the page posts GitHub's code and its
//     PKCE verifier, and the relay adds the sign-in app's secret and asks
//     GitHub for the token (the token-mediating backend of the IETF's
//     browser-based-apps guidance);
//   - /config, the sign-in app's public client id;
//   - /healthz, for the host's health check.
//
// It is abcd's second listener, admitted by adr-2610032150581128's amendment
// of 2026-10-14: started on purpose, on loopback unless an address is chosen,
// and reached on a person's machine through Tailscale Serve only. It writes
// nothing to disk and logs nothing: no token, no secret, no brief text.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	core "github.com/intentdriven/abcd/internal/core/session"
)

// DefaultOrigin is the page's own origin. With loopback pages, for a page run
// locally, it is the default allowlist, which Config.Origins replaces.
const DefaultOrigin = "https://abcdesign.app"

// The kinds of GitHub sign-in app a team registers. A GitHub App's user
// access token holds only the permissions both the app and the person have,
// on the repositories the app is installed on, and its authorization takes no
// scope: "GitHub Apps do not request scopes during authorization to decide
// which resources the resulting token can access"
// (https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app).
// An OAuth app's token is scoped by the page's request instead.
const (
	AppTypeGitHubApp = "github-app"
	AppTypeOAuthApp  = "oauth-app"
)

// The GitHub endpoints the relay talks to unless a test names others.
const (
	DefaultGitHubAPI = "https://api.github.com"
	DefaultGitHubWeb = "https://github.com"
)

// The server's own bounds.
const (
	maxAuthBody   = 8 << 10
	githubTimeout = 15 * time.Second
	writeTimeout  = 10 * time.Second
	// sendQueue is how many items one connection in a session may have
	// waiting, a log replay counting as one; a connection that falls further
	// behind is dropped. preRoomQueue is the bound before it enters a
	// session, when nothing but answers to its own messages is sent to it.
	sendQueue    = 1024
	preRoomQueue = 64
	// SweepInterval is how often the relay looks for idle sessions.
	SweepInterval = time.Minute
	// pingEvery and pingWait keep a WebSocket honest: a connection that does
	// not answer a ping in time is dropped, so a vanished device does not
	// hold a participant's place.
	pingEvery = 30 * time.Second
	pingWait  = 20 * time.Second
	// httpWait bounds the reading and writing of a plain request; the
	// WebSocket is bounded by its pings instead, since a deadline on the
	// server would outlive the upgrade.
	httpWait = 30 * time.Second
)

// Config is what the front door needs.
type Config struct {
	// Origins, each an exact scheme://host[:port], replace the default
	// allowlist (DefaultOrigin and any loopback page) when given: a relay
	// serving a page from its own origin admits nothing else.
	Origins []string
	// BehindProxy trusts the last X-Forwarded-For entry as the client's
	// address, which is right only behind a proxy that sets it.
	BehindProxy bool
	// LoopbackOnly says the listener is bound to loopback alone: the host's
	// own machine, reached through Tailscale Serve. Then, and only for a
	// loopback peer, the tailnet address Serve forwards is the client's.
	LoopbackOnly bool
	// Repos (owner/name) and Owners are the repositories the relay serves
	// sessions for (core.Options). A relay that is not LoopbackOnly must be
	// given at least one: otherwise anyone signed in to GitHub could hold
	// its sessions with a repository of their own.
	Repos  []string
	Owners []string
	// ClientID is the team's GitHub sign-in app's public id.
	ClientID string
	// ClientSecret is that app's secret, read from the environment by the
	// caller. It never leaves the relay but in the exchange with GitHub.
	ClientSecret string
	// AppType is the sign-in app's kind, AppTypeGitHubApp unless set.
	AppType string
	// GitHubAPI and GitHubWeb default to GitHub's own.
	GitHubAPI string
	GitHubWeb string
	// HTTPClient talks to GitHub; nil is a client with a timeout.
	HTTPClient *http.Client
	Limits     core.Limits
	Now        func() time.Time
	Rand       io.Reader
}

// Server is the relay's HTTP handler.
type Server struct {
	cfg     Config
	origins map[string]bool // the given origins; nil admits the default list
	relay   *core.Relay
	client  *http.Client
}

// ErrNoRepositories is New's refusal of a relay beyond loopback that names
// no repository to serve.
var ErrNoRepositories = errors.New("a relay listening beyond this computer serves only the repositories it is given; name at least one")

// New builds the handler, refusing an origin that is not an exact
// scheme://host[:port] over http or https, a repository that is not
// owner/name, an owner that is not a GitHub name, and a relay beyond
// loopback with neither.
func New(cfg Config) (*Server, error) {
	for _, r := range cfg.Repos {
		if !core.ValidRepo(r) {
			return nil, fmt.Errorf("%q is not a repository: give owner/name", r)
		}
	}
	for _, o := range cfg.Owners {
		if !core.ValidOwner(o) {
			return nil, fmt.Errorf("%q is not a GitHub owner", o)
		}
	}
	if !cfg.LoopbackOnly && len(cfg.Repos) == 0 && len(cfg.Owners) == 0 {
		return nil, ErrNoRepositories
	}
	switch cfg.AppType {
	case "":
		cfg.AppType = AppTypeGitHubApp
	case AppTypeGitHubApp, AppTypeOAuthApp:
	default:
		return nil, fmt.Errorf("%q is not a sign-in app type: give %s or %s", cfg.AppType, AppTypeGitHubApp, AppTypeOAuthApp)
	}
	var origins map[string]bool
	for _, o := range cfg.Origins {
		if origins == nil {
			origins = map[string]bool{}
		}
		n, ok := normalizeOrigin(o)
		if !ok {
			return nil, fmt.Errorf("%q is not an origin: give scheme://host[:port], over http or https, with no path or wildcard", o)
		}
		origins[n] = true
	}
	if cfg.GitHubAPI == "" {
		cfg.GitHubAPI = DefaultGitHubAPI
	}
	if cfg.GitHubWeb == "" {
		cfg.GitHubWeb = DefaultGitHubWeb
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: githubTimeout}
	}
	// A redirect would carry the request somewhere GitHub did not answer
	// from; the relay follows none.
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if c.Timeout == 0 {
		c.Timeout = githubTimeout
	}
	s := &Server{cfg: cfg, origins: origins, client: &c}
	s.relay = core.New(core.Options{
		Limits:   cfg.Limits,
		Verifier: GitHub{API: cfg.GitHubAPI, Client: &c},
		Now:      cfg.Now,
		Rand:     cfg.Rand,
		Repos:    cfg.Repos,
		Owners:   cfg.Owners,
	})
	return s, nil
}

// Relay is the relay the handler feeds.
func (s *Server) Relay() *core.Relay { return s.relay }

// Sweep purges idle sessions every interval until ctx ends.
func (s *Server) Sweep(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.relay.Sweep()
		}
	}
}

// ServeHTTP is the fixed route table.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	if r.URL.Path != "/session" {
		rc := http.NewResponseController(w)
		_ = rc.SetReadDeadline(time.Now().Add(httpWait))
		_ = rc.SetWriteDeadline(time.Now().Add(httpWait))
	}
	switch r.URL.Path {
	case "/session":
		s.serveSession(w, r)
	case "/auth/token":
		s.serveToken(w, r)
	case "/config":
		s.serveConfig(w, r)
	case "/healthz":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	default:
		http.NotFound(w, r)
	}
}

// normalizeOrigin reads s as an exact origin: http or https, a host, an
// optional port, and nothing else.
func normalizeOrigin(s string) (string, bool) {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" ||
		u.Hostname() == "" || strings.ContainsAny(u.Hostname(), "*?[]") {
		return "", false
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), true
}

// originAllowed judges a request's Origin: one of the given origins exactly,
// or, when none was given, DefaultOrigin or a loopback page. A request with
// no Origin is refused: every browser sends one on a WebSocket and on a
// cross-origin POST.
func (s *Server) originAllowed(origin string) bool {
	n, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}
	if s.origins != nil {
		return s.origins[n]
	}
	if n == DefaultOrigin {
		return true
	}
	u, _ := url.Parse(n)
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// clientAddress is the address the failed-join limit counts.
//
//   - Behind a declared proxy (--behind-proxy, the team relay), it is the
//     rightmost X-Forwarded-For entry, across every line of the header: the
//     one that proxy appended, one trusted hop. Anything before it the client
//     could have written. Render's proxy appends rather than replaces, "so a
//     caller who sends their own copy of the header controls the leftmost
//     entry" (https://render.com/articles/host-pocketbase-on-render), and
//     Render's advice is to trust one hop, Express's 'trust proxy' 1, which
//     reads the rightmost entry
//     (https://render.com/articles/how-render-handles-ddos-attacks). An entry
//     that is not a bare IP address reads as missing.
//   - On the host's own machine the relay is bound to loopback alone and
//     Tailscale Serve proxies to it, so every tailnet visitor's socket address
//     is loopback and one visitor's wrong codes would shut out everyone. There,
//     and only when the socket's peer is itself loopback, it is the address
//     Serve forwards (tailnetForwarded).
//   - Otherwise, and whenever the header is missing or malformed, it is the
//     socket's peer.
func clientAddress(r *http.Request, behindProxy, loopbackOnly bool) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if behindProxy {
		if fwd := r.Header.Values("X-Forwarded-For"); len(fwd) > 0 {
			parts := strings.Split(fwd[len(fwd)-1], ",")
			if a, err := netip.ParseAddr(strings.TrimSpace(parts[len(parts)-1])); err == nil {
				return a.Unmap().WithZone("").String()
			}
		}
		return host
	}
	if loopbackOnly {
		if peer, err := netip.ParseAddr(host); err == nil && peer.Unmap().IsLoopback() {
			if fwd, ok := tailnetForwarded(r); ok {
				return fwd
			}
		}
	}
	return host
}

// tailnetForwarded reads the visitor's address as Tailscale Serve forwards
// it. Serve's reverse proxy sets X-Forwarded-For to the tailnet address of the
// device that connected, replacing whatever the device sent, alongside
// X-Forwarded-Host, X-Forwarded-Proto (https) and, for a device that is not
// tagged, the Tailscale-User-Login, Tailscale-User-Name and
// Tailscale-User-Profile-Pic identity headers, which it strips from the
// incoming request first:
// https://tailscale.com/kb/1312/serve (the identity headers) and
// https://github.com/tailscale/tailscale/blob/main/ipn/ipnlocal/serve.go
// (addProxyForwardedHeaders, which sets X-Forwarded-For). So the header is
// accepted only in the shape Serve writes it: one line holding one address
// in Tailscale's ranges. Anything else is not Serve's, and reads as missing.
func tailnetForwarded(r *http.Request) (string, bool) {
	fwd := r.Header.Values("X-Forwarded-For")
	if len(fwd) != 1 {
		return "", false
	}
	a, err := netip.ParseAddr(strings.TrimSpace(fwd[0]))
	if err != nil || a.Zone() != "" || !core.IsTailnetAddr(a) {
		return "", false
	}
	return a.Unmap().String(), true
}

// serveSession upgrades to the session's WebSocket and runs one connection.
// Past the relay's connection caps the upgrade is refused with 503, before
// anything is allocated for it and without touching another connection.
func (s *Server) serveSession(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r.Header.Get("Origin")) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	// The request's context ends when the server shuts down (Run's
	// BaseContext), which ends every session connection with it.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	p := newPeer(cancel, preRoomQueue)
	conn, err := s.relay.Connect(p, clientAddress(r, s.cfg.BehindProxy, s.cfg.LoopbackOnly))
	if err != nil {
		http.Error(w, "the relay is busy; try again later", http.StatusServiceUnavailable)
		return
	}
	// The origin was judged above against the relay's own allowlist, which
	// the library's same-host rule would widen.
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		conn.Leave()
		return
	}
	ws.SetReadLimit(int64(s.relay.Limits().MaxMessageBytes))
	p.ws = ws
	expire := time.AfterFunc(s.relay.Limits().AdmitWithin, conn.Expire)
	defer expire.Stop()
	done := make(chan struct{})
	go func() { p.writeLoop(ctx); close(done) }()
	go p.pingLoop(ctx)
	for {
		typ, data, err := ws.Read(ctx)
		if err != nil {
			break
		}
		conn.Receive(ctx, core.Frame{Binary: typ == websocket.MessageBinary, Data: data})
	}
	conn.Leave()
	cancel()
	<-done
	_ = ws.CloseNow()
}

// outFrame is one queued item: a frame, a session's log replay, or the close
// that ends the queue.
type outFrame struct {
	frame  core.Frame
	log    [][]byte
	close  bool
	status websocket.StatusCode
	reason string
}

// peer is the relay's handle on one WebSocket: a queue its writer drains in
// order, so a close the relay asks for follows every item queued before it.
// The queue holds at most limit items; a log replay is one item however long
// the log, written frame by frame as the connection takes them.
type peer struct {
	ws     *websocket.Conn
	cancel context.CancelFunc
	wake   chan struct{} // holds one token while items wait
	mu     sync.Mutex
	queue  []outFrame
	limit  int
	closed bool
}

func newPeer(cancel context.CancelFunc, limit int) *peer {
	return &peer{cancel: cancel, wake: make(chan struct{}, 1), limit: limit}
}

// push queues one item under the lock, or reports that the queue is full.
func (p *peer) push(f outFrame) bool {
	if len(p.queue) >= p.limit {
		return false
	}
	p.queue = append(p.queue, f)
	select {
	case p.wake <- struct{}{}:
	default:
	}
	return true
}

func (p *peer) Send(f core.Frame) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return true
	}
	return p.push(outFrame{frame: f})
}

func (p *peer) SendLog(entries [][]byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return true
	}
	return p.push(outFrame{log: entries})
}

// Entered lets a connection that is in a session have the full queue.
func (p *peer) Entered() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limit = sendQueue
}

func (p *peer) Close(policy bool, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	status := websocket.StatusNormalClosure
	if policy {
		status = websocket.StatusPolicyViolation
	}
	if !p.push(outFrame{close: true, status: status, reason: reason}) {
		p.cancel()
	}
}

// next takes the oldest queued item, waiting for one until ctx ends.
func (p *peer) next(ctx context.Context) (outFrame, bool) {
	for {
		p.mu.Lock()
		if len(p.queue) > 0 {
			f := p.queue[0]
			p.queue[0] = outFrame{}
			p.queue = p.queue[1:]
			p.mu.Unlock()
			return f, true
		}
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return outFrame{}, false
		case <-p.wake:
		}
	}
}

// pingLoop pings the connection until ctx ends, and ends the connection when
// a ping goes unanswered.
func (p *peer) pingLoop(ctx context.Context) {
	t := time.NewTicker(pingEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pctx, cancel := context.WithTimeout(ctx, pingWait)
			err := p.ws.Ping(pctx)
			cancel()
			if err != nil {
				p.cancel()
				return
			}
		}
	}
}

func (p *peer) writeLoop(ctx context.Context) {
	for {
		f, ok := p.next(ctx)
		if !ok {
			return
		}
		if f.close {
			_ = p.ws.Close(f.status, f.reason)
			p.cancel()
			return
		}
		if f.log != nil {
			for _, entry := range f.log {
				if !p.write(ctx, core.Frame{Binary: true, Data: entry}) {
					return
				}
			}
			continue
		}
		if !p.write(ctx, f.frame) {
			return
		}
	}
}

// write writes one frame, ending the connection when it cannot.
func (p *peer) write(ctx context.Context, f core.Frame) bool {
	typ := websocket.MessageText
	if f.Binary {
		typ = websocket.MessageBinary
	}
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	err := p.ws.Write(wctx, typ, f.Data)
	cancel()
	if err != nil {
		p.cancel()
		return false
	}
	return true
}

// writeJSON answers with a JSON body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// cors answers a request from an allowed origin with the headers a browser
// needs to read the answer, and reports whether the origin is allowed.
func (s *Server) cors(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if !s.originAllowed(origin) {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	return true
}

// serveConfig names the sign-in app's public client id and kind, and whether
// sign-in is set up at all. It never names the secret. The authorize URL is
// bare: it carries no scope, which a GitHub App does not use, and the page
// adds the request's own parameters (a scope only for an OAuth app).
func (s *Server) serveConfig(w http.ResponseWriter, r *http.Request) {
	s.cors(w, r)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     s.cfg.ClientID,
		"sign_in":       s.cfg.ClientID != "" && s.cfg.ClientSecret != "",
		"authorize_url": s.cfg.GitHubWeb + "/login/oauth/authorize",
		"app_type":      s.cfg.AppType,
	})
}

// pkceVerifierRe is RFC 7636's code_verifier: 43 to 128 unreserved characters.
var pkceVerifierRe = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// oauthCodeRe bounds GitHub's authorization code.
var oauthCodeRe = regexp.MustCompile(`^[A-Za-z0-9._~-]{1,256}$`)

// githubErrorRe is the shape of an OAuth error code; anything else GitHub
// says is not passed on.
var githubErrorRe = regexp.MustCompile(`^[a-z_]{1,64}$`)

// serveToken is the sign-in exchange.
func (s *Server) serveToken(w http.ResponseWriter, r *http.Request) {
	allowed := s.cors(w, r)
	if r.Method == http.MethodOptions {
		if !allowed {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin-not-allowed"})
		return
	}
	if s.cfg.ClientID == "" || s.cfg.ClientSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "sign-in-not-configured"})
		return
	}
	var req struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAuthBody))
	if err := dec.Decode(&req); err != nil || !oauthCodeRe.MatchString(req.Code) || !pkceVerifierRe.MatchString(req.CodeVerifier) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad-request"})
		return
	}
	if req.RedirectURI != "" {
		u, err := url.Parse(req.RedirectURI)
		if err != nil || !u.IsAbs() || !s.originAllowed(u.Scheme+"://"+u.Host) || u.User != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad-redirect"})
			return
		}
	}
	form := url.Values{
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"code":          {req.Code},
		"code_verifier": {req.CodeVerifier},
	}
	if req.RedirectURI != "" {
		form.Set("redirect_uri", req.RedirectURI)
	}
	ctx, cancel := context.WithTimeout(r.Context(), githubTimeout)
	defer cancel()
	greq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.GitHubWeb+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
		return
	}
	greq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	greq.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(greq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
		return
	}
	defer resp.Body.Close()
	var got struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&got) != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
		return
	}
	// Only named fields pass, and none that carries the secret.
	for _, v := range []string{got.AccessToken, got.TokenType, got.Scope, got.Error} {
		if strings.Contains(v, s.cfg.ClientSecret) {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
			return
		}
	}
	if got.AccessToken == "" {
		code := "exchange-failed"
		if githubErrorRe.MatchString(got.Error) {
			code = got.Error
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"access_token": got.AccessToken, "token_type": got.TokenType, "scope": got.Scope})
}

// GitHub is the relay's Verifier: it asks GitHub's API who a token belongs to
// and whether that login may push to the repository.
type GitHub struct {
	API    string
	Client *http.Client
}

// errVerify is the one error a failed verify reports; it never carries the
// token or what GitHub said.
var errVerify = errors.New("GitHub did not accept the token")

// Verify answers the login and the repository's permissions.push. A
// repository GitHub hides from the token (404) is no push, not an error.
func (g GitHub) Verify(ctx context.Context, token, repo string) (string, bool, error) {
	if !core.ValidRepo(repo) {
		return "", false, errors.New("not an owner/name repository")
	}
	var user struct {
		Login string `json:"login"`
	}
	status, err := g.get(ctx, token, "/user", &user)
	if err != nil || status != http.StatusOK || user.Login == "" {
		return "", false, errVerify
	}
	var rp struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	status, err = g.get(ctx, token, "/repos/"+repo, &rp)
	if err != nil {
		return "", false, errVerify
	}
	if status != http.StatusOK {
		return user.Login, false, nil
	}
	return user.Login, rp.Permissions.Push, nil
}

func (g GitHub) get(ctx context.Context, token, path string, into any) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, githubTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.API+path, nil)
	if err != nil {
		return 0, errVerify
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: githubTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, errVerify
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(into) != nil {
			return 0, errVerify
		}
	}
	return resp.StatusCode, nil
}
