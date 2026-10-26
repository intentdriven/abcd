package tools

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/urlguard"
)

// The download transport holds to the rules the updater's own fetch holds to
// (internal/core/update): https only, the environment never chooses where the
// client connects (no proxy, the CA overrides held out of the root pool),
// urlguard's address policy on every dial and every redirect, a redirect only
// to a host the request's purpose names, and a bound on time and on size.
// It is a second copy of that policy rather than a call into it because
// update imports ahoy, which imports this package; the two are held to the
// same words by their tests rather than by one function.

const (
	// maxArchiveBytes bounds one tool download. gh's largest asset is about
	// 20 MiB and gitleaks' about 8 MiB.
	maxArchiveBytes = 256 << 20
	// maxBundleBytes bounds one attestation answer or bundle.
	maxBundleBytes = 4 << 20
	// downloadTimeout bounds one request, connection to last byte.
	downloadTimeout = 5 * time.Minute
	maxRedirects    = 8
)

// releaseHosts are the hosts a release download may reach: the release page
// itself and the asset hosts GitHub redirects a download to.
var releaseHosts = []string{"github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com"}

// apiHosts are the hosts a provenance check may reach: GitHub's attestations
// API, the blob store it hands a bundle out of when it does not inline it, and
// the Sigstore TUF repository the trust root is read from. Every byte read
// from them is verified by Sigstore before it is believed, so they choose
// nothing that runs.
var apiHosts = []string{"api.github.com", "tmaproduction.blob.core.windows.net", "tuf-repo-cdn.sigstore.dev"}

// fetcher performs bounded GETs under the pinned transport.
type fetcher struct {
	client    *http.Client
	allowHTTP bool // test seam: the shipping constructor never sets it
	hosts     map[string]bool
}

// errNotFound404 is a 404 answer.
type errNotFound404 struct{ url string }

func (e *errNotFound404) Error() string { return "not found: " + e.url }

// newFetcher builds the transport. blocked is the address policy (nil allows
// every address, for a loopback test server); hosts are the only hosts a
// request or a redirect may name.
func newFetcher(blocked func(net.IP) bool, allowHTTP bool, hosts ...string) *fetcher {
	f := &fetcher{allowHTTP: allowHTTP, hosts: map[string]bool{}}
	for _, h := range hosts {
		f.hosts[h] = true
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	if blocked != nil {
		dialer.Control = urlguard.DialControl(blocked)
	}
	f.client = &http.Client{
		Timeout: downloadTimeout,
		Transport: &http.Transport{
			Proxy:               nil, // by construction: the environment never chooses the route
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: overrideFreeRoots()},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if err := f.admitURL(req.URL); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			if blocked != nil {
				if err := urlguard.CheckHostWith(req.URL.Hostname(), blocked); err != nil {
					return fmt.Errorf("redirect refused: %w", err)
				}
			}
			return nil
		},
	}
	return f
}

// productionFetcher is the shipping transport for a purpose's hosts.
func productionFetcher(hosts []string) *fetcher {
	return newFetcher(urlguard.BlockedIP, false, hosts...)
}

func (f *fetcher) admitURL(u *url.URL) error {
	if u.Scheme != "https" && !(f.allowHTTP && u.Scheme == "http") {
		return fmt.Errorf("%s is not https", u.Redacted())
	}
	if !f.hosts[u.Host] {
		return fmt.Errorf("%s is not a host this download may reach", u.Host)
	}
	return nil
}

// get reads rawURL whole, refusing an answer over limit bytes.
func (f *fetcher) get(ctx context.Context, rawURL string, limit int64, headers map[string]string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if err := f.admitURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "abcd-tools")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, &errNotFound404{url: u.Redacted()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", u.Redacted(), resp.Status)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("GET %s: %d bytes exceeds the %d-byte bound", u.Redacted(), resp.ContentLength, limit)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("GET %s: more than the %d-byte bound", u.Redacted(), limit)
	}
	return b, nil
}

// caOverrides are the variables crypto/x509 reads when it loads the system
// root pool.
var caOverrides = []string{"SSL_CERT_FILE", "SSL_CERT_DIR"}

// overrideFreeRoots loads the system root pool with the CA overrides out of
// the environment for the load alone, then puts back exactly what it took
// out: the same rule as the updater's, so an environment variable never
// chooses which certificate authorities a tool download trusts. A nil pool
// falls back to the platform default.
func overrideFreeRoots() *x509.CertPool {
	taken := map[string]string{}
	for _, name := range caOverrides {
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
