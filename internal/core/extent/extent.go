// Package extent holds the statement of how far one run of a command looked
// (itd-2611211024360819): what it looked at, what it did not look at and why,
// what it could not check this time, and, where a model did the judging, how
// hard that judging looked. Every declared command attaches it to its result
// under one member, `extent`; the name is not `coverage`, which several results
// and the lifeboat probe already use for something else.
//
// The package imports no other core, so every core can import it. The core
// returns the statement and writes nothing; Lines, in render.go, is the one
// renderer every surface prints.
//
// Every identifier a recorded statement carries comes from a closed set, and
// encoding refuses one outside it. The reasons and causes are held by
// testdata/identifiers.golden: a release may add one and never rename or
// remove one (criterion 24).
package extent

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Reason says why a check could not be made in this run.
type Reason string

// The five reasons, in the order the renderer counts them.
const (
	// ReasonUnreadable: a store or file could not be read.
	ReasonUnreadable Reason = "unreadable"
	// ReasonUnreachable: a remote or forge did not answer.
	ReasonUnreachable Reason = "unreachable"
	// ReasonToolMissing: a program the check needs is absent.
	ReasonToolMissing Reason = "tool_missing"
	// ReasonTimedOut: the check ran out of time.
	ReasonTimedOut Reason = "timed_out"
	// ReasonCheckFailed: the check ran and returned an error of none of the
	// other kinds.
	ReasonCheckFailed Reason = "check_failed"
)

var reasons = []Reason{ReasonUnreadable, ReasonUnreachable, ReasonToolMissing, ReasonTimedOut, ReasonCheckFailed}

// Cause says why a registry kind was left out in this run.
type Cause string

// The four causes, in their order of precedence: where more than one applies,
// an entry carries the first (Statement.LeaveOut picks it).
const (
	// CauseHostRun: the host's agents run it, not this command.
	CauseHostRun Cause = "host_run"
	// CauseOtherRun: it is left to another way of running the command.
	CauseOtherRun Cause = "other_run"
	// CauseNotArmed: the configuration armed no such check.
	CauseNotArmed Cause = "not_armed"
	// CauseNothingToRead: the check was armed and the run had nothing to read
	// it against.
	CauseNothingToRead Cause = "nothing_to_read"
)

var causes = []Cause{CauseHostRun, CauseOtherRun, CauseNotArmed, CauseNothingToRead}

// Source says where a not-looked-at entry comes from.
type Source string

const (
	// SourceLeftOut: a registry kind this run did not look at, taken from the
	// code, with a Cause.
	SourceLeftOut Source = "left_out"
	// SourceExclusion: what the command's code leaves out by a rule it reads
	// itself. It stands in every run.
	SourceExclusion Source = "exclusion"
	// SourceListed: a kind on the command's hand-written list. It stands in
	// every run.
	SourceListed Source = "listed"
)

var sources = []Source{SourceLeftOut, SourceExclusion, SourceListed}

// Verdict is the statement's own verdict. It is derived from the entries,
// never set, so it cannot be made to read clean.
type Verdict string

const (
	// VerdictClear: something was looked at, nothing was found and every
	// check was made.
	VerdictClear Verdict = "clear"
	// VerdictClearExceptGaps: something was looked at and nothing was found,
	// and at least one check could not be made.
	VerdictClearExceptGaps Verdict = "clear_except_gaps"
	// VerdictFound: something was found.
	VerdictFound Verdict = "found"
	// VerdictNone: no registry kind was looked at, whatever the reason, so the
	// statement gives no verdict.
	VerdictNone Verdict = "none"
)

var verdicts = []Verdict{VerdictClear, VerdictClearExceptGaps, VerdictFound, VerdictNone}

// Origin says where a depth value came from.
type Origin string

const (
	// OriginRunState: the run's own state recorded it as it handed out the
	// judging run.
	OriginRunState Origin = "run_state"
	// OriginReceipt: a receipt on disk carries it, the host's own account,
	// relayed and not vouched for.
	OriginReceipt Origin = "receipt"
)

var origins = []Origin{OriginRunState, OriginReceipt}

// Mark stands in place of a depth value abcd does not have.
type Mark string

const (
	// MarkNotTold: the host did the judging and no receipt says.
	MarkNotTold Mark = "not_told"
	// MarkNoneAsked: abcd started the run and asked for none.
	MarkNoneAsked Mark = "none_asked"
)

var marks = []Mark{MarkNotTold, MarkNoneAsked}

func checkIn[T ~string](set []T, v T, what string) error {
	if !slices.Contains(set, v) {
		return fmt.Errorf("extent: %s %q is not one of %v", what, string(v), set)
	}
	return nil
}

func decodeIn[T ~string](set []T, text []byte, what string, into *T) error {
	v := T(text)
	if err := checkIn(set, v, what); err != nil {
		return err
	}
	*into = v
	return nil
}

// MarshalText refuses a reason outside the closed set.
func (r Reason) MarshalText() ([]byte, error) { return []byte(r), checkIn(reasons, r, "reason") }

// UnmarshalText refuses a reason outside the closed set.
func (r *Reason) UnmarshalText(b []byte) error { return decodeIn(reasons, b, "reason", r) }

// MarshalText refuses a cause outside the closed set.
func (c Cause) MarshalText() ([]byte, error) { return []byte(c), checkIn(causes, c, "cause") }

// UnmarshalText refuses a cause outside the closed set.
func (c *Cause) UnmarshalText(b []byte) error { return decodeIn(causes, b, "cause", c) }

// MarshalText refuses a source outside the closed set.
func (s Source) MarshalText() ([]byte, error) { return []byte(s), checkIn(sources, s, "source") }

// UnmarshalText refuses a source outside the closed set.
func (s *Source) UnmarshalText(b []byte) error { return decodeIn(sources, b, "source", s) }

// MarshalText refuses a verdict outside the closed set.
func (v Verdict) MarshalText() ([]byte, error) { return []byte(v), checkIn(verdicts, v, "verdict") }

// UnmarshalText refuses a verdict outside the closed set.
func (v *Verdict) UnmarshalText(b []byte) error { return decodeIn(verdicts, b, "verdict", v) }

// MarshalText refuses an origin outside the closed set.
func (o Origin) MarshalText() ([]byte, error) { return []byte(o), checkIn(origins, o, "origin") }

// UnmarshalText refuses an origin outside the closed set.
func (o *Origin) UnmarshalText(b []byte) error { return decodeIn(origins, b, "origin", o) }

// MarshalText refuses a mark outside the closed set.
func (m Mark) MarshalText() ([]byte, error) { return []byte(m), checkIn(marks, m, "mark") }

// UnmarshalText refuses a mark outside the closed set.
func (m *Mark) UnmarshalText(b []byte) error { return decodeIn(marks, b, "mark", m) }

// Looked is one registry kind the run looked at, with how many it found by
// what the command counts as a finding. A kind in which nothing was found is
// an entry with a count of zero.
type Looked struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// NotLooked is one kind the run did not look at. Only a SourceLeftOut entry
// carries a Cause.
type NotLooked struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Source Source `json:"source"`
	Cause  Cause  `json:"cause,omitempty"`
}

// Gap is one check that could not be made in this run. Kind names the kind it
// bears on, where there is one: a kind in LookedAt when the run read it all
// the same, or a kind in neither list when the failed check stopped it being
// read at all.
type Gap struct {
	Check  string `json:"check"`
	Kind   string `json:"kind,omitempty"`
	Reason Reason `json:"reason"`
	Text   string `json:"text"`
}

// Told is a depth value that is a list (the classes the judging runs ran at,
// the efforts each was asked for): either the values with where they came
// from, or a Mark in their place.
type Told struct {
	Values []string `json:"values,omitempty"`
	From   Origin   `json:"from,omitempty"`
	Mark   Mark     `json:"mark,omitempty"`
}

// Runs is the number of judging runs: either the count with where it came
// from, or a Mark in its place.
type Runs struct {
	Count int
	From  Origin
	Mark  Mark
}

type runsWire struct {
	Count *int   `json:"count,omitempty"`
	From  Origin `json:"from,omitempty"`
	Mark  Mark   `json:"mark,omitempty"`
}

// MarshalJSON writes the count only where no Mark stands in its place, so a
// count of zero is recorded and a count abcd was not told is not.
func (r Runs) MarshalJSON() ([]byte, error) {
	w := runsWire{From: r.From, Mark: r.Mark}
	if r.Mark == "" {
		w.Count = &r.Count
	}
	return json.Marshal(w)
}

// UnmarshalJSON reads what MarshalJSON writes.
func (r *Runs) UnmarshalJSON(b []byte) error {
	var w runsWire
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	*r = Runs{From: w.From, Mark: w.Mark}
	if w.Count != nil {
		r.Count = *w.Count
	}
	return nil
}

// Depth is how hard one judging role looked, keyed by the lane whose change
// it judged or by the issue whose remedy it judged, or by its role alone. An
// entry whose role ran no judging run gives a count of zero and neither a
// class nor an effort.
type Depth struct {
	Role   string `json:"role"`
	Lane   string `json:"lane,omitempty"`
	Issue  string `json:"issue,omitempty"`
	Class  *Told  `json:"class,omitempty"`
	Runs   Runs   `json:"runs"`
	Effort *Told  `json:"effort,omitempty"`
}

// Statement is how far one run looked. The three lists are recorded on every
// run, empty or not. Depth is recorded only where a model did the judging.
type Statement struct {
	LookedAt      []Looked
	NotLookedAt   []NotLooked
	CouldNotCheck []Gap
	Depth         []Depth
	// NoClearState marks a command that has no clear state (the drive of a
	// drain): its statement carries no verdict member at all.
	NoClearState bool
}

// LeaveOut records a registry kind this run did not look at. Where more than
// one cause applies it records the first of host_run, other_run, not_armed and
// nothing_to_read; given no cause, the entry is left without one and the
// statement fails to encode.
func (s *Statement) LeaveOut(id, label string, applies ...Cause) {
	var cause Cause
	for _, c := range causes {
		if slices.Contains(applies, c) {
			cause = c
			break
		}
	}
	if cause == "" && len(applies) > 0 {
		cause = applies[0] // outside the set: kept so encoding names it
	}
	s.NotLookedAt = append(s.NotLookedAt, NotLooked{ID: id, Label: label, Source: SourceLeftOut, Cause: cause})
}

// Verdict derives the statement's verdict, in this order: found whenever
// something was found; otherwise none when no registry kind was looked at;
// otherwise clear_except_gaps when a check could not be made; otherwise clear.
// It reports false for a command with no clear state.
func (s Statement) Verdict() (Verdict, bool) {
	if s.NoClearState {
		return "", false
	}
	for _, l := range s.LookedAt {
		if l.Count > 0 {
			return VerdictFound, true
		}
	}
	switch {
	case len(s.LookedAt) == 0:
		return VerdictNone, true
	case len(s.CouldNotCheck) > 0:
		return VerdictClearExceptGaps, true
	}
	return VerdictClear, true
}

// Validate reports the first way s cannot be recorded: an identifier outside
// its set, a kind both looked at and not looked at (criterion 12) or named
// twice, a cause on an entry not left out in this run or none on one that is,
// an entry missing its identifier, label, check or text, or a depth entry
// that gives a value and a mark at once, or neither.
func (s Statement) Validate() error {
	seen := map[string]string{}
	claim := func(id, where string) error {
		if id == "" {
			return fmt.Errorf("extent: an entry of %s has no identifier", where)
		}
		if prev, ok := seen[id]; ok {
			if prev != where {
				return fmt.Errorf("extent: kind %q is both %s and %s", id, prev, where)
			}
			return fmt.Errorf("extent: kind %q is named twice in %s", id, where)
		}
		seen[id] = where
		return nil
	}
	for _, l := range s.LookedAt {
		if err := claim(l.ID, "looked at"); err != nil {
			return err
		}
		if l.Label == "" {
			return fmt.Errorf("extent: looked-at kind %q has no label", l.ID)
		}
		if l.Count < 0 {
			return fmt.Errorf("extent: looked-at kind %q has a negative count", l.ID)
		}
	}
	for _, n := range s.NotLookedAt {
		if err := claim(n.ID, "not looked at"); err != nil {
			return err
		}
		if n.Label == "" {
			return fmt.Errorf("extent: not-looked-at kind %q has no label", n.ID)
		}
		if err := checkIn(sources, n.Source, "source"); err != nil {
			return err
		}
		switch {
		case n.Source == SourceLeftOut && n.Cause == "":
			return fmt.Errorf("extent: kind %q is left out in this run with no cause", n.ID)
		case n.Source != SourceLeftOut && n.Cause != "":
			return fmt.Errorf("extent: kind %q is a %s entry and carries a cause", n.ID, n.Source)
		}
		if n.Cause != "" {
			if err := checkIn(causes, n.Cause, "cause"); err != nil {
				return err
			}
		}
	}
	for _, g := range s.CouldNotCheck {
		if g.Check == "" || g.Text == "" {
			return fmt.Errorf("extent: a check that could not be made needs its name and a line of text (check %q)", g.Check)
		}
		if err := checkIn(reasons, g.Reason, "reason"); err != nil {
			return err
		}
	}
	for _, d := range s.Depth {
		if err := d.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d Depth) validate() error {
	if d.Role == "" {
		return fmt.Errorf("extent: a depth entry has no role")
	}
	if d.Lane != "" && d.Issue != "" {
		return fmt.Errorf("extent: depth entry %q is keyed by a lane and an issue at once", d.Role)
	}
	if err := oneOf(d.Runs.From, d.Runs.Mark, d.Runs.Mark == "" || d.Runs.Count == 0); err != nil {
		return fmt.Errorf("extent: depth entry %q, runs: %w", d.Role, err)
	}
	if d.Runs.Count < 0 {
		return fmt.Errorf("extent: depth entry %q has a negative number of runs", d.Role)
	}
	noRun := d.Runs.Mark == "" && d.Runs.Count == 0
	for _, part := range []struct {
		name string
		told *Told
	}{{"class", d.Class}, {"effort", d.Effort}} {
		if part.told == nil {
			continue
		}
		if noRun {
			return fmt.Errorf("extent: depth entry %q ran no judging run and gives a %s", d.Role, part.name)
		}
		t := part.told
		if err := oneOf(t.From, t.Mark, (t.Mark == "") == (len(t.Values) > 0)); err != nil {
			return fmt.Errorf("extent: depth entry %q, %s: %w", d.Role, part.name, err)
		}
	}
	return nil
}

// oneOf holds a depth value to exactly one of a value with where it came from
// and a mark in its place; shapeOK says the value side agrees with that.
func oneOf(from Origin, mark Mark, shapeOK bool) error {
	switch {
	case from != "" && mark != "":
		return fmt.Errorf("gives a value and the mark %q at once", mark)
	case from == "" && mark == "":
		return fmt.Errorf("gives neither where its value came from nor a mark")
	case !shapeOK:
		return fmt.Errorf("a value is given beside a mark, or none beside where it came from")
	case from != "":
		return checkIn(origins, from, "origin")
	}
	return checkIn(marks, mark, "mark")
}

type statementWire struct {
	LookedAt      []Looked    `json:"looked_at"`
	NotLookedAt   []NotLooked `json:"not_looked_at"`
	CouldNotCheck []Gap       `json:"could_not_check"`
	Verdict       *Verdict    `json:"verdict,omitempty"`
	Depth         []Depth     `json:"depth,omitempty"`
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// MarshalJSON validates s and writes it with its derived verdict. A statement
// Validate refuses fails to encode.
func (s Statement) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	w := statementWire{
		LookedAt:      orEmpty(s.LookedAt),
		NotLookedAt:   orEmpty(s.NotLookedAt),
		CouldNotCheck: orEmpty(s.CouldNotCheck),
		Depth:         s.Depth,
	}
	if v, ok := s.Verdict(); ok {
		w.Verdict = &v
	}
	return json.Marshal(w)
}

// UnmarshalJSON reads what MarshalJSON writes. An absent verdict reads as no
// clear state, and a verdict the entries do not derive is refused.
func (s *Statement) UnmarshalJSON(b []byte) error {
	var w statementWire
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	got := Statement{
		LookedAt:      w.LookedAt,
		NotLookedAt:   w.NotLookedAt,
		CouldNotCheck: w.CouldNotCheck,
		Depth:         w.Depth,
		NoClearState:  w.Verdict == nil,
	}
	if err := got.Validate(); err != nil {
		return err
	}
	if w.Verdict != nil {
		if want, _ := got.Verdict(); want != *w.Verdict {
			return fmt.Errorf("extent: recorded verdict %q, but the entries give %q", *w.Verdict, want)
		}
	}
	*s = got
	return nil
}

// identifiers lists the reasons and causes as "<set> <identifier>", the lines
// testdata/identifiers.golden holds.
func identifiers() []string {
	var out []string
	for _, r := range reasons {
		out = append(out, "reason "+string(r))
	}
	for _, c := range causes {
		out = append(out, "cause "+string(c))
	}
	return out
}

// missingIdentifiers returns the golden identifiers current no longer holds:
// each is a rename or a removal. An identifier current adds is not reported.
func missingIdentifiers(golden, current []string) []string {
	var out []string
	for _, g := range golden {
		if !slices.Contains(current, g) {
			out = append(out, g)
		}
	}
	return out
}
