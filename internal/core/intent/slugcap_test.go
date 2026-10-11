package intent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

const capWant = "a-record-s-file-name-has-no-cap-tied-to"

// TestCreateFromTextCutsTheSlugAtTheRecordCap is iss-2610100626320367 at the
// intent verb: the slug derived from a long text is cut at recordid.MaxSlugLen,
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
	if it.Slug != capWant || len(it.Slug) > recordid.MaxSlugLen {
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
	claimers := []spec.Spec{{ID: "spc-2", Status: spec.StatusOpen, Slug: recordid.CapSlug(long, recordid.MaxSlugLen)}}
	if _, ok := openRemainderWithSlug(claimers, long, "spc-1"); !ok {
		t.Fatalf("a retry asking for %q did not recognise the remainder minted as %q", long, claimers[0].Slug)
	}
}
