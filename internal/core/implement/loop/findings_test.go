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
	if want := []string{"a.go:3"}; !slices.Equal(at(v.Findings), want) {
		t.Fatalf("a spaced worktree, matched in its own case only: %v, want %v", at(v.Findings), want)
	}
	v.recordFindings("### Findings\n\n- **x/work/lane-1/a.go:1** — x\n- **/work/lane-1/b.go:2** — x\n- `/WORK/LANE-1/c.go:3`\n", "/work/lane-1")
	if want := []string{"x/work/lane-1/a.go:1", "b.go:2"}; !slices.Equal(at(v.Findings), want) {
		t.Fatalf("the worktree is taken out only at a token's start, in its own case: %v, want %v", at(v.Findings), want)
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
// line's first location of that one. Only a sentence mark (`.`, `;`, `:`,
// `!` or `?` before a space or the line's end, or a dash) ends a clause: a
// comma, and the dot inside a file name, do not.
func TestTheFixReportSaysAppliedOrRejectedOfEachFinding(t *testing.T) {
	report := "# Fix report\n\n- a.go:1 applied in abc123\n- `b.go:2`: Rejected, not a defect\n" +
		"- c.go:3 looked at\n- d.go:4 applied, then rejected\n- e.go:5 applied\n- e.go:5 rejected after all\n- f.go:6:2 applied\n" +
		"- h.go:8: could not be applied without an API break, left as is\n- i.go:9 was not yet applied\n- j.go:10 wasn't applied\n" +
		"- k.go:11 unapplied, rejected instead\n- l.go:12 cannot be rejected, applied\n" +
		"- m.go:13 applied; n.go:14 needed nothing\n- Rejected: `o.go:15` (not a defect)\n" +
		"- q.go:17 and r.go:18 applied in abc\n- s.go:19 applied (see s.go:19 above)\n- Not applied: t.go:20\n" +
		"- the caller at u.go:21, not a defect, rejected\n- v.go:22 never applied\n- w.go:23 non-applied\n- x.go:24 wasn\u2019t applied\n" +
		"- y.go:25 no fix applied\n- z.go:26 nothing applied\n- aa.go:27 failed to be applied\n- ab.go:28 was not, on reflection, applied\n" +
		"- ac.go:29 not a defect. Rejected\n- ad.go:30 not a defect; rejected\n- ae.go:31 not a defect \u2014 rejected\n" +
		"- af.go:32 not a defect: rejected\n- ag.go:33 not in x.go, applied\n- ah.go:34 not a defect -- rejected\n"
	want := map[string]string{"a.go:1": FindingApplied, "b.go:2": FindingRejected, "c.go:3": FindingUnknown,
		"y.go:25": FindingUnknown, "z.go:26": FindingUnknown, "aa.go:27": FindingUnknown, "ab.go:28": FindingUnknown,
		"ac.go:29": FindingRejected, "ad.go:30": FindingRejected, "ae.go:31": FindingRejected, "af.go:32": FindingRejected,
		"ag.go:33": FindingUnknown, "ah.go:34": FindingRejected,
		"d.go:4": FindingUnknown, "e.go:5": FindingUnknown, "f.go:6": FindingApplied, "g.go:7": FindingUnknown,
		"h.go:8": FindingUnknown, "i.go:9": FindingUnknown, "j.go:10": FindingUnknown, "k.go:11": FindingUnknown,
		"l.go:12": FindingUnknown, "m.go:13": FindingApplied, "n.go:14": FindingUnknown, "o.go:15": FindingRejected,
		"q.go:17": FindingUnknown, "r.go:18": FindingApplied, "s.go:19": FindingApplied, "t.go:20": FindingUnknown,
		"u.go:21": FindingUnknown, "v.go:22": FindingUnknown, "w.go:23": FindingUnknown, "x.go:24": FindingUnknown}
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

// TestAReportLineOfManyStatusWordsIsReadQuickly: a fix report of the most
// bytes a report may be, one line holding one location and a status word
// repeated after it, or before it, is joined in well under the time a
// quadratic read would take.
func TestAReportLineOfManyStatusWordsIsReadQuickly(t *testing.T) {
	words := strings.Repeat(" applied", 130_000)
	for name, report := range map[string]string{"after": "- a.go:1" + words, "before": "-" + words + " a.go:1"} {
		if len(report) > maxReportBytes {
			t.Fatalf("%s: the fixture is %d bytes, more than a report may be", name, len(report))
		}
		done := make(chan string, 1)
		go func() {
			n := 1
			r := ValidationRound{Validators: []ValidatorRun{{Role: RoleRuthless, Located: &n, Findings: []Finding{{File: "a.go", Line: 1}}}}}
			joinFixReport(&r, report, "")
			done <- r.Validators[0].Findings[0].Status
		}()
		select {
		case got := <-done:
			if got != FindingApplied {
				t.Fatalf("%s: %s, want applied", name, got)
			}
		case <-time.After(15 * time.Second):
			t.Fatalf("%s: a one-line report of many status words took more than 15s to read", name)
		}
	}
}

// TestALineNamingALocationWithoutAStatusWordSaysNothingOfIt: a line that
// names a location but says no status word of it, the fix brief's list of
// located findings repeated among them, leaves what the other lines say of
// it; a location only such lines name is unknown.
func TestALineNamingALocationWithoutAStatusWordSaysNothingOfIt(t *testing.T) {
	report := "located findings: a.go:1, b.go:2, c.go:3\n\n- a.go:1 applied in abc\n- see a.go:1 above\n- b.go:2 rejected\n- c.go:3 looked at\n"
	n := 3
	r := ValidationRound{Validators: []ValidatorRun{{Role: RoleRuthless, Located: &n,
		Findings: []Finding{{File: "a.go", Line: 1}, {File: "b.go", Line: 2}, {File: "c.go", Line: 3}}}}}
	joinFixReport(&r, report, "")
	var got []string
	for _, f := range r.Validators[0].Findings {
		got = append(got, f.Status)
	}
	if want := []string{FindingApplied, FindingRejected, FindingUnknown}; !slices.Equal(got, want) {
		t.Fatalf("a bare mention says nothing: %v, want %v", got, want)
	}
}

// TestAStoredCountPastAnyReturnIsRefused: a stored count of located findings
// greater than a return of the most bytes it may be could hold is refused at
// read, however many findings it keeps; the bound itself reads.
func TestAStoredCountPastAnyReturnIsRefused(t *testing.T) {
	var fs []Finding
	for i := range maxLocatedFindings {
		fs = append(fs, Finding{File: "a.go", Line: i + 1})
	}
	state := func(n int) State {
		return State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{{Round: 1, Validators: []ValidatorRun{
			{Role: RoleRuthless, Verdict: "FIX FIRST", Located: &n, Findings: fs}}}}}}}
	}
	if !state(maxLocatedCount).findingsWritten() {
		t.Fatal("a count at the bound reads")
	}
	for _, n := range []int{maxLocatedCount + 1, 1 << 62} {
		if state(n).findingsWritten() {
			t.Errorf("a count of %d is refused", n)
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

// TestAMaximalReturnOfTheShortestBulletsRecordsACountTheReadAccepts: a return
// of the most bytes a return may be, packed with the shortest distinct lead
// bullets (`- .a:1`, seven bytes with its newline), records a count the
// state's own read accepts, so the run goes on.
func TestAMaximalReturnOfTheShortestBulletsRecordsACountTheReadAccepts(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implementedOnce(t, repo, id, "one.txt")
	r0 := stepAwait(t, repo, id, stages, Options{})

	const tail = "\n### Verdict\n\n- **SHIP** — as stated.\n"
	budget := maxReturnBytes - 160 - len(tail)
	var b strings.Builder
	b.WriteString("### Findings\n")
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bullet := func(s string) bool {
		if b.Len()+len(s) > budget {
			return false
		}
		b.WriteString(s)
		return true
	}
	full := false
	for _, x := range letters { // seven bytes: a dot, a letter, one digit
		for d := 1; d <= 9 && !full; d++ {
			full = !bullet(fmt.Sprintf("- .%c:%d\n", x, d))
		}
	}
	for _, x := range letters { // eight bytes: three path bytes and one digit
		for _, y := range letters {
			for d := 1; d <= 9 && !full; d++ {
				full = !bullet(fmt.Sprintf("- .%c%c:%d\n", x, y, d))
				full = full || !bullet(fmt.Sprintf("- %c.%c:%d\n", x, y, d))
			}
		}
	}
	if !full {
		t.Fatal("the fixture does not fill a return")
	}
	b.WriteString(tail)
	returnAt(t, repo, id, stages, r0, b.String())

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatalf("the state the loop wrote reads: %v", err)
	}
	var located *int
	for _, r := range st.Lanes[0].Validation {
		for _, v := range r.Validators {
			if v.Located != nil && *v.Located > 0 {
				located = v.Located
			}
		}
	}
	if located == nil || *located <= maxReturnBytes/8 || *located > maxLocatedCount {
		t.Fatalf("the count is past what eight-byte bullets could hold and within the bound: %v", located)
	}
}

// TestALineInACodeFenceIsNeitherAHeadingNorAFinding: a `#` line inside a
// fenced code block under the Findings heading does not end the section, a
// bullet inside a fence is no finding, a shorter run or the other mark does
// not close a fence, and a one-line ```code``` opens none.
func TestALineInACodeFenceIsNeitherAHeadingNorAFinding(t *testing.T) {
	text := "### Findings\n\n" +
		"- **a.go:1** — a defect.\n" +
		"  - *Evidence:*\n" +
		"    ````sh\n" +
		"    # a comment\n" +
		"    - in.go:9\n" +
		"    ```\n" +
		"    ## not a heading\n" +
		"    ````\n" +
		"- **b.go:2** — after the fence.\n" +
		"~~~\n# tilde comment\n- tilde.go:3\n```\n~~~~\n" +
		"```c.go:3``` inline, then\n" +
		"- **d.go:4**\n" +
		"\n### Verdict\n\n- **e.go:5** FIX FIRST\n"
	var v ValidatorRun
	v.recordFindings(text, "")
	if want := []string{"a.go:1", "b.go:2", "d.go:4"}; !slices.Equal(at(v.Findings), want) || *v.Located != len(want) {
		t.Fatalf("the located findings: %v, want %v", at(v.Findings), want)
	}
}

// hostileFindings is a return leading the most findings a record keeps, each
// at a distinct path of the most bytes a location may be.
func hostileFindings(verdict string) string {
	var b strings.Builder
	b.WriteString("### Findings\n\n")
	for j := range maxLocatedFindings {
		p := fmt.Sprintf("d%03d/", j) + strings.Repeat("a", maxLocationPath-len("d000/")-len(".go")) + ".go"
		fmt.Fprintf(&b, "- **%s:1000000** — x\n", p)
	}
	b.WriteString("\n### Verdict\n\n- **" + verdict + "** — as stated.\n")
	return b.String()
}

// TestTheMostFindingsReviewersCanWriteLeaveAStateTheReadAccepts: many lanes,
// each of several rounds failed by two reviewers whose returns lead the most
// findings a record keeps at the longest paths, every location cited by both
// and answered by a fix report, make a state the loop writes no larger than
// its read accepts and reads back, each count kept.
func TestTheMostFindingsReviewersCanWriteLeaveAStateTheReadAccepts(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implementedOnce(t, repo, id, "one.txt")
	stepAwait(t, repo, id, stages, Options{})
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	const lanes, rounds = 16, 4
	base, tmpl := st.Lanes[0], st.Lanes[0].Validation[0]
	var hostile []int
	for k, v := range tmpl.Validators {
		if (v.Role == RoleRuthless || v.Role == RoleSecurity) && v.Reviewer == 0 {
			hostile = append(hostile, k)
		}
	}
	if len(hostile) != 2 {
		t.Fatalf("the round hands out a ruthless and a security reviewer: %+v", tmpl.Validators)
	}
	st.Lanes = nil
	for i := range lanes {
		l := base
		l.ID = fmt.Sprintf("%s-%d", base.ID, i)
		l.Validation = nil
		for k := range rounds {
			r := tmpl
			r.Round = k + 1
			r.Validators = slices.Clone(tmpl.Validators)
			for _, h := range hostile {
				v := &r.Validators[h]
				v.Verdict, v.Pass = "FIX FIRST", false
				v.recordFindings(hostileFindings("FIX FIRST"), l.Worktree)
			}
			markCitedBy(&r)
			r.Fix = tmpl.Validators[hostile[0]].Return
			joinFixReport(&r, "", l.Worktree)
			l.Validation = append(l.Validation, r)
		}
		st.Lanes = append(st.Lanes, l)
	}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := writeState(root, st); err != nil {
		t.Fatalf("the loop writes the state: %v", err)
	}
	if n := len(stateBytes(t, repo.Root(), id)); n > maxStateBytes {
		t.Fatalf("the state is %d bytes, past the %d its read accepts", n, maxStateBytes)
	}
	got, err := readListedState(root, id)
	if err != nil {
		t.Fatalf("the state the loop wrote reads: %v", err)
	}
	kept := 0
	for _, l := range got.Lanes {
		for _, r := range l.Validation {
			for _, h := range hostile {
				v := r.Validators[h]
				if v.Located == nil || *v.Located != maxLocatedFindings {
					t.Fatalf("%s round %d keeps its count: %v", l.ID, r.Round, v.Located)
				}
				kept += len(v.Findings)
			}
		}
	}
	if kept == 0 {
		t.Fatal("the latest rounds that fit keep their findings")
	}
}

// TestAStateTooLargeEvenWithoutFindingsIsNotWritten: a state past what its
// read accepts with no located finding left to drop is refused by writeState
// and by a reservation save, and the file already there stands.
func TestAStateTooLargeEvenWithoutFindingsIsNotWritten(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	before := stateBytes(t, repo.Root(), id)
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	big := Entry{At: time.Now().UTC(), Stage: "receipt", Note: strings.Repeat("x", maxStateBytes)}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	grown := st
	grown.Record = append(slices.Clone(st.Record), big)
	if r := mustRefusal(t, writeState(root, grown)); !strings.Contains(r.Reason, "nothing was written") {
		t.Fatalf("the refusal says nothing was written: %+v", r)
	}
	err = mutateReservation(repo.Root(), id, func(s *State) (bool, error) {
		s.Record = append(s.Record, big)
		return true, nil
	})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "nothing was written") {
		t.Fatalf("a reservation save says nothing was written: %+v", r)
	}
	if after := stateBytes(t, repo.Root(), id); string(after) != string(before) {
		t.Fatal("the file already there stands")
	}
	if _, err := ReadState(repo.Root(), id); err != nil {
		t.Fatalf("and still reads: %v", err)
	}
}

// TestFindingsAreKeptOnALanesLatestRoundOnly: the writer drops an earlier
// round's findings, keeping its count; the read refuses a state carrying
// findings on an earlier round, and accepts a latest round whose findings the
// writer dropped.
func TestFindingsAreKeptOnALanesLatestRoundOnly(t *testing.T) {
	n := 1
	round := func(no int) ValidationRound {
		return ValidationRound{Round: no, Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "FIX FIRST", Located: &n,
			Findings: []Finding{{File: "a.go", Line: 1}}}}}
	}
	st := State{Lanes: []Lane{{ID: "lane-1", Validation: []ValidationRound{round(1), round(2)}}}}
	if st.findingsWritten() {
		t.Fatal("findings on an earlier round are refused")
	}
	kept := st
	kept.Lanes = slices.Clone(st.Lanes)
	kept.Lanes[0].Validation = slices.Clone(st.Lanes[0].Validation)
	kept.keepLatestFindings()
	if !kept.findingsWritten() || len(kept.Lanes[0].Validation[0].Validators[0].Findings) != 0 ||
		*kept.Lanes[0].Validation[0].Validators[0].Located != 1 || len(kept.Lanes[0].Validation[1].Validators[0].Findings) != 1 {
		t.Fatalf("the writer keeps the latest round's findings and every count: %+v", kept.Lanes[0].Validation)
	}
	if len(st.Lanes[0].Validation[0].Validators[0].Findings) != 1 {
		t.Fatal("the state the writer was handed is not changed")
	}
	if !kept.trimOldestFindings() || kept.trimOldestFindings() || !kept.findingsWritten() {
		t.Fatalf("a latest round whose findings the writer dropped reads: %+v", kept.Lanes[0].Validation)
	}
}

// TestAWorktreeSpellingNoPatternCompilesLeavesTheTextAsWritten: a worktree
// spelling that is no valid UTF-8 leaves the text unchanged rather than
// stopping the verifier.
func TestAWorktreeSpellingNoPatternCompilesLeavesTheTextAsWritten(t *testing.T) {
	const text = "- **/wt\xff/a.go:1** — x"
	if got := relativeTo(text, "/wt\xff"); got != text {
		t.Fatalf("the text is read as written: %q", got)
	}
}

// TestALineInAnHTMLCommentIsNeitherAHeadingNorAFinding: a heading inside an
// HTML comment under the Findings heading does not end the section, and a
// bullet inside one is no finding.
func TestALineInAnHTMLCommentIsNeitherAHeadingNorAFinding(t *testing.T) {
	text := "### Findings\n\n" +
		"- **a.go:1** — a defect.\n" +
		"<!--\n" +
		"## parked\n" +
		"- **in.go:9** — parked.\n" +
		"-->\n" +
		"- **b.go:2** — after the comment.\n" +
		"\n### Verdict\n\n- **FIX FIRST**\n"
	var v ValidatorRun
	v.recordFindings(text, "")
	if want := []string{"a.go:1", "b.go:2"}; !slices.Equal(at(v.Findings), want) || *v.Located != len(want) {
		t.Fatalf("the located findings: %v, want %v", at(v.Findings), want)
	}
}
