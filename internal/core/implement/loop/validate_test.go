package loop

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// withLanding is the production stages with a fake landing that merges the
// lane's branch into the default branch, as a landed pull request would, so a
// later lane is cut from a base that carries it. The landing is piece 9's.
func withLanding(repo *gittest.Repo) Stages {
	stages := DefaultStages()
	for i := range stages {
		if stages[i].Name == StageLand {
			stages[i].Run = func(c Context, lane *Lane) (Outcome, error) {
				repo.Git("merge", "--no-ff", "-q", "-m", "land "+lane.ID, lane.Branch)
				return Outcome{Note: "landed " + lane.Branch}, nil
			}
		}
	}
	return stages
}

// stepTo advances the run with stages until its current lane's next stage is
// want and it awaits nothing.
func stepTo(t *testing.T, repo *gittest.Repo, runID string, stages Stages, want Stage) Lane {
	t.Helper()
	for range 4 * len(Sequence) {
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		if i := st.current(); i >= 0 && st.Lanes[i].Stage == want && st.Lanes[i].awaiting() == nil {
			return st.Lanes[i]
		}
		if _, err := Advance(repo.Root(), runID, stages, Options{}); err != nil {
			t.Fatalf("advancing to %s: %v", want, err)
		}
	}
	t.Fatalf("the lane never reached %s", want)
	return Lane{}
}

// currentLane is the lane the loop works on.
func currentLane(t *testing.T, repo *gittest.Repo, runID string) Lane {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	i := st.current()
	if i < 0 {
		t.Fatal("no lane is in progress")
	}
	return st.Lanes[i]
}

// implemented drives the current lane through its implement stage: one commit
// in its worktree and a receipt that verifies. It returns the lane at validate.
func implemented(t *testing.T, repo *gittest.Repo, runID string, stages Stages, file string) Lane {
	t.Helper()
	stepTo(t, repo, runID, stages, StageImplement)
	res, err := Advance(repo.Root(), runID, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, l.ID)
	sha := laneCommit(t, repo, l, file)
	path := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, sha))
	if _, err := Receipt(repo.Root(), runID, path, stages, Options{}); err != nil {
		t.Fatal(err)
	}
	l = currentLane(t, repo, runID)
	if l.Stage != StageValidate || l.HeadSHA != sha {
		t.Fatalf("a verified receipt hands the lane to its validators at its head: %+v", l)
	}
	return l
}

// reviewerReturn is a validator's return in its agent definition's shape.
func reviewerReturn(verdict string) string {
	return "### Analysis\n\nRead the lane's diff; nothing else.\n\n### Findings\n\n- none survived refutation\n\n### Verdict\n\n- **" +
		verdict + "** — as stated.\n"
}

var provenanceRe = regexp.MustCompile(`(?m)^- (rubric_hash|prompt_hash): (sha256:[0-9a-f]{64})$`)

// auditorVerdict is an intent-auditor's verdict on the request it was handed:
// ac-1 judged verdict, echoing the request's receipt and provenance.
func auditorVerdict(t *testing.T, request, verdict string) string {
	t.Helper()
	req, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, m := range provenanceRe.FindAllStringSubmatch(string(req), -1) {
		hashes[m[1]] = m[2]
	}
	rcp := regexp.MustCompile(`receipt_id: (rcp-[0-9a-f]{12})`).FindStringSubmatch(string(req))
	if len(hashes) != 2 || rcp == nil {
		t.Fatalf("the request states its receipt and both hashes:\n%s", req)
	}
	rollup := map[string]int{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}
	rollup[verdict] = 1
	v := map[string]any{
		"_type": "abcd/intent-fidelity-verdict/v1", "receipt_id": rcp[1],
		"verifier":           map[string]any{"id": "intent-auditor", "version": "a-model"},
		"policy":             map[string]any{"rubric_hash": hashes["rubric_hash"], "prompt_hash": hashes["prompt_hash"]},
		"input_attestations": []any{},
		"criteria": []any{map[string]any{"criterion_id": "ac-1", "verdict": verdict, "rationale": "read the delivery",
			"evidence": []any{map[string]any{"ref": "one.txt:1", "quote": "one.txt"}}}},
		"acceptance_rollup": rollup,
		"gap_audit":         map[string]any{"honoured": []any{}, "diverged": []any{}, "missing": []any{}},
		"scope_conditions":  []any{},
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// handBack takes the validate stage's next hand-out, checks it names role, and
// hands back body as that agent's return. It returns the receipt's result.
func handBack(t *testing.T, repo *gittest.Repo, runID string, stages Stages, role, body string) StepResult {
	t.Helper()
	res, err := Advance(repo.Root(), runID, stages, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != role || res.PerformedStage != "" {
		t.Fatalf("the validate stage hands the lane to a fresh %s: %+v", role, res)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	if role == RoleAuditor {
		body = auditorVerdict(t, filepath.Join(filepath.Dir(path), AuditRequestFileName), body)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Receipt(repo.Root(), runID, path, stages, Options{})
	if err != nil {
		t.Fatalf("the %s's return: %v", role, err)
	}
	return got
}

// passRound hands back a passing verdict from every validator the round names.
func passRound(t *testing.T, repo *gittest.Repo, runID string, stages Stages, roles ...string) {
	t.Helper()
	pass := map[string]string{RoleRuthless: reviewerReturn("SHIP"), RoleSecurity: reviewerReturn("APPROVE"), RoleAuditor: "MET"}
	for _, role := range roles {
		handBack(t, repo, runID, stages, role, pass[role])
	}
}

// lanesRequests lists the audit requests under a lane's directory.
func lanesRequests(t *testing.T, repo *gittest.Repo, runID, laneID string) []string {
	t.Helper()
	var out []string
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, laneID)
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() == AuditRequestFileName {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// TestTheValidatorsAreFreshAgentsAndOnlyTheLoopRecordsAVerdict is criteria 5 and
// 12 on the lane that closes the spec: the validate stage hands the lane's diff
// to a fresh ruthless reviewer, then a fresh security reviewer, then the
// intent-auditor, one at a time; each return's verdict is the one the loop
// parses and records, the lane stays at validate until the last, and a round
// that passes advances the lane to its landing.
func TestTheValidatorsAreFreshAgentsAndOnlyTheLoopRecordsAVerdict(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")

	res, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the first validator is a fresh ruthless reviewer: %+v %v", res, err)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"fresh", "did not implement", l.BaseSHA + ".." + l.HeadSHA, l.Branch, "### Verdict", "SHIP", "FIX FIRST",
		filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))} {
		if !strings.Contains(string(brief), want) {
			t.Fatalf("the reviewer's brief names %q:\n%s", want, brief)
		}
	}
	// The ruthless reviewer is out; its return is handed back before the next
	// step, which would otherwise hand the security reviewer out beside it.
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt)), []byte(reviewerReturn("SHIP")), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Receipt(repo.Root(), id, filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt)), stages, Options{})
	if err != nil || got.PerformedStage != "" || got.Stage != StageValidate || got.Awaiting != nil {
		t.Fatalf("a validator's return leaves the lane at validate for the next: %+v", got)
	}
	st, _ := ReadState(repo.Root(), id)
	if v := st.Lanes[0].Validation; len(v) != 1 || len(v[0].Validators) != 3 || v[0].Validators[0].Verdict != "SHIP" || !v[0].Validators[0].Pass {
		t.Fatalf("the loop records the verdict it parsed from the return: %+v", v)
	}
	handBack(t, repo, id, stages, RoleSecurity, reviewerReturn("APPROVE"))
	handBack(t, repo, id, stages, RoleAuditor, "MET")

	done, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil || done.PerformedStage != StageValidate || done.Stage != StageLand {
		t.Fatalf("a passing round completes the validate stage: %+v %v", done, err)
	}
	st, _ = ReadState(repo.Root(), id)
	var record strings.Builder
	for _, e := range st.Record {
		record.WriteString(e.Note + "\n")
	}
	for _, want := range []string{"ruthless-reviewer's verdict SHIP", "security-reviewer's verdict APPROVE", "intent-auditor's verdict MET"} {
		if !strings.Contains(record.String(), want) {
			t.Fatalf("the run record names %q:\n%s", want, record.String())
		}
	}
	a := st.Lanes[0].Validation[0].Validators[2].Audit
	if a == nil || a.BaseSHA != l.BaseSHA || a.HeadSHA != l.HeadSHA || !strings.HasPrefix(a.ReceiptID, "rcp-") || a.Worst != "MET" {
		t.Fatalf("the audit's receipt and range are the run's, for the close to consume: %+v", a)
	}
	// The landing (piece 9) takes the lane from here, one step at a time.
	res, err = Advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Stage != StageLand || res.PerformedStage != "" {
		t.Fatalf("the landing's first step leaves the lane at land: %+v %v", res, err)
	}
}

// TestTheAuditRunsOnceOnTheClosingLaneOverTheWholeDelivery is ruling AI: a
// lane whose landing does not close the spec takes no audit step, and the lane
// whose landing does takes it once, over the whole delivery — the run's own
// lanes' changes, lane by lane, each lane's head against its base (or the
// default-branch sha it last merged in), never one range from the first lane's
// base, which after a sync would carry outside work (spc-2609202134341288).
func TestTheAuditRunsOnceOnTheClosingLaneOverTheWholeDelivery(t *testing.T) {
	repo := steppedBriefRepo(t, "1. The parser\n2. The loop\n")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, withLanding(repo)

	first := implemented(t, repo, id, stages, "one.txt")
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	res, err := Advance(repo.Root(), id, stages, Options{})
	if err != nil || res.PerformedStage != StageValidate {
		t.Fatalf("a lane that does not close the spec completes its validation without an audit: %+v %v", res, err)
	}
	if got := lanesRequests(t, repo, id, first.ID); len(got) != 0 {
		t.Fatalf("no audit request for a lane that does not close the spec: %v", got)
	}
	st, _ := ReadState(repo.Root(), id)
	if v := st.Lanes[0].Validation[0].Validators; len(v) != 2 {
		t.Fatalf("the non-closing lane's validators are the two reviewers: %+v", v)
	}
	if _, err := Advance(repo.Root(), id, stages, Options{}); err != nil {
		t.Fatal(err)
	}

	closing := implemented(t, repo, id, stages, "two.txt")
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity, RoleAuditor)
	reqs := lanesRequests(t, repo, id, closing.ID)
	if len(reqs) != 1 {
		t.Fatalf("the closing lane takes the audit once: %v", reqs)
	}
	req, err := os.ReadFile(reqs[0])
	if err != nil {
		t.Fatal(err)
	}
	st, _ = ReadState(repo.Root(), id)
	if whole := st.Lanes[0].BaseSHA + ".." + closing.HeadSHA; strings.Contains(string(req), whole) {
		t.Fatalf("the delivery is lane by lane, not the range %s:\n%s", whole, req)
	}
	for _, want := range []string{st.Lanes[0].BaseSHA + ".." + st.Lanes[0].HeadSHA, closing.BaseSHA + ".." + closing.HeadSHA,
		"lane-1", "lane-2", "## Acceptance Criteria", "## Provenance"} {
		if !strings.Contains(string(req), want) {
			t.Fatalf("the audit request carries %q:\n%s", want, req)
		}
	}
	a := st.Lanes[1].Validation[0].Validators[2].Audit
	if a == nil || a.BaseSHA != closing.BaseSHA || a.HeadSHA != closing.HeadSHA {
		t.Fatalf("the audit's own range is the closing lane's diff: %+v", a)
	}
	if res, err := Advance(repo.Root(), id, stages, Options{}); err != nil || res.PerformedStage != StageValidate {
		t.Fatalf("the closing lane's passing round completes its validation: %+v %v", res, err)
	}
}

// fixed hands the round's findings to the fresh implementer the stage names and
// hands back its receipt, naming commits (the lane's own when it made none).
func fixed(t *testing.T, repo *gittest.Repo, runID string, stages Stages, report string, commits ...string) {
	t.Helper()
	res, err := Advance(repo.Root(), runID, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("a round that did not pass hands its findings to a fresh implementer: %+v %v", res, err)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"fresh", "did not pass", "reject it in writing"} {
		if !strings.Contains(string(brief), want) {
			t.Fatalf("the fix brief names %q:\n%s", want, brief)
		}
	}
	l := currentLane(t, repo, runID)
	laneDir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, l.ID)
	rel, err := filepath.Rel(laneDir, filepath.Dir(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))))
	if err != nil {
		t.Fatal(err)
	}
	rel = filepath.ToSlash(rel)
	for name, body := range map[string]string{rel + "/" + ReportFileName: report, rel + "/" + DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: runID, Lane: l.ID, Branch: l.Branch, Commits: commits,
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: rel + "/" + DoDFileName}, Report: rel + "/" + ReportFileName}
	data, _ := json.Marshal(rc)
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Receipt(repo.Root(), runID, path, stages, Options{})
	if err != nil || got.PerformedStage != "" || got.Stage != StageValidate {
		t.Fatalf("a verified fix receipt returns the lane to its validators: %+v %v", got, err)
	}
}

// TestAFindingIsAppliedByAFreshImplementerOrRejectedInWriting is criterion 5's
// findings: a round a validator did not pass hands its returns to a fresh
// implementer, who applies each finding with a commit or rejects it in writing
// in its report; the next round then judges the lane's head afresh, every
// validator again, so no verdict stands over a head it did not read.
func TestAFindingIsAppliedByAFreshImplementerOrRejectedInWriting(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")

	// Round 1: a finding, applied with a commit.
	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
	passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
	fix := laneCommit(t, repo, l, "fix.txt")
	fixed(t, repo, id, stages, "applied the finding in "+fix+"\n", fix)
	if h := currentLane(t, repo, id).HeadSHA; h != fix {
		t.Fatalf("an applied finding moves the lane's head: %s, want %s", h, fix)
	}

	// Round 2: every validator again over the new head; a finding rejected in writing.
	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("SHIP"))
	handBack(t, repo, id, stages, RoleSecurity, reviewerReturn("BLOCK"))
	handBack(t, repo, id, stages, RoleAuditor, "MET")
	fixed(t, repo, id, stages, "Rejected in writing: the finding names a path no caller reaches.\n", fix)

	// Round 3 judges the same head again, and passes.
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity, RoleAuditor)
	if res, err := Advance(repo.Root(), id, stages, Options{}); err != nil || res.PerformedStage != StageValidate {
		t.Fatalf("a passing round completes the stage: %+v %v", res, err)
	}
	st, _ := ReadState(repo.Root(), id)
	v := st.Lanes[0].Validation
	if len(v) != 3 || v[0].HeadSHA != l.HeadSHA || v[1].HeadSHA != fix || v[2].HeadSHA != fix || v[0].Fix == "" || v[1].Fix == "" || v[2].Fix != "" {
		t.Fatalf("each round is recorded with the head it judged and the fix it handed to: %+v", v)
	}
}

// TestAReportCarryingAVerdictIsRefusedAtTheAdvance is the itd-58 invariant's
// refusal: only the loop writes a verdict, so a lane report stating one is
// refused when the stage would advance, naming the report, and nothing moves;
// the report without it lets the advance proceed.
func TestAReportCarryingAVerdictIsRefusedAtTheAdvance(t *testing.T) {
	for _, line := range []string{"Verdict: SHIP", "**Verdict:** APPROVE", "- verdict: MET", "### Verdict\n\n- **SHIP** — mine."} {
		t.Run(line, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			id, stages := start.RunID, DefaultStages()
			implemented(t, repo, id, stages, "one.txt")
			report := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, "lane-1", ReportFileName)
			if err := os.WriteFile(report, []byte("built it\n\n"+line+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			passRound(t, repo, id, stages, RoleRuthless, RoleSecurity, RoleAuditor)
			before := stateBytes(t, repo.Root(), id)
			_, err = Advance(repo.Root(), id, stages, Options{})
			r := mustRefusal(t, err)
			if r.Stage != string(StageValidate) || !strings.Contains(r.Reason, RunRelDir+"/"+id+"/lane-1/"+ReportFileName) || !strings.Contains(r.Reason, "verdict") {
				t.Fatalf("the refusal names the report carrying a verdict: %+v", r)
			}
			if string(before) != string(stateBytes(t, repo.Root(), id)) {
				t.Fatal("a refused advance moves nothing")
			}
			if err := os.WriteFile(report, []byte("built it; the reviewers judge it\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if res, err := Advance(repo.Root(), id, stages, Options{}); err != nil || res.PerformedStage != StageValidate {
				t.Fatalf("the loop's own recorded SHIP lets the advance proceed: %+v %v", res, err)
			}
		})
	}
}

// TestAReturnTheLoopCannotReadAVerdictFromIsRefused: a return with no verdict,
// two, a word outside its role's vocabulary, or an audit verdict echoing a hash
// the request did not issue is refused at the receipt, and the lane still
// awaits that validator.
func TestAReturnTheLoopCannotReadAVerdictFromIsRefused(t *testing.T) {
	cases := map[string]struct{ role, body, reason string }{
		"no verdict":         {RoleRuthless, "### Analysis\n\nLooks fine.\n", "Verdict"},
		"two verdicts":       {RoleRuthless, reviewerReturn("SHIP") + "\n### Verdict\n\n- **FIX FIRST**\n", "Verdict"},
		"another role's":     {RoleRuthless, reviewerReturn("APPROVE"), "SHIP"},
		"no word":            {RoleSecurity, "### Verdict\n\nLGTM\n", "APPROVE"},
		"a foreign hash":     {RoleAuditor, "", "prompt_hash"},
		"not a verdict JSON": {RoleAuditor, "{}", "verdict"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			id, stages := start.RunID, DefaultStages()
			implemented(t, repo, id, stages, "one.txt")
			if tc.role != RoleRuthless {
				passRound(t, repo, id, stages, RoleRuthless)
			}
			if tc.role == RoleAuditor {
				passRound(t, repo, id, stages, RoleSecurity)
			}
			res, err := Advance(repo.Root(), id, stages, Options{})
			if err != nil || res.Awaiting == nil || res.Awaiting.Role != tc.role {
				t.Fatalf("want the %s handed out: %+v %v", tc.role, res, err)
			}
			path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
			body := tc.body
			if name == "a foreign hash" {
				body = regexp.MustCompile(`"prompt_hash":"sha256:[0-9a-f]{64}"`).ReplaceAllString(
					auditorVerdict(t, filepath.Join(filepath.Dir(path), AuditRequestFileName), "MET"),
					`"prompt_hash":"sha256:`+strings.Repeat("0", 64)+`"`)
			}
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			before := stateBytes(t, repo.Root(), id)
			_, err = Receipt(repo.Root(), id, path, stages, Options{})
			r := mustRefusal(t, err)
			if r.Stage != "receipt" || !strings.Contains(r.Reason, tc.reason) {
				t.Fatalf("want a receipt refusal naming %q: %+v", tc.reason, r)
			}
			if string(before) != string(stateBytes(t, repo.Root(), id)) {
				t.Fatal("a refused return moves nothing")
			}
		})
	}
}

// TestAVersion4StateReadsAsOneNoValidatorHasJudged: version 5 added the
// validate stage's record, so a version-4 file is read as a run no validator has
// judged yet and written back at the current version, and a version-4 file
// carrying a validation is not one version 4 wrote, and is refused.
func TestAVersion4StateReadsAsOneNoValidatorHasJudged(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	if err := os.WriteFile(path, downgraded(t, stateBytes(t, repo.Root(), id), 4), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-4 file reads as the current version: %d %v", st.SchemaVersion, err)
	}
	handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("SHIP"))
	if !strings.Contains(string(stateBytes(t, repo.Root(), id)), cur) {
		t.Fatalf("the next mutation writes the file back at version %d", SchemaVersion)
	}
	if err := os.WriteFile(path, downgraded(t, stateBytes(t, repo.Root(), id), 4), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ReadState(repo.Root(), id)
	if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, "validation") {
		t.Fatalf("a version-4 file carrying a validation is refused: %+v", r)
	}
}

// downgraded is a state file as an earlier version would carry it: at version
// v, without the pace's fix-round cap, which version 6 added.
func downgraded(t *testing.T, data []byte, v int) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	m["schema_version"] = v
	if p, ok := m["pace"].(map[string]any); ok && v <= schemaVersionUncapped {
		delete(p, "fix_rounds")
	}
	if v <= schemaVersionUnlanded {
		delete(m, "transcripts")
		lanes, _ := m["lanes"].([]any)
		for _, l := range lanes {
			if lm, ok := l.(map[string]any); ok {
				for _, k := range []string{"receipts", "resolves", "landing"} {
					delete(lm, k)
				}
			}
		}
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return out
}
