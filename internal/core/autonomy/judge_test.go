package autonomy

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// judge_test.go — step 5 of spc-2610180914469489: the stop verdict, pure over
// the owed set, the session's state and the host's payload.

func awayRec(now time.Time, holdPushes bool) *Record {
	r, err := Begin(nil, alice, now.Add(-time.Hour), time.Time{}, holdPushes)
	if err != nil {
		panic(err)
	}
	return r
}

var (
	behindPR    = OwedItem{Kind: OwedBehind, Capped: true, Branch: "feat/behind", PR: 12, Head: "aaaa"}
	failingPR   = OwedItem{Kind: OwedFailing, Capped: true, Branch: "feat/failing", PR: 13, Head: "bbbb"}
	unpushedBr  = OwedItem{Kind: OwedUnpushed, Capped: true, Branch: "feat/unpushed", Head: "cccc"}
	stackedBr   = OwedItem{Kind: OwedStacked, Capped: true, Branch: "feat/stacked", Head: "dddd"}
	uncarried   = OwedItem{Kind: OwedUncarried, Capped: true, Record: "iss-5"}
	renewalDue  = OwedItem{Kind: OwedRenewal, Record: "iss-7"}
	stackedWait = OwedItem{Kind: OwedStacked, Branch: "feat/waiting", Head: "eeee"} // waiting only on running checks
)

func owedOf(items ...OwedItem) Owed { return Owed{Items: items} }

func crons(exprs ...string) []SessionCron {
	var out []SessionCron
	for i, e := range exprs {
		out = append(out, SessionCron{ID: "c" + string(rune('0'+i)), Schedule: e, Recurring: true, Prompt: "look"})
	}
	return out
}

// judgeAt runs Judge read in London, on a minute boundary.
func judgeAt(t *testing.T, rec *Record, owed Owed, in StopInput, st SessionState, now time.Time) Verdict {
	t.Helper()
	if in.Location == nil {
		in.Location = london(t)
	}
	return Judge(rec, owed, in, st, now)
}

// TestJudgeRows: each of the seven away rows and both present rows, each with
// stop_hook_active true and false.
func TestJudgeRows(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	rec := awayRec(now, false)
	owed := owedOf(behindPR)
	seen := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint()}
	none := Owed{}
	seenNone := SessionState{Fingerprint: none.Fingerprint(), WakeFingerprint: none.CappedFingerprint()}
	wake := crons("*/10 * * * *")

	for _, active := range []bool{false, true} {
		for _, c := range []struct {
			name  string
			rec   *Record
			owed  Owed
			in    StopInput
			st    SessionState
			row   int
			block bool
			state string
			want  []string
		}{
			{"1 prose question", rec, owed, StopInput{LastAssistantMessage: "Done.\nShall I merge it?\n"}, seen, 1, true, StateWorking, []string{"abcd decide", "abcd mode queue"}},
			{"2 owed, first block", rec, owed, StopInput{SessionCrons: wake}, SessionState{}, 2, true, StateWorking, []string{"feat/behind", "#12", "1 item"}},
			{"2 owed, set changed", rec, owedOf(behindPR, unpushedBr), StopInput{SessionCrons: wake}, seen, 2, true, StateWorking, []string{"feat/behind", "2 items"}},
			{"3 owed, unchanged, wake", rec, owed, StopInput{SessionCrons: wake}, seen, 3, false, StateWaiting, nil},
			{"4 owed, unchanged, wake, two wakes", rec, owed, StopInput{SessionCrons: wake}, withWakes(seen, 2), 4, true, StateWorking, []string{"feat/behind", "two wakes"}},
			{"5 owed, unchanged, no wake", rec, owed, StopInput{}, seen, 5, true, StateWorking, []string{"feat/behind", "take another", "schedule a look before"}},
			{"6 none, wake", rec, none, StopInput{SessionCrons: crons("*/30 * * * *")}, seenNone, 6, false, StateWaiting, nil},
			{"7 none, no wake", rec, none, StopInput{}, seenNone, 7, true, StateWorking, []string{"next ready item", "reserve", "schedule a look before"}},
			{"present, owed", nil, owed, StopInput{}, SessionState{}, 0, false, "", nil},
			{"present, none", nil, none, StopInput{}, SessionState{}, 0, false, "", nil},
		} {
			c.in.StopHookActive = active
			v := judgeAt(t, c.rec, c.owed, c.in, c.st, now)
			name := c.name
			if active {
				name += " (stop_hook_active)"
			}
			if v.Row != c.row || v.Block != c.block {
				t.Errorf("%s: row %d block %v, want row %d block %v (reason %q)", name, v.Row, v.Block, c.row, c.block, v.Reason)
				continue
			}
			if c.rec != nil && v.State.State != c.state {
				t.Errorf("%s: state %q, want %q", name, v.State.State, c.state)
			}
			for _, w := range c.want {
				if !strings.Contains(v.Reason, w) {
					t.Errorf("%s: the reason misses %q: %q", name, w, v.Reason)
				}
			}
			if !c.block && v.Reason != "" {
				t.Errorf("%s: an allow carries a reason: %q", name, v.Reason)
			}
		}
	}
	// Present: with owed work, allowed with a line naming it; with none, silent.
	v := judgeAt(t, nil, owed, StopInput{}, SessionState{}, now)
	if v.Block || !strings.Contains(v.SystemMessage, "feat/behind") || v.Save {
		t.Errorf("present with work owed: %+v; want allowed, a line naming the work, nothing saved", v)
	}
	if v := judgeAt(t, nil, Owed{}, StopInput{}, SessionState{}, now); v.Block || v.SystemMessage != "" || v.Save {
		t.Errorf("present with nothing owed: %+v; want a silent allow", v)
	}
	// A lapsed record is present.
	lapsed := awayRec(now.Add(-8*24*time.Hour), false)
	if v := judgeAt(t, lapsed, owed, StopInput{}, seen, now); v.Block || v.Row != 0 {
		t.Errorf("a lapsed record judged as away: %+v", v)
	}
}

func withWakes(s SessionState, n int) SessionState { s.Wakes = n; return s }

// sequence runs Judge over a run of stops, carrying the state through a save
// and a load of the session's state file, as a new hook process would.
type sequence struct {
	t    *testing.T
	rec  *Record
	root string
	id   string
	now  time.Time
}

func newSequence(t *testing.T, rec *Record, now time.Time) *sequence {
	return &sequence{t: t, rec: rec, root: t.TempDir(), id: "sess-1", now: now}
}

func (s *sequence) stop(owed Owed, in StopInput) Verdict {
	s.t.Helper()
	st, err := LoadSession(s.root, s.id)
	if err != nil {
		s.t.Fatal(err)
	}
	v := judgeAt(s.t, s.rec, owed, in, st, s.now)
	if v.Save {
		if err := SaveSession(s.root, s.id, v.State); err != nil {
			s.t.Fatal(err)
		}
	}
	s.now = s.now.Add(10 * time.Minute)
	return v
}

// TestWakeReleasesUnchangedSetTwice: with `*/10 * * * *` and an unchanged
// capped item, the first two stops with the wake are allowed and the third
// and each later one blocked; a set of only a renewal due, or of a stacked
// branch waiting only on running checks, keeps releasing; a renewal falling
// due between the wakes neither resets nor adds to the count; a change in the
// capped items resets it; the count is read back from the state file.
func TestWakeReleasesUnchangedSetTwice(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	for _, item := range []OwedItem{behindPR, failingPR, unpushedBr, stackedBr, uncarried} {
		s := newSequence(t, awayRec(now, false), now)
		owed := owedOf(item)
		if v := s.stop(owed, wake); v.Row != 2 {
			t.Fatalf("%s: the first stop is row %d, want the first block", item.Kind, v.Row)
		}
		for i, want := range []bool{false, false, true, true, true} {
			v := s.stop(owed, wake)
			if v.Block != want {
				t.Errorf("%s: wake %d: block %v, want %v (row %d)", item.Kind, i+1, v.Block, want, v.Row)
			}
		}
	}
	for _, item := range []OwedItem{renewalDue, stackedWait} {
		s := newSequence(t, awayRec(now, false), now)
		owed := owedOf(item)
		s.stop(owed, wake)
		for i := 0; i < 5; i++ {
			if v := s.stop(owed, wake); v.Block {
				t.Errorf("%s (not capped): wake %d blocked (row %d); want it to keep releasing", item.Kind, i+1, v.Row)
			}
		}
	}
	// A renewal falling due between the wakes: the full set changes (one
	// block), the capped count neither resets nor adds.
	s := newSequence(t, awayRec(now, false), now)
	s.stop(owedOf(behindPR), wake)
	if v := s.stop(owedOf(behindPR), wake); v.Block {
		t.Fatal("first wake blocked")
	}
	if v := s.stop(owedOf(behindPR, renewalDue), wake); v.Row != 2 {
		t.Fatalf("a renewal falling due is row %d, want the set-changed block", v.Row)
	}
	if v := s.stop(owedOf(behindPR, renewalDue), wake); v.Block {
		t.Errorf("second wake after a renewal fell due blocked: the renewal reset or added to the count")
	}
	if v := s.stop(owedOf(behindPR, renewalDue), wake); !v.Block || v.Row != 4 {
		t.Errorf("third wake: row %d block %v, want row 4", v.Row, v.Block)
	}
	// A change in the capped items resets it.
	if v := s.stop(owedOf(failingPR, renewalDue), wake); v.Row != 2 {
		t.Fatalf("a changed capped set is row %d, want 2", v.Row)
	}
	for i := 0; i < 2; i++ {
		if v := s.stop(owedOf(failingPR, renewalDue), wake); v.Block {
			t.Errorf("after the capped set changed, wake %d blocked: the count did not reset", i+1)
		}
	}
	st, _ := LoadSession(s.root, s.id)
	if st.Wakes != 2 || st.WakeFingerprint != owedOf(failingPR).CappedFingerprint() {
		t.Errorf("state file holds wakes=%d fp=%q", st.Wakes, st.WakeFingerprint)
	}
}

// TestJudgeEscapeBoundReproducesNoStall: owed, unchanged, stop_hook_active,
// no wake: blocked, not allowed.
func TestJudgeEscapeBoundReproducesNoStall(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	owed := owedOf(behindPR)
	st := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint(), Blocks: 3}
	v := judgeAt(t, awayRec(now, false), owed, StopInput{StopHookActive: true}, st, now)
	if !v.Block {
		t.Fatalf("owed, unchanged, stop_hook_active, no wake: allowed (%+v)", v)
	}
}

// TestJudgeCapRecordsStop: the block passing the cap, and one passing a cap
// the environment raised, records the stop labelled possibly forced by the
// cap; rows 2, 4 and 5 as work owed, row 7 as no next look.
func TestJudgeCapRecordsStop(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	rec := awayRec(now, false)
	owed := owedOf(behindPR)
	seen := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint()}
	for _, c := range []struct {
		cap, blocks int
		record      bool
	}{{0, 7, false}, {0, 8, true}, {0, 12, true}, {12, 8, false}, {12, 12, true}} {
		st := seen
		st.Blocks = c.blocks
		v := judgeAt(t, rec, owed, StopInput{StopHookActive: true, Cap: c.cap}, st, now)
		if !v.Block {
			t.Fatalf("cap %d after %d blocks: allowed", c.cap, c.blocks)
		}
		if got := v.Stop != nil; got != c.record {
			t.Errorf("cap %d after %d blocks: stop recorded %v, want %v", c.cap, c.blocks, got, c.record)
			continue
		}
		if c.record && (v.Stop.Kind != StopWorkOwed || !v.Stop.PossiblyForcedByCap || !strings.Contains(v.Stop.Detail, "feat/behind")) {
			t.Errorf("cap %d: recorded %+v", c.cap, v.Stop)
		}
		if c.record && v.State.State != StateStopped {
			t.Errorf("cap %d: state %q after the cap, want stopped", c.cap, v.State.State)
		}
	}
	none := Owed{}
	st := SessionState{Fingerprint: none.Fingerprint(), WakeFingerprint: none.CappedFingerprint(), Blocks: 8}
	v := judgeAt(t, rec, none, StopInput{StopHookActive: true}, st, now)
	if v.Row != 7 || v.Stop == nil || v.Stop.Kind != StopNoNextLook || !v.Stop.PossiblyForcedByCap {
		t.Errorf("row 7 past the cap: row %d stop %+v; want stopped with no next look", v.Row, v.Stop)
	}
}

// TestCountResetsOnlyOnInactive: a change in the owed set at the sixth block
// does not reset the count, and the ninth consecutive block still records the
// stop; a block with stop_hook_active false sets it to one.
func TestCountResetsOnlyOnInactive(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	s := newSequence(t, awayRec(now, false), now)
	var v Verdict
	for i := 1; i <= 9; i++ {
		owed := owedOf(behindPR)
		if i >= 6 {
			owed = owedOf(behindPR, renewalDue)
		}
		v = s.stop(owed, StopInput{StopHookActive: i > 1})
		if !v.Block || v.State.Blocks != i {
			t.Fatalf("block %d: block %v count %d", i, v.Block, v.State.Blocks)
		}
		if (v.Stop != nil) != (i == 9) {
			t.Fatalf("block %d: stop recorded %v, want only at the ninth", i, v.Stop != nil)
		}
	}
	v = s.stop(owedOf(behindPR, renewalDue), StopInput{StopHookActive: false})
	if v.State.Blocks != 1 {
		t.Errorf("a block with stop_hook_active false leaves the count at %d, want 1", v.State.Blocks)
	}
}

// TestPushItemReasonDefersToStandingInstruction: without --hold-pushes the
// reason for an unpushed branch says to push if the standing instruction
// allows, to queue only if the person has said pushes wait, and that a queued
// push stays owed; with it, to finish the local work and then queue; neither
// names the away mode as permission.
func TestPushItemReasonDefersToStandingInstruction(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	owed := owedOf(unpushedBr)
	v := judgeAt(t, awayRec(now, false), owed, StopInput{}, SessionState{}, now)
	for _, want := range []string{"feat/unpushed", "push if your standing instruction allows", "queue only if the person has said pushes wait", "stays owed", "abcd mode queue"} {
		if !strings.Contains(v.Reason, want) {
			t.Errorf("without --hold-pushes the reason misses %q: %q", want, v.Reason)
		}
	}
	h := judgeAt(t, awayRec(now, true), owed, StopInput{}, SessionState{}, now)
	for _, want := range []string{"feat/unpushed", "the person has said pushes wait", "finish the local work", "then queue the push"} {
		if !strings.Contains(h.Reason, want) {
			t.Errorf("with --hold-pushes the reason misses %q: %q", want, h.Reason)
		}
	}
	for _, r := range []string{v.Reason, h.Reason} {
		low := strings.ToLower(r)
		for _, bad := range []string{"away mode allows", "away mode permits", "allowed by the away mode", "you may push"} {
			if strings.Contains(low, bad) {
				t.Errorf("the reason names the away mode as permission (%q): %q", bad, r)
			}
		}
	}
}

// TestJudgeHandBackOnlyUnderHoldPushes: the verdict accepts a hand-back only
// of an unpushed branch's item, with its head, under --hold-pushes; any other
// item handed back stays owed (F1, dcn-2610181035355741).
func TestJudgeHandBackOnlyUnderHoldPushes(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	wake := StopInput{SessionCrons: crons("*/10 * * * *")}
	handed := Owed{HandedBack: []OwedItem{unpushedBr}}
	v := judgeAt(t, awayRec(now, true), handed, wake, SessionState{}, now)
	if len(v.HandBacks) != 1 || v.HandBacks[0].Branch != "feat/unpushed" || v.Row != 6 {
		t.Errorf("with --hold-pushes an unpushed item handed back: hand-backs %v row %d, want accepted and nothing owed", v.HandBacks, v.Row)
	}
	v = judgeAt(t, awayRec(now, false), handed, wake, SessionState{}, now)
	if len(v.HandBacks) != 0 || !v.Block || !strings.Contains(v.Reason, "feat/unpushed") {
		t.Errorf("without --hold-pushes: hand-backs %v block %v; want none accepted and the item still owed", v.HandBacks, v.Block)
	}
	for _, item := range []OwedItem{behindPR, failingPR, stackedBr, uncarried, renewalDue, {Kind: OwedUnpushed, Capped: true, Branch: "feat/nohead"}} {
		v := judgeAt(t, awayRec(now, true), Owed{HandedBack: []OwedItem{item}}, wake, SessionState{}, now)
		if len(v.HandBacks) != 0 || !v.Block {
			t.Errorf("%s %s handed back under --hold-pushes: accepted %v block %v; want it kept owed", item.Kind, item.Branch, v.HandBacks, v.Block)
		}
	}
}

// TestDeadlineCeiling: with work owed and no reservation, now plus 30
// minutes; with nothing owed, now plus one hour; with a renewal due sooner,
// that minus the skew.
func TestDeadlineCeiling(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	if got := Deadline(owedOf(behindPR), now); !got.Equal(now.Add(30 * time.Minute)) {
		t.Errorf("owed, no reservation: %s", got)
	}
	if got := Deadline(Owed{}, now); !got.Equal(now.Add(time.Hour)) {
		t.Errorf("nothing owed: %s", got)
	}
	soon := Owed{NextRenewal: now.Add(20 * time.Minute)}
	if got := Deadline(soon, now); !got.Equal(now.Add(15 * time.Minute)) {
		t.Errorf("renewal in 20 minutes: deadline %s, want 15 minutes on", got)
	}
	late := Owed{Items: []OwedItem{behindPR}, NextRenewal: now.Add(12 * time.Hour)}
	if got := Deadline(late, now); !got.Equal(now.Add(30 * time.Minute)) {
		t.Errorf("renewal in 12 hours: deadline %s, want the ceiling", got)
	}
}

// TestQualifyingWakeDistantCron: a release reservation with a 24-hour lease
// claimed at 20:00, and `0 7 * * *` seen at 21:00: does not qualify.
func TestQualifyingWakeDistantCron(t *testing.T) {
	loc := london(t)
	claimed := at(t, "2026-10-18 20:00")
	now := at(t, "2026-10-18 21:00")
	owed := Owed{Items: []OwedItem{uncarried}, NextRenewal: claimed.Add(12 * time.Hour)}
	if _, _, ok := QualifyingWake(crons("0 7 * * *"), now, Deadline(owed, now), loc); ok {
		t.Error("0 7 * * * seen at 21:00 qualified with work owed")
	}
	if _, _, ok := QualifyingWake(crons("0 7 * * *"), now, Deadline(Owed{NextRenewal: owed.NextRenewal}, now), loc); ok {
		t.Error("0 7 * * * seen at 21:00 qualified with nothing owed")
	}
}

// TestQualifyingWakeJitter: an hourly cron whose next match is 3 minutes
// before the deadline does not qualify; `*/5 * * * *` and `*/10 * * * *` do;
// a one-shot at the deadline does.
func TestQualifyingWakeJitter(t *testing.T) {
	loc := london(t)
	now := at(t, "2026-10-18 21:00")
	deadline := Deadline(owedOf(behindPR), now) // 21:30
	if _, _, ok := QualifyingWake(crons("27 * * * *"), now, deadline, loc); ok {
		t.Error("an hourly cron at 21:27 qualified for a 21:30 deadline; it may run 30 minutes late")
	}
	for _, e := range []string{"*/5 * * * *", "*/10 * * * *"} {
		if _, _, ok := QualifyingWake(crons(e), now, deadline, loc); !ok {
			t.Errorf("%q did not qualify", e)
		}
	}
	oneShot := []SessionCron{{ID: "once", Schedule: "30 21 18 10 *", Recurring: false}}
	if _, look, ok := QualifyingWake(oneShot, now, deadline, loc); !ok || !look.Equal(deadline) {
		t.Errorf("a one-shot at the deadline: ok %v look %s", ok, look)
	}
	// */5 runs on time, any other every-five-minutes form gets half its gap.
	_, lateFive, _ := LatestFire(SessionCron{Schedule: "*/5 * * * *", Recurring: true}, now, deadline, loc)
	if !lateFive.Equal(now.Add(5 * time.Minute)) {
		t.Errorf("*/5 latest fire %s, want its match", lateFive)
	}
	_, lateTen, _ := LatestFire(SessionCron{Schedule: "*/10 * * * *", Recurring: true}, now, deadline, loc)
	if !lateTen.Equal(now.Add(15 * time.Minute)) {
		t.Errorf("*/10 latest fire %s, want its match plus five minutes", lateTen)
	}
}

// TestQualifyingWakeRecurringCron: `*/30 * * * *` with nothing owed does.
func TestQualifyingWakeRecurringCron(t *testing.T) {
	loc := london(t)
	now := at(t, "2026-10-18 21:05")
	if _, look, ok := QualifyingWake(crons("*/30 * * * *"), now, Deadline(Owed{}, now), loc); !ok || !look.Equal(at(t, "2026-10-18 21:30")) {
		t.Errorf("*/30 with nothing owed: ok %v look %s", ok, look)
	}
}

// TestQualifyingWakeAfterLeaseLapse: a wake past the earliest renewal due
// minus the skew does not.
func TestQualifyingWakeAfterLeaseLapse(t *testing.T) {
	loc := london(t)
	now := at(t, "2026-10-18 21:00")
	owed := Owed{Items: []OwedItem{renewalDue}, NextRenewal: now.Add(12 * time.Minute)} // deadline 21:07
	if _, _, ok := QualifyingWake([]SessionCron{{Schedule: "10 21 * * *"}}, now, Deadline(owed, now), loc); ok {
		t.Error("a one-shot at 21:10 qualified past a renewal due at 21:12 less the skew")
	}
	if _, _, ok := QualifyingWake([]SessionCron{{Schedule: "6 21 * * *"}}, now, Deadline(owed, now), loc); !ok {
		t.Error("a one-shot at 21:06 did not qualify before 21:07")
	}
}

// TestBackgroundTaskNotAWake: a shell task running `sleep 36000` or a
// `tail -f` does not count.
func TestBackgroundTaskNotAWake(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	owed := owedOf(behindPR)
	st := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint()}
	in := StopInput{BackgroundTasks: []BackgroundTask{
		{ID: "t1", Type: "shell", Status: "running", Command: "sleep 36000"},
		{ID: "t2", Type: "shell", Status: "running", Command: "tail -f /var/log/syslog"},
	}}
	if v := judgeAt(t, awayRec(now, false), owed, in, st, now); !v.Block || v.Row != 5 {
		t.Errorf("background tasks counted as a wake: row %d block %v", v.Row, v.Block)
	}
	none := Owed{}
	st = SessionState{Fingerprint: none.Fingerprint(), WakeFingerprint: none.CappedFingerprint()}
	if v := judgeAt(t, awayRec(now, false), none, in, st, now); !v.Block || v.Row != 7 {
		t.Errorf("background tasks counted as a wake with nothing owed: row %d", v.Row)
	}
}

// TestNextLookStored: a waiting verdict keeps the wake's next fire time as
// the next look.
func TestNextLookStored(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	s := newSequence(t, awayRec(now, false), now)
	none := Owed{}
	v := s.stop(none, StopInput{SessionCrons: crons("*/10 * * * *")})
	if v.Block || v.State.State != StateWaiting || !v.State.NextLook.Equal(at(t, "2026-10-18 21:10")) {
		t.Fatalf("waiting verdict: %+v", v.State)
	}
	st, _ := LoadSession(s.root, s.id)
	if !st.NextLook.Equal(at(t, "2026-10-18 21:10")) || st.State != StateWaiting {
		t.Errorf("state file: next look %s state %q", st.NextLook, st.State)
	}
}

// TestProseQuestionBlockedOnce: a last message ending on a question is
// blocked once; the same message again goes on to the other rows.
func TestProseQuestionBlockedOnce(t *testing.T) {
	now := at(t, "2026-10-18 21:01")
	s := newSequence(t, awayRec(now, false), now)
	in := StopInput{LastAssistantMessage: "Should I rebase onto main?  \n\n", SessionCrons: crons("*/10 * * * *")}
	if v := s.stop(Owed{}, in); v.Row != 1 || !v.Block {
		t.Fatalf("first: row %d", v.Row)
	}
	in.StopHookActive = true
	if v := s.stop(Owed{}, in); v.Row == 1 {
		t.Errorf("the same question was blocked twice")
	}
	in.LastAssistantMessage = "I merged it. Is CI green? I'll check.\n"
	if v := s.stop(Owed{}, in); v.Row == 1 {
		t.Errorf("a message not ending on a question line was blocked as one")
	}
	in.LastAssistantMessage = "Next question:\nwhich base should it use?"
	if v := s.stop(Owed{}, in); v.Row != 1 {
		t.Errorf("a new question was not blocked: row %d", v.Row)
	}
}

// TestNothingOwedReason names taking the next ready item first.
func TestNothingOwedReason(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	v := judgeAt(t, awayRec(now, false), Owed{}, StopInput{}, SessionState{}, now)
	if v.Row != 7 || !strings.Contains(v.Reason, "take the next ready item first") || !strings.Contains(v.Reason, "abcd reserve") {
		t.Errorf("row %d reason %q", v.Row, v.Reason)
	}
	if strings.Index(v.Reason, "next ready item") > strings.Index(v.Reason, "schedule a look") {
		t.Errorf("the reason offers the look before the next item: %q", v.Reason)
	}
}

// TestSessionIDShape: a session id outside [A-Za-z0-9_-]{1,128} is refused.
func TestSessionIDShape(t *testing.T) {
	root := t.TempDir()
	for _, ok := range []string{"a", "abc-DEF_123", strings.Repeat("x", 128), "0b9c1d2e-aaaa-bbbb-cccc-0123456789ab"} {
		if !ValidSessionID(ok) {
			t.Errorf("%q refused", ok)
		}
		if err := SaveSession(root, ok, SessionState{State: StateWorking}); err != nil {
			t.Errorf("SaveSession(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "../x", "a/b", "a.b", strings.Repeat("x", 129), "a b", "x\n"} {
		if ValidSessionID(bad) {
			t.Errorf("%q accepted", bad)
		}
		if err := SaveSession(root, bad, SessionState{}); err == nil {
			t.Errorf("SaveSession(%q) wrote", bad)
		}
		if _, err := LoadSession(root, bad); err == nil {
			t.Errorf("LoadSession(%q) read", bad)
		}
	}
	p, _ := SessionStatePath(root, "abc")
	if want := filepath.Join(root, ".abcd", ".work.local", "autonomy", "sessions", "abc.json"); p != want {
		t.Errorf("state path %s, want %s", p, want)
	}
}

// TestReasonSaysItOnce: an item's words appear once, in the agent's reason,
// the person's line and the recorded stop alike, with the reader's reason as
// it writes it.
func TestReasonSaysItOnce(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	item := OwedItem{Kind: OwedUnpushed, Capped: true, Branch: "feat/x", Head: "cccc", Reason: "branch feat/x has commits not pushed"}
	owed := owedOf(item)
	if got, want := presentLine(owed), "abcd: stopped with work owed (1 item); next: branch feat/x has commits not pushed."; got != want {
		t.Errorf("the person's line:\n got %q\nwant %q", got, want)
	}
	v := judgeAt(t, awayRec(now, false), owed, StopInput{}, SessionState{}, now)
	if want := "abcd: the person is away and work is owed (1 item). Next: branch feat/x has commits not pushed. Do the local work"; !strings.HasPrefix(v.Reason, want) {
		t.Errorf("the reason:\n got %q\nwant it to begin %q", v.Reason, want)
	}
	seen := SessionState{Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint(), Blocks: DefaultBlockCap}
	v = judgeAt(t, awayRec(now, false), owed, StopInput{StopHookActive: true}, seen, now)
	if v.Stop == nil || v.Stop.Detail != "branch feat/x has commits not pushed" {
		t.Errorf("the recorded stop: %+v", v.Stop)
	}
}

// TestSuggestedQueueCommandQuotesBranch: a branch name holding `$(…)`, a
// backquote or a single quote, all legal in a refname, reaches the suggested
// `abcd mode queue` command as one single-quoted shell word.
func TestSuggestedQueueCommandQuotesBranch(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	for branch, want := range map[string]string{
		"feat/x":                   "`abcd mode queue 'push feat/x'`",
		"feat/$(touch${IFS}pwned)": "`abcd mode queue 'push feat/$(touch${IFS}pwned)'`",
		"feat/`id`":                "`abcd mode queue 'push feat/`id`'`",
		"feat/it's":                `abcd mode queue 'push feat/it'\''s'`,
	} {
		item := OwedItem{Kind: OwedUnpushed, Capped: true, Branch: branch, Head: "cccc"}
		v := judgeAt(t, awayRec(now, true), owedOf(item), StopInput{}, SessionState{}, now)
		if !strings.Contains(v.Reason, want) {
			t.Errorf("branch %q: the reason lacks %s:\n%s", branch, want, v.Reason)
		}
	}
}

// TestRowOnePastCapRecordsStop: a question blocked once more past the cap
// records the stop: work owed when anything is, else no next look.
func TestRowOnePastCapRecordsStop(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	rec := awayRec(now, false)
	for _, c := range []struct {
		owed Owed
		kind StopKind
	}{{owedOf(behindPR), StopWorkOwed}, {Owed{}, StopNoNextLook}} {
		st := SessionState{Fingerprint: c.owed.Fingerprint(), WakeFingerprint: c.owed.CappedFingerprint(), Blocks: DefaultBlockCap}
		v := judgeAt(t, rec, c.owed, StopInput{StopHookActive: true, LastAssistantMessage: "Shall I merge it?"}, st, now)
		if v.Row != 1 || !v.Block {
			t.Fatalf("row %d block %v, want row 1 blocked", v.Row, v.Block)
		}
		if v.Stop == nil || v.Stop.Kind != c.kind || !v.Stop.PossiblyForcedByCap || v.State.State != StateStopped {
			t.Errorf("row 1 past the cap with %d owed: stop %+v state %q, want %s", len(c.owed.Items), v.Stop, v.State.State, c.kind)
		}
	}
}

// TestIncompleteReadAllows: a reading the time budget cut short allows the
// stop, present or away, naming what was not read to the person; while away
// it records the stop.
func TestIncompleteReadAllows(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	owed := Owed{Items: []OwedItem{behindPR}, Incomplete: true, Notes: []string{"the check ran out of its time budget before reading 3 of 9 own branches"}}
	for _, away := range []bool{true, false} {
		var rec *Record
		if away {
			rec = awayRec(now, false)
		}
		v := judgeAt(t, rec, owed, StopInput{}, SessionState{}, now)
		if v.Block || !strings.Contains(v.SystemMessage, "3 of 9 own branches") {
			t.Errorf("away %v: block %v message %q, want allowed and named", away, v.Block, v.SystemMessage)
		}
		if got := v.Stop != nil && v.Stop.Kind == StopCheckIncomplete; got != away {
			t.Errorf("away %v: stop %+v", away, v.Stop)
		}
	}
}
