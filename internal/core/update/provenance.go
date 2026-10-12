package update

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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
	maxAttestationBytes = 4 << 20 // 4 MiB: one bundle is about 12 KiB
	maxTUFBytes         = 4 << 20 // a TUF role or trusted_root.json is a few KiB
)

// signerSAN is the certificate identity the policy requires: the workflow file
// at the repository's main branch (auto-release's call) or at a vX tag (a tag
// push), the two ways release.yml runs to publish.
func (p provenancePolicy) signerSAN() string {
	return "^" + regexp.QuoteMeta("https://github.com/"+p.repo+"/"+p.workflow) + `@refs/(heads/main|tags/v[0-9][^@]*)$`
}

// provenanceVerifier checks a release asset's build provenance by its sha256,
// returning the verified signer identity.
type provenanceVerifier interface {
	verify(digest []byte) (signer string, err error)
}

// githubAttestations fetches the asset's attestations from GitHub's API under
// the updater's pinned transport and verifies each until one holds.
type githubAttestations struct {
	u        *Updater
	endpoint string
	trusted  func() (root.TrustedMaterial, error)
	policy   provenancePolicy
}

func newGitHubAttestations(u *Updater) *githubAttestations {
	return &githubAttestations{u: u, endpoint: attestationEndpoint, trusted: u.sigstoreTrustedRoot, policy: releasePolicy}
}

func (g *githubAttestations) verify(digest []byte) (string, error) {
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
		signer, err := verifyBundle(b, digest, tm, g.policy)
		if err == nil {
			return signer, nil
		}
		failures = append(failures, err.Error())
	}
	return "", fmt.Errorf("no attestation for sha256 %s verifies as signed by %s/%s: %s",
		hexDigest, g.policy.repo, g.policy.workflow, strings.Join(failures, "; "))
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
// issuer, the statement is SLSA provenance, and one of its subjects is digest.
// It returns the signer identity (the certificate's SAN).
func verifyBundle(raw []byte, digest []byte, tm root.TrustedMaterial, p provenancePolicy) (string, error) {
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
	id, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		SourceRepositoryURI: "https://github.com/" + p.repo,
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
	if res.Signature == nil || res.Signature.Certificate == nil {
		return "", errors.New("the verified attestation names no signing certificate")
	}
	return res.Signature.Certificate.SubjectAlternativeName, nil
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
