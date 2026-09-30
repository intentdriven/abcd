package lifeboat

import (
	"testing"
)

// TestSynthesisDoesNotPerturbManifest packs a real lifeboat, runs the
// deterministic principles + press-release seams over it, and asserts the pinned
// manifest still verifies and _provenance.json's manifest_sha256 is unchanged —
// the synthesis layer is post-pack and excluded from the seal.
func TestSynthesisDoesNotPerturbManifest(t *testing.T) {
	src := embarkableSourceFixture(t)
	lb := packSource(t, src)

	if err := verifyManifest(lb); err != nil {
		t.Fatalf("freshly packed lifeboat must verify: %v", err)
	}
	before := readProvenanceFile(t, lb).ManifestSHA256

	if _, err := SynthesizePrinciples(lb, nil); err != nil {
		t.Fatalf("SynthesizePrinciples: %v", err)
	}
	if _, err := ComposePressRelease(lb, nil); err != nil {
		t.Fatalf("ComposePressRelease: %v", err)
	}

	if err := verifyManifest(lb); err != nil {
		t.Fatalf("manifest must still verify after synthesis writes: %v", err)
	}
	if after := readProvenanceFile(t, lb).ManifestSHA256; after != before {
		t.Errorf("manifest_sha256 changed: before=%s after=%s", before, after)
	}
}

// TestEmbarkToleratesSynthesis asserts that after synthesis the embark probe
// reports the synthesis files as report-only (not unknown), so embark neither
// mis-flags nor writes them.
func TestEmbarkToleratesSynthesis(t *testing.T) {
	src := embarkableSourceFixture(t)
	lb := packSource(t, src)
	if _, err := SynthesizePrinciples(lb, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ComposePressRelease(lb, nil); err != nil {
		t.Fatal(err)
	}

	plan, err := EmbarkProbe(lb, t.TempDir())
	if err != nil {
		t.Fatalf("EmbarkProbe after synthesis: %v", err)
	}
	reportOnly := map[string]bool{}
	for _, ig := range plan.Ignored {
		if ig.Reason == IgnoredReportOnly {
			reportOnly[ig.LifeboatPath] = true
		}
		if ig.Reason == IgnoredUnknown &&
			(ig.LifeboatPath == "principles.json" || ig.LifeboatPath == "principles.md" ||
				ig.LifeboatPath == "press-release.json" || ig.LifeboatPath == "press-release.md") {
			t.Errorf("%s reported unknown, want report-only", ig.LifeboatPath)
		}
	}
	for _, want := range []string{"principles.json", "principles.md", "press-release.json", "press-release.md"} {
		if !reportOnly[want] {
			t.Errorf("%s not reported report-only", want)
		}
	}
}
