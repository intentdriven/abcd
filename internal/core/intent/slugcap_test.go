package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

const capWant = "a-record-s-file-name-has-no-cap-tied-to"

// TestCreateFromTextCutsTheSlugAtTheRecordCap is iss-2610100626320367 at the
// intent verb: the slug derived from a long text is cut at recordid.DefaultMaxSlugLen,
// while the derived title keeps its own, longer budget.
func TestCreateFromTextCutsTheSlugAtTheRecordCap(t *testing.T) {
	root := t.TempDir()
	it, err := CreateFromText(root, "A record's file name has no cap tied to the length of its full path on Windows", TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	if it.Slug != capWant {
		t.Fatalf("slug = %q, want %q", it.Slug, capWant)
	}
	if !strings.HasSuffix(filepath.Base(it.Path), "-"+capWant+".md") {
		t.Fatalf("filename %q does not carry the capped slug", it.Path)
	}
}

// TestCreateDraftCutsAnExplicitSlugAtTheRecordCap: a slug handed in whole (the
// promote routes carry one from the record they graduate) is cut at the same
// cap before it becomes a filename.
func TestCreateDraftCutsAnExplicitSlugAtTheRecordCap(t *testing.T) {
	root := t.TempDir()
	it, err := CreateDraft(root, DraftOptions{
		Slug:  "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its",
		Title: "A record's file name has no cap", SeedBody: "seed",
	})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if it.Slug != capWant || len(it.Slug) > recordid.DefaultMaxSlugLen {
		t.Fatalf("slug = %q, want %q", it.Slug, capWant)
	}
	if !strings.HasSuffix(filepath.Base(it.Path), "-"+capWant+".md") {
		t.Fatalf("filename %q does not carry the capped slug", it.Path)
	}
}

// TestOpenRemainderMatchesAfterTheCap: the remainder mint is idempotent by
// slug, and a long requested slug is minted cut, so a retry asking for the same
// long slug must still recognise the remainder the first attempt left behind.
func TestOpenRemainderMatchesAfterTheCap(t *testing.T) {
	long := "the-remaining-work-that-a-close-carries-forward-to-the-next-spec"
	claimers := []spec.Spec{{ID: "spc-2", Status: spec.StatusOpen, Slug: recordid.CapSlug(long, recordid.DefaultMaxSlugLen)}}
	if _, ok := openRemainderWithSlug(claimers, long, "spc-1", recordid.DefaultMaxSlugLen); !ok {
		t.Fatalf("a retry asking for %q did not recognise the remainder minted as %q", long, claimers[0].Slug)
	}
}

// TestIntentMintsCutAtTheRepositorysCap is iss-2611052241331003 at the intent
// verb: free text and a slug handed in whole both meet records.slug_cap, a retry
// of a remainder recognises the slug at that cap, and a nonsense value refuses.
func TestIntentMintsCutAtTheRepositorysCap(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, ".abcd", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": 20}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	const want = "a-record-s-file-name"
	it, err := CreateFromText(root, "A record's file name has no cap tied to the length of its full path", TextOptions{})
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	if it.Slug != want {
		t.Fatalf("slug from text = %q, want %q", it.Slug, want)
	}
	drafted, err := CreateDraft(root, DraftOptions{Slug: "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its",
		Title: "A record's file name has no cap", SeedBody: "seed"})
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if drafted.Slug != want {
		t.Fatalf("slug handed in whole = %q, want %q", drafted.Slug, want)
	}
	claimers := []spec.Spec{{ID: "spc-2", Status: spec.StatusOpen, Slug: want}}
	if _, ok := openRemainderWithSlug(claimers, "a-record-s-file-name-has-no-cap", "spc-1", 20); !ok {
		t.Fatal("a retry at the repository's cap did not recognise the remainder")
	}
	if err := os.WriteFile(cfg, []byte(`{"records": {"slug_cap": "long"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFromText(root, "Another text entirely", TextOptions{}); err == nil {
		t.Fatal("a nonsense cap minted an intent from text")
	}
	if _, err := CreateDraft(root, DraftOptions{Slug: "another", Title: "Another", SeedBody: "seed"}); err == nil {
		t.Fatal("a nonsense cap minted a draft")
	}
}
