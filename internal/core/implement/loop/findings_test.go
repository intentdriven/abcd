package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The per-reviewer findings record (itd-2610240922461281,
// spc-2610240951109420 step 4).

// findingsReturn is a reviewer's return citing each location under its
// Findings heading.
func findingsReturn(verdict string, locations ...string) string {
	var b strings.Builder
	b.WriteString("### Analysis\n\nChecked other.go:1 and ruled it out.\n\n### Findings\n\n")
	for _, l := range locations {
		fmt.Fprintf(&b, "- **%s** — a defect.\n  - *Evidence:* the line, quoted.\n", l)
	}
	b.WriteString("\n### Verdict\n\n- **" + verdict + "** — as stated; see verdict.go:9.\n")
	return b.String()
}

// at names a finding by its location.
func at(fs []Finding) []string {
	out := []string{}
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s:%d", f.File, f.Line))
	}
	return out
}

// TestEachCitedLocationUnderTheFindingsIsOneLocatedFinding: every
// `file:line` under a return's Findings heading, down to the next heading at
// its depth, is one finding of the reviewer; a location cited twice is one, a
// range or a column is its first line, a path under the worktree is taken
// relative to it, and a path outside it, one climbing out, a time and a word
// with no path in it are no location. A return citing none records zero.
func TestEachCitedLocationUnderTheFindingsIsOneLocatedFinding(t *testing.T) {
	const wt = "/work/lane-1"
	text := "Reviewed head: x\n\n### Analysis\n\nRead before.go:4.\n\n### Findings\n\n" +
		"- **internal/a/b.go:12** — a defect; called from `c.go:3-9`.\n" +
		"  - *Evidence:* the line at internal/a/b.go:12, quoted.\n" +
		"#### A note\n\n- " + wt + "/d/e.go:4:7 (abs), and /etc/hosts.txt:1, ../up.go:3, round:2, at 12:30, ok.go:0\n" +
		"- [Makefile.am:40](link), **f.go:5**\n" +
		"\n### Verdict\n\n- **FIX FIRST** — verdict.go:9\n"
	var v ValidatorRun
	v.Role = RoleRuthless
	v.recordFindings(text, wt)
	want := []string{"internal/a/b.go:12", "c.go:3", "d/e.go:4", "Makefile.am:40", "f.go:5"}
	if v.Located == nil || *v.Located != len(want) || !slices.Equal(at(v.Findings), want) {
		t.Fatalf("the located findings: %v (count %v), want %v", at(v.Findings), v.Located, want)
	}

	var none ValidatorRun
	none.recordFindings(reviewerReturn("SHIP"), wt)
	if none.Located == nil || *none.Located != 0 || none.Findings != nil {
		t.Fatalf("a return citing no location records zero: %+v", none)
	}
	var many strings.Builder
	many.WriteString("### Findings\n\n")
	for i := range maxLocatedFindings + 5 {
		fmt.Fprintf(&many, "- a.go:%d\n", i+1)
	}
	var big ValidatorRun
	big.recordFindings(many.String(), wt)
	if *big.Located != maxLocatedFindings+5 || len(big.Findings) != maxLocatedFindings {
		t.Fatalf("the count counts every location and the record keeps the bound: %d, %d", *big.Located, len(big.Findings))
	}
}

// TestTheFixReportSaysAppliedOrRejectedOfEachFinding: a finding takes the
// word the fix report's lines citing its location say, and unknown where none
// does, or where they say both or neither.
func TestTheFixReportSaysAppliedOrRejectedOfEachFinding(t *testing.T) {
	report := strings.Split("# Fix report\n\n- a.go:1 applied in abc123\n- `b.go:2`: Rejected, not a defect\n"+
		"- c.go:3 looked at\n- d.go:4 applied, then rejected\n- e.go:5 applied\n- e.go:5 rejected after all\n- f.go:6:2 applied\n", "\n")
	for loc, want := range map[string]string{"a.go:1": FindingApplied, "b.go:2": FindingRejected, "c.go:3": FindingUnknown,
		"d.go:4": FindingUnknown, "e.go:5": FindingUnknown, "f.go:6": FindingApplied, "g.go:7": FindingUnknown} {
		f := citations(loc, "")[0]
		if got := fixStatus(report, f, ""); got != want {
			t.Errorf("%s: %s, want %s", loc, got, want)
		}
	}
}

// TestALocationTwoReviewersCiteIsMarkedCitedByBoth: a file and line two
// reviewers of a round cite is marked, on each of their findings, with both;
// one only one cites carries no mark.
func TestALocationTwoReviewersCiteIsMarkedCitedByBoth(t *testing.T) {
	n := 2
	r := ValidationRound{Validators: []ValidatorRun{
		{Role: RoleRuthless, Reviewer: 0, Located: &n, Findings: []Finding{{File: "a.go", Line: 1}, {File: "b.go", Line: 2}}},
		{Role: RoleRuthless, Reviewer: 1, Located: &n, Findings: []Finding{{File: "a.go", Line: 1}, {File: "a.go", Line: 2}}},
		{Role: RoleAuditor},
	}}
	markCitedBy(&r)
	both := []Citer{{RoleRuthless, 0}, {RoleRuthless, 1}}
	for _, i := range []int{0, 1} {
		if f := r.Validators[i].Findings[0]; !slices.Equal(f.CitedBy, both) {
			t.Fatalf("reviewer %d's a.go:1 is cited by both: %+v", i, f)
		}
		if f := r.Validators[i].Findings[1]; f.CitedBy != nil {
			t.Fatalf("reviewer %d's second finding is its own: %+v", i, f)
		}
	}
}

// TestEachReviewersLocatedFindingsAreRecordedAndJoinedToTheFixReport: under
// the bundled pair, each reviewer's verdict is recorded with the locations its
// return cites, zero included; a location both ruthless reviewers cite is
// marked cited by both; once the fix report is verified, each finding of a
// reviewer that failed the round carries applied, rejected or unknown, and a
// passing reviewer's findings, which no fix brief handed on, carry none; and
// the run record's --json carries them.
func TestEachReviewersLocatedFindingsAreRecordedAndJoinedToTheFixReport(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implementedOnce(t, repo, id, "one.txt")

	r0 := stepAwait(t, repo, id, stages, Options{})
	r1 := stepAwait(t, repo, id, stages, Options{})
	returnAt(t, repo, id, stages, r0, findingsReturn("SHIP", "shared.go:3"))
	returnAt(t, repo, id, stages, r1, findingsReturn("FIX FIRST", "shared.go:3", filepath.Join(l.Worktree, "fixme.go")+":7", "quiet.go:11"))
	s0 := stepAwait(t, repo, id, stages, Options{})
	s1 := stepAwait(t, repo, id, stages, Options{})
	returnAt(t, repo, id, stages, s0, reviewerReturn("APPROVE"))
	returnAt(t, repo, id, stages, s1, reviewerReturn("APPROVE"))
	aud := stepAwait(t, repo, id, stages, Options{})
	returnAt(t, repo, id, stages, aud, "MET")

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	vs := st.Lanes[0].Validation[0].Validators
	if !slices.Equal(at(vs[0].Findings), []string{"shared.go:3"}) || !slices.Equal(at(vs[1].Findings), []string{"shared.go:3", "fixme.go:7", "quiet.go:11"}) {
		t.Fatalf("each reviewer's located findings: %+v / %+v", vs[0].Findings, vs[1].Findings)
	}
	for _, i := range []int{2, 3} {
		if vs[i].Located == nil || *vs[i].Located != 0 || len(vs[i].Findings) != 0 {
			t.Fatalf("a return citing none records zero and is not refused: %+v", vs[i])
		}
	}
	if vs[4].Role != RoleAuditor || vs[4].Located != nil {
		t.Fatalf("the auditor records no located findings: %+v", vs[4])
	}
	both := []Citer{{RoleRuthless, 0}, {RoleRuthless, 1}}
	if !slices.Equal(vs[0].Findings[0].CitedBy, both) || !slices.Equal(vs[1].Findings[0].CitedBy, both) || vs[1].Findings[1].CitedBy != nil {
		t.Fatalf("shared.go:3 is cited by both: %+v / %+v", vs[0].Findings, vs[1].Findings)
	}

	fix := laneCommit(t, repo, l, "fixme.go")
	fixedWith(t, repo, id, stages, "shared.go:3, fixme.go:7, quiet.go:11", "- fixme.go:7 applied in "+fix+"\n- `shared.go:3` rejected: the caller checks it first\n", fix)

	st, err = ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	vs = st.Lanes[0].Validation[0].Validators
	status := func(fs []Finding) []string {
		out := []string{}
		for _, f := range fs {
			out = append(out, f.Status)
		}
		return out
	}
	if got := status(vs[1].Findings); !slices.Equal(got, []string{FindingRejected, FindingApplied, FindingUnknown}) {
		t.Fatalf("the fix report's word is joined to each finding, unknown where it is silent: %v", got)
	}
	if got := status(vs[0].Findings); !slices.Equal(got, []string{""}) {
		t.Fatalf("a passing reviewer's findings were handed on to no fix: %v", got)
	}

	rec, err := ReadRecord(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"located":3`, `"located":0`, `{"file":"fixme.go","line":7,"status":"applied"}`,
		`{"file":"shared.go","line":3,"status":"rejected","cited_by":[{"role":"ruthless-reviewer","reviewer":0},{"role":"ruthless-reviewer","reviewer":1}]}`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("implement record --json carries %s:\n%s", want, raw)
		}
	}
	var notes strings.Builder
	for _, e := range st.Record {
		notes.WriteString(e.Note + "\n")
	}
	if !strings.Contains(notes.String(), "3 located finding(s)") || !strings.Contains(notes.String(), "1 applied, 1 rejected, 1 the fix report does not say") {
		t.Fatalf("the record's lines count the findings and their fixes:\n%s", notes.String())
	}
}

// fixedWith is fixed, checking first that the fix brief names a located
// finding and asks the report to say applied or rejected of it.
func fixedWith(t *testing.T, repo *gittest.Repo, id string, stages Stages, loc, report string, commits ...string) {
	t.Helper()
	fixed(t, repo, id, stages, report, commits...)
	l := currentLane(t, repo, id)
	dir, err := roundDir(id, l.ID, 1, FixDirName)
	if err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(dir), BriefFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"located findings: " + loc, "`applied` or `rejected`"} {
		if !strings.Contains(string(brief), want) {
			t.Fatalf("the fix brief names %q:\n%s", want, brief)
		}
	}
}

// TestAStoredFindingTheLoopNeverWritesIsRefused: a state whose located
// findings the loop could not have written is refused at read, and an older
// version carrying them is refused naming them.
func TestAStoredFindingTheLoopNeverWritesIsRefused(t *testing.T) {
	n := 1
	round := func(mut func(*ValidatorRun, *ValidationRound)) State {
		r := ValidationRound{Round: 1, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST", Located: &n,
			Findings: []Finding{{File: "a.go", Line: 1}}}}}
		mut(&r.Validators[0], &r)
		return State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{r}}}}
	}
	if !round(func(*ValidatorRun, *ValidationRound) {}).findingsWritten() {
		t.Fatal("a finding the loop writes reads")
	}
	two := 2
	for name, mut := range map[string]func(*ValidatorRun, *ValidationRound){
		"no verdict":           func(v *ValidatorRun, _ *ValidationRound) { v.Verdict = "" },
		"the auditor":          func(v *ValidatorRun, _ *ValidationRound) { v.Role = RoleAuditor },
		"a count off":          func(v *ValidatorRun, _ *ValidationRound) { v.Located = &two },
		"findings uncounted":   func(v *ValidatorRun, _ *ValidationRound) { v.Located = nil },
		"an absolute path":     func(v *ValidatorRun, _ *ValidationRound) { v.Findings[0].File = "/etc/a.go" },
		"a climbing path":      func(v *ValidatorRun, _ *ValidationRound) { v.Findings[0].File = "../a.go" },
		"a spaced path":        func(v *ValidatorRun, _ *ValidationRound) { v.Findings[0].File = "a b.go" },
		"line zero":            func(v *ValidatorRun, _ *ValidationRound) { v.Findings[0].Line = 0 },
		"an unanswered status": func(v *ValidatorRun, _ *ValidationRound) { v.Findings[0].Status = FindingApplied },
		"an unknown status":    func(v *ValidatorRun, r *ValidationRound) { r.Fix = "fix/receipt.json"; v.Findings[0].Status = "done" },
		"an answer missing":    func(_ *ValidatorRun, r *ValidationRound) { r.Fix = "fix/receipt.json" },
		"a forged citer": func(v *ValidatorRun, _ *ValidationRound) {
			v.Findings[0].CitedBy = []Citer{{RoleSecurity, 0}, {RoleRuthless, 0}}
		},
	} {
		if round(mut).findingsWritten() {
			t.Errorf("%s is refused", name)
		}
	}
	if s := round(func(*ValidatorRun, *ValidationRound) {}).located(); !strings.Contains(s, "located") || !strings.Contains(s, "lane-1") {
		t.Fatalf("the version check names the findings: %q", s)
	}
}

// TestLocatedFindingsAreTheVersionAfterTheUnlocated: a version-23 file
// carrying located findings is refused naming them, and one without them
// reads at the current version.
func TestLocatedFindingsAreTheVersionAfterTheUnlocated(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implementedOnce(t, repo, id, "one.txt")
	r0 := stepAwait(t, repo, id, stages, Options{})
	returnAt(t, repo, id, stages, r0, findingsReturn("SHIP", "a.go:1"))
	root := repo.Root()
	data := stateBytes(t, root, id)
	if !strings.Contains(string(data), `"located": 1`) {
		t.Fatalf("the round keeps the count: %s", data)
	}
	state := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	if err := os.WriteFile(state, []byte(strings.Replace(string(data), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion),
		fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnlocated), 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "located") {
		t.Fatalf("an older file carrying them is refused naming them: %v", err)
	}
	if err := os.WriteFile(state, downgraded(t, data, schemaVersionUnlocated), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, id)
	if err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("an older file without them reads at the current version: %v %+v", err, st.SchemaVersion)
	}
}
