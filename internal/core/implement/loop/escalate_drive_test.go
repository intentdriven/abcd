package loop

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The production wiring of the tier escalation, held through Drive, the call
// `implement step` makes: the runner configuration names the runner a role
// runs on, the reviewer route and a role pointed at a provider name no tier,
// and the tables LoadTiering reads decide the tier.

// driveRound starts a run, takes its lane to its validators and drives one
// step with o and cfg, returning the step and the ruthless reviewer's route
// in round 1.
func driveRound(t *testing.T, o Options, rs Runners) (*gittest.Repo, string, StepResult, Round) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
	if rs.Transcripts == nil {
		rs.Transcripts = &memTranscripts{}
	}
	res, err := Drive(context.Background(), repo.Root(), start.RunID, DefaultStages(), o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the step hands the ruthless reviewer out: %+v", res)
	}
	return repo, start.RunID, res, routeOfRole(t, repo, start.RunID, 1, RoleRuthless)
}

// machineRoots is a home holding the machine's files and the repository's
// root, for LoadTiering.
func machineRoots(t *testing.T, repo string, files map[string]string) layered.Roots {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(abcdhome.Path(home, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return layered.Roots{Repo: repo, Home: home}
}

// fakeReviewer is a reviewer route that is reachable.
type fakeReviewer struct{}

func (fakeReviewer) Review(context.Context, string) (string, error) { return "", nil }

// TestAStepWhoseReviewTheReviewerRouteTakesRecordsNoTier: with the reviewer
// route reachable, the ruthless review it takes names no tier; with the route
// named and no server for it, every review falls back to the host, so the
// review is asked for its row's tier like any host-run one.
func TestAStepWhoseReviewTheReviewerRouteTakesRecordsNoTier(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	roots := machineRoots(t, repo.Root(), map[string]string{"oracle-routing.json": `{"schema_version":1,"agents":{"ruthless-reviewer":{"tier":"economy"}}}`})
	tiers, _, err := LoadTiering(roots, &ReviewRoute{Name: oracle.ReviewRP, Reviewer: fakeReviewer{}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, res, r := driveRound(t, Options{Tiers: tiers}, Runners{Config: runnerConfig(t, "", "")})
	if res.Tier != "" || r.Tier != "" || !strings.Contains(r.Note, "rp reviewer route") {
		t.Fatalf("a review the reviewer route takes is asked for no tier: %+v %+v", res.Tier, r)
	}
	absent, _, err := LoadTiering(roots, &ReviewRoute{Name: oracle.ReviewRP, Absent: "oracle.review is rp, and no oracle.rp.url is named"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, res, r = driveRound(t, Options{Tiers: absent}, Runners{Config: runnerConfig(t, "", "")})
	if res.Tier != oracle.Economy || r.Tier != oracle.Economy || strings.Contains(r.Note, "reviewer route") {
		t.Fatalf("a reviewer route with no server takes no review, so the host's is asked for its tier: %+v %+v", res.Tier, r)
	}
}

// pointedConns is a machine whose one provider the ruthless reviewer's role
// points at; its adapter accepts no setting.
type pointedConns struct{}

func (pointedConns) Serves(oracle.Tier) (oracle.Connection, bool) { return oracle.Connection{}, false }
func (pointedConns) Named(string) (oracle.Connection, bool)       { return oracle.Connection{}, false }
func (pointedConns) Pointed(agent string) (oracle.Connection, bool) {
	return oracle.Connection{Name: "desk-llm", Models: []string{"example/model-1"},
		Roles: map[string]string{RoleRuthless: "example/model-1"}}, agent == RoleRuthless
}

// TestAPointedRoleRecordsNoTierAndItsRowNeverStopsTheStep: a role pointed at
// a provider names a model, not a tier, and the loop never sends it to that
// provider, so a routing row whose settings that provider would refuse does
// not stop the step: the role is recorded as not moving.
func TestAPointedRoleRecordsNoTierAndItsRowNeverStopsTheStep(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	tiers := tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy","settings":{"temperature":0.2}}}`)
	tiers.Conns = pointedConns{}
	_, _, res, r := driveRound(t, Options{Tiers: tiers}, Runners{Config: runnerConfig(t, "", "")})
	if res.Tier != "" || r.Tier != "" || !strings.Contains(r.Note, "pointed at desk-llm") {
		t.Fatalf("a pointed role is asked for no tier and says why: %+v %+v", res.Tier, r)
	}
}

// TestARaisedRowTakesEffectAtTheNextRound: a row raised during the run is the
// floor the next round starts from, so raising a table takes effect at once
// and a lane is never moved below the row; a lowered row never takes a lane
// back down.
func TestARaisedRowTakesEffectAtTheNextRound(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	lane := Lane{ID: "lane-1", Validation: []ValidationRound{
		{Round: 1, Fix: "fix-1", Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST", Return: "r"}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Local}}},
		{Round: 2},
	}}
	raised := &roundRouter{tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)}
	r, _, err := raised.route(lane, 2, RoleRuthless, failedBefore(lane, 2))
	if err != nil || r.Tier != oracle.Frontier || r.EscalatedFrom != oracle.Economy {
		t.Fatalf("a row raised to economy is the floor the failed round moves up from: %+v %v", r, err)
	}
	passed := Lane{ID: "lane-1", Validation: []ValidationRound{
		{Round: 1, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "SHIP", Pass: true}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Local}}},
		{Round: 2},
	}}
	if r, _, err := raised.route(passed, 2, RoleRuthless, nil); err != nil || r.Tier != oracle.Economy {
		t.Fatalf("a row raised after a passing round takes effect at the next: %+v %v", r, err)
	}
	lowered := &roundRouter{tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"local"}}`)}
	high := Lane{ID: "lane-1", Validation: []ValidationRound{
		{Round: 1, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "SHIP", Pass: true}},
			Routes: []Round{{Role: RoleRuthless, Tier: oracle.Frontier}}},
		{Round: 2},
	}}
	if r, _, err := lowered.route(high, 2, RoleRuthless, nil); err != nil || r.Tier != oracle.Frontier {
		t.Fatalf("a lowered row never takes a lane down: %+v %v", r, err)
	}
}

// TestTheGateOutputStaysBoundedAfterItsMask: the mask can lengthen text (an
// invalid byte becomes a visible escape), so the gate output is bounded again
// after it, and stored within its cap whatever bytes it carried.
func TestTheGateOutputStaysBoundedAfterItsMask(t *testing.T) {
	st := State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{{Round: 1,
		Routes: []Round{{Role: RoleRuthless, GateOutput: strings.Repeat("\xff", maxGateOutputBytes)}}}}}}}
	got := st.masked().Lanes[0].Validation[0].Routes[0].GateOutput
	if len(got) > maxGateOutputBytes {
		t.Fatalf("the masked gate output is %d bytes, over its %d-byte cap", len(got), maxGateOutputBytes)
	}
	if again := (State{Lanes: []Lane{{Validation: []ValidationRound{{Routes: []Round{{GateOutput: got}}}}}}}).masked(); again.Lanes[0].Validation[0].Routes[0].GateOutput != got {
		t.Fatal("the mask and its bound are idempotent, so a stored gate output round-trips")
	}
}

// TestLoadTieringCarriesTheProviderDiagnostics: the provider configuration's
// non-fatal reports are handed to the caller beside the tables', as a
// delegating verb prints them.
func TestLoadTieringCarriesTheProviderDiagnostics(t *testing.T) {
	roots := machineRoots(t, t.TempDir(), map[string]string{"config.json": `{"oracle":{"roles":{"nobody-agent":"x/y"}}}`})
	_, diags, err := LoadTiering(roots, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 || !strings.Contains(diags[0], "oracle adapter") || !strings.Contains(diags[0], "nobody-agent") {
		t.Fatalf("the provider configuration's diagnostic is carried: %q", diags)
	}
}

// TestAStepOnARunnerRecordsThatItDoesNotMove: a role the machine's runner
// configuration hands to a runner names a runner and a model, so the step asks
// it for no tier and its round says it does not move.
func TestAStepOnARunnerRecordsThatItDoesNotMove(t *testing.T) {
	cfg := runnerConfig(t, `{"roles":{"ruthless-reviewer":{"runner":"opencode"}}}`, "")
	repo := briefRepo(t, agentsMarked)
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)}
	_, _, res, r := driveRound(t, o, Runners{Config: cfg})
	if res.Tier != "" || r.Tier != "" || !strings.Contains(r.Note, "opencode runner") || !strings.Contains(r.Note, "does not move") {
		t.Fatalf("a role on a runner is asked for no tier and says why: %+v %+v", res.Tier, r)
	}
}

// TestARunnerRoutesNoteIsMaskedWhenStored: a round's note carries the --route
// text the person typed, so it is stored masked, its home directory redacted,
// as every other free text of the state is.
func TestARunnerRoutesNoteIsMaskedWhenStored(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	// The fixture points HOME at a directory of its own; that is the home the
	// mask redacts.
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == "/" {
		t.Skip("no home directory to redact")
	}
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
	o := Options{Tiers: tiering(t, repo, `{"ruthless-reviewer":{"tier":"economy"}}`)}
	text := "ruthless-reviewer=runner:opencode@example/‮model " + home + "/secret"
	o.runnerRoute = func(role string) (string, string) {
		if role == RoleRuthless {
			return "opencode", text
		}
		return "", ""
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	raw := string(stateBytes(t, repo.Root(), start.RunID))
	note := routeOfRole(t, repo, start.RunID, 1, RoleRuthless).Note
	for _, got := range []string{raw, note} {
		if strings.ContainsRune(got, '‮') || strings.Contains(got, home+"/secret") {
			t.Fatalf("the note's --route text is stored masked, its home redacted:\n%s", note)
		}
	}
	if !strings.Contains(note, "/secret") {
		t.Fatalf("the note keeps the --route text it masks: %q", note)
	}
}

// TestLoadTieringRefusesAnUnreadableTableAtRouting: a routing table the step
// cannot read decides which tier a round asks for, so it is refused at the
// routing stage before anything moves.
func TestLoadTieringRefusesAnUnreadableTableAtRouting(t *testing.T) {
	roots := machineRoots(t, t.TempDir(), map[string]string{"oracle-routing.json": `{"schema_version":1,"agents":{"scribe":{"tier":"cheap"}}}`})
	_, _, err := LoadTiering(roots, nil)
	if r := mustRefusal(t, err); r.Stage != StageRouting || !strings.Contains(r.Reason, "cheap") {
		t.Fatalf("an unreadable table is refused at the routing stage, naming the fault: %+v", r)
	}
}
