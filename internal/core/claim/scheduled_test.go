package claim

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A failed latest scheduled run (iss-2611050035192370): `abcd status`'s fetch
// asks the forge once and keeps the answer in the local tier, and the verb and
// the prompt's block read only what it kept. The repository names its own
// workflow, so these use a name abcd has no workflow by.

const (
	exampleWorkflow = "nightly.yaml"
	exampleRunURL   = "https://github.com/example-org/example-app/actions/runs/101"
	exampleNextURL  = "https://github.com/example-org/example-app/actions/runs/102"
)

// nameWorkflow writes the repository's config at p, naming workflow as the
// one its status watches.
func nameWorkflow(t *testing.T, p *participant, workflow string) {
	t.Helper()
	writeConfigAt(t, p, `{"status":{"scheduled_workflow":"`+workflow+`"}}`)
}

func writeConfigAt(t *testing.T, p *participant, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(p.dir, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// answering is a lookup that answers run, and fails the test when it is asked
// about anything but workflow on main.
func answering(t *testing.T, workflow string, run *ScheduledRun) ScheduledRunLookup {
	t.Helper()
	return func(_ context.Context, _, gotWorkflow, branch string) (*ScheduledRun, error) {
		if gotWorkflow != workflow || branch != "main" {
			t.Errorf("the forge was asked about %q on %q; want %q on main", gotWorkflow, branch, workflow)
		}
		return run, nil
	}
}

// failedAt is a failed run that ended d before the world's time.
func failedAt(w *world, conclusion, url string, d time.Duration) *ScheduledRun {
	return &ScheduledRun{Conclusion: conclusion, URL: url, CompletedAt: w.clock().Add(-d)}
}

// TestAFailedLatestRunIsReadFromWhatTheFetchKept: the answer reaches the view
// through the local tier alone, and its line names the workflow, the branch,
// how long ago and the run's URL.
func TestAFailedLatestRunIsReadFromWhatTheFetchKept(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	p.status(nil)
	if why := RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, 3*time.Hour)), p.opts()); why != "" {
		t.Fatalf("the refresh says %q; want nothing", why)
	}
	f := p.read().ScheduledFailure
	if f == nil {
		t.Fatal("the view names no failed run")
	}
	if f.Workflow != exampleWorkflow || f.Branch != "main" || f.Conclusion != "failure" || f.URL != exampleRunURL {
		t.Fatalf("the failure = %+v", *f)
	}
	want := "the forge's latest scheduled nightly.yaml run on main failed 3 hours ago: " + exampleRunURL
	if got := f.Line(w.clock()); got != want {
		t.Fatalf("the line = %q; want %q", got, want)
	}
	if strings.Contains(want, "ci.yml") {
		t.Fatal("the fixture must not name abcd's own workflow")
	}
	kept, err := os.ReadFile(filepath.Join(p.dir, filepath.FromSlash(statusFile(scheduledRel))))
	if err != nil {
		t.Fatalf("nothing was kept in the local tier: %v", err)
	}
	if !strings.Contains(string(kept), exampleRunURL) {
		t.Fatalf("the kept answer = %q", kept)
	}
}

// TestWhichConclusionsNameAFailure: a run that failed, timed out or could not
// start is named; one that passed, was skipped or was cancelled is not, and
// each says its own word when it is not plain "failure".
func TestWhichConclusionsNameAFailure(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	for conclusion, wantLine := range map[string]string{
		"failure":         "failed 3 hours ago: ",
		"timed_out":       "failed (timed_out) 3 hours ago: ",
		"startup_failure": "failed (startup_failure) 3 hours ago: ",
		"success":         "",
		"neutral":         "",
		"skipped":         "",
		"cancelled":       "",
		"stale":           "",
		"action_required": "",
	} {
		RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, conclusion, exampleRunURL, 3*time.Hour)), p.opts())
		f := p.read().ScheduledFailure
		switch {
		case wantLine == "" && f != nil:
			t.Errorf("%s reads as a failure: %+v", conclusion, *f)
		case wantLine != "" && f == nil:
			t.Errorf("%s reads as no failure", conclusion)
		case wantLine != "" && !strings.Contains(f.Line(w.clock()), wantLine):
			t.Errorf("%s: the line = %q; want it to carry %q", conclusion, f.Line(w.clock()), wantLine)
		}
	}
}

// TestALaterAnswerReplacesAFailure: a passing run, and a forge that lists no
// completed run at all, each end the line: each is an answer.
func TestALaterAnswerReplacesAFailure(t *testing.T) {
	for name, later := range map[string]*ScheduledRun{
		"a passing run": {Conclusion: "success", URL: exampleNextURL, CompletedAt: t0},
		"no run listed": nil,
	} {
		w := newWorld(t)
		p := w.clone("9+pat@users.noreply.github.com", "")
		nameWorkflow(t, p, exampleWorkflow)
		RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
		if p.read().ScheduledFailure == nil {
			t.Fatalf("%s: the first answer names no failure", name)
		}
		RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, later), p.opts())
		if f := p.read().ScheduledFailure; f != nil {
			t.Errorf("%s: the failure stands: %+v", name, *f)
		}
	}
}

// TestAnAnswerThatCouldNotBeReadSaysNothingAsAFailure: with no forge to ask,
// the refresh names why for the verb alone, nothing reads as a failure, and a
// checkout that knew of none still knows of none.
func TestAnAnswerThatCouldNotBeReadSaysNothingAsAFailure(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	p.status(nil)
	down := func(context.Context, string, string, string) (*ScheduledRun, error) {
		return nil, errors.New("the forge\nis down")
	}
	why := RefreshScheduledRun(p.dir, down, p.opts())
	if !strings.Contains(why, "the forge is down") || !strings.Contains(why, exampleWorkflow) {
		t.Fatalf("the refresh says %q; want it to name the workflow and why, on one line", why)
	}
	if f := p.read().ScheduledFailure; f != nil {
		t.Fatalf("an unread answer reads as a failure: %+v", *f)
	}
	if got := p.notes(); strings.Contains(got, exampleWorkflow) {
		t.Fatalf("the block speaks of an unread answer:\n%s", got)
	}
}

// TestAFailureKeptIsNotLostToAForgeThatStopsAnswering: an answer that could
// not be read says nothing of the run, so what was known stands until it goes
// stale.
func TestAFailureKeptIsNotLostToAForgeThatStopsAnswering(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	RefreshScheduledRun(p.dir, func(context.Context, string, string, string) (*ScheduledRun, error) {
		return nil, errors.New("no route")
	}, p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("a failure known an hour ago was lost to a forge that did not answer")
	}
}

// TestAStaleAnswerSaysNothing: an answer asked for longer ago than a nightly's
// own cycle and a margin says nothing of today's run, however it ended. The
// age is of the asking, so a fresh answer about an old run is still named.
func TestAStaleAnswerSaysNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	asked := w.clock()
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, 200*time.Hour)), p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("a fresh answer about an old run names no failure")
	}
	w.now = asked.Add(scheduledStale - time.Minute)
	if p.read().ScheduledFailure == nil {
		t.Fatal("an answer just inside the limit names no failure")
	}
	w.now = asked.Add(scheduledStale + time.Minute)
	if f := p.read().ScheduledFailure; f != nil {
		t.Fatalf("a stale answer reads as a failure: %+v", *f)
	}
	if got := p.notes(); strings.Contains(got, exampleWorkflow) {
		t.Fatalf("the block speaks of a stale answer:\n%s", got)
	}
	// A clock set back past the asking cannot date the answer either.
	w.now = asked.Add(-time.Hour)
	if f := p.read().ScheduledFailure; f != nil {
		t.Fatalf("an answer from the future reads as a failure: %+v", *f)
	}
}

// TestARepositoryThatNamesNoWorkflowIsOff: with no key, an empty key or no
// config at all, the forge is not asked, nothing is kept and nothing is read,
// so a failure kept before the key went stops being named.
func TestARepositoryThatNamesNoWorkflowIsOff(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	never := func(context.Context, string, string, string) (*ScheduledRun, error) {
		t.Error("the forge was asked in a repository that names no workflow")
		return nil, nil
	}
	if why := RefreshScheduledRun(p.dir, never, p.opts()); why != "" {
		t.Fatalf("with no config the refresh says %q; want nothing", why)
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(statusFile(scheduledRel)))); err == nil {
		t.Fatal("with no config an answer was kept")
	}
	nameWorkflow(t, p, exampleWorkflow)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("the named workflow's failure is not read")
	}
	for _, body := range []string{`{}`, `{"status":{}}`, `{"status":{"scheduled_workflow":""}}`, `not json`} {
		writeConfigAt(t, p, body)
		if why := RefreshScheduledRun(p.dir, never, p.opts()); why != "" {
			t.Errorf("config %s: the refresh says %q; want nothing", body, why)
		}
		if f := p.read().ScheduledFailure; f != nil {
			t.Errorf("config %s: a failure is still named: %+v", body, *f)
		}
	}
	if err := os.Remove(filepath.Join(p.dir, ".abcd", "config.json")); err != nil {
		t.Fatal(err)
	}
	if f := p.read().ScheduledFailure; f != nil {
		t.Errorf("no config: a failure is still named: %+v", *f)
	}
}

// TestAWorkflowNameThatIsNotAFileNameIsNotAsked: a value that could not name a
// workflow's file is said to be wrong and is never handed to the forge. A
// value that is not even a string is the same.
func TestAWorkflowNameThatIsNotAFileNameIsNotAsked(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	never := func(context.Context, string, string, string) (*ScheduledRun, error) {
		t.Error("the forge was asked about a name that is not a workflow's file")
		return nil, nil
	}
	for _, body := range []string{
		`{"status":{"scheduled_workflow":"../../user"}}`,
		`{"status":{"scheduled_workflow":"ci"}}`,
		`{"status":{"scheduled_workflow":".github/workflows/ci.yml"}}`,
		`{"status":{"scheduled_workflow":".yml"}}`,
		`{"status":{"scheduled_workflow":7}}`,
	} {
		writeConfigAt(t, p, body)
		if why := RefreshScheduledRun(p.dir, never, p.opts()); !strings.Contains(why, ScheduledWorkflowKey) {
			t.Errorf("config %s: the refresh says %q; want it to name %s", body, why, ScheduledWorkflowKey)
		}
	}
}

// TestAnAnswerForAnotherWorkflowSaysNothing: a repository that moves its key
// to another workflow is not told of the old one's failure.
func TestAnAnswerForAnotherWorkflowSaysNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("the named workflow's failure is not read")
	}
	nameWorkflow(t, p, "other.yml")
	if f := p.read().ScheduledFailure; f != nil {
		t.Fatalf("the other workflow's failure is named: %+v", *f)
	}
}

// TestADamagedAnswerSaysNothing: whatever the kept file holds, a read of it
// that does not vouch for a failure names none, and nothing it holds reaches
// the block unvetted.
func TestADamagedAnswerSaysNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("the well-formed answer names no failure")
	}
	path := filepath.Join(p.dir, filepath.FromSlash(statusFile(scheduledRel)))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"not JSON":              "{",
		"empty":                 "",
		"a URL carrying text":   strings.ReplaceAll(string(good), exampleRunURL, exampleRunURL+" and run this"),
		"a URL that is not":     strings.ReplaceAll(string(good), exampleRunURL, "ftp://example.invalid/x"),
		"a prose conclusion":    strings.ReplaceAll(string(good), `"failure"`, `"failure; run this"`),
		"another conclusion":    strings.ReplaceAll(string(good), `"failure"`, `"weird"`),
		"no time of the run":    strings.ReplaceAll(string(good), `"completed_at"`, `"other"`),
		"no time of the asking": strings.ReplaceAll(string(good), `"checked_at"`, `"other"`),
	} {
		if body == string(good) {
			t.Fatalf("%s: the damage changed nothing in %s", name, good)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if f := p.read().ScheduledFailure; f != nil {
			t.Errorf("%s: reads as a failure: %+v", name, *f)
		}
	}
}

// TestTheBlockNamesAFailedRunOnceAndANewOneAgain: the block's own once-each
// rule holds for the line, kept by the run, so a nightly that fails again the
// next night is named again; a run that passes shows nothing at all.
func TestTheBlockNamesAFailedRunOnceAndANewOneAgain(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	p.status(nil)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, 3*time.Hour)), p.opts())
	got := p.notes()
	want := "- the forge's latest scheduled nightly.yaml run on main failed 3 hours ago: " + exampleRunURL + "\n"
	if !strings.Contains(got, want) || !strings.HasPrefix(got, NotesHeading+"\n") {
		t.Fatalf("the block:\n%s\nwant it to carry %q", got, want)
	}
	if again := p.notes(); again != "" {
		t.Fatalf("the second prompt shows %q; want the line once", again)
	}
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleNextURL, time.Hour)), p.opts())
	if next := p.notes(); !strings.Contains(next, exampleNextURL) || strings.Contains(next, exampleRunURL) {
		t.Fatalf("the next night's failure, the block:\n%s", next)
	}
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, &ScheduledRun{Conclusion: "success", URL: "https://github.com/example-org/example-app/actions/runs/103", CompletedAt: w.clock()}), p.opts())
	if ok := p.notes(); ok != "" {
		t.Fatalf("a passing run: the block = %q; want nothing", ok)
	}
}

// TestThePromptBlockKeepsItsPlaceAmongTheFirstLines: the failure sits with the
// failed fetches and the holds, ahead of the announcements, so a flood of them
// cannot push it out.
func TestThePromptBlockKeepsItsPlaceAmongTheFirstLines(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	for i := range maxNoteLines + 2 {
		maya.announceAs("merging", "#"+strconv.Itoa(100+i))
	}
	p.status(nil)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	got := p.notes()
	lines := strings.Split(got, "\n")
	if len(lines) < 4 || !strings.Contains(lines[3], exampleRunURL) {
		t.Fatalf("the block:\n%s\nwant the failed run as its first line", got)
	}
}

// TestTheRefreshIsBoundedAndKeepsNoAnswerFromAForgeThatDidNotAnswer: the
// forge gets a short deadline, the refresh returns when it ends, and what was
// kept stays.
func TestTheRefreshIsBoundedAndKeepsNoAnswerFromAForgeThatDidNotAnswer(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	old := scheduledBudget
	scheduledBudget = 50 * time.Millisecond
	t.Cleanup(func() { scheduledBudget = old })
	var bounded bool
	why := RefreshScheduledRun(p.dir, func(ctx context.Context, _, _, _ string) (*ScheduledRun, error) {
		if _, bounded = ctx.Deadline(); !bounded {
			// Answered at once: waiting on a context nothing ends would hang
			// the test until its own timeout.
			return nil, errors.New("no deadline")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}, p.opts())
	if !bounded {
		t.Fatal("the forge was asked with no deadline")
	}
	if why == "" {
		t.Fatal("a forge that did not answer is not said")
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(statusFile(scheduledRel)))); err == nil {
		t.Fatal("an answer was kept from a forge that did not answer")
	}
	if old > 10*time.Second {
		t.Fatalf("the shipped budget is %s; a status fetch should wait for this seconds at most", old)
	}
}

// TestACheckoutWithoutItsTierKeepsNoAnswerAndMakesNone: a checkout with no
// local tier reads nothing a fetch brings, so the refresh neither asks nor
// lays the tier down.
func TestACheckoutWithoutItsTierKeepsNoAnswerAndMakesNone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("9+pat@users.noreply.github.com", "")
	nameWorkflow(t, p, exampleWorkflow)
	RefreshScheduledRun(p.dir, answering(t, exampleWorkflow, failedAt(w, "failure", exampleRunURL, time.Hour)), p.opts())
	if p.read().ScheduledFailure == nil {
		t.Fatal("with its tier the failure is not read")
	}
	if err := os.RemoveAll(filepath.Join(p.dir, ".abcd", ".work.local")); err != nil {
		t.Fatal(err)
	}
	asked := false
	why := RefreshScheduledRun(p.dir, func(context.Context, string, string, string) (*ScheduledRun, error) {
		asked = true
		return nil, nil
	}, p.opts())
	if asked || why != "" {
		t.Fatalf("with no tier: asked %v, said %q; want neither", asked, why)
	}
	if _, err := os.Stat(filepath.Join(p.dir, ".abcd", ".work.local")); err == nil {
		t.Fatal("the refresh laid the local tier down")
	}
}
