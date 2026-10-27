package update

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/urlguard"
)

// The updater downloads under urlguard's Download, the rule the tool
// installer holds to as well: every request, not only every redirect, names
// a host the release's purpose does.

func TestUpdaterRequestsOnlyItsReleaseHosts(t *testing.T) {
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	defer elsewhere.Close()
	o := newTestOrigin(t)
	u := testUpdater(t, o)

	if _, _, err := u.get(elsewhere.URL+"/x", 1<<20); err == nil || !strings.Contains(err.Error(), "not a host this download may reach") {
		t.Errorf("a request to a host outside the release origin must be refused: %v", err)
	}
}

// Each purpose reaches its own hosts: a release download the release origin
// and GitHub's asset hosts, the provenance check the attestations API and the
// Sigstore TUF repository, so neither's requests or redirects reach the
// other's.
func TestShippingDownloadsReachOnlyTheirPurposesHosts(t *testing.T) {
	release := []string{"github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com"}
	check := []string{"api.github.com", "tuf-repo-cdn.sigstore.dev"}
	never := []string{"example.com", "github.com.example.com", "tmaproduction.blob.core.windows.net"}
	hold := func(name string, dl *urlguard.Download, reach, refuse []string) {
		t.Helper()
		for _, h := range reach {
			if err := dl.Admit(&url.URL{Scheme: "https", Host: h}); err != nil {
				t.Errorf("%s must reach %s: %v", name, h, err)
			}
		}
		for _, h := range append(refuse, never...) {
			if err := dl.Admit(&url.URL{Scheme: "https", Host: h}); err == nil {
				t.Errorf("%s must not reach %s", name, h)
			}
		}
		if err := dl.Admit(&url.URL{Scheme: "http", Host: reach[0]}); err == nil {
			t.Errorf("%s must not fetch plain http", name)
		}
	}

	u := NewGitHubUpdater()
	hold("the updater's release download", u.dl, release, check)
	g, ok := u.provenance.(*githubAttestations)
	if !ok {
		t.Fatalf("the shipping updater's provenance check is %T", u.provenance)
	}
	hold("the updater's provenance check", g.dl, check, release)
	hold("ahoy's provenance check", newReleaseProvenance(releaseOrigin, urlguard.BlockedIP, false).dl, check, release)
	a, err := NewReleaseAssets(releaseOrigin)
	if err != nil {
		t.Fatal(err)
	}
	hold("launch's release-asset fetch", a.u.dl, release, check)
}
