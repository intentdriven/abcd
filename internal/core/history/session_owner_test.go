package history

import (
	"path/filepath"
	"testing"
)

// TestSessionOwnershipHasOneDefinition pins iss-2609091911066372. The rule
// "the one store that already claims a session owns it; none or several is no
// answer" had two homes: an exported helper nothing called, and an inline copy
// in ingest's session placement that was the one that ran. It has one now,
// ownerIndex.owner, and the placement pass reaches it: a session two stores
// claim is refused by the rule and left unplaced by ingest alike.
func TestSessionOwnershipHasOneDefinition(t *testing.T) {
	const a, b = testRootSHA, "b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0"
	idx := ownerIndex{"one": {a}, "two": {a, b}}
	if sha, err := idx.owner("one"); err != nil || sha != a {
		t.Errorf("owner(one) = %q, %v; want %s", sha, err, a)
	}
	for _, id := range []string{"two", "none", "../x"} {
		if sha, err := idx.owner(id); err == nil {
			t.Errorf("owner(%s) = %q with no error; want a refusal", id, sha)
		}
	}

	// The placement pass reads the same rule off the on-disk index.
	repoRoot, _ := setupStore(t)
	other := t.TempDir()
	fakeRepos(t, map[string]string{repoRoot: a, other: b})
	for _, pair := range []struct{ root, sha, id string }{
		{repoRoot, a, "sess-one"}, {repoRoot, a, "sess-two"}, {other, b, "sess-two"},
	} {
		if err := NoteSessionRepo(pair.root, pair.sha, pair.id); err != nil {
			t.Fatal(err)
		}
	}
	got := placeSessions([]transcriptProbe{
		{path: filepath.Join(other, "x.jsonl"), sessionID: "sess-one"},
		{path: filepath.Join(other, "y.jsonl"), sessionID: "sess-two"},
	})
	if p := got["sess-one"]; p.rootSHA != a || p.via != "store" {
		t.Errorf("sess-one placed %+v, want the one claiming store", p)
	}
	if p := got["sess-two"]; p.rootSHA != "" {
		t.Errorf("sess-two, claimed by two stores, placed %+v; want no placement", p)
	}
}
