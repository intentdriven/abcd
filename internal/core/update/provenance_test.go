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
		mustHex(t, linuxAmd64V0134), commitV0134, trustedMaterial(t), releasePolicy)
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
		mustHex(t, siteTarV0134), commitV0134, trustedMaterial(t), releasePolicy)
	if err == nil {
		t.Fatal("provenance signed by site.yml must not stand in for release.yml's")
	}
	if !strings.Contains(err.Error(), "site.yml") {
		t.Errorf("the refusal should name the signer it found: %v", err)
	}
}

func TestVerifyBundleRefusesAnotherRepository(t *testing.T) {
	_, err := verifyBundle(fixtureBundle(t, "cli-v2.102.0.attestations.json"),
		mustHex(t, ghLinux386Deb), commitV0134, trustedMaterial(t), releasePolicy)
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
		tampered[:], commitV0134, trustedMaterial(t), releasePolicy)
	if err == nil {
		t.Fatal("a real bundle must not vouch for bytes it does not name")
	}
}

// --- the fetch: the attestations API, keyed by the digest ---

// The release tags the recorded bundles are bound to, as GitHub's git API
// answered for them (2026-10-12): v0.13.4 is an annotated tag whose tag object
// points at the commit its provenance names; v0.13.3 is the release before it.
const (
	commitV0134 = "2de70c93b510b606f54ce02ee7fe2603a36d5ef0"
	tagObjV0134 = "dc7f461c5ee42e56f459de6484c5f9db9aca72f6"
	commitV0133 = "33681cf8905216c5ea60d180739a0ba3868eef23"
)

// recordedRefs is the git API as GitHub answers it for v0.13.4 (annotated:
// ref -> tag object -> commit), plus v0.6.2, the synthetic release the Apply
// tests serve, as a lightweight tag on the same commit, so those tests reach
// the attestation check.
func recordedRefs() map[string]string {
	return map[string]string{
		"/git/ref/tags/v0.13.4":    `{"ref":"refs/tags/v0.13.4","object":{"sha":"` + tagObjV0134 + `","type":"tag"}}`,
		"/git/tags/" + tagObjV0134: `{"sha":"` + tagObjV0134 + `","tag":"v0.13.4","object":{"sha":"` + commitV0134 + `","type":"commit"}}`,
		"/git/ref/tags/v0.6.2":     `{"ref":"refs/tags/v0.6.2","object":{"sha":"` + commitV0134 + `","type":"commit"}}`,
	}
}

// attestationAPI serves recorded responses at /attestations/sha256:<hex>; any
// digest not in byDigest is a 404, which is what GitHub answers. It serves the
// recorded git refs too, so a tag resolves as GitHub resolves it.
func attestationAPI(t *testing.T, byDigest map[string][]byte) *httptest.Server {
	t.Helper()
	return repoAPI(t, byDigest, recordedRefs())
}

// repoAPI is attestationAPI with the git API's answers given: refs maps a
// request path under the repository (/git/ref/tags/<tag>, /git/tags/<sha>) to
// its JSON body; any other path is a 404.
func repoAPI(t *testing.T, byDigest map[string][]byte, refs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/git/") {
			b, ok := refs[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(b))
			return
		}
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
	return &githubAttestations{u: u, endpoint: api.URL + "/attestations/", gitEndpoint: api.URL + "/git/", trusted: fixtureTrusted(t), policy: releasePolicy}
}

func TestAttestationsVerifyThroughTheAPI(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	signer, err := testAttestations(t, u, api).verify("v0.13.4", mustHex(t, linuxAmd64V0134))
	if err != nil {
		t.Fatalf("a recorded release attestation fetched by digest must verify for its own tag: %v", err)
	}
	if !strings.Contains(signer, "release.yml") {
		t.Errorf("signer = %q", signer)
	}
}

func TestAttestationsRefuseAMissingBundle(t *testing.T) {
	api := attestationAPI(t, nil)
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	_, err := testAttestations(t, u, api).verify("v0.13.4", mustHex(t, linuxAmd64V0134))
	if err == nil || !strings.Contains(err.Error(), "no build-provenance attestation") {
		t.Fatalf("a digest with no attestation must refuse, naming the absence: %v", err)
	}
}

func TestAttestationsRefuseAResponseWithNoInlineBundle(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{"*": []byte(`{"attestations":[{"repository_id":1,"bundle":null}]}`)})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	_, err := testAttestations(t, u, api).verify("v0.13.4", mustHex(t, linuxAmd64V0134))
	if err == nil {
		t.Fatal("an attestation list with no bundle in it verifies nothing")
	}
}

func TestAttestationsRefuseWhenTheTrustedRootIsUnavailable(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	g := testAttestations(t, u, api)
	g.trusted = func() (root.TrustedMaterial, error) { return nil, errors.New("tuf unreachable") }
	if _, err := g.verify("v0.13.4", mustHex(t, linuxAmd64V0134)); err == nil || !strings.Contains(err.Error(), "trusted root") {
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
	tags  []string
	err   error
}

func (r *recordingProvenance) verify(tag string, digest []byte) (string, error) {
	r.asked = append(r.asked, digest)
	r.tags = append(r.tags, tag)
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
	if len(rec.tags) != 1 || rec.tags[0] != "v0.6.2" {
		t.Fatalf("provenance must be bound to the tag being installed: %q", rec.tags)
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
	if g.gitEndpoint != "https://api.github.com/repos/intentdriven/abcd/git/" {
		t.Errorf("gitEndpoint = %q", g.gitEndpoint)
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

// --- the check other verbs run before promoting a release asset ---

// TestReleaseProvenanceIsTheUpdatersCheck: the check ahoy install runs
// (iss-2610120617515834) is the updater's own attestation check, pinned to
// the same API, policy and trusted-root source, one primitive for both verbs.
func TestReleaseProvenanceIsTheUpdatersCheck(t *testing.T) {
	g := newReleaseProvenance(releaseOrigin, nil, false)
	if g.endpoint != attestationEndpoint {
		t.Errorf("endpoint = %q", g.endpoint)
	}
	if g.gitEndpoint != gitEndpoint {
		t.Errorf("gitEndpoint = %q", g.gitEndpoint)
	}
	if g.policy != releasePolicy {
		t.Errorf("policy = %+v", g.policy)
	}
	if g.trusted == nil {
		t.Error("the check must fetch a trusted root")
	}
}

// TestReleaseProvenanceLeavesTheEnvironmentAlone: unlike the updater, which
// owns its process, the check runs inside another verb, so it ignores the
// proxy and CA overrides without unsetting them.
func TestReleaseProvenanceLeavesTheEnvironmentAlone(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://192.0.2.1:3128")
	t.Setenv("SSL_CERT_FILE", "/nonexistent/ca.pem")
	g := newReleaseProvenance(releaseOrigin, nil, false)
	if got := os.Getenv("HTTPS_PROXY"); got != "http://192.0.2.1:3128" {
		t.Errorf("HTTPS_PROXY was changed to %q", got)
	}
	if got := os.Getenv("SSL_CERT_FILE"); got != "/nonexistent/ca.pem" {
		t.Errorf("SSL_CERT_FILE was changed to %q", got)
	}
	tr, ok := g.u.client.Transport.(*http.Transport)
	if !ok || tr.Proxy != nil {
		t.Errorf("the check's client must honour no proxy: %#v", g.u.client.Transport)
	}
}

// TestReleaseProvenanceForTestVerifiesRecordedBundles: the test-built check
// is the same verifier against a served API and a fixed trusted root: the
// release's own bundle verifies its digest, and refuses other bytes.
func TestReleaseProvenanceForTestVerifiesRecordedBundles(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{"*": fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	check, err := ReleaseProvenanceForTest(api.URL, fixture(t, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}
	if signer, err := check("v0.13.4", mustHex(t, linuxAmd64V0134)); err != nil || !strings.Contains(signer, "release.yml") {
		t.Fatalf("the release's own provenance must verify: %q %v", signer, err)
	}
	other := sha256.Sum256([]byte("a binary the release workflow never built"))
	if _, err := check("v0.13.4", other[:]); err == nil {
		t.Fatal("a real bundle must not vouch for bytes it does not name")
	}
}

// --- the provenance is bound to the release tag, not only to the workflow ---

// A genuine release build verifies as signed by release.yml whichever release
// page it is uploaded to: the workflow identity says who built it, not which
// release. Whoever can replace a release's assets could upload an older
// release's genuine binary and checksums under a newer tag, so the check also
// requires the provenance to name the commit the tag resolves to.

// TestAttestationsRefuseAGenuineBundleForTheWrongTag: v0.13.4's real
// provenance, presented for a tag that resolves to another commit (v0.13.3's),
// is refused, naming both commits.
func TestAttestationsRefuseAGenuineBundleForTheWrongTag(t *testing.T) {
	refs := map[string]string{
		"/git/ref/tags/v0.13.5": `{"ref":"refs/tags/v0.13.5","object":{"sha":"` + commitV0133 + `","type":"commit"}}`,
	}
	api := repoAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")}, refs)
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	_, err := testAttestations(t, u, api).verify("v0.13.5", mustHex(t, linuxAmd64V0134))
	if err == nil {
		t.Fatal("a genuine build of another release must not verify for this tag")
	}
	for _, want := range []string{commitV0133, commitV0134} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %s: %v", want, err)
		}
	}
}

// TestAttestationsVerifyTheRightTagThroughAnAnnotatedTag: the same bundle for
// its own tag verifies, the annotated tag dereferenced to its commit.
func TestAttestationsVerifyTheRightTagThroughAnAnnotatedTag(t *testing.T) {
	api := attestationAPI(t, map[string][]byte{linuxAmd64V0134: fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")})
	u := newUpdater(api.URL, nil, testAssetName, true, nil)
	if _, err := testAttestations(t, u, api).verify("v0.13.4", mustHex(t, linuxAmd64V0134)); err != nil {
		t.Fatalf("the release's provenance must verify for its own annotated tag: %v", err)
	}
}

// TestAttestationsRefuseATagThatDoesNotResolve: any failure to resolve the
// tag's commit refuses — a missing tag, a malformed answer, an object that is
// not a commit, a tag chain that never ends — and nothing is verified.
func TestAttestationsRefuseATagThatDoesNotResolve(t *testing.T) {
	loop := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name string
		tag  string
		refs map[string]string
	}{
		{"no such tag", "v0.13.4", map[string]string{}},
		{"not JSON", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `<html>`}},
		{"a prefix match list", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `[{"ref":"refs/tags/v0.13.40","object":{"sha":"` + commitV0134 + `","type":"commit"}}]`}},
		{"another ref", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `{"ref":"refs/tags/v0.13.40","object":{"sha":"` + commitV0134 + `","type":"commit"}}`}},
		{"a malformed sha", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `{"ref":"refs/tags/v0.13.4","object":{"sha":"2de70c93","type":"commit"}}`}},
		{"a tree", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `{"ref":"refs/tags/v0.13.4","object":{"sha":"` + commitV0134 + `","type":"tree"}}`}},
		{"an unresolvable tag object", "v0.13.4", map[string]string{"/git/ref/tags/v0.13.4": `{"ref":"refs/tags/v0.13.4","object":{"sha":"` + tagObjV0134 + `","type":"tag"}}`}},
		{"a tag chain that never ends", "v0.13.4", map[string]string{
			"/git/ref/tags/v0.13.4": `{"ref":"refs/tags/v0.13.4","object":{"sha":"` + loop + `","type":"tag"}}`,
			"/git/tags/" + loop:     `{"sha":"` + loop + `","object":{"sha":"` + loop + `","type":"tag"}}`,
		}},
		{"a path-shaped tag", "../v0.13.4", recordedRefs()},
		{"no tag at all", "", recordedRefs()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := repoAPI(t, map[string][]byte{"*": fixture(t, "abcd-v0.13.4-linux-amd64.attestations.json")}, tc.refs)
			u := newUpdater(api.URL, nil, testAssetName, true, nil)
			_, err := testAttestations(t, u, api).verify(tc.tag, mustHex(t, linuxAmd64V0134))
			if err == nil {
				t.Fatal("a tag whose commit cannot be resolved must refuse")
			}
			if !strings.Contains(err.Error(), "commit") {
				t.Errorf("the refusal must say the release's commit could not be resolved: %v", err)
			}
		})
	}
}

// TestStatementNamesCommit holds the statement half of the binding: the SLSA
// provenance's resolved source dependency must be this repository at the
// tag's commit. The certificate half (the Fulcio-attested source digest) is
// what the wrong-tag test above trips first, so this is checked on its own.
func TestStatementNamesCommit(t *testing.T) {
	var b struct {
		DSSEEnvelope struct {
			Payload []byte `json:"payload"`
		} `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(fixtureBundle(t, "abcd-v0.13.4-linux-amd64.attestations.json"), &b); err != nil {
		t.Fatal(err)
	}
	var st struct {
		Predicate json.RawMessage `json:"predicate"`
	}
	if err := json.Unmarshal(b.DSSEEnvelope.Payload, &st); err != nil {
		t.Fatal(err)
	}
	if err := predicateNamesCommit(st.Predicate, releasePolicy.repo, commitV0134); err != nil {
		t.Fatalf("v0.13.4's provenance names its own commit: %v", err)
	}
	if err := predicateNamesCommit(st.Predicate, releasePolicy.repo, commitV0133); err == nil {
		t.Fatal("v0.13.4's provenance must not stand for v0.13.3's commit")
	}
	if err := predicateNamesCommit(st.Predicate, "cli/cli", commitV0134); err == nil {
		t.Fatal("the commit must be named as this repository's source")
	}
	if err := predicateNamesCommit(json.RawMessage(`{"buildDefinition":{}}`), releasePolicy.repo, commitV0134); err == nil {
		t.Fatal("a predicate naming no source commit binds nothing")
	}
}

// TestApplyChecksProvenanceBeforeCallingAMatchingFileCurrent: a target whose
// bytes already match the tag's checksums.txt is reported current only once
// those bytes verify for that tag. Otherwise a release page re-uploaded with
// an older genuine build would call the old build current at the new tag, and
// the CLI would record the new tag as the newest release installed.
func TestApplyChecksProvenanceBeforeCallingAMatchingFileCurrent(t *testing.T) {
	o := newTestOrigin(t)
	bin := []byte("same-binary-bytes")
	o.addRelease("v0.6.2", testAssetName, bin)
	target := writeTarget(t, bin)
	u := newUpdater(o.srv.URL, nil, testAssetName, true, nil)
	rec := &recordingProvenance{err: errors.New("the provenance names commit 33681cf, not v0.6.2's")}
	u.provenance = rec

	rep, err := u.Apply(target, "v0.6.2", nil)
	if err != nil {
		t.Fatalf("a provenance failure is a named refusal, not an error: %v", err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil || rep.Refusal.Shape != "unverified-provenance" {
		t.Fatalf("a matching file whose provenance fails is not current: %+v (refusal %+v)", rep, rep.Refusal)
	}
	if len(rec.tags) != 1 || rec.tags[0] != "v0.6.2" {
		t.Fatalf("the current file's provenance must be checked for the tag: %q", rec.tags)
	}

	rec.err = nil
	rep, err = u.Apply(target, "v0.6.2", nil)
	if err != nil || rep.Action != ActionCurrent {
		t.Fatalf("a matching file whose provenance verifies is current: %+v %v", rep, err)
	}
	if !strings.Contains(rep.AttestedBy, "release.yml") {
		t.Errorf("the receipt must name the verified signer: %+v", rep)
	}
}
