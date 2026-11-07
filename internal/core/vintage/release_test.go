package vintage

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// allowLoopback relaxes only the loopback rule so the guarded client can reach an
// httptest server, inheriting every other SSRF refusal unchanged — the same seam
// the citation checker's tests use.
func allowLoopback(ip net.IP) bool { return false }

func TestGitHubReleaseFetcherReadsTagFromRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/intentdriven/abcd/releases/tag/v2.0.0", http.StatusFound)
	}))
	defer srv.Close()

	f := newGitHubReleaseFetcher(srv.URL, allowLoopback, 5*time.Second)
	tag, err := f.LatestTag()
	if err != nil {
		t.Fatalf("LatestTag: %v", err)
	}
	if tag != "v2.0.0" {
		t.Fatalf("tag = %q, want v2.0.0", tag)
	}
}

func TestGitHubReleaseFetcherNoTagIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/somewhere/else", http.StatusFound)
	}))
	defer srv.Close()

	f := newGitHubReleaseFetcher(srv.URL, allowLoopback, 5*time.Second)
	if _, err := f.LatestTag(); err == nil {
		t.Fatal("expected an error when the redirect names no tag")
	}
}

// stubFetcher answers a fixed tag/error, standing in for the network on the
// disk paths' side of the comparator.
type stubFetcher struct {
	tag string
	err error
}

func (s stubFetcher) LatestTag() (string, error) { return s.tag, s.err }

func TestReleaseProviderComparison(t *testing.T) {
	t.Run("running version matches latest is fresh", func(t *testing.T) {
		rep := Compare(Current{Revision: "v2.0.0", Known: true}, ReleaseProvider(stubFetcher{tag: "v2.0.0"}))
		if rep.Outcome != Fresh {
			t.Fatalf("outcome = %q, want fresh", rep.Outcome)
		}
	})
	t.Run("newer latest than running is stale", func(t *testing.T) {
		rep := Compare(Current{Revision: "v1.0.0", Known: true}, ReleaseProvider(stubFetcher{tag: "v2.0.0"}))
		if rep.Outcome != Stale {
			t.Fatalf("outcome = %q, want stale", rep.Outcome)
		}
	})
	t.Run("fetch failure is an unresolved reference", func(t *testing.T) {
		rep := Compare(Current{Revision: "v1.0.0", Known: true}, ReleaseProvider(stubFetcher{err: http.ErrHandlerTimeout}))
		if rep.Outcome != Unknown {
			t.Fatalf("outcome = %q, want unknown", rep.Outcome)
		}
	})
}

// TestTheDrivesReleaseClientIsAShortCheck: the one request a drive makes each interval
// is held to the check's own bound, so a dead network costs a lane boundary no more.
func TestTheDrivesReleaseClientIsAShortCheck(t *testing.T) {
	f, ok := NewGitHubReleaseFetcherWithRoots(nil).(*githubReleaseFetcher)
	if !ok {
		t.Fatal("not the release fetcher")
	}
	if f.client.Timeout != releaseCheckTimeout || releaseCheckTimeout > 15*time.Second {
		t.Fatalf("the check's bound is %s", f.client.Timeout)
	}
}
