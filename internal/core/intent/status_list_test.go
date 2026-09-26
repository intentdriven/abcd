package intent

import (
	"testing"
)

// status_list_test.go — the status view lists every intent (iss-242). A
// planning sweep over drafts/ needed, per intent, its id, title, bucket,
// whether its Acceptance Criteria are real or still the seeded placeholder, and
// when it was filed; the view reported bucket counts and spec links only, so the
// sweep grepped files and git history instead.

func TestStatusListsEveryIntentWithItsACState(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-2609211500001234-seeded-one.md",
		"---\nid: itd-2609211500001234\nslug: seeded-one\nspec_id: null\nkind: null\n---\n# A seeded draft\n\n"+
			"## Acceptance Criteria\n\n> _Required (the itd-1 discipline): add at least one Given-When-Then bullet._\n")
	writeFile(t, root, plannedDir+"/itd-7-real.md",
		"---\nid: itd-7\nslug: real\nspec_id: null\nkind: standalone\n---\n# A real intent\n\n"+
			"## Acceptance Criteria\n\n- Given a thing, when it runs, then it holds\n")

	v, err := Status(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Intents) != 2 {
		t.Fatalf("want 2 listed intents, got %+v", v.Intents)
	}
	byID := map[string]IntentListing{}
	for _, l := range v.Intents {
		byID[l.ID] = l
	}
	seeded := byID["itd-2609211500001234"]
	if seeded.Title != "A seeded draft" || seeded.Bucket != BucketDrafts || seeded.ACState != ACStateSeeded {
		t.Errorf("seeded draft listed as %+v", seeded)
	}
	if seeded.Filed == nil || *seeded.Filed != "2026-09-21" {
		t.Errorf("a timestamp id's filing date is its stamp's date, got %v", seeded.Filed)
	}
	real := byID["itd-7"]
	if real.Title != "A real intent" || real.Bucket != BucketPlanned || real.ACState != ACStateReal {
		t.Errorf("real intent listed as %+v", real)
	}
	if real.Filed != nil {
		t.Errorf("an ordinal id carries no date, so filed must be null, got %q", *real.Filed)
	}
}
