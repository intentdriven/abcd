package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// The provenance check is the one internal/core/update runs over abcd's own
// binary, generalised to a Signer (release.go): a Sigstore bundle whose
// certificate Fulcio issued to the signer's workflow, logged in Rekor, over
// SLSA build provenance naming the asset's sha256, verified in process with
// sigstore-go against the Sigstore trust root fetched through TUF. A
// checksums file beside the asset proves nothing alone, since whoever can
// replace one can replace both; the pinned sha256 already closes that here,
// and the provenance adds that the pinned bytes were built by the makers'
// own workflow on a GitHub-hosted runner.
//
// Unlike abcd's own check it does not bind the provenance to the release
// tag's commit: that binding exists there because the digest is learnt from
// the release page, and an older genuine build could be replayed under a newer
// tag. Here the digest is fixed in abcd's source, so no replay can change the
// bytes that are installed.

const (
	githubActionsIssuer = "https://token.actions.githubusercontent.com"
	slsaProvenanceV1    = "https://slsa.dev/provenance/v1"
	githubHostedRunner  = "github-hosted"
	attestationsAPI     = "https://api.github.com/repos/"
	// bundleStoreHost is the one host a bundle_url may name.
	bundleStoreHost = "tmaproduction.blob.core.windows.net"
)

// Attester checks an asset's build provenance against a Signer, returning the
// verified signer identity.
type Attester func(ctx context.Context, s Signer, digest []byte) (string, error)

// githubAttester is the production Attester's state: where it asks, and the
// trust root it verifies against.
type githubAttester struct {
	f        *fetcher
	apiBase  string // "https://api.github.com/repos/"
	bundleOK func(*url.URL) bool
	trusted  func() (root.TrustedMaterial, error)
}

// defaultAttester is the shipping provenance check: GitHub's attestations API
// and the Sigstore public-good trust root, over the pinned transport.
func defaultAttester() Attester {
	g := &githubAttester{f: productionFetcher(apiHosts), apiBase: attestationsAPI,
		bundleOK: func(u *url.URL) bool { return u.Scheme == "https" && u.Host == bundleStoreHost }}
	g.trusted = g.sigstoreTrustedRoot
	return g.verify
}

func (g *githubAttester) verify(ctx context.Context, s Signer, digest []byte) (string, error) {
	hexDigest := hex.EncodeToString(digest)
	bundles, err := g.fetch(ctx, s.Repo, hexDigest)
	if err != nil {
		return "", err
	}
	tm, err := g.trusted()
	if err != nil {
		return "", fmt.Errorf("could not load the Sigstore trust root to check the signed build record against: %w", err)
	}
	var failures []string
	for _, b := range bundles {
		signer, err := verifyBundle(b, digest, tm, s)
		if err == nil {
			return signer, nil
		}
		failures = append(failures, err.Error())
	}
	return "", fmt.Errorf("no signed build record for sha256 %s verifies as signed by %s: %s",
		hexDigest, s.SAN(), strings.Join(failures, "; "))
}

// fetch reads the attestations GitHub holds for one digest in repo: each
// entry's inline bundle, or else the bundle its bundle_url names, which the
// store serves snappy-compressed. An entry that yields no bundle is skipped.
func (g *githubAttester) fetch(ctx context.Context, repo, hexDigest string) ([][]byte, error) {
	raw, err := g.f.get(ctx, g.apiBase+repo+"/attestations/sha256:"+hexDigest+"?per_page=100", maxBundleBytes, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		var nf *errNotFound404
		if errors.As(err, &nf) {
			return nil, fmt.Errorf("%s publishes no signed build record for sha256 %s", repo, hexDigest)
		}
		return nil, fmt.Errorf("could not read the signed build records for sha256 %s: %w", hexDigest, err)
	}
	var resp struct {
		Attestations []struct {
			Bundle    json.RawMessage `json:"bundle"`
			BundleURL string          `json:"bundle_url"`
		} `json:"attestations"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("the signed build records answer for sha256 %s is not JSON: %w", hexDigest, err)
	}
	var out [][]byte
	var skipped []string
	for _, a := range resp.Attestations {
		if len(a.Bundle) > 0 && string(a.Bundle) != "null" {
			out = append(out, a.Bundle)
			continue
		}
		if a.BundleURL == "" {
			continue
		}
		u, err := url.Parse(a.BundleURL)
		if err != nil || !g.bundleOK(u) {
			skipped = append(skipped, "a bundle_url outside GitHub's attestation store")
			continue
		}
		sn, err := g.f.get(ctx, a.BundleURL, maxBundleBytes, nil)
		if err != nil {
			skipped = append(skipped, err.Error())
			continue
		}
		b, err := decodeSnappy(sn, maxBundleBytes)
		if err != nil {
			skipped = append(skipped, err.Error())
			continue
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		why := ""
		if len(skipped) > 0 {
			why = " (" + strings.Join(skipped, "; ") + ")"
		}
		return nil, fmt.Errorf("%s publishes no readable signed build record for sha256 %s%s", repo, hexDigest, why)
	}
	return out, nil
}

// verifyBundle verifies one Sigstore bundle: the certificate chains to the
// trust root's Fulcio CA with an embedded SCT, the entry is in the trusted
// Rekor log and was integrated while the certificate was valid, the
// certificate names the Signer's workflow at its ref exactly under GitHub
// Actions' issuer, built from the Signer's repository at that ref on a
// GitHub-hosted runner, and the statement is SLSA provenance with digest
// among its subjects.
func verifyBundle(raw []byte, digest []byte, tm root.TrustedMaterial, s Signer) (string, error) {
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
	san, err := verify.NewSANMatcher(s.SAN(), "")
	if err != nil {
		return "", err
	}
	issuer, err := verify.NewIssuerMatcher(githubActionsIssuer, "")
	if err != nil {
		return "", err
	}
	id, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		SourceRepositoryURI: "https://github.com/" + s.Repo,
		SourceRepositoryRef: s.Ref,
		RunnerEnvironment:   githubHostedRunner,
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
		return "", fmt.Errorf("the record is %q, not SLSA build provenance", got)
	}
	if res.Signature == nil || res.Signature.Certificate == nil {
		return "", errors.New("the verified record names no signing certificate")
	}
	return res.Signature.Certificate.SubjectAlternativeName, nil
}

// sigstoreTrustedRoot fetches the Sigstore public-good trust root through
// TUF, anchored on the root.json sigstore-go embeds, over the pinned
// transport, from an empty temporary cache it never keeps (the default would
// write ~/.sigstore).
func (g *githubAttester) sigstoreTrustedRoot() (root.TrustedMaterial, error) {
	dir, err := os.MkdirTemp("", "abcd-tuf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	return root.FetchTrustedRootWithOptions(&tuf.Options{
		Root:              tuf.DefaultRoot(),
		RepositoryBaseURL: tuf.DefaultMirror,
		CachePath:         dir,
		DisableLocalCache: true,
		Fetcher:           tufFetcher{f: g.f},
	})
}

// tufFetcher is go-tuf's fetcher over the pinned transport.
type tufFetcher struct{ f *fetcher }

func (t tufFetcher) DownloadFile(urlPath string, maxLength int64, _ time.Duration) ([]byte, error) {
	if maxLength <= 0 || maxLength > maxBundleBytes {
		maxLength = maxBundleBytes
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	b, err := t.f.get(ctx, urlPath, maxLength, nil)
	if err != nil {
		var nf *errNotFound404
		if errors.As(err, &nf) {
			// go-tuf walks root versions until a 404.
			return nil, &metadata.ErrDownloadHTTP{StatusCode: 404, URL: urlPath}
		}
		return nil, err
	}
	return b, nil
}
