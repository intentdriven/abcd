package loop

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// masked returns the state with every string it stores passed through
// termsafe's mask (iss-2610121542240257): every id, path, sha, step title,
// note and reason, at any depth, by the explicit walk below, which
// TestMaskedReachesEveryStringOfTheState holds complete: a string field added
// to the state without a line here fails that test by its path. The text a
// repository or a tool wrote legitimately carries what they wrote (a spec in a
// right-to-left language, a coloured hook failure), so a control, bidi or
// zero-width rune in it is masked rather than refused: a run abcd wrote stays
// readable. The ruling is "both: mask when stored and when shown"
// (.abcd/work/DECISIONS.md, 2026-10-12): writeState stores what masked
// returns, and readStateIn returns it, so every surface that shows a run's
// state shows it masked, a run written before the mask included.
//
// One line is masked whole (termsafe.Sanitize), and free text a reader is
// shown has the home directory redacted first (free); a transcript's scan gap
// keeps its line breaks, and the prose a lane hands in (a resolution's note
// and grounds) its line breaks and tabs. The hand-back
// and receipt text the loop already holds to one plain line is also refused
// when it is read (handBacksWritten, receiptsWritten), so here it is clean
// already. The mask is idempotent and leaves clean text as it was, so a state
// of clean text round-trips byte for byte, and an id or a step title compared
// with the text it came from (brief.go's step title, the pick's commit) still
// equals it.
//
// masked copies every slice and pointer it rewrites, so the state it was
// called on is never changed underneath its caller.
func (st State) masked() State {
	st.RunID, st.Key, st.Intent, st.Spec = line(st.RunID), line(st.Key), line(st.Intent), line(st.Spec)
	st.Driver = Driver(line(string(st.Driver)))
	st.Pace = maskPtr(st.Pace, Pace.masked)
	st.Pick = maskPtr(st.Pick, RunPick.masked)
	st.Lanes = maskEach(st.Lanes, Lane.masked)
	st.Waiting = maskEach(st.Waiting, func(w Waiting) Waiting {
		w.Lane, w.Role = line(w.Lane), line(w.Role)
		return w
	})
	st.Pending = maskEach(st.Pending, func(p PendingStep) PendingStep {
		p.Title = line(p.Title)
		return p
	})
	st.Record = maskEach(st.Record, func(e Entry) Entry {
		e.Lane, e.Stage, e.Note = line(e.Lane), line(e.Stage), free(e.Note)
		return e
	})
	st.Transcripts = maskEach(st.Transcripts, Transcript.masked)
	st.Fallbacks = maskEach(st.Fallbacks, func(f runner.FallbackReceipt) runner.FallbackReceipt {
		f.Role, f.Asked, f.Detail, f.Ran = line(f.Role), line(f.Asked), free(f.Detail), line(f.Ran)
		f.Reason = runner.Reason(line(string(f.Reason)))
		return f
	})
	return st
}

// masked is State.masked for one lane.
func (l Lane) masked() Lane {
	l.ID, l.Key, l.StepTitle = line(l.ID), line(l.Key), line(l.StepTitle)
	l.Stage = Stage(line(string(l.Stage)))
	l.Awaits = maskEach(l.Awaits, func(a Await) Await {
		a.Role, a.Brief, a.Receipt, a.Kept = line(a.Role), line(a.Brief), line(a.Receipt), free(a.Kept)
		a.Answered = maskPtr(a.Answered, func(r RouteAnswer) RouteAnswer {
			r.Route, r.SHA256 = line(r.Route), line(r.SHA256)
			return r
		})
		return a
	})
	l.Syncs = maskEach(l.Syncs, func(s Sync) Sync {
		s.Siblings, s.Paths = maskEach(s.Siblings, line), maskEach(s.Paths, line)
		s.Merged, s.Brief, s.Receipt, s.Head = line(s.Merged), line(s.Brief), line(s.Receipt), line(s.Head)
		return s
	})
	l.Hold = maskPtr(l.Hold, func(h Hold) Hold {
		h.Cause, h.Head, h.Before = line(h.Cause), line(h.Head), line(h.Before)
		return h
	})
	l.Branch, l.BaseSHA, l.HeadSHA = line(l.Branch), line(l.BaseSHA), line(l.HeadSHA)
	l.Worktree, l.Brief, l.Receipt, l.PickSHA = line(l.Worktree), line(l.Brief), line(l.Receipt), line(l.PickSHA)
	l.Validation = maskEach(l.Validation, ValidationRound.masked)
	l.HandBack = maskPtr(l.HandBack, HandBack.masked)
	l.Receipts = maskEach(l.Receipts, func(r ReceiptRecord) ReceiptRecord {
		r.Role, r.Receipt, r.Model = line(r.Role), line(r.Receipt), line(r.Model)
		r.Route = maskPtr(r.Route, maskRoute)
		return r
	})
	l.Resolves = maskEach(l.Resolves, func(r Resolution) Resolution {
		r.Issue, r.Commit, r.Impact = line(r.Issue), line(r.Commit), line(r.Impact)
		r.Note, r.Grounds = prose(r.Note), prose(r.Grounds)
		return r
	})
	l.Landing = maskPtr(l.Landing, func(ld Landing) Landing {
		ld.Records, ld.PreflightReceipt, ld.Pushed = line(ld.Records), line(ld.PreflightReceipt), line(ld.Pushed)
		ld.Body, ld.PRURL, ld.Merge, ld.Merged = line(ld.Body), line(ld.PRURL), line(ld.Merge), line(ld.Merged)
		return ld
	})
	return l
}

// masked is State.masked for one validation round.
func (r ValidationRound) masked() ValidationRound {
	r.HeadSHA, r.Fix = line(r.HeadSHA), line(r.Fix)
	r.Validators = maskEach(r.Validators, func(v ValidatorRun) ValidatorRun {
		v.Role, v.Brief, v.Return, v.Verdict = line(v.Role), line(v.Brief), line(v.Return), line(v.Verdict)
		v.Audit = maskPtr(v.Audit, func(a AuditRun) AuditRun {
			a.ReceiptID, a.Request, a.BaseSHA, a.HeadSHA, a.Worst = line(a.ReceiptID), line(a.Request), line(a.BaseSHA), line(a.HeadSHA), line(a.Worst)
			a.NotMet, a.Inconclusive = maskEach(a.NotMet, line), maskEach(a.Inconclusive, line)
			return a
		})
		v.Route = maskPtr(v.Route, maskRoute)
		return v
	})
	return r
}

// masked is State.masked for one hand-back.
func (h HandBack) masked() HandBack {
	h.Kind, h.Reason, h.Home, h.Discarded = line(h.Kind), line(h.Reason), line(h.Home), line(h.Discarded)
	h.Verdict, h.Verdicts = line(h.Verdict), line(h.Verdicts)
	h.Findings, h.NotMet, h.Undecided = maskEach(h.Findings, line), maskEach(h.NotMet, line), maskEach(h.Undecided, line)
	return h
}

// masked is State.masked for the pace a run started on.
func (p Pace) masked() Pace {
	for _, v := range []*PaceValue{&p.WorkMinutes, &p.PauseMinutes, &p.SubAgents, &p.FixRounds} {
		v.Layer, v.Origin = line(v.Layer), line(v.Origin)
	}
	return p
}

// masked is State.masked for the pick a run was started from. Its entry is
// masked and never rewritten otherwise: the pick commit carries it, and the
// check that the lane's record holds it compares the two.
func (p RunPick) masked() RunPick {
	p.Lane, p.Entry = line(p.Lane), line(p.Entry)
	p.Pick.Chosen = maskCandidate(p.Pick.Chosen)
	p.Pick.RunnerUp = maskPtr(p.Pick.RunnerUp, maskCandidate)
	p.Pick.Candidates = maskEach(p.Pick.Candidates, maskCandidate)
	p.Pick.Rule, p.Pick.Falsifier = line(p.Pick.Rule), line(p.Pick.Falsifier)
	p.Excluded = maskEach(p.Excluded, func(e Excluded) Excluded {
		e.ID, e.Check, e.Reason = line(e.ID), line(e.Check), free(e.Reason)
		return e
	})
	return p
}

// maskCandidate masks a pick candidate's id and the details of its score.
func maskCandidate(c intent.PickCandidate) intent.PickCandidate {
	c.ID = line(c.ID)
	for _, part := range []*intent.ReadinessPart{&c.Score.Criteria, &c.Score.TestPath, &c.Score.Footprint} {
		part.Detail = line(part.Detail)
	}
	return c
}

// masked is State.masked for one captured transcript. Its scan gap is the
// missing tools' explanations, one per line, which the record renders line by
// line, so the mask keeps its line breaks (termsafe.SanitizeBlock).
func (t Transcript) masked() Transcript {
	t.Path, t.Session, t.Stored = free(t.Path), line(t.Session), free(t.Stored)
	t.ScanGap = termsafe.SanitizeBlock(redactHome(t.ScanGap))
	return t
}

// maskRoute masks the route a runner reported, its model above all.
func maskRoute(r runner.RouteRecord) runner.RouteRecord {
	r.Asked, r.Ran, r.Model = line(r.Asked), line(r.Ran), line(r.Model)
	return r
}

// line masks one line of stored text whole: a newline in it is masked too.
func line(s string) string { return termsafe.Sanitize(s) }

// free masks one line of free text the loop composes for a reader (a record
// note, a kept or exclusion reason, a fallback's detail, a transcript's
// home-redacted paths), redacting the home directory first. Every surface
// redacts the home again when it shows the text, but after the mask: a home
// whose name carries a rune the mask replaces no longer matches there, so the
// redaction that holds is this one, made before the mask runs. It is applied
// only to text nothing compares with its source: a step title (compared with
// the spec's), the pick's entry (with the pick commit's), the receipts and a
// lane's worktree (a path the loop acts on) are masked by line alone.
func free(s string) string { return termsafe.Sanitize(redactHome(s)) }

// redactHome is fsutil.RedactHome, skipped for empty text.
func redactHome(s string) string {
	if s == "" {
		return s
	}
	return fsutil.RedactHome(s)
}

// prose masks prose a lane hands in, keeping its line breaks (the CR of a
// CRLF pair included) and its tabs, the shape proseWritten admits.
func prose(s string) string {
	parts := strings.Split(s, "\t")
	for i, p := range parts {
		parts[i] = termsafe.SanitizeBlock(p)
	}
	return strings.Join(parts, "\t")
}

// maskEach returns a copy of s with f applied to each element; nil stays nil
// and empty stays empty, so the stored shape is unchanged.
func maskEach[T any](s []T, f func(T) T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// maskPtr returns a pointer to f applied to a copy of *p; nil stays nil.
func maskPtr[T any](p *T, f func(T) T) *T {
	if p == nil {
		return nil
	}
	v := f(*p)
	return &v
}
