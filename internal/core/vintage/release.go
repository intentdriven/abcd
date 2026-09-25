package vintage

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/urlguard"
)

// latestReleaseURL is GitHub's stable "latest release" redirector: it answers a
// 302 to /releases/tag/<tag>. Reading the tag off that redirect is exactly how
// hooks/bootstrap.sh discovers the release to pin, so `update --check` and the
// provisioner agree on what "latest" means.
const latestReleaseURL = "https://github.com/intentdriven/abcd/releases/latest"

// releaseCheckTimeout bounds the one attempt update --check makes.
const releaseCheckTimeout = 15 * time.Second

// ReleaseFetcher fetches the latest published release tag. It is the seam behind
// the network provider — the one network touch itd-111 adds (adr-38 tier 2),
// reached only by an explicit `abcd update --check`. A disk path never
// constructs one.
type ReleaseFetcher interface {
	LatestTag() (string, error)
}

// ReleaseProvider adapts a ReleaseFetcher to the comparator's Provider
// interface, so the explicit check compares through the same Compare as every
// disk path. A fetch failure is an unresolved reference (Unknown), never a
// silent match.
func ReleaseProvider(f ReleaseFetcher) Provider { return releaseProvider{f} }

type releaseProvider struct{ f ReleaseFetcher }

func (p releaseProvider) Expected() Expected {
	tag, err := p.f.LatestTag()
	if err != nil || tag == "" {
		return Expected{Known: false}
	}
	return Expected{Revision: tag, Known: true}
}

// githubReleaseFetcher reads the latest release tag off GitHub's redirect, under
// the same urlguard SSRF policy the citation checker runs. It never follows the
// redirect and never reads a body: the tag is in the Location header.
type githubReleaseFetcher struct {
	url     string
	blocked func(net.IP) bool
	client  *http.Client
}

// NewGitHubReleaseFetcher builds the release fetcher as it ships: the full
// urlguard SSRF policy and a bounded timeout.
func NewGitHubReleaseFetcher() ReleaseFetcher {
	return newGitHubReleaseFetcher(latestReleaseURL, urlguard.BlockedIP, releaseCheckTimeout)
}

// newGitHubReleaseFetcher builds a fetcher with an explicit URL, address policy,
// and timeout. The policy is a parameter for exactly the reason cite's is: it
// lets the fetch path be exercised against an httptest server (loopback) without
// relaxing what NewGitHubReleaseFetcher ships.
func newGitHubReleaseFetcher(rawURL string, blocked func(net.IP) bool, timeout time.Duration) *githubReleaseFetcher {
	dialer := &net.Dialer{
		Timeout: timeout,
		// The connect-time re-check closes the DNS-rebinding gap between the name
		// guard in LatestTag and the transport's own resolution.
		Control: urlguard.DialControl(blocked),
	}
	return &githubReleaseFetcher{
		url:     rawURL,
		blocked: blocked,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:         dialer.DialContext,
				TLSHandshakeTimeout: timeout,
			},
			// Do not follow the redirect: the tag is in the first hop's Location.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// tagFromLocation extracts <tag> from a /releases/tag/<tag> redirect target.
var tagFromLocation = regexp.MustCompile(`/releases/tag/([^/?#]+)`)

// LatestTag performs one bounded attempt and reads the tag out of the redirect.
func (f *githubReleaseFetcher) LatestTag() (string, error) {
	parsed, err := url.Parse(f.url)
	if err != nil {
		return "", err
	}
	if err := urlguard.CheckHostWith(parsed.Hostname(), f.blocked); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "abcd-version-check")
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	// The body is never read: the tag is in the Location header.
	defer resp.Body.Close()
	m := tagFromLocation.FindStringSubmatch(resp.Header.Get("Location"))
	if m == nil {
		return "", errors.New("the latest-release redirect did not name a tag")
	}
	return m[1], nil
}
