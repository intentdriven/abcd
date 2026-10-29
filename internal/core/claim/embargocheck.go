package claim

import (
	"sync/atomic"
	"testing"
)

// defaultEmbargo is the embargo check an act whose Options name none is held
// to. The front door registers it once (RegisterEmbargoCheck), so every caller
// that builds its own Options, the implement loop's among them, is held to the
// same check without naming it.
var defaultEmbargo atomic.Pointer[EmbargoCheck]

// RegisterEmbargoCheck sets the embargo check an act uses when its Options
// name none. The front door calls it once, before any act runs.
func RegisterEmbargoCheck(c EmbargoCheck) {
	if c == nil {
		defaultEmbargo.Store(nil)
		return
	}
	defaultEmbargo.Store(&c)
}

// uncheckedInTests lets a test binary that registered no check and named none
// reserve unchecked: a package's own tests build Options without the front
// door. Every other binary is refused instead.
var uncheckedInTests = testing.Testing()

// embargoCheck is the check o holds an act to: its own, else the registered
// one, else nil.
func (o Options) embargoCheck() EmbargoCheck {
	if o.Embargo != nil {
		return o.Embargo
	}
	if c := defaultEmbargo.Load(); c != nil {
		return *c
	}
	return nil
}

// checkEmbargo asks the embargo check about the message a reservation, a
// renewal or a take-over would push, and the record it names, before the remote
// is asked anything. A held record or text, a check that cannot run, and no
// check at all refuse the act. A release is not asked, so a reservation can
// always be ended.
func checkEmbargo(p *prepared, o Options) error {
	if p.req.Release {
		return nil
	}
	check := o.embargoCheck()
	if check == nil {
		if uncheckedInTests {
			return nil
		}
		return refuse("the embargo store on this machine could not be checked: no embargo check was given; %s", nothingPushed)
	}
	m := p.reservation()
	if len(p.req.Narrow) > 0 {
		m.Footprint = p.req.Narrow
	}
	var records []string
	if !isRelease(p.id) {
		records = []string{p.id}
	}
	held, err := check(p.root, p.home, m.text(), records)
	if err != nil {
		return refuse("a reservation of %s is refused: the embargo store on this machine could not be checked: %s; %s",
			p.id, redacted(err), nothingPushed)
	}
	if held {
		return refuse("a reservation of %s is refused: the embargo store on this machine holds it, or text the reservation carries, under embargo; %s",
			p.id, nothingPushed)
	}
	return nil
}
