package urlguard

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Download is the one transport rule every download of release bytes holds
// to, abcd's own update path and the tool installer alike: https only, the
// environment never chooses where the client connects (no proxy, and a root
// pool the caller loaded with the CA overrides held out), the address policy
// on every dial and every redirect, every request and every redirect only to
// a host the download's purpose names, and a bound on redirects and on time.
// A size bound is the caller's, since it differs by what is read.
type Download struct {
	// Client is the pinned client. A request made through it must first pass
	// Admit: the client checks redirects, not the URL it was handed.
	Client    *http.Client
	hosts     map[string]bool
	blocked   func(net.IP) bool
	allowHTTP bool // test seam: no shipping constructor sets it
}

// MaxRedirects bounds the redirects one download follows.
const MaxRedirects = 8

// NewDownload builds the transport. blocked is the address policy (nil allows
// every address, for a loopback test server); rootCAs is the trust pool (nil
// is the platform default); hosts are the only hosts a request or a redirect
// may name.
func NewDownload(blocked func(net.IP) bool, allowHTTP bool, timeout time.Duration, rootCAs *x509.CertPool, hosts ...string) *Download {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	if blocked != nil {
		dialer.Control = DialControl(blocked)
	}
	transport := &http.Transport{
		Proxy:               nil, // by construction: the environment never chooses the route
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 30 * time.Second,
		TLSClientConfig:     &tls.Config{RootCAs: rootCAs},
	}
	return newDownload(transport, timeout, blocked, allowHTTP, hosts)
}

// Sibling is a download for another purpose over the same pinned transport,
// root pool and address policy, reaching only hosts: a purpose's hosts are
// its own, so one purpose's requests and redirects never reach another's.
func (d *Download) Sibling(hosts ...string) *Download {
	return newDownload(d.Client.Transport, d.Client.Timeout, d.blocked, d.allowHTTP, hosts)
}

func newDownload(transport http.RoundTripper, timeout time.Duration, blocked func(net.IP) bool, allowHTTP bool, hosts []string) *Download {
	d := &Download{hosts: map[string]bool{}, blocked: blocked, allowHTTP: allowHTTP}
	for _, h := range hosts {
		d.AllowHost(h)
	}
	d.Client = &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return errors.New("too many redirects")
			}
			if err := d.Admit(req.URL); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			if blocked != nil {
				if err := CheckHostWith(req.URL.Hostname(), blocked); err != nil {
					return fmt.Errorf("redirect refused: %w", err)
				}
			}
			return nil
		},
	}
	return d
}

// AllowHost adds a host a request or a redirect may name. It is called while
// the download is being set up, before its first request: the host set is
// read, unlocked, by the client's redirect check.
func (d *Download) AllowHost(host string) { d.hosts[host] = true }

// Admit refuses a URL that is not https or names a host outside the
// download's purpose.
func (d *Download) Admit(u *url.URL) error {
	if u.Scheme != "https" && !(d.allowHTTP && u.Scheme == "http") {
		return fmt.Errorf("%s is not https", u.Redacted())
	}
	if !d.hosts[u.Host] {
		return fmt.Errorf("%s is not a host this download may reach", u.Host)
	}
	return nil
}

// CAOverrides are the variables crypto/x509 reads when it loads the system
// root pool.
var CAOverrides = []string{"SSL_CERT_FILE", "SSL_CERT_DIR"}

// OverrideFreeRoots loads the system root pool with the CA overrides out of
// the environment for the load alone, then puts back exactly what it took
// out, so an environment variable never chooses which certificate
// authorities a download trusts and the caller's environment is left as it
// was. crypto/x509 loads that pool once per process: when this is the first
// load, the process's default pool ignores the overrides from then on too,
// while the variables stay set for anything the process starts. A nil pool
// (SystemCertPool error) falls back to the platform default.
func OverrideFreeRoots() *x509.CertPool {
	taken := map[string]string{}
	for _, name := range CAOverrides {
		if v, ok := os.LookupEnv(name); ok {
			taken[name] = v
			os.Unsetenv(name)
		}
	}
	defer func() {
		for name, v := range taken {
			os.Setenv(name, v)
		}
	}()
	if pool, err := x509.SystemCertPool(); err == nil {
		return pool
	}
	return nil
}
