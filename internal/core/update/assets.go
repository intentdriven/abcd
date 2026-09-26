package update

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/intentdriven/abcd/internal/urlguard"
)

// ReleaseAssets fetches named assets of one repository's releases under the
// updater's transport policy: the pinned origin, https only, the urlguard
// address policy, GitHub's own asset hosts as the only legal redirects, and
// the proxy and CA overrides ignored by its own client. It is the fetcher
// `abcd launch`'s parity diff reads a previous release's payload archive and
// checksums.txt through, and only when the operator asks for it (adr-38 tier
// 2). It verifies nothing itself: the caller checks the archive against the
// release's own checksums.
type ReleaseAssets struct{ u *Updater }

// githubOriginRe is the only origin the shipping constructor accepts.
var githubOriginRe = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// NewReleaseAssets builds the fetcher for the repository at origin, an
// https://github.com/<owner>/<repo> address.
func NewReleaseAssets(origin string) (*ReleaseAssets, error) {
	if !githubOriginRe.MatchString(origin) {
		return nil, fmt.Errorf("the release origin %q is not an https://github.com/<owner>/<repo> address", origin)
	}
	a := newReleaseAssets(origin, urlguard.BlockedIP, false)
	a.u.redirectHost["objects.githubusercontent.com"] = true
	a.u.redirectHost["release-assets.githubusercontent.com"] = true
	return a, nil
}

// newReleaseAssets is the test-reachable constructor, mirroring newUpdater's
// seam: an explicit origin, address policy and scheme policy.
//
// Unlike the updater it does not unset the overrides in the process: the
// launch preview that builds it is a read-only verb, and its client needs no
// process-wide scrub to ignore them. The proxy is nil by construction, and the
// root pool is loaded with the CA overrides held out for the load alone
// (overrideFreeRoots). The names that were set are kept for EnvIgnored, so
// the caller can say what the fetch did not honour (iss-2609251902444497).
func newReleaseAssets(origin string, blocked func(net.IP) bool, allowHTTP bool) *ReleaseAssets {
	ignored := setOverrides()
	return &ReleaseAssets{u: buildUpdater(origin, blocked, "", allowHTTP, nil, ignored, overrideFreeRoots())}
}

// FetchReleaseAsset downloads one asset of tag. found is false, with no error,
// when the release or the asset does not exist; url is what was requested.
func (a *ReleaseAssets) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	if !tagShape.MatchString(tag) {
		return nil, "", false, fmt.Errorf("%q is not a release tag", tag)
	}
	url := a.u.assetURL(tag, name)
	body, _, err := a.u.get(url, maxAssetBytes)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return nil, url, false, nil
		}
		return nil, url, false, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, url, false, err
	}
	return data, url, true, nil
}

// EnvIgnored names the transport-override variables that were set when the
// fetcher was built and that its client does not honour.
func (a *ReleaseAssets) EnvIgnored() []string { return a.u.envIgnored }
