package release

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/gittest"
)

// armedRelease is a released repository whose brief is judged (it carries the
// surfaces chapters), with one intent shipped since the tag.
func armedRelease(t *testing.T) *gittest.Repo {
	t.Helper()
	r := releasedRepo(t)
	r.Write(docfidelity.ChaptersDir+"/04-launch.md", "### `abcd launch`\n")
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived from what shipped.\n")
	r.Commit("ship an intent")
	return r
}

func docFidelityRefusal(cut Cut) (Refusal, bool) {
	for _, ref := range cut.Refusals {
		if ref.Kind == RefusalDocFidelity {
			return ref, true
		}
	}
	return Refusal{}, false
}

func TestEmitRefusesACutWithNoSavedDocsReview(t *testing.T) {
	cut := emit(t, armedRelease(t))
	ref, ok := docFidelityRefusal(cut)
	if !ok || cut.Ready {
		t.Fatalf("the cut did not refuse on a missing docs review: %+v", cut.Refusals)
	}
	if !slices.Contains(ref.Records, "itd-73") || !strings.Contains(ref.Reason, "itd-73") ||
		!strings.Contains(ref.Reason, docfidelity.RunReviewFirst) {
		t.Fatalf("the refusal does not name the intent and the remedy: %+v", ref)
	}
}

func TestEmitRefusesACutWhileAShippedIntentsChapterLags(t *testing.T) {
	r := armedRelease(t)
	if _, _, err := docfidelity.Record(r.Root(), []byte(`{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full",
		"failing": [{"doc": "brief", "chapter": "04-launch.md", "sentence": "launch prints the version twice.", "evidence": "ship.go prints it once", "disposition": "confirmed"}]}`),
		time.Now()); err != nil {
		t.Fatal(err)
	}
	ref, ok := docFidelityRefusal(emit(t, r))
	if !ok || !strings.Contains(ref.Reason, "itd-73") || !strings.Contains(ref.Reason, `"launch prints the version twice."`) {
		t.Fatalf("the refusal does not name the intent and the sentence: %+v", ref)
	}
}

func TestEmitProceedsOnAMatchingDocsReview(t *testing.T) {
	r := armedRelease(t)
	if _, _, err := docfidelity.Record(r.Root(), []byte(`{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`),
		time.Now()); err != nil {
		t.Fatal(err)
	}
	cut := emit(t, r)
	if _, ok := docFidelityRefusal(cut); ok || !cut.Ready {
		t.Fatalf("a matching review did not let the cut proceed: %+v", cut.Refusals)
	}
}

func TestEmitWithNoIntentShippedJudgesNoPopulation(t *testing.T) {
	r := releasedRepo(t)
	r.Write(docfidelity.ChaptersDir+"/04-launch.md", "### `abcd launch`\n")
	r.Write(resolvedDir+"iss-24-x.md", "---\nid: iss-24\nslug: x\nimpact: fix\n---\n\nfixed.\n")
	r.Commit("resolve an issue")
	if ref, ok := docFidelityRefusal(emit(t, r)); ok {
		t.Fatalf("an empty population refused: %+v", ref)
	}
}

// A patch cut ships no intent, yet a surface an issue fix added still meets
// layer 1: the coverage floor runs at every enforcement point whatever the
// population, and only layer 2's saved review waits on a shipped intent
// (iss-2610020728118137).
func TestEmitWithNoIntentShippedStillRefusesAnUncoveredSurface(t *testing.T) {
	r := releasedRepo(t)
	r.Write(docfidelity.ChaptersDir+"/04-launch.md", "### `abcd launch`\n")
	r.Write(docfidelity.AgentsDir+"/scribe.md", "# scribe\n")
	r.Write(resolvedDir+"iss-24-x.md", "---\nid: iss-24\nslug: x\nimpact: fix\n---\n\nfixed.\n")
	r.Commit("resolve an issue that adds an agent no chapter names")
	cut := emit(t, r)
	ref, ok := docFidelityRefusal(cut)
	if !ok || cut.Ready {
		t.Fatalf("a patch cut with an uncovered surface did not refuse: %+v", cut.Refusals)
	}
	if !strings.Contains(ref.Reason, "`scribe`") || strings.Contains(ref.Reason, docfidelity.RunReviewFirst) {
		t.Fatalf("the refusal does not name the uncovered surface alone: %+v", ref)
	}
}
