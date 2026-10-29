package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// A reservation, a renewal and a take-over consult the embargo check, as an
// announcement does, before the remote is asked anything.

// reserveHeld is p.do with the embargo check e.
func (p *participant) reserveHeld(req Request, e *embargoProbe) (Result, error) {
	o := p.opts()
	if e != nil {
		o.Embargo = e.check
	}
	return Reserve(p.dir, req, o)
}

func TestAReservationOfARecordUnderEmbargoIsRefusedBeforeTheRemote(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")

	held := &embargoProbe{held: true}
	log := traced(t)
	_, err := p.reserveHeld(Request{Record: "iss-5", Footprint: []string{"internal/a.go"}}, held)
	want := "a reservation of iss-5 is refused: the embargo store on this machine holds it, or text the reservation carries, under embargo; nothing was pushed"
	if !errors.Is(err, ErrRefused) || err.Error() != want {
		t.Fatalf("reserve = %v; want %q", err, want)
	}
	if len(held.texts) != 1 || !strings.Contains(held.texts[0], "Abcd-Footprint: internal/a.go") ||
		!slices.Equal(held.records[0], []string{"iss-5"}) {
		t.Fatalf("the check was asked %q about %q; want the reservation's message and its record", held.texts, held.records)
	}
	if slices.Contains(log.subcommands(), "fetch") || slices.Contains(log.subcommands(), "push") {
		t.Fatalf("the remote was asked before the embargo check refused: %v", log.subcommands())
	}
	if tip := w.remoteTip("iss-5"); tip != "" {
		t.Fatalf("a refused reservation reached origin: %s", tip)
	}

	_, err = p.reserveHeld(Request{Record: "iss-5", Footprint: []string{"a.go"}}, &embargoProbe{err: errors.New("the store's state file is refused")})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "the embargo store on this machine could not be checked: the store's state file is refused") {
		t.Fatalf("reserve with a store that cannot be checked = %v; want it refused", err)
	}
	if tip := w.remoteTip("iss-5"); tip != "" {
		t.Fatalf("a refused reservation reached origin: %s", tip)
	}
}

// TestARenewalAndATakeOverAreHeldToTheEmbargoCheckButAReleaseIsNot: a renewal
// and a take-over of a record under embargo are refused, and its reservation
// can still be released.
func TestARenewalAndATakeOverAreHeldToTheEmbargoCheckButAReleaseIsNot(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserveHeld(Request{Record: "iss-5", Footprint: []string{"a.go"}}, &embargoProbe{}); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	tip := w.remoteTip("iss-5")
	for _, req := range []Request{{Record: "iss-5", Renew: true}, {Record: "iss-5", Footprint: []string{"a.go"}, TakeOver: true}} {
		if _, err := p.reserveHeld(req, &embargoProbe{held: true}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "under embargo") {
			t.Fatalf("%+v = %v; want the embargo refusal", req, err)
		}
		if got := w.remoteTip("iss-5"); got != tip {
			t.Fatalf("%+v moved origin's tip from %s to %s", req, tip, got)
		}
	}
	rel := &embargoProbe{held: true}
	if res, err := p.reserveHeld(Request{Record: "iss-5", Release: true}, rel); err != nil || res.Act != ActReleased {
		t.Fatalf("release = %+v, %v; want it released", res, err)
	}
	if len(rel.texts) != 0 {
		t.Fatalf("a release asked the embargo check %q; want it not asked", rel.texts)
	}
}

// TestAReleaseSubjectIsCheckedByItsTextAlone: a release subject is not a
// record id the store links to an advisory, so only its message is asked.
func TestAReleaseSubjectIsCheckedByItsTextAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	e := &embargoProbe{}
	if _, err := p.reserveHeld(Request{Record: "rel-v9.9.9"}, e); err != nil {
		t.Fatalf("reserve a release subject: %v", err)
	}
	if len(e.texts) != 1 || len(e.records[0]) != 0 {
		t.Fatalf("the check was asked %q about %q; want the message and no record", e.texts, e.records)
	}
}

// TestACallerNamingNoCheckUsesTheRegisteredOne: a caller whose
// Options name no check is held to the one the front door registered, and with
// neither the act is refused.
func TestACallerNamingNoCheckUsesTheRegisteredOne(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")

	t.Cleanup(func() { defaultEmbargo.Store(nil) })
	reg := &embargoProbe{held: true}
	RegisterEmbargoCheck(reg.check)
	if _, err := p.reserveHeld(Request{Record: "iss-5", Footprint: []string{"a.go"}}, nil); !errors.Is(err, ErrRefused) ||
		!strings.Contains(err.Error(), "under embargo") || len(reg.texts) != 1 {
		t.Fatalf("reserve with only the registered check = %v (asked %d times); want the embargo refusal", err, len(reg.texts))
	}
	own := &embargoProbe{}
	if _, err := p.reserveHeld(Request{Record: "iss-5", Footprint: []string{"a.go"}}, own); err != nil || len(own.texts) != 1 || len(reg.texts) != 1 {
		t.Fatalf("reserve with its own check = %v; want the caller's check asked, not the registered one", err)
	}

	defaultEmbargo.Store(nil)
	saved := uncheckedInTests
	uncheckedInTests = false
	t.Cleanup(func() { uncheckedInTests = saved })
	if _, err := p.reserveHeld(Request{Record: "iss-6", Footprint: []string{"a.go"}}, nil); !errors.Is(err, ErrRefused) ||
		!strings.Contains(err.Error(), "no embargo check was given") {
		t.Fatalf("reserve with no embargo check = %v; want it refused", err)
	}
	if tip := w.remoteTip("iss-6"); tip != "" {
		t.Fatalf("a refused reservation reached origin: %s", tip)
	}
}
