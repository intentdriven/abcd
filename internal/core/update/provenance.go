package update

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/urlguard"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// Build provenance is the check that does not read its trust from the release
// page (iss-2609231050273096, ruling E5 of 2026-10-11). checksums.txt is
// published beside the binary, so whoever can replace one can replace both and
// the same-page check still passes. The release workflow also signs SLSA
// build provenance over every asset (release.yml, actions/attest): a Sigstore
// bundle whose certificate Fulcio issued to this repository's release.yml run,
// logged in Rekor. Verifying that bundle, in process with sigstore-go, against
// the Sigstore trusted root fetched through TUF, proves the bytes were built by
// release.yml in intentdriven/abcd — something a release-page edit cannot forge.
//
// The workflow identity says who built the bytes, not which release they are:
// every release's genuine build verifies as signed by release.yml, so whoever
// can replace a release's assets could upload an older release's binary and
// checksums under a newer tag and every check above would pass. So the
// provenance is also bound to the release being installed: the tag is
// resolved to its commit through GitHub's git API, over the same pinned
// client, and both the certificate's source-repository digest (which Fulcio
// takes from the run's OIDC token, so no workflow step can write it) and the
// SLSA statement's resolved source dependency must name that commit. A
// certificate whose ref is exactly refs/tags/<tag> is not accepted as a
// substitute: release.yml also runs from main (auto-release), whose builds
// carry refs/heads/main, and a tag can be moved after a tag-push build, so
// the commit is the one proof both release paths share.

// provenancePolicy is who must have signed the provenance: the repository and
// the workflow file inside it.
type provenancePolicy struct {
	repo     string // owner/name
	workflow string // path of the signing workflow inside the repository
}

// releasePolicy pins the provenance to this repository's release workflow.
var releasePolicy = provenancePolicy{repo: "intentdriven/abcd", workflow: ".github/workflows/release.yml"}

const (
	// githubActionsIssuer is the OIDC issuer Fulcio records for a GitHub
	// Actions run.
	githubActionsIssuer = "https://token.actions.githubusercontent.com"
	// slsaProvenanceV1 is the predicate actions/attest writes with no predicate
	// input. The receipts attestation release.yml also makes has its own type
	// and never names a binary, but the check holds to this one regardless.
	slsaProvenanceV1 = "https://slsa.dev/provenance/v1"
	// attestationEndpoint is GitHub's attestations API for this repository;
	// "sha256:<hex>" is appended. It answers a public repository's attestations
	// without a token, which abcd does not hold.
	attestationEndpoint = "https://api.github.com/repos/intentdriven/abcd/attestations/"
	gitEndpoint         = "https://api.github.com/repos/intentdriven/abcd/git/"
	maxAttestationBytes = 4 << 20 // 4 MiB: one bundle is about 12 KiB
	maxTUFBytes         = 4 << 20 // a TUF role or trusted_root.json is a few KiB
	maxGitObjectBytes   = 1 << 20 // a ref or tag object is well under 1 KiB
)

// gitCommitShape is a full SHA-1 object name, the only shape a resolved
// release commit may take.
var gitCommitShape = regexp.MustCompile(`^[0-9a-f]{40}$`)

// maxTagDepth bounds how many annotated tag objects a ref may chain through
// before it reaches its commit. A release tag is one hop; a chain longer than
// this is refused rather than followed.
const maxTagDepth = 4

// signerSAN is the certificate identity the policy requires: the workflow file
// at the repository's main branch (auto-release's call) or at a vX tag (a tag
// push), the two ways release.yml runs to publish.
func (p provenancePolicy) signerSAN() string {
	return "^" + regexp.QuoteMeta("https://github.com/"+p.repo+"/"+p.workflow) + `@refs/(heads/main|tags/v[0-9][^@]*)$`
}

// provenanceVerifier checks a release asset's build provenance by its sha256,
// returning the verified signer identity.
type provenanceVerifier interface {
	verify(tag string, digest []byte) (signer string, err error)
}

// githubAttestations fetches the asset's attestations from GitHub's API under
// the updater's pinned transport and verifies each until one holds.
type githubAttestations struct {
	u           *Updater
	endpoint    string
	gitEndpoint string
	trusted     func() (root.TrustedMaterial, error)
	policy      provenancePolicy
}

func newGitHubAttestations(u *Updater) *githubAttestations {
	return &githubAttestations{u: u, endpoint: attestationEndpoint, gitEndpoint: gitEndpoint, trusted: u.sigstoreTrustedRoot, policy: releasePolicy}
}

// VerifyReleaseProvenance verifies one release asset's SLSA build provenance
// by its sha256 and returns the verified signer identity: the check Apply
// runs before a swap, for a verb that promotes a release asset it did not
// download itself. ahoy install runs it before promoting the plugin's cached
// release binary to the PATH copy (iss-2610120617515834), so both verbs hold
// a release binary to one primitive. The client is built per call, only when
// a promotion needs it, and like ReleaseAssets' it ignores the proxy and CA
// overrides without unsetting them: it runs inside another verb's process,
// which the updater's process-wide scrub would change under it. The network
// it touches is GitHub's attestations API and the Sigstore TUF repository,
// keyed by the digest of a release already chosen: it discovers nothing
// (adr-38 tier 3).
func VerifyReleaseProvenance(tag string, digest []byte) (string, error) {
	return newReleaseProvenance(releaseOrigin, urlguard.BlockedIP, false).verify(tag, digest)
}

// newReleaseProvenance is the test-reachable constructor behind
// VerifyReleaseProvenance: the updater's attestation check over a client that
// leaves the process environment alone.
func newReleaseProvenance(origin string, blocked func(net.IP) bool, allowHTTP bool) *githubAttestations {
	return newGitHubAttestations(newReleaseAssets(origin, blocked, allowHTTP).u)
}

// ReleaseProvenanceForTest builds the same check against an attestations
// endpoint a test serves and a fixed Sigstore trusted root, so a caller's
// tests can hold their wiring to recorded real bundles without the network.
// Never used by the shipping binary.
func ReleaseProvenanceForTest(apiBase string, trustedRootJSON []byte) (func(tag string, digest []byte) (string, error), error) {
	tr, err := root.NewTrustedRootFromJSON(trustedRootJSON)
	if err != nil {
		return nil, err
	}
	g := newReleaseProvenance(releaseOrigin, nil, true)
	g.endpoint = apiBase + "/attestations/"
	g.gitEndpoint = apiBase + "/git/"
	g.trusted = func() (root.TrustedMaterial, error) { return tr, nil }
	return g.verify, nil
}

func (g *githubAttestations) verify(tag string, digest []byte) (string, error) {
	commit, err := g.resolveTagCommit(tag)
	if err != nil {
		return "", err
	}
	hexDigest := hex.EncodeToString(digest)
	bundles, err := g.fetch(hexDigest)
	if err != nil {
		return "", err
	}
	tm, err := g.trusted()
	if err != nil {
		return "", fmt.Errorf("could not load the Sigstore trusted root to verify against: %w", err)
	}
	var failures []string
	for _, b := range bundles {
		signer, err := verifyBundle(b, digest, commit, tm, g.policy)
		if err == nil {
			return signer, nil
		}
		failures = append(failures, err.Error())
	}
	return "", fmt.Errorf("no attestation for sha256 %s verifies as signed by %s/%s at %s's commit %s: %s",
		hexDigest, g.policy.repo, g.policy.workflow, tag, commit, strings.Join(failures, "; "))
}

// gitObject is the part of a git ref or tag object the resolution reads.
type gitObject struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	} `json:"object"`
}

// resolveTagCommit resolves a release tag to the commit it names, through
// GitHub's git API under the updater's pinned client: the ref, then each
// annotated tag object it points through. Any failure refuses — a tag that is
// not a release tag, a missing ref, an answer for another ref, an object that
// is not a commit, a malformed name, a chain past maxTagDepth — because the
// provenance can only be bound to a release whose commit is known.
func (g *githubAttestations) resolveTagCommit(tag string) (string, error) {
	if !tagShape.MatchString(tag) {
		return "", fmt.Errorf("%q is not a release tag, so its commit cannot be resolved to bind the build provenance to", tag)
	}
	var obj gitObject
	if err := g.getGitObject(g.gitEndpoint+"ref/tags/"+tag, &obj); err != nil {
		return "", fmt.Errorf("could not resolve release %s to its commit: %w", tag, err)
	}
	if obj.Ref != "refs/tags/"+tag {
		return "", fmt.Errorf("could not resolve release %s to its commit: the git API answered for %q", tag, obj.Ref)
	}
	for depth := 0; ; depth++ {
		sha, typ := obj.Object.SHA, obj.Object.Type
		if !gitCommitShape.MatchString(sha) {
			return "", fmt.Errorf("could not resolve release %s to its commit: the git API named the object %q", tag, sha)
		}
		switch typ {
		case "commit":
			return sha, nil
		case "tag":
			if depth >= maxTagDepth {
				return "", fmt.Errorf("could not resolve release %s to its commit: more than %d tag objects deep", tag, maxTagDepth)
			}
			obj = gitObject{}
			if err := g.getGitObject(g.gitEndpoint+"tags/"+sha, &obj); err != nil {
				return "", fmt.Errorf("could not resolve release %s to its commit: tag object %s: %w", tag, sha, err)
			}
		default:
			return "", fmt.Errorf("could not resolve release %s to its commit: it names a %q, not a commit", tag, typ)
		}
	}
}

// getGitObject reads one git API answer into obj.
func (g *githubAttestations) getGitObject(rawURL string, obj *gitObject) error {
	body, _, err := g.u.getWith(rawURL, maxGitObjectBytes, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return errors.New("no such object is published")
		}
		return err
	}
	defer body.Close()
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, obj); err != nil {
		return fmt.Errorf("the answer is not a git object: %w", err)
	}
	return nil
}

// fetch reads the attestation list for one digest and returns the bundles it
// carries inline. A 404 or an empty list is "none"; an entry without an inline
// bundle is skipped, and a list of only those verifies nothing.
func (g *githubAttestations) fetch(hexDigest string) ([]json.RawMessage, error) {
	rawURL := g.endpoint + "sha256:" + hexDigest + "?per_page=100"
	body, _, err := g.u.getWith(rawURL, maxAttestationBytes, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return nil, fmt.Errorf("no build-provenance attestation is published for sha256 %s", hexDigest)
		}
		return nil, fmt.Errorf("could not fetch the attestations for sha256 %s: %w", hexDigest, err)
	}
	defer body.Close()
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("could not read the attestations for sha256 %s: %w", hexDigest, err)
	}
	var resp struct {
		Attestations []struct {
			Bundle json.RawMessage `json:"bundle"`
		} `json:"attestations"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("the attestations answer for sha256 %s is not JSON: %w", hexDigest, err)
	}
	var bundles []json.RawMessage
	for _, a := range resp.Attestations {
		if len(a.Bundle) == 0 || string(a.Bundle) == "null" {
			continue
		}
		bundles = append(bundles, a.Bundle)
	}
	if len(bundles) == 0 {
		if len(resp.Attestations) > 0 {
			return nil, fmt.Errorf("the attestations for sha256 %s carry no inline bundle to verify", hexDigest)
		}
		return nil, fmt.Errorf("no build-provenance attestation is published for sha256 %s", hexDigest)
	}
	return bundles, nil
}

// verifyBundle verifies one Sigstore bundle: the certificate chains to the
// trusted root's Fulcio CA with an embedded SCT, the entry is in the trusted
// Rekor log and was integrated while the certificate was valid, the signer is
// the policy's workflow in the policy's repository under GitHub Actions' OIDC
// issuer, the statement is SLSA provenance, one of its subjects is digest, and
// both the certificate's source-repository digest and the statement's resolved
// source dependency are commit. It returns the signer identity (the
// certificate's SAN).
func verifyBundle(raw []byte, digest []byte, commit string, tm root.TrustedMaterial, p provenancePolicy) (string, error) {
	var b bundle.Bundle
	if err := b.UnmarshalJSON(raw); err != nil {
		return "", fmt.Errorf("unreadable bundle: %w", err)
	}
	v, err := verify.NewVerifier(tm,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1))
	if err != nil {
		return "", err
	}
	san, err := verify.NewSANMatcher("", p.signerSAN())
	if err != nil {
		return "", err
	}
	issuer, err := verify.NewIssuerMatcher(githubActionsIssuer, "")
	if err != nil {
		return "", err
	}
	if !gitCommitShape.MatchString(commit) {
		return "", fmt.Errorf("no release commit %q to bind the provenance to", commit)
	}
	id, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		SourceRepositoryURI:    "https://github.com/" + p.repo,
		SourceRepositoryDigest: commit,
	})
	if err != nil {
		return "", err
	}
	res, err := v.Verify(&b, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(id)))
	if err != nil {
		return "", err
	}
	if res.Statement == nil || res.Statement.GetPredicateType() != slsaProvenanceV1 {
		got := ""
		if res.Statement != nil {
			got = res.Statement.GetPredicateType()
		}
		return "", fmt.Errorf("the attestation is %q, not SLSA build provenance", got)
	}
	predicate, err := json.Marshal(res.Statement.GetPredicate().AsMap())
	if err != nil {
		return "", fmt.Errorf("unreadable provenance predicate: %w", err)
	}
	if err := predicateNamesCommit(predicate, p.repo, commit); err != nil {
		return "", err
	}
	if res.Signature == nil || res.Signature.Certificate == nil {
		return "", errors.New("the verified attestation names no signing certificate")
	}
	return res.Signature.Certificate.SubjectAlternativeName, nil
}

// predicateNamesCommit checks the SLSA v1 predicate's resolved dependencies
// for repo's source at commit: an entry whose uri is this repository's git
// URL and whose gitCommit digest is commit. The certificate's source digest
// already binds the run to the commit; this holds the statement the run
// signed to the same commit, so neither can name another release.
func predicateNamesCommit(predicate []byte, repo, commit string) error {
	var pr struct {
		BuildDefinition struct {
			ResolvedDependencies []struct {
				URI    string            `json:"uri"`
				Digest map[string]string `json:"digest"`
			} `json:"resolvedDependencies"`
		} `json:"buildDefinition"`
	}
	if err := json.Unmarshal(predicate, &pr); err != nil {
		return fmt.Errorf("unreadable provenance predicate: %w", err)
	}
	source := "git+https://github.com/" + repo + "@"
	var named []string
	for _, d := range pr.BuildDefinition.ResolvedDependencies {
		if !strings.HasPrefix(d.URI, source) {
			continue
		}
		if d.Digest["gitCommit"] == commit {
			return nil
		}
		named = append(named, d.Digest["gitCommit"])
	}
	if len(named) == 0 {
		return fmt.Errorf("the provenance names no %s source commit, so it cannot be bound to commit %s", repo, commit)
	}
	return fmt.Errorf("the provenance was built from commit %s, not the release's commit %s", strings.Join(named, ", "), commit)
}

// sigstoreTrustedRoot fetches the Sigstore public-good trusted root through
// TUF, anchored on the root.json sigstore-go embeds, over the updater's pinned
// transport. Nothing is cached: the client starts from an empty temporary
// directory it never writes to, so a stale or planted cache can play no part
// and nothing is left behind (the default would write ~/.sigstore).
func (u *Updater) sigstoreTrustedRoot() (root.TrustedMaterial, error) {
	dir, err := os.MkdirTemp("", "abcd-tuf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	opts := &tuf.Options{
		Root:              tuf.DefaultRoot(),
		RepositoryBaseURL: tuf.DefaultMirror,
		CachePath:         dir,
		DisableLocalCache: true,
		Fetcher:           tufFetcher{u: u},
	}
	return root.FetchTrustedRootWithOptions(opts)
}

// tufFetcher is go-tuf's fetcher over the updater's client: no proxy, no CA
// override, the urlguard policy, redirects only within the pinned hosts.
type tufFetcher struct{ u *Updater }

func (f tufFetcher) DownloadFile(urlPath string, maxLength int64, _ time.Duration) ([]byte, error) {
	if maxLength <= 0 || maxLength > maxTUFBytes {
		maxLength = maxTUFBytes
	}
	body, _, err := f.u.get(urlPath, maxLength)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			// go-tuf walks root versions until this error with a 404.
			return nil, &metadata.ErrDownloadHTTP{StatusCode: http.StatusNotFound, URL: urlPath}
		}
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}
