package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// The fixtures under testdata/provenance are recorded, not synthesised
// (2026-10-12):
//
//   - abcd-v0.13.4-linux-amd64.attestations.json: the GitHub attestations API's
//     answer for the sha256 of v0.13.4's abcd-linux-amd64, unauthenticated, with
//     each entry's short-lived signed bundle_url removed. Its one bundle is the
//     SLSA provenance release.yml signed over the six release assets.
//   - abcd-v0.13.4-site.attestations.json: the same API's answer for v0.13.4's
//     site.tar.gz, signed by this repository's site.yml: right repository,
//     wrong workflow.
//   - cli-v2.102.0.attestations.json: a real SLSA provenance bundle from another
//     repository (cli/cli's v2.102.0 release), wrapped in the API's response
//     shape: wrong repository.
//   - trusted_root.json: the Sigstore public-good trusted root, the one every
//     bundle above chains to (its Fulcio CA and its Rekor log), as `gh
//     attestation trusted-root` printed it.
const (
	linuxAmd64V0134 = "45aadc79c8db255805c81f2f31cdab7ecd473ade44c6e0197995721435524a54"
	siteTarV0134    = "e5d55c1d20daba8b3190916b89c40a93fdbe9fed09fbe1abeb7874a16d581e19"
	ghLinux386Deb   = "759b31942b78c05434a5d958fdacfc66aaf81446e283b4a8b1444104cf9de850"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "provenance", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fixtureTrusted(t *testing.T) func() (root.TrustedMaterial, error) {
	t.Helper()
	tr, err := root.NewTrustedRootFromJSON(fixture(t, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}
	return func() (root.TrustedMaterial, error) { return tr, nil }
}

// fixtureBundle returns the first bundle of a recorded API response.
func fixtureBundle(t *testing.T, name string) []byte {
	t.Helper()
	var resp struct {
		Attestations []struct {
			Bundle json.RawMessage `json:"bundle"`
		} `json:"attestations"`
	}
	if err := json.Unmarshal(fixture(t, name), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Attestations[0].Bundle
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func trustedMaterial(t *testing.T) root.TrustedMaterial {
	tm, err := fixtureTrusted(t)()
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

// --- the bundle check itself, against recorded real bundles ---

func TestVerifyBundleAcceptsTheReleaseProvenance(t *testing.T) {
	signer, err := verifyBundle(fixtureBundle(t, "abcd-v0.13.4-linux-amd64.attestations.json"),
		mustHex(t, linuxAmd64V0134), trustedMaterial(t), releasePolicy)
	if err != nil {
		t.Fatalf("the release's own provenance must verify: %v", err)
	}
	want := "https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main"
	if signer != want {
		t.Errorf("signer = %q, want %q", signer, want)
	}
}

func TestVerifyBundleRefusesAnotherWorkflow(t *testing.T) {
	_, err := verifyBundle(fixtureBundle(t, "abcd-v0.13.4-site.attestations.json"),
		mustHex(t, siteTarV0134), trustedMaterial(t), releasePolicy)
	if err == nil {
		t.Fatal("provenance signed by site.yml must not stand in for release.yml's")
	}
	if !strings.Contains(err.Error(), "site.yml") {
		t.Errorf("the refusal should name the signer it found: %v", err)
	}
}

func TestVerifyBundleRefusesAnotherRepository(t *testing.T) {
	_, err := verifyBundle(fixtureBundle(t, "cli-v2.102.0.attestations.json"),
		mustHex(t, ghLinux386Deb), trustedMaterial(t), releasePolicy)
	if err == nil {
		t.Fatal("another repository's provenance must not verify an abcd release")
	}
	if !strings.Contains(err.Error(), "cli/cli") {
		t.Errorf("the refusal should name the signer it found: %v", err)
	}
}

func TestVerifyBundleRefusesATamperedBinary(t *testing.T) {
	tampered := sha256.Sum256([]byte("a binary the release workflow never built"))
	_, err := verifyBundle(fixtureBundle(t, "abcd-v0.13.4-linux-amd64.attestations.json"),
		tampered[:], trustedMaterial(t), releasePolicy)
	if err == nil {
		t.Fatal("a real bundle must not vouch for bytes it does not name")
	}
}

// --- the fetch: the attestations API, keyed by the digest ---

// attestationAPI serves recorded responses at /attestations/sha256:<hex>; any
// digest not in byDigest is a 404, which is what GitHub answers.
func attestationAPI(t *testing.T, byDigest map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := strings.TrimPrefix(r.URL.Path, "/attestations/sha256:")
		b, ok := byDigest[d]
		if !ok {
			b, ok = byDigest["*"]
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testAttestations(t *testing.T, u *Updater, api *httptest.Server) *githubAttestations {
	t.Helper()
	return &githubAttestations{u: u, endpoint: api.URL + "/attestations/", trusted: fixtureTrusted(t), policy: releasePolicy}
}

func TestAttestationsVerifyThroughTheAPI(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	signer, err := testAttestations(t, u, api).verify(mustHex(t, linuxAmd64V0134))
	if err != nil {
		t.Fatalf("a recorded release attestation fetched by digest must verify: %v", err)
	}
	if !strings.Contains(signer, "release.yml") {
		t.Errorf("signer = %q", signer)
	}
}

func TestAttestationsRefuseAMissingBundle(t *testing.T) {
	api := attestationAPI(t, nil)
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	_, err := testAttestations(t, u, api).verify(mustHex(t, linuxAmd64V0134))
	if err == nil || !strings.Contains(err.Error(), "no build-provenance attestation") {
		t.Fatalf("a digest with no attestation must refuse, naming the absence: %v", err)
	}
}

func TestAttestationsRefuseAResponseWithNoInlineBundle(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{"*": []byte(`{"attestations":[{"repository_id":1,"bundle":null}]}`)})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	_, err := testAttestations(t, u, api).verify(mustHex(t, linuxAmd64V0134))
	if err == nil {
		t.Fatal("an attestation list with no bundle in it verifies nothing")
	}
}

func TestAttestationsRefuseWhenTheTrustedRootIsUnavailable(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	g := testAttestations(t, u, api)
	g.trusted = func() (root.TrustedMaterial, error) { return nil, errors.New("tuf unreachable") }
	if _, err := g.verify(mustHex(t, linuxAmd64V0134)); err == nil || !strings.Contains(err.Error(), "trusted root") {
		t.Fatalf("no trusted root means nothing can be verified: %v", err)
	}
}

// --- Apply: provenance is checked before the swap, and refuses it ---

// releaseWithProvenance builds an origin whose v0.6.2 asset is newBin, an
// attestation API answering as byDigest says, and an updater wired to both
// through the real attestation check.
func releaseWithProvenance(t *testing.T, newBin []byte, byDigest map[string][]byte) (*Updater, string, []byte) {
	t.Helper()
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, newBin)
	api := attestationAPI(t, byDigest)
	u := newUpdater(o.srv.URL, nil, testAssetName, true, nil)
	u.provenance = testAttestations(t, u, api)
	return u, writeTarget(t, oldBin), oldBin
}

func assertRefusedBeforeSwap(t *testing.T, rep Report, err error, target string, oldBin []byte) {
	t.Helper()
	if err != nil {
		t.Fatalf("a provenance failure is a named refusal, not an error: %v", err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil || rep.Refusal.Shape != "unverified-provenance" {
		t.Fatalf("want an unverified-provenance refusal, got %+v (refusal %+v)", rep, rep.Refusal)
	}
	if !strings.Contains(rep.Refusal.Remedy, "gh attestation verify") || !strings.Contains(rep.Refusal.Remedy, "releases/tag/v0.6.2") {
		t.Errorf("the remedy must name how to install by hand: %q", rep.Refusal.Remedy)
	}
	assertNoStagingFile(t, target, oldBin)
}

// TestApplyRefusesAReplacedBinaryAndManifest is the record's proof
// (iss-2609231050273096): the release page serves a replaced binary AND a
// checksums.txt rewritten to match it, so the same-page check passes; the
// binary has no provenance signed by release.yml, and the swap is refused.
func TestApplyRefusesAReplacedBinaryAndManifest(t *testing.T) {
	evil := []byte("a binary someone uploaded over the release")
	// The API answers with the real release attestation whatever is asked: the
	// strongest stand-in for a page that can show a real bundle beside a fake.
	u, target, oldBin := releaseWithProvenance(t, evil, map[string][]byte{"*": fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	rep, err := u.Apply(target, "v0.6.2", nil)
	assertRefusedBeforeSwap(t, rep, err, target, oldBin)
}

func TestApplyRefusesWhenNoAttestationExists(t *testing.T) {
	u, target, oldBin := releaseWithProvenance(t, []byte("new-binary-bytes"), nil)
	rep, err := u.Apply(target, "v0.6.2", nil)
	assertRefusedBeforeSwap(t, rep, err, target, oldBin)
	if !strings.Contains(rep.Refusal.Detail, "no build-provenance attestation") {
		t.Errorf("the refusal must say what failed: %q", rep.Refusal.Detail)
	}
}

func TestApplyRefusesAnotherWorkflowsProvenance(t *testing.T) {
	u, target, oldBin := releaseWithProvenance(t, []byte("new-binary-bytes"), map[string][]byte{"*": fixture(t, "abcd-v0.13.4-site.attestations.json")})
	rep, err := u.Apply(target, "v0.6.2", nil)
	assertRefusedBeforeSwap(t, rep, err, target, oldBin)
}

func TestApplyRefusesAnotherRepositorysProvenance(t *testing.T) {
	u, target, oldBin := releaseWithProvenance(t, []byte("new-binary-bytes"), map[string][]byte{"*": fixture(t, "cli-v2.102.0.attestations.json")})
	rep, err := u.Apply(target, "v0.6.2", nil)
	assertRefusedBeforeSwap(t, rep, err, target, oldBin)
}

func TestApplyRefusesWhenTheAttestationAPIIsUnreachable(t *testing.T) {
	u, target, oldBin := releaseWithProvenance(t, []byte("new-binary-bytes"), nil)
	g := u.provenance.(*githubAttestations)
	g.endpoint = "http://127.0.0.1:1/attestations/"
	rep, err := u.Apply(target, "v0.6.2", nil)
	assertRefusedBeforeSwap(t, rep, err, target, oldBin)
}

// recordingProvenance stands in for the attestation check where the bytes are
// synthetic: it records the digest it was asked about and answers as told.
type recordingProvenance struct {
	asked [][]byte
	err   error
}

func (r *recordingProvenance) verify(digest []byte) (string, error) {
	r.asked = append(r.asked, digest)
	if r.err != nil {
		return "", r.err
	}
	return "https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main", nil
}

func TestApplyChecksTheDownloadedBytesProvenanceAndRecordsTheSigner(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("old-binary-bytes")
	newBin := []byte("new-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, newBin)
	target := writeTarget(t, oldBin)
	u := newUpdater(o.srv.URL, nil, testAssetName, true, nil)
	rec := &recordingProvenance{}
	u.provenance = rec

	rep, err := u.Apply(target, "v0.6.2", nil)
	if err != nil || rep.Action != ActionSwapped {
		t.Fatalf("a verified release swaps: %+v %v", rep, err)
	}
	want := sha256.Sum256(newBin)
	if len(rec.asked) != 1 || hex.EncodeToString(rec.asked[0]) != hex.EncodeToString(want[:]) {
		t.Fatalf("provenance must be asked once, about the downloaded bytes: %x", rec.asked)
	}
	if !strings.Contains(rep.AttestedBy, "release.yml") {
		t.Errorf("the receipt must name the verified signer: %+v", rep)
	}
}

// --- the shipping wiring ---

func TestShippingUpdaterPinsTheAttestationCheck(t *testing.T) {
	u := NewGitHubUpdater()
	g, ok := u.provenance.(*githubAttestations)
	if !ok {
		t.Fatalf("the shipping updater must verify attestations, got %T", u.provenance)
	}
	if g.endpoint != "https://api.github.com/repos/intentdriven/abcd/attestations/" {
		t.Errorf("endpoint = %q", g.endpoint)
	}
	if g.policy.repo != "intentdriven/abcd" || g.policy.workflow != ".github/workflows/release.yml" {
		t.Errorf("policy = %+v", g.policy)
	}
	if g.trusted == nil {
		t.Error("the shipping check must fetch a trusted root")
	}
}

// TestTUFFetcherSignalsANotFound pins the contract go-tuf walks root versions
// by: a 404 must arrive as metadata.ErrDownloadHTTP, or the walk for the next
// root.json never terminates cleanly.
func TestTUFFetcherSignalsANotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	u := newUpdater(srv.URL, nil, testAssetName, true, nil)
	_, err := tufFetcher{u: u}.DownloadFile(srv.URL+"/2.root.json", 1<<20, 0)
	var httpErr *metadata.ErrDownloadHTTP
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		t.Fatalf("a 404 must be metadata.ErrDownloadHTTP{404}, got %#v", err)
	}
}
