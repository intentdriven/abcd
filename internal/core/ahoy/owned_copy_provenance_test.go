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
	err   error
}

func (r *recordingProvenance) check(digest []byte) (string, error) {
	r.asked = append(r.asked, append([]byte(nil), digest...))
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

// TestInstallLeavesAMatchingCopyWithoutAsking: an entry that already is the
// attested artefact is not written again, so nothing is promoted and the
// network is not asked.
func TestInstallLeavesAMatchingCopyWithoutAsking(t *testing.T) {
	repo, target := boundCacheInstall(t)
	first := installOpts()
	first.VerifyProvenance = (&recordingProvenance{}).check
	if _, err := Install(repo, first, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	rec := &recordingProvenance{err: errors.New("must not be asked")}
	again := installOpts()
	again.VerifyProvenance = rec.check
	res, err := Install(repo, again, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.asked) != 0 {
		t.Errorf("an entry already holding the attested bytes needs no promotion; asked %x, notes %v", rec.asked, res.Notes)
	}
	if got, _ := os.ReadFile(target); string(got) != string(cacheArtefact) {
		t.Errorf("the verified copy must stay in place")
	}
}
