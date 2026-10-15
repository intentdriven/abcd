package loop

// next.go is `abcd build next` (itd-2609211116005482, spc-2609212015048113):
// the run picks the intent it builds. The candidates are every planned intent
// that passes every pre-start check `abcd build <itd-N>` runs, judged by the
// same function (Check's rows, scope 1), less one this checkout already has a
// run for; each is scored from its record (intent.ReadinessIn) and the pick
// order (intent.PickLess) takes the readiest, the oldest among equals. The
// pick then starts the run `abcd build <itd-N>` would start for that intent,
// with the pick and its reason in the state, and the lane's worktree stage
// commits the reason onto the intent as the lane branch's first commit
// (lane.go, pickCommit), a record-only commit the receipt verifier does not
// count as the implementer's (receipt.go).
//
// One pick by default; `--max <n>` and `--until-empty` begin a series that
// picks again once each pick is delivered or its lane stops, under the pace
// rule (criterion 5, nextseries.go).

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// StagePick is the refusal stage of a pick that takes nothing.
const StagePick = "pick"

// CheckRun is the exclusion of a planned intent this checkout already has a
// run in progress for: the pick starts a run, never resumes one.
const CheckRun = "run"

// CheckFootprint is the exclusion of a planned intent whose spec names no
// footprint packages to reserve it with, in a checkout with a remote named
// origin: its start would be refused at the reservation.
const CheckFootprint = "footprint"

// Excluded is a planned intent the pick did not consider, with the first
// check that excluded it.
type Excluded struct {
	ID     string `json:"id"`
	Check  string `json:"check"`
	Reason string `json:"reason"`
}

// RunPick is the pick a run was started from, as the state carries it.
type RunPick struct {
	// Lane is the lane whose branch carries the pick's entry as its first
	// commit: the run's first lane.
	Lane string `json:"lane"`
	// Entry is the grounds entry's text, written after the `pursued:` token.
	Entry string `json:"entry"`
	// Pick is the choice: the candidates in the pick order with their scores,
	// the rule, the runner-up.
	Pick intent.Pick `json:"pick"`
	// Excluded are the planned intents the checks excluded.
	Excluded []Excluded `json:"excluded"`
}

// CandidateSet is the pick's view of the planned intents.
type CandidateSet struct {
	// Candidates pass every check, in the pick order.
	Candidates []intent.PickCandidate `json:"candidates"`
	// Excluded fail one, in record order.
	Excluded []Excluded `json:"excluded"`
}

// NextOptions are the run count the caller asked for.
type NextOptions struct {
	// Max is --max as typed; 0 when not given. An explicit --max 0 is told
	// apart from an absent flag only at the front door, which refuses it with
	// RunCountRefusal.
	Max int
	// UntilEmpty is --until-empty.
	UntilEmpty bool
}

// NextResult is what Next returns: the candidate set, the pick, the entry it
// writes, and the run it started; Pick and Start are nil when a series' move
// picked nothing (its last pick is in progress, its window has elapsed, or it
// ended). Series is the series the call moved, nil for the default single
// pick; Next is the move to make.
type NextResult struct {
	CandidateSet
	Pick   *intent.Pick  `json:"pick"`
	Entry  string        `json:"entry"`
	Start  *StartResult  `json:"start"`
	Series *SeriesResult `json:"series"`
	Next   string        `json:"next"`
}

// candidates judges every planned intent with the pre-start checks and scores
// the ones that pass. It writes nothing. A fault in reading the checkout is an
// error; an intent that may not start is an exclusion.
func candidates(repoRoot, session string) (CandidateSet, error) {
	return candidatesAt(repoRoot, session, time.Now())
}

// candidatesAt is candidates, judging the reservations as last fetched at now.
func candidatesAt(repoRoot, session string, now time.Time) (CandidateSet, error) {
	set := CandidateSet{Candidates: []intent.PickCandidate{}, Excluded: []Excluded{}}
	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return set, err
	}
	var planned []string
	for _, it := range corpus.Intents {
		if it.Bucket == intent.BucketPlanned {
			planned = append(planned, it.ID)
		}
	}
	slices.SortFunc(planned, func(a, b string) int {
		switch {
		case intent.IDOlder(a, b):
			return -1
		case intent.IDOlder(b, a):
			return 1
		}
		return 0
	})
	live := map[string]State{}
	if fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(RunRelDir))) {
		runs, err := Runs(repoRoot)
		if err != nil {
			return set, err
		}
		for _, st := range runs {
			if !st.Complete() {
				live[st.Intent] = st
			}
		}
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return set, err
	}
	snap, err := readPeers(repoRoot, now)
	if err != nil {
		return set, err
	}
	// With a remote named origin the start reserves the intent with its
	// spec's footprint packages, so one that names none is no candidate.
	reserves, err := claim.HasOrigin(repoRoot)
	if err != nil {
		return set, err
	}
	for _, id := range planned {
		if st, ok := live[id]; ok {
			reason := "run " + st.RunID + " in this checkout builds it; resume it with `abcd implement step`"
			if st.handedBack() {
				reason = "run " + st.RunID + " in this checkout handed it back, and the loop starts nothing further for it; " +
					handedBackWayOut(st)
			}
			set.Excluded = append(set.Excluded, Excluded{ID: id, Check: CheckRun, Reason: reason})
			continue
		}
		chk, err := check(repoRoot, id, session, snap)
		if err != nil {
			return set, err
		}
		if !chk.OK {
			for _, row := range chk.Checks {
				if !row.OK {
					set.Excluded = append(set.Excluded, Excluded{ID: id, Check: row.Name, Reason: termsafe.Sanitize(row.Detail)})
					break
				}
			}
			continue
		}
		if reserves {
			fp, err := specPackages(repoRoot, chk.Spec)
			if err != nil {
				return set, err
			}
			if len(fp) == 0 {
				set.Excluded = append(set.Excluded, Excluded{ID: id, Check: CheckFootprint, Reason: noFootprint(chk.Spec, id)})
				continue
			}
		}
		score, err := readiness(repoRoot, corpus, store, id, chk.Spec)
		if err != nil {
			return set, err
		}
		set.Candidates = append(set.Candidates, intent.PickCandidate{ID: id, Score: score})
	}
	intent.PickOrder(set.Candidates)
	return set, nil
}

// readiness scores one candidate from its record and its open spec's, through
// the one scoring read the status block's head uses too.
func readiness(repoRoot string, corpus intent.Corpus, store spec.Store, id, specID string) (intent.ReadinessScore, error) {
	it, ok := corpus.Lookup(id)
	if !ok {
		return intent.ReadinessScore{}, fmt.Errorf("%s left the intent store while the pick read it", id)
	}
	return intent.ReadinessIn(repoRoot, store, it, specID)
}

// RunCountRefusal is the pick's refusal of a --max below 1, which names no
// run count. Next refuses a negative Max with it; Max reads 0 when --max is
// not given, so a front door refuses an explicit --max 0 with it before
// calling Next, rather than let it read as no count (iss-2610121210280182).
func RunCountRefusal(max int) error {
	return refuse(StagePick, "", "", fmt.Sprintf("--max %d is not a run count", max), "give --max a whole number of picks, 1 or more")
}

// Next picks the readiest planned intent and starts its run. An empty
// candidate set is refused naming every excluded intent and its check, and
// writes nothing. With --max above 1 or --until-empty it begins a series
// (nextseries.go); while a series is in progress every call continues it, and
// a count other than the series' own is refused. The pick's entry is checked
// against the grounds writer's own gate before the run starts, so a pick never
// starts a run whose reason it cannot write.
func Next(repoRoot string, o Options, n NextOptions) (NextResult, error) {
	switch {
	case n.Max < 0:
		return NextResult{}, RunCountRefusal(n.Max)
	case n.UntilEmpty && n.Max > 0:
		return NextResult{}, refuse(StagePick, "", "", fmt.Sprintf("--max %d and --until-empty name two run counts", n.Max),
			"give one: --max <n> stops after n picks, --until-empty when no candidate is left")
	}
	flags, err := parsePaceFlags(o.Pace, o.SubAgents, o.FixRounds)
	if err != nil {
		return NextResult{}, err
	}
	if err := tierPresent(repoRoot); err != nil {
		return NextResult{}, err
	}
	if _, live, err := readSeries(repoRoot); err != nil {
		return NextResult{}, err
	} else if live {
		return continueSeries(repoRoot, o, n, flags)
	}
	if n.many() {
		return beginSeries(repoRoot, o, n, flags)
	}
	set, p, ok, err := choose(repoRoot, o, nil)
	if err != nil {
		return NextResult{}, err
	}
	if !ok {
		return NextResult{}, emptyRefusal(set)
	}
	res, err := startPick(repoRoot, o, set, p)
	if err != nil {
		return NextResult{}, err
	}
	res.Next = res.Start.Next
	return res, nil
}

// choose reads the candidate set, less every intent the series has picked
// when ser is not nil, and takes the readiest; ok is false when none is left.
func choose(repoRoot string, o Options, ser *NextSeries) (CandidateSet, intent.Pick, bool, error) {
	set, err := candidatesAt(repoRoot, o.Session, o.now())
	if err != nil {
		return set, intent.Pick{}, false, err
	}
	if ser != nil {
		set = ser.exclude(set)
	}
	p, ok := intent.Choose(set.Candidates)
	return set, p, ok, nil
}

// emptyRefusal is the refusal of a pick with no candidate.
func emptyRefusal(set CandidateSet) error {
	ref := refuse(StagePick, "", "", noCandidateReason(set.Excluded),
		"settle what each exclusion names (`abcd build <itd-N>` shows every check of one), or plan an intent")
	ref.Excluded = set.Excluded
	return ref
}

// startPick starts the run `abcd build <itd-N>` would start for the pick's
// chosen intent, with the pick and its reason in the run's state.
func startPick(repoRoot string, o Options, set CandidateSet, p intent.Pick) (NextResult, error) {
	// The entry's gate, asked of the text as it will be written but for the
	// run id, which is minted when the run is created and has one shape.
	if _, err := grounds.New(grounds.Pursued, intent.PickEntryText("run-0000000000000000", o.now(), p)); err != nil {
		return NextResult{}, refuse(StagePick, "", "", "the pick's reason cannot be written as a grounds entry: "+err.Error(),
			"report this: the reason is computed, and a computed reason the writer refuses is a defect")
	}
	started, err := start(repoRoot, p.Chosen.ID, o, &RunPick{Pick: p, Excluded: set.Excluded})
	if err != nil {
		return NextResult{}, err
	}
	res := NextResult{CandidateSet: set, Pick: &p, Start: &started}
	if st, err := ReadState(repoRoot, started.RunID); err == nil && st.Pick != nil {
		res.Entry = st.Pick.Entry
	}
	return res, nil
}

// noCandidateReason names every excluded intent and the check that excluded
// it.
func noCandidateReason(ex []Excluded) string {
	if len(ex) == 0 {
		return "no planned intent to pick: planned/ is empty"
	}
	parts := make([]string, 0, len(ex))
	for _, e := range ex {
		parts = append(parts, e.ID+" ("+e.Check+": "+e.Reason+")")
	}
	return fmt.Sprintf("no planned intent passes every check `abcd build <itd-N>` runs; %d excluded: %s", len(ex), strings.Join(parts, "; "))
}
