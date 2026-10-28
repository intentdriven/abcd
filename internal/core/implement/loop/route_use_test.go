package loop

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// TestAStepsRouteForARoleItDoesNotHandOutIsRefused: a --route the step would
// not use is refused before anything is written, naming the role the step
// hands out or that it hands none: a reviewer routed while the step hands out
// the implementer, and the implementer routed while the step re-tells the
// await an earlier step began.
func TestAStepsRouteForARoleItDoesNotHandOutIsRefused(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok", "claude")
	routed := func(t *testing.T, text, role string) *runner.Config {
		t.Helper()
		cfg := runnerConfig(t, `{"runner":{"claude":{}}}`, "")
		routes, err := oracle.ParseRunnerRoutes([]string{text}, []string{role})
		if err != nil {
			t.Fatal(err)
		}
		if err := cfg.Override(routes); err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	before := stateBytes(t, root, id)
	_, err := Drive(context.Background(), root, id, steps, o,
		Runners{Config: routed(t, "security-reviewer=runner:claude", RoleSecurity), Transcripts: &memTranscripts{}})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "security-reviewer=runner:claude") || !strings.Contains(r.Reason, "lane-1's implementer") {
		t.Fatalf("the refusal names the --route and the role the step hands out: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, root, id)) {
		t.Fatal("a refused --route wrote the run's state")
	}

	// The host is handed the implementer; a second step re-tells that await.
	if res, err := advance(root, id, steps, o); err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	before = stateBytes(t, root, id)
	_, err = Drive(context.Background(), root, id, steps, o,
		Runners{Config: routed(t, "implementer=runner:claude", RoleImplementer), Transcripts: &memTranscripts{}})
	r = mustRefusal(t, err)
	if !strings.Contains(r.Reason, "hands no role") || !strings.Contains(r.Reason, "lane-1's implementer") {
		t.Fatalf("the refusal says the step hands no role, the implementer already out: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, root, id)) {
		t.Fatal("a refused --route wrote the run's state")
	}
}

// TestARouteALandingMayPreemptIsJudgedAfterTheMove: a lane at its landing
// moves first unless the forge holds it, so whether the step hands out the
// role a --route names is known only once the landing has moved or waited.
// The prediction leaves that route to the move; a route for any other role
// is refused, naming both outcomes; and a route the move did not use is named
// in the record and the result.
func TestARouteALandingMayPreemptIsJudgedAfterTheMove(t *testing.T) {
	st := State{RunID: "run-0000000000000001", Lanes: []Lane{
		{ID: "lane-1", SpecStep: 1, Stage: StageLand},
		{ID: "lane-2", SpecStep: 2, Stage: StageImplement},
	}}
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	unsure, err := checkRoutes(st, map[string]string{RoleImplementer: "implementer=runner:claude"}, &outageGate{}, now)
	if err != nil || unsure != RoleImplementer {
		t.Fatalf("the implementer's route waits on the landing's move: %q %v", unsure, err)
	}
	_, err = checkRoutes(st, map[string]string{RoleRuthless: "ruthless-reviewer=runner:claude"}, &outageGate{}, now)
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "lane-1") || !strings.Contains(r.Reason, "lane-2's implementer") {
		t.Fatalf("the refusal names the landing and the role behind it: %+v", r)
	}

	res := StepResult{Lane: "lane-1", PerformedStage: StageLand}
	if !noteUnusedRoute(&st, &res, map[string]string{RoleImplementer: "implementer=runner:claude"}, RoleImplementer, now) {
		t.Fatal("a route the move did not use is written to the record")
	}
	if len(res.RouteUnused) != 1 || res.RouteUnused[0] != "implementer=runner:claude" {
		t.Fatalf("the result names the unused route: %+v", res.RouteUnused)
	}
	last := st.Record[len(st.Record)-1]
	if !strings.Contains(last.Note, "--route implementer=runner:claude was not used") {
		t.Fatalf("the record says the route was not used: %+v", last)
	}
	used := StepResult{Lane: "lane-2", Awaiting: &Await{Role: RoleImplementer}, handed: true}
	if noteUnusedRoute(&st, &used, map[string]string{RoleImplementer: "implementer=runner:claude"}, RoleImplementer, now) {
		t.Fatal("a route the move used is not noted as unused")
	}
}

// TestARouteTheOutageHoldsIsRefusedBeforeTheWait pins why advanceWith's wait
// branch, where the call ends in a wait after a landing contended, never
// meets a --route left to the move: the agent decision checkRoutes reads is
// the one moveAgent makes, on the same gate. A landing the network outage
// holds waits, and the implementer it would hand out, started through a
// runner, waits on the outage too: the move answers with that wait, and
// checkRoutes refuses the implementer's --route before anything is written.
// A landing that meets a network failure of its own collects a report
// (g.take) and leaves the gate's outage as it was, so the agent decision
// after it is the one checkRoutes read. A landing the outage holds is read as
// waiting, through the gate move asks, not as one that may move.
func TestARouteTheOutageHoldsIsRefusedBeforeTheWait(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	st := State{RunID: "run-0000000000000001", Lanes: []Lane{
		{ID: "lane-1", SpecStep: 1, Stage: StageLand, Landing: &Landing{RecordsDone: true}},
		{ID: "lane-2", SpecStep: 2, Stage: StageImplement},
	}}
	runs := func(string) bool { return true }
	g := &outageGate{now: now, runner: runs, cur: &implement.Outage{Status: implement.OutageOpen,
		Services: []string{implement.ServiceNetwork}, Down: []string{implement.ServiceNetwork}, NextProbeAt: now.Add(time.Hour)}}
	routes := map[string]string{RoleImplementer: "implementer=runner:claude"}

	moving := st
	moving.Lanes = slices.Clone(st.Lanes)
	_, _, err := move("", &moving, Stages{}, now, g, nil)
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "implementer of lane-2 is not started") {
		t.Fatalf("the move ends waiting on the outage for the implementer: %+v", r)
	}
	unsure, err := checkRoutes(st, routes, g, now)
	r = mustRefusal(t, err)
	if unsure != "" || !strings.Contains(r.Reason, "hands no role") || !strings.Contains(r.Reason, "outage") {
		t.Fatalf("the --route is refused up front, naming the outage: %q %+v", unsure, r)
	}

	// An implementer the host starts is not held: the landing the outage
	// holds surely waits, so the step surely hands the implementer out, and
	// its --route is not left to the move.
	host := &outageGate{now: now, runner: func(string) bool { return false }, cur: g.cur}
	if unsure, err := checkRoutes(st, routes, host, now); err != nil || unsure != "" {
		t.Fatalf("a landing the outage holds leaves nothing to the move: %q %v", unsure, err)
	}

	quiet := &outageGate{now: now, runner: runs}
	role, lane, none := agentHand(st, quiet)
	if !quiet.take(netFailure(StageLand, "lane-1", "git push", errors.New("fatal: unable to access: Could not resolve host: github.com"))) {
		t.Fatal("the landing's network failure is collected")
	}
	if r2, l2, n2 := agentHand(st, quiet); r2 != role || l2 != lane || n2 != none || role != RoleImplementer {
		t.Fatalf("a collected report leaves the agent decision as it was: %q %q %q, then %q %q %q", role, lane, none, r2, l2, n2)
	}
}
