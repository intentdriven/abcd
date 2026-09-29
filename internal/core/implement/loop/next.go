package loop

// next.go is `abcd build next` (itd-2609211116005482, spc-2609212015048113):
// the run picks the intent it builds. The candidates are every planned intent
// that passes every pre-start check `abcd build <itd-N>` runs, judged by the
// same function (Check's rows, scope 1), less one this checkout already has a
// run for; each is scored from its record (intent.ReadinessIn) and the pick
// order (intent.PickLess) takes the readiest, the oldest among equals. The
// pick then starts the run `abcd build <itd-N>` would start for that intent,
// with the pick and its reason in the state, and the lane's worktree step
// commits the reason onto the intent as the lane branch's first commit
// (lane.go, pickCommit), a record-only commit the receipt verifier does not
// count as the implementer's (receipt.go).
//
// One pick per invocation: continuing under the pace rule (`--max`,
// `--until-empty`, criterion 5) is not built in this abcd, and asking for it
// is refused by name.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// StepPick is the refusal step of a pick that takes nothing.
const StepPick = "pick"

// CheckRun is the exclusion of a planned intent this checkout already has a
// run in progress for: the pick starts a run, never resumes one.
const CheckRun = "run"

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
	// Max is --max as typed; 0 when not given.
	Max int
	// UntilEmpty is --until-empty.
	UntilEmpty bool
}

// NextResult is what Next returns: the candidate set, the pick, the entry it
// writes, and the run it started.
type NextResult struct {
	CandidateSet
	Pick  intent.Pick `json:"pick"`
	Entry string      `json:"entry"`
	Start StartResult `json:"start"`
}

// candidates judges every planned intent with the pre-start checks and scores
// the ones that pass. It writes nothing. A fault in reading the checkout is an
// error; an intent that may not start is an exclusion.
func candidates(repoRoot, session string) (CandidateSet, error) {
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
	live := map[string]string{}
	if fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(RunRelDir))) {
		runs, err := Runs(repoRoot)
		if err != nil {
			return set, err
		}
		for _, st := range runs {
			if !st.Complete() {
				live[st.Intent] = st.RunID
			}
		}
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return set, err
	}
	snap, err := readPeers(repoRoot)
	if err != nil {
		return set, err
	}
	for _, id := range planned {
		if runID, ok := live[id]; ok {
			set.Excluded = append(set.Excluded, Excluded{ID: id, Check: CheckRun,
				Reason: "run " + runID + " in this checkout builds it; resume it with `abcd implement step`"})
			continue
		}
		chk, err := check(repoRoot, id, session, snap)
		if err != nil {
			return set, err
		}
		if !chk.OK {
			for _, row := range chk.Checks {
				if !row.OK {
					set.Excluded = append(set.Excluded, Excluded{ID: id, Check: row.Name, Reason: row.Detail})
					break
				}
			}
			continue
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

// Next picks the readiest planned intent and starts its run. A run count past
// one is refused (criterion 5 is not built here); an empty candidate set is
// refused naming every excluded intent and its check, and writes nothing.
// The pick's entry is checked against the grounds writer's own gate before
// the run starts, so a pick never starts a run whose reason it cannot write.
func Next(repoRoot string, o Options, n NextOptions) (NextResult, error) {
	if n.UntilEmpty || n.Max > 1 {
		asked := fmt.Sprintf("--max %d", n.Max)
		if n.UntilEmpty {
			asked = "--until-empty"
		}
		return NextResult{}, refuse(StepPick, "", "", asked+" asks for more than one pick, and picking again under the pace rule (criterion 5 of itd-2609211116005482) is not built in this abcd",
			"run `abcd build next` once per pick; each run picks one intent")
	}
	if n.Max < 0 {
		return NextResult{}, refuse(StepPick, "", "", fmt.Sprintf("--max %d is not a run count", n.Max), "give --max a whole number of picks, 1 or more")
	}
	if err := tierPresent(repoRoot); err != nil {
		return NextResult{}, err
	}
	set, err := candidates(repoRoot, o.Session)
	if err != nil {
		return NextResult{}, err
	}
	p, ok := intent.Choose(set.Candidates)
	if !ok {
		ref := refuse(StepPick, "", "", noCandidateReason(set.Excluded),
			"settle what each exclusion names (`abcd build <itd-N>` shows every check of one), or plan an intent")
		ref.Excluded = set.Excluded
		return NextResult{}, ref
	}
	// The entry's gate, asked of the text as it will be written but for the
	// run id, which is minted when the run is created and has one shape.
	if _, err := grounds.New(grounds.Pursued, intent.PickEntryText("run-0000000000000000", o.now(), p)); err != nil {
		return NextResult{}, refuse(StepPick, "", "", "the pick's reason cannot be written as a grounds entry: "+err.Error(),
			"report this: the reason is computed, and a computed reason the writer refuses is a defect")
	}
	started, err := start(repoRoot, p.Chosen.ID, o, &RunPick{Pick: p, Excluded: set.Excluded})
	if err != nil {
		return NextResult{}, err
	}
	res := NextResult{CandidateSet: set, Pick: p, Start: started}
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
