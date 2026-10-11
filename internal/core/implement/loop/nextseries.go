package loop

// nextseries.go is the run count of `abcd build next` (itd-2609211116005482
// criterion 5, spc-2609212015048113 scope 7): one pick by default, and with
// --max <n> or --until-empty a series of picks, each made once the one before
// it is delivered or its lane stops, under the pace rule's window, each
// further reason written onto its intent the same way as the first.
//
// The loop is driven by the host session, so a series is too: each `abcd
// build next` makes one move and exits. While a series is in progress every
// `abcd build next` continues it: it reads what the series' last pick has come
// to, and either names that run to drive, says why it picks nothing (the
// window has elapsed, the cap is reached, no candidate is left), or picks
// again. The series' own state is one file beside the runs,
// `.abcd/.work.local/run/next.json`; each pick is an ordinary run the loop's
// own verbs drive. The default single pick writes no series.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// NextSeriesRel is the series' state file, beside the runs. Its name is not a
// run id, so the run listing never reads it as one.
const NextSeriesRel = RunRelDir + "/next.json"

// nextLockRel is the lock one series move holds from its first read to its
// last write, so two calls never make two picks. It is not the runs' lock,
// which the pick's start takes inside it.
const nextLockRel = RunRelDir + "/next.lock"

// NextSeriesSchemaVersion is the series state's shape.
const NextSeriesSchemaVersion = 1

// maxNextSeriesBytes caps the series state read.
const maxNextSeriesBytes = 4 << 20

// CheckSeries is the exclusion of a planned intent the series in progress has
// already picked: a series picks each intent once, whatever its run came to.
const CheckSeries = "series"

// Why a series ended.
const (
	// SeriesStoppedCap: the series made --max picks.
	SeriesStoppedCap = "cap"
	// SeriesStoppedEmpty: no candidate is left.
	SeriesStoppedEmpty = "empty"
)

// What a series' pick has come to, as the series reads its run.
const (
	SeriesPickInProgress = "in-progress"
	// SeriesPickDelivered: the run is complete.
	SeriesPickDelivered = "delivered"
	// SeriesPickHandedBack: a lane of the run was handed back, so the lane
	// stopped; the run stays for the person.
	SeriesPickHandedBack = "handed-back"
	// SeriesPickRemoved: the run's directory was removed, the way out of a
	// handed-back run.
	SeriesPickRemoved = "removed"
)

// NextSeries is one series of picks, between invocations.
type NextSeries struct {
	SchemaVersion int       `json:"schema_version"`
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Max is the cap on picks; 0 with UntilEmpty, which picks until no
	// candidate is left.
	Max        int  `json:"max"`
	UntilEmpty bool `json:"until_empty"`
	// Flags are the pace flags as typed when the series began; every run the
	// series starts is started with them, so each runs on the series' pace.
	Flags SeriesFlags `json:"flags"`
	// Pace is the pace the series began on; its window and pause bound when
	// the series picks.
	Pace            *Pace        `json:"pace"`
	WindowStartedAt *time.Time   `json:"window_started_at,omitempty"`
	NextEligibleAt  *time.Time   `json:"next_eligible_at,omitempty"`
	Picks           []SeriesPick `json:"picks"`
	// Stopped says why the series ended, and EndedAt when; empty while it
	// runs.
	Stopped string     `json:"stopped,omitempty"`
	EndedAt *time.Time `json:"ended_at,omitempty"`
}

// SeriesFlags are --pace, --sub-agents and --fix-rounds as typed; absent when
// not given.
type SeriesFlags struct {
	Pace      *string `json:"pace,omitempty"`
	SubAgents *string `json:"sub_agents,omitempty"`
	FixRounds *string `json:"fix_rounds,omitempty"`
}

// SeriesPick is one pick of a series and what its run has come to.
type SeriesPick struct {
	Intent   string    `json:"intent"`
	RunID    string    `json:"run_id"`
	PickedAt time.Time `json:"picked_at"`
	Outcome  string    `json:"outcome"`
	// Falsified is true once the run's state met the pick's falsifier.
	Falsified bool `json:"falsified"`
}

// SeriesResult is the series as one call leaves it.
type SeriesResult struct {
	// State is the series' state file, relative to the checkout root.
	State string `json:"state"`
	// Started is true when this call began the series.
	Started bool `json:"started"`
	NextSeries
	Complete bool `json:"complete"`
}

// many reports whether the options ask for more than one pick.
func (n NextOptions) many() bool { return n.UntilEmpty || n.Max > 1 }

// countPhrase names a series' run count.
func countPhrase(max int, untilEmpty bool) string {
	if untilEmpty {
		return "--until-empty"
	}
	return fmt.Sprintf("--max %d", max)
}

// beginSeries makes a series' first pick and writes the series. Everything
// that may refuse (the pace, an empty candidate set) is judged before anything
// is written.
func beginSeries(repoRoot string, o Options, n NextOptions, flags paceFlags) (NextResult, error) {
	pace, err := resolvePace(o.roots(repoRoot), flags)
	if err != nil {
		return NextResult{}, err
	}
	set, p, ok, err := choose(repoRoot, o, nil)
	if err != nil {
		return NextResult{}, err
	}
	if !ok {
		return NextResult{}, emptyRefusal(set)
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, RunRelDir, dirPerm); err != nil {
		return NextResult{}, fmt.Errorf("creating %s: %w", RunRelDir, err)
	}
	var res NextResult
	err = withSeriesLock(repoRoot, func() error {
		if _, live, err := readSeries(repoRoot); err != nil {
			return err
		} else if live {
			return contend(StagePick, "", "", "another `abcd build next` began a series in this checkout while this one picked",
				"run `abcd build next` again: it continues that series")
		}
		var err error
		if res, err = startPick(repoRoot, o, set, p); err != nil {
			return err
		}
		now := o.now()
		st := NextSeries{SchemaVersion: NextSeriesSchemaVersion, StartedAt: now, UpdatedAt: now, Max: n.Max, UntilEmpty: n.UntilEmpty,
			Flags: SeriesFlags{Pace: o.Pace, SubAgents: o.SubAgents, FixRounds: o.FixRounds}, Pace: &pace, WindowStartedAt: &now,
			Picks: []SeriesPick{{Intent: p.Chosen.ID, RunID: res.Start.RunID, PickedAt: now, Outcome: SeriesPickInProgress}}}
		if err := writeSeries(repoRoot, st); err != nil {
			return err
		}
		res.Series = &SeriesResult{State: NextSeriesRel, Started: true, NextSeries: st}
		res.Next = seriesPickedNext(res)
		return nil
	})
	return res, err
}

// continueSeries is one move of the series in progress, under its lock: route
// the last pick, stop at the cap, honour the window clock, then pick again or
// end with nothing left.
func continueSeries(repoRoot string, o Options, n NextOptions, flags paceFlags) (NextResult, error) {
	var res NextResult
	err := withSeriesLock(repoRoot, func() error {
		st, live, err := readSeries(repoRoot)
		if err != nil {
			return err
		}
		if !live {
			return contend(StagePick, "", "", "the series in progress in this checkout ended while this call waited for it",
				"run `abcd build next` again")
		}
		if n.many() && (n.UntilEmpty != st.UntilEmpty || n.Max != st.Max) {
			return refuse(StagePick, "", "", fmt.Sprintf("the series started %s is in progress with %s; %s names another, and a run count is set when a series starts",
				st.StartedAt.UTC().Format(time.RFC3339), countPhrase(st.Max, st.UntilEmpty), countPhrase(n.Max, n.UntilEmpty)),
				"run `abcd build next` without a count; the series keeps the one it started with")
		}
		if flags.set {
			if err := resumeWithFlags(StartResult{RunID: "the series started " + st.StartedAt.UTC().Format(time.RFC3339), Pace: st.Pace}, flags); err != nil {
				return err
			}
		}
		res, err = seriesMove(repoRoot, o, &st)
		return err
	})
	return res, err
}

// seriesMove is continueSeries' move over the series it read.
func seriesMove(repoRoot string, o Options, st *NextSeries) (NextResult, error) {
	now := o.now()
	res := NextResult{CandidateSet: CandidateSet{Candidates: []intent.PickCandidate{}, Excluded: []Excluded{}}}
	finish := func(write bool) (NextResult, error) {
		if write {
			st.UpdatedAt = now
			if err := writeSeries(repoRoot, *st); err != nil {
				return NextResult{}, err
			}
		}
		res.Series = &SeriesResult{State: NextSeriesRel, NextSeries: *st, Complete: st.Stopped != ""}
		return res, nil
	}

	// The last pick: what its run has come to, or wait on it.
	changed := false
	for i := range st.Picks {
		pk := &st.Picks[i]
		if pk.Outcome != SeriesPickInProgress {
			continue
		}
		outcome, falsified, err := seriesOutcome(repoRoot, pk.RunID)
		if err != nil {
			return NextResult{}, err
		}
		if outcome == SeriesPickInProgress {
			res.Next = fmt.Sprintf("drive %s's run %s: `abcd implement step --run %s`; run `abcd build next` again once it is delivered or its lane stops, and the series picks again",
				pk.Intent, pk.RunID, pk.RunID)
			return finish(changed)
		}
		pk.Outcome, pk.Falsified, changed = outcome, falsified, true
	}

	if st.Max > 0 && len(st.Picks) >= st.Max {
		st.Stopped, st.EndedAt = SeriesStoppedCap, &now
		res.Next = fmt.Sprintf("the cap is reached: --max %d, and the series made %d pick(s); it stops here. `abcd build next` again makes one pick, or begins a new series with a count",
			st.Max, len(st.Picks))
		return finish(true)
	}

	// The window clock, as a run keeps it: before next_eligible_at nothing is
	// picked; a pause that has ended opens the next window; a window that has
	// elapsed closes here, and the call picks nothing.
	if st.NextEligibleAt != nil {
		if now.Before(*st.NextEligibleAt) {
			at := st.NextEligibleAt.UTC().Format(time.RFC3339)
			res.Next = "nothing before " + at + ": the series' window has elapsed; run `abcd build next` again at or after " + at
			return finish(changed)
		}
		st.NextEligibleAt, st.WindowStartedAt, changed = nil, &now, true
	}
	if st.Pace != nil && st.WindowStartedAt != nil {
		if end := st.WindowStartedAt.Add(time.Duration(st.Pace.WorkMinutes.Value) * time.Minute); !now.Before(end) {
			if st.Pace.PauseMinutes.Value == 0 {
				// No pause: the next window opens at once.
				st.WindowStartedAt, changed = &now, true
			} else {
				until := now.Add(time.Duration(st.Pace.PauseMinutes.Value) * time.Minute)
				st.NextEligibleAt = &until
				at := until.UTC().Format(time.RFC3339)
				res.Next = fmt.Sprintf("nothing before %s: the series' %d-minute window has elapsed (a %d-minute pause); run `abcd build next` again at or after %s",
					at, st.Pace.WorkMinutes.Value, st.Pace.PauseMinutes.Value, at)
				return finish(true)
			}
		}
	}

	// Pick again, on the series' own pace flags.
	so := o
	so.Pace, so.SubAgents, so.FixRounds = st.Flags.Pace, st.Flags.SubAgents, st.Flags.FixRounds
	set, p, ok, err := choose(repoRoot, so, st)
	if err != nil {
		return NextResult{}, err
	}
	res.CandidateSet = set
	if !ok {
		st.Stopped, st.EndedAt = SeriesStoppedEmpty, &now
		res.Next = fmt.Sprintf("no candidate is left: the series made %d pick(s) and ends here; %s",
			len(st.Picks), strings.TrimPrefix(noCandidateReason(set.Excluded), "no planned intent passes every check `abcd build <itd-N>` runs; "))
		return finish(true)
	}
	// What the last pick came to is written before the next one starts, so a
	// refused start loses none of it.
	if changed {
		st.UpdatedAt = now
		if err := writeSeries(repoRoot, *st); err != nil {
			return NextResult{}, err
		}
	}
	picked, err := startPick(repoRoot, so, set, p)
	if err != nil {
		return NextResult{}, err
	}
	st.Picks = append(st.Picks, SeriesPick{Intent: p.Chosen.ID, RunID: picked.Start.RunID, PickedAt: now, Outcome: SeriesPickInProgress})
	res = picked
	out, err := finish(true)
	if err != nil {
		return NextResult{}, err
	}
	out.Next = seriesPickedNext(out)
	return out, nil
}

// seriesPickedNext is what a call that picked within a series says next.
func seriesPickedNext(res NextResult) string {
	return res.Start.Next + "; once it is delivered or its lane stops, run `abcd build next` again and the series picks again (" +
		countPhrase(res.Series.Max, res.Series.UntilEmpty) + ")"
}

// seriesOutcome reads what a pick's run has come to, and whether its state
// met the pick's falsifier.
func seriesOutcome(repoRoot, runID string) (string, bool, error) {
	if !fsutil.IsRealDir(filepath.Join(repoRoot, filepath.FromSlash(runRel(runID)))) {
		return SeriesPickRemoved, false, nil
	}
	run, err := ReadState(repoRoot, runID)
	if err != nil {
		return "", false, err
	}
	_, _, falsified := pickFalsifiedBy(run)
	switch {
	case run.Complete():
		return SeriesPickDelivered, falsified, nil
	case run.handedBack():
		return SeriesPickHandedBack, falsified, nil
	}
	return SeriesPickInProgress, falsified, nil
}

// exclude takes out of the candidate set every intent the series has picked,
// naming its run: a series picks each intent once.
func (s NextSeries) exclude(set CandidateSet) CandidateSet {
	kept := set.Candidates[:0:0]
	for _, c := range set.Candidates {
		i := slices.IndexFunc(s.Picks, func(p SeriesPick) bool { return p.Intent == c.ID })
		if i < 0 {
			kept = append(kept, c)
			continue
		}
		set.Excluded = append(set.Excluded, Excluded{ID: c.ID, Check: CheckSeries,
			Reason: "this series picked it as run " + s.Picks[i].RunID + ", which came to " + s.Picks[i].Outcome})
	}
	set.Candidates = kept
	return set
}

// withSeriesLock runs fn holding the series' lock.
func withSeriesLock(repoRoot string, fn func() error) error {
	err := fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(nextLockRel)), lockTimeout, fn)
	if errors.Is(err, fsutil.ErrLockContention) {
		return contend(StagePick, "", "", "another `abcd build next` is moving in this checkout", "back off and retry")
	}
	return err
}

// readSeries reads the series' state; live is false when there is none, or
// the one there has ended.
func readSeries(repoRoot string) (NextSeries, bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return NextSeries{}, false, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, NextSeriesRel, maxNextSeriesBytes)
	if errors.Is(err, os.ErrNotExist) {
		return NextSeries{}, false, nil
	}
	if err != nil {
		return NextSeries{}, false, refuse(StagePick, "", "", fmt.Sprintf("%s cannot be read as the series' state: %v", NextSeriesRel, err),
			"`abcd build next` writes a regular file in a real directory; restore that, or remove "+NextSeriesRel)
	}
	var st NextSeries
	if err := jsonstrict.Decode(data, &st); err != nil || st.SchemaVersion != NextSeriesSchemaVersion {
		why := fmt.Sprintf("schema_version %d (this abcd reads %d)", st.SchemaVersion, NextSeriesSchemaVersion)
		if err != nil {
			why = err.Error()
		}
		return NextSeries{}, false, refuse(StagePick, "", "", NextSeriesRel+" does not parse as the series' state: "+why,
			"`abcd build next` is the file's only writer; restore it or remove it")
	}
	if st.Max < 0 || (st.UntilEmpty && st.Max != 0) || (!st.UntilEmpty && st.Max < 2) {
		return NextSeries{}, false, refuse(StagePick, "", "", NextSeriesRel+" names no run count a series takes",
			"`abcd build next` is the file's only writer; restore it or remove it")
	}
	for _, p := range st.Picks {
		if !recordid.ValidIntentID(p.Intent) || !ValidRunID(p.RunID) {
			return NextSeries{}, false, refuse(StagePick, "", "", NextSeriesRel+" names a pick that is not an intent and a run",
				"`abcd build next` is the file's only writer; restore it or remove it")
		}
	}
	return st, st.Stopped == "", nil
}

// writeSeries writes the series' state atomically.
func writeSeries(repoRoot string, st NextSeries) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the series' state: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomicInRoot(root, NextSeriesRel, data, filePerm)
}
