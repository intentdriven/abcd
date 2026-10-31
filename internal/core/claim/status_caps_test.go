package claim

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// What one read of the status takes in, and what it counts past its caps.

// TestStatusBranchesPastTheCapAreCountedNotRead: one listing fetches at most
// maxStatusBranches status branches, the first in origin's order, and counts
// the rest; an intake reads at most as many of the branches this checkout
// holds, and counts the rest, whatever fetch brought them.
func TestStatusBranchesPastTheCapAreCountedNotRead(t *testing.T) {
	old := maxStatusBranches
	maxStatusBranches = 2
	t.Cleanup(func() { maxStatusBranches = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	for _, b := range []string{"aa", "bb", "cc"} {
		w.pushStatus(b, announcementMsg("merging", "#1", b, t0, 30*time.Minute), "")
	}
	pat.status(nil)
	v := pat.read()
	if len(v.Announcements) != 2 || v.Announcements[0].Branch == "cc" || v.Announcements[1].Branch == "cc" || v.BranchesNotRead != 1 {
		t.Fatalf("announcements = %+v, branches not read = %d; want aa's and bb's, and cc counted", v.Announcements, v.BranchesNotRead)
	}
	if refs := pat.git("for-each-ref", "--format=%(refname)", StatusTrackingPrefix); strings.Contains(refs, "/cc") {
		t.Fatalf("cc was fetched past the cap: %s", refs)
	}
	if got := pat.notes(); !strings.Contains(got, "- 1 status branch past the 2 one read takes, not read\n") {
		t.Fatalf("the block:\n%s\nwant the branch past the cap counted", got)
	}
	// A fetch of the whole namespace brings cc in: the intake still reads two.
	pat.git("fetch", "-q", "origin", "+"+StatusRefPrefix+"*:"+StatusTrackingPrefix+"*")
	if v := pat.read(); len(v.Announcements) != 2 || v.BranchesNotRead != 1 {
		t.Fatalf("with cc held, announcements = %+v, branches not read = %d; want two read and one counted", v.Announcements, v.BranchesNotRead)
	}
}

// TestKeptAnnouncementsPastTheCapAreCounted: one read takes in at most
// maxKeptAnnouncements of the announcements kept here, the newest kept, and
// counts the rest.
func TestKeptAnnouncementsPastTheCapAreCounted(t *testing.T) {
	old := maxKeptAnnouncements
	maxKeptAnnouncements = 3
	t.Cleanup(func() { maxKeptAnnouncements = old })
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	for i := 1; i <= 5; i++ {
		maya.announceAs("merging", fmt.Sprintf("#%d", i))
		pat.status(nil)
		w.set(w.clock().Add(time.Second))
	}
	v := pat.read()
	if len(v.Announcements) != 3 || v.Announcements[0].Item != "#5" || v.Announcements[2].Item != "#3" || v.AnnouncementsNotRead != 2 {
		t.Fatalf("announcements = %+v, not read = %d; want #5 to #3, and two counted", v.Announcements, v.AnnouncementsNotRead)
	}
	if got := pat.notes(); !strings.Contains(got, "- 2 announcements kept here past the 3 one read takes, not shown\n") {
		t.Fatalf("the block:\n%s\nwant the announcements past the cap counted", got)
	}
}

// TestAListingPastTheOutputBoundStillReads: origin lists more status branches
// than a megabyte of `git ls-remote` output holds. The listing is read line by
// line, never whole, so the fetch still reads the branches it names and
// counts every name it does not read.
func TestAListingPastTheOutputBoundStillReads(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	var names []string
	for i := 0; i < 12000; i++ {
		names = append(names, fmt.Sprintf("Z-%05d-%s", i, strings.Repeat("z", 40)))
	}
	w.pushNames(names)
	runGit(t, w.bare, "pack-refs", "--all")
	pat.status(nil)
	v := pat.read()
	if len(v.Announcements) != 1 || v.Announcements[0].Sender != maya.keptHolderID() || v.UnreadNames != 12000 {
		t.Fatalf("announcements = %+v, unread names = %d; want maya's, and 12000 counted", v.Announcements, v.UnreadNames)
	}
}
