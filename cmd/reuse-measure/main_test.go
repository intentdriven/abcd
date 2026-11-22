package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The tails below are the lines make preflight prints at its end, as a passing
// run's log holds them (#1729's report, preflight-matrix-step6-0733.log, and
// #1702's records-only one); the test-stage lines above them are cut, since
// nothing here reads them.
const fullTail = `check-tests: race stage reused 91 of 133 packages (48 with no test run, 43 with only their always-rerun tests run); 85 ran
check-tests: race stage checked in 41m59s (waited 0s at the check directory's lock and the race gate)
preflight: push receipt minted for %s — a push of that commit passes the pre-push gate.
           the unit stage reused 5 of 16 packages and the race stage 91 of 133 (reuse).
preflight: stage gates: 4m22s
preflight: stage unit: 44s
preflight: stage race: 41m59s
preflight: checked in 47m05s (waited 1m08s: load gate 47s, race gate 21s)
exit 0
`

const recordsTail = `preflight: a records-only change, so the unit and race stages are skipped and precheck's tests run in their place; CI's unit lane still runs every test
preflight: push receipt minted for %s, records-only (the unit and race stages were skipped)
           — a push of that commit passes the pre-push gate.
preflight: checked in 4m09s (waited 7m54s: load gate 7m54s)
exit 0
`

// untimedTail is a report from before the preflight printed its time (#1657's).
const untimedTail = `preflight: push receipt minted for %s — a push of that commit passes the pre-push gate.
           the unit stage reused 8 of 16 packages and the race stage 110 of 132 (reuse).
exit 0
`

// coldTail is a run that reused nothing, as make preflight PREFLIGHT_FRESH=1
// reports one.
const coldTail = `preflight: push receipt minted for %s — a push of that commit passes the pre-push gate.
           the unit stage reused 0 of 16 packages and the race stage 0 of 133 (cut).
preflight: stage gates: 4m30s
preflight: stage unit: 2m10s
preflight: stage race: 1h02m00s
preflight: checked in 1h08m40s (waited 0s)
exit 0
`

func writeLog(t *testing.T, dir, name, tail, head string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := "preflight: load check\n" + strings.Replace(tail, "%s", head, 1)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestALandingLogGivesItsCountsAndTimes: the report is read from the lines the
// preflight prints after its receipt, the reuse counts of both stages and the
// times, so a landing's figures are the printed ones and nothing else.
func TestALandingLogGivesItsCountsAndTimes(t *testing.T) {
	path := writeLog(t, t.TempDir(), "preflight-a.log", fullTail, "a91bcdac3472")
	runs, err := readRuns(path)
	if err != nil || len(runs) != 1 {
		t.Fatalf("readRuns = %+v, err %v", runs, err)
	}
	want := report{
		path: path, head: "a91bcdac3472", scope: "full", mode: "reuse",
		unitReused: 5, unitTotal: 16, raceReused: 91, raceTotal: 133,
		race: 41*time.Minute + 59*time.Second, checked: 47*time.Minute + 5*time.Second, waited: 68 * time.Second,
		counted: true, timed: true,
	}
	if runs[0] != want {
		t.Errorf("readRuns =\n %+v\nwant\n %+v", runs[0], want)
	}
}

// TestARecordsOnlyLogHasATimeAndNoCounts: a records-only landing ran no test
// stage, so it reports a time and no reuse counts.
func TestARecordsOnlyLogHasATimeAndNoCounts(t *testing.T) {
	path := writeLog(t, t.TempDir(), "preflight-r.log", recordsTail, "81739e558641")
	runs, err := readRuns(path)
	if err != nil || len(runs) != 1 {
		t.Fatalf("readRuns = %+v, err %v", runs, err)
	}
	if r := runs[0]; r.scope != "records-only" || r.head != "81739e558641" || r.checked != 4*time.Minute+9*time.Second || !r.timed || r.unitTotal != 0 {
		t.Errorf("readRuns = %+v", r)
	}
}

// TestALogWithNoReceiptIsNoLandingReport: a run that failed, or a log of
// something else, mints no receipt, and only a receipt shows the run passed.
func TestALogWithNoReceiptIsNoLandingReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preflight-red.log")
	if err := os.WriteFile(path, []byte("--- FAIL: TestX (0.01s)\nFAIL\nmake: *** [preflight] Error 1\nexit 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runs, err := readRuns(path); len(runs) != 0 || err != nil {
		t.Errorf("readRuns = %+v, err %v; want no report and no error", runs, err)
	}
}

// TestEachRunInALogIsItsOwnReport: a log that holds a run and then a rerun
// reports each run's own figures, never a mix of the two; the first run's
// times stay with the first.
func TestEachRunInALogIsItsOwnReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preflight-twice.log")
	body := strings.Replace(fullTail, "%s", "aaaaaaaaaaaa", 1) + strings.Replace(untimedTail, "%s", "bbbbbbbbbbbb", 1)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := readRuns(path)
	if err != nil || len(runs) != 2 {
		t.Fatalf("readRuns = %+v, err %v; want two runs", runs, err)
	}
	if r := runs[0]; r.head != "aaaaaaaaaaaa" || !r.timed || r.raceReused != 91 || r.race != 41*time.Minute+59*time.Second {
		t.Errorf("the first run = %+v, want its own timed report", r)
	}
	if r := runs[1]; r.head != "bbbbbbbbbbbb" || r.timed || r.raceReused != 110 || r.race != 0 {
		t.Errorf("the second run = %+v, want its own untimed report", r)
	}
}

// fixture builds a default branch with one merge per pull request number, in
// order, and returns the repository and each merge's second parent, the head
// the pull request carried.
func fixture(t *testing.T, prs ...string) (*gittest.Repo, map[string]string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	repo.Commit("root")
	heads := map[string]string{}
	for _, pr := range prs {
		repo.Git("checkout", "-q", "-b", "pr"+pr)
		repo.Write("f"+pr, pr+"\n")
		repo.Commit("change " + pr)
		heads[pr] = repo.Git("rev-parse", "HEAD")
		repo.Git("checkout", "-q", "main")
		repo.Git("merge", "-q", "--no-ff", "-m", "Merge pull request #"+pr+" from intentdriven/pr"+pr, "pr"+pr)
	}
	// A commit straight on the branch is no landing.
	repo.Commit("direct")
	return repo, heads
}

// TestTheLandingsAreTheLastMergesOnTheBranch: a landing is a merge on the
// default branch's first-parent line, newest first, and its head is the merge's
// last parent, which is the commit the preflight checked.
func TestTheLandingsAreTheLastMergesOnTheBranch(t *testing.T) {
	repo, heads := fixture(t, "11", "12", "13")
	got, err := readLandings(repo.Root(), "main", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []landing{{pr: "#13", head: heads["13"]}, {pr: "#12", head: heads["12"]}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("readLandings = %+v, want %+v", got, want)
	}
}

// TestAMeasureShortOfALandingStatesNoSaving: a landing with no report here, an
// untimed report and no timed no-reuse run each leave the measure incomplete;
// the output names every gap, gives the figures it has, and states no saving.
func TestAMeasureShortOfALandingStatesNoSaving(t *testing.T) {
	repo, heads := fixture(t, "21", "22", "23")
	logs := t.TempDir()
	writeLog(t, logs, "preflight-21.log", untimedTail, heads["21"][:12])
	writeLog(t, logs, "preflight-23.log", fullTail, heads["23"][:12])
	var out, errb bytes.Buffer
	code := run([]string{"-ref", "main", "-n", "3", logs}, repo.Root(), &out, &errb)
	if code != exitIncomplete {
		t.Fatalf("run exited %d, want %d; stderr: %s", code, exitIncomplete, errb.String())
	}
	got := out.String()
	for _, w := range []string{
		"| #22 | " + heads["22"][:12] + " | no report here |",
		"| #23 | " + heads["23"][:12] + " | preflight-23.log | 5 of 16 | 91 of 133 | 41m59s | 47m05s | 1m08s |",
		"| #21 | " + heads["21"][:12] + " | preflight-21.log | 8 of 16 | 110 of 132 | not printed | not printed | not printed |",
		"unit 13 of 32 (40.6%), race 201 of 265 (75.8%), both 214 of 297 (72.1%)",
		"0 of 133 packages reused, and no time recorded",
		"No saving is stated",
		"#22 has no report here",
		"#21's report prints no time",
		"no timed run that reused nothing was given",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("output lacks %q:\n%s", w, got)
		}
	}
	if strings.Contains(got, "Saving:") {
		t.Errorf("an incomplete measure states a saving:\n%s", got)
	}
}

// TestACompleteMeasureStatesTheSavingAgainstTheColdRun: with a timed report for
// every landing and a timed run that reused nothing, the saving is the cold
// run's time less the landings' median, for the whole check and the race stage;
// a records-only landing counts as a landing and stays out of both medians.
func TestACompleteMeasureStatesTheSavingAgainstTheColdRun(t *testing.T) {
	repo, heads := fixture(t, "31", "32")
	logs := t.TempDir()
	writeLog(t, logs, "preflight-31.log", recordsTail, heads["31"][:12])
	writeLog(t, logs, "preflight-32.log", fullTail, heads["32"][:12])
	cold := writeLog(t, t.TempDir(), "preflight-cold.log", coldTail, "cccccccccccc")
	var out, errb bytes.Buffer
	code := run([]string{"-ref", "main", "-n", "2", "-baseline", cold, logs}, repo.Root(), &out, &errb)
	if code != 0 {
		t.Fatalf("run exited %d; stderr: %s\n%s", code, errb.String(), out.String())
	}
	got := out.String()
	for _, w := range []string{
		"| #31 | " + heads["31"][:12] + " | preflight-31.log | records-only | records-only | records-only | 4m09s | 7m54s |",
		"preflight-cold.log, head cccccccccccc, reused 0 of 149 packages: race stage 1h02m00s, checked in 1h08m40s",
		"Saving: the median check of the 1 landings that ran test stages took 47m05s against the cold run's 1h08m40s, 21m35s less; its race stage 41m59s against 1h02m00s, 20m01s less.",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("output lacks %q:\n%s", w, got)
		}
	}
}

// TestABaselineThatReusedIsRefused: the baseline is a run that reused nothing,
// so a log whose run reused a package, or that prints no time, is refused, and
// no measure is printed against it.
func TestABaselineThatReusedIsRefused(t *testing.T) {
	repo, _ := fixture(t, "41")
	logs := t.TempDir()
	for name, tail := range map[string]string{"reused.log": fullTail, "untimed.log": strings.NewReplacer("8 of 16", "0 of 16", "110 of 132", "0 of 132").Replace(untimedTail)} {
		base := writeLog(t, logs, name, tail, "dddddddddddd")
		var out, errb bytes.Buffer
		code := run([]string{"-ref", "main", "-n", "1", "-baseline", base, t.TempDir()}, repo.Root(), &out, &errb)
		if code != exitUsage || !strings.Contains(errb.String(), name) {
			t.Errorf("%s: run exited %d, want %d naming it; stderr: %s", name, code, exitUsage, errb.String())
		}
		if out.Len() != 0 {
			t.Errorf("%s: a refused baseline printed a measure:\n%s", name, out.String())
		}
	}
}

// receiptScript runs this repository's scripts/preflight-receipt.sh in repo with
// the clock named, and returns what it prints, which is what make preflight's log
// holds at the run's end.
func receiptScript(t *testing.T, repo *gittest.Repo, clock string, args ...string) string {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "preflight-receipt.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir, cmd.Env = repo.Root(), append(repo.Env(), "ABCD_PREFLIGHT_CLOCK="+clock)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("preflight-receipt.sh %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// keepLocalTier has git ignore repo's local tier without a tracked .gitignore,
// so the stage reports, the receipt and the clock leave the tree clean.
func keepLocalTier(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	exclude := filepath.Join(repo.Root(), repo.Git("rev-parse", "--git-path", "info/exclude"))
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exclude, []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mintRun ends a run at repo's HEAD the way make preflight does: the real script
// mints (or refuses, when the run began with the tree "dirty"), and then prints
// the run's time from a clock that began ago seconds since, holding notes. A full
// run writes both test stages' passing reports first, reusing unit[0] of unit[1]
// packages and race[0] of race[1].
func mintRun(t *testing.T, repo *gittest.Repo, began string, ago int64, notes, scope string, unit, race [2]int) string {
	t.Helper()
	head := repo.Git("rev-parse", "HEAD")
	local := filepath.Join(repo.Root(), ".abcd", ".work.local")
	if scope == "full" {
		tree := repo.Git("rev-parse", "HEAD^{tree}")
		for stage, c := range map[string][2]int{"unit": unit, "race": race} {
			body := fmt.Sprintf("commit %s\ntree %s\nstage %s\nmode reuse\nreused %d\npackages %d\nresult pass\n", head, tree, stage, c[0], c[1])
			path := filepath.Join(local, "preflight-reuse", head, stage+"-aaaaaaaaaaaa")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	clock := filepath.Join(local, "preflight-clock", strconv.FormatInt(time.Now().Unix()-ago, 10)+"-"+strconv.FormatInt(ago, 10))
	if err := os.MkdirAll(filepath.Dir(clock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(clock, []byte(notes), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"mint", head + " " + began}
	if scope == "records-only" {
		args = append(args, "records-only")
	}
	return receiptScript(t, repo, clock, args...)
}

// near reports whether got is want a second either way: the script's clock reads
// the run's start from its name, to the second.
func near(got, want time.Duration) bool {
	return got >= want-time.Second && got <= want+time.Second
}

// TestARefusedRunAfterAGoodOneLeavesTheGoodRunsReport: one log can hold several
// runs, and a run whose mint is refused still prints its stage and checked-in
// lines. Each run is judged alone, and only one that mints a receipt reports, so
// a good run for a landing's head followed by a refused one reports the good
// run's counts and times, never the refused run's times over the good run's
// counts. The log is the real script's output for both runs.
func TestARefusedRunAfterAGoodOneLeavesTheGoodRunsReport(t *testing.T) {
	repo, heads := fixture(t, "51", "52")
	keepLocalTier(t, repo)
	repo.Git("checkout", "-q", "--detach", heads["52"])
	good := mintRun(t, repo, "clean", 3600, "wait load 600\nstage unit 120\nstage race 1500\n", "full", [2]int{3, 7}, [2]int{5, 11})
	refused := mintRun(t, repo, "dirty", 7200, "wait load 60\nstage unit 300\nstage race 2400\n", "full", [2]int{3, 7}, [2]int{5, 11})
	repo.Git("checkout", "-q", "main")
	if !strings.Contains(refused, "no push receipt minted") || !strings.Contains(refused, "preflight: stage race: 40m00s") {
		t.Fatalf("the refused run does not print its refusal and its time:\n%s", refused)
	}
	logs := t.TempDir()
	path := filepath.Join(logs, "preflight-52.log")
	body := "preflight: load check\n" + good + "exit 0\npreflight: load check\n" + refused + "exit 0\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := readRuns(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("readRuns = %+v, want the good run alone", runs)
	}
	r := runs[0]
	if r.head != heads["52"][:12] || r.unitReused != 3 || r.unitTotal != 7 || r.raceReused != 5 || r.raceTotal != 11 ||
		r.race != 25*time.Minute || !r.timed || !near(r.checked, 50*time.Minute) || r.waited != 10*time.Minute {
		t.Errorf("readRuns = %+v, want the good run's counts, race stage 25m00s and checked in 50m00s", r)
	}

	var out, errb bytes.Buffer
	run([]string{"-ref", "main", "-n", "1", logs}, repo.Root(), &out, &errb)
	if !strings.Contains(out.String(), "| #52 | "+heads["52"][:12]+" | preflight-52.log | 3 of 7 | 5 of 11 | 25m00s |") || strings.Contains(out.String(), "40m00s") {
		t.Errorf("the measure does not give #52 the good run's figures alone:\n%s%s", out.String(), errb.String())
	}
}

// TestTheScriptsOwnLinesAreTheOnesRead ties the patterns to the wording: the
// receipt, counts, refusal and time lines come from running the real
// scripts/preflight-receipt.sh, for a full run, a records-only run and a
// refused run, so a change to what the script prints fails here rather than
// leaving the measure reading no report.
func TestTheScriptsOwnLinesAreTheOnesRead(t *testing.T) {
	dir := t.TempDir()
	logOf := func(name, out string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	repo := gittest.NewRepo(t)
	keepLocalTier(t, repo)
	repo.Write("a.txt", "one\n")
	repo.Commit("one")
	head := repo.Git("rev-parse", "HEAD")[:12]
	full := mintRun(t, repo, "clean", 3600, "wait load 600\nstage unit 120\nstage race 1500\n", "full", [2]int{3, 7}, [2]int{5, 11})
	runs, err := readRuns(logOf("full.log", full))
	if err != nil || len(runs) != 1 {
		t.Fatalf("the script's full run reads as %+v, %v:\n%s", runs, err, full)
	}
	r := runs[0]
	if r.head != head || r.scope != "full" || r.mode != "reuse" || r.unitReused != 3 || r.unitTotal != 7 || r.raceReused != 5 || r.raceTotal != 11 ||
		r.race != 25*time.Minute || !r.timed || !near(r.checked, 50*time.Minute) || r.waited != 10*time.Minute {
		t.Errorf("the script's full run reads as %+v:\n%s", r, full)
	}

	refused := mintRun(t, repo, "dirty", 3600, "wait load 600\nstage unit 120\nstage race 1500\n", "full", [2]int{3, 7}, [2]int{5, 11})
	if first, _, _ := strings.Cut(refused, "\n"); !refusedLine.MatchString(first) {
		t.Errorf("the script's refusal %q does not read as one", first)
	}
	if runs, err := readRuns(logOf("refused.log", refused)); err != nil || len(runs) != 0 {
		t.Errorf("the script's refused run reads as %+v, %v:\n%s", runs, err, refused)
	}

	rec := gittest.NewRepo(t)
	keepLocalTier(t, rec)
	classifier, err := os.ReadFile(filepath.Join("..", "..", "scripts", "records-only.sh"))
	if err != nil {
		t.Fatal(err)
	}
	rec.Write("scripts/records-only.sh", string(classifier))
	if err := os.Chmod(filepath.Join(rec.Root(), "scripts", "records-only.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec.Commit("base")
	rec.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	rec.Write(".abcd/work/decisions/dcn-2611210000000000-a-record.md", "a record\n")
	rec.Commit("a record")
	records := mintRun(t, rec, "clean", 600, "wait load 30\n", "records-only", [2]int{}, [2]int{})
	runs, err = readRuns(logOf("records.log", records))
	if err != nil || len(runs) != 1 {
		t.Fatalf("the script's records-only run reads as %+v, %v:\n%s", runs, err, records)
	}
	if r := runs[0]; r.head != rec.Git("rev-parse", "HEAD")[:12] || r.scope != "records-only" || !r.timed || !near(r.checked, 570*time.Second) || r.waited != 30*time.Second {
		t.Errorf("the script's records-only run reads as %+v:\n%s", r, records)
	}
}

// TestAFullReportWithNoCountsIsNamedAndNotCounted: a full run's report whose
// counts line is missing or reworded has no reuse figures to give, so it is
// named as unreadable, kept out of the shares and the medians, and no saving is
// stated over it; it never reads as 0 of 0.
func TestAFullReportWithNoCountsIsNamedAndNotCounted(t *testing.T) {
	repo, heads := fixture(t, "71", "72")
	logs := t.TempDir()
	writeLog(t, logs, "preflight-71.log", fullTail, heads["71"][:12])
	noCounts := strings.Replace(fullTail, "           the unit stage reused 5 of 16 packages and the race stage 91 of 133 (reuse).\n", "", 1)
	writeLog(t, logs, "preflight-72.log", noCounts, heads["72"][:12])
	cold := writeLog(t, t.TempDir(), "preflight-cold.log", coldTail, "cccccccccccc")
	var out, errb bytes.Buffer
	code := run([]string{"-ref", "main", "-n", "2", "-baseline", cold, logs}, repo.Root(), &out, &errb)
	got := out.String()
	if code != exitIncomplete {
		t.Errorf("run exited %d, want %d; stderr: %s\n%s", code, exitIncomplete, errb.String(), got)
	}
	for _, w := range []string{
		"| #72 | " + heads["72"][:12] + " | preflight-72.log | not printed | not printed |",
		"#72's report prints no reuse counts",
		"unit 5 of 16 (31.2%), race 91 of 133 (68.4%)",
		"Check time of the 1 timed landings",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("output lacks %q:\n%s", w, got)
		}
	}
	if strings.Contains(got, "0 of 0") || strings.Contains(got, "Saving:") {
		t.Errorf("a report with no counts was counted:\n%s", got)
	}
}

// runWithin runs the command and fails rather than wait past ten seconds: a read
// that opens a FIFO with no writer would block for ever.
func runWithin(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	type result struct {
		code     int
		out, err string
	}
	done := make(chan result, 1)
	go func() {
		var out, errb bytes.Buffer
		code := run(args, root, &out, &errb)
		done <- result{code, out.String(), errb.String()}
	}()
	select {
	case r := <-done:
		return r.code, r.out, r.err
	case <-time.After(10 * time.Second):
		t.Fatalf("run %v blocked", args)
	}
	return 0, "", ""
}

// TestReadsAreBounded: a path named outright must be a directory or a regular
// file, not a symlink or a FIFO; a log over the size cap is refused when named
// and named as not read when found in a directory; a line over the line cap is
// read as no line of a report.
func TestReadsAreBounded(t *testing.T) {
	repo, heads := fixture(t, "61")
	short := heads["61"][:12]
	logs := t.TempDir()
	good := writeLog(t, logs, "preflight-61.log", fullTail, short)

	t.Run("a symlink named outright", func(t *testing.T) {
		link, coldLink := filepath.Join(t.TempDir(), "link.log"), filepath.Join(t.TempDir(), "cold-link.log")
		if err := os.Symlink(good, link); err != nil {
			t.Fatal(err)
		}
		// A cold run's log, which a regular file named outright would pass as the baseline.
		if err := os.Symlink(writeLog(t, t.TempDir(), "cold.log", coldTail, "cccccccccccc"), coldLink); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{link}, {"-baseline", coldLink, logs}} {
			link := args[len(args)-1]
			if len(args) > 1 {
				link = coldLink
			}
			code, _, stderr := runWithin(t, repo.Root(), append([]string{"-ref", "main", "-n", "1"}, args...)...)
			if code != exitUsage || !strings.Contains(stderr, link) {
				t.Errorf("%v: run exited %d, want %d naming the link; stderr: %s", args, code, exitUsage, stderr)
			}
		}
	})
	t.Run("a FIFO named outright", func(t *testing.T) {
		fifo := filepath.Join(t.TempDir(), "fifo.log")
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skip("no FIFO here:", err)
		}
		for _, args := range [][]string{{fifo}, {"-baseline", fifo, logs}} {
			code, _, stderr := runWithin(t, repo.Root(), append([]string{"-ref", "main", "-n", "1"}, args...)...)
			if code != exitUsage || !strings.Contains(stderr, fifo) {
				t.Errorf("%v: run exited %d, want %d naming the FIFO; stderr: %s", args, code, exitUsage, stderr)
			}
		}
	})
	t.Run("a log over the size cap", func(t *testing.T) {
		old := maxLog
		maxLog = 4096
		t.Cleanup(func() { maxLog = old })
		dir := t.TempDir()
		writeLog(t, dir, "preflight-61.log", fullTail, short)
		big := writeLog(t, dir, "preflight-big.log", strings.Repeat("x\n", 4096)+fullTail, "bbbbbbbbbbbb")
		code, _, stderr := runWithin(t, repo.Root(), "-ref", "main", "-n", "1", big)
		if code != exitUsage || !strings.Contains(stderr, big) {
			t.Errorf("named outright: run exited %d, want %d naming it; stderr: %s", code, exitUsage, stderr)
		}
		code, out, stderr := runWithin(t, repo.Root(), "-ref", "main", "-n", "1", dir)
		if code != exitIncomplete || !strings.Contains(out, "| #61 | "+short+" | preflight-61.log | 5 of 16 |") || !strings.Contains(out, "Not read: preflight-big.log") {
			t.Errorf("in a directory: run exited %d, want %d with #61 read and the big log named as not read; stderr: %s\n%s", code, exitIncomplete, stderr, out)
		}
	})
	t.Run("a log that grows past the size cap as it is read", func(t *testing.T) {
		path := writeLog(t, t.TempDir(), "preflight-grows.log", fullTail, short)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		oldMax, oldHook := maxLog, afterStat
		maxLog = info.Size() + 16
		t.Cleanup(func() { maxLog, afterStat = oldMax, oldHook })
		// The log passes the size check, then grows past the cap before it is read.
		afterStat = func(*os.File) {
			w, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if _, err := w.WriteString(strings.Repeat("x\n", 64)); err != nil {
				t.Fatal(err)
			}
		}
		if runs, err := readRuns(path); !errors.Is(err, errUnreadable) {
			t.Errorf("readRuns = %+v, %v; want the log refused as grown past the cap, not read cut short", runs, err)
		}
	})
	t.Run("a symlink in a directory", func(t *testing.T) {
		dir := t.TempDir()
		writeLog(t, dir, "preflight-61.log", fullTail, short)
		if err := os.Symlink(good, filepath.Join(dir, "link.log")); err != nil {
			t.Fatal(err)
		}
		code, out, stderr := runWithin(t, repo.Root(), "-ref", "main", "-n", "1", dir)
		if code != exitIncomplete || !strings.Contains(out, "| #61 | "+short+" | preflight-61.log | 5 of 16 |") || !strings.Contains(out, "Not read: link.log (not a regular file)") {
			t.Errorf("run exited %d, want %d with #61 read and the link named as not read; stderr: %s\n%s", code, exitIncomplete, stderr, out)
		}
	})
	t.Run("a log in a directory that cannot be opened", func(t *testing.T) {
		dir := t.TempDir()
		writeLog(t, dir, "preflight-61.log", fullTail, short)
		locked := writeLog(t, dir, "preflight-locked.log", fullTail, "bbbbbbbbbbbb")
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(locked, 0o644) })
		if f, err := os.Open(locked); err == nil {
			f.Close()
			t.Skip("this account opens a file of mode 000")
		}
		code, out, stderr := runWithin(t, repo.Root(), "-ref", "main", "-n", "1", dir)
		if code != exitIncomplete || !strings.Contains(out, "Not read: preflight-locked.log (") || strings.Contains(out, dir) {
			t.Errorf("run exited %d, want %d with the locked log named by its base name alone and the directory named nowhere; stderr: %s\n%s", code, exitIncomplete, stderr, out)
		}
	})
	t.Run("a line over the line cap", func(t *testing.T) {
		old := maxLine
		maxLine = 256
		t.Cleanup(func() { maxLine = old })
		path := filepath.Join(t.TempDir(), "long.log")
		body := "preflight: push receipt minted for bbbbbbbbbbbb " + strings.Repeat("x", 1000) + "\npreflight: checked in 4m09s (waited 0s)\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if runs, err := readRuns(path); err != nil || len(runs) != 0 {
			t.Errorf("readRuns = %+v, %v; want a line over the cap read as no receipt", runs, err)
		}
		if runs, err := readRuns(good); err != nil || len(runs) != 1 {
			t.Errorf("readRuns of a log within the caps = %+v, %v", runs, err)
		}
	})
}

// TestALogNameCannotBreakTheTable: a log's name is printed in a table cell, so
// a | in it is escaped and a line break written out, and the row keeps its cells.
func TestALogNameCannotBreakTheTable(t *testing.T) {
	repo, heads := fixture(t, "81")
	short := heads["81"][:12]
	logs := t.TempDir()
	writeLog(t, logs, "preflight|81\n.log", fullTail, short)
	var out, errb bytes.Buffer
	run([]string{"-ref", "main", "-n", "1", logs}, repo.Root(), &out, &errb)
	if want := "| #81 | " + short + ` | preflight\|81\n.log | 5 of 16 | 91 of 133 |`; !strings.Contains(out.String(), want) {
		t.Errorf("output lacks %q:\n%s%s", want, out.String(), errb.String())
	}
}

// TestALogNameCannotRewriteTheTerminal: a log's name reaches the terminal in the
// Report column and on the "Not read:" line, so an escape, any other C0 or C1
// control and a bidi control in it are masked, never printed raw.
func TestALogNameCannotRewriteTheTerminal(t *testing.T) {
	repo, heads := fixture(t, "82")
	short := heads["82"][:12]
	logs := t.TempDir()
	good := writeLog(t, logs, "preflight-\x1b[2J\u202e\u009b82.log", fullTail, short)
	if err := os.Symlink(good, filepath.Join(logs, "link-\x1b]0;x\a.log")); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	run([]string{"-ref", "main", "-n", "1", logs}, repo.Root(), &out, &errb)
	got := out.String()
	for _, r := range []rune{0x1b, 0x07, 0x9b, 0x202e} {
		if strings.ContainsRune(got, r) {
			t.Errorf("output holds %U raw:\n%q", r, got)
		}
	}
	for _, want := range []string{
		"| #82 | " + short + " | preflight-?[2J??82.log | 5 of 16 |",
		"Not read: link-?]0;x?.log (not a regular file).",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s%s", want, got, errb.String())
		}
	}
}

// TestMedianOfAnEvenCountIsTheMeanOfTheMiddleTwo: the median of the timed
// landings takes the two middle times' mean, to the second.
func TestMedianOfAnEvenCountIsTheMeanOfTheMiddleTwo(t *testing.T) {
	got := median([]time.Duration{40 * time.Second, 10 * time.Second, 30 * time.Second, 21 * time.Second})
	if got != 26*time.Second {
		t.Errorf("median = %v, want 26s", got)
	}
}
