package loop

// receipt_standing_test.go proves iss-2611031337115764: a step refused at the
// moment it would collect a finished agent loses nothing. The agent's receipt
// stands at the path the lane awaits, so the next step verifies it, as
// `implement receipt` would, before anything asks for a fresh agent.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
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
	if len(l.Awaits) != 0 || len(l.Receipts) != 1 || l.Receipts[0].Route == nil || l.Receipts[0].Route.Ran != runner.Claude {
		t.Fatalf("the verified receipt names the runner its claim named: %+v", l)
	}
	if !recordSays(t, root, id, "receipt", "stood at the path", "verified") {
		t.Fatalf("the record says the receipt was collected where it stood: %+v", st.Record)
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
// refused once pauses the role, and a paused role is retried on its runner.
// A retry is one more paid launch, so a receipt that verifies now ends the
// pause instead.
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
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("the paused role is not launched again over a receipt that verifies: %d launches", n)
	}
}

// TestARouteDoesNotStartARunnerOverAReceiptThatVerifies: a --route retries a
// paused role at once, before the step's own move, and the launch itself is
// where a standing receipt is read for that path.
func TestARouteDoesNotStartARunnerOverAReceiptThatVerifies(t *testing.T) {
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
	res, err := Drive(context.Background(), root, id, refusing, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate {
		t.Fatalf("the --route collects the receipt: %+v", res)
	}
	if len(res.RouteUnused) != 1 || res.RouteUnused[0] != "implementer=runner:claude" {
		t.Fatalf("a --route that started nothing is named as unused: %+v", res)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("the --route launches nothing over a receipt that verifies: %d launches", n)
	}
}

// TestARouteStaysRefusedOverAStandingReceiptAndAPlainStepCollectsIt: a --route
// is refused, never ignored (route_use.go), so a step carrying one for a role
// it will not hand out is refused as ever, with nothing written, and collects
// nothing. The next plain step collects the receipt.
func TestARouteStaysRefusedOverAStandingReceiptAndAPlainStepCollectsIt(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the host is handed the implementer: %+v %v", res, err)
	}
	if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := newDriveEnv(t, "ok", "claude")
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
	if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err == nil {
		t.Fatal("a --route the step will not use is refused")
	}
	after, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a refused --route wrote the state:\n%s", after)
	}
	res, err = Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate {
		t.Fatalf("the plain step collects the receipt: %+v", res)
	}
	if n := env.launches(t, "claude"); n != 0 {
		t.Fatalf("nothing is launched: %d launches", n)
	}
}

// TestAHostStepCollectsAReceiptAndALateHandInIsHarmless: a role left on the
// host is the same: a step that finds the receipt verifies it rather than
// telling the host to start a fresh agent. The host's own hand-in, arriving
// after, is answered rather than refused, and moves nothing twice; so is a
// runner's, whose dispatch ends after a peer's step collected its receipt.
func TestAHostStepCollectsAReceiptAndALateHandInIsHarmless(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the host is handed the implementer: %+v %v", res, err)
	}
	handIn := receipt
	receipt := res.Awaiting.Receipt
	if err := os.WriteFile(receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate || res.PerformedStage != StageImplement {
		t.Fatalf("the step collects the receipt: %+v", res)
	}
	if strings.Contains(res.Next, "start a fresh implementer") {
		t.Fatalf("the step asks for a fresh agent: %s", res.Next)
	}
	before, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	for name, hand := range map[string]func() (StepResult, error){
		"the host's": func() (StepResult, error) { return Receipt(root, id, receipt, steps, o) },
		"a runner's": func() (StepResult, error) {
			return handIn(root, id, receipt, steps, o, &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude})
		},
	} {
		res, err := hand()
		if err != nil {
			t.Fatalf("%s hand-in after a step collected it is refused: %v", name, err)
		}
		if res.Stage != StageValidate || res.PerformedStage != "" {
			t.Fatalf("%s hand-in moves nothing: %+v", name, res)
		}
	}
	after, err := os.ReadFile(runStatePath(root, id))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a hand-in of a receipt already collected wrote the state:\n%s", after)
	}
}

// TestAStandingReceiptThatDoesNotVerifyIsSaidAndKept: the step asks for
// nothing new on its account: the await stays out, the file stays where it is,
// and the step says why it was not taken.
func TestAStandingReceiptThatDoesNotVerifyIsSaidAndKept(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the host is handed the implementer: %+v %v", res, err)
	}
	receipt := res.Awaiting.Receipt
	if err := os.WriteFile(receipt, []byte("half a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageImplement || res.Awaiting == nil || res.Awaiting.Receipt != receipt {
		t.Fatalf("the await stays out: %+v", res)
	}
	if !strings.Contains(res.Next, "does not verify") || !strings.Contains(res.Next, "no receipt the contract accepts") {
		t.Fatalf("the step says the standing receipt did not verify, and why: %s", res.Next)
	}
	if body, err := os.ReadFile(receipt); err != nil || string(body) != "half a receipt" {
		t.Fatalf("the file is left where it stands: %q %v", body, err)
	}
}

// TestTheRealVerifierCollectsAStandingReceipt: the same through the implement
// stage's own verifier, which moves the lane's head to its branch's tip, and a
// reviewer's return the same way through the validate stage's.
func TestTheRealVerifierCollectsAStandingReceipt(t *testing.T) {
	repo, runID, l, dir := awaitingLane(t)
	c := laneCommit(t, repo, l, "one.txt")
	writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c))

	res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate || res.PerformedStage != StageImplement {
		t.Fatalf("the verified receipt moves the lane on: %+v", res)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Lanes[0]; got.HeadSHA != c || got.Receipt == "" || len(got.Awaits) != 0 {
		t.Fatalf("the lane's head is its branch's tip and its receipt is recorded: %+v", got)
	}

	res, err = advance(repo.Root(), runID, DefaultStages(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the validate stage hands out its reviewer: %+v", res)
	}
	back := absIn(repo.Root(), res.Awaiting.Receipt)
	if err := os.MkdirAll(filepath.Dir(back), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(back, []byte(reviewReturn), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = advance(repo.Root(), runID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err = ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	vs := st.Lanes[0].Validation[len(st.Lanes[0].Validation)-1].Validators
	if vs[0].Verdict != "SHIP" {
		t.Fatalf("the reviewer's standing return is recorded: %+v", vs)
	}
}

// TestTheImplementStageReadsItsReceiptPathBeforeItAsksForAnAgent: a lane at
// the implement stage with no await out, the await having been lost, finds the
// receipt at the path the stage would await, and asks for no agent if it
// verifies. One that does not is said, and a fresh agent takes the brief. One
// the lane has already verified is not the answer to a later hand-out.
func TestTheImplementStageReadsItsReceiptPathBeforeItAsksForAnAgent(t *testing.T) {
	dropAwait := func(t *testing.T, root, runID string, keep func(*Lane)) {
		t.Helper()
		if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].Awaits = nil
			if keep != nil {
				keep(&st.Lanes[0])
			}
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("a receipt that verifies", func(t *testing.T) {
		repo, runID, l, dir := awaitingLane(t)
		c := laneCommit(t, repo, l, "one.txt")
		writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c))
		dropAwait(t, repo.Root(), runID, nil)
		res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Stage != StageValidate || res.Awaiting != nil {
			t.Fatalf("the stage takes the receipt and asks for no agent: %+v", res)
		}
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		if got := st.Lanes[0]; got.HeadSHA != c || got.Receipt != l.awaiting().Receipt {
			t.Fatalf("the lane records the receipt: %+v", got)
		}
	})
	t.Run("a receipt that does not", func(t *testing.T) {
		repo, runID, _, dir := awaitingLane(t)
		writeReceipt(t, dir, "half a receipt")
		dropAwait(t, repo.Root(), runID, nil)
		res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
			t.Fatalf("a fresh agent takes the brief: %+v", res)
		}
		if !recordSays(t, repo.Root(), runID, string(StageImplement), "did not verify") {
			t.Fatal("the record says the standing receipt did not verify")
		}
	})
	t.Run("a receipt the lane already took", func(t *testing.T) {
		repo, runID, l, dir := awaitingLane(t)
		c := laneCommit(t, repo, l, "one.txt")
		writeReceipt(t, dir, goodReceipt(t, runID, l, dir, c))
		dropAwait(t, repo.Root(), runID, func(lane *Lane) { lane.Receipt = l.awaiting().Receipt })
		res, err := advance(repo.Root(), runID, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Stage != StageImplement || res.Awaiting == nil {
			t.Fatalf("a later hand-out is not answered by the receipt the lane already verified: %+v", res)
		}
	})
}
