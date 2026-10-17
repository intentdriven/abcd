package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// registerRepoAt writes the ahoy registry entry (~/.abcd.noindex/history/
// index.json) recording rootSHA's checkout at path, the one place the store
// keeps where a key's repository lives.
func registerRepoAt(t *testing.T, home, rootSHA, path string) {
	t.Helper()
	dir := abcdhome.Path(home, "history")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"schema": 1, "repos": []any{
		map[string]any{"root_commit": rootSHA, "name": "gone-repo", "path": path, "status": "active"},
	}}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestSurveyBacklogMarksAKeyWhoseCheckoutIsGoneAsOrphaned is
// iss-2609091717146700: a staged transcript whose repository was deleted can
// never be drained, because the drain needs that repository's own redaction
// configuration. The survey stats the key's recorded root, marks the pile
// orphaned, and a discard by root SHA alone, with no checkout to stand in,
// empties it.
func TestSurveyBacklogMarksAKeyWhoseCheckoutIsGoneAsOrphaned(t *testing.T) {
	_, home := setupStore(t)
	const goneSHA = "cccccccccccccccccccccccccccccccccccccccc"
	gone := filepath.Join(t.TempDir(), "deleted-checkout")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	registerRepoAt(t, home, goneSHA, gone)
	for _, s := range []string{"sess-a", "sess-b"} {
		if _, err := Stage(gone, goneSHA, mainStage(s), []byte("raw text of "+s+"\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	repos, err := SurveyBacklog()
	if err != nil {
		t.Fatalf("SurveyBacklog: %v", err)
	}
	if len(repos) != 1 || repos[0].RootSHA != goneSHA {
		t.Fatalf("want the gone repository's pile reported, got %+v", repos)
	}
	if !repos[0].Orphaned {
		t.Fatal("a key whose recorded checkout no longer exists was not marked orphaned")
	}
	if repos[0].Staged != 2 || repos[0].StagedBytes == 0 {
		t.Errorf("staged=%d bytes=%d, want the whole pile named", repos[0].Staged, repos[0].StagedBytes)
	}

	staged, err := ListStaged("", goneSHA)
	if err != nil {
		t.Fatalf("listing the orphaned pile by root SHA alone: %v", err)
	}
	if len(staged) != 2 {
		t.Fatalf("listed %d staged files by root SHA, want 2", len(staged))
	}
	for _, s := range staged {
		if _, err := Discard("", goneSHA, filepath.Base(s.Path)); err != nil {
			t.Fatalf("discard by root SHA with the checkout gone: %v", err)
		}
	}
	repos, err = SurveyBacklog()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Errorf("the discard left a pile behind: %+v", repos)
	}
}

// TestSurveyBacklogDoesNotOrphanALiveOrUnregisteredKey keeps the mark honest:
// a key whose recorded checkout exists is the ordinary backlog, and a key with
// no recorded root is unknown, not gone. Neither is called orphaned.
func TestSurveyBacklogDoesNotOrphanALiveOrUnregisteredKey(t *testing.T) {
	repoRoot, home := setupStore(t)
	registerRepoAt(t, home, testRootSHA, repoRoot)
	const unregistered = "dddddddddddddddddddddddddddddddddddddddd"
	for _, sha := range []string{testRootSHA, unregistered} {
		if _, err := Stage(repoRoot, sha, mainStage("sess-"+sha[:4]), []byte("raw\n")); err != nil {
			t.Fatal(err)
		}
	}
	repos, err := SurveyBacklog()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("want both piles reported, got %+v", repos)
	}
	for _, r := range repos {
		if r.Orphaned {
			t.Errorf("%s was marked orphaned", r.RootSHA)
		}
	}
}
