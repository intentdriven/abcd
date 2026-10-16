package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// A drain's hand-back is published on origin (spc-2610161044461448 step 3;
// iss-2610161015370582; dcn-2610161040365515): the release of a handed-back
// issue carries `Abcd-Handed-Back: <kind>`, one of a closed vocabulary, and
// `Abcd-Record: <the blob id of the record on origin's default branch>`, and
// never free text. A reservation a drain makes backs off such a tip while the
// record's blob is unchanged; a person's own reservation goes ahead and is
// told of the hand-back.

// fileOnMain writes id's open record on origin's default branch with body, and
// returns the record's blob id there.
func (w *world) fileOnMain(id, body string) string {
	w.t.Helper()
	rel := ".abcd/work/issues/open/" + id + "-the-widget.md"
	w.seed.Write(rel, "---\nid: "+id+"\n---\n# The widget\n\n"+body+"\n")
	w.seed.Commit("file " + id)
	w.seed.Git("push", "-q", "origin", "main")
	return w.seed.Git("rev-parse", "HEAD:"+rel)
}

// trailerValues are the values of key in a commit's message as origin holds it.
func (w *world) trailerValues(sha, key string) []string {
	w.t.Helper()
	var out []string
	for _, l := range strings.Split(w.remoteMessage(sha), "\n") {
		if v, ok := strings.CutPrefix(l, key+": "); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestAHandBackTombstoneRoundTripsEveryKind: a tombstone carrying each
// hand-back kind, stuck and closed included, and a record blob reads back as
// it was written, and the reservation it releases is measured against the
// longest of them.
func TestAHandBackTombstoneRoundTripsEveryKind(t *testing.T) {
	for _, k := range []string{"stuck", "closed"} {
		if !slices.Contains(HandBackKinds, k) {
			t.Fatalf("the hand-back kinds %v do not include %q", HandBackKinds, k)
		}
	}
	blob := strings.Repeat("ab", 20)
	for _, kind := range HandBackKinds {
		m := message{
			Record: "iss-2609020716570699", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55",
			Footprint: []string{"internal/core/claim/claim.go"},
			ClaimedAt: t0, ExpiresAt: t0, ReleasedAt: t0, HandedBack: kind, RecordBlob: blob, AssistedBy: "abcd:v0.9.0",
		}
		text := m.text()
		if !strings.Contains(text, "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Handed-Back: "+kind+"\nAbcd-Record: "+blob+"\n") {
			t.Fatalf("%s: the tombstone =\n%s", kind, text)
		}
		back, reason := parseMessage("iss-2609020716570699", text)
		if reason != "" || back.HandedBack != kind || back.RecordBlob != blob || !back.released() {
			t.Fatalf("%s: the tombstone reads back as %+v (%q)", kind, back, reason)
		}
	}
	// A released tip on origin carries them too, and the view reads them.
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	want := w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	rel, err := p.do(Request{Record: "iss-5", Release: true, HandBack: "stuck"})
	if err != nil || rel.Act != ActReleased || rel.HandBackUnpublished != "" {
		t.Fatalf("the hand-back release = %+v, %v", rel, err)
	}
	tip := w.remoteTip("iss-5")
	if got := w.trailerValues(tip, "Abcd-Handed-Back"); !slices.Equal(got, []string{"stuck"}) {
		t.Fatalf("the tombstone's hand-back = %v; want stuck", got)
	}
	if got := w.trailerValues(tip, "Abcd-Record"); !slices.Equal(got, []string{want}) {
		t.Fatalf("the tombstone's record blob = %v; want %s, the record on origin's default branch", got, want)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := snap.Lookup("iss-5"); !ok || r.HandedBack != "stuck" || r.RecordChanged || r.State != StateReleased {
		t.Fatalf("the view = %+v; want the released tip handed back as stuck, its record unchanged", r)
	}
}

// TestAHandBackTombstoneCarriesNoFreeText: a release naming a kind outside the
// vocabulary is refused before anything runs, and so is a hand-back on a
// reservation; a tip whose hand-back is free text, lacks its record blob,
// names a blob that is not an object name, or rides a reservation that is not
// released, reads as unreadable.
func TestAHandBackTombstoneCarriesNoFreeText(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	before := w.remoteTip("iss-5")
	blob := strings.Repeat("ab", 20)
	for _, c := range []struct {
		req  Request
		says string
	}{
		{Request{Record: "iss-5", Release: true, HandBack: "the fix changes the page a user reads"}, "is not one of " + strings.Join(HandBackKinds, ", ")},
		{Request{Record: "iss-5", Release: true, HandBack: "Stuck"}, "is not one of " + strings.Join(HandBackKinds, ", ")},
		{Request{Record: "iss-5", Footprint: []string{"a.go"}, HandBack: "stuck"}, "only a release publishes a hand-back"},
		{Request{Record: "iss-5", Release: true, Drain: true}, "only a reservation is a drain's to back off a hand-back"},
		{Request{Record: "iss-5", Release: true, HandBackBlob: blob}, "only a hand-back names the record blob it was judged over"},
		{Request{Record: "iss-5", Release: true, HandBack: "stuck", HandBackBlob: "the widget's record"}, "is not an object name"},
	} {
		msg := p.mustErr(c.req)
		if _, err := p.do(c.req); !errors.Is(err, ErrRefused) || !strings.HasSuffix(err.Error(), nothingPushed) {
			t.Fatalf("%+v = %v; want it refused with nothing pushed", c.req, err)
		}
		if !strings.Contains(msg, c.says) {
			t.Fatalf("%+v: the refusal %q does not say %q", c.req, msg, c.says)
		}
	}
	if w.remoteTip("iss-5") != before {
		t.Fatal("a refused hand-back pushed")
	}
	head := "abcd reservation released: iss-6\n\nAbcd-Reservation: iss-6\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Footprint: a.go\nAbcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T14:00:00Z\n"
	for name, msg := range map[string]string{
		"free text":     head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Handed-Back: the fix changes the page\nAbcd-Record: " + blob + "\n",
		"no blob":       head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Handed-Back: stuck\n",
		"blob alone":    head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Record: " + blob + "\n",
		"blob not name": head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Handed-Back: stuck\nAbcd-Record: the widget's record\n",
		"two kinds":     head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Handed-Back: stuck\nAbcd-Handed-Back: closed\nAbcd-Record: " + blob + "\n",
		"not released":  head + "Abcd-Handed-Back: stuck\nAbcd-Record: " + blob + "\n",
	} {
		if _, reason := parseMessage("iss-6", msg); reason != reasonHandBack {
			t.Fatalf("%s: the tip reads with reason %q; want %q", name, reason, reasonHandBack)
		}
	}
}

// mustErr is the refusal req meets; the test fails when it meets none.
func (p *participant) mustErr(req Request) string {
	p.w.t.Helper()
	_, err := p.do(req)
	if err == nil {
		p.w.t.Fatalf("%+v was not refused", req)
	}
	return err.Error()
}

// TestADrainBacksOffAHandBackUntilItsRecordChanges: P's drain hands iss-5
// back; Q's drain reserving it backs off with nothing pushed, while the
// record on origin's default branch is unchanged, and reserves it once a
// person has changed the record there.
func TestADrainBacksOffAHandBackUntilItsRecordChanges(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(10 * time.Minute))
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: "user-visible"}); err != nil {
		t.Fatal(err)
	}
	tomb := w.remoteTip("iss-5")
	w.set(t0.Add(20 * time.Minute))
	_, err := q.do(Request{Record: "iss-5", Footprint: []string{"b.go"}, Drain: true})
	want := "iss-5 was handed back by octo as user-visible at 2026-10-03T14:10:00Z, and its record on origin's default branch is unchanged since: " +
		"a drain takes it again only once a person changes the record; take other work; nothing was pushed"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("Q's drain reserving the handed-back issue = %v; want %q", err, want)
	}
	if w.remoteTip("iss-5") != tomb {
		t.Fatal("the back-off pushed")
	}
	snap, err := View(q.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if why := snap.HandBack("iss-5", w.clock()); !strings.HasPrefix(why, "iss-5 was handed back by octo as user-visible at 2026-10-03T14:10:00Z") {
		t.Fatalf("the view's hand-back = %q; want it named", why)
	}
	w.fileOnMain("iss-5", "The widget breaks; a person settled the user-visible question: keep the page as it is.")
	res, err := q.do(Request{Record: "iss-5", Footprint: []string{"b.go"}, Drain: true})
	if err != nil || res.Act != ActReserved || res.HandedBack != nil {
		t.Fatalf("Q's drain reserving once the record changed = %+v, %v; want it reserved", res, err)
	}
	if snap, err := View(q.dir, w.clock()); err != nil || snap.HandBack("iss-5", w.clock()) != "" {
		t.Fatalf("the view still names a hand-back over a live reservation: %v", err)
	}
}

// TestAPersonsOwnReserveOverAHandBackGoesAheadWithANote: a reservation a
// person makes over a hand-back whose record is unchanged is not a drain's: it
// is made, and the result names the hand-back it was made over.
func TestAPersonsOwnReserveOverAHandBackGoesAheadWithANote(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: "trust-rule"}); err != nil {
		t.Fatal(err)
	}
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActReserved {
		t.Fatalf("Q's own reserve over the hand-back = %+v, %v; want it reserved", res, err)
	}
	if h := res.HandedBack; h == nil || h.Kind != "trust-rule" || h.Holder != "octo" || !h.At.Equal(t0) {
		t.Fatalf("the result's hand-back = %+v; want octo's trust-rule hand-back", h)
	}
	if note := res.HandedBack.Note("iss-5"); note != "iss-5 was handed back by octo as trust-rule at 2026-10-03T14:00:00Z, "+
		"and its record on origin's default branch is unchanged since; this reservation is a person's own, so it was made" {
		t.Fatalf("the note = %q", note)
	}
}

// TestAHandBackOfARecordOriginDoesNotHoldIsAPlainTombstone: with no record of
// the issue on origin's default branch there is no blob to name, so the
// release pushes a plain tombstone and says why it marks no hand-back.
func TestAHandBackOfARecordOriginDoesNotHoldIsAPlainTombstone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	rel, err := p.do(Request{Record: "iss-5", Release: true, HandBack: "closed"})
	if err != nil || rel.Act != ActReleased {
		t.Fatalf("the release = %+v, %v", rel, err)
	}
	if !strings.Contains(rel.HandBackUnpublished, "holds no one record of iss-5") {
		t.Fatalf("the release says why it marks no hand-back: %q", rel.HandBackUnpublished)
	}
	if got := w.trailerValues(w.remoteTip("iss-5"), "Abcd-Handed-Back"); got != nil {
		t.Fatalf("the plain tombstone carries a hand-back: %v", got)
	}
}
