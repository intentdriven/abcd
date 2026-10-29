package update

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// Every path that takes a release binary holds the bytes to the release their own build metadata claims
// (verifyRelease). The fixtures' bytes claim nothing, so these tests stand in
// a claim: each binary claims the release it was published as first, and
// ahoy's tests hold the reader to real bytes.

// claimFirstPublished stands in a claim for the test origin's binaries: the
// bytes claim the tag they were first published under in o.
func claimFirstPublished(t *testing.T, o *testOrigin) {
	t.Helper()
	prev := checkReleaseClaim
	checkReleaseClaim = func(b []byte, tag string) error {
		for _, tg := range o.order {
			for name, body := range o.assets[tg] {
				if name != "checksums.txt" && string(body) == string(b) {
					if tg != tag {
						return errors.New("its build metadata names release " + tg + ", not " + tag + ", so it is not this release's build")
					}
					return nil
				}
			}
		}
		return errors.New("it carries no Go build metadata, so it names no release")
	}
	t.Cleanup(func() { checkReleaseClaim = prev })
}

// republished adds a release that carries another release's build.
func republished(o *testOrigin, from, tag string) {
	o.addRelease(tag, testAssetName, o.assets[from][testAssetName])
}

func TestApplyRefusesAnotherReleasesBuildUnderANewTag(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("v0.6.1-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, []byte("v0.6.2-binary-bytes"))
	republished(o, "v0.6.1", "v99.0.0")
	claimFirstPublished(t, o)
	target := writeTarget(t, []byte("v0.6.2-binary-bytes"))

	rep, err := testUpdater(t, o).Apply(target, "v99.0.0", nil)
	if err != nil {
		t.Fatalf("another release's build is a named refusal, not an error: %v", err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil || rep.Refusal.Shape != shapeNotThisRelease {
		t.Fatalf("want an %s refusal, got %+v (refusal %+v)", shapeNotThisRelease, rep, rep.Refusal)
	}
	if !strings.Contains(rep.Refusal.Detail, "names release v0.6.1, not v99.0.0") || rep.Refusal.Remedy != remedyNotThisRelease {
		t.Errorf("the refusal must name both releases and keep the binary in place: %+v", rep.Refusal)
	}
	assertNoStagingFile(t, target, []byte("v0.6.2-binary-bytes"))
}

func TestApplyRefusesToCallAnotherReleasesBuildCurrent(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("v0.6.1-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	republished(o, "v0.6.1", "v99.0.0")
	claimFirstPublished(t, o)
	target := writeTarget(t, oldBin)

	rep, err := testUpdater(t, o).Apply(target, "v99.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Action != ActionRefused || rep.Refusal == nil || rep.Refusal.Shape != shapeNotThisRelease {
		t.Fatalf("a matching file that claims another release is not current: %+v (refusal %+v)", rep, rep.Refusal)
	}
	if !strings.Contains(rep.Refusal.Detail, "not reported current at v99.0.0") {
		t.Errorf("the refusal must say the file is not current: %q", rep.Refusal.Detail)
	}

	rep, err = testUpdater(t, o).Apply(target, "v0.6.1", nil)
	if err != nil || rep.Action != ActionCurrent {
		t.Fatalf("the same file is current at the release it claims: %+v %v", rep, err)
	}
}

func TestApplySwapsABuildClaimingItsRelease(t *testing.T) {
	o := newTestOrigin(t)
	oldBin := []byte("v0.6.1-binary-bytes")
	newBin := []byte("v0.6.2-binary-bytes")
	o.addRelease("v0.6.1", testAssetName, oldBin)
	o.addRelease("v0.6.2", testAssetName, newBin)
	claimFirstPublished(t, o)
	target := writeTarget(t, oldBin)

	rep, err := testUpdater(t, o).Apply(target, "v0.6.2", nil)
	if err != nil || rep.Action != ActionSwapped {
		t.Fatalf("a build claiming the release it is published as swaps: %+v %v", rep, err)
	}
	if got, _ := os.ReadFile(target); string(got) != string(newBin) {
		t.Fatalf("target holds %q, want the new binary", got)
	}
}

func TestFetchVerifiedRefusesAnotherReleasesBuild(t *testing.T) {
	o := newTestOrigin(t)
	o.addRelease("v0.6.1", testAssetName, []byte("v0.6.1-binary-bytes"))
	republished(o, "v0.6.1", "v99.0.0")
	claimFirstPublished(t, o)
	u := testUpdater(t, o)

	if _, err := u.FetchVerified("v99.0.0", nil); err == nil || !strings.Contains(err.Error(), "names release v0.6.1, not v99.0.0") {
		t.Errorf("a fetch of another release's build must refuse, naming both; got %v", err)
	}
	if got, err := u.FetchVerified("v0.6.1", nil); err != nil || string(got.Bytes) != "v0.6.1-binary-bytes" {
		t.Errorf("a fetch of the release a build claims returns it: %q %v", got.Bytes, err)
	}
}
