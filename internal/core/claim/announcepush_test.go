package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// The announcement before abcd's own pushes (iss-2610161645224876,
// dcn-2610170704189712): AnnounceBeforePush posts "merging <item>" through
// Announce for each item, sends nothing for one whose announcement still
// stands, passes over a record that is not public yet, and answers every other
// failure, so the caller pushes nothing.

func (p *participant) beforePush(items []string, e *embargoProbe) ([]PushNotice, error) {
	o := p.opts()
	o.Embargo = e.check
	return AnnounceBeforePush(p.dir, items, o)
}

// TestPushItemsKeepsTheAnnouncementItemsInTheirCanonicalForm: only issue and
// intent ids and pull-request numbers are items, each once, first kept.
func TestPushItemsKeepsTheAnnouncementItemsInTheirCanonicalForm(t *testing.T) {
	got := PushItems([]string{"iss-5", "spc-2610150728396006", "dcn-2610170655202547", "#12", "12", "itd-7", "iss-5", "the widget", "ISS-5"})
	if want := []string{"iss-5", "#12", "itd-7"}; !slices.Equal(got, want) {
		t.Fatalf("PushItems = %v; want %v", got, want)
	}
}

// TestAStandingAnnouncementIsNotSentAgainBeforeAPush: the first push announces
// merging iss-5; a second before it lapses sends nothing and reports the one
// standing; once less than a minute of it is left, a new one is sent.
func TestAStandingAnnouncementIsNotSentAgainBeforeAPush(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	e := &embargoProbe{}
	first, err := p.beforePush([]string{"iss-5"}, e)
	if err != nil || len(first) != 1 || first[0].Live || first[0].NotPublic {
		t.Fatalf("first push's announcement = %+v, %v; want one sent", first, err)
	}
	tip := w.statusTip("octo")
	if first[0].Line() != "announced merging iss-5 on abcd-status/octo, standing until 2026-10-03T14:30:00Z" {
		t.Fatalf("line = %q", first[0].Line())
	}

	w.set(t0.Add(20 * time.Minute))
	again, err := p.beforePush([]string{"iss-5"}, e)
	if err != nil || len(again) != 1 || !again[0].Live || w.statusTip("octo") != tip {
		t.Fatalf("second push's announcement = %+v, %v, tip %s→%s; want the standing one, nothing sent", again, err, tip, w.statusTip("octo"))
	}
	if again[0].Line() != "merging iss-5 was already announced on abcd-status/octo, standing until 2026-10-03T14:30:00Z" {
		t.Fatalf("line = %q", again[0].Line())
	}
	// Another item's announcement on top does not hide the one standing.
	if _, err := p.announce("asking", "#7"); err != nil {
		t.Fatal(err)
	}
	if n, err := p.beforePush([]string{"iss-5"}, e); err != nil || !n[0].Live {
		t.Fatalf("with another item's asking on top, the merging one still stands: %+v, %v", n, err)
	}
	// Another kind of the same item on top is what readers now show for it,
	// so the merging under it no longer stands, and a new one is sent.
	if _, err := p.announce("asking", "iss-5"); err != nil {
		t.Fatal(err)
	}
	asking := w.statusTip("octo")
	n, err := p.beforePush([]string{"iss-5"}, e)
	if err != nil || len(n) != 1 || n[0].Live || w.parentOf(w.statusTip("octo")) != asking {
		t.Fatalf("with an asking of iss-5 on top = %+v, %v; want a new merging sent on top of it", n, err)
	}

	w.set(t0.Add(49*time.Minute + 30*time.Second))
	before := w.statusTip("octo")
	late, err := p.beforePush([]string{"iss-5"}, e)
	if err != nil || len(late) != 1 || late[0].Live || w.statusTip("octo") == before {
		t.Fatalf("with under a minute left = %+v, %v; want a new one sent", late, err)
	}
	if w.parentOf(w.statusTip("octo")) != before {
		t.Fatalf("the new announcement is not a child of the branch's tip")
	}
}

// TestAMergedOnTopEndsTheStandingMerging: octo lands naming iss-5 (merging
// sent), then announces merged iss-5; a follow-up landing ten minutes later
// naming iss-5 again announces merging anew, since every reader now shows
// "merged iss-5", the newest of the item.
func TestAMergedOnTopEndsTheStandingMerging(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	e := &embargoProbe{}
	if n, err := p.beforePush([]string{"iss-5"}, e); err != nil || len(n) != 1 || n[0].Live {
		t.Fatalf("first push's announcement = %+v, %v; want one sent", n, err)
	}
	if _, err := p.announce("merged", "iss-5"); err != nil {
		t.Fatal(err)
	}
	merged := w.statusTip("octo")
	w.set(t0.Add(10 * time.Minute))
	n, err := p.beforePush([]string{"iss-5"}, e)
	if err != nil || len(n) != 1 || n[0].Live || w.parentOf(w.statusTip("octo")) != merged {
		t.Fatalf("follow-up push after merged iss-5 = %+v, %v; want a new merging sent on top of the merged", n, err)
	}
}

// TestALostRaceRechecksForAStandingAnnouncement: two sessions of one person
// push work naming iss-5 at the same moment. The session whose push loses
// the race finds, on the new tip, the merging iss-5 the other just sent, and
// sends nothing more: one announcement, not two.
func TestALostRaceRechecksForAStandingAnnouncement(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.announce("asking", "#7"); err != nil {
		t.Fatal(err)
	}
	var rival string
	beforePush = func(root string) {
		if root != p.dir {
			return
		}
		beforePush = nil
		n, err := q.beforePush([]string{"iss-5"}, &embargoProbe{})
		if err != nil || len(n) != 1 || n[0].Live {
			t.Errorf("the rival announcement: %+v, %v", n, err)
		}
		rival = w.statusTip("octo")
	}
	t.Cleanup(func() { beforePush = nil })
	n, err := p.beforePush([]string{"iss-5"}, &embargoProbe{})
	if err != nil || len(n) != 1 || !n[0].Live || rival == "" || w.statusTip("octo") != rival {
		t.Fatalf("the push that lost the race = %+v, %v, tip %s, rival %s; want the rival's standing one, nothing more sent", n, err, w.statusTip("octo"), rival)
	}
}

// TestAStandingAnnouncementOfAnotherPersonIsNotTheOneOwed: a colleague's own
// merging iss-5 is theirs; this person's push still announces its own.
func TestAStandingAnnouncementOfAnotherPersonIsNotTheOneOwed(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := q.beforePush([]string{"iss-5"}, &embargoProbe{}); err != nil {
		t.Fatal(err)
	}
	p := w.clone("1234+octo@users.noreply.github.com", "")
	n, err := p.beforePush([]string{"iss-5"}, &embargoProbe{})
	if err != nil || len(n) != 1 || n[0].Live || w.statusTip("octo") == "" {
		t.Fatalf("octo's push after quinn's announcement = %+v, %v; want octo's own sent", n, err)
	}
}

// TestARecordNotYetPublicIsPassedOverBeforeAPush: a record only the branch
// holds is not announced, and the notice says so; the items after it are
// still announced.
func TestARecordNotYetPublicIsPassedOverBeforeAPush(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	n, err := p.beforePush([]string{"iss-9", "iss-5"}, &embargoProbe{})
	if err != nil || len(n) != 2 || !n[0].NotPublic || n[0].Item != "iss-9" || n[1].NotPublic || n[1].Live {
		t.Fatalf("notices = %+v, %v; want iss-9 passed over and iss-5 sent", n, err)
	}
	if n[0].Line() != "did not announce iss-9: it is not on origin's default branch yet, and an announcement names only a public record" {
		t.Fatalf("line = %q", n[0].Line())
	}
	var np *NotPublicError
	if _, err := p.announce("merging", "iss-9"); !errors.As(err, &np) || np.Item != "iss-9" || !errors.Is(err, ErrRefused) {
		t.Fatalf("announce iss-9 = %v; want a *NotPublicError, an ErrRefused refusal", err)
	}
}

// TestAnAnnouncementThatCannotBePublishedStopsBeforeThePush: an embargo store
// that cannot be checked refuses the announcement, and AnnounceBeforePush
// answers it naming the item, with what was sent before it.
func TestAnAnnouncementThatCannotBePublishedStopsBeforeThePush(t *testing.T) {
	w := newWorld(t)
	w.publish(".abcd/work/issues/open/iss-5-the-widget.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	n, err := p.beforePush([]string{"iss-5"}, &embargoProbe{err: errors.New("the store is unreadable")})
	var pe *PushAnnounceError
	if !errors.As(err, &pe) || pe.Item != "iss-5" || len(n) != 0 || w.statusTip("octo") != "" {
		t.Fatalf("notices %+v, err %v, tip %q; want a *PushAnnounceError for iss-5 and nothing sent", n, err, w.statusTip("octo"))
	}
	if !strings.HasPrefix(err.Error(), "the announcement merging iss-5 could not be published: an announcement about iss-5 is refused: the embargo store on this machine could not be checked: the store is unreadable") {
		t.Fatalf("error = %q", err)
	}
}
