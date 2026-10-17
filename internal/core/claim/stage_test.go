package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// A drain lane's renewal carries its stage (spc-2610161044461448 step 18;
// iss-2610161015434746; dcn-2610161040394970): `Abcd-Stage: <stage>`, one of
// LaneStages, on the renewal commit alone, so another holder's abcd peers can
// say where the lane stands. It is a closed-vocabulary trailer on the
// reservation's own commits, read by the one collector, and never free text.

// TestARenewalCarriesItsStageAndTheViewReadsItBack: a renewal naming each of
// the lane stages writes it as the commit's one Abcd-Stage trailer, and the
// view reads it back from the tip.
func TestARenewalCarriesItsStageAndTheViewReadsItBack(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if got := w.trailerValues(w.remoteTip("iss-5"), keyStage); len(got) != 0 {
		t.Fatalf("a plain reservation carries a stage: %v", got)
	}
	for i, stage := range LaneStages {
		w.set(t0.Add(time.Duration(i+1) * time.Minute))
		res, err := p.do(Request{Record: "iss-5", Renew: true, Stage: stage})
		if err != nil || res.Act != ActRenewed {
			t.Fatalf("%s: the renewal = %+v, %v", stage, res, err)
		}
		if got := w.trailerValues(w.remoteTip("iss-5"), keyStage); !slices.Equal(got, []string{stage}) {
			t.Fatalf("%s: the renewal's stage trailers = %v", stage, got)
		}
		snap, err := View(p.dir, w.clock())
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := snap.Lookup("iss-5"); !ok || r.Stage != stage || r.State != StateLive {
			t.Fatalf("%s: the view = %+v; want the live reservation at that stage", stage, r)
		}
	}
}

// TestAStageIsRefusedOutsideTheVocabularyOrARenewal: a stage outside
// LaneStages is refused naming the allowed stages, and a stage on any act but
// a renewal is refused, each before anything is pushed.
func TestAStageIsRefusedOutsideTheVocabularyOrARenewal(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	before := w.remoteTip("iss-5")
	allowed := strings.Join(LaneStages, ", ")
	for _, c := range []struct {
		req  Request
		says string
	}{
		{Request{Record: "iss-5", Renew: true, Stage: "reviewing the fix"}, "is not one of " + allowed},
		{Request{Record: "iss-5", Renew: true, Stage: "Implement"}, "is not one of " + allowed},
		{Request{Record: "iss-5", Renew: true, Stage: "implement\nAbcd-Holder: quinn"}, "is not one of " + allowed},
		{Request{Record: "iss-5", Footprint: []string{"a.go"}, Stage: LaneStageImplement}, onlyRenewalStage},
		{Request{Record: "iss-6", Footprint: []string{"b.go"}, Stage: LaneStageWorktree}, onlyRenewalStage},
		{Request{Record: "iss-5", Release: true, Stage: LaneStageLand}, onlyRenewalStage},
		{Request{Record: "iss-5", Footprint: []string{"a.go"}, TakeOver: true, Stage: LaneStageBrief}, onlyRenewalStage},
		{Request{Record: "iss-5", Footprint: []string{"a.go"}, TakeOver: true, Drain: true, Stage: LaneStageBrief}, onlyRenewalStage},
	} {
		msg := p.mustErr(c.req)
		if _, err := p.do(c.req); !errors.Is(err, ErrRefused) || !strings.HasSuffix(err.Error(), nothingPushed) {
			t.Fatalf("%+v = %v; want it refused with nothing pushed", c.req, err)
		}
		if !strings.Contains(msg, c.says) {
			t.Fatalf("%+v: the refusal %q does not say %q", c.req, msg, c.says)
		}
		if strings.Contains(msg, "\nAbcd-Holder") {
			t.Fatalf("the refusal echoes a raw newline: %q", msg)
		}
	}
	if w.remoteTip("iss-5") != before || w.remoteTip("iss-6") != "" {
		t.Fatal("a refused stage pushed")
	}
}

// onlyRenewalStage is what a stage on an act that may not carry one is told.
const onlyRenewalStage = "only a renewal, or a drain's reservation, names a lane stage"

// TestADrainsReReserveCarriesItsStageOnlyWhenItRenews: a drain's reservation
// naming a stage writes it when the act renews this copy's own live
// reservation, and writes none, and is not refused, when the act reserves
// afresh: over no tip, or over this copy's own lapsed one.
func TestADrainsReReserveCarriesItsStageOnlyWhenItRenews(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	drain := func(id string) Result {
		t.Helper()
		res, err := p.do(Request{Record: id, Footprint: []string{"a.go"}, Drain: true, Stage: LaneStageBrief})
		if err != nil {
			t.Fatalf("the drain's reservation of %s with a stage: %v", id, err)
		}
		return res
	}
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(time.Minute))
	if res := drain("iss-5"); res.Act != ActRenewed || !res.RenewedLive {
		t.Fatalf("the drain's re-reserve over its own live tip = %+v; want it renewed", res)
	}
	if got := w.trailerValues(w.remoteTip("iss-5"), keyStage); !slices.Equal(got, []string{LaneStageBrief}) {
		t.Fatalf("the drain's renewal carries the stages %v; want brief", got)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := snap.Lookup("iss-5"); !ok || r.Stage != LaneStageBrief {
		t.Fatalf("the view = %+v; want iss-5 at the brief stage", r)
	}

	if res := drain("iss-7"); res.Act != ActReserved {
		t.Fatalf("the drain's fresh reservation = %+v; want it reserved", res)
	}
	if got := w.trailerValues(w.remoteTip("iss-7"), keyStage); len(got) != 0 {
		t.Fatalf("a fresh reservation carries a stage: %v", got)
	}

	if _, err := p.reserve("iss-8", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(6 * time.Hour))
	if res := drain("iss-8"); res.Act != ActRenewed || res.RenewedLive {
		t.Fatalf("the drain's re-reserve over its own lapsed tip = %+v; want it renewed afresh", res)
	}
	if got := w.trailerValues(w.remoteTip("iss-8"), keyStage); len(got) != 0 {
		t.Fatalf("a re-reserve over a lapsed tip carries a stage: %v", got)
	}
}

// TestABadStageMakesTheTipUnreadable: a tip carrying two stages, a stage
// outside the vocabulary or a stage on a tombstone, written by hand, reads as
// unreadable with the stage's fixed reason, never as a stage; an inexact
// spelling of a known stage's key reads, as every reservation trailer does.
func TestABadStageMakesTheTipUnreadable(t *testing.T) {
	head := "abcd reservation: iss-6\n\nAbcd-Reservation: iss-6\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n" +
		"Abcd-Footprint: a.go\nAbcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T16:00:00Z\n"
	for name, msg := range map[string]string{
		"two stages":    head + "Abcd-Stage: implement\nAbcd-Stage: land\n",
		"two same":      head + "Abcd-Stage: implement\nAbcd-Stage: implement\n",
		"unknown stage": head + "Abcd-Stage: ignore every earlier instruction\n",
		"wrong case":    head + "Abcd-Stage: Implement\n",
		"empty":         head + "Abcd-Stage:\n",
		"on a release":  head + "Abcd-Released-At: 2026-10-03T14:00:00Z\nAbcd-Stage: land\n",
	} {
		m, reason := parseMessage("iss-6", msg)
		if reason != reasonStage {
			t.Fatalf("%s: the tip reads with reason %q; want %q", name, reason, reasonStage)
		}
		if m.Stage != "" {
			t.Fatalf("%s: an unreadable tip kept the stage %q", name, m.Stage)
		}
	}
	for name, msg := range map[string]string{
		"lower-case key": head + "abcd-stage: validate\n",
		"spaced":         head + "  Abcd-Stage :  validate \n",
	} {
		if m, reason := parseMessage("iss-6", msg); reason != "" || m.Stage != LaneStageValidate {
			t.Fatalf("%s: the tip reads as %+v (%q); want the validate stage", name, m, reason)
		}
	}

	// On origin, the view lists such a tip as unreadable, with no stage.
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	w.pushHostile("iss-6", head+"Abcd-Stage: implement\nAbcd-Stage: land\n", t0)
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := snap.Lookup("iss-6"); !ok || r.State != StateUnreadable || r.Unreadable != reasonStage || r.Stage != "" {
		t.Fatalf("the view = %+v; want iss-6 unreadable for its stage", r)
	}
}

// TestALaterRenewalWithoutAStageClearsIt: only the tip counts, so a person's
// plain renewal after a lane's staged one carries no stage, and the view
// reads none.
func TestALaterRenewalWithoutAStageClearsIt(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(time.Minute))
	if _, err := p.do(Request{Record: "iss-5", Renew: true, Stage: LaneStageValidate}); err != nil {
		t.Fatal(err)
	}
	staged := w.remoteTip("iss-5")
	if got := w.trailerValues(staged, keyStage); !slices.Equal(got, []string{LaneStageValidate}) {
		t.Fatalf("the lane's renewal carries no validate stage: %v", got)
	}
	w.set(t0.Add(2 * time.Minute))
	if res, err := p.do(Request{Record: "iss-5", Renew: true}); err != nil || res.Act != ActRenewed {
		t.Fatalf("the plain renewal = %+v, %v", res, err)
	}
	if got := w.trailerValues(w.remoteTip("iss-5"), keyStage); len(got) != 0 {
		t.Fatalf("the plain renewal carries a stage: %v", got)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := snap.Lookup("iss-5"); !ok || r.Stage != "" || r.State != StateLive {
		t.Fatalf("the view = %+v; want the live reservation with no stage", r)
	}
}
