package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// tiering is the model-tier routing a test step resolves through: the
// machine's routing table holding agents, read at the repository and a home of
// the test's own, with no provider configured.
func tiering(t *testing.T, repo *gittest.Repo, agents string) *Tiering {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "oracle-routing.json"), []byte(`{"schema_version":1,"agents":`+agents+`}`), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := oracle.Load(layered.Roots{Repo: repo.Root(), Home: home})
	if err != nil {
		t.Fatal(err)
	}
	return &Tiering{Table: l, Conns: oracle.NoConnections{}}
}

// tieredHandBack is handBack, stepping with o, and returns the step that
// handed the role out.
func tieredHandBack(t *testing.T, repo *gittest.Repo, runID string, o Options, role, body string) StepResult {
	t.Helper()
	res, err := advance(repo.Root(), runID, DefaultStages(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != role {
		t.Fatalf("the step hands out a fresh %s: %+v", role, res)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	if role == RoleAuditor {
		body = auditorVerdict(t, filepath.Join(filepath.Dir(path), AuditRequestFileName), body)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(repo.Root(), runID, path, DefaultStages(), o); err != nil {
		t.Fatalf("the %s's return: %v", role, err)
	}
	return res
}

// tieredFix hands the failed round's findings to a fresh implementer, stepping
// with o, and hands back its receipt applying one commit.
func tieredFix(t *testing.T, repo *gittest.Repo, runID string, o Options, file string) StepResult {
	t.Helper()
	res, err := advance(repo.Root(), runID, DefaultStages(), o)
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("a failed round goes to a fresh implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, runID)
	sha := laneCommit(t, repo, l, file)
	laneDir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, l.ID)
	rel, _ := filepath.Rel(laneDir, filepath.Dir(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))))
	rel = filepath.ToSlash(rel)
	for name, body := range map[string]string{rel + "/" + ReportFileName: "applied it in " + sha + "\n", rel + "/" + DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: runID, Lane: l.ID, Branch: l.Branch, Commits: []string{sha},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: rel + "/" + DoDFileName}, Report: rel + "/" + ReportFileName}
	data, _ := json.Marshal(rc)
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(repo.Root(), runID, path, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	return res
}

// routeOfRole is role's route in round n of the run's first lane, read back
// from the state file.
func routeOfRole(t *testing.T, repo *gittest.Repo, runID string, n int, role string) Round {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range st.Lanes[0].Validation[n-1].Routes {
		if r.Role == role {
			return r
		}
	}
	t.Fatalf("round %d records no route for the %s: %+v", n, role, st.Lanes[0].Validation[n-1].Routes)
	return Round{}
}

// TestAFailedRoundRunsTheNextOneTierUp is spc-2609251028149555's AC 10: a lane
// whose round did not pass runs its next round one tier up from the tier each
// role ran at, and the state file records the round, the failing gate output
// and the switch, one step per failed round, never two.
func TestAFailedRoundRunsTheNextOneTierUp(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"},"security-reviewer":{"tier":"local"}}`)}
	implemented(t, repo, id, DefaultStages(), "one.txt")

	// Round 1: each role at its own row's tier; the ruthless reviewer fails it.
	if res := tieredHandBack(t, repo, id, o, RoleRuthless, reviewerReturn("FIX FIRST")); res.Tier != oracle.Economy ||
		!strings.Contains(res.Next, "at model tier economy") {
		t.Fatalf("round 1 hands the ruthless reviewer out at its row's tier: %+v", res)
	}
	tieredHandBack(t, repo, id, o, RoleSecurity, reviewerReturn("APPROVE"))
	tieredHandBack(t, repo, id, o, RoleAuditor, "MET")
	if r := routeOfRole(t, repo, id, 1, RoleRuthless); r.Tier != oracle.Economy || r.EscalatedFrom != "" || r.GateOutput != "" {
		t.Fatalf("round 1 records the row's tier and no switch: %+v", r)
	}
	tieredFix(t, repo, id, o, "fix-1.txt")
	if r := routeOfRole(t, repo, id, 1, RoleImplementer); r.Tier != "" || r.EscalatedFrom != "" ||
		!strings.Contains(r.GateOutput, "round 1 did not pass") || !strings.Contains(r.Note, "names no tier") {
		t.Fatalf("the fix implementer, which has no routing row, is recorded with the failure and does not move: %+v", r)
	}

	// Round 2: every role one step up; the security reviewer fails it.
	res := tieredHandBack(t, repo, id, o, RoleRuthless, reviewerReturn("SHIP"))
	if res.Tier != oracle.Frontier {
		t.Fatalf("round 2 hands the ruthless reviewer out one tier up: %+v", res)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil || !strings.Contains(string(brief), "tier: frontier") {
		t.Fatalf("the brief's routing section asks for the escalated tier: %v\n%s", err, brief)
	}
	tieredHandBack(t, repo, id, o, RoleSecurity, reviewerReturn("BLOCK"))
	tieredHandBack(t, repo, id, o, RoleAuditor, "MET")
	r := routeOfRole(t, repo, id, 2, RoleRuthless)
	if r.Tier != oracle.Frontier || r.EscalatedFrom != oracle.Economy {
		t.Fatalf("round 2's switch is recorded: %+v", r)
	}
	for _, want := range []string{"round 1 did not pass", "ruthless-reviewer FIX FIRST", "round-1/ruthless-reviewer/return.md"} {
		if !strings.Contains(r.GateOutput, want) {
			t.Fatalf("round 2 records the failing gate output (%q): %q", want, r.GateOutput)
		}
	}
	if s := routeOfRole(t, repo, id, 2, RoleSecurity); s.Tier != oracle.Economy || s.EscalatedFrom != oracle.Local {
		t.Fatalf("the security reviewer moves one step, from local to economy: %+v", s)
	}
	if a := routeOfRole(t, repo, id, 2, RoleAuditor); a.Tier != oracle.Frontier || a.EscalatedFrom != "" || !strings.Contains(a.Note, "top of the ladder") {
		t.Fatalf("the auditor at frontier does not move and says why: %+v", a)
	}
	tieredFix(t, repo, id, o, "fix-2.txt")

	// Round 3: one step from round 2's tier, never two from the row's.
	tieredHandBack(t, repo, id, o, RoleRuthless, reviewerReturn("SHIP"))
	tieredHandBack(t, repo, id, o, RoleSecurity, reviewerReturn("APPROVE"))
	if s := routeOfRole(t, repo, id, 3, RoleSecurity); s.Tier != oracle.Frontier || s.EscalatedFrom != oracle.Economy ||
		!strings.Contains(s.GateOutput, "round 2 did not pass") || !strings.Contains(s.GateOutput, "security-reviewer BLOCK") {
		t.Fatalf("round 3 moves the security reviewer from round 2's economy to frontier: %+v", s)
	}
	if r := routeOfRole(t, repo, id, 3, RoleRuthless); r.Tier != oracle.Frontier || r.EscalatedFrom != "" {
		t.Fatalf("the ruthless reviewer stays at the top: %+v", r)
	}
	st, _ := ReadState(repo.Root(), id)
	var record strings.Builder
	for _, e := range st.Record {
		record.WriteString(e.Note + "\n")
	}
	if !strings.Contains(record.String(), "moves up from economy to frontier after the round before did not pass") {
		t.Fatalf("the run record says the switch:\n%s", record.String())
	}
}

// TestARouteThatNamesNoTierDoesNotMove: a role the step hands to a runner (a
// machine route or a --route naming a runner and a model), a role whose row is
// host-decides, and the implementer, which has no routing row, name no tier:
// after a failed round each is recorded with the failure and a note saying so,
// escalated_from empty, and none is swapped for another route.
func TestARouteThatNamesNoTierDoesNotMove(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"},"security-reviewer":{"tier":"host-decides"}}`)}
	const text = "ruthless-reviewer=runner:opencode@example/model-1"
	o.runnerRoute = func(role string) (string, string) {
		if role == RoleRuthless {
			return "opencode", text
		}
		return "", ""
	}
	implemented(t, repo, id, DefaultStages(), "one.txt")
	if res := tieredHandBack(t, repo, id, o, RoleRuthless, reviewerReturn("FIX FIRST")); res.Tier != "" || strings.Contains(res.Next, "model tier") {
		t.Fatalf("a role on a runner is asked for no tier: %+v", res)
	}
	tieredHandBack(t, repo, id, o, RoleSecurity, reviewerReturn("APPROVE"))
	tieredHandBack(t, repo, id, o, RoleAuditor, "MET")
	tieredFix(t, repo, id, o, "fix.txt")
	res := tieredHandBack(t, repo, id, o, RoleRuthless, reviewerReturn("SHIP"))
	brief, _ := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if res.Tier != "" || strings.Contains(string(brief), "## Routing") {
		t.Fatalf("a runner route after a failed round asks for no tier: %+v\n%s", res, brief)
	}
	tieredHandBack(t, repo, id, o, RoleSecurity, reviewerReturn("APPROVE"))
	for _, c := range []struct{ role, note string }{
		{RoleRuthless, "opencode runner under --route " + text},
		{RoleSecurity, "host-decides, which names no tier"},
	} {
		r := routeOfRole(t, repo, id, 2, c.role)
		if r.Tier != "" || r.EscalatedFrom != "" || !strings.Contains(r.GateOutput, "round 1 did not pass") ||
			!strings.Contains(r.Note, c.note) || !strings.Contains(r.Note, "does not move") {
			t.Errorf("the %s after a failed round is recorded with the failure and why it does not move: %+v", c.role, r)
		}
	}
}

// TestAPassingRoundHoldsTheTierTheLaneReached: a round that passed moves
// nothing, up or down, so a fresh round after it (a sync) runs at the tier the
// lane's failures took it to, and a role new to the lane starts at its row's.
func TestAPassingRoundHoldsTheTierTheLaneReached(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	rt := &roundRouter{tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"local"},"security-reviewer":{"tier":"local"}}`)}
	lane := Lane{ID: "lane-1", Validation: []ValidationRound{
		{Round: 1, Fix: "fix-1", Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST"}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Local}}},
		{Round: 2, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "SHIP", Pass: true}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Economy, EscalatedFrom: oracle.Local}}},
		{Round: 3},
	}}
	if failedBefore(lane, 3) != nil || failedBefore(lane, 2) == nil {
		t.Fatal("round 3 follows a passing round, round 2 a failed one")
	}
	r, route, err := rt.route(lane, 3, RoleRuthless, failedBefore(lane, 3))
	if err != nil || r.Tier != oracle.Economy || r.EscalatedFrom != "" || r.GateOutput != "" || route == nil || route.Row.Tier != oracle.Economy {
		t.Fatalf("a round after a pass keeps the tier the lane reached, neither up nor down: %+v %+v %v", r, route, err)
	}
	if s, _, err := rt.route(lane, 3, RoleSecurity, nil); err != nil || s.Tier != oracle.Local {
		t.Fatalf("a role new to the lane starts at its row's tier: %+v %v", s, err)
	}
}

// TestAnEscalatedTierOffTheAllowlistRefusesTheStep is AC 11 under AC 10: an
// escalated tier a connection serves is held to that connection's allowlist,
// and a route it does not admit refuses the step, naming the provider, rather
// than running the round on another route.
func TestAnEscalatedTierOffTheAllowlistRefusesTheStep(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	tiers := tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)
	tiers.Conns = servesFrontier{}
	rt := &roundRouter{tiers: tiers}
	lane := Lane{ID: "lane-1", Validation: []ValidationRound{
		{Round: 1, Fix: "fix-1", Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST", Return: "r"}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Economy}}},
		{Round: 2},
	}}
	_, _, err := rt.route(lane, 2, RoleRuthless, failedBefore(lane, 2))
	r := mustRefusal(t, err)
	if r.Stage != string(StageValidate) || !strings.Contains(r.Reason, "frontier") || !strings.Contains(r.Reason, "big-cloud") ||
		!strings.Contains(r.Remedy, "nothing was written") {
		t.Fatalf("the escalated route the allowlist refuses refuses the step, naming the tier and the provider: %+v", r)
	}
}

// servesFrontier is a machine whose one connection serves frontier and lists
// no model, so it admits no route.
type servesFrontier struct{}

func (servesFrontier) Serves(t oracle.Tier) (oracle.Connection, bool) {
	return oracle.Connection{Name: "big-cloud"}, t == oracle.Frontier
}
func (servesFrontier) Named(string) (oracle.Connection, bool)   { return oracle.Connection{}, false }
func (servesFrontier) Pointed(string) (oracle.Connection, bool) { return oracle.Connection{}, false }

// TestARoundsRouteIsStateVersion16s: a round's route joined version 16 before
// any release wrote that version, so a file of version 15 or lower carrying
// one is not one its version wrote and is refused, naming the version and the
// key; one of version 15, 14 or 13 carrying none reads as the current version.
func TestARoundsRouteIsStateVersion16s(t *testing.T) {
	if schemaVersionUnplanned != 16 {
		t.Fatalf("a round's route is state version 16's, got %d", schemaVersionUnplanned)
	}
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)}
	implemented(t, repo, id, DefaultStages(), "one.txt")
	if _, err := advance(repo.Root(), id, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	current, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(current), `"routes"`) {
		t.Fatalf("the round records its route: %v\n%s", err, current)
	}
	if _, err := ReadState(repo.Root(), id); err != nil {
		t.Fatalf("version 16 reads a round's route: %v", err)
	}
	for _, version := range []int{15, 14, 13, 12, 9} {
		if err := os.WriteFile(path, current, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, repo.Root(), id, func(doc map[string]any) { doc["schema_version"] = version })
		_, err := ReadState(repo.Root(), id)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", version)) || !strings.Contains(r.Reason, "`routes`") {
			t.Errorf("version %d carrying a round's route is refused naming the version and the key: %+v", version, r)
		}
	}
	for _, version := range []int{15, 14, 13} {
		if err := os.WriteFile(path, current, 0o600); err != nil {
			t.Fatal(err)
		}
		editState(t, repo.Root(), id, func(doc map[string]any) {
			doc["schema_version"] = version
			for _, v := range doc["lanes"].([]any)[0].(map[string]any)["validation"].([]any) {
				delete(v.(map[string]any), "routes")
			}
		})
		if st, err := ReadState(repo.Root(), id); err != nil || st.SchemaVersion != SchemaVersion {
			t.Fatalf("a version-%d file with no round's route reads as the current version: %v %v", version, st.SchemaVersion, err)
		}
	}
}

// TestARoundsGateOutputIsBoundedAndMasked: the gate output a round's route
// stores is free text, so it is held to its bound and masked when it is read,
// as every other free text of the state is.
func TestARoundsGateOutputIsBoundedAndMasked(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)}
	implemented(t, repo, id, DefaultStages(), "one.txt")
	if _, err := advance(repo.Root(), id, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	long := "round 1 did not pass ‮" + strings.Repeat("é", 4*maxGateOutputBytes)
	editState(t, repo.Root(), id, func(doc map[string]any) {
		v := doc["lanes"].([]any)[0].(map[string]any)["validation"].([]any)[0].(map[string]any)
		v["routes"].([]any)[0].(map[string]any)["gate_output"] = long
	})
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	got := st.Lanes[0].Validation[0].Routes[0].GateOutput
	if strings.ContainsRune(got, '‮') || !strings.HasPrefix(got, "round 1 did not pass") {
		t.Fatalf("the gate output is masked: %q", got)
	}
	if len(got) > maxGateOutputBytes+16 || !strings.HasSuffix(got, "...") {
		t.Fatalf("the gate output is bounded to %d bytes: %d", maxGateOutputBytes, len(got))
	}
	if g := gateOutput(ValidationRound{Round: 1, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST",
		Return: strings.Repeat("x/", maxGateOutputBytes)}}}); len(g) > maxGateOutputBytes {
		t.Fatalf("the gate output is bounded when it is composed: %d bytes", len(g))
	}
}
