package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/intentdriven/abcd/internal/urlguard"
)

const (
	// maxArchiveBytes bounds one tool download. gh's largest asset is about
	// 20 MiB and gitleaks' about 8 MiB.
	maxArchiveBytes = 256 << 20
	// maxBundleBytes bounds one attestation answer or bundle.
	maxBundleBytes = 4 << 20
	// downloadTimeout bounds one request, connection to last byte.
	downloadTimeout = 5 * time.Minute
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

// fetcher performs bounded GETs under urlguard's Download, the one transport
// rule the updater's own fetch (internal/core/update) holds to as well.
type fetcher struct{ dl *urlguard.Download }

// errNotFound404 is a 404 answer.
type errNotFound404 struct{ url string }

func (e *errNotFound404) Error() string { return "not found: " + e.url }

// newFetcher builds the transport. blocked is the address policy (nil allows
// every address, for a loopback test server); hosts are the only hosts a
// request or a redirect may name.
func newFetcher(blocked func(net.IP) bool, allowHTTP bool, hosts ...string) *fetcher {
	return &fetcher{dl: urlguard.NewDownload(blocked, allowHTTP, downloadTimeout, urlguard.OverrideFreeRoots(), hosts...)}
}

// productionFetcher is the shipping transport for a purpose's hosts.
func productionFetcher(hosts []string) *fetcher {
	return newFetcher(urlguard.BlockedIP, false, hosts...)
}

// get reads rawURL whole, refusing an answer over limit bytes.
func (f *fetcher) get(ctx context.Context, rawURL string, limit int64, headers map[string]string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if err := f.dl.Admit(u); err != nil {
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
	resp, err := f.dl.Client.Do(req)
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
