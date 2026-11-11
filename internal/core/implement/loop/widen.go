package loop

// widen.go is the widening of an issue lane's hold to the files the lane
// changes beyond it (iss-2610311224444576, iss-2611110638266241,
// iss-2611101316537446; dcn-2610311251002713, dcn-2610311544325397). A build
// and a drain reserve an issue with the path its found_at names, and a fix may
// need to change a further file. The loop widens the hold itself, from the
// process that drives the run, outside any wall a drive puts round a stage:
// once an implementer hands its receipt back, and before the receipt is
// judged, it reserves the issue again with the footprint its reservation holds
// and every path the lane's branch changes that the footprint does not cover
// (widenLaneChanges), judged for overlap and against the cap like any
// reservation. The same widening is made again before each step at the lane's
// validators (holdBeforeValidators) and at the fence before a landing's push
// and its arming (holdLaneChanges), so a path the lane's branch came to change
// since is reserved before anything judges or lands it. The run's own records
// the lane changes, a decision record, the issue's record its landing moves,
// are reserved like any other path.
//
// Another copy's live hold on one of the paths is contention, and a wait by
// design: nothing is pushed, the receipt is taken all the same, and the lane
// waits before its validators until that hold ends (Refusal.pathHeld), never
// counted as a refused step and never handed back as stuck. A receipt is
// refused on the hold only when the widening itself could not be made for a
// reason the implementer can act on, a path no reservation may carry. `abcd
// implement widen` (Widen) is for a host session's implementer that wants a
// path held before its edit; it is the same widening (widenHold). The run
// record keeps each path widened to (StageWidenedTo), so a reservation made
// again names it whatever became of the reservation branch.

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageWidenedTo is the run record's stage for a path a lane's reservation was
// widened to; its note is the path and nothing else. A reservation made again
// names these beside the found_at (widenedPaths), so they outlive a deleted tip.
const StageWidenedTo = "reservation_widened_to"

// WidenResult is a widening's outcome: the run and lane, the paths newly held
// (none when the reservation already covered every one), the footprint the
// reservation holds after, and the run record's line.
type WidenResult struct {
	RunID     string   `json:"run_id"`
	Lane      string   `json:"lane"`
	Record    string   `json:"record"`
	Added     []string `json:"added"`
	Footprint []string `json:"footprint"`
	Note      string   `json:"note"`
}

// widenStages are the stages at which a lane's implementer edits: the
// implement stage, and the validate stage, whose fix rounds hand a fresh
// implementer the lane.
var widenStages = []Stage{StageImplement, StageValidate}

// Widen widens the reservation of issue run runID, for its current lane
// laneID, to cover paths before the lane's implementer edits them: a host
// session's step, since the loop widens the hold to what the lane's branch
// changes itself once the receipt is handed back (widenLaneChanges). A path
// the reservation already covers adds nothing, and when it covers every one
// nothing is pushed. A run that holds no reservation (no origin) has nothing
// to widen and says so. Refused for a person (exit 2): an intent's run, whose
// footprint is its spec's; a lane that is not the run's current one or not at
// an editing stage; a path no reservation may carry; a reservation the view
// does not show this copy holding. Contention (exit 3): another copy's live
// hold on one of the paths, with nothing pushed and a run record line saying
// the widening backed off; or a locked run state. A lane's own abcd may run
// it, so the save is the reservation's alone (mutateReservation,
// iss-2611020719494343).
func Widen(repoRoot, runID, laneID string, paths []string, o Options) (WidenResult, error) {
	res := WidenResult{RunID: runID, Lane: laneID, Added: []string{}}
	var backedOff error
	err := mutateReservation(repoRoot, runID, func(st *State) (bool, error) {
		now := o.now()
		res.Record = st.Key
		if st.Issue() == "" {
			return false, refuse(StageReservation, "", laneID, st.Key+"'s run holds the footprint its spec names, and only an issue lane widens its hold",
				"name the path on the `- packages:` line of "+specOf(*st)+"'s `## Footprint` section, then run `abcd build "+st.Key+"` again")
		}
		i := st.current()
		if i < 0 || st.Lanes[i].ID != laneID {
			named := laneID
			if !ValidLaneID(laneID) {
				named = termsafe.DescribeRefused(laneID)
			}
			return false, refuse(StageReservation, "", "", named+" is not the current lane of "+runID,
				"name the lane the brief names; `abcd implement status` lists the run's lanes")
		}
		lane := st.Lanes[i]
		if !slices.Contains(widenStages, lane.Stage) {
			return false, refuse(StageReservation, "", laneID, fmt.Sprintf("%s is at its %s stage, where no implementer edits, so its hold is not widened", laneID, lane.Stage),
				"widen only before an edit an implementer makes at the implement or validate stage")
		}
		want, err := widenPaths(laneID, paths)
		if err != nil {
			return false, err
		}
		if st.ReservedAt == nil {
			res.Note = runID + " holds no reservation, so there is nothing to widen"
			return false, nil
		}
		snap, err := claim.View(repoRoot, now)
		if err != nil {
			return false, refuse(StageReservation, "", laneID, "the reservation on "+st.Key+" cannot be read to widen it: "+fsutil.RedactHome(err.Error()),
				"settle what it names, then run `abcd implement widen` again")
		}
		own, ok := snap.Lookup(st.Key)
		if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
			return false, refuse(StageReservation, "", laneID, "this copy holds no reservation on "+st.Key+" to widen",
				"run `abcd implement step`, whose land fence settles a lost reservation, before editing further")
		}
		res.Footprint = slices.Clone(own.Footprint)
		w, err := widenHold(repoRoot, st.Key, own, want, now)
		if len(w.added) == 0 && err == nil {
			res.Note = "the reservation on " + st.Key + " already holds " + strings.Join(want, ", ")
			return false, nil
		}
		res.Added = w.added
		if err != nil {
			rf, ok := AsRefusal(err)
			if !ok || !rf.Contention {
				return false, err
			}
			note := fmt.Sprintf("the widening of %s's hold to %s backed off: %s; nothing was pushed, and the lane edits none of them",
				st.Key, strings.Join(res.Added, ", "), rf.Reason)
			st.Record = append(st.Record, Entry{At: now, Lane: laneID, Stage: StageReservation, Note: note})
			st.UpdatedAt = now
			backedOff = contend(StageReservation, CheckReservation, laneID, note,
				"edit none of them; stop without a receipt, and the lane is handed to an implementer again on a later step")
			return true, nil
		}
		res.Footprint = w.footprint
		res.Note = w.r.note + ", widened to " + strings.Join(res.Added, ", ") + " before the lane's implementer edits it"
		st.Record = append(st.Record, widenedLines(laneID, res.Added, res.Note, now)...)
		st.UpdatedAt = now
		return true, nil
	})
	if err != nil {
		return WidenResult{}, err
	}
	if backedOff != nil {
		return WidenResult{}, backedOff
	}
	return res, nil
}

// widened is what widenHold came to: each path newly held, the footprint the
// reservation names after, and the reservation act's outcome.
type widened struct {
	added     []string
	footprint []string
	r         reserved
}

// maxWidenPaths bounds the paths one widening compares: comfortably above
// any number a reservation message can name within its cap
// (claim.MaxMessageBytes), so a set of paths past it is refused as over the
// cap at once, before any comparison is made.
const maxWidenPaths = 512

// overCapRefusal is the refusal of a widening of key's hold whose footprint
// cannot fit a reservation message: n paths, or more than n when more is
// set. It is marked overCap, which no relaunch changes, so a receipt names it
// as a gap of its own (receiptHoldGaps).
func overCapRefusal(key string, n int, more bool) *Refusal {
	count := fmt.Sprintf("the %d", n)
	if more {
		count = fmt.Sprintf("more than %d", n)
	}
	r := refuse(StageReservation, CheckReservation, "", fmt.Sprintf(
		"the reservation on %s cannot carry that many paths: %s its hold would name pass what a reservation message holds within its %d-byte cap",
		key, count, claim.MaxMessageBytes),
		"take up fewer paths beyond the issue's place in one lane")
	r.overCap = true
	return r
}

// widenHold is the one widening of an issue run's hold, whoever asks for it:
// Widen, for a host session's implementer that names the paths before its
// edit, and the loop's own (widenLaneChanges), once a lane hands its receipt
// back, before each step at its validators and at the fence before its
// landing's push and its arming. It reserves key again with own's footprint
// and each path of want that footprint does not cover, each by its own name,
// keeping own's term (widenReservation), judged for overlap like any
// reservation. Nothing is pushed when own covers every path, and the result
// names none added. Paths past maxWidenPaths, or too many for the message's
// cap (claim.FootprintFits), are refused at once (overCapRefusal). Any other
// refusal is the reservation act's (reservedNote), with the paths it would
// have added.
func widenHold(repoRoot, key string, own claim.Reservation, want []string, now time.Time) (widened, error) {
	var w widened
	held := newPathSet(own.Footprint)
	for _, p := range want {
		if held.covers(p) {
			continue
		}
		if len(w.added) == maxWidenPaths {
			return widened{}, overCapRefusal(key, len(own.Footprint)+maxWidenPaths, true)
		}
		held.add(p)
		w.added = append(w.added, p)
	}
	if len(w.added) == 0 {
		w.footprint = slices.Clone(own.Footprint)
		return w, nil
	}
	fp := append(slices.Clone(own.Footprint), w.added...)
	if !claim.FootprintFits(fp) {
		return w, overCapRefusal(key, len(fp), false)
	}
	r, err := widenReservation(repoRoot, key, fp, false, false, own, now)
	if err != nil {
		return w, err
	}
	w.footprint, w.r = fp, r
	return w, nil
}

// pathSet is a set of clean paths that answers whether one of them covers a
// path (covers) in time linear in the path's depth, whatever its size.
type pathSet map[string]bool

// newPathSet is a pathSet holding each of paths, cleaned.
func newPathSet(paths []string) pathSet {
	s := make(pathSet, len(paths))
	for _, p := range paths {
		s.add(p)
	}
	return s
}

// add puts p, cleaned, in the set.
func (s pathSet) add(p string) { s[path.Clean(p)] = true }

// covers reports whether the set holds p or a directory above it, as covers
// (reservation.go) reads a footprint.
func (s pathSet) covers(p string) bool {
	for p = path.Clean(p); ; {
		if s[p] {
			return true
		}
		i := strings.LastIndexByte(p, '/')
		if i < 0 {
			return false
		}
		p = p[:i]
	}
}

// widenedLines are the run record's lines for a widening made: one
// StageWidenedTo line for each path added, then the reservation's line, note.
func widenedLines(laneID string, added []string, note string, now time.Time) []Entry {
	out := make([]Entry, 0, len(added)+1)
	for _, p := range added {
		out = append(out, Entry{At: now, Lane: laneID, Stage: StageWidenedTo, Note: p})
	}
	return append(out, Entry{At: now, Lane: laneID, Stage: StageReservation, Note: note})
}

// widenedPaths is each path the run record says the run's lanes widened to.
func widenedPaths(st State) []string {
	var out []string
	for _, e := range st.Record {
		if e.Stage == StageWidenedTo && !slices.Contains(out, e.Note) {
			out = append(out, e.Note)
		}
	}
	return out
}

// widenPaths is paths cleaned, deduplicated and sorted, refused when one is
// empty or a path no reservation may carry.
func widenPaths(laneID string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, refuse(StageReservation, "", laneID, "no path was named to widen the hold to",
			"name each repository-relative path the lane is about to edit")
	}
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !claim.ValidFootprintPath(p) {
			return nil, refuse(StageReservation, "", laneID, termsafe.DescribeRefused(p)+" is a path no reservation may carry",
				"name each path relative to the repository root, without `..`, a leading `/` or a control character")
		}
		if p = path.Clean(p); !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out, nil
}

// peerHold reports whether err, the refusal of a widening of key's hold to
// paths, is a back-off over another holder's live hold on one of the paths,
// read from the reservations as the act fetched them: the tip of key is still
// this copy's, so the act did not back off the record itself, and a path is
// shared with another's live reservation (claim.Snapshot.Contention), this
// person's other copy included. A back-off over the record itself, gone to
// another, is not: that is the land fence's to settle.
func peerHold(repoRoot, key string, paths []string, err error, now time.Time) bool {
	if rf, ok := AsRefusal(err); !ok || !rf.Contention {
		return false
	}
	snap, verr := claim.View(repoRoot, now)
	if verr != nil {
		return false
	}
	own, ok := snap.Lookup(key)
	if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return false
	}
	return snap.Contention(key, paths, now, claim.Self{}) != ""
}

// laneHold is what the loop's own widening of an issue lane's hold came to
// (widenLaneChanges).
type laneHold struct {
	// own is the reservation this copy held on the issue before the widening.
	own claim.Reservation
	// paths is each path the lane's branch changes that own did not cover.
	paths []string
	// note is the run record's clause for a widening made, and lines its
	// lines; both empty when every path was held.
	note  string
	lines []Entry
	// gap names what could not be read to judge the paths, in a receipt's
	// words.
	gap string
	// err is the widening's refusal, and peer marks it a back-off over
	// another holder's live hold on one of the paths (peerHold).
	err  error
	peer bool
}

// shown is the paths the lane's branch changes beyond its hold, each made
// safe for a terminal, joined for a line of text: the first ten, and how
// many more there are.
func (h laneHold) shown() string {
	const most = 10
	n := min(len(h.paths), most)
	out := make([]string, n, n+1)
	for i, p := range h.paths[:n] {
		out[i] = termsafe.Sanitize(p)
	}
	if rest := len(h.paths) - n; rest > 0 {
		out = append(out, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(out, ", ")
}

// widenLaneChanges is the loop's own widening of issue lane's hold to every
// path its branch changes, at its tip, that the reservation this copy holds
// does not cover (unheldPaths), made by the process driving the run, which
// holds the credential for origin a drive's stage is walled off from
// (iss-2611110638266241). Each path is checked as Widen checks a path named
// (widenPaths) and reserved through the one widening (widenHold). An intent's
// run, and a reservation the view does not show this copy holding, widen
// nothing; the caller decides what a run that recorded no reservation does.
func widenLaneChanges(c Context, lane Lane) laneHold {
	var h laneHold
	h.own, h.paths, h.gap = unheldPaths(c, lane, c.Now)
	if h.gap != "" || len(h.paths) == 0 {
		return h
	}
	if len(h.paths) > maxWidenPaths {
		h.err = overCapRefusal(c.State.Key, len(h.own.Footprint)+maxWidenPaths, true)
		return h
	}
	want, err := widenPaths(lane.ID, h.paths)
	if err != nil {
		h.err = err
		return h
	}
	w, err := widenHold(c.RepoRoot, c.State.Key, h.own, want, c.Now)
	if err != nil {
		h.err, h.peer = err, peerHold(c.RepoRoot, c.State.Key, w.added, err, c.Now)
		return h
	}
	if len(w.added) == 0 {
		return h
	}
	h.note = w.r.note + ", widened to " + strings.Join(w.added, ", ") + ", which the lane's branch changes beyond it"
	h.lines = widenedLines(lane.ID, w.added, h.note, c.Now)
	return h
}

// receiptHoldGaps is the receipt's check of the files lane's branch changes
// against the issue's reservation, made after the loop's own widening
// (widenLaneChanges), so a receipt is refused on the hold only when the
// widening could not be made. A gap names what could not be read, or a path
// the widening refused for a reason the implementer can act on (a path no
// reservation may carry). A widening that could not be made yet is no gap:
// another holder's live hold on a path, an origin it could not reach, or the
// cap; the receipt is taken, the record says so, and the widening is made
// again before the lane's validators (holdBeforeValidators), which wait for
// it. A run that holds no reservation, an intent's run, and a reservation the
// view does not show this copy holding (the land fence's to settle) give
// none. The record's lines go to c.widened.
func receiptHoldGaps(c Context, lane Lane) []string {
	st := c.State
	if st.ReservedAt == nil {
		return nil
	}
	h := widenLaneChanges(c, lane)
	record := func(lines ...Entry) {
		if c.widened != nil {
			*c.widened = append(*c.widened, lines...)
		}
	}
	if h.gap != "" {
		return []string{h.gap}
	}
	if h.err == nil {
		record(h.lines...)
		return nil
	}
	rf, ok := AsRefusal(h.err)
	reason := fsutil.RedactHome(h.err.Error())
	if ok {
		reason = rf.Reason
	}
	if ok && rf.overCap {
		return []string{fmt.Sprintf("a hold on %s, which the lane's branch changes beyond its reservation on %s (%s): %s",
			h.shown(), st.Key, strings.Join(h.own.Footprint, ", "), reason)}
	}
	if ok && (rf.Contention || rf.originUnread || len(rf.capped) > 0) {
		then := "the receipt is taken, and the widening is made again before the lane's validators, which wait for it"
		if h.peer {
			then = "the receipt is taken, and the lane's validators wait for that hold to end"
		}
		record(Entry{At: c.Now, Lane: lane.ID, Stage: StageReservation,
			Note: fmt.Sprintf("the loop's widening of %s's hold to %s, which the lane's branch changes beyond it, could not be made yet: %s; %s",
				st.Key, h.shown(), reason, then)})
		return nil
	}
	return []string{fmt.Sprintf("a hold on %s, which the lane's branch changes beyond its reservation on %s (%s) and the loop could not reserve (%s): take the change out of the branch",
		h.shown(), st.Key, strings.Join(h.own.Footprint, ", "), reason)}
}

// validatorsAct is what a refusal of the hold before a lane's validators
// (holdBeforeValidators) says the lane does not do.
var validatorsAct = fenceAct{not: "go on to its validators", none: "no validator is started", stays: "no validator is started"}

// holdBeforeValidators is the loop's own widening of an issue lane's hold
// before each step at its validators (holdLaneChanges): a path the lane's
// branch changes beyond its reservation, which a receipt's widening could not
// reserve yet or a fix round has changed since, is reserved before a validator
// judges it. Another holder's live hold on it is a wait by design until that
// hold ends. lines are the run record's lines for a widening made. A run that
// holds no reservation widens nothing here, and neither does a lane whose
// branch or reservation cannot be read here: the receipt already named what
// was missing, and the land fence refuses the push on it, so the validators
// are not held back on a reading the landing makes again.
func holdBeforeValidators(c Context, lane Lane) ([]Entry, error) {
	if c.State.ReservedAt == nil {
		return nil, nil
	}
	_, lines, err := holdLaneChanges(c, lane, StageValidate, validatorsAct)
	return lines, err
}

// unheldPaths is the reading every widening of the loop shares: the
// reservation this copy holds on the issue, and each path lane's branch
// changes at its tip, as git names it, that the reservation does not cover,
// the records the lane's landing moves among them. gap, set when one of them
// cannot be read, says what is missing in a receipt's words. An intent's run,
// and a reservation the view does not show this copy holding, give none.
func unheldPaths(c Context, lane Lane, now time.Time) (own claim.Reservation, unheld []string, gap string) {
	st := c.State
	if st.Issue() == "" {
		return own, nil, ""
	}
	snap, err := claim.View(c.RepoRoot, now)
	if err != nil {
		return own, nil, "a reservation that can be read to check the files the lane changed (" + fsutil.RedactHome(err.Error()) + ")"
	}
	own, ok := snap.Lookup(st.Key)
	if !ok || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return own, nil, ""
	}
	head, err := branchTip(c, lane)
	if err != nil {
		return own, nil, "a branch whose tip can be read to check the files the lane changed (" + fsutil.RedactHome(err.Error()) + ")"
	}
	changed, declined := laneChangedFiles(c.RepoRoot, lane, head)
	if declined != "" {
		return own, nil, "the files the lane changed, to check them against its reservation (" + declined + ")"
	}
	held := newPathSet(own.Footprint)
	for _, p := range changed {
		if !held.covers(p) {
			unheld = append(unheld, p)
		}
	}
	return own, unheld, ""
}

// holdLaneChanges is the loop's own widening of an issue lane's hold as a step
// makes it, at stage: before the lane's validators (holdBeforeValidators), and
// at the fence before its landing's push and its arming, where no receipt
// stands between a lane at its validators and its landing, so a path its
// reservation came to leave out would otherwise land unjudged. A path the
// reservation does not cover is reserved now (widenLaneChanges), keeping the
// tip's term. Another holder's live hold on it is a wait by design until that
// hold ends (Refusal.pathHeld); an origin it cannot reach is a wait, as at the
// fence's other reads (fenceUnread); any other refusal stops act, naming what
// it met. A branch or reservation it cannot read stops the landing alone. It reads the reservation this copy holds whether the run recorded
// one or a person made it by hand. note is the run record's clause for a
// reservation made, and lines its lines, both empty when every path was held.
func holdLaneChanges(c Context, lane Lane, stage Stage, act fenceAct) (note string, lines []Entry, err error) {
	by := "the land fence"
	if stage != StageLand {
		by = "the hold before the lane's validators"
	}
	h := widenLaneChanges(c, lane)
	if h.gap != "" && stage != StageLand {
		return "", nil, nil
	}
	if h.gap != "" {
		return "", nil, refuse(string(stage), "", lane.ID, by+" is missing "+h.gap+", so this lane does not "+act.not,
			"settle what it names, then run `abcd implement step` again; "+act.stays)
	}
	if h.err == nil {
		return h.note, h.lines, nil
	}
	reason := fsutil.RedactHome(h.err.Error())
	rf, ok := AsRefusal(h.err)
	if ok {
		reason = rf.Reason
	}
	what := fmt.Sprintf("the lane's branch changes %s beyond its reservation on %s, which could not be reserved: %s; so this lane does not %s",
		h.shown(), c.State.Key, reason, act.not)
	switch {
	case ok && rf.originUnread:
		r := fenceUnread(lane, what, reason, "git push the reservation on "+c.State.Key)
		r.Stage = string(stage)
		return "", nil, r
	case h.peer:
		r := contend(string(stage), CheckReservation, lane.ID, what,
			act.stays+"; the lane waits for the hold it names to end, and goes on once it has: settle it with its holder to go on sooner, or run `abcd implement step` again once it ends")
		r.pathHeld = true
		return "", nil, r
	case ok && rf.Contention:
		return "", nil, contend(string(stage), "", lane.ID, what,
			act.stays+"; wait for the reservation it names to end, or settle it with its holder, then run `abcd implement step` again")
	}
	out := refuse(string(stage), "", lane.ID, what, "settle what it names, then run `abcd implement step` again; "+act.stays)
	if ok {
		out.capped = rf.capped
	}
	return "", nil, out
}

// heldWith is fp with every path of held that fp does not cover appended.
func heldWith(fp, held []string) []string {
	out := slices.Clone(fp)
	set := newPathSet(fp)
	for _, p := range held {
		if !set.covers(p) {
			set.add(p)
			out = append(out, p)
		}
	}
	return out
}
