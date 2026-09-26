package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestReleaseAssetsFetchesNamedAssets: the release-asset fetcher the launch
// parity baseline uses reads one named asset of one tag from the pinned origin,
// reports a missing asset as not found rather than an error, and refuses a tag
// that is not tag-shaped before any request is built. The server is loopback;
// no test reaches the network.
func TestReleaseAssetsFetchesNamedAssets(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/releases/download/v1.2.3/checksums.txt" {
			_, _ = w.Write([]byte("sums\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	a := newReleaseAssets(srv.URL, nil, true)
	data, url, found, err := a.FetchReleaseAsset("v1.2.3", "checksums.txt")
	if err != nil || !found || string(data) != "sums\n" || url != srv.URL+"/releases/download/v1.2.3/checksums.txt" {
		t.Fatalf("a present asset must be returned with its URL, got %q %q %v %v", data, url, found, err)
	}
	if _, _, found, err := a.FetchReleaseAsset("v1.2.3", "absent.zip"); err != nil || found {
		t.Errorf("a missing asset is not found, not an error: %v %v", found, err)
	}
	before := len(paths)
	if _, _, _, err := a.FetchReleaseAsset("../v1", "x"); err == nil || !strings.Contains(err.Error(), "not a release tag") {
		t.Errorf("a path-shaped tag must be refused, got %v", err)
	}
	if len(paths) != before {
		t.Error("a refused tag must build no request")
	}
}

// TestNewReleaseAssetsPinsAGitHubOrigin: the shipping constructor accepts only
// an https GitHub repository address.
func TestNewReleaseAssetsPinsAGitHubOrigin(t *testing.T) {
	if _, err := NewReleaseAssets("https://github.com/example/abcd"); err != nil {
		t.Errorf("a GitHub repository is a valid origin: %v", err)
	}
	for _, bad := range []string{"http://github.com/example/abcd", "https://example.com/abcd", "https://github.com/example/abcd/releases"} {
		if _, err := NewReleaseAssets(bad); err == nil {
			t.Errorf("%s must be refused", bad)
		}
	}
}

// TestReleaseAssetsScopesTheScrubToItsOwnClient: the launch parity fetcher
// ignores the proxy and CA overrides, as the updater does, but only for its own
// client: the process environment is left as it was found, and the names it
// ignored are returned so the preview can say so (iss-2609251902444497). The
// fetch still connects directly: a proxy at an unroutable address is never
// dialled.
func TestReleaseAssetsScopesTheScrubToItsOwnClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("sums\n"))
	}))
	defer srv.Close()
	set := map[string]string{
		"HTTPS_PROXY":   "http://192.0.2.1:3128",
		"http_proxy":    "http://192.0.2.1:3128",
		"SSL_CERT_FILE": "/nonexistent/ca.pem",
	}
	for k, v := range set {
		t.Setenv(k, v)
	}

	a := newReleaseAssets(srv.URL, nil, true)
	for k, v := range set {
		if got, ok := os.LookupEnv(k); !ok || got != v {
			t.Errorf("%s must be left in the process environment, got %q (set %v)", k, got, ok)
		}
	}
	ignored := strings.Join(a.EnvIgnored(), ",")
	for k := range set {
		if !strings.Contains(ignored, k) {
			t.Errorf("the fetcher must name %s as ignored, got %q", k, ignored)
		}
	}
	if data, _, found, err := a.FetchReleaseAsset("v1.2.3", "checksums.txt"); err != nil || !found || string(data) != "sums\n" {
		t.Fatalf("the fetch must connect directly, whatever the proxy variables say: %q %v %v", data, found, err)
	}
}
