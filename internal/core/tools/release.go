package tools

import (
	"fmt"
	"regexp"
	"strconv"
)

// Release is the one build of a tool abcd installs, fixed in abcd's own
// release (brief invariant 7: "provisioning fetches only a manifest-named,
// pinned, checksum-verified artifact. No version-discovery request exists
// anywhere in abcd"). Every field is compiled in: the download URL is spelled
// out in full, the asset's sha256 is pinned beside it, and nothing about it is
// read from the network, a flag, a repository file or the environment. A tool
// therefore updates only when abcd does (dcn-2610240845548776, and the
// intent's third decision).
type Release struct {
	// Repo is the tool's GitHub repository, owner/name.
	Repo string
	// Version is the pinned release, without its leading v.
	Version string
	// Assets is the download for each platform abcd supports, keyed
	// "<GOOS>/<GOARCH>". A platform with no entry has no install.
	Assets map[string]Asset
	// Signer is who must have signed the asset's SLSA build provenance, for a
	// tool whose makers publish one. nil is a tool that publishes no signed
	// build record: its download is checked by checksum only, and every
	// explanation and result says so.
	Signer *Signer
	// Compatible is the range of the tool's versions abcd works with. A copy
	// already installed is used, and nothing installed, only when its version
	// falls inside it (the intent's third criterion). The pinned Version must
	// fall inside it too (TestEveryPinnedReleaseIsCompatible), which is the
	// hook a release-time compatibility check moves (iss-2610240959030370).
	// The version is read from the output of the tool's Verify argv.
	Compatible VersionRange
}

// Asset is one platform's download.
type Asset struct {
	// URL is the release asset's full https://github.com/<repo>/releases/download/
	// address: no "latest", no redirect a release page could move.
	URL string
	// SHA256 is the asset's digest, copied from the release's published
	// checksums file and checked against the downloaded bytes when it was
	// pinned. It is the check every download must pass before anything is
	// written.
	SHA256 string
	// Archive is the asset's packaging.
	Archive Archive
	// Member is the program's path inside the archive.
	Member string
}

// Archive is how an asset is packaged.
type Archive string

const (
	// TarGz is a gzip-compressed tar archive.
	TarGz Archive = "tar.gz"
	// Zip is a zip archive.
	Zip Archive = "zip"
)

// Signer is a build-provenance policy: the certificate identity a Sigstore
// bundle over the asset must carry. It is pinned exactly (no prefix match):
// the repository, the workflow file inside it, and the ref that workflow ran
// from. The digest is pinned in abcd's own source, so the provenance is not
// bound to a commit: it proves the pinned bytes were built by the makers'
// workflow, which a release page edit cannot forge (provenance.go's rationale).
type Signer struct {
	Repo     string // owner/name
	Workflow string // path of the signing workflow inside the repository
	Ref      string // the git ref the signing workflow ran from
}

// SAN is the certificate's subject alternative name the policy requires.
func (s Signer) SAN() string {
	return "https://github.com/" + s.Repo + "/" + s.Workflow + "@" + s.Ref
}

// VersionRange is [Min, Below): Min inclusive, Below exclusive. Below empty
// means no upper bound.
type VersionRange struct {
	Min   string
	Below string
}

// String is the range as a person reads it.
func (r VersionRange) String() string {
	if r.Below == "" {
		return r.Min + " or later"
	}
	return r.Min + " or later, below " + r.Below
}

// Contains reports whether v lies in the range. A version that does not parse
// lies in no range.
func (r VersionRange) Contains(v string) bool {
	got, ok := parseVersion(v)
	if !ok {
		return false
	}
	if lo, ok := parseVersion(r.Min); !ok || compareVersion(got, lo) < 0 {
		return false
	}
	if r.Below == "" {
		return true
	}
	hi, ok := parseVersion(r.Below)
	return ok && compareVersion(got, hi) < 0
}

var versionShape = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// parseVersion reads the first major.minor.patch in s.
func parseVersion(s string) ([3]int, bool) {
	m := versionShape.FindStringSubmatch(s)
	if m == nil {
		return [3]int{}, false
	}
	var v [3]int
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}

func compareVersion(a, b [3]int) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

// versionIn is the first major.minor.patch in a tool's version output, or "".
func versionIn(out string) string {
	v, ok := parseVersion(out)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
}

// Checks says how far a download of this release is checked, in the words a
// person reads: always the sha256 abcd's release fixes, and the signed build
// record only where the makers publish one.
func (r Release) Checks() string {
	if r.Signer == nil {
		return "checked by checksum only: the sha256 abcd's release fixes for it (" + r.Repo +
			" publishes no signed build record to check it against)"
	}
	return "checked against the sha256 abcd's release fixes for it, and against its signed build record (" +
		r.Signer.SAN() + ")"
}

// platform is the Assets key for a GOOS and GOARCH.
func platform(goos, goarch string) string { return goos + "/" + goarch }

// releaseDownload is the GitHub release download URL for repo's tag and file:
// the one URL shape the table holds (TestEveryAssetURLIsAPinnedReleaseDownload).
func releaseDownload(repo, tag, file string) string {
	return "https://github.com/" + repo + "/releases/download/" + tag + "/" + file
}

// gitleaksRelease is gitleaks v8.30.1 (2026-03-21). The sha256 values were
// copied on 2026-10-24 from the release's gitleaks_8.30.1_checksums.txt and
// each matched the downloaded asset. gitleaks signs nothing (its
// .goreleaser.yml has no signing step and no attestation is published), so a
// download is checked by checksum only, and setup says so. Compatible starts
// at 8.19.0, the release that added the `dir` command the adapter runs
// (internal/adapter/gitleaks/runner.go), and stops below 9.0.0, the next
// major release, whose report format abcd has not read.
var gitleaksRelease = func() Release {
	const repo, v = "gitleaks/gitleaks", "8.30.1"
	asset := func(file, sum string) Asset {
		return Asset{URL: releaseDownload(repo, "v"+v, file), SHA256: sum, Archive: TarGz, Member: "gitleaks"}
	}
	return Release{
		Repo:    repo,
		Version: v,
		Assets: map[string]Asset{
			"darwin/arm64": asset("gitleaks_8.30.1_darwin_arm64.tar.gz", "b40ab0ae55c505963e365f271a8d3846efbc170aa17f2607f13df610a9aeb6a5"),
			"darwin/amd64": asset("gitleaks_8.30.1_darwin_x64.tar.gz", "dfe101a4db2255fc85120ac7f3d25e4342c3c20cf749f2c20a18081af1952709"),
			"linux/arm64":  asset("gitleaks_8.30.1_linux_arm64.tar.gz", "e4a487ee7ccd7d3a7f7ec08657610aa3606637dab924210b3aee62570fb4b080"),
			"linux/amd64":  asset("gitleaks_8.30.1_linux_x64.tar.gz", "551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb"),
		},
		Compatible: VersionRange{Min: "8.19.0", Below: "9.0.0"},
	}
}()

// ghRelease is gh v2.103.2 (2026-10-23). The sha256 values were copied on
// 2026-10-24 from the release's gh_2.103.2_checksums.txt and each matched the
// downloaded asset. cli/cli signs SLSA build provenance over every release
// asset from .github/workflows/deployment.yml on refs/heads/trunk (the
// certificate of every recorded bundle names it), so a download is checked
// against that too. Compatible starts at 2.102.0, the release that made `gh
// attestation verify --source-ref` compare the ref exactly rather than
// ignoring case: abcd's by-hand fallback for its own binary relies on that
// flag (internal/core/ahoy/owned_copy.go). It stops below 3.0.0.
var ghRelease = func() Release {
	const repo, v = "cli/cli", "2.103.2"
	asset := func(file, sum string, a Archive, dir string) Asset {
		return Asset{URL: releaseDownload(repo, "v"+v, file), SHA256: sum, Archive: a, Member: dir + "/bin/gh"}
	}
	return Release{
		Repo:    repo,
		Version: v,
		Assets: map[string]Asset{
			"darwin/arm64": asset("gh_2.103.2_macOS_arm64.zip", "5416000f5d17580a187339e0c4068a01ca5c87f7d4fe6ca64933696f967bc418", Zip, "gh_2.103.2_macOS_arm64"),
			"darwin/amd64": asset("gh_2.103.2_macOS_amd64.zip", "2dc7f1fc4761f96732995f6e4f4c6b4d705f2635da4126620c8b30eaa67b9c98", Zip, "gh_2.103.2_macOS_amd64"),
			"linux/arm64":  asset("gh_2.103.2_linux_arm64.tar.gz", "4c3e99a50bd8b181a3fe9d69b25dd978bac768148d88f01507c6b4d1ef61a443", TarGz, "gh_2.103.2_linux_arm64"),
			"linux/amd64":  asset("gh_2.103.2_linux_amd64.tar.gz", "62ccf747115ddc61237efffdde66a3c8f70ba249433dc64f70801089b59c59c8", TarGz, "gh_2.103.2_linux_amd64"),
		},
		Signer:     &Signer{Repo: repo, Workflow: ".github/workflows/deployment.yml", Ref: "refs/heads/trunk"},
		Compatible: VersionRange{Min: "2.102.0", Below: "3.0.0"},
	}
}()

// assetFor is the release's download for a platform. The scheme is the
// fetcher's to hold to https (fetch.go), and the table's URLs are held to
// GitHub release downloads by TestEveryPinnedReleaseIsComplete.
func (r Release) assetFor(goos, goarch string) (Asset, bool) {
	a, ok := r.Assets[platform(goos, goarch)]
	return a, ok && a.URL != ""
}
