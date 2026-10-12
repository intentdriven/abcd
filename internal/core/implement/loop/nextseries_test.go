package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The run count of `abcd build next` (itd-2609211116005482 criterion 5,
// spc-2609212015048113 scope 7): one pick by default; --max <n> and
// --until-empty keep the series going, one pick per invocation once the last
// pick is delivered or its lane stops, under the pace rule's window, each
// further reason written the same way.

// seriesRepo holds three planned intents every check passes, itd-10 the
// readiest (its spec carries a footprint), then itd-11 and itd-12 tied.
func seriesRepo(t *testing.T) string {
	t.Helper()
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", "")},
		"12": {pickIntent("12", "", settledQuestions, gwt), pickSpec("12", "")},
	})
	return repo.Root()
}

// deliver marks every lane of a run done and its spec steps landed, as a
// delivered run's state reads.
func deliver(t *testing.T, root, runID string) {
	t.Helper()
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		for i := range st.Lanes {
			st.Lanes[i].Stage = StageDone
			st.Lanes[i].Awaits = nil
		}
		st.Pending = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// handBackUnachievable stops a run's first lane as the fix-round bound does.
func handBackUnachievable(t *testing.T, root, runID string, at time.Time) {
	t.Helper()
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		handBackLane(st, &st.Lanes[0], HandBack{Verdict: VerdictUnachievable, Round: 1, FixRounds: 0, Verdicts: "ruthless FIX FIRST"}, "", at)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

func seriesAbsent(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(NextSeriesRel))); !os.IsNotExist(err) {
		t.Fatalf("no series state is written: %v", err)
	}
}

// TestADefaultPickLeavesNoSeries is criterion 5's default half: one pick, and
// nothing that would pick again once it is delivered.
func TestADefaultPickLeavesNoSeries(t *testing.T) {
	root := seriesRepo(t)
	for _, n := range []NextOptions{{}, {Max: 1}} {
		res, err := Next(root, Options{}, n)
		if err != nil {
			t.Fatal(err)
		}
		if res.Pick == nil || res.Start == nil || res.Series != nil {
			t.Fatalf("%+v makes one pick and starts no series: %+v", n, res)
		}
		seriesAbsent(t, root)
	}
}

// TestMaxPicksAgainOnceThePickIsDelivered is criterion 5's flag half under
// --max: the first call picks and starts a series; a call while the pick's
// lane is in progress picks nothing and names the run to drive; once it is
// delivered the next call picks again among the rest, writing the reason the
// same way; at the cap the series stops.
func TestMaxPicksAgainOnceThePickIsDelivered(t *testing.T) {
	root := seriesRepo(t)
	first, err := Next(root, Options{}, NextOptions{Max: 2})
	if err != nil {
		t.Fatal(err)
	}
	if first.Pick == nil || first.Pick.Chosen.ID != "itd-10" || first.Series == nil || !first.Series.Started ||
		first.Series.Max != 2 || len(first.Series.Picks) != 1 || first.Series.Picks[0].Outcome != SeriesPickInProgress {
		t.Fatalf("the first call picks the readiest and starts the series: %+v", first)
	}

	waiting, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if waiting.Pick != nil || waiting.Start != nil || waiting.Series == nil ||
		!strings.Contains(waiting.Next, "abcd implement step --run "+first.Start.RunID) {
		t.Fatalf("while the pick's lane is in progress nothing is picked and the run to drive is named: %+v", waiting)
	}

	deliver(t, root, first.Start.RunID)
	second, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Pick == nil || second.Pick.Chosen.ID != "itd-11" || !second.Pick.TieBrokenByAge || second.Start == nil {
		t.Fatalf("once delivered the series picks again, the oldest among the tied: %+v", second)
	}
	if second.Series.Picks[0].Outcome != SeriesPickDelivered || len(second.Series.Picks) != 2 {
		t.Fatalf("the series records the first pick delivered and the second in progress: %+v", second.Series.Picks)
	}
	excludedBySeries := false
	for _, e := range second.Excluded {
		if e.ID == "itd-10" && e.Check == CheckSeries && strings.Contains(e.Reason, first.Start.RunID) {
			excludedBySeries = true
		}
	}
	if !excludedBySeries {
		t.Fatalf("an intent the series already picked is excluded naming its run: %+v", second.Excluded)
	}
	st, err := ReadState(root, second.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Pick == nil || !strings.HasPrefix(st.Pick.Entry, "picked by run "+second.Start.RunID+" on ") ||
		!strings.Contains(st.Pick.Entry, "tie was broken by age") || second.Entry != st.Pick.Entry {
		t.Fatalf("the further pick's reason is written the same way: %+v", st.Pick)
	}

	deliver(t, root, second.Start.RunID)
	capped, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if capped.Pick != nil || capped.Series.Stopped != SeriesStoppedCap || !capped.Series.Complete || !strings.Contains(capped.Next, "--max 2") {
		t.Fatalf("at the cap the series stops and names it: %+v", capped)
	}
	if runs, _ := Runs(root); len(runs) != 2 {
		t.Fatalf("no pick past the cap: %d runs", len(runs))
	}

	// The series has ended: a plain call is the default single pick again.
	again, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Pick == nil || again.Series != nil {
		t.Fatalf("after a series ends a plain call is one pick: %+v", again)
	}
}

// TestUntilEmptyPicksPastAHandBackAndEndsWhenNothingIsLeft: a handed-back
// pick is a lane that stopped, so the series picks again, and records the
// pick falsified; with no candidate left the series ends, naming each
// exclusion, without a refusal.
func TestUntilEmptyPicksPastAHandBackAndEndsWhenNothingIsLeft(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", "")},
	})
	root := repo.Root()
	first, err := Next(root, Options{}, NextOptions{UntilEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	if first.Series == nil || !first.Series.UntilEmpty || first.Pick.Chosen.ID != "itd-10" {
		t.Fatalf("--until-empty starts a series: %+v", first)
	}
	handBackUnachievable(t, root, first.Start.RunID, time.Now())
	second, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Pick == nil || second.Pick.Chosen.ID != "itd-11" {
		t.Fatalf("a handed-back pick's lane stopped, so the series picks again: %+v", second)
	}
	if p := second.Series.Picks[0]; p.Outcome != SeriesPickHandedBack || !p.Falsified {
		t.Fatalf("the series records the handed-back pick falsified: %+v", p)
	}
	deliver(t, root, second.Start.RunID)
	empty, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatalf("an emptied series ends; it is not a refusal: %v", err)
	}
	if empty.Pick != nil || empty.Series.Stopped != SeriesStoppedEmpty || !empty.Series.Complete || len(empty.Excluded) != 2 {
		t.Fatalf("with nothing left the series ends naming each exclusion: %+v", empty)
	}
	if !strings.Contains(empty.Next, "itd-10 (run") || !strings.Contains(empty.Next, "itd-11 (series") {
		t.Fatalf("the ending names each excluded intent and its check: %s", empty.Next)
	}
}

// TestASeriesPicksOnlyInsideItsWindow is the pace rule over the series, under
// a fixed clock: a pick due after the window's end writes next_eligible_at
// and picks nothing; before that time nothing is picked; after it the next
// call picks, and every run the series starts runs on the pace it began with.
func TestASeriesPicksOnlyInsideItsWindow(t *testing.T) {
	root := seriesRepo(t)
	c := &clock{t: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}
	pace := "1/5"
	first, err := Next(root, Options{Now: c.now, Pace: &pace}, NextOptions{Max: 3})
	if err != nil {
		t.Fatal(err)
	}
	if first.Series.Pace == nil || first.Series.Pace.WorkMinutes.Value != 1 || first.Series.Pace.PauseMinutes.Value != 5 {
		t.Fatalf("the series runs on the pace it was started with: %+v", first.Series)
	}
	deliver(t, root, first.Start.RunID)

	c.t = c.t.Add(2 * time.Minute)
	paused, err := Next(root, Options{Now: c.now}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := c.t.Add(5 * time.Minute)
	if paused.Pick != nil || paused.Series.NextEligibleAt == nil || !paused.Series.NextEligibleAt.Equal(want) ||
		paused.Series.Picks[0].Outcome != SeriesPickDelivered {
		t.Fatalf("past the window's end nothing is picked and next_eligible_at is written: %+v", paused.Series)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(NextSeriesRel)))
	if err != nil || !strings.Contains(string(raw), "\"next_eligible_at\"") {
		t.Fatalf("the series' state takes next_eligible_at: %v %s", err, raw)
	}

	c.t = c.t.Add(time.Minute)
	if still, err := Next(root, Options{Now: c.now}, NextOptions{}); err != nil || still.Pick != nil || !strings.Contains(still.Next, "nothing before") {
		t.Fatalf("before next_eligible_at nothing is picked: %+v %v", still, err)
	}

	c.t = want.Add(time.Second)
	resumed, err := Next(root, Options{Now: c.now}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Pick == nil || resumed.Start == nil || resumed.Start.Pace == nil ||
		resumed.Start.Pace.WorkMinutes.Value != 1 || resumed.Start.Pace.PauseMinutes.Value != 5 {
		t.Fatalf("after next_eligible_at the series picks, and the run takes the series' pace: %+v", resumed)
	}
}

// TestASeriesRefusesWhatItCannotHonour: a negative count, two counts at once,
// a live series asked for another count or pace, and a first pick with no
// candidate, each refused at the pick writing nothing.
func TestASeriesRefusesWhatItCannotHonour(t *testing.T) {
	root := seriesRepo(t)
	for _, n := range []NextOptions{{Max: -1}, {Max: 2, UntilEmpty: true}} {
		if _, err := Next(root, Options{}, n); mustRefusal(t, err).Stage != StagePick {
			t.Fatalf("%+v is refused at the pick", n)
		}
	}
	runTierAbsent(t, root)

	if _, err := Next(root, Options{}, NextOptions{Max: 2}); err != nil {
		t.Fatal(err)
	}
	// --max 1 is a count too (iss-2610120925509505): given on a live series it
	// is refused, never taken as a move of that series.
	for _, n := range []NextOptions{{Max: 3}, {Max: 1}, {UntilEmpty: true}} {
		r := mustRefusal(t, func() error { _, err := Next(root, Options{}, n); return err }())
		if r.Stage != StagePick || !strings.Contains(r.Reason, "--max 2") {
			t.Fatalf("a live series keeps the count it started with: %+v", r)
		}
	}
	if runs, _ := Runs(root); len(runs) != 1 {
		t.Fatalf("a refused count starts no run: %d runs", len(runs))
	}
	if res, err := Next(root, Options{}, NextOptions{Max: 2}); err != nil || res.Series == nil {
		t.Fatalf("the series' own count continues it: %+v %v", res, err)
	}
	pace := "7/7"
	if _, err := Next(root, Options{Pace: &pace}, NextOptions{}); err == nil {
		t.Fatal("a live series keeps the pace it started with")
	}
}

// TestAnEmptyFirstPickOfASeriesWritesNothing: criterion 1's refusal holds for
// a series' first pick.
func TestAnEmptyFirstPickOfASeriesWritesNothing(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "held: \"not yet\"\n", settledQuestions, gwt), pickSpec("10", "")}})
	_, err := Next(repo.Root(), Options{}, NextOptions{UntilEmpty: true})
	if r := mustRefusal(t, err); r.Stage != StagePick || len(r.Excluded) != 1 {
		t.Fatalf("the empty set is refused naming every exclusion: %+v", r)
	}
	runTierAbsent(t, repo.Root())
}

// TestTheRunRecordCarriesThePickAndItsFalsifier is criterion 6 and scope 8:
// the run record names the pick with its score table, and whether its
// falsifier was met, read from the lane's state; a run `build <itd-N>`
// started carries no pick.
func TestTheRunRecordCarriesThePickAndItsFalsifier(t *testing.T) {
	root := seriesRepo(t)
	res, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ReadRecord(root, res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Pick == nil || rec.Pick.Chosen.ID != "itd-10" || len(rec.Pick.Candidates) != 3 || rec.Pick.Candidates[0].Score.Total != 300 ||
		rec.Pick.Falsifier == "" || rec.Pick.Falsified || rec.Pick.Entry != res.Entry || rec.Pick.Excluded == nil {
		t.Fatalf("the record carries the pick's score table, its falsifier not met: %+v", rec.Pick)
	}
	handBackUnachievable(t, root, res.Start.RunID, time.Now())
	rec, err = ReadRecord(root, res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Pick.Falsified || !strings.Contains(rec.Pick.FalsifiedBy, "lane-1") || !strings.Contains(rec.Pick.FalsifiedBy, VerdictUnachievable) {
		t.Fatalf("a handed-back lane meets the falsifier, and the record names it: %+v", rec.Pick)
	}

	other := seriesRepo(t)
	plain, err := Start(other, "itd-11", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rec, err := ReadRecord(other, plain.RunID); err != nil || rec.Pick != nil {
		t.Fatalf("a run `build <itd-N>` started carries no pick: %+v %v", rec.Pick, err)
	}
}

// TestARemovedPickIsPickedPast: a pick whose run directory was removed (the
// way out of a handed-back run) is a lane that stopped, so the series picks
// again, records the pick removed, and excludes the intent naming it.
func TestARemovedPickIsPickedPast(t *testing.T) {
	root := seriesRepo(t)
	first, err := Next(root, Options{}, NextOptions{Max: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(runRel(first.Start.RunID)))); err != nil {
		t.Fatal(err)
	}
	second, err := Next(root, Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Pick == nil || second.Pick.Chosen.ID != "itd-11" || second.Start == nil {
		t.Fatalf("a removed pick's lane stopped, so the series picks again: %+v", second)
	}
	if p := second.Series.Picks[0]; p.Outcome != SeriesPickRemoved || p.Falsified || len(second.Series.Picks) != 2 {
		t.Fatalf("the series records the first pick removed: %+v", second.Series.Picks)
	}
	removed := false
	for _, e := range second.Excluded {
		if e.ID == "itd-10" && e.Check == CheckSeries && strings.Contains(e.Reason, first.Start.RunID+", which came to "+SeriesPickRemoved) {
			removed = true
		}
	}
	if !removed {
		t.Fatalf("the removed pick's intent is excluded naming what its run came to: %+v", second.Excluded)
	}
}

// TestASeriesMoveBacksOffWhileAnotherHoldsItsLock: a move that cannot take
// next.lock within the lock timeout is refused as contention (exit 3 at the
// front door), at the pick, and starts nothing.
func TestASeriesMoveBacksOffWhileAnotherHoldsItsLock(t *testing.T) {
	root := seriesRepo(t)
	first, err := Next(root, Options{}, NextOptions{Max: 2})
	if err != nil {
		t.Fatal(err)
	}
	deliver(t, root, first.Start.RunID)
	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(NextSeriesRel)))
	if err != nil {
		t.Fatal(err)
	}
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(root, filepath.FromSlash(nextLockRel)), time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	_, err = Next(root, Options{}, NextOptions{})
	close(release)
	if lockErr := <-done; lockErr != nil {
		t.Fatal(lockErr)
	}
	if r := mustRefusal(t, err); !r.Contention || r.Stage != StagePick || !strings.Contains(r.Reason, "another `abcd build next` is moving") {
		t.Fatalf("a held series lock is contention at the pick: %+v", r)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(NextSeriesRel)))
	if err != nil || string(after) != string(before) {
		t.Fatalf("a move that backed off changes nothing in the series (%v)", err)
	}
	if runs, _ := Runs(root); len(runs) != 1 {
		t.Fatalf("a move that backed off starts no run: %d runs", len(runs))
	}
}

// TestASeriesRefusesAnOutcomeItNeverWrites: a pick's outcome in next.json is
// one of the four the series writes; any other text (a hand edit, or a file a
// hostile checkout ships, carrying terminal control runes) is refused at the
// pick, so it never reaches a render or the --json payload.
func TestASeriesRefusesAnOutcomeItNeverWrites(t *testing.T) {
	root := seriesRepo(t)
	if _, err := Next(root, Options{}, NextOptions{Max: 2}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(NextSeriesRel))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`\u001b[2Jdelivered`, `shipped`, ``} {
		forged := strings.Replace(string(raw), `"outcome": "in-progress"`, `"outcome": "`+bad+`"`, 1)
		if forged == string(raw) {
			t.Fatal("the series' state names its pick's outcome")
		}
		if err := os.WriteFile(path, []byte(forged), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Next(root, Options{}, NextOptions{})
		if r := mustRefusal(t, err); r.Stage != StagePick || !strings.Contains(r.Reason, "outcome the series never writes") {
			t.Fatalf("outcome %q is refused at the pick: %+v", bad, r)
		}
		if runs, _ := Runs(root); len(runs) != 1 {
			t.Fatalf("a refused series starts no run: %d runs", len(runs))
		}
	}
}

// TestASeriesRefusesPaceFlagsItNeverWrites (iss-2610120925598775): the
// series' stored pace flags are read in the form the flags take, as typed; a
// file whose flags carry other text (a hand edit, or a file a hostile
// checkout ships with bidi or C1 control runes) is refused at the pick, so it
// never reaches the --json payload, and a series written with real flags
// still reads back.
func TestASeriesRefusesPaceFlagsItNeverWrites(t *testing.T) {
	root := seriesRepo(t)
	if _, err := Next(root, Options{Pace: strp("90/0"), SubAgents: strp("2"), FixRounds: strp("1")}, NextOptions{Max: 2}); err != nil {
		t.Fatal(err)
	}
	if res, err := Next(root, Options{}, NextOptions{}); err != nil || res.Series == nil || res.Series.Flags.Pace == nil ||
		*res.Series.Flags.Pace != "90/0" || *res.Series.Flags.SubAgents != "2" || *res.Series.Flags.FixRounds != "1" {
		t.Fatalf("a series written with real flags reads back: %+v %v", res.Series, err)
	}
	path := filepath.Join(root, filepath.FromSlash(NextSeriesRel))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ field, bad string }{
		{`"pace": "90/0"`, `"pace": "\u202e90/0"`},
		{`"sub_agents": "2"`, `"sub_agents": "\u009b2"`},
		{`"fix_rounds": "1"`, `"fix_rounds": "1\u202e\u009b"`},
	} {
		forged := strings.Replace(string(raw), c.field, c.bad, 1)
		if forged == string(raw) {
			t.Fatalf("the series' state carries %s", c.field)
		}
		if err := os.WriteFile(path, []byte(forged), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Next(root, Options{}, NextOptions{})
		r := mustRefusal(t, err)
		if r.Stage != StagePick || !strings.Contains(r.Reason, "pace flags") {
			t.Fatalf("%s is refused at the pick: %+v", c.bad, r)
		}
		if strings.ContainsAny(r.Reason+r.Remedy, "\u202e\u009b") {
			t.Fatalf("the refusal carries a raw control rune: %q", r.Reason+r.Remedy)
		}
		if runs, _ := Runs(root); len(runs) != 1 {
			t.Fatalf("a refused series starts no run: %d runs", len(runs))
		}
	}
}

// TestAPickedRunHandedBackAfterItsFixRoundsLeavesTheEntryAlone is criterion
// 6's "the entry is not edited": a picked run whose lane takes its one fix
// round and is then handed back as unachievable meets the pick's falsifier in
// the run record, and the intent's grounds, as the pick commit wrote them on
// the lane's branch, are byte for byte what they were; the checkout the run
// started from is never written.
func TestAPickedRunHandedBackAfterItsFixRoundsLeavesTheEntryAlone(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)}})
	ir, _ := pickRel("10")
	before, err := os.ReadFile(filepath.Join(repo.Root(), ir))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Next(repo.Root(), Options{FixRounds: strp("1")}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := res.Start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")
	picked := repo.Git("show", l.PickSHA+":"+ir)
	if strings.Count(picked, "- pursued: picked by run "+id+" on ") != 1 {
		t.Fatalf("the pick commit carries the run's one entry:\n%s", picked)
	}

	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
	passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
	fix := laneCommit(t, repo, l, "fix.txt")
	fixed(t, repo, id, stages, "applied the finding in "+fix+"\n", fix)
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	handBack(t, repo, id, stages, RoleAuditor, "NOT_MET")
	step, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || step.Stage != StageHandedBack || step.HandBack == nil {
		t.Fatalf("the round past the cap hands the picked lane back: %+v %v", step, err)
	}
	rec, err := ReadRecord(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Pick == nil || !rec.Pick.Falsified || !strings.Contains(rec.Pick.FalsifiedBy, VerdictUnachievable) {
		t.Fatalf("the hand-back meets the pick's falsifier in the run record: %+v", rec.Pick)
	}

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	if head := repo.Git("show", lane.Branch+":"+ir); head != picked {
		t.Fatalf("the intent on the lane's branch is the pick commit's, unedited:\n%s", head)
	}
	got, err := os.ReadFile(filepath.Join(lane.Worktree, ir))
	if err != nil || strings.TrimRight(string(got), "\n") != strings.TrimRight(picked, "\n") {
		t.Fatalf("the intent in the lane's worktree is the pick commit's, unedited (%v):\n%s", err, got)
	}
	if !strings.Contains(string(got), "- pursued: "+rec.Pick.Entry) {
		t.Fatalf("the grounds entry is the one the record names:\n%s", got)
	}
	if now, err := os.ReadFile(filepath.Join(repo.Root(), ir)); err != nil || string(now) != string(before) {
		t.Fatalf("the checkout the run started from is never written (%v)", err)
	}
}
