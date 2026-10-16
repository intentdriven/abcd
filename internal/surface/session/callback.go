package session

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// The web sign-in's callback is the relay's own address, not the session
// page's: the team's GitHub App registers https://<relay>/auth/callback, and
// GitHub sends the person back there and nowhere else. GitHub matches a
// redirect_uri against the registered callback URLs exactly when wildcard
// matching is off, and, when it is on, by host and port and a path beneath
// the callback's; the user-authorization page says the redirect_uri "can't
// contain any additional parameters" and documents nothing about a query in
// a registered callback
// (https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/about-the-user-authorization-callback-url,
// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app).
// So the relay a code is bound to is named by the callback's host, which both
// matching modes compare, rather than by a query on the page's address, which
// neither page promises to compare.
//
// The relay then sends the person on to the session page the sign-in started
// from, naming itself, and the page posts the code and its PKCE verifier to
// that relay alone. The exchange takes only a code given for this relay's
// own callback (serveToken), so a code reaches the relay GitHub returned the
// person to, and no other.

// CallbackPath is the web sign-in's callback on the relay.
const CallbackPath = "/auth/callback"

// maxSignInState bounds the state the page sends: its random part, a dot,
// and its own address, base64url.
const maxSignInState = 2048

// pageRouteRe is a session page's path after its base path:
// <owner>/<repo>/brief/. The base path is the one the page's origin entry
// names (Server.pageBases), never read from the path itself: a static origin
// can carry other publishers' pages beside the session page, and a code is
// forwarded only to the page's own base path.
var pageRouteRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]{0,38})/([A-Za-z0-9._-]{1,100})/brief/$`)

// ownOrigin is the relay's origin as the request reached it: https, or http
// for a loopback host, where a local relay is reached directly.
func ownOrigin(r *http.Request) string {
	host := strings.ToLower(r.Host)
	scheme := "https"
	if h, _, err := splitHost(host); err == nil && isLoopbackName(h) {
		scheme = "http"
	}
	return scheme + "://" + host
}

// callbackURL is the relay's own callback, as GitHub was told it.
func callbackURL(r *http.Request) string { return ownOrigin(r) + CallbackPath }

func splitHost(host string) (string, string, error) {
	u, err := url.Parse("//" + host)
	if err != nil {
		return "", "", err
	}
	return u.Hostname(), u.Port(), nil
}

func isLoopbackName(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// statePage reads the session page a sign-in started from out of its state,
// and reports whether the relay may send a code there: an origin it admits,
// a session page's path under exactly the base path that origin's entry
// names, for a repository it serves, and nothing else.
func (s *Server) statePage(state string) (string, bool) {
	if state == "" || len(state) > maxSignInState {
		return "", false
	}
	_, enc, ok := strings.Cut(state, ".")
	if !ok {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", false
	}
	u, err := url.Parse(string(raw))
	if err != nil || !u.IsAbs() || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || strings.Contains(string(raw), "#") || u.RawPath != "" {
		return "", false
	}
	origin, ok := normalizeOrigin(u.Scheme + "://" + u.Host)
	if !ok || !s.originAllowed(origin) {
		return "", false
	}
	for _, seg := range strings.Split(u.Path, "/") {
		if seg == "." || seg == ".." {
			return "", false
		}
	}
	for _, base := range s.pageBases(origin) {
		rest, under := strings.CutPrefix(u.Path, base)
		if !under {
			continue
		}
		if m := pageRouteRe.FindStringSubmatch(rest); m != nil && s.relay.Serves(m[1]+"/"+m[2]) {
			return origin + u.Path, true
		}
	}
	return "", false
}

// serveCallback takes GitHub's return from the web sign-in and sends the
// person on to the session page the sign-in started from, with GitHub's code
// (or its error), the state, and this relay's own origin. It exchanges
// nothing itself: the PKCE verifier stays with the page until the page posts
// it here.
func (s *Server) serveCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.ClientID == "" || s.cfg.ClientSecret == "" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	refuse := func() {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "This sign-in cannot go back to its session page. Open the session page again and sign in from there.\n")
	}
	page, ok := s.statePage(q.Get("state"))
	if !ok {
		refuse()
		return
	}
	out := url.Values{"state": {q.Get("state")}, "relay": {ownOrigin(r)}}
	switch code, gerr := q.Get("code"), q.Get("error"); {
	case code != "" && oauthCodeRe.MatchString(code):
		out.Set("code", code)
	case code == "" && githubErrorRe.MatchString(gerr):
		out.Set("error", gerr)
	default:
		refuse()
		return
	}
	http.Redirect(w, r, page+"?"+out.Encode(), http.StatusSeeOther)
}
