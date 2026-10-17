package ahoy

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// iss-2610120617515834: the cache `ahoy install` promotes was checked only
// against checksums.txt from the same release page as the binary, so whoever
// can change that page can change both. A promotion now also verifies the
// artefact's SLSA build provenance through the check `abcd update` runs
// (update.ReleaseProvenance, wired by the front door), and any failure of it
// refuses: no PATH entry is written and nothing vouches for the bytes.
//
// The check itself, against recorded real bundles, is proven where it lives
// (internal/core/update/provenance_test.go) and through the CLI's wiring
// (internal/surface/cli/ahoy_provenance_test.go); here it is a stand-in that
// records what it was asked.

// recordingProvenance answers as told and records each digest it was asked.
type recordingProvenance struct {
	asked [][]byte
	tags  []string
	err   error
}

func (r *recordingProvenance) check(tag string, digest []byte) (string, error) {
	r.asked = append(r.asked, append([]byte(nil), digest...))
	r.tags = append(r.tags, tag)
	if r.err != nil {
		return "", r.err
	}
	return "https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main", nil
}

// boundCacheInstall lays a cache the home-scoped attestation binds, as the
// bootstrap would, and returns the PATH target an install would write.
func boundCacheInstall(t *testing.T) (repo, target string) {
	t.Helper()
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	attestDataCache(t, data, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	return adoptableRepo(t), filepath.Join(binDir, "abcd")
}

func TestInstallChecksTheCachedArtefactsProvenanceBeforePromoting(t *testing.T) {
	repo, target := boundCacheInstall(t)
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != string(cacheArtefact) {
		t.Fatalf("a verified cache must be promoted; got %q (%v), notes %v", got, err, res.Notes)
	}
	want := sha256.Sum256(cacheArtefact)
	if len(rec.asked) != 1 || string(rec.asked[0]) != string(want[:]) {
		t.Fatalf("provenance must be asked once, about the bytes promoted: %x", rec.asked)
	}
}

func TestInstallRefusesACacheWhoseProvenanceFails(t *testing.T) {
	repo, target := boundCacheInstall(t)
	rec := &recordingProvenance{err: errors.New("no build-provenance attestation is published for sha256 abc")}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOwnedCopy(t, target, res)
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("a cache whose provenance fails must leave no PATH entry: %v", err)
	}
	joined := notesJoined(res.Notes)
	for _, want := range []string{"build provenance", "no build-provenance attestation is published", "gh attestation verify", installRemedyAnchor} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal must say %q; notes = %v", want, res.Notes)
		}
	}
}

func TestInstallRefusesWhenNoProvenanceCheckIsConfigured(t *testing.T) {
	repo, target := boundCacheInstall(t)
	opts := installOpts()
	opts.VerifyProvenance = nil

	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOwnedCopy(t, target, res)
	if !strings.Contains(notesJoined(res.Notes), "build provenance") {
		t.Errorf("a missing check is a refusal, never an unverified promotion; notes = %v", res.Notes)
	}
}

// TestInstallChecksAMatchingCopyAndLeavesIt: an entry that already holds the
// attested artefact is not written again, but its provenance is still
// checked, bound to the attested release. The bootstrap refreshes the PATH
// copy from the cache on a checksum alone where gh cannot verify, so equal
// bytes are no proof that a provenance check ever passed.
func TestInstallChecksAMatchingCopyAndLeavesIt(t *testing.T) {
	repo, target := boundCacheInstall(t)
	data := os.Getenv("CLAUDE_PLUGIN_DATA")
	writeUserCacheAttestation(t, attestationBodyAt(data, "v0.13.4", cacheArtefact))
	first := installOpts()
	if _, err := Install(repo, first, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(target)
	rec := &recordingProvenance{}
	again := installOpts()
	again.VerifyProvenance = rec.check
	res, err := Install(repo, again, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(cacheArtefact)
	if len(rec.asked) != 1 || string(rec.asked[0]) != string(want[:]) || rec.tags[0] != "v0.13.4" {
		t.Fatalf("a matching copy must be checked once, as the attested v0.13.4; asked %x for %q, notes %v", rec.asked, rec.tags, res.Notes)
	}
	if res.Status != "already_up_to_date" {
		t.Errorf("a genuine matching copy leaves the repository up to date: %q, notes %v", res.Status, res.Notes)
	}
	after, _ := os.Stat(target)
	if got, _ := os.ReadFile(target); string(got) != string(cacheArtefact) || !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("the verified copy must be left in place, unwritten")
	}
}

// TestInstallReportsAMatchingCopyWhoseCheckFails: the bootstrap's
// checksum-only refresh put the cache's bytes on PATH, and the check fails:
// the copy is left as it is and the run reports it unverified, never clean.
func TestInstallReportsAMatchingCopyWhoseCheckFails(t *testing.T) {
	repo, target := boundCacheInstall(t)
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{err: errors.New("no build-provenance attestation is published")}).check
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertUnverifiedCopyReported(t, res, target)
	if got, _ := os.ReadFile(target); string(got) != string(cacheArtefact) {
		t.Errorf("the check is read-only, yet the copy changed")
	}
}

// TestGapDrivenMatchingCopyWhoseCheckFailsIsUnverified: the same, with a
// stranded entry ahead driving the install step onto the matching copy.
func TestGapDrivenMatchingCopyWhoseCheckFailsIsUnverified(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	stale := filepath.Join(t.TempDir(), "usr-local-bin")
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	attestDataCache(t, data, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	repo := adoptableRepo(t)
	target := filepath.Join(binDir, "abcd")
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stale+string(os.PathListSeparator)+binDir)
	linkStranded(t, filepath.Join(stale, "abcd"), pluginRoot)
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{err: errors.New("no build-provenance attestation is published")}).check
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertUnverifiedCopyReported(t, res, target)
}

// TestInstallBindsTheProvenanceCheckToTheAttestedRelease: the check is asked
// about the release the bootstrap attested the cache at, so a cache holding
// an older release's genuine build under a newer tag fails it: the
// provenance must name the commit that tag resolves to.
func TestInstallBindsTheProvenanceCheckToTheAttestedRelease(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	writeUserCacheAttestation(t, attestationBodyAt(data, "v0.13.4", cacheArtefact))
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	if _, err := Install(adoptableRepo(t), opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if len(rec.tags) != 1 || rec.tags[0] != "v0.13.4" {
		t.Fatalf("the provenance must be checked for the attested release v0.13.4, asked for %q", rec.tags)
	}
}
