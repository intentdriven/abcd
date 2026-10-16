package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitHub's device flow, for a relay holding the GitHub App's client id and
// no secret: a local session on the host's own computer, where no secret may
// sit (adr-2610161400542916). "The client_secret is not needed for the device
// flow" (https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps).
// GitHub documents no CORS for its device endpoints, so the relay asks them
// for the page.
//
// The device code is the secret of a sign-in under way: whoever holds it
// collects the token once the person confirms. So the relay keeps it, under a
// random handle, and the page sees only the handle, the user code it shows the
// person, and the timings. The relay polls GitHub for a handle no faster than
// GitHub's interval, however often the page asks, so the number of sign-ins
// held (maxDeviceLogins) bounds what the relay can ask of GitHub.

// deviceGrant is the device flow's grant type (RFC 8628 section 3.4).
const deviceGrant = "urn:ietf:params:oauth:grant-type:device_code"

// The device flow's own bounds.
const (
	// deviceInterval is GitHub's documented minimum between polls, and the
	// interval taken when GitHub names none.
	deviceInterval = 5 * time.Second
	// slowDownStep is what GitHub adds to the interval on slow_down.
	slowDownStep = 5 * time.Second
	// maxDeviceLife caps how long a sign-in is held, whatever GitHub says:
	// GitHub's own default is 900 seconds.
	maxDeviceLife = 15 * time.Minute
	// maxDeviceInterval caps the interval GitHub may ask for.
	maxDeviceInterval = 60 * time.Second
)

// deviceLogin is one device sign-in under way.
type deviceLogin struct {
	code     string
	interval time.Duration
	next     time.Time // GitHub is not asked before this
	expires  time.Time
	asking   bool // a poll is with GitHub now
}

// deviceHandleRe is a handle the relay gave: 16 random bytes, base64url.
var deviceHandleRe = regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)

// userCodeRe bounds the user code GitHub hands out ("8 characters with a
// hyphen in the middle"), which the page shows.
var userCodeRe = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}$`)

// deviceOffered says the relay offers the device flow: a client id and no
// secret. A relay with the secret offers the web flow instead.
func (s *Server) deviceOffered() bool {
	return s.cfg.ClientID != "" && s.cfg.ClientSecret == ""
}

func (s *Server) now() time.Time {
	if s.cfg.Now != nil {
		return s.cfg.Now()
	}
	return time.Now()
}

// preflight answers a CORS preflight for a POST route, and reports whether r
// was one.
func (s *Server) preflight(w http.ResponseWriter, r *http.Request, allowed bool) bool {
	if r.Method != http.MethodOptions {
		return false
	}
	if !allowed {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return true
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
	return true
}

// postFrom admits a POST from an allowed origin, answering anything else.
func (s *Server) postFrom(w http.ResponseWriter, r *http.Request) bool {
	allowed := s.cors(w, r)
	if s.preflight(w, r, allowed) {
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin-not-allowed"})
		return false
	}
	return true
}

// pruneDevices drops every sign-in whose device code has expired. The
// caller holds s.mu.
func (s *Server) pruneDevices(now time.Time) {
	for h, d := range s.devices {
		if !now.Before(d.expires) {
			delete(s.devices, h)
		}
	}
}

// serveDeviceStart starts a device sign-in: the relay asks GitHub for a
// device code, keeps it, and answers with a handle, the user code and
// GitHub's verification address.
func (s *Server) serveDeviceStart(w http.ResponseWriter, r *http.Request) {
	if !s.postFrom(w, r) {
		return
	}
	if !s.deviceOffered() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "device-flow-not-offered"})
		return
	}
	now := s.now()
	s.mu.Lock()
	s.pruneDevices(now)
	full := len(s.devices) >= maxDeviceLogins
	s.mu.Unlock()
	if full {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "busy"})
		return
	}
	// Counted with the web flow's exchanges, per address, before GitHub is
	// asked: a device code is a request against the app's own limits.
	if !s.relay.AllowExchange(clientAddress(r, s.cfg.BehindProxy, s.cfg.LoopbackOnly)) {
		w.Header().Set("Retry-After", strconv.Itoa(int(s.relay.Limits().ExchangeWindow/time.Second)))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate-limited"})
		return
	}
	var got struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		ExpiresIn  int    `json:"expires_in"`
		Interval   int    `json:"interval"`
		Error      string `json:"error"`
	}
	if !s.askGitHub(r.Context(), "/login/device/code", url.Values{"client_id": {s.cfg.ClientID}}, &got) {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
		return
	}
	if got.DeviceCode == "" || !userCodeRe.MatchString(got.UserCode) {
		code := "exchange-failed"
		if got.Error != "" && githubErrorRe.MatchString(got.Error) {
			code = got.Error // device_flow_disabled, above all
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": code})
		return
	}
	life := time.Duration(got.ExpiresIn) * time.Second
	if life <= 0 || life > maxDeviceLife {
		life = maxDeviceLife
	}
	interval := clampInterval(got.Interval, deviceInterval)
	handle, err := s.newHandle()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exchange-failed"})
		return
	}
	s.mu.Lock()
	if len(s.devices) >= maxDeviceLogins {
		s.mu.Unlock()
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "busy"})
		return
	}
	s.devices[handle] = &deviceLogin{code: got.DeviceCode, interval: interval, next: now.Add(interval), expires: now.Add(life)}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"handle":           handle,
		"user_code":        got.UserCode,
		"verification_uri": s.cfg.GitHubWeb + "/login/device",
		"expires_in":       int(life / time.Second),
		"interval":         int(interval / time.Second),
	})
}

// clampInterval reads GitHub's interval in seconds: at least floor, at most
// maxDeviceInterval.
func clampInterval(seconds int, floor time.Duration) time.Duration {
	d := time.Duration(seconds) * time.Second
	if d < floor {
		d = floor
	}
	if d > maxDeviceInterval {
		d = maxDeviceInterval
	}
	return d
}

// newHandle is 16 random bytes, base64url: a sign-in's handle.
func (s *Server) newHandle() (string, error) {
	src := s.cfg.Rand
	if src == nil {
		src = rand.Reader
	}
	b := make([]byte, 16)
	if _, err := io.ReadFull(src, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// serveDevicePoll asks after a device sign-in. Before the sign-in's next
// poll is due the relay answers authorization_pending without asking GitHub;
// otherwise it asks once, and hands over the token when GitHub gives one. A
// sign-in GitHub ends (denied, expired, disabled) is dropped.
func (s *Server) serveDevicePoll(w http.ResponseWriter, r *http.Request) {
	if !s.postFrom(w, r) {
		return
	}
	var req struct {
		Handle string `json:"handle"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAuthBody))
	if err := dec.Decode(&req); err != nil || !deviceHandleRe.MatchString(req.Handle) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad-request"})
		return
	}
	now := s.now()
	s.mu.Lock()
	s.pruneDevices(now)
	d, ok := s.devices[req.Handle]
	if !ok {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired_token"})
		return
	}
	if d.asking || now.Before(d.next) {
		interval := d.interval
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"error": "authorization_pending", "interval": int(interval / time.Second)})
		return
	}
	d.asking = true
	code := d.code
	s.mu.Unlock()

	var got struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		Interval    int    `json:"interval"`
	}
	asked := s.askGitHub(r.Context(), "/login/oauth/access_token", url.Values{
		"client_id": {s.cfg.ClientID}, "device_code": {code}, "grant_type": {deviceGrant},
	}, &got)

	s.mu.Lock()
	defer s.mu.Unlock()
	d.asking = false
	d.next = s.now().Add(d.interval)
	switch {
	case !asked:
		// GitHub did not answer: the sign-in stays, and the page asks again.
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "exchange-failed"})
	case got.AccessToken != "":
		delete(s.devices, req.Handle)
		writeJSON(w, http.StatusOK, map[string]string{"access_token": got.AccessToken, "token_type": got.TokenType, "scope": got.Scope})
	case got.Error == "authorization_pending":
		writeJSON(w, http.StatusOK, map[string]any{"error": "authorization_pending", "interval": int(d.interval / time.Second)})
	case got.Error == "slow_down":
		// GitHub adds five seconds and names the new interval.
		d.interval = clampInterval(got.Interval, d.interval+slowDownStep)
		d.next = s.now().Add(d.interval)
		writeJSON(w, http.StatusOK, map[string]any{"error": "slow_down", "interval": int(d.interval / time.Second)})
	default:
		delete(s.devices, req.Handle)
		out := "exchange-failed"
		if githubErrorRe.MatchString(got.Error) {
			out = got.Error
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": out})
	}
}

// askGitHub posts form to GitHub's sign-in host at path and decodes its JSON
// answer into into, reporting whether GitHub answered with JSON at all.
func (s *Server) askGitHub(ctx context.Context, path string, form url.Values, into any) bool {
	ctx, cancel := context.WithTimeout(ctx, githubTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.GitHubWeb+path, strings.NewReader(form.Encode()))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(into) == nil
}
