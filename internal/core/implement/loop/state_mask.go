package loop

import (
	"slices"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// masked returns the state with the repository and tool text it stores passed
// through termsafe.Sanitize (iss-2610121542240257): each lane's step title,
// sync paths, kept reason, landing URL and merge rule, and the model a runner
// reported for a receipt or a validator; each pending step's title; the
// pick's entry and every exclusion's reason; every record note; every
// transcript's path, session, store path and scan gap; and every fallback's
// text. That text legitimately carries what a repository or a tool wrote (a
// spec in a right-to-left language, a coloured hook failure), so a control,
// bidi or zero-width rune in it is masked rather than refused: a run abcd
// wrote stays readable. The ruling is "both: mask when stored and when shown"
// (.abcd/work/DECISIONS.md, 2026-10-12): writeState stores what masked
// returns, and readStateIn returns it, so every surface that shows a run's
// state shows it masked, a run written before the mask included.
//
// The hand-back and receipt text the loop already holds to one plain line is
// refused when it is read (handBacksWritten, receiptsWritten) and is not
// touched here. Sanitize is idempotent and leaves clean text as it was, so a
// state of clean text round-trips byte for byte.
//
// masked copies every slice and pointer it rewrites, so the state it was
// called on is never changed underneath its caller.
func (st State) masked() State {
	line := termsafe.Sanitize
	st.Pending = slices.Clone(st.Pending)
	for i := range st.Pending {
		st.Pending[i].Title = line(st.Pending[i].Title)
	}
	if st.Pick != nil {
		p := *st.Pick
		p.Entry = line(p.Entry)
		p.Excluded = maskExcluded(p.Excluded)
		st.Pick = &p
	}
	st.Record = slices.Clone(st.Record)
	for i := range st.Record {
		st.Record[i].Note = line(st.Record[i].Note)
	}
	st.Transcripts = slices.Clone(st.Transcripts)
	for i := range st.Transcripts {
		st.Transcripts[i] = st.Transcripts[i].masked()
	}
	st.Fallbacks = slices.Clone(st.Fallbacks)
	for i := range st.Fallbacks {
		f := &st.Fallbacks[i]
		f.Asked, f.Detail, f.Ran = line(f.Asked), line(f.Detail), line(f.Ran)
	}
	st.Lanes = slices.Clone(st.Lanes)
	for i := range st.Lanes {
		st.Lanes[i] = st.Lanes[i].masked()
	}
	return st
}

// masked is State.masked for one lane.
func (l Lane) masked() Lane {
	line := termsafe.Sanitize
	l.StepTitle = line(l.StepTitle)
	l.Awaits = slices.Clone(l.Awaits)
	for i := range l.Awaits {
		l.Awaits[i].Kept = line(l.Awaits[i].Kept)
	}
	l.Syncs = slices.Clone(l.Syncs)
	for i := range l.Syncs {
		l.Syncs[i].Paths = termsafe.SanitizeAll(l.Syncs[i].Paths)
	}
	if l.Landing != nil {
		ld := *l.Landing
		ld.PRURL, ld.Merge = line(ld.PRURL), line(ld.Merge)
		l.Landing = &ld
	}
	l.Receipts = slices.Clone(l.Receipts)
	for i := range l.Receipts {
		l.Receipts[i].Route = maskRoute(l.Receipts[i].Route)
	}
	l.Validation = slices.Clone(l.Validation)
	for i := range l.Validation {
		vs := slices.Clone(l.Validation[i].Validators)
		for j := range vs {
			vs[j].Route = maskRoute(vs[j].Route)
		}
		l.Validation[i].Validators = vs
	}
	return l
}

// masked is State.masked for one captured transcript.
func (t Transcript) masked() Transcript {
	t.Path, t.Session = termsafe.Sanitize(t.Path), termsafe.Sanitize(t.Session)
	t.Stored, t.ScanGap = termsafe.Sanitize(t.Stored), termsafe.Sanitize(t.ScanGap)
	return t
}

// maskRoute masks the route a runner reported, its model above all.
func maskRoute(r *runner.RouteRecord) *runner.RouteRecord {
	if r == nil {
		return nil
	}
	m := *r
	m.Asked, m.Ran, m.Model = termsafe.Sanitize(m.Asked), termsafe.Sanitize(m.Ran), termsafe.Sanitize(m.Model)
	return &m
}

// maskExcluded masks each exclusion's reason: a check's detail quotes the
// records it read.
func maskExcluded(ex []Excluded) []Excluded {
	if ex == nil {
		return nil
	}
	out := slices.Clone(ex)
	for i := range out {
		out[i].ID, out[i].Check, out[i].Reason = termsafe.Sanitize(out[i].ID), termsafe.Sanitize(out[i].Check), termsafe.Sanitize(out[i].Reason)
	}
	return out
}
