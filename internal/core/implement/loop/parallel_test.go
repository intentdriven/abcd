package loop

// parallel_test.go is criterion 6 of itd-2609201925079472 made concrete (ruling
// DR6, spc-2609202134341288 C1 to C13): a run works in parallel up to its
// ceiling, through the step interface, with fake agents writing the receipts
// and returns, the clock Options carries, a bare local remote and the stub
// forge client.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// parFixture is a run over a stepped spec on a repository whose default branch
// is on a local bare remote, with the stub forge first on PATH.
type parFixture struct {
	repo   *gittest.Repo
	gh     string
	runID  string
	stages Stages
	now    time.Time
}

func newParFixture(t *testing.T, steps string, o Options) *parFixture {
	t.Helper()
	repo := loopRepo(t, readyIntent("impact: additive\n", settledQuestions), specWithSteps(steps))
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	repo.Write("AGENTS.md", agentsMarked)
	repo.Write(".abcd/work/rulesets/main-protection.json", queueRuleset("MERGE"))
	repo.Commit("the record")
	bare := filepath.Join(t.TempDir(), "origin.git")
	repo.Git("init", "-q", "--bare", "--initial-branch=main", bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", gh+string(os.PathListSeparator)+os.Getenv("PATH"))
	f := &parFixture{repo: repo, gh: gh, stages: DefaultStages(), now: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)}
	o.Now = f.clock
	start, err := Start(repo.Root(), "itd-10", o)
	if err != nil {
		t.Fatal(err)
	}
	f.runID = start.RunID
	return f
}

func (f *parFixture) clock() time.Time { return f.now }
func (f *parFixture) opts() Options    { return Options{Now: f.clock} }

func (f *parFixture) state(t *testing.T) State {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func (f *parFixture) lane(t *testing.T, id string) Lane {
	t.Helper()
	for _, l := range f.state(t).Lanes {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("no lane %s", id)
	return Lane{}
}

func (f *parFixture) step(t *testing.T) StepResult {
	t.Helper()
	res, err := Advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	return res
}

// stepUntil steps until done reports true of the state, failing after a bound.
func (f *parFixture) stepUntil(t *testing.T, what string, done func(State) bool) StepResult {
	t.Helper()
	var res StepResult
	for range 40 {
		if done(f.state(t)) {
			return res
		}
		res = f.step(t)
	}
	t.Fatalf("never reached: %s; state %+v", what, f.state(t))
	return res
}

// await is the lane's outstanding await in role.
func (f *parFixture) await(t *testing.T, laneID, role string) Await {
	t.Helper()
	for _, a := range f.lane(t, laneID).Awaits {
		if a.Role == role {
			return a
		}
	}
	t.Fatalf("%s awaits no %s: %+v", laneID, role, f.lane(t, laneID).Awaits)
	return Await{}
}

func (f *parFixture) abs(rel string) string {
	return filepath.Join(f.repo.Root(), filepath.FromSlash(rel))
}

// implement hands back a verified receipt for the lane's implementer, with one
// commit of file in its worktree.
func (f *parFixture) implement(t *testing.T, laneID, file string) StepResult {
	t.Helper()
	a := f.await(t, laneID, RoleImplementer)
	return f.receipt(t, laneID, a, laneCommit(t, f.repo, f.lane(t, laneID), file))
}

// receipt writes an implementer's receipt at the await's path, its report and
// output beside it, naming commits, and hands it back.
func (f *parFixture) receipt(t *testing.T, laneID string, a Await, commits ...string) StepResult {
	t.Helper()
	l := f.lane(t, laneID)
	laneDir := f.abs(RunRelDir + "/" + f.runID + "/" + laneID)
	rel, err := filepath.Rel(laneDir, filepath.Dir(f.abs(a.Receipt)))
	if err != nil {
		t.Fatal(err)
	}
	rel = filepath.ToSlash(rel)
	prefix := ""
	if rel != "." {
		prefix = rel + "/"
	}
	for name, body := range map[string]string{prefix + ReportFileName: "done\n", prefix + DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: laneID, Branch: l.Branch, Commits: commits,
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: prefix + DoDFileName}, Report: prefix + ReportFileName, Model: "claude-test-5"}
	data, _ := json.Marshal(rc)
	if err := os.WriteFile(f.abs(a.Receipt), data, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Receipt(f.repo.Root(), f.runID, f.abs(a.Receipt), f.stages, f.opts())
	if err != nil {
		t.Fatalf("%s's receipt: %v", laneID, err)
	}
	return res
}

// ret hands back a validator's return in role for the lane.
func (f *parFixture) ret(t *testing.T, laneID, role, verdict string) StepResult {
	t.Helper()
	a := f.await(t, laneID, role)
	body := reviewerReturn(verdict)
	if role == RoleAuditor {
		body = auditorVerdict(t, filepath.Join(filepath.Dir(f.abs(a.Receipt)), AuditRequestFileName), verdict)
	}
	if err := os.WriteFile(f.abs(a.Receipt), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Receipt(f.repo.Root(), f.runID, f.abs(a.Receipt), f.stages, f.opts())
	if err != nil {
		t.Fatalf("%s's %s return: %v", laneID, role, err)
	}
	return res
}

// passAll hands back a passing return from every validator the lane has out.
func (f *parFixture) passAll(t *testing.T, laneID string) {
	t.Helper()
	pass := map[string]string{RoleRuthless: "SHIP", RoleSecurity: "APPROVE", RoleAuditor: "MET"}
	for _, a := range f.lane(t, laneID).Awaits {
		f.ret(t, laneID, a.Role, pass[a.Role])
	}
}

// roundPassed steps and hands back passing returns until the lane leaves its
// validate stage.
func (f *parFixture) roundPassed(t *testing.T, laneID string) {
	t.Helper()
	for range 20 {
		l := f.lane(t, laneID)
		if l.Stage != StageValidate {
			return
		}
		if len(l.Awaits) > 0 {
			f.passAll(t, laneID)
			continue
		}
		f.step(t)
	}
	t.Fatalf("%s's round never passed: %+v", laneID, f.lane(t, laneID))
}

func (f *parFixture) ghLog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.gh, "gh.log"))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// landed takes a lane at its landing through the stub forge: preflight
// receipts for each head it pushes, the merge queue's merge, and the ancestor
// check, until the lane is done. The stub's pull request is reset after.
func (f *parFixture) landed(t *testing.T, laneID string) {
	t.Helper()
	for range 20 {
		l := f.lane(t, laneID)
		if l.Stage == StageDone {
			for _, name := range []string{"pr.json", "state"} {
				_ = os.Remove(filepath.Join(f.gh, name))
			}
			return
		}
		if l.Stage != StageLand {
			t.Fatalf("%s left its landing for %s", laneID, l.Stage)
		}
		if l.Landing != nil && l.Landing.RecordsDone && l.Landing.Pushed == "" {
			preflighted(t, l, l.HeadSHA)
		}
		if l.Landing != nil && l.Landing.Merge != "" {
			f.repo.Git("push", "-q", "origin", l.Landing.Pushed+":refs/heads/main")
			if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("MERGED\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		f.step(t)
	}
	t.Fatalf("%s never landed", laneID)
}

// gitErr runs one git command in the fixture and returns its error, for a
// command expected to fail.
func gitErr(repo *gittest.Repo, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", repo.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture"}, args...)...)
	cmd.Env = repo.Env()
	return cmd.Run()
}

func recordText(st State) string {
	var b strings.Builder
	for _, e := range st.Record {
		b.WriteString(e.Lane + " " + e.Stage + " " + e.Note + "\n")
	}
	return b.String()
}

// C1, the count, and C2, the ceiling reached, and C3, the slot filled and the
// wait counted: a lane's validators await at once; at the ceiling a step hands
// out nothing, names every lane alive with the role and receipt it awaits, and
// holds the work in `waiting` with the time first held, unchanged by a second
// call; once a receipt frees a slot the held work takes it, and the record
// names the minutes it waited.
func TestTheCeilingCountsValidatorsAndHoldsTheWaitingWork(t *testing.T) {
	f := newParFixture(t, "", Options{SubAgents: strp("2")})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")

	// C1: two calls, two reviewers out at once.
	if r := f.step(t); r.Awaiting == nil || r.Awaiting.Role != RoleRuthless {
		t.Fatalf("the first call hands out the ruthless reviewer: %+v", r)
	}
	if r := f.step(t); r.Awaiting == nil || r.Awaiting.Role != RoleSecurity || r.Slots != 2 || r.Ceiling != 2 {
		t.Fatalf("the second call hands out the security reviewer beside it, 2 of 2 slots: %+v", r)
	}
	st := f.state(t)
	if aw := st.Lanes[0].Awaits; len(aw) != 2 || aw[0].Role != RoleRuthless || aw[1].Role != RoleSecurity {
		t.Fatalf("the state file carries two awaits on the lane: %+v", aw)
	}
	if st.SlotsInUse() != 2 || st.Ceiling() != 2 {
		t.Fatalf("status names 2 of 2 slots in use: %d of %d", st.SlotsInUse(), st.Ceiling())
	}

	// C2: the ceiling reached. The closing lane's auditor waits.
	held := f.now
	r := f.step(t)
	if !r.CeilingReached || r.Awaiting == nil || len(r.Alive) != 1 || len(r.Alive[0].Awaits) != 2 {
		t.Fatalf("at the ceiling the call hands out nothing and names the lanes alive with their awaits: %+v", r)
	}
	for _, a := range r.Alive[0].Awaits {
		if !strings.Contains(r.Next, a.Receipt) || !strings.Contains(r.Next, a.Role) {
			t.Fatalf("the next move names %s at %s: %s", a.Role, a.Receipt, r.Next)
		}
	}
	st = f.state(t)
	if len(st.Waiting) != 1 || st.Waiting[0].Lane != "lane-1" || st.Waiting[0].Role != RoleAuditor || !st.Waiting[0].Since.Equal(held) {
		t.Fatalf("the held work is written into waiting with the time first held: %+v", st.Waiting)
	}
	before := stateBytes(t, f.repo.Root(), f.runID)
	f.now = f.now.Add(5 * time.Minute)
	if r := f.step(t); !r.CeilingReached {
		t.Fatalf("a second call before any receipt still finds the ceiling: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a second call before any receipt leaves the waiting time unchanged")
	}

	// C3: 14 minutes after the call that first held it, a receipt frees a slot.
	f.now = held.Add(14 * time.Minute)
	f.ret(t, "lane-1", RoleRuthless, "SHIP")
	if r := f.step(t); r.Awaiting == nil || r.Awaiting.Role != RoleAuditor {
		t.Fatalf("the held auditor takes the freed slot: %+v", r)
	}
	st = f.state(t)
	if len(st.Waiting) != 0 || !strings.Contains(recordText(st), "the intent-auditor of lane-1 took a freed slot after waiting 14 minute(s)") {
		t.Fatalf("the record names the lane, the role and 14 minutes waited:\n%s", recordText(st))
	}
}

// rewrite replaces the run's state through the loop's own writer: a test's
// way to stand up a Given the steps alone do not reach in one run.
func (f *parFixture) rewrite(t *testing.T, fn func(*State)) {
	t.Helper()
	st := f.state(t)
	fn(&st)
	root, err := os.OpenRoot(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := writeState(root, st); err != nil {
		t.Fatal(err)
	}
}

// needs sets the resolved needs of the pending steps numbered in steps.
func needs(st *State, n []int, steps ...int) {
	for i := range st.Pending {
		if slices.Contains(steps, st.Pending[i].Number) {
			st.Pending[i].Needs = append([]int{}, n...)
		}
	}
}

// C4, implementers and reviewers together: with three slots, lane 1's two
// reviewers out and step 2 marked `- needs: none`, stepping until lane 2's
// implementer is handed its brief opens lane 2 in its own worktree, and its
// implementer takes the third slot; the next call finds the ceiling reached. A
// stage the binary performs itself proceeds at the ceiling.
func TestAnImplementerAndReviewersShareTheSlots(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: none\n", Options{SubAgents: strp("3")})
	// The Given: steps 2 and 3 wait until lane 1's reviewers are out.
	f.rewrite(t, func(st *State) { needs(st, []int{1}, 2, 3) })
	f.stepUntil(t, "lane-1's implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.stepUntil(t, "lane-1's reviewers are out", func(st State) bool { return len(st.Lanes[0].Awaits) == 2 })
	f.rewrite(t, func(st *State) { needs(st, []int{}, 2, 3) })

	res := f.stepUntil(t, "lane-2's implementer is out", func(st State) bool {
		return len(st.Lanes) >= 2 && len(st.Lanes[1].Awaits) == 1
	})
	st := f.state(t)
	if len(st.Lanes) != 2 || st.Lanes[1].SpecStep != 2 || st.Lanes[1].Worktree == "" || st.Lanes[1].Worktree == st.Lanes[0].Worktree || st.Lanes[1].Branch == st.Lanes[0].Branch {
		t.Fatalf("lane 2 opens in its own worktree on its own branch, and step 3 waits: %+v", st.Lanes)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || st.SlotsInUse() != 3 {
		t.Fatalf("lane 2's implementer takes the third slot: %+v, %d in use", res, st.SlotsInUse())
	}
	if r := f.step(t); !r.CeilingReached {
		t.Fatalf("the next call finds the ceiling reached: %+v", r)
	}
	if st := f.state(t); len(st.Waiting) != 1 || st.Waiting[0].Lane != "step 3" {
		t.Fatalf("the new lane for step 3 waits on the ceiling: %+v", st.Waiting)
	}
	// A lane at a stage the binary owns moves at the ceiling.
	f.rewrite(t, func(st *State) { openLane(st, 0) })
	if r := f.step(t); r.PerformedStage != StageWorktree || r.Lane != "lane-3" || r.Slots != 3 {
		t.Fatalf("the binary's own stage proceeds at the ceiling: %+v", r)
	}
}

// C5, the order: with lane 2's security reviewer, lane 1's fix implementer and
// a new lane for step 4 all waiting, the first freed slot goes to lane 1's fix
// implementer, the next to lane 2's reviewer, and step 4's lane opens last.
func TestAFreedSlotGoesToOpenLanesBeforeNewOnes(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: none\n4. Four\n   - needs: none\n", Options{SubAgents: strp("3")})
	f.rewrite(t, func(st *State) { needs(st, []int{1}, 2, 3, 4) })
	f.stepUntil(t, "lane-1's implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.stepUntil(t, "lane-1's reviewers are out", func(st State) bool { return len(st.Lanes[0].Awaits) == 2 })
	f.rewrite(t, func(st *State) { needs(st, []int{}, 2) })
	f.ret(t, "lane-1", RoleSecurity, "APPROVE")
	f.stepUntil(t, "lane-2's implementer is out", func(st State) bool { return len(st.Lanes) == 2 && len(st.Lanes[1].Awaits) == 1 })
	f.implement(t, "lane-2", "two.txt")
	if r := f.step(t); r.Awaiting == nil || r.Lane != "lane-2" || r.Awaiting.Role != RoleRuthless {
		t.Fatalf("lane 2's ruthless reviewer is out: %+v", r)
	}
	f.ret(t, "lane-1", RoleRuthless, "FIX FIRST")
	// The run is full: one slot, lane 2's ruthless reviewer in it; step 4 is
	// ready, lane 1's findings wait for a fix implementer, lane 2's security
	// reviewer for its turn.
	f.rewrite(t, func(st *State) {
		st.Pace.SubAgents.Value = 1
		needs(st, []int{}, 4)
	})
	if r := f.step(t); !r.CeilingReached {
		t.Fatalf("the run is full: %+v", r)
	}
	st := f.state(t)
	var queue []string
	for _, w := range st.Waiting {
		queue = append(queue, w.Lane+" "+w.Role)
	}
	if want := []string{"lane-1 implementer", "lane-2 security-reviewer", "step 4 implementer"}; !slices.Equal(queue, want) {
		t.Fatalf("the waiting work, in the order a freed slot takes it: %v, want %v", queue, want)
	}
	f.rewrite(t, func(st *State) { st.Pace.SubAgents.Value = 2 })
	if r := f.step(t); r.Lane != "lane-1" || r.Awaiting == nil || r.Awaiting.Role != RoleImplementer {
		t.Fatalf("the first freed slot goes to lane 1's fix implementer: %+v", r)
	}
	f.ret(t, "lane-2", RoleRuthless, "SHIP")
	if r := f.step(t); r.Lane != "lane-2" || r.Awaiting == nil || r.Awaiting.Role != RoleSecurity {
		t.Fatalf("the next freed slot goes to lane 2's security reviewer: %+v", r)
	}
	if n := len(f.state(t).Lanes); n != 2 {
		t.Fatalf("step 4's lane has not opened yet: %d lanes", n)
	}
	f.ret(t, "lane-2", RoleSecurity, "APPROVE")
	f.stepUntil(t, "step 4's lane opens", func(st State) bool { return len(st.Lanes) == 3 })
	if st := f.state(t); st.Lanes[2].SpecStep != 4 || !strings.Contains(recordText(st), "step 4 took a freed slot") {
		t.Fatalf("step 4's lane opens last, and the record names its wait:\n%s", recordText(st))
	}
}

// C6, needs: a step without the line needs every step before it (ruling
// DR6b), so no lane opens for it until lane 1's pull request is an ancestor of
// the default branch, even with a slot free; a needs line naming a later step
// is refused before a run starts, naming the line.
func TestAStepWithoutANeedsLineWaitsForEveryEarlierStep(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n", Options{SubAgents: strp("3")})
	f.stepUntil(t, "lane-1's implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")
	for range 3 {
		if st := f.state(t); len(st.Lanes) != 1 || st.SlotsInUse() >= st.Ceiling() {
			t.Fatalf("no lane opens for step 2 while lane 1 is open, with slots free: %+v", st.Lanes)
		}
		l := f.lane(t, "lane-1")
		if l.Landing != nil && l.Landing.RecordsDone && l.Landing.Pushed == "" {
			preflighted(t, l, l.HeadSHA)
		}
		f.step(t)
	}
	f.landed(t, "lane-1")
	st := f.state(t)
	if len(st.Lanes) != 2 || st.Lanes[1].SpecStep != 2 {
		t.Fatalf("step 2's lane opens once lane 1 has landed: %+v", st.Lanes)
	}

	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. One\n2. Two\n   - needs: 3\n3. Three\n"))
	_, err := Start(repo.Root(), "itd-10", Options{})
	if err == nil || !strings.Contains(err.Error(), "line") || !strings.Contains(err.Error(), "needs") {
		t.Fatalf("a needs line naming a later step is refused before the run starts, naming the line: %v", err)
	}
}

// C7, isolation and the receipt: of two lanes each awaiting its implementer,
// lane 2's receipt advances lane 2 and leaves lane 1 unchanged; their
// worktrees and branches differ; a path no await names is refused and frees
// nothing.
func TestAReceiptAdvancesOnlyTheLaneItBelongsTo(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("2")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	one, _ := json.Marshal(f.lane(t, "lane-1"))
	st := f.state(t)
	if st.Lanes[0].Worktree == st.Lanes[1].Worktree || st.Lanes[0].Branch == st.Lanes[1].Branch {
		t.Fatalf("two lanes never share a worktree or a branch: %+v", st.Lanes)
	}
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := Receipt(f.repo.Root(), f.runID, f.abs(RunRelDir+"/"+f.runID+"/nowhere.json"), f.stages, f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "lane-1's implementer") || !strings.Contains(r.Reason, "lane-2's implementer") {
		t.Fatalf("a path no await names is refused naming the awaits there are: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused receipt frees nothing")
	}
	res := f.implement(t, "lane-2", "two.txt")
	if res.Lane != "lane-2" || res.Stage != StageValidate {
		t.Fatalf("lane 2 advances: %+v", res)
	}
	if now, _ := json.Marshal(f.lane(t, "lane-1")); !bytes.Equal(one, now) {
		t.Fatalf("lane 1 is unchanged:\n%s\n%s", one, now)
	}
	if n := f.state(t).SlotsInUse(); n != 1 {
		t.Fatalf("lane 2's verified receipt frees its slot: %d in use", n)
	}
}

// C8, a clean sync, and C10, one landing at a time: two lanes whose rounds
// pass land one after the other, the lower step first; lane 2 waits while lane
// 1 lands, then merges the default branch in with a merge commit (never a
// rebase), and a fresh round, the closing lane's with its auditor, judges the
// merge head before lane 2 arms; no fix round is counted.
func TestASiblingLandingSyncsTheLaneBeforeItArms(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	if v := f.lane(t, "lane-2").Validation[0].Validators; len(v) != 2 {
		t.Fatalf("lane 2's first round takes no audit while lane 1 is open: %+v", v)
	}
	judged := f.lane(t, "lane-2").HeadSHA

	// C10: lane 1 lands first; lane 2 waits at its landing.
	for range 30 {
		l1 := f.lane(t, "lane-1")
		if l1.Stage == StageDone {
			break
		}
		if l2 := f.lane(t, "lane-2"); l2.Landing != nil {
			t.Fatalf("lane 2 waits while lane 1 lands: %+v", l2.Landing)
		}
		if l1.Landing != nil && l1.Landing.RecordsDone && l1.Landing.Pushed == "" {
			preflighted(t, l1, l1.HeadSHA)
		}
		if l1.Landing != nil && l1.Landing.Merge != "" {
			f.repo.Git("push", "-q", "origin", l1.Landing.Pushed+":refs/heads/main")
			if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("MERGED\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		f.step(t)
	}
	if strings.Count(f.ghLog(t), "pr merge 7 --auto") != 1 {
		t.Fatalf("lane 1 armed once, alone:\n%s", f.ghLog(t))
	}
	for _, name := range []string{"pr.json", "state"} {
		_ = os.Remove(filepath.Join(f.gh, name))
	}

	// C8: the sync.
	res := f.step(t)
	l2 := f.lane(t, "lane-2")
	if l2.Stage != StageValidate || len(l2.Syncs) != 1 || l2.Syncs[0].Conflicted || l2.Syncs[0].Siblings[0] != "lane-1" || l2.Landing != nil {
		t.Fatalf("lane 2 is synced before its landing, back to a fresh round: %+v %+v", res, l2)
	}
	parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", l2.HeadSHA))
	if len(parents) != 3 || parents[1] != judged || parents[2] != l2.Syncs[0].Merged {
		t.Fatalf("the sync is a merge commit over the judged head, never a rebase: %v", parents)
	}
	f.roundPassed(t, "lane-2")
	l2 = f.lane(t, "lane-2")
	r := l2.Validation[len(l2.Validation)-1]
	if len(l2.Validation) != 2 || r.HeadSHA != l2.Syncs[0].Head || len(r.Validators) != 3 || l2.fixRoundsTaken() != 0 {
		t.Fatalf("a fresh round, with the closing lane's auditor, judges the merge head and counts no fix round: %+v", l2.Validation)
	}
	if !strings.Contains(recordText(f.state(t)), "synced lane-2") {
		t.Fatalf("the record names the sync:\n%s", recordText(f.state(t)))
	}
	f.landed(t, "lane-2")
	if st := f.state(t); !st.Complete() || !f.lane(t, "lane-2").Landing.Closes {
		t.Fatalf("lane 2 lands as the closing lane and the run completes: %+v", st)
	}
	if strings.Count(f.ghLog(t), "pr merge 7 --auto") != 2 {
		t.Fatalf("lane 2 arms only after its sync's round:\n%s", f.ghLog(t))
	}
}

// C9, a conflicting sync: lane 1 landed a change to a file lane 2 also
// changed; the merge is aborted with the branch unchanged, a fresh implementer
// is handed a sync brief naming the file and lane 1, a receipt whose head does
// not contain the merged sha is refused, and a verified one opens a fresh
// round; no fix round is counted.
func TestAConflictingSyncGoesToAFreshImplementer(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "same.txt")
	l2 := f.lane(t, "lane-2")
	if err := os.WriteFile(filepath.Join(l2.Worktree, "same.txt"), []byte("lane two's words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l2.Worktree, "add", "--", "same.txt")
	f.repo.Git("-C", l2.Worktree, "commit", "-q", "-m", "lane two")
	f.receipt(t, "lane-2", f.await(t, "lane-2", RoleImplementer), strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD")))
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	judged := f.lane(t, "lane-2").HeadSHA

	f.step(t)
	l2 = f.lane(t, "lane-2")
	if len(l2.Syncs) != 1 || !l2.Syncs[0].Conflicted || !slices.Equal(l2.Syncs[0].Paths, []string{"same.txt"}) || l2.HeadSHA != judged {
		t.Fatalf("the conflicting merge is aborted, the branch unchanged: %+v", l2)
	}
	if tip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/"+l2.Branch)); tip != judged {
		t.Fatalf("the branch stays at the judged head: %s", tip)
	}
	if st := strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "status", "--porcelain")); st != "" {
		t.Fatalf("the aborted merge leaves the worktree clean: %s", st)
	}
	res := f.step(t)
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Awaiting.Receipt != l2.Syncs[0].Receipt {
		t.Fatalf("a fresh implementer is handed the sync brief: %+v", res)
	}
	brief, err := os.ReadFile(f.abs(res.Awaiting.Brief))
	if err != nil || !strings.Contains(string(brief), "same.txt") || !strings.Contains(string(brief), "lane-1") || !strings.Contains(string(brief), l2.Syncs[0].Merged) {
		t.Fatalf("the sync brief names the file, lane 1 and the merged sha: %v\n%s", err, brief)
	}
	// A commit that does not merge the default branch in is refused.
	plain := laneCommit(t, f.repo, l2, "other.txt")
	a := f.await(t, "lane-2", RoleImplementer)
	laneDir := f.abs(RunRelDir + "/" + f.runID + "/lane-2")
	rel, _ := filepath.Rel(laneDir, filepath.Dir(f.abs(a.Receipt)))
	for name, body := range map[string]string{filepath.Join(rel, ReportFileName): "merged\n", filepath.Join(rel, DoDFileName): "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: "lane-2", Branch: l2.Branch, Commits: []string{plain},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: filepath.ToSlash(filepath.Join(rel, DoDFileName))},
		Report:           filepath.ToSlash(filepath.Join(rel, ReportFileName)), Model: "claude-test-5"}
	data, _ := json.Marshal(rc)
	if err := os.WriteFile(f.abs(a.Receipt), data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Receipt(f.repo.Root(), f.runID, f.abs(a.Receipt), f.stages, f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "does not contain") {
		t.Fatalf("a receipt whose head does not contain the merged sha is refused: %+v", r)
	}
	// The implementer merges and resolves.
	cmd := []string{"-C", l2.Worktree, "merge", "--no-edit", l2.Syncs[0].Merged}
	_ = gitErr(f.repo, cmd...)
	if err := os.WriteFile(filepath.Join(l2.Worktree, "same.txt"), []byte("both lanes' words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l2.Worktree, "add", "--", "same.txt")
	f.repo.Git("-C", l2.Worktree, "commit", "-q", "--no-edit")
	merge := strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD"))
	rc.Commits = []string{plain, merge}
	data, _ = json.Marshal(rc)
	if err := os.WriteFile(f.abs(a.Receipt), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(f.repo.Root(), f.runID, f.abs(a.Receipt), f.stages, f.opts()); err != nil {
		t.Fatalf("a receipt carrying the merged sha verifies: %v", err)
	}
	if r := f.step(t); r.Awaiting == nil || r.Awaiting.Role != RoleRuthless {
		t.Fatalf("a verified sync opens a fresh round: %+v", r)
	}
	l2 = f.lane(t, "lane-2")
	if n := len(l2.Validation); n != 2 || l2.Validation[1].HeadSHA != merge || l2.fixRoundsTaken() != 0 || l2.Syncs[0].Head != merge {
		t.Fatalf("the fresh round judges the resolved head and counts no fix round: %+v", l2)
	}
}

// C11, the pace across lanes: with two lanes each with an agent out and the
// window elapsed, a step starts nothing on either lane, both receipts are still
// verified, and next_eligible_at is written once.
func TestAnElapsedWindowStartsNothingOnAnyLane(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("2"), Pace: strp("30/60")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.now = f.now.Add(31 * time.Minute)
	res := f.step(t)
	if res.NextEligibleAt == nil || res.Awaiting == nil {
		t.Fatalf("the elapsed window closes and starts nothing: %+v", res)
	}
	f.implement(t, "lane-1", "one.txt")
	f.implement(t, "lane-2", "two.txt")
	st := f.state(t)
	if st.SlotsInUse() != 0 || st.Lanes[0].Stage != StageValidate || st.Lanes[1].Stage != StageValidate {
		t.Fatalf("both receipts are verified during the pause: %+v", st.Lanes)
	}
	if _, err := Advance(f.repo.Root(), f.runID, f.stages, f.opts()); err == nil {
		t.Fatal("a step before next_eligible_at is refused as a pause")
	}
	pauses := 0
	for _, e := range f.state(t).Record {
		if e.Stage == "pause" {
			pauses++
		}
	}
	if pauses != 1 || f.state(t).NextEligibleAt == nil {
		t.Fatalf("next_eligible_at is written once for the run:\n%s", recordText(f.state(t)))
	}
}

// heldRun stands up C13's Given: three slots, one fix round, steps 2 and 3
// marked `- needs: none`, lane 1 handed back while lane 2's implementer is
// out; lane 2's receipt is verified, its fake validators pass, and the loop is
// stepped until nothing moves.
func heldRun(t *testing.T) *parFixture {
	t.Helper()
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: none\n", Options{SubAgents: strp("3"), FixRounds: strp("1")})
	f.rewrite(t, func(st *State) { needs(st, []int{1}, 3) })
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	for round := 1; round <= 2; round++ {
		f.stepUntil(t, "lane-1's reviewers are out", func(st State) bool { return len(st.Lanes[0].Awaits) == 2 })
		f.ret(t, "lane-1", RoleRuthless, "FIX FIRST")
		f.ret(t, "lane-1", RoleSecurity, "APPROVE")
		if round == 1 {
			f.stepUntil(t, "lane-1's fix implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
			f.receipt(t, "lane-1", f.await(t, "lane-1", RoleImplementer), laneCommit(t, f.repo, f.lane(t, "lane-1"), "fix.txt"))
		}
	}
	f.stepUntil(t, "lane-1 is handed back", func(st State) bool { return st.Lanes[0].Stage == StageHandedBack })
	f.rewrite(t, func(st *State) { needs(st, []int{}, 3) })
	if len(f.lane(t, "lane-2").Awaits) != 1 {
		t.Fatal("lane 2's implementer is out at the hand-back")
	}
	f.implement(t, "lane-2", "two.txt")
	f.roundPassed(t, "lane-2")
	for range 10 {
		if f.lane(t, "lane-2").Stage == StageHeld {
			break
		}
		f.step(t)
	}
	return f
}

// C13, the hold and the person's decision (ruling DR6c), and C11's hand-back:
// after lane 1 is handed back, lane 2 finishes and is held before its push,
// never armed; no lane opens for step 3 and no lane closes the spec; status
// shows the held lane with its cause, head and the two flags and no slot in
// use; the next step refuses naming the hand-back and the held lane. A release
// of a lane that is not held changes nothing; one of lane 2 takes it through
// its landing on the fake forge.
func TestAfterAHandBackTheSiblingsFinishAndAreHeld(t *testing.T) {
	f := heldRun(t)
	st := f.state(t)
	l2 := f.lane(t, "lane-2")
	judged := l2.Validation[len(l2.Validation)-1].HeadSHA
	if l2.Stage != StageHeld || l2.Hold == nil || l2.Hold.Cause != "lane-1" || l2.Hold.Head != judged || l2.Hold.Before != HoldBeforePush {
		t.Fatalf("lane 2 is held before its push, naming lane 1 and its judged head: %+v", l2)
	}
	if log := f.ghLog(t); log != "" {
		t.Fatalf("the fake forge records no pull request and no arming:\n%s", log)
	}
	if f.repo.Git("ls-remote", "origin", "refs/heads/"+l2.Branch) != "" {
		t.Fatal("nothing is pushed")
	}
	if len(st.Lanes) != 2 || len(st.Pending) != 1 {
		t.Fatalf("no lane opens for step 3: %+v", st.Lanes)
	}
	if v := l2.Validation[0].Validators; len(v) != 2 {
		t.Fatalf("no lane closes the spec, so no audit is taken: %+v", v)
	}
	if !strings.Contains(recordText(st), "lane-2 held after lane-1's hand-back") {
		t.Fatalf("the run record names the hold:\n%s", recordText(st))
	}
	if st.SlotsInUse() != 0 {
		t.Fatalf("a held lane holds no slot: %d", st.SlotsInUse())
	}
	res, err := Advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "unachievable") || !strings.Contains(r.Reason, "lane-2 is held") || !strings.Contains(r.Reason, "--release lane-2") {
		t.Fatalf("the next step refuses naming lane 1's hand-back and lane 2 held: %+v %+v", r, res)
	}

	before := stateBytes(t, f.repo.Root(), f.runID)
	if _, err := Release(f.repo.Root(), f.runID, "lane-1", f.opts()); err == nil {
		t.Fatal("releasing a lane that is not held is refused")
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused release leaves the state byte-identical")
	}
	rel, err := Release(f.repo.Root(), f.runID, "lane-2", f.opts())
	if err != nil || rel.Stage != StageLand {
		t.Fatalf("a released lane returns to its landing: %+v %v", rel, err)
	}
	if !strings.Contains(recordText(f.state(t)), "the person released lane-2") {
		t.Fatalf("the run record names the release:\n%s", recordText(f.state(t)))
	}
	f.landed(t, "lane-2")
	if log := f.ghLog(t); !strings.Contains(log, "pr create") || !strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("the released lane lands through the fake forge:\n%s", log)
	}
	if l := f.lane(t, "lane-2"); l.Landing.Closes {
		t.Fatal("a run with a hand-back never closes the spec")
	}
}

// C13's discard: the same held lane in a second run, discarded, has its
// worktree and branch gone and the stage `discarded`; its step stays unlanded
// in the spec, and the fake forge records nothing.
func TestADiscardedHeldLaneLeavesItsStepUnlanded(t *testing.T) {
	f := heldRun(t)
	l2 := f.lane(t, "lane-2")
	res, err := Discard(f.repo.Root(), f.runID, "lane-2", f.opts())
	if err != nil || res.Stage != StageDiscarded {
		t.Fatalf("the held lane is discarded: %+v %v", res, err)
	}
	if _, err := os.Stat(l2.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is gone: %v", err)
	}
	if gitErr(f.repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+l2.Branch) == nil {
		t.Fatal("the lane's branch is gone")
	}
	spec, err := os.ReadFile(f.abs(specRel))
	if err != nil || strings.Contains(string(spec), "landed:") {
		t.Fatalf("step 2 stays unlanded in the spec: %v\n%s", err, spec)
	}
	if log := f.ghLog(t); log != "" {
		t.Fatalf("the fake forge records nothing:\n%s", log)
	}
	if !strings.Contains(recordText(f.state(t)), "the person discarded lane-2") {
		t.Fatalf("the run record names the discard:\n%s", recordText(f.state(t)))
	}
}

// C12, the schema: a version-7 state file with one lane awaiting its
// implementer reads as one await and runs on; the next write is version 8 and
// carries `awaits`, never `awaiting`. A version-7 file carrying what only
// version 8 writes, and a version-8 file carrying `awaiting`, are refused
// naming the version and the key.
func TestAVersion7StateMigratesItsAwaitAndVersion8IsHeldToItsShape(t *testing.T) {
	f := newParFixture(t, "", Options{})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	path := f.abs(StateRelPath(f.runID))
	v8, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// as rewrites the file as a version with lane 0 and the pending steps
	// edited by edit.
	as := func(version int, edit func(doc, lane map[string]any)) []byte {
		var doc map[string]any
		if err := json.Unmarshal(v8, &doc); err != nil {
			t.Fatal(err)
		}
		doc["schema_version"] = version
		lane := doc["lanes"].([]any)[0].(map[string]any)
		edit(doc, lane)
		out, _ := json.MarshalIndent(doc, "", "  ")
		if err := os.WriteFile(path, out, 0o600); err != nil {
			t.Fatal(err)
		}
		return out
	}
	serial := func(doc, lane map[string]any) {
		lane["awaiting"] = lane["awaits"].([]any)[0]
		delete(lane, "awaits")
	}

	as(7, serial)
	st := f.state(t)
	if st.SchemaVersion != SchemaVersion || len(st.Lanes[0].Awaits) != 1 || st.Lanes[0].Awaits[0].Role != RoleImplementer {
		t.Fatalf("a version-7 file reads as a lane with one await: %+v", st.Lanes[0])
	}
	f.implement(t, "lane-1", "one.txt")
	written, _ := os.ReadFile(path)
	if !strings.Contains(string(written), `"schema_version": 8`) || strings.Contains(string(written), `"awaiting"`) {
		t.Fatalf("the next write is version 8 and never writes awaiting:\n%s", written)
	}
	f.step(t)
	if written, _ = os.ReadFile(path); !strings.Contains(string(written), `"awaits"`) {
		t.Fatalf("version 8 writes awaits:\n%s", written)
	}
	v8 = written

	for name, tc := range map[string]struct {
		version int
		edit    func(doc, lane map[string]any)
		key     string
	}{
		"v7 awaits": {7, func(doc, lane map[string]any) {}, "`awaits`"},
		"v7 waiting": {7, func(doc, lane map[string]any) {
			serial(doc, lane)
			doc["waiting"] = []any{map[string]any{"lane": "lane-1", "role": "x", "since": "2026-09-30T09:00:00Z"}}
		}, "`waiting`"},
		"v7 syncs": {7, func(doc, lane map[string]any) {
			serial(doc, lane)
			lane["syncs"] = []any{map[string]any{"at": "2026-09-30T09:00:00Z", "siblings": []any{}, "merged": "x", "conflicted": false}}
		}, "`syncs`"},
		"v7 held":     {7, func(doc, lane map[string]any) { serial(doc, lane); lane["stage"] = "held" }, "`held`"},
		"v8 awaiting": {8, func(doc, lane map[string]any) { lane["awaiting"] = lane["awaits"].([]any)[0] }, "`awaiting`"},
	} {
		as(tc.version, tc.edit)
		_, err := ReadState(f.repo.Root(), f.runID)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, "schema version "+strconv.Itoa(tc.version)) || !strings.Contains(r.Reason, tc.key) {
			t.Errorf("%s: refused naming the version and the key: %+v", name, r)
		}
	}
}
