package sweep

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/reservesweep"
	"github.com/intentdriven/abcd/internal/gittest"
)

// workflowForge is a fake forge client for the reservation workflow: it
// lists the newest dispatched run from runs, one answer a call (the last one
// repeating), and takes or refuses `gh workflow run` by dispatchErr. It
// answers `gh repo view` with view, or hostedView when that is empty. Every
// call is recorded. onDispatch, when set, runs as the dispatch is asked.
type workflowForge struct {
	mu          sync.Mutex
	runs        []string
	view        string
	dispatchErr error
	listErr     error
	onDispatch  func()
	calls       [][]string
}

func (f *workflowForge) gh(_ string, args ...string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, args)
	switch {
	case isRepoView(args):
		return cmp.Or(f.view, hostedView), nil
	case len(args) >= 2 && args[0] == "workflow" && args[1] == "run":
		if f.onDispatch != nil {
			f.onDispatch()
		}
		return "", f.dispatchErr
	case len(args) >= 2 && args[0] == "run" && args[1] == "list":
		if f.listErr != nil {
			return "", f.listErr
		}
		if len(f.runs) == 0 {
			return "[]", nil
		}
		out := f.runs[0]
		if len(f.runs) > 1 {
			f.runs = f.runs[1:]
		}
		return out, nil
	}
	return "", errors.New("the fake forge was asked something else: " + strings.Join(args, " "))
}

// dispatches is the `gh workflow run` calls the fake was asked.
func (f *workflowForge) dispatches() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out [][]string
	for _, c := range f.calls {
		if len(c) >= 2 && c[0] == "workflow" && c[1] == "run" {
			out = append(out, c)
		}
	}
	return out
}

func (f *workflowForge) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// runListing is `gh run list --json databaseId,url` naming one run.
func runListing(t *testing.T, id int64) string {
	t.Helper()
	b, err := json.Marshal([]workflowRun{{ID: id, URL: "https://forge.example/actions/runs/" + strconv.FormatInt(id, 10)}})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// sweepClock is the run's clock in these tests: a Sunday morning, before the
// workflow's 04:37 run of the next day.
var sweepClock = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

// judgedReservations is a reservation judgement that answers rep and records
// each call's options, so a test sees it was asked with Apply false.
type judgedReservations struct {
	rep   reservesweep.Report
	err   error
	asked []reservesweep.Options
	onAsk func()
}

func (j *judgedReservations) judge(_ string, o reservesweep.Options) (reservesweep.Report, error) {
	j.asked = append(j.asked, o)
	if j.onAsk != nil {
		j.onAsk()
	}
	return j.rep, j.err
}

// spentAndKept is a judgement of two reservations: one released, so spent,
// and one held too recently, so kept.
func spentAndKept() *judgedReservations {
	return &judgedReservations{rep: reservesweep.Report{AsOf: sweepClock, Branches: []reservesweep.Branch{
		{Record: "iss-1111111111111111", Ref: "refs/heads/abcd-reserved/iss-1111111111111111", Commit: strings.Repeat("a", 40),
			Kind: reservesweep.KindPlainTombstone, Spent: true, Reason: reservesweep.SpentReleased, SettledAt: sweepClock.Add(-48 * time.Hour)},
		{Record: "itd-2222222222222222", Ref: "refs/heads/abcd-reserved/itd-2222222222222222", Commit: strings.Repeat("b", 40),
			Kind: reservesweep.KindReservation, Reason: reservesweep.KeptRecent, SettledAt: sweepClock.Add(-time.Hour)},
	}}}
}

// withWorkflow writes the reservation sweep's workflow file into the
// repository's working tree with the cron line given.
func withWorkflow(t *testing.T, repo *gittest.Repo, cronLine string) {
	t.Helper()
	dir := filepath.Join(repo.Root(), ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "name: reservation-sweep\n# cron: \"1 1 1 1 1\" in a comment is no schedule\non:\n  schedule:\n" + cronLine + "\n  workflow_dispatch:\n"
	write(t, filepath.Join(dir, "reservation-sweep.yml"), body)
}

func reservationOpts(j *judgedReservations, f *workflowForge) Options {
	o := Options{Probe: ProbeOptions{Now: at(sweepClock)}, reservations: j.judge}
	if f != nil {
		o.Forge = f.gh
	}
	return o
}

func reservationRowFor(t *testing.T, res Result, record string) Row {
	t.Helper()
	for _, r := range res.Rows {
		if r.Kind == RowReservation && r.Name == record {
			return r
		}
	}
	t.Fatalf("no reservation row named %q in %+v", record, res.Rows)
	return Row{}
}

func noWaitBetweenRunListings(t *testing.T) {
	t.Helper()
	prev := runListWait
	runListWait = 0
	t.Cleanup(func() { runListWait = prev })
}

const (
	spentRecord = "iss-1111111111111111"
	keptRecord  = "itd-2222222222222222"
	nextRun     = "its next scheduled run is at 2026-11-16T04:37:00Z"
)

// TestSpentReservationsDispatchTheWorkflowAndNameItsRun is criterion 9: a
// dry run lists the spent reservation with the action dispatch and asks the
// forge nothing; apply dispatches the reservation sweep's workflow once, on
// origin's default branch, and names the run it started, its id and URL. The
// judgement is asked with Apply false both times, so the sweep deletes no
// reservation itself.
func TestSpentReservationsDispatchTheWorkflowAndNameItsRun(t *testing.T) {
	repo, _ := originRepo(t)
	withWorkflow(t, repo, `    - cron: "37 4 * * *"`)
	noWaitBetweenRunListings(t)
	j := spentAndKept()
	f := &workflowForge{runs: []string{runListing(t, 10), runListing(t, 10), runListing(t, 11)}}

	dry, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(j, f), false)
	if err != nil {
		t.Fatal(err)
	}
	if r := reservationRowFor(t, dry, spentRecord); r.Action != ActionDispatch || !strings.Contains(r.Reason, "spent: released by a plain tombstone") ||
		!strings.Contains(r.Reason, "the sweep never deletes a reservation itself") || !hasEvidence(r, KindReservationSweep, Proven) {
		t.Fatalf("the spent reservation in the dry run: %+v", r)
	}
	if r := reservationRowFor(t, dry, keptRecord); r.Action != ActionKeep || !strings.Contains(r.Reason, "touched or held too recently") {
		t.Fatalf("the kept reservation in the dry run: %+v", r)
	}
	if n := f.callCount(); n != 0 || dry.ExitCode() != ExitActed || dry.Workflow != "" {
		t.Fatalf("a dry run asks the forge nothing and would dispatch: %d calls, exit %d, workflow %q", n, dry.ExitCode(), dry.Workflow)
	}

	res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(j, f), true)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.dispatches(); len(got) != 1 || !slices.Equal(got[0], []string{"workflow", "run", "reservation-sweep.yml", "--ref=main", pinnedArg}) {
		t.Fatalf("dispatches = %q; want one, of reservation-sweep.yml on main", got)
	}
	run := "run 11, https://forge.example/actions/runs/11"
	r := reservationRowFor(t, res, spentRecord)
	if r.Outcome != OutcomeDispatched || !strings.Contains(r.Reason, "dispatched to the reservation sweep's workflow, which deletes it ("+run+")") ||
		!hasEvidence(r, KindWorkflow, Proven) {
		t.Fatalf("the spent reservation: %+v; want it dispatched, naming %s", r, run)
	}
	if k := reservationRowFor(t, res, keptRecord); k.Outcome != OutcomeKept {
		t.Fatalf("the kept reservation: %+v", k)
	}
	if !strings.Contains(res.Workflow, "dispatched the reservation sweep's workflow on main; the newest dispatched run on main, which may be another person's dispatch, is "+run) ||
		!slices.Contains(res.Notes(), res.Workflow) {
		t.Fatalf("the result's workflow line %q, notes %q; want the run named once", res.Workflow, res.Notes())
	}
	if res.ExitCode() != ExitActed || res.Refusal != "" {
		t.Fatalf("a dispatch is an act: exit %d, refusal %q", res.ExitCode(), res.Refusal)
	}
	for _, o := range j.asked {
		if o.Apply {
			t.Fatal("the reservation sweep's judgement was asked to delete")
		}
		if o.Now == nil || !o.Now().Equal(sweepClock) {
			t.Fatal("the judgement was not given the run's clock")
		}
	}
	for _, c := range f.calls {
		if c[0] == "run" && !slices.Equal(c, []string{"run", "list", "--workflow=reservation-sweep.yml", "--branch=main", "--event=workflow_dispatch", "--limit=1", "--json=databaseId,url", pinnedArg}) {
			t.Fatalf("a run listing asked %q", c)
		}
	}
	assertPinned(t, f.calls)
}

// TestTheDispatchGoesToTheBranchOriginNamesItsDefault: the workflow is
// dispatched on the branch origin names as its default when asked, not on
// the one this checkout last fetched as its default; and a run whose URL is
// not an https one is named by its id alone.
func TestTheDispatchGoesToTheBranchOriginNamesItsDefault(t *testing.T) {
	repo, origin := originRepo(t)
	noWaitBetweenRunListings(t)
	repo.Git("push", "-q", "origin", "main:trunk")
	repo.Git("--git-dir", origin, "symbolic-ref", "HEAD", "refs/heads/trunk")
	plain := func(id int64, url string) string {
		b, err := json.Marshal([]workflowRun{{ID: id, URL: url}})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	f := &workflowForge{runs: []string{plain(10, ""), plain(11, "javascript:alert(1)")}}
	res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), f), true)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.dispatches(); len(got) != 1 || !slices.Contains(got[0], "--ref=trunk") {
		t.Fatalf("dispatches = %q; want one, on trunk", got)
	}
	if !strings.Contains(res.Workflow, "is run 11;") || strings.Contains(res.Workflow, "javascript") {
		t.Fatalf("workflow line %q; want run 11 named by its id alone", res.Workflow)
	}
}

// TestARunNotListedAfterTheDispatchIsSaidSo: when the forge lists no run
// later than the one before the dispatch, however often it is asked, the
// reservations are still dispatched, and the line says the run was not read
// back rather than naming the earlier run.
func TestARunNotListedAfterTheDispatchIsSaidSo(t *testing.T) {
	repo, _ := originRepo(t)
	noWaitBetweenRunListings(t)
	f := &workflowForge{runs: []string{runListing(t, 10)}}
	res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), f), true)
	if err != nil {
		t.Fatal(err)
	}
	if r := reservationRowFor(t, res, spentRecord); r.Outcome != OutcomeDispatched || !strings.Contains(r.Reason, "its run was not listed") {
		t.Fatalf("%+v; want it dispatched, its run not listed", r)
	}
	if !strings.Contains(res.Workflow, "the run it started was not read back (the forge listed no run later than the dispatch)") ||
		strings.Contains(res.Workflow, "runs/10") {
		t.Fatalf("workflow line %q; want the run not read back, and the earlier run not named", res.Workflow)
	}
	if len(f.dispatches()) != 1 {
		t.Fatalf("dispatched %d times; want once", len(f.dispatches()))
	}
}

// TestWithNoForgeClientTheNextScheduledTimeIsGiven: where gh is absent, or
// no forge client is given at all, nothing is dispatched and the spent
// reservation is kept, naming the workflow's next scheduled time from its
// cron; that is no fault.
func TestWithNoForgeClientTheNextScheduledTimeIsGiven(t *testing.T) {
	repo, _ := originRepo(t)
	withWorkflow(t, repo, `    - cron: "37 4 * * *"  # once a day`)
	absent := errors.New("the forge client `gh` is not installed")
	for name, f := range map[string]*workflowForge{
		"gh absent":       {listErr: absent, dispatchErr: absent},
		"no forge client": nil,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), f), true)
			if err != nil || res.Refusal != "" {
				t.Fatalf("a fallback is no fault: err %v, refusal %q", err, res.Refusal)
			}
			r := reservationRowFor(t, res, spentRecord)
			if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "kept until the workflow runs") || !strings.Contains(r.Reason, nextRun) {
				t.Fatalf("%+v; want it kept, naming %q", r, nextRun)
			}
			if !strings.Contains(res.Workflow, "the reservation sweep's workflow was not dispatched") || !strings.Contains(res.Workflow, nextRun) ||
				!strings.Contains(res.Workflow, `from its cron "37 4 * * *" in .github/workflows/reservation-sweep.yml`) {
				t.Fatalf("workflow line %q", res.Workflow)
			}
			if res.ExitCode() != ExitNothing {
				t.Fatalf("nothing was dispatched, but the run exits %d", res.ExitCode())
			}
		})
	}
}

// TestAFailedDispatchFallsBackToTheSchedule: a dispatch the forge refuses is
// asked once, and the spent reservation is kept, naming the refusal and the
// workflow's next scheduled time.
func TestAFailedDispatchFallsBackToTheSchedule(t *testing.T) {
	repo, _ := originRepo(t)
	withWorkflow(t, repo, `    - cron: '37 4 * * *'`)
	f := &workflowForge{runs: []string{runListing(t, 10)}, dispatchErr: errors.New("gh workflow run: exit status 1 (HTTP 403: Resource not accessible)")}
	res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), f), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.dispatches()) != 1 {
		t.Fatalf("dispatched %d times; want once", len(f.dispatches()))
	}
	r := reservationRowFor(t, res, spentRecord)
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the forge client failed: gh workflow run: exit status 1 (HTTP 403") || !strings.Contains(r.Reason, nextRun) {
		t.Fatalf("%+v; want it kept, naming the refusal and %q", r, nextRun)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a dispatch that failed acted on nothing, but the run exits %d", res.ExitCode())
	}
}

// TestNothingIsDispatchedInADryRunOrWhenNothingIsSpent: a dry run with a
// spent reservation, and an applied run with none spent, ask the forge
// nothing.
func TestNothingIsDispatchedInADryRunOrWhenNothingIsSpent(t *testing.T) {
	repo, _ := originRepo(t)
	f := &workflowForge{runs: []string{runListing(t, 10)}}
	if _, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), f), false); err != nil {
		t.Fatal(err)
	}
	j := spentAndKept()
	j.rep.Branches = j.rep.Branches[1:]
	res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(j, f), true)
	if err != nil {
		t.Fatal(err)
	}
	if n := f.callCount(); n != 0 {
		t.Fatalf("the forge was asked %d times: %q", n, f.calls)
	}
	if res.Workflow != "" || res.ExitCode() != ExitNothing || reservationRowFor(t, res, keptRecord).Outcome != OutcomeKept {
		t.Fatalf("an applied run with nothing spent: %+v", res)
	}
}

// TestAScheduleTheSweepCannotReadIsSaidSo: a cron outside the plain
// five-field form, a workflow file with no cron and no workflow file at all
// each give no time, saying the schedule could not be read.
func TestAScheduleTheSweepCannotReadIsSaidSo(t *testing.T) {
	for name, line := range map[string]string{
		"a step":  `    - cron: "*/15 * * * *"`,
		"a list":  `    - cron: "0 4,16 * * *"`,
		"a name":  `    - cron: "0 4 * * MON"`,
		"no cron": ``,
		"no file": "-",
	} {
		t.Run(name, func(t *testing.T) {
			repo, _ := originRepo(t)
			if line != "-" {
				withWorkflow(t, repo, line)
			}
			res, err := sweepReservations(context.Background(), repo.Root(), reservationOpts(spentAndKept(), nil), true)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(res.Workflow, "its schedule could not be read") || !strings.Contains(res.Workflow, "so no time for its next run is given") ||
				strings.Contains(res.Workflow, "next scheduled run is at") {
				t.Fatalf("workflow line %q; want the schedule said unread, with no time", res.Workflow)
			}
		})
	}
}

// TestTheCronsNextTime: the next time a plain cron names, in UTC, across a
// day, a month and a year, by the day of the week, with a day of the month
// and a day of the week both given matching either, and a cron naming no
// day that comes.
func TestTheCronsNextTime(t *testing.T) {
	for _, c := range []struct {
		expr, now, want string
	}{
		{"37 4 * * *", "2026-11-15T10:00:00Z", "2026-11-16T04:37:00Z"},
		{"37 4 * * *", "2026-11-15T04:36:59Z", "2026-11-15T04:37:00Z"},
		{"37 4 * * *", "2026-11-15T04:37:00Z", "2026-11-16T04:37:00Z"},
		{"37 4 * * *", "2026-12-31T23:59:00Z", "2027-01-01T04:37:00Z"},
		{"* * * * *", "2026-11-15T10:00:30Z", "2026-11-15T10:01:00Z"},
		{"0 0 * * 1", "2026-11-15T10:00:00Z", "2026-11-16T00:00:00Z"},
		{"0 0 * * 7", "2026-11-16T10:00:00Z", "2026-11-22T00:00:00Z"},
		{"0 12 1 * 3", "2026-11-15T10:00:00Z", "2026-11-18T12:00:00Z"},
		{"0 0 29 2 *", "2026-11-15T10:00:00Z", "2028-02-29T00:00:00Z"},
		{"0 0 31 2 *", "2026-11-15T10:00:00Z", ""},
	} {
		spec, err := parseCron(c.expr)
		if err != nil {
			t.Fatalf("%q: %v", c.expr, err)
		}
		now, _ := time.Parse(time.RFC3339, c.now)
		got, ok := spec.next(now)
		if c.want == "" {
			if ok {
				t.Errorf("%q after %s = %s; want none", c.expr, c.now, got)
			}
			continue
		}
		if !ok || got.Format(time.RFC3339) != c.want {
			t.Errorf("%q after %s = %s (%v); want %s", c.expr, c.now, got.Format(time.RFC3339), ok, c.want)
		}
	}
	for _, bad := range []string{"", "37 4 * *", "37 4 * * * 2026", "60 4 * * *", "37 24 * * *", "0 0 0 * *", "0 0 * 13 *", "0 0 * * 8", "-1 4 * * *", "1-5 4 * * *", "007 4 * * *"} {
		if _, err := parseCron(bad); err == nil {
			t.Errorf("parseCron(%q) read it", bad)
		}
	}
}

// TestAStoppedRunDispatchesNothingAndSaysSo: a run whose context ends while
// the reservations are judged refuses with no row and dispatches nothing;
// one whose context ends while the dispatch is asked keeps the spent
// reservation, says whether the dispatch was taken is not known, and is the
// run's fault.
func TestAStoppedRunDispatchesNothingAndSaysSo(t *testing.T) {
	repo, _ := originRepo(t)
	t.Run("while judging", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		j := spentAndKept()
		j.onAsk = cancel
		f := &workflowForge{}
		res, err := sweepReservations(ctx, repo.Root(), reservationOpts(j, f), true)
		if err == nil || res.ExitCode() != ExitRefused || len(res.Rows) != 0 || f.callCount() != 0 {
			t.Fatalf("err %v, exit %d, rows %+v, forge calls %d", err, res.ExitCode(), res.Rows, f.callCount())
		}
	})
	t.Run("while dispatching", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := &workflowForge{runs: []string{runListing(t, 10)}, onDispatch: cancel}
		res, err := sweepReservations(ctx, repo.Root(), reservationOpts(spentAndKept(), f), true)
		if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "the run was stopped while the reservation sweep's workflow was dispatched") {
			t.Fatalf("err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
		}
		r := reservationRowFor(t, res, spentRecord)
		if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "whether the forge took the dispatch is not known") {
			t.Fatalf("%+v", r)
		}
		if len(f.dispatches()) != 1 {
			t.Fatalf("dispatched %d times", len(f.dispatches()))
		}
	})
}

// TestTheReservationSweepsOwnJudgementIsRead: with no judgement set, the
// reservation sweep's own runs, fetching from origin; a checkout whose origin
// holds no reservation has no row and dispatches nothing, and a checkout with
// no origin has none to judge, which is no refusal.
func TestTheReservationSweepsOwnJudgementIsRead(t *testing.T) {
	repo, _ := originRepo(t)
	f := &workflowForge{}
	res, err := sweepReservations(context.Background(), repo.Root(), Options{Forge: f.gh}, true)
	if err != nil || len(res.Rows) != 0 || f.callCount() != 0 || res.ExitCode() != ExitNothing {
		t.Fatalf("err %v, rows %+v, forge calls %d, exit %d", err, res.Rows, f.callCount(), res.ExitCode())
	}
	bare, _ := sweepRepo(t)
	res, err = sweepReservations(context.Background(), bare.Root(), Options{Forge: f.gh}, false)
	if err != nil || len(res.Rows) != 0 || !strings.Contains(res.Workflow, "this checkout has no origin") {
		t.Fatalf("no origin: err %v, %+v", err, res)
	}
}

// TestReservationTextIsCleaned: a record id and a reason's detail written by
// another account reach the row with every control byte masked.
func TestReservationTextIsCleaned(t *testing.T) {
	r := reservationRow(reservesweep.Branch{Record: "iss-1\x1b[2Jx", Ref: "refs/heads/abcd-reserved/iss-1\x1b[2Jx", Commit: strings.Repeat("c", 40),
		Reason: reservesweep.KeptUnknown, Why: "a subject \x07 with ‮ a bell"}, "")
	for _, s := range []string{r.Name, r.State, r.Reason, r.Evidence[0].Detail} {
		if strings.ContainsAny(s, "\x1b\x07‮") {
			t.Fatalf("a control byte reached the row: %q", s)
		}
	}
	if r.Action != ActionKeep || !hasEvidence(r, KindReservationSweep, Hint) {
		t.Fatalf("an unknown commit is kept as a hint: %+v", r)
	}
}

// TestOnlyACronUnderOnScheduleIsRead: the schedule is read from the workflow
// file's on.schedule alone, parsed as YAML; a `cron:` in a job's steps or in
// a script is no schedule. A file with no cron there, one that does not
// parse, one whose on.schedule is written twice or given by an alias, and
// one past the size the sweep reads each give no time, saying why.
func TestOnlyACronUnderOnScheduleIsRead(t *testing.T) {
	repo, _ := originRepo(t)
	scheduleOf := func(body string) string {
		dir := filepath.Join(repo.Root(), ".github", "workflows")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, "reservation-sweep.yml"), body)
		return nextScheduled(repo.Root(), sweepClock)
	}
	steps := "jobs:\n  sweep:\n    runs-on: ubuntu-latest\n    steps:\n      - cron: \"1 1 1 1 1\"\n      - run: |\n          - cron: \"2 2 2 2 2\"\n"
	got := scheduleOf("name: reservation-sweep\non:\n  schedule:\n    - cron: \"37 4 * * *\"\n  workflow_dispatch:\n" + steps)
	if !strings.Contains(got, nextRun) || !strings.Contains(got, `from its cron "37 4 * * *"`) {
		t.Fatalf("with a schedule and crons in the steps: %q; want %q from the schedule", got, nextRun)
	}
	for name, c := range map[string]struct{ body, why string }{
		"crons only in the steps":     {"on:\n  workflow_dispatch:\n" + steps, "names no cron under on.schedule"},
		"not YAML":                    {"on: [\n  schedule:\n", "does not parse as YAML"},
		"on written twice":            {"on:\n  schedule:\n    - cron: \"37 4 * * *\"\non:\n  push:\n", "`on` written twice"},
		"schedule written twice":      {"on:\n  schedule:\n    - cron: \"37 4 * * *\"\n  schedule:\n    - cron: \"1 1 1 1 1\"\n", "on.schedule written twice"},
		"a schedule by an alias":      {"x: &s\n  - cron: \"37 4 * * *\"\non:\n  schedule: *s\n", "not a list"},
		"an entry that is no cron":    {"on:\n  schedule:\n    - cron: \"37 4 * * *\"\n    - at: \"1 1 1 1 1\"\n", "not one `cron:` value"},
		"a cron with another key":     {"on:\n  schedule:\n    - cron: \"37 4 * * *\"\n      also: \"1 1 1 1 1\"\n", "not one `cron:` value"},
		"larger than the sweep reads": {"on:\n  schedule:\n    - cron: \"37 4 * * *\"\n# " + strings.Repeat("x", workflowFileCap) + "\n", "is too large"},
	} {
		got := scheduleOf(c.body)
		if !strings.Contains(got, "its schedule could not be read") || !strings.Contains(got, c.why) || strings.Contains(got, "next scheduled run is at") {
			t.Errorf("%s: %q; want the schedule said unread because it %s", name, got, c.why)
		}
	}
}
