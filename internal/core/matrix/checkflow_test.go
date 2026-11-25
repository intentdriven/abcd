package matrix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// checkTrial is copy key's trial as its hand-back writes it: passed, failed
// with its reason, or set apart by the compliance check.
func checkTrial(t *testing.T, rec RunRecord, key TrialKey, status TrialStatus) TrialRecord {
	t.Helper()
	pin := pinOf(t, rec, key.Cell)
	tr := TrialRecord{SchemaVersion: TrialSchema, Run: rec.RunID, Key: key, Kind: KindCheck, FrozenCommit: rec.FrozenCommit,
		Source: pin.Source, Runner: pin.Runner, HarnessVersion: pin.HarnessVersion, ModelAsked: pin.Model, ModelServed: pin.Model,
		Status: status, StartedAt: t0, FinishedAt: t0.Add(time.Minute), Compliance: compliantResult()}
	if pin.Source.Metered() {
		tr.Cost = &oracle.Cost{Amount: 0.1, Unit: oracle.CostUSD}
	}
	switch status {
	case StatusFailed:
		tr.Reason = "the copy's answer is refused, so the copy failed"
	case StatusExcluded:
		tr.Reason, tr.Compliance = "the copy read another copy's clone", resultWith(RuleOwnCopyOnly, StateBroken)
	}
	return tr
}

// checkCopyAt is one check copy of a flow's fixture: its key, its answer and
// the status its trial is written with.
type checkCopyAt struct {
	key    TrialKey
	answer []byte
	status TrialStatus
}

// checkFlowLab is a recorded run of the guard chapter's check, each edit made
// to its inputs first, with each copy's findings collected from its answer and
// its trial written, as its hand-back leaves them.
func checkFlowLab(t *testing.T, copies []checkCopyAt, edits ...func(*runInputs)) (*gittest.Repo, string, lab.Minted, RunRecord) {
	t.Helper()
	r, home, m, rec, store := checkLab(t, edits...)
	for _, c := range copies {
		if _, _, err := collect(t, r, m, rec, store, c.key, c.answer); err != nil {
			t.Fatalf("collecting %s: %v", c.key, err)
		}
		if err := writeTrial(r.Root(), m.ID, checkTrial(t, rec, c.key, c.status)); err != nil {
			t.Fatalf("writeTrial(%s): %v", c.key, err)
		}
	}
	return r, home, m, rec
}

// hostJudges makes goodDeclaration's run one with a matcher and a proposer
// on the host's route, outside the matrix, its host builder moved to an
// OpenRouter model so that no builder is the host.
func hostJudges(in *runInputs) {
	in.Declaration = []byte(strings.Replace(withJudges(`"matcher": {"id": "match-host", "route": "host"},
  "proposer": {"id": "propose-host", "route": "host"}`),
		`{"id": "host-big", "route": "host"}`, `{"id": "or-big", "route": "builder=runner:opencode@openrouter/vendor/big"}`, 1))
	in.Cells[0] = CellPin{Cell: "or-big", Role: RoleBuilder, Source: SourceOpenRouter, Runner: "opencode", Model: "openrouter/vendor/big",
		HarnessVersion: "1.2.3", ProviderPin: &ProviderPin{Order: []string{"provider-a"}}}
	in.Cells = append(in.Cells,
		CellPin{Cell: "match-host", Role: RoleMatcher, Source: SourceHost, Runner: "host"},
		CellPin{Cell: "propose-host", Role: RoleProposer, Source: SourceHost, Runner: "host"})
}

// checkState is the path of a check's record in the run's state.
func checkState(home string, m lab.Minted, run, name string) string {
	return filepath.Join(runDir(home, m, run), "check", checkTask, name)
}

func absent(t *testing.T, what, p string) {
	t.Helper()
	if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("%s is written: %v", what, err)
	}
}

// With no matcher and no proposer declared, a check is judged from the command
// line in two phases: the packet is drawn from the copies whose trials count,
// shuffled, with no model named and no verdict proposed; the findings no exact
// key joins stay rows of their own; the verdicts lock once; the table is
// refused until they do; the matcher's agreement is not estimable; and the
// release gates' receipts and the ledger are left as they were (spec step 9).
func TestACheckIsJudgedFromTheCommandLineInTwoPhases(t *testing.T) {
	const registry = "The guard reads its registry from the plugin root."
	r, home, m, rec := checkFlowLab(t, []checkCopyAt{
		{TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1}, dfAnswer(t, registry, "public:docs/guide.md:Run ahoy first."), StatusPassed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}, dfAnswer(t, "The guard  reads its registry\tfrom the plugin root.",
			"public:docs/guide.md:Always run ahoy before anything else."), StatusPassed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 2}, []byte("{"), StatusFailed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 3}, dfAnswer(t, "It refuses a hazard"), StatusExcluded},
	})
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, nil); err == nil {
		t.Fatal("a table was built before any packet was drawn")
	}
	j, err := DrawCheck(r.Root(), m.ID, rec.RunID, checkTask)
	if err != nil {
		t.Fatalf("DrawCheck: %v", err)
	}
	// The copy set apart is in no comparison; the failed one is, unanswered.
	if j.Stage != CheckAwaitingVerdicts || j.Copies != 3 || j.Leftovers != 2 || j.Packet == nil || len(j.Packet.Items) != 3 ||
		j.MatcherRequest != nil || j.Matcher != nil || j.Proposer != nil {
		t.Fatalf("judging = %+v; want the packet of three findings over three copies, two left apart", j)
	}
	js, _ := json.Marshal(j.Packet)
	for _, leak := range []string{"host-big", "or-cheap", "openrouter", rec.RunID, `"` + checkTask + `"`, "valid", "proposed",
		"raised", "matched", "members", "hazard"} {
		if bytes.Contains(js, []byte(leak)) {
			t.Errorf("the packet carries %q: %s", leak, js)
		}
	}
	// Asked again, the verb shows the same packet and draws nothing.
	again, err := DrawCheck(r.Root(), m.ID, rec.RunID, checkTask)
	if err != nil || again.PacketDigest != j.PacketDigest || again.Stage != CheckAwaitingVerdicts {
		t.Fatalf("a second draw = %+v, %v; want the packet drawn, shown again", again, err)
	}
	// The table is refused until the verdicts lock, and a proposal for a
	// proposer the run never declared is refused.
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, nil); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("the table before the verdicts = %v; want a refusal saying they are not locked", err)
	}
	absent(t, "the table", checkState(home, m, rec.RunID, "table.json"))
	k := CheckKey{Packet: j.PacketDigest}
	verdicts := checkVerdicts(t, *j.Packet, k, true, map[string]bool{j.Packet.Items[0].Code: true})
	if _, err := RecordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, verdicts); err != nil {
		t.Fatalf("RecordCheckVerdicts: %v", err)
	}
	if _, err := RecordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, *j.Packet, k, true, nil)); err == nil ||
		!strings.Contains(err.Error(), "locked already") {
		t.Fatalf("a second verdict file = %v; want it refused, the first kept", err)
	}
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{}`), nil); err == nil || !strings.Contains(err.Error(), "no proposer") {
		t.Fatalf("proposals with no proposer declared = %v; want them refused", err)
	}
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, []byte(`{}`)); err == nil || !strings.Contains(err.Error(), "no matcher") {
		t.Fatalf("labels with no matcher declared = %v; want them refused", err)
	}
	table, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, nil)
	if err != nil {
		t.Fatalf("RecordCheckTable: %v", err)
	}
	alone := 0
	for _, row := range table.Rows {
		if row.Proposed != nil {
			t.Errorf("row %s proposes a verdict with no proposer declared", row.Code)
		}
		if row.MatchedBy == MatchedAlone {
			alone++
		}
	}
	if len(table.Rows) != 3 || alone != 2 {
		t.Fatalf("rows = %+v; want three, the two leftovers rows of their own", table.Rows)
	}
	if a := table.Matcher; a == nil || a.Agreed.Estimable() || a.Leftovers != 2 || !strings.Contains(a.NotEstimable, "no matcher cell is declared") {
		t.Fatalf("matcher = %+v; want its agreement not estimable, no matcher declared", table.Matcher)
	}
	for _, s := range table.Scores {
		if s.Cell == "or-cheap" && (s.Copies != 1 || s.Unanswered != 1) {
			t.Errorf("or-cheap's column = %+v; want its one answered copy and its failed one, the excluded one in neither", s)
		}
	}
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, nil); err == nil || !strings.Contains(err.Error(), "built once") {
		t.Fatalf("a second table = %v; want it refused", err)
	}
	if j, err := DrawCheck(r.Root(), m.ID, rec.RunID, checkTask); err != nil || j.Stage != CheckTabled {
		t.Fatalf("the judging once tabled = %+v, %v", j, err)
	}
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil || len(rep.Check) != 1 || rep.Check[0].Table == nil {
		t.Fatalf("the report's check = %+v, %v; want the table", rep.Check, err)
	}
	if err := gates.unchanged(r.Root()); err != nil {
		t.Fatalf("judging the check changed what the release gates read: %v", err)
	}
	// A task that is not one of the run's checks is refused.
	if _, err := DrawCheck(r.Root(), m.ID, rec.RunID, "doc-page"); err == nil || !strings.Contains(err.Error(), "no check task") {
		t.Fatalf("a writing task drawn as a check = %v", err)
	}
}

// With a matcher declared, the findings no exact key joins, and only those, go
// to the matcher; its answer is untrusted input, refused whole with nothing
// recorded when it names a finding it was not shown, and taken once; the
// proposer's verdicts are required with the table; and the matcher's
// agreement on fewer labelled pairs than the floor is not estimable.
func TestACheckMatcherIsAskedOnlyTheLeftoversAndItsAnswerIsRefusedWhole(t *testing.T) {
	const shared = "The guard reads its registry from the plugin root."
	r, home, m, rec := checkFlowLab(t, []checkCopyAt{
		{TrialKey{Task: checkTask, Cell: "or-big", Repeat: 1}, dfAnswer(t, shared, "public:docs/guide.md:Run ahoy first."), StatusPassed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}, dfAnswer(t, shared, "public:docs/guide.md:Always run ahoy before anything else."), StatusPassed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 2}, dfAnswer(t, shared, "public:docs/guide.md:Ahoy is optional."), StatusPassed},
	}, hostJudges)
	j, err := DrawCheck(r.Root(), m.ID, rec.RunID, checkTask)
	if err != nil {
		t.Fatalf("DrawCheck: %v", err)
	}
	if j.Stage != CheckAwaitingMatcher || j.Packet != nil || j.MatcherRequest == nil || len(j.MatcherRequest.Items) != 3 ||
		j.Matcher == nil || j.Matcher.Cell != "match-host" || j.Matcher.AwaitingRunner {
		t.Fatalf("judging = %+v; want the matcher's request about the three leftovers", j)
	}
	js, _ := json.Marshal(j.MatcherRequest)
	for _, leak := range []string{"or-big", "or-cheap", "registry from the plugin root", rec.RunID} {
		if bytes.Contains(js, []byte(leak)) {
			t.Errorf("the matcher's request carries %q: %s", leak, js)
		}
	}
	absent(t, "the packet", checkState(home, m, rec.RunID, "packet.json"))
	if _, err := RecordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{}`)); err == nil {
		t.Fatal("verdicts were recorded before the packet was drawn")
	}
	id := map[string]string{}
	for _, it := range j.MatcherRequest.Items {
		id[it.Ends[0].Quote] = it.ID
	}
	x, y := id["Run ahoy first."], id["Always run ahoy before anything else."]
	// An answer naming a finding it was not shown is refused whole.
	if _, err := RecordCheckMatcher(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{"schema_version":1,"same":[["`+x+`","`+y+`"],["`+x+`","m-ffffffff"]]}`)); err == nil ||
		!strings.Contains(err.Error(), "nothing of it is recorded") {
		t.Fatalf("an answer naming an unknown finding = %v; want it refused whole", err)
	}
	absent(t, "the matcher's answer", checkState(home, m, rec.RunID, "matcher.json"))
	absent(t, "the packet", checkState(home, m, rec.RunID, "packet.json"))
	j, err = RecordCheckMatcher(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{"schema_version":1,"same":[["`+x+`","`+y+`"]]}`))
	if err != nil {
		t.Fatalf("RecordCheckMatcher: %v", err)
	}
	if j.Stage != CheckAwaitingVerdicts || j.Packet == nil || len(j.Packet.Items) != 3 || len(j.Sample) != 3 {
		t.Fatalf("judging = %+v; want three rows, the matcher's pair joined, and its sample of three pairs", j)
	}
	if _, err := RecordCheckMatcher(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{"schema_version":1,"same":[]}`)); err == nil ||
		!strings.Contains(err.Error(), "recorded already") {
		t.Fatalf("a second matcher answer = %v; want it refused", err)
	}
	k := CheckKey{Packet: j.PacketDigest}
	if _, err := RecordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, *j.Packet, k, true, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, nil, nil); err == nil || !strings.Contains(err.Error(), "propose-host") {
		t.Fatalf("a table without the declared proposer's verdicts = %v; want it refused", err)
	}
	proposals := CheckProposalFile{SchemaVersion: CheckJudgingSchema, Packet: j.PacketDigest}
	for _, it := range j.Packet.Items {
		proposals.Proposed = append(proposals.Proposed, CheckProposal{Code: it.Code, Valid: true, Reason: "the code agrees"})
	}
	pdata, _ := json.Marshal(proposals)
	outside := fmt.Sprintf(`{"schema_version":1,"labels":[{"a":%q,"b":%q,"same":true}]}`, x, x+"0")
	if _, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, pdata, []byte(outside)); err == nil || !strings.Contains(err.Error(), "labels are refused") {
		t.Fatalf("labels on a finding the matcher was not shown = %v; want them refused", err)
	}
	labels := fmt.Sprintf(`{"schema_version":1,"labels":[{"a":%q,"b":%q,"same":true},{"a":%q,"b":%q,"same":false}]}`, j.Sample[0][0], j.Sample[0][1], j.Sample[1][0], j.Sample[1][1])
	table, err := RecordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, pdata, []byte(labels))
	if err != nil {
		t.Fatalf("RecordCheckTable: %v", err)
	}
	joined := 0
	for _, row := range table.Rows {
		if row.Proposed == nil || !row.Proposed.Valid {
			t.Errorf("row %s carries no proposed verdict", row.Code)
		}
		if row.MatchedBy == MatchedByMatcher {
			joined++
		}
	}
	if joined != 1 {
		t.Fatalf("rows = %+v; want the matcher's pair joined in one row", table.Rows)
	}
	if a := table.Matcher; a == nil || a.Agreed.Of != 2 || a.Joined != 1 || a.Leftovers != 3 || !strings.Contains(a.NotEstimable, "under the floor") {
		t.Fatalf("matcher = %+v; want two labelled pairs, under the floor, not estimable", table.Matcher)
	}
}

// The matcher's agreement is reported as a figure from the floor of labelled
// pairs up, and only on pairs of the sample drawn for it.
func TestTheMatchersAgreementIsReportedFromTheFloorOfLabelledPairs(t *testing.T) {
	st := &checkMatchState{Request: MatcherRequest{SchemaVersion: 1}}
	for i := range 5 {
		st.Request.Items = append(st.Request.Items, MatcherItem{ID: fmt.Sprintf("m-%08x", i)})
	}
	mr := &checkMatcherRecord{Pairs: [][2]string{{"m-00000000", "m-00000001"}}}
	var labels []string
	for i := range 5 {
		for k := i + 1; k < 5; k++ {
			a, b := st.Request.Items[i].ID, st.Request.Items[k].ID
			mr.Sample = append(mr.Sample, [2]string{a, b})
			labels = append(labels, fmt.Sprintf(`{"a":%q,"b":%q,"same":%v}`, a, b, i == 0 && k == 1))
		}
	}
	j := CheckJudging{Task: checkTask, Matcher: &JudgeCell{Cell: "match-host", Source: SourceHost}, Packet: &CheckPacket{}}
	a, err := agreementOf(j, st, mr, []byte(`{"schema_version":1,"labels":[`+strings.Join(labels, ",")+`]}`))
	if err != nil {
		t.Fatalf("agreementOf: %v", err)
	}
	if a.Agreed != (Rate{Hits: matcherLabelFloor, Of: matcherLabelFloor}) || a.NotEstimable != "" {
		t.Fatalf("agreement = %+v; want all ten labelled pairs agreed, reported as a figure", a)
	}
	a, err = agreementOf(j, st, mr, []byte(`{"schema_version":1,"labels":[`+strings.Join(labels[:matcherLabelFloor-1], ",")+`]}`))
	if err != nil || !strings.Contains(a.NotEstimable, "under the floor") {
		t.Fatalf("agreement under the floor = %+v, %v; want it not estimable", a, err)
	}
	if a, err := agreementOf(j, st, mr, nil); err != nil || !strings.Contains(a.NotEstimable, "no pair") {
		t.Fatalf("agreement with no labels = %+v, %v; want it not estimable", a, err)
	}
	mr.Sample = mr.Sample[1:]
	if _, err := agreementOf(j, st, mr, []byte(`{"schema_version":1,"labels":[`+labels[0]+`]}`)); err == nil || !strings.Contains(err.Error(), "sample does not hold") {
		t.Fatalf("a label on a pair outside the sample = %v; want it refused", err)
	}
}

// A matcher or a proposer on a runner route answers through a launcher, and
// nothing stands in for it: its request is drawn and waits, and an answer the
// host hands in for it is refused.
func TestARunnerMatcherWaitsForItsLauncher(t *testing.T) {
	const shared = "The guard reads its registry from the plugin root."
	runnerMatcher := func(in *runInputs) {
		in.Declaration = []byte(withJudges(`"matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/judge"}`))
		in.Cells = append(in.Cells, CellPin{Cell: "match-or", Role: RoleMatcher, Source: SourceOpenRouter, Runner: "opencode",
			Model: "openrouter/vendor/judge", HarnessVersion: "1.2.3", ProviderPin: &ProviderPin{Order: []string{"provider-a"}}})
	}
	r, _, m, rec := checkFlowLab(t, []checkCopyAt{
		{TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1}, dfAnswer(t, shared, "public:docs/guide.md:Run ahoy first."), StatusPassed},
		{TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}, dfAnswer(t, shared, "public:docs/guide.md:Ahoy is optional."), StatusPassed},
	}, runnerMatcher)
	j, err := DrawCheck(r.Root(), m.ID, rec.RunID, checkTask)
	if err != nil {
		t.Fatalf("DrawCheck: %v", err)
	}
	if j.Stage != CheckAwaitingMatcher || j.Matcher == nil || !j.Matcher.AwaitingRunner {
		t.Fatalf("judging = %+v; want the request waiting for the runner matcher", j)
	}
	a, b := j.MatcherRequest.Items[0].ID, j.MatcherRequest.Items[1].ID
	if _, err := RecordCheckMatcher(r.Root(), m.ID, rec.RunID, checkTask, []byte(`{"schema_version":1,"same":[["`+a+`","`+b+`"]]}`)); err == nil ||
		!strings.Contains(err.Error(), "runner cell") {
		t.Fatalf("a host answer for a runner matcher = %v; want it refused", err)
	}
}
