package session

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// noFollow is a client that reads a redirect rather than following it.
var noFollow = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// signInState is the state the page sends: a random part, a dot, and the
// page's own address, base64url.
func signInState(page string) string {
	return "Zm9vYmFyYmF6cXV4cXV1eA." + base64.RawURLEncoding.EncodeToString([]byte(page))
}

func callback(t *testing.T, f *fixture, query url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := noFollow.Get(f.http.URL + "/auth/callback?" + query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

// TestTheCallbackReturnsTheCodeNamingItsOwnRelay: GitHub returns the person
// to the relay's own callback, the address the team's app registers, and the
// relay sends them on to the session page the sign-in started from, naming
// itself. The page then posts the code to that relay alone.
func TestTheCallbackReturnsTheCodeNamingItsOwnRelay(t *testing.T) {
	f := newFixture(t, nil)
	for _, page := range []string{"https://abcdesign.app/acme/brief/brief/"} {
		state := signInState(page)
		resp, body := callback(t, f, url.Values{"code": {"good-code"}, "state": {state}})
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("%s: %d %s, want 303", page, resp.StatusCode, body)
		}
		loc, err := url.Parse(resp.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		q := loc.Query()
		if got := loc.Scheme + "://" + loc.Host + loc.Path; got != page {
			t.Errorf("sent on to %s, want %s", got, page)
		}
		if q.Get("code") != "good-code" || q.Get("state") != state || q.Get("relay") != f.http.URL {
			t.Errorf("sent on with %v, want the code, the state and relay %s", q, f.http.URL)
		}
	}
	// GitHub's refusal goes back the same way, naming the relay.
	resp, _ := callback(t, f, url.Values{"error": {"access_denied"}, "state": {signInState("https://abcdesign.app/acme/brief/brief/")}})
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusSeeOther || loc.Query().Get("error") != "access_denied" || loc.Query().Get("relay") != f.http.URL || loc.Query().Has("code") {
		t.Errorf("a refusal: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// TestTheCallbackSendsNothingToAPageItDoesNotServe: the callback sends a code
// only to a session page on an origin the relay admits, for a repository it
// serves; anything else is refused with nothing sent on.
func TestTheCallbackSendsNothingToAPageItDoesNotServe(t *testing.T) {
	f := newFixture(t, nil)
	for name, q := range map[string]url.Values{
		"another origin":     {"code": {"good-code"}, "state": {signInState("https://evil.example/acme/brief/brief/")}},
		"not a brief page":   {"code": {"good-code"}, "state": {signInState("https://abcdesign.app/acme/brief/")}},
		"another repository": {"code": {"good-code"}, "state": {signInState("https://abcdesign.app/other/repo/brief/")}},
		"a query":            {"code": {"good-code"}, "state": {signInState("https://abcdesign.app/acme/brief/brief/?x=1")}},
		"a fragment":         {"code": {"good-code"}, "state": {signInState("https://abcdesign.app/acme/brief/brief/#x")}},
		"user info":          {"code": {"good-code"}, "state": {signInState("https://a@abcdesign.app/acme/brief/brief/")}},
		"dot segments":       {"code": {"good-code"}, "state": {signInState("https://abcdesign.app/x/../acme/brief/brief/")}},
		"no page":            {"code": {"good-code"}, "state": {"Zm9vYmFyYmF6cXV4cXV1eA"}},
		"not base64":         {"code": {"good-code"}, "state": {"Zm9vYmFyYmF6cXV4cXV1eA.!!!"}},
		"no code":            {"state": {signInState("https://abcdesign.app/acme/brief/brief/")}},
		"a bad code":         {"code": {"a b"}, "state": {signInState("https://abcdesign.app/acme/brief/brief/")}},
		"no state":           {"code": {"good-code"}},
	} {
		resp, body := callback(t, f, q)
		if resp.StatusCode != http.StatusBadRequest || resp.Header.Get("Location") != "" || strings.Contains(body, "good-code") {
			t.Errorf("%s: %d %q %s, want 400 with nothing sent on", name, resp.StatusCode, resp.Header.Get("Location"), body)
		}
	}
	// A relay without the secret offers no web sign-in, and so no callback.
	g := newFixture(t, func(c *Config) { c.ClientSecret = "" })
	if resp, _ := callback(t, g, url.Values{"code": {"good-code"}, "state": {signInState("https://abcdesign.app/acme/brief/brief/")}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("a relay without the secret: %d, want 404", resp.StatusCode)
	}
}

// TestTheCallbackForwardsOnlyToThePagesOwnBasePath: the callback forwards
// only to the page's own base path, the one its origin entry names, root
// when it names none. A static origin can carry other publishers' pages
// beside the session page, so a page under any other path on that origin is
// not the session page and is sent nothing.
func TestTheCallbackForwardsOnlyToThePagesOwnBasePath(t *testing.T) {
	sends := func(f *fixture, page string) bool {
		t.Helper()
		resp, body := callback(t, f, url.Values{"code": {"good-code"}, "state": {signInState(page)}})
		switch resp.StatusCode {
		case http.StatusSeeOther:
			loc, err := url.Parse(resp.Header.Get("Location"))
			if err != nil || loc.Scheme+"://"+loc.Host+loc.Path != page {
				t.Errorf("%s: sent on to %s", page, resp.Header.Get("Location"))
			}
			return true
		case http.StatusBadRequest:
			if resp.Header.Get("Location") != "" || strings.Contains(body, "good-code") {
				t.Errorf("%s: refused but sent on %q %s", page, resp.Header.Get("Location"), body)
			}
			return false
		}
		t.Fatalf("%s: %d %s", page, resp.StatusCode, body)
		return false
	}
	// The default origin's page is at the root.
	f := newFixture(t, nil)
	for page, want := range map[string]bool{
		"https://abcdesign.app/acme/brief/brief/":           true,
		"https://abcdesign.app/sites/acme/brief/brief/":     false,
		"https://abcdesign.app/a/b/acme/brief/brief/":       false,
		"https://abcdesign.app/acme/brief/brief/x/":         false,
		"https://abcdesign.app/acme/brief/acme/brief/brief": false,
	} {
		if got := sends(f, page); got != want {
			t.Errorf("default origins: %s sent=%v, want %v", page, got, want)
		}
	}
	// An origin entry naming a base path admits the page there and nowhere
	// else on that origin; an entry with none is the root.
	g := newFixture(t, func(c *Config) {
		c.Origins = []string{"https://pages.example/abcdesign/", "https://Other.Example"}
	})
	for page, want := range map[string]bool{
		"https://pages.example/abcdesign/acme/brief/brief/":       true,
		"https://pages.example/acme/brief/brief/":                 false,
		"https://pages.example/other/abcdesign/acme/brief/brief/": false,
		"https://pages.example/abcdesign/x/acme/brief/brief/":     false,
		"https://pages.example/AbcDesign/acme/brief/brief/":       false,
		"https://other.example/acme/brief/brief/":                 true,
		"https://other.example/abcdesign/acme/brief/brief/":       false,
		"https://abcdesign.app/acme/brief/brief/":                 false,
	} {
		if got := sends(g, page); got != want {
			t.Errorf("configured origins: %s sent=%v, want %v", page, got, want)
		}
	}
	// The browser's Origin carries no path, so the entry still admits its
	// origin wherever an origin alone is judged.
	for o, want := range map[string]bool{"https://pages.example": true, "https://other.example": true, "https://abcdesign.app": false} {
		if got := g.srv.originAllowed(o); got != want {
			t.Errorf("Origin %s admitted=%v, want %v", o, got, want)
		}
	}
}

// TestTheExchangeTakesOnlyItsOwnCallback: the relay exchanges only a code
// GitHub gave for its own callback address. A code for any other address,
// the session page's included, or one sent with none, is refused before
// GitHub is asked, so a code can be redeemed only through the relay GitHub
// returned the person to.
func TestTheExchangeTakesOnlyItsOwnCallback(t *testing.T) {
	f := newFixture(t, nil)
	own := f.http.URL + "/auth/callback"
	for _, redirect := range []string{"", "https://abcdesign.app/acme/brief/brief/", "https://other-relay.example/auth/callback", own + "?x=1", f.http.URL + "/auth/token"} {
		body := `{"code":"good-code","code_verifier":"` + verifier + `"`
		if redirect != "" {
			body += `,"redirect_uri":` + strconv.Quote(redirect)
		}
		resp, got := post(t, f.http.URL+"/auth/token", origin, body+"}")
		if resp.StatusCode != http.StatusBadRequest || strings.Contains(got, "ghu_") {
			t.Errorf("redirect_uri %q: %d %s, want a refusal", redirect, resp.StatusCode, got)
		}
	}
	f.github.mu.Lock()
	n := len(f.github.exchanges)
	f.github.mu.Unlock()
	if n != 0 {
		t.Fatalf("GitHub was asked %d times for codes the relay should refuse", n)
	}
	resp, got := post(t, f.http.URL+"/auth/token", origin, `{"code":"good-code","code_verifier":"`+verifier+`","redirect_uri":`+strconv.Quote(own)+`}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(got, "ghu_session_token") {
		t.Fatalf("its own callback: %d %s", resp.StatusCode, got)
	}
	f.github.mu.Lock()
	sent := f.github.exchanges[0].Get("redirect_uri")
	f.github.mu.Unlock()
	if sent != own {
		t.Errorf("GitHub was sent redirect_uri %q, want %q", sent, own)
	}
}
