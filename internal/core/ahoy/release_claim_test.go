package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A promotion holds the bytes
// to the release their own build metadata claims (CheckReleaseClaim). The
// package's fixtures carry no build metadata, so these tests stand in the
// claim through copyRelease (stubCopyRelease) and put the binding back.

// realReleaseClaim puts the binding back over the stood-in claim.
func realReleaseClaim(t *testing.T) {
	t.Helper()
	prev := checkReleaseClaim
	checkReleaseClaim = claimedRelease
	t.Cleanup(func() { checkReleaseClaim = prev })
}

// cacheAttestedAs lays a cache the home-scoped attestation binds to tag and
// returns the PATH target an install would write.
func cacheAttestedAs(t *testing.T, tag string) string {
	t.Helper()
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	writeUserCacheAttestation(t, attestationBodyAt(data, tag, cacheArtefact))
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	return filepath.Join(binDir, "abcd")
}

func TestInstallRefusesACacheThatClaimsAnotherRelease(t *testing.T) {
	realReleaseClaim(t)
	target := cacheAttestedAs(t, "v99.0.0")
	rec := &recordingProvenance{}
	opts := installOpts()
	opts.VerifyProvenance = rec.check

	res, err := Install(adoptableRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOwnedCopy(t, target, res)
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("a cache that claims another release must leave no PATH entry: %v", err)
	}
	joined := notesJoined(res.Notes)
	for _, want := range []string{"names release " + stubCopyRelease, "not v99.0.0", "nothing was installed"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal must say %q; notes = %v", want, res.Notes)
		}
	}
}

func TestInstallPromotesACacheThatClaimsItsRelease(t *testing.T) {
	realReleaseClaim(t)
	target := cacheAttestedAs(t, stubCopyRelease)
	opts := installOpts()
	opts.VerifyProvenance = (&recordingProvenance{}).check

	res, err := Install(adoptableRepo(t), opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != string(cacheArtefact) {
		t.Fatalf("a cache claiming the release its attestation names must be promoted; got %q (%v), notes %v", got, err, res.Notes)
	}
}

func TestOwnedCopyBoundToATagMustClaimIt(t *testing.T) {
	realReleaseClaim(t)
	target := filepath.Join(t.TempDir(), "abcd")
	if err := os.WriteFile(target, cacheArtefact, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := &recordingProvenance{}
	note, _, _ := checkOwnedCopy(target, "v99.0.0", rec.check)
	if !strings.Contains(note, "names release "+stubCopyRelease) {
		t.Errorf("a copy bound to a release it does not claim must not be adopted; note %q", note)
	}
	if strings.Contains(note, "Retry once") || !strings.Contains(note, "A retry will not change that") {
		t.Errorf("the refusal must say no retry changes which release a build is; note %q", note)
	}
	if note, _, _ := checkOwnedCopy(target, stubCopyRelease, rec.check); note != "" {
		t.Errorf("a copy bound to the release it claims must be adopted; note %q", note)
	}
}

func TestCheckReleaseClaimReadsTheBytes(t *testing.T) {
	if err := CheckReleaseClaim(cacheArtefact, stubCopyRelease); err == nil || !strings.Contains(err.Error(), "no Go build metadata") {
		t.Errorf("bytes with no build metadata claim no release; got %v", err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckReleaseClaim(bin, stubCopyRelease); err == nil || !strings.Contains(err.Error(), "not an abcd build") {
		t.Errorf("a test binary claims no abcd release; got %v", err)
	}
}

func TestClaimIsHoldsTheClaimToTheTag(t *testing.T) {
	if err := claimIs("v0.13.4", nil, "v0.13.4"); err != nil {
		t.Errorf("a build claiming the tag is that release's: %v", err)
	}
	err := claimIs("v0.13.4", nil, "v99.0.0")
	if err == nil || !strings.Contains(err.Error(), "names release v0.13.4, not v99.0.0") {
		t.Errorf("a build claiming another release must be refused, naming both; got %v", err)
	}
}
