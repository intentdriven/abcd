package cite

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// tlsChecker is testChecker trusting srv's self-signed certificate, so a chain
// that starts on https can be followed end to end against loopback servers.
func tlsChecker(t *testing.T, srv *httptest.Server) *HTTPChecker {
	t.Helper()
	c := testChecker(5 * time.Second)
	tr, ok := c.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("the checker's transport is %T, not *http.Transport", c.client.Transport)
	}
	tr.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	return c
}

// TestCheckDoesNotFollowARedirectOffHTTPS: the fetcher re-guarded the host on
// every hop but pinned no scheme, so an https citation redirected to plaintext
// http was followed, and the address the plaintext leg reported reached the
// committed baseline as the citation's final URL (iss-2609012037440084, the
// shape GHSA-35fj-9w6f-7h62 closed in memory ingest). A downgrade is refused,
// and — since it is no evidence the source is dead — routed to the manual
// queue as blocked rather than recorded as broken.
func TestCheckDoesNotFollowARedirectOffHTTPS(t *testing.T) {
	plainHit := false
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/landed?token=SECRET-QUERY-VALUE", http.StatusFound)
	}))
	defer secure.Close()

	got := tlsChecker(t, secure).Check(secure.URL + "/cited")
	if plainHit {
		t.Fatal("the checker followed an https citation down to plaintext http")
	}
	if got.Status != StatusBlocked {
		t.Fatalf("status = %q (%s), want %q: a downgrade is not evidence the source is dead", got.Status, got.Detail, StatusBlocked)
	}
	if !got.Answered {
		t.Error("a host answered with a 3xx, so the outcome must say so")
	}
	if got.FinalURL != "" {
		t.Errorf("final URL = %q, want none: the plaintext address must not reach the baseline", got.FinalURL)
	}
	if strings.Contains(got.Detail, "SECRET-QUERY-VALUE") || strings.Contains(got.Detail, plain.URL) {
		t.Errorf("the refusal echoes the plaintext hop: %q", got.Detail)
	}
	if !strings.Contains(got.Detail, "https") {
		t.Errorf("the refusal does not say the chain left https: %q", got.Detail)
	}
}

// TestCheckStillFollowsAnHTTPCitationUpgrade: the pin is against LEAVING https,
// not against http. A citation written as http:// that upgrades to https — the
// commonest redirect there is — and an http chain that never reached https are
// followed exactly as before; refusing them would record a live source as
// unverifiable.
func TestCheckStillFollowsAnHTTPCitationUpgrade(t *testing.T) {
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer secure.Close()
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upgrade":
			http.Redirect(w, r, secure.URL+"/page", http.StatusMovedPermanently)
		case "/moved":
			http.Redirect(w, r, "/page", http.StatusMovedPermanently)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer plain.Close()

	c := tlsChecker(t, secure)
	if got := c.Check(plain.URL + "/upgrade"); got.Status != StatusOK || got.FinalURL != secure.URL+"/page" {
		t.Errorf("http->https upgrade: status %q final %q (%s), want ok at %s", got.Status, got.FinalURL, got.Detail, secure.URL+"/page")
	}
	if got := c.Check(plain.URL + "/moved"); got.Status != StatusOK || got.FinalURL != plain.URL+"/page" {
		t.Errorf("http->http move: status %q final %q (%s), want ok at %s", got.Status, got.FinalURL, got.Detail, plain.URL+"/page")
	}
}
