package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
)

// armedCloseRepo is a checkout whose brief the doc-fidelity gate judges: the
// command-tree snapshot, a chapter naming every surface the live tree ships
// except those listed in omit, one planned intent and its open spec, committed.
func armedCloseRepo(t *testing.T, omit ...string) string {
	t.Helper()
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	snap, err := SurfaceSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, s := range docfidelity.ShippedForTest(snap.Commands, nil) {
		skip := false
		for _, o := range omit {
			skip = skip || s.Name == o
		}
		if !skip {
			b.WriteString("### `" + s.Name + "`\n")
		}
	}
	writeRepoFile(t, repo, docfidelity.SnapshotPath, "{}\n")
	writeRepoFile(t, repo, docfidelity.ChaptersDir+"/01-all.md", b.String())
	writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: additive\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	gitCommit(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "fixture")
	return repo
}

func assertUnmoved(t *testing.T, repo string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repo, cliSpecsOpen, "spc-1-alpha.md")); err != nil {
		t.Fatalf("the spec left open/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, cliPlanned, "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent left planned/: %v", err)
	}
}

func TestSpecCloseRefusesWithNoSavedDocsReview(t *testing.T) {
	repo := armedCloseRepo(t)
	out, err := runCLIErr(t, "spec", "close", "spc-1")
	if err == nil {
		t.Fatalf("spec close shipped with no docs review:\n%s", out)
	}
	if !strings.Contains(err.Error(), docfidelity.RunReviewFirst) || !strings.Contains(err.Error(), "itd-10") {
		t.Fatalf("refusal lacks the remedy or the intent: %v", err)
	}
	assertUnmoved(t, repo)
}

func TestSpecCloseRefusesOnAConfirmedFalseSentence(t *testing.T) {
	repo := armedCloseRepo(t)
	if _, _, err := docfidelity.Record(repo, []byte(`{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full",
		"failing": [{"doc": "brief", "chapter": "01-all.md", "sentence": "alpha prints YAML.", "evidence": "it prints JSON", "disposition": "confirmed"}]}`), time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err := runCLIErr(t, "spec", "close", "spc-1")
	if err == nil || !strings.Contains(err.Error(), `"alpha prints YAML."`) {
		t.Fatalf("the close did not refuse naming the sentence: %v", err)
	}
	assertUnmoved(t, repo)
}

func TestSpecCloseRefusesAnUndocumentedSurface(t *testing.T) {
	repo := armedCloseRepo(t, "abcd spec close")
	// Even a matching PROMOTE review does not excuse a missing chapter.
	if _, _, err := docfidelity.Record(repo, []byte(`{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`), time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err := runCLIErr(t, "spec", "close", "spc-1")
	if err == nil || !strings.Contains(err.Error(), "`abcd spec close`") {
		t.Fatalf("the close did not refuse the undocumented sub-verb: %v", err)
	}
	assertUnmoved(t, repo)
}

func TestSpecCloseProceedsOnAMatchingDocsReview(t *testing.T) {
	repo := armedCloseRepo(t)
	if _, _, err := docfidelity.Record(repo, []byte(`{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`), time.Now()); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "spec", "close", "spc-1")
	if _, err := os.Stat(filepath.Join(repo, ".abcd/development/intents/shipped", "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent did not ship: %v", err)
	}
}

// A --remainder close ships nothing, so it has no shipped move to refuse.
func TestSpecCloseWithARemainderIsNotGated(t *testing.T) {
	repo := armedCloseRepo(t)
	runCLI(t, "spec", "close", "spc-1", "--remainder", "the-rest", "--production-mode", "dictated-and-formatted")
	if _, err := os.Stat(filepath.Join(repo, cliPlanned, "itd-10-alpha.md")); err != nil {
		t.Fatalf("the intent left planned/ on a remainder close: %v", err)
	}
}
