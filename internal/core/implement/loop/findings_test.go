package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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

// TestEachFindingBulletsLeadLocationIsOneLocatedFinding: the location a
// finding bullet under a return's Findings heading leads with, down to the
// next heading at its depth, is one finding of the reviewer; a location led
// with twice is one, a range or a column is its first line, a path under the
// worktree is taken relative to it, and a path outside it, one climbing out,
// a time and a word with no path in it are no location. A location cited in
// a bullet's text, a nested Evidence line, or behind a word, is no finding.
// A return citing none records zero.
func TestEachFindingBulletsLeadLocationIsOneLocatedFinding(t *testing.T) {
	const wt = "/work/lane-1"
	text := "Reviewed head: x\n\n### Analysis\n\n- before.go:4 read first.\n\n### Findings\n\n" +
		"- **internal/a/b.go:12** — a defect; called from `c.go:3-9`.\n" +
		"  - *Evidence:* the line at internal/a/b.go:12, quoted; refuted at y.go:12.\n" +
		"  - x/nested.go:8 is the caller\n" +
		"#### A note\n\n- " + wt + "/d/e.go:4:7 (abs), and /etc/hosts.txt:1\n" +
		"- /etc/hosts.txt:1 outside\n- ../up.go:3\n- round:2\n- at 12:30\n- 12:30\n- ok.go:0\n" +
		"- [Makefile.am:40](link), **f.go:5**\n" +
		"1. `g.go:6` numbered\n" +
		"- **minor** h.go:7 behind a word\n" +
		"- **internal/a/b.go:12** again\n" +
		"\n### Verdict\n\n- **verdict.go:9** FIX FIRST\n"
	var v ValidatorRun
	v.Role = RoleRuthless
	v.recordFindings(text, wt)
	want := []string{"internal/a/b.go:12", "d/e.go:4", "Makefile.am:40", "g.go:6"}
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

// TestALocationsPathIsAtMost255Bytes: a path longer than 255 bytes is no
// location, and a stored one is refused at read.
func TestALocationsPathIsAtMost255Bytes(t *testing.T) {
	long := strings.Repeat("a/", 125)
	fits, over := long+"bc.go", long+"bcd.go"
	if len(fits) != 255 || len(over) != 256 {
		t.Fatalf("the fixture's lengths: %d, %d", len(fits), len(over))
	}
	var v ValidatorRun
	v.Role = RoleRuthless
	v.recordFindings("### Findings\n\n- **"+fits+":1**\n- **"+over+":2**\n", "")
	if want := []string{fits + ":1"}; !slices.Equal(at(v.Findings), want) || *v.Located != 1 {
		t.Fatalf("a path past 255 bytes is no location: %v", at(v.Findings))
	}
	n := 1
	s := State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{{Round: 1, Validators: []ValidatorRun{
		{Role: RoleRuthless, Verdict: "FIX FIRST", Located: &n, Findings: []Finding{{File: over, Line: 1}}}}}}}}}
	if s.findingsWritten() {
		t.Fatal("a stored path past 255 bytes is refused")
	}
}

// TestAWorktreeSpelledAnotherWayIsStillTheWorktree: a worktree whose path
// holds a space, or that a citation spells in another case, through its
// /private twin or through the directory its link resolves to, is taken
// relative to it, in a return and in a fix report; an absolute path holding
// a space outside it is no location.
func TestAWorktreeSpelledAnotherWayIsStillTheWorktree(t *testing.T) {
	const spaced = "/Users/Jane Doe/work trees/s/l"
	var v ValidatorRun
	v.Role = RoleRuthless
	v.recordFindings("### Findings\n\n- **"+spaced+"/a.go:3** — x\n- **/USERS/jane doe/work trees/s/l/b.go:4** — x\n"+
		"- **/Users/Other Person/c.go:5** — x\n- `/private/var/w/l/d.go:6`\n", spaced)
	if want := []string{"a.go:3", "b.go:4"}; !slices.Equal(at(v.Findings), want) {
		t.Fatalf("a spaced worktree and its case: %v, want %v", at(v.Findings), want)
	}
	v.recordFindings("### Findings\n\n- `/private/var/w/l/d.go:6`\n- `/var/w/l/e.go:7`\n", "/var/w/l")
	if want := []string{"d.go:6", "e.go:7"}; !slices.Equal(at(v.Findings), want) {
		t.Fatalf("the /private twin: %v, want %v", at(v.Findings), want)
	}

	dir := t.TempDir()
	real, link := filepath.Join(dir, "real"), filepath.Join(dir, "link")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	v.recordFindings("### Findings\n\n- **"+filepath.ToSlash(resolved)+"/f.go:8**\n- **"+filepath.ToSlash(link)+"/g.go:9**\n", filepath.ToSlash(link))
	if want := []string{"f.go:8", "g.go:9"}; !slices.Equal(at(v.Findings), want) {
		t.Fatalf("the directory a linked worktree resolves to: %v, want %v", at(v.Findings), want)
	}

	n := 1
	r := ValidationRound{Validators: []ValidatorRun{{Role: RoleRuthless, Located: &n, Findings: []Finding{{File: "a.go", Line: 3}}}}}
	joinFixReport(&r, "- "+spaced+"/a.go:3 applied in abc\n", spaced)
	if got := r.Validators[0].Findings[0].Status; got != FindingApplied {
		t.Fatalf("a fix report citing the spaced worktree: %s", got)
	}
}

// loc is the finding a `file:line` names.
func loc(t *testing.T, s string) Finding {
	t.Helper()
	i := strings.LastIndex(s, ":")
	var n int
	if _, err := fmt.Sscanf(s[i+1:], "%d", &n); err != nil {
		t.Fatalf("%s: %v", s, err)
	}
	return Finding{File: s[:i], Line: n}
}

// TestTheFixReportSaysAppliedOrRejectedOfEachFinding: a finding takes the
// word the fix report's lines say of its location, and unknown where none
// does, where they say both or neither, where a negation stands before the
// word in its clause, or where the word follows another location on the
// line: a word speaks of the location it follows, and the words before a
// line's first location of that one.
func TestTheFixReportSaysAppliedOrRejectedOfEachFinding(t *testing.T) {
	report := "# Fix report\n\n- a.go:1 applied in abc123\n- `b.go:2`: Rejected, not a defect\n" +
		"- c.go:3 looked at\n- d.go:4 applied, then rejected\n- e.go:5 applied\n- e.go:5 rejected after all\n- f.go:6:2 applied\n" +
		"- h.go:8: could not be applied without an API break, left as is\n- i.go:9 was not yet applied\n- j.go:10 wasn't applied\n" +
		"- k.go:11 unapplied, rejected instead\n- l.go:12 cannot be rejected, applied\n" +
		"- m.go:13 applied; n.go:14 needed nothing\n- Rejected: `o.go:15` (not a defect)\n" +
		"- q.go:17 and r.go:18 applied in abc\n- s.go:19 applied (see s.go:19 above)\n- Not applied: t.go:20\n" +
		"- the caller at u.go:21, not a defect, rejected\n- v.go:22 never applied\n- w.go:23 non-applied\n- x.go:24 wasn\u2019t applied\n"
	want := map[string]string{"a.go:1": FindingApplied, "b.go:2": FindingRejected, "c.go:3": FindingUnknown,
		"d.go:4": FindingUnknown, "e.go:5": FindingUnknown, "f.go:6": FindingApplied, "g.go:7": FindingUnknown,
		"h.go:8": FindingUnknown, "i.go:9": FindingUnknown, "j.go:10": FindingUnknown, "k.go:11": FindingUnknown,
		"l.go:12": FindingUnknown, "m.go:13": FindingApplied, "n.go:14": FindingUnknown, "o.go:15": FindingRejected,
		"q.go:17": FindingUnknown, "r.go:18": FindingApplied, "s.go:19": FindingApplied, "t.go:20": FindingUnknown,
		"u.go:21": FindingRejected, "v.go:22": FindingUnknown, "w.go:23": FindingUnknown, "x.go:24": FindingUnknown}
	var fs []Finding
	for l := range want {
		fs = append(fs, loc(t, l))
	}
	n := len(fs)
	r := ValidationRound{Validators: []ValidatorRun{{Role: RoleRuthless, Located: &n, Findings: fs}}}
	joinFixReport(&r, report, "")
	for _, f := range r.Validators[0].Findings {
		if l := fmt.Sprintf("%s:%d", f.File, f.Line); f.Status != want[l] {
			t.Errorf("%s: %s, want %s", l, f.Status, want[l])
		}
	}
}

// TestAMaximumSizeReturnAndFixReportAreReadQuickly: a return of the most
// bytes a return may be, every line a finding bullet or a Findings heading,
// for each of four failing reviewers, and a fix report of the most bytes a
// report may be on one line of distinct citations, are recorded and joined
// in well under the time a quadratic read would take.
func TestAMaximumSizeReturnAndFixReportAreReadQuickly(t *testing.T) {
	var ret strings.Builder
	for i := 1; ret.Len() < maxReturnBytes-64; i++ {
		if i%2 == 1 {
			ret.WriteString("### Findings\n")
		}
		fmt.Fprintf(&ret, "- **a.go:%d** — x, see b.go:%d\n", i, i)
	}
	var report strings.Builder
	report.WriteString("- applied")
	for i := 1; report.Len() < maxReportBytes-64; i++ {
		fmt.Fprintf(&report, " a.go:%d", i)
	}
	done := make(chan ValidationRound, 1)
	go func() {
		var r ValidationRound
		for _, role := range []string{RoleRuthless, RoleRuthless, RoleSecurity, RoleSecurity} {
			r.Validators = append(r.Validators, ValidatorRun{Role: role, Verdict: "FIX FIRST"})
			v := &r.Validators[len(r.Validators)-1]
			v.recordFindings(ret.String(), "/work/lane-1")
			markCitedBy(&r)
		}
		joinFixReport(&r, report.String(), "/work/lane-1")
		done <- r
	}()
	select {
	case r := <-done:
		if v := r.Validators[3]; *v.Located < 5_000 || len(v.Findings) != maxLocatedFindings || v.Findings[0].Status != FindingApplied ||
			len(v.Findings[0].CitedBy) != 4 {
			t.Fatalf("the record: count %d, %d kept, %+v", *v.Located, len(v.Findings), v.Findings[0])
		}
	case <-time.After(15 * time.Second):
		t.Fatal("a maximum-size return and fix report took more than 15s to read")
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
