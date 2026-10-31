package claim

import (
	"slices"
	"testing"
)

// TestADoneMarkKeptBeforeTheDeclarationChangedIsNotReadAfterIt: maya's own
// reservation recorded #14 under her id. Declared by octo with no login
// beside it, nothing of it is octo's to act on, so octo's sweep keeps the
// tip as done; once octo pairs the id with maya's login, #14 is listed due
// and octo's next act re-arms it.
func TestADoneMarkKeptBeforeTheDeclarationChangedIsNotReadAfterIt(t *testing.T) {
	w := newWorld(t)
	f := fourArmedForge(t)
	heldWithDeclaredEntry(t, w, f, mayaB, "maya")
	w.tagOrigin("v0.16.0")
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	declareHolderIDs(t, octo.home, mayaB+"\n")
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	if got := actedOn14(f); len(got) != 0 {
		t.Fatalf("acted on #14 while its id was unpaired: %v", got)
	}
	declareHolderIDs(t, octo.home, mayaB+" maya\n")
	if rw := listingOf(t, w, octo, "rel-v0.16.0"); !slices.Equal(numbers(rw.Due), []string{"#14"}) {
		t.Fatalf("octo's listing once paired = %+v; want #14 due", rw)
	}
	f.reset()
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.mutations(), "enablePullRequestAutoMerge #14") {
		t.Fatalf("mutations %v; want #14 re-armed once its id is paired", f.mutations())
	}
}
