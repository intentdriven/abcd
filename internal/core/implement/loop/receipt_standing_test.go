package loop

// receipt_standing_test.go proves iss-2611031337115764 and what it stops at. A
// step collects the receipt of a routed implementer the run's own state shows a
// runner was started for, so a step refused between that agent's finish and its
// collection loses nothing. Everything else stays with the path it always took:
// an agent's return is handed back by its runner's dispatch or by
// `implement receipt`, and a hand-out starts its agent.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// refuseVerifyOnce makes the implement stage's verifier refuse its first call,
// after running before, and pass from then on.
func refuseVerifyOnce(steps Stages, before func()) Stages {
	out := append(Stages(nil), steps...)
	for i := range out {
		if out[i].Name != StageImplement {
			continue
		}
		verify := out[i].Verify
		refused := false
		out[i].Verify = func(c Context, l *Lane, receipt string) error {
			if !refused {
				refused = true
				if before != nil {
					before()
				}
				return refuse("receipt", "", l.ID, "refused once", "hand it back again")
			}
			return verify(c, l, receipt)
		}
	}
	return out
}

// claimAwait records a runner's claim on the lane's await at receipt, as
// dispatchAwait does before it starts the runner.
func claimAwait(t *testing.T, root, id, receipt string, until time.Time) {
	t.Helper()
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		i, k := st.findAwait(root, receipt)
		if i < 0 {
			t.Fatalf("no await at %s", receipt)
		}
		st.Lanes[i].Awaits[k].Running = &RunnerClaim{Runner: runner.Claude, Session: "the-claim", Until: until}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// routeOnClaude routes a role to the claude runner.
func routeOnClaude(role string) string {
	return fmt.Sprintf(`{"roles":{%q:{"runner":"claude"}},"runner":{"claude":{}}}`, role)
}

// TestAFinishedAgentIsCollectedAfterTheStateRefusedItsCollection is the issue's
// own case: the routed implementer finishes and writes its receipt, the run's
// state is raised under the collecting step, so that step, its pause and its
// claim's release are all refused and the claim stays standing; and once the
// state is put right, the next step reaches validate with no new launch.
func TestAFinishedAgentIsCollectedAfterTheStateRefusedItsCollection(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	rs := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}
	path := runStatePath(root, id)
	var good []byte
	refusing := refuseVerifyOnce(steps, func() {
		var err error
		if good, err = os.ReadFile(path); err != nil {
			t.Fatal(err)
		}
		from := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
		raised := bytes.Replace(good, []byte(from), []byte(fmt.Sprintf(`"schema_version": %d,`, SchemaVersion+1)), 1)
		if err := os.WriteFile(path, raised, 0o600); err != nil {
			t.Fatal(err)
		}
	})

	if _, err := Drive(context.Background(), root, id, refusing, o, rs); err == nil || !strings.Contains(err.Error(), "newer abcd") {
		t.Fatalf("the collecting step is refused on the state: %v", err)
	}
	if err := os.WriteFile(path, good, 0o600); err != nil {
		t.Fatal(err)
	}
	if st, err := ReadState(root, id); err != nil || st.Lanes[0].Awaits[0].Running == nil {
		t.Fatalf("the refused step could not drop its claim: %+v %v", st.Lanes, err)
	}

	res, err := Drive(context.Background(), root, id, refusing, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate || res.PerformedStage != StageImplement || res.Awaiting != nil {
		t.Fatalf("the next step collects the finished agent and the lane reaches validate: %+v", res)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("no second implementer is launched: %d launches", n)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	if len(l.Awaits) != 0 || len(l.Receipts) != 1 {
		t.Fatalf("the receipt is taken: %+v", l)
	}
	// The step read a file, and no runner reported to it: the route says what
	// was asked, and leaves what ran unknown.
	if r := l.Receipts[0].Route; r == nil || r.Asked != runner.Claude || r.Ran != "" {
		t.Fatalf("the route names the runner asked for and no runner as having run it: %+v", l.Receipts[0].Route)
	}
	if !recordSays(t, root, id, "receipt", "stood at the path", "verified") {
		t.Fatalf("the record says the receipt was collected where it stood: %+v", st.Record)
	}
	if !recordSays(t, root, id, StageRunner, "routed to claude", "no runner's report reached") {
		t.Fatalf("the record says no report arrived: %+v", st.Record)
	}
	if recordSays(t, root, id, StageRunner, "ran on the") {
		t.Fatalf("the record does not say a runner ran what no runner reported: %+v", st.Record)
	}
	// The run goes on past it, still with no second launch.
	if _, err := Drive(context.Background(), root, id, refusing, o, rs); err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("the run goes on with no second implementer: %d launches", n)
	}
}

// TestARefusedReceiptThatNowVerifiesIsNotRetriedOnTheRunner: a verifier that
// refused a runner's answer pauses the role, and the pause records that runner.
// The next step collects a receipt that verifies now, reports no retry, and
// names the runner the pause recorded.
func TestARefusedReceiptThatNowVerifiesIsNotRetriedOnTheRunner(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	rs := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}
	refusing := refuseVerifyOnce(steps, nil)

	res, err := Drive(context.Background(), root, id, refusing, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil {
		t.Fatalf("a receipt the verifier refuses pauses the role: %+v", res)
	}
	res, err = Drive(context.Background(), root, id, refusing, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate || res.PerformedStage != StageImplement {
		t.Fatalf("the receipt is collected: %+v", res)
	}
	if res.Retried != nil {
		t.Fatalf("a collection starts nothing, so it is no retry: %+v", res.Retried)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("the paused role is not launched again over a receipt that verifies: %d launches", n)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Lanes[0].Receipts[0].Route; r == nil || r.Asked != runner.Claude || r.Ran != "" {
		t.Fatalf("the receipt names the runner its pause recorded: %+v", r)
	}
}

// TestARouteOverACollectableReceiptIsRefusedAndAPlainStepCollectsIt: a --route
// is refused, never ignored (route_use.go), and the role a receipt stands for
// is not one a --route retries. Nothing is written or started and no retry is
// reported, and the next plain step collects.
func TestARouteOverACollectableReceiptIsRefusedAndAPlainStepCollectsIt(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	refusing := refuseVerifyOnce(steps, nil)
	if res, err := Drive(context.Background(), root, id, refusing, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}); err != nil || res.Paused == nil {
		t.Fatalf("the role pauses: %+v %v", res, err)
	}
	cfg := runnerConfig(t, claudeImplementer, "")
	routes, err := oracle.ParseRunnerRoutes([]string{"implementer=runner:claude"}, []string{RoleImplementer})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Drive(context.Background(), root, id, refusing, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err == nil {
		t.Fatalf("a --route the step will not use is refused: %+v", res)
	}
	if res.Retried != nil {
		t.Fatalf("nothing started, so no retry is reported: %+v", res.Retried)
	}
	after, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a refused --route wrote the state:\n%s", after)
	}
	res, err = Drive(context.Background(), root, id, refusing, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate {
		t.Fatalf("the plain step collects the receipt: %+v", res)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("nothing is started over a receipt that verifies: %d launches", n)
	}
}

// TestAHostsAwaitIsHandedBackByTheHost: nothing in a host's await shows an
// agent was started for it, so a step leaves a file at its path alone and
// re-tells it, and `implement receipt` is what hands it back.
func TestAHostsAwaitIsHandedBackByTheHost(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the host is handed the implementer: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageImplement || res.Awaiting == nil || res.Awaiting.Receipt != path {
		t.Fatalf("the step re-tells the await: %+v", res)
	}
	res, err = Receipt(root, id, path, steps, o)
	if err != nil || res.Stage != StageValidate {
		t.Fatalf("the host's hand-in moves the lane on: %+v %v", res, err)
	}
}

// TestALaterHandInOfACollectedReceiptIsReadBeforeItIsAnswered: after a step
// collected a receipt, a hand-in of the same bytes changes nothing unless it
// brings the runner's own report, which replaces the route the step could not
// know. A file that has since changed is refused, and the lane is left where
// it is for a person.
func TestALaterHandInOfACollectedReceiptIsReadBeforeItIsAnswered(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	claimAwait(t, root, id, path, time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC))
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err = advance(root, id, steps, o); err != nil || res.Stage != StageValidate {
		t.Fatalf("the step collects the receipt: %+v %v", res, err)
	}

	before, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	res, err = Receipt(root, id, path, steps, o)
	if err != nil {
		t.Fatalf("a hand-in of the bytes the step took is answered: %v", err)
	}
	if res.Stage != StageValidate || res.PerformedStage != "" {
		t.Fatalf("it moves nothing: %+v", res)
	}
	after, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a hand-in of the same bytes wrote the state:\n%s", after)
	}

	reported := &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: "the-model"}
	if _, err := receipt(root, id, path, steps, o, reported); err != nil {
		t.Fatalf("the runner's hand-in of the same bytes is answered: %v", err)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Lanes[0].Receipts[0].Route; r == nil || r.Ran != runner.Claude || r.Model != "the-model" {
		t.Fatalf("the runner's report replaces what the step could not know: %+v", r)
	}
	if !recordSays(t, root, id, StageRunner, "ran on the claude runner", "the-model") {
		t.Fatalf("the record names the run the runner reported: %+v", st.Record)
	}

	if err := os.WriteFile(path, []byte("{\"changed\": true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err = os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	for name, hand := range map[string]func() (StepResult, error){
		"the host's": func() (StepResult, error) { return Receipt(root, id, path, steps, o) },
		"a runner's": func() (StepResult, error) { return receipt(root, id, path, steps, o, reported) },
	} {
		if _, err := hand(); err == nil || !strings.Contains(err.Error(), "changed since") {
			t.Fatalf("%s hand-in of a file that changed is refused: %v", name, err)
		}
	}
	after, err = os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a refused hand-in wrote the state:\n%s", after)
	}
}

// TestAReviewersReturnAtATakenPathIsNotAnsweredUnread: a hand-in at a path whose
// verdict the loop has recorded is refused as a path no await names, whatever
// the file now says, and the recorded verdict stands.
func TestAReviewersReturnAtATakenPathIsNotAnsweredUnread(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	handBackOne(t, repo, id, stages, RoleRuthless, reviewerReturn("SHIP"))
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	back := absIn(repo.Root(), st.Lanes[0].Validation[0].Validators[0].Return)
	if err := os.WriteFile(back, []byte(reviewerReturn("FIX FIRST")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(repo.Root(), id, back, stages, Options{}); err == nil {
		t.Fatal("a second, different return at a taken path is refused")
	}
	st, err = ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" {
		t.Fatalf("the recorded verdict stands: %+v", v)
	}
}

// TestAStandingReceiptThatDoesNotVerifyIsSaidAndKept: for a role a runner was
// started for, the step asks for nothing new on the receipt's account: the
// await stays out, the file stays where it is, and the step says why it was not
// taken.
func TestAStandingReceiptThatDoesNotVerifyIsSaidAndKept(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the host is handed the implementer: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	claimAwait(t, root, id, path, time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC))
	if err := os.WriteFile(path, []byte("half a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageImplement || res.Awaiting == nil || res.Awaiting.Receipt != path {
		t.Fatalf("the await stays out: %+v", res)
	}
	if !strings.Contains(res.Next, "does not verify") || !strings.Contains(res.Next, "no receipt the contract accepts") {
		t.Fatalf("the step says the standing receipt did not verify, and why: %s", res.Next)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "half a receipt" {
		t.Fatalf("the file is left where it stands: %q %v", body, err)
	}
}

// TestARunnersReceiptMissingItsModelIsNotRelaunchedOver: a routed implementer
// may leave the model to its runner's report, and a refused hand-in loses that
// report, so its finished receipt cannot verify again. No second runner is
// started over that work, and every step says what the receipt lacks.
func TestARunnersReceiptMissingItsModelIsNotRelaunchedOver(t *testing.T) {
	repo, id, l, dir := awaitingLane(t)
	c := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, c)
	body := map[string]any{"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report}
	t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), body))
	env := newDriveEnv(t, "receipt", "claude")
	stages := refuseVerifyOnce(DefaultStages(), nil)
	rs := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}

	res, err := Drive(context.Background(), repo.Root(), id, stages, Options{}, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil {
		t.Fatalf("the refused hand-in pauses the role: %+v", res)
	}
	for range 2 {
		if res, err = Drive(context.Background(), repo.Root(), id, stages, Options{}, rs); err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("a finished agent's work is not paid for twice: %d launches", n)
		}
		if res.Stage != StageImplement || !strings.Contains(res.Next, "does not verify") || !strings.Contains(res.Next, "the model your harness runs") {
			t.Fatalf("the step says what the receipt lacks: %s", res.Next)
		}
	}
}

// TestAFirstHandOutStartsItsAgentWhateverStandsAtItsPath: no agent has been
// started for an await a step is only now handing out, so what stands at its
// path is not its answer. The reviewer's return, and the receipt of a fix
// round, each start their agent on a routed run, and a host's step collects
// neither.
func TestAFirstHandOutStartsItsAgentWhateverStandsAtItsPath(t *testing.T) {
	// A review set's reviewer asks for its class's model, which these fakes
	// have no provider for: a run of one reviewer per role, asking for none.
	reviewed := func(t *testing.T) (*gittest.Repo, string, Stages) {
		t.Helper()
		repo := briefRepo(t, agentsMarked)
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := mutate(repo.Root(), start.RunID, func(_ *os.Root, st *State) (bool, error) {
			st.Review = nil
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
		return repo, start.RunID, DefaultStages()
	}
	write := func(t *testing.T, root, rel, body string) {
		t.Helper()
		abs := absIn(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	returnOf := func(t *testing.T, id, lane string) string {
		t.Helper()
		dir, err := reviewerDir(id, lane, 1, RoleRuthless, 0)
		if err != nil {
			t.Fatal(err)
		}
		return dir + "/" + ReturnFileName
	}

	t.Run("a routed reviewer", func(t *testing.T) {
		repo, id, stages := reviewed(t)
		l := currentLane(t, repo, id)
		write(t, repo.Root(), returnOf(t, id, l.ID), reviewerReturn("SHIP"))
		env := newDriveEnv(t, "ok", "claude")
		res, err := Drive(context.Background(), repo.Root(), id, stages, Options{},
			Runners{Config: runnerConfig(t, routeOnClaude(RoleRuthless), ""), Transcripts: &memTranscripts{}})
		if err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("the reviewer is started: %d launches, %+v", n, res)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; v.Route == nil || v.Route.Ran != runner.Claude {
			t.Fatalf("the verdict is the started reviewer's: %+v", v)
		}
	})
	t.Run("a host's reviewer", func(t *testing.T) {
		repo, id, stages := reviewed(t)
		l := currentLane(t, repo, id)
		write(t, repo.Root(), returnOf(t, id, l.ID), reviewerReturn("SHIP"))
		if res, err := advance(repo.Root(), id, stages, Options{}); err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
			t.Fatalf("the reviewer is handed out: %+v %v", res, err)
		}
		if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" {
			t.Fatalf("no verdict is recorded until the host hands one back: %+v", v)
		}
	})
	t.Run("a reviewer with a runner's claim", func(t *testing.T) {
		for name, until := range map[string]time.Time{
			"running": time.Now().Add(time.Hour),
			"lapsed":  time.Now().Add(-time.Hour),
		} {
			t.Run(name, func(t *testing.T) {
				repo, id, stages := reviewed(t)
				res, err := advance(repo.Root(), id, stages, Options{})
				if err != nil || res.Awaiting == nil {
					t.Fatalf("the reviewer is handed out: %+v %v", res, err)
				}
				claimAwait(t, repo.Root(), id, res.Awaiting.Receipt, until)
				write(t, repo.Root(), res.Awaiting.Receipt, reviewerReturn("SHIP"))
				if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
					t.Fatal(err)
				}
				st, err := ReadState(repo.Root(), id)
				if err != nil {
					t.Fatal(err)
				}
				if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" {
					t.Fatalf("a step reads no reviewer's return: %+v", v)
				}
			})
		}
	})
	t.Run("a fix round's implementer", func(t *testing.T) {
		repo, id, stages := reviewed(t)
		l := currentLane(t, repo, id)
		handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
		passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
		fixDir, err := roundDir(id, l.ID, 1, FixDirName)
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
		first, err := os.ReadFile(filepath.Join(dir, ReceiptFileName))
		if err != nil {
			t.Fatal(err)
		}
		write(t, repo.Root(), fixDir+"/"+ReceiptFileName, string(first))
		res, err := advance(repo.Root(), id, stages, Options{})
		if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
			t.Fatalf("the fix round's implementer is handed out: %+v %v", res, err)
		}
		claimAwait(t, repo.Root(), id, res.Awaiting.Receipt, time.Now().Add(time.Hour))
		if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if lane := st.Lanes[0]; lane.Validation[0].Fix != "" || len(lane.Awaits) != 1 {
			t.Fatalf("the round stays open for its implementer: %+v", lane.Validation[0])
		}
	})
	t.Run("the implement stage's first hand-out", func(t *testing.T) {
		repo, id, l, dir := awaitingLane(t)
		c := laneCommit(t, repo, l, "one.txt")
		writeReceipt(t, dir, goodReceipt(t, id, l, dir, c))
		if err := mutate(repo.Root(), id, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].Awaits = nil
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		res, err := advance(repo.Root(), id, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Stage != StageImplement || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
			t.Fatalf("the stage hands out its agent: %+v", res)
		}
	})
}

// TestAStepCollectingWhileTheAgentWorksLandsNothingUnjudged: a peer step can
// collect while the runner's agent is still working, since a claim names no
// process to ask. The agent's later commit moves the branch past the head the
// lane took, its rewritten receipt is refused rather than answered, and the
// landing refuses the moved tip, so nothing the validators did not judge lands.
func TestAStepCollectingWhileTheAgentWorksLandsNothingUnjudged(t *testing.T) {
	repo, id, l, dir := awaitingLane(t)
	stages := DefaultStages()
	claimAwait(t, repo.Root(), id, l.awaiting().Receipt, time.Now().Add(time.Hour))
	first := laneCommit(t, repo, l, "one.txt")
	path := writeReceipt(t, dir, goodReceipt(t, id, l, dir, first))

	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Stage != StageValidate {
		t.Fatalf("the peer's step collects the receipt: %+v %v", res, err)
	}
	if got := currentLane(t, repo, id); got.HeadSHA != first {
		t.Fatalf("the lane takes the head the receipt verified at: %+v", got)
	}

	// The agent goes on working, and rewrites its receipt.
	second := laneCommit(t, repo, l, "two.txt")
	writeReceipt(t, dir, goodReceipt(t, id, l, dir, first, second))
	if _, err := receipt(repo.Root(), id, path, stages, Options{}, &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: "the-model"}); err == nil ||
		!strings.Contains(err.Error(), "changed since") {
		t.Fatalf("the agent's rewritten receipt is refused: %v", err)
	}

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	if lane.HeadSHA != first {
		t.Fatalf("the refused hand-in moves no head: %+v", lane)
	}
	_, err = landPrepare(Context{RepoRoot: repo.Root(), RunDir: runRel(id), State: st, Now: time.Now()}, &lane)
	if err == nil || !strings.Contains(err.Error(), "not at the head") {
		t.Fatalf("the landing refuses a tip past the judged head: %v", err)
	}
}
