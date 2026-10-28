package urlguard

import (
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// get makes one request the way every caller of Download does: Admit first,
// then the pinned client.
func get(d *Download, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if err := d.Admit(u); err != nil {
		return err
	}
	resp, err := d.Client.Get(raw)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func hostOf(s *httptest.Server) string { u, _ := url.Parse(s.URL); return u.Host }

func TestDownloadReachesOnlyItsPurposesHosts(t *testing.T) {
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("x")) }))
	defer elsewhere.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, elsewhere.URL+"/x", http.StatusFound)
	}))
	defer redirect.Close()
	loop := httptest.NewServer(nil)
	loop.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, loop.URL+"/again", http.StatusFound)
	})
	defer loop.Close()
	dl := func(allowHTTP bool, hosts ...string) *Download {
		return NewDownload(nil, allowHTTP, time.Minute, nil, hosts...)
	}

	if err := get(dl(true, hostOf(redirect)), redirect.URL); err == nil || !strings.Contains(err.Error(), "redirect refused") || !strings.Contains(err.Error(), "not a host") {
		t.Errorf("a redirect to a host outside the purpose must be refused: %v", err)
	}
	if err := get(dl(true, hostOf(redirect), hostOf(elsewhere)), redirect.URL); err != nil {
		t.Errorf("a redirect to a host the purpose names is followed: %v", err)
	}
	if err := get(dl(true, hostOf(redirect)), elsewhere.URL); err == nil || !strings.Contains(err.Error(), "not a host") {
		t.Errorf("a request to a host outside the purpose must be refused: %v", err)
	}
	if err := get(dl(false, hostOf(elsewhere)), elsewhere.URL); err == nil || !strings.Contains(err.Error(), "not https") {
		t.Errorf("plain http must be refused: %v", err)
	}
	if err := get(dl(true, hostOf(loop)), loop.URL); err == nil || !strings.Contains(err.Error(), "too many redirects") {
		t.Errorf("a redirect loop must stop at the bound: %v", err)
	}
}

func TestDownloadHoldsTheAddressPolicyOnEveryHop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("x")) }))
	defer srv.Close()
	d := NewDownload(BlockedIP, true, time.Minute, nil, hostOf(srv))
	if err := get(d, srv.URL); err == nil || !strings.Contains(err.Error(), "refusing to connect") {
		t.Errorf("a loopback dial must be refused under the shipped policy: %v", err)
	}
}

func TestDownloadHonoursNoProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	tr, ok := NewDownload(nil, false, time.Minute, nil).Client.Transport.(*http.Transport)
	if !ok || tr.Proxy != nil {
		t.Fatalf("the download's transport must name no proxy: %#v", tr)
	}
}

func TestOverrideFreeRootsPutsTheEnvironmentBack(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "/nonexistent/ca.pem")
	_ = OverrideFreeRoots()
	if got, _ := os.LookupEnv("SSL_CERT_FILE"); got != "/nonexistent/ca.pem" {
		t.Errorf("SSL_CERT_FILE = %q after the load, want it put back", got)
	}
}

func TestDownloadHoldsTheAddressPolicyOnARedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "http://127.0.0.2:1/x", http.StatusFound)
	}))
	defer srv.Close()
	// The policy blocks only the redirect's target, so the first hop dials and
	// the refusal comes from the redirect check, before any dial to it.
	blocked := func(ip net.IP) bool { return ip.Equal(net.ParseIP("127.0.0.2")) }
	d := NewDownload(blocked, true, time.Minute, nil, hostOf(srv), "127.0.0.2:1")
	if err := get(d, srv.URL); err == nil || !strings.Contains(err.Error(), "redirect refused") || !strings.Contains(err.Error(), "127.0.0.2") {
		t.Errorf("a redirect to a blocked address must be refused at the redirect: %v", err)
	}
}

func TestDownloadTrustsTheRootPoolItIsGiven(t *testing.T) {
	pool := x509.NewCertPool()
	tr, ok := NewDownload(nil, false, time.Minute, pool).Client.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs != pool {
		t.Fatalf("the download must trust exactly the pool it was given: %#v", tr)
	}
}

func TestSiblingSharesTheTransportAndNotTheHosts(t *testing.T) {
	d := NewDownload(BlockedIP, false, time.Minute, x509.NewCertPool(), "a.example")
	s := d.Sibling("b.example")
	if s.Client.Transport != d.Client.Transport || s.Client.Timeout != d.Client.Timeout {
		t.Error("a sibling must use the same pinned transport and time bound")
	}
	for dl, want := range map[*Download][2]bool{d: {true, false}, s: {false, true}} {
		for i, h := range []string{"a.example", "b.example"} {
			if got := dl.Admit(&url.URL{Scheme: "https", Host: h}) == nil; got != want[i] {
				t.Errorf("admit %s = %v, want %v", h, got, want[i])
			}
		}
	}
	if err := s.Admit(&url.URL{Scheme: "http", Host: "b.example"}); err == nil {
		t.Error("a sibling of an https-only download must refuse plain http")
	}
}
