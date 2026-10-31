package claim

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestAReadNeverMintsAHolderID (iss-2610290809526033): every path that only
// reads what is held — the fetch and view, the cap's ceiling, the status
// fetch and read, the prompt hook's block, the release holds, the listing of
// a release's withdrawals and the listing's fallback settle — runs in a home
// that keeps no holder id and leaves it keeping none: an id is minted only by
// an act that pushes under it.
func TestAReadNeverMintsAHolderID(t *testing.T) {
	w := newWorld(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := octo.announce("merging", "#1012"); err != nil {
		t.Fatal(err)
	}
	maya := w.clone("5678+maya@users.noreply.github.com", t.TempDir())
	kept := abcdhome.Path(maya.home, "holder-id")
	assertNone := func(after string) {
		t.Helper()
		if _, err := os.Lstat(kept); !os.IsNotExist(err) {
			t.Fatalf("after %s the reader's home keeps a holder id (%v); a read must mint none", after, err)
		}
	}

	if _, err := Fetch(maya.dir, maya.opts()); err != nil {
		t.Fatal(err)
	}
	assertNone("the fetch")
	snap, err := View(maya.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Reservations) != 1 {
		t.Fatalf("the view holds %d reservations; want octo's one", len(snap.Reservations))
	}
	if _, ok := CeilingFor(maya.dir, &snap, maya.opts()); !ok {
		t.Fatal("the reader's ceiling was not read")
	}
	assertNone("the ceiling")
	_ = snap.ReleaseWithdrawals(maya.dir, maya.self(), w.clock())
	assertNone("the release withdrawals")
	if _, err := SettleEndedReleases(maya.dir, maya.opts()); err != nil {
		t.Fatal(err)
	}
	assertNone("the listing's settle")
	if _, err := ReleaseHoldsForTest(maya.dir, w.clock(), maya.opts()); err != nil {
		t.Fatal(err)
	}
	assertNone("the release holds")
	maya.status(nil)
	assertNone("the status fetch")
	if v := maya.read(); len(v.Announcements) == 0 {
		t.Fatalf("the status view = %+v; want octo's announcement", v)
	}
	assertNone("the status read")
	ctx, cancel := context.WithTimeout(context.Background(), slack(5*time.Second))
	defer cancel()
	if _, err := PromptNotes(ctx, maya.dir, w.clock(), maya.opts()); err != nil {
		t.Fatal(err)
	}
	assertNone("the prompt block")
	if s := maya.self(); s.ID != "" || s.Login != "maya" {
		t.Fatalf("the reader is %+v; want maya's login and no id", s)
	}
	assertNone("the reader's own ids")
}
