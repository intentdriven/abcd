package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	r, ok, err := readReport(path)
	if err != nil || !ok {
		t.Fatalf("readReport = ok %v, err %v", ok, err)
	}
	want := report{
		path: path, head: "a91bcdac3472", scope: "full", mode: "reuse",
		unitReused: 5, unitTotal: 16, raceReused: 91, raceTotal: 133,
		race: 41*time.Minute + 59*time.Second, checked: 47*time.Minute + 5*time.Second, waited: 68 * time.Second,
		timed: true,
	}
	if r != want {
		t.Errorf("readReport =\n %+v\nwant\n %+v", r, want)
	}
}

// TestARecordsOnlyLogHasATimeAndNoCounts: a records-only landing ran no test
// stage, so it reports a time and no reuse counts.
func TestARecordsOnlyLogHasATimeAndNoCounts(t *testing.T) {
	path := writeLog(t, t.TempDir(), "preflight-r.log", recordsTail, "81739e558641")
	r, ok, err := readReport(path)
	if err != nil || !ok {
		t.Fatalf("readReport = ok %v, err %v", ok, err)
	}
	if r.scope != "records-only" || r.head != "81739e558641" || r.checked != 4*time.Minute+9*time.Second || !r.timed || r.unitTotal != 0 {
		t.Errorf("readReport = %+v", r)
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
	if _, ok, err := readReport(path); ok || err != nil {
		t.Errorf("readReport = ok %v, err %v; want no report and no error", ok, err)
	}
}

// TestTheLastRunInALogIsItsReport: a log that holds a run and then a rerun
// reports the rerun's figures, not a mix of the two.
func TestTheLastRunInALogIsItsReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preflight-twice.log")
	body := strings.Replace(fullTail, "%s", "aaaaaaaaaaaa", 1) + strings.Replace(untimedTail, "%s", "bbbbbbbbbbbb", 1)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r, ok, err := readReport(path)
	if err != nil || !ok {
		t.Fatalf("readReport = ok %v, err %v", ok, err)
	}
	if r.head != "bbbbbbbbbbbb" || r.timed || r.raceReused != 110 {
		t.Errorf("readReport = %+v, want the second run's untimed report", r)
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

// TestMedianOfAnEvenCountIsTheMeanOfTheMiddleTwo: the median of the timed
// landings takes the two middle times' mean, to the second.
func TestMedianOfAnEvenCountIsTheMeanOfTheMiddleTwo(t *testing.T) {
	got := median([]time.Duration{40 * time.Second, 10 * time.Second, 30 * time.Second, 21 * time.Second})
	if got != 26*time.Second {
		t.Errorf("median = %v, want 26s", got)
	}
}
