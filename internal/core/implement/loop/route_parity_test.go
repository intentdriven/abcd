package loop

// route_parity_test.go holds `implement step --route`'s prediction to the
// move it predicts (iss-2610161801358637): predictHand reads, from the state
// alone, the role the step hands an agent, and move makes the step. The two
// share binaryMoves, mayWait and chooseAgent but run their own control flow,
// so these tests run both on the same state and compare what they say.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
)

// parityLand is how a fake landing body moves a lane: it waits on its full
// check (a contention refusal, as the landing's preflight wait is), or it
// takes a step of the landing and stays at it.
type parityLand int

const (
	landWaits parityLand = iota
	landMoves
)

// paritySteps is a lane sequence with no git and no forge: the worktree and
// the brief complete, the implement stage hands out its implementer, the
// validate stage chooses whom it hands the lane to as validateStage does, and
// the landing waits or moves as lands says of the lane.
func paritySteps(lands map[string]parityLand) Stages {
	done := func(Context, *Lane) (Outcome, error) { return Outcome{Note: "done"}, nil }
	return Stages{
		{Name: StageWorktree, Piece: 6, Run: done},
		{Name: StageBrief, Piece: 5, Run: done},
		{Name: StageImplement, Piece: 7, Run: func(c Context, l *Lane) (Outcome, error) {
			return Outcome{Await: &Await{Role: RoleImplementer, Brief: "brief-" + l.ID, Receipt: "receipt-" + l.ID}}, nil
		}},
		{Name: StageValidate, Piece: 8, Repeats: true, Run: func(c Context, l *Lane) (Outcome, error) {
			// validateStage's choice, without the round's files: a conflicting
			// sync's implementer, a fresh round's first validator, the next
			// validator not out, the fix implementer of a round that did not
			// pass, or the stage complete.
			if s := l.pendingSync(); s != nil {
				return Outcome{Await: &Await{Role: RoleImplementer, Brief: s.Brief, Receipt: s.Receipt}}, nil
			}
			n := len(l.Validation)
			if n == 0 || l.Validation[n-1].Fix != "" || l.syncRoundDue() {
				l.Validation = append(slices.Clone(l.Validation), ValidationRound{Round: n + 1, Validators: []ValidatorRun{
					{Role: RoleRuthless, Return: "ruthless-" + l.ID}, {Role: RoleSecurity, Return: "security-" + l.ID}}})
				n++
			}
			cur := l.Validation[n-1]
			for _, v := range cur.Validators {
				if v.Verdict == "" && !l.awaited(v.Return) {
					return Outcome{Await: &Await{Role: v.Role, Brief: "brief-" + v.Role, Receipt: v.Return}}, nil
				}
			}
			for _, v := range cur.Validators {
				if v.Verdict == "" {
					return Outcome{}, contend(string(StageValidate), "", l.ID, "the validators are all out", "hand back their returns")
				}
			}
			for _, v := range cur.Validators {
				if !v.Pass {
					if l.fixRoundsTaken() >= c.State.FixRoundCap() {
						return Outcome{HandBack: &HandBack{Verdict: VerdictUnachievable}}, nil
					}
					return Outcome{Await: &Await{Role: RoleImplementer, Brief: "fix-" + l.ID, Receipt: "fix-receipt-" + l.ID}}, nil
				}
			}
			return Outcome{Note: "round passed"}, nil
		}},
		{Name: StageLand, Piece: 9, Run: func(c Context, l *Lane) (Outcome, error) {
			if lands[l.ID] == landMoves {
				return Outcome{Stay: true, Note: "a landing step"}, nil
			}
			return Outcome{}, contend(string(StageLand), "", l.ID, "waiting for its full check", "mint the preflight receipt")
		}},
	}
}

// subAgentPace is a pace whose ceiling is n agents.
func subAgentPace(n int) *Pace { return &Pace{SubAgents: PaceValue{Value: n}} }

// cloneState copies the parts of the state move replaces in place, so the
// move on the copy leaves the original for the prediction.
func cloneState(st State) State {
	c := st
	c.Lanes = slices.Clone(st.Lanes)
	c.Pending = slices.Clone(st.Pending)
	c.Waiting = slices.Clone(st.Waiting)
	c.Record = slices.Clone(st.Record)
	return c
}

// assertAgrees holds the move to the prediction: a move that hands an agent
// hands the predicted role on the predicted lane; a move that hands none was
// predicted to hand none, or was the landing the prediction named as moving
// first (the one case the state cannot settle, left to noteUnusedRoute).
func assertAgrees(t *testing.T, h stepHand, res StepResult, err error) {
	t.Helper()
	if err == nil && res.handed {
		if res.Awaiting == nil || res.Awaiting.Role != h.role || res.Lane != h.lane {
			t.Fatalf("the move handed %s's %+v; the prediction said %+v", res.Lane, res.Awaiting, h)
		}
		return
	}
	if h.role == "" {
		return
	}
	if h.landing == "" || err != nil || res.Lane != h.landing {
		t.Fatalf("the move handed no agent (%+v, %v); the prediction said %s's %s with no landing ahead of it: %+v", res, err, h.lane, h.role, h)
	}
}

// TestPredictHandAgreesWithTheMove runs predictHand and move on the same
// state, case by case, and holds the move to the prediction. Each case also
// pins what the move does, so a case that stops reaching its scenario fails
// rather than agreeing vacuously.
func TestPredictHandAgreesWithTheMove(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	const run = "run-0000000000000001"
	out := func(role, id string) []Await { return []Await{{Role: role, Receipt: role + "-" + id}} }
	round := func(vs ...ValidatorRun) []ValidationRound { return []ValidationRound{{Round: 1, Validators: vs}} }
	pass := func(role string) ValidatorRun {
		return ValidatorRun{Role: role, Return: role + "-ret", Verdict: "SHIP", Pass: true}
	}
	open := func(role string) ValidatorRun { return ValidatorRun{Role: role, Return: role + "-ret"} }
	quiet := &outageGate{now: now}
	netDown := &implement.Outage{Status: implement.OutageOpen, Services: []string{implement.ServiceNetwork},
		Down: []string{implement.ServiceNetwork}, NextProbeAt: now.Add(time.Hour)}
	runnerRouted := &outageGate{now: now, cur: netDown, runner: func(string) bool { return true }}
	hostStarted := &outageGate{now: now, cur: netDown, runner: func(string) bool { return false }}

	cases := []struct {
		name  string
		st    State
		g     *outageGate
		lands map[string]parityLand
		// role is the role the move hands out, "" for none; wait is whether
		// the move ends in a contention refusal.
		role string
		wait bool
		// says is a phrase the prediction's reason for handing none carries.
		says string
	}{
		{name: "a fresh lane at its worktree", st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageWorktree}}}},
		{name: "a lane at its brief", st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageBrief}}}},
		{name: "the implement stage", role: RoleImplementer,
			st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageImplement}}}},
		{name: "a fresh round's first validator", role: RoleRuthless,
			st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageValidate}}}},
		{name: "the next validator while the first is out", role: RoleSecurity, st: State{Pace: subAgentPace(2), Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageValidate, Awaits: []Await{{Role: RoleRuthless, Receipt: RoleRuthless + "-ret"}},
			Validation: round(open(RoleRuthless), open(RoleSecurity))}}}},
		{name: "the validators all out", st: State{Pace: subAgentPace(3), Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageValidate,
			Awaits:     []Await{{Role: RoleRuthless, Receipt: RoleRuthless + "-ret"}, {Role: RoleSecurity, Receipt: RoleSecurity + "-ret"}},
			Validation: round(open(RoleRuthless), open(RoleSecurity))}}}},
		{name: "a round that did not pass goes to a fix implementer", role: RoleImplementer, st: State{Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageValidate,
			Validation: round(pass(RoleRuthless), ValidatorRun{Role: RoleSecurity, Return: RoleSecurity + "-ret", Verdict: "REJECT"})}}}},
		{name: "a passed round completes the stage", st: State{Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageValidate, Validation: round(pass(RoleRuthless), pass(RoleSecurity))}}}},
		{name: "a conflicting sync goes to an implementer", role: RoleImplementer, st: State{Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageValidate, Validation: round(pass(RoleRuthless), pass(RoleSecurity)),
			Syncs: []Sync{{Conflicted: true, Brief: "sync-brief", Receipt: "sync-receipt"}}}}}},
		{name: "a re-told await", says: "already out", st: State{Lanes: []Lane{{
			ID: "lane-1", SpecStep: 1, Stage: StageImplement, Awaits: out(RoleImplementer, "lane-1")}}}},
		{name: "the agent ceiling reached", says: "ceiling of 1", st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageValidate, Awaits: []Await{{Role: RoleRuthless, Receipt: RoleRuthless + "-ret"}},
				Validation: round(open(RoleRuthless), open(RoleSecurity))},
			{ID: "lane-2", SpecStep: 2, Stage: StageImplement}}}},
		{name: "the network outage holds a runner-routed role", g: runnerRouted, wait: true, says: "outage",
			st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageImplement}}}},
		{name: "the network outage leaves a host-started role", g: hostStarted, role: RoleImplementer,
			st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageImplement}}}},
		{name: "the network outage holds a landing and the role behind it", g: runnerRouted, wait: true, st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageLand, Landing: &Landing{RecordsDone: true}},
			{ID: "lane-2", SpecStep: 2, Stage: StageImplement}}}},
		{name: "a ready step opens a lane while a helper is free", says: "opens a lane", st: State{Pace: subAgentPace(2),
			Lanes:   []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageImplement, Awaits: out(RoleImplementer, "lane-1")}},
			Pending: []PendingStep{{Number: 2, Title: "Two", Needs: []int{}}}}},
		{name: "no free helper for a ready step", g: hostStarted, says: "no helper is free", st: State{Pace: subAgentPace(2),
			Lanes: []Lane{
				{ID: "lane-1", SpecStep: 1, Stage: StageLand, Landing: &Landing{RecordsDone: true}},
				{ID: "lane-2", SpecStep: 2, Stage: StageLand}},
			Pending: []PendingStep{{Number: 3, Title: "Three", Needs: []int{}}}}},
		{name: "no free helper behind a landing that waits on its check", says: "no helper is free", st: State{Pace: subAgentPace(2),
			Lanes: []Lane{
				{ID: "lane-1", SpecStep: 1, Stage: StageLand, Landing: &Landing{RecordsDone: true}},
				{ID: "lane-2", SpecStep: 2, Stage: StageLand}},
			Pending: []PendingStep{{Number: 3, Title: "Three", Needs: []int{}}}}},
		{name: "a landing waiting on its check, then an implementer", role: RoleImplementer, st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageLand},
			{ID: "lane-2", SpecStep: 2, Stage: StageImplement}}}},
		{name: "a landing that moves ahead of an implementer", lands: map[string]parityLand{"lane-1": landMoves}, st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageLand},
			{ID: "lane-2", SpecStep: 2, Stage: StageImplement}}}},
		{name: "a landing waiting on its check, then a validate the binary performs", st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageLand},
			{ID: "lane-2", SpecStep: 2, Stage: StageValidate, Validation: round(pass(RoleRuthless), pass(RoleSecurity))}}}},
		{name: "a landing that moves ahead of a validate the binary performs", lands: map[string]parityLand{"lane-1": landMoves}, st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageLand},
			{ID: "lane-2", SpecStep: 2, Stage: StageValidate, Validation: round(pass(RoleRuthless), pass(RoleSecurity))}}}},
		{name: "a lone landing waiting on its check", wait: true, st: State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: StageLand}}}},
		{name: "a finished run", says: "is done", st: State{Lanes: []Lane{
			{ID: "lane-1", SpecStep: 1, Stage: StageDone}, {ID: "lane-2", SpecStep: 2, Stage: StageDone}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.st.RunID = run
			g := tc.g
			if g == nil {
				g = quiet
			}
			h := predictHand(tc.st, g, now)
			if tc.says != "" && !strings.Contains(h.none, tc.says) {
				t.Fatalf("the prediction says %q, not %q: %+v", h.none, tc.says, h)
			}
			moving := cloneState(tc.st)
			res, _, err := move("", &moving, paritySteps(tc.lands), now, g)
			if err != nil {
				if r, ok := AsRefusal(err); !ok || !r.Contention || !tc.wait {
					t.Fatalf("the move refused: %v", err)
				}
			} else if tc.wait {
				t.Fatalf("the move ends waiting: %+v", res)
			}
			handed := ""
			if err == nil && res.handed && res.Awaiting != nil {
				handed = res.Awaiting.Role
			}
			if handed != tc.role {
				t.Fatalf("the move handed %q, the case is of %q: %+v", handed, tc.role, res)
			}
			assertAgrees(t, h, res, err)
		})
	}
}

// TestPredictHandAgreesWithEveryMoveOfARealRun walks a whole two-lane run
// through the lane sequence this build carries, over the stub forge, and
// holds every step's move to the prediction made from the state before it.
// Each agent the run hands out is answered only once a step hands none, so
// the walk meets re-told awaits and the ceiling; a lane's first push waits on
// its full check, so a landing waits while the other lane moves.
func TestPredictHandAgreesWithEveryMoveOfARealRun(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("2")})
	pass := map[string]string{RoleRuthless: "SHIP", RoleSecurity: "APPROVE", RoleAuditor: "MET"}
	handed := map[string]int{}
	var none, waits int
	cleared := map[string]bool{}
	for range 150 {
		st := f.state(t)
		if st.Complete() {
			break
		}
		for _, l := range st.Lanes {
			if l.Stage == StageDone && !cleared[l.ID] {
				// The stub forge's pull request is reset once a lane lands.
				for _, name := range []string{"pr.json", "state"} {
					_ = os.Remove(filepath.Join(f.gh, name))
				}
				cleared[l.ID] = true
			}
			if l.Stage != StageLand || l.Landing == nil {
				continue
			}
			if l.Landing.RecordsDone && l.Landing.Pushed == "" && l.Landing.CheckWaitSince != nil {
				preflighted(t, l, l.HeadSHA)
			}
			if l.Landing.Merge != "" {
				f.repo.Git("push", "-q", "origin", l.Landing.Pushed+":refs/heads/main")
				if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("MERGED\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		st = f.state(t)
		h := predictHand(st, &outageGate{now: f.now}, f.now)
		res, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
		if err != nil {
			if r, ok := AsRefusal(err); !ok || !r.Contention {
				t.Fatalf("step: %v", err)
			}
			waits++
		}
		assertAgrees(t, h, res, err)
		if err == nil && res.handed {
			handed[res.Awaiting.Role]++
			continue
		}
		none++
		// The step handed none: every agent out hands back.
		for _, l := range f.state(t).Lanes {
			for _, a := range l.Awaits {
				if a.Role == RoleImplementer {
					f.receipt(t, l.ID, a, laneCommit(t, f.repo, l, l.ID+".txt"))
				} else {
					f.ret(t, l.ID, a.Role, pass[a.Role])
				}
			}
		}
	}
	if !f.state(t).Complete() {
		t.Fatalf("the run never completed: %+v", f.state(t).Lanes)
	}
	// The walk reached what it is for: both lanes' implementers, the
	// reviewers, the closing lane's auditor, steps that handed none, and a
	// landing that waited.
	if handed[RoleImplementer] < 2 || handed[RoleRuthless] < 2 || handed[RoleSecurity] < 2 || handed[RoleAuditor] < 1 || none == 0 || waits == 0 {
		t.Fatalf("the walk missed a move: handed %v, %d handing none, %d waits", handed, none, waits)
	}
}

// TestOnlyALandingMayWait pins mayWait: a move the binary performs at any
// stage but the landing that ends in a contention refusal stops the call,
// and no later lane moves, so the prediction, which reads such a move as one
// that runs, cannot be wrong about it. A change that lets another stage wait
// fails here first.
func TestOnlyALandingMayWait(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	for _, stage := range []Stage{StageWorktree, StageBrief, StageValidate} {
		t.Run(string(stage), func(t *testing.T) {
			st := State{RunID: "run-0000000000000001", Lanes: []Lane{
				{ID: "lane-1", SpecStep: 1, Stage: stage},
				{ID: "lane-2", SpecStep: 2, Stage: StageImplement}}}
			if stage == StageValidate {
				st.Lanes[0].Validation = []ValidationRound{{Round: 1, Validators: []ValidatorRun{
					{Role: RoleRuthless, Return: "r", Verdict: "SHIP", Pass: true}}}}
			}
			ws := binaryMoves(st)
			if len(ws) != 1 || ws[0].lane != 0 || mayWait(st, ws[0]) {
				t.Fatalf("lane 1's %s is a binary move that may not wait: %+v", stage, ws)
			}
			h := predictHand(st, &outageGate{now: now}, now)
			if h.role != "" || h.landing != "" {
				t.Fatalf("the prediction reads lane 1's %s as the move: %+v", stage, h)
			}
			steps := paritySteps(nil)
			for k := range steps {
				if steps[k].Name == stage {
					steps[k].Run = func(c Context, l *Lane) (Outcome, error) {
						return Outcome{}, contend(string(stage), "", l.ID, "held by someone else", "step again")
					}
				}
			}
			moving := cloneState(st)
			_, _, err := move("", &moving, steps, now, &outageGate{now: now})
			r := mustRefusal(t, err)
			if !r.Contention || r.Lane != "lane-1" || r.Stage != string(stage) {
				t.Fatalf("lane 1's contended %s is the call's answer: %+v", stage, r)
			}
			if len(moving.Lanes[1].Awaits) != 0 {
				t.Fatalf("no later lane moves after a contended %s: %+v", stage, moving.Lanes[1])
			}
		})
	}

	// Every stage a lane can stand at: only the landing may wait.
	for _, stage := range append(slices.Clone(Sequence), StageDone, StageDiscarded, StageHandedBack) {
		st := State{Lanes: []Lane{{ID: "lane-1", SpecStep: 1, Stage: stage}}}
		if got := mayWait(st, want{lane: 0, kind: wantBinary, open: -1}); got != (stage == StageLand) {
			t.Fatalf("mayWait at %s = %v", stage, got)
		}
	}
}
