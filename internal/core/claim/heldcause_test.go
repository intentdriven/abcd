package claim

// heldcause_test.go holds ruling 2 of fix round 5 of iss-2611131308266006:
// a reservation's push or fetch the drive's check did not run comes back with
// its text redacted as before and the git not run still reachable through
// errors.As, so a drive tells its own stop from a failure of the act.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// refusingCheck sets, for the rest of the test, a check that refuses every
// git it is asked about with refused.
func refusingCheck(t *testing.T, refused error) {
	t.Helper()
	unset, err := gitutil.SetBeforeHookedGit(func([]string) error { return refused })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
}

// TestAnActTheDrivesCheckStoppedKeepsTheStopReachable: with the check
// refusing, a reservation and a fetch of the reservations each fail; the
// error says git was not run, and the check's own error and the
// gitutil.NotRunError are reachable through errors.As and errors.Is, while
// the refusal's class (ErrRefused) holds as before. Watched failing before
// the refusal and the fetch's error carried the cause: errors.As found no
// NotRunError in either.
func TestAnActTheDrivesCheckStoppedKeepsTheStopReachable(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	o := Options{Now: w.clock, Home: p.home}
	refused := errors.New("held setting changed: core.sshcommand")
	refusingCheck(t, refused)
	_, rerr := Reserve(p.dir, Request{Record: "iss-5", Footprint: []string{"a.txt"}}, o)
	_, ferr := Fetch(p.dir, o)
	for name, err := range map[string]error{"the reservation": rerr, "the fetch": ferr} {
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) || !errors.Is(err, refused) {
			t.Errorf("%s keeps the git not run reachable: %v", name, err)
			continue
		}
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "was not run: held setting changed") {
			t.Errorf("%s is refused as before, saying git was not run: %v", name, err)
		}
	}
}

// TestAFetchTheDrivesCheckStoppedIsAskedOnce: a fetch of the reservations
// that names the default branch, stopped by the drive's check, is not tried
// again without it, so the check is asked once and the stop comes back as
// the fetch's error. Watched failing before fix round 6: the check was asked
// twice.
func TestAFetchTheDrivesCheckStoppedIsAskedOnce(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	refused := errors.New("held setting changed: core.sshcommand")
	fetches := 0
	unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
		if gitutil.Subcommand(args) == "fetch" {
			fetches++
		}
		return refused
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	_, ferr := Fetch(p.dir, Options{Now: w.clock, Home: p.home})
	var nr *gitutil.NotRunError
	if !errors.As(ferr, &nr) || fetches != 1 {
		t.Fatalf("the stopped fetch is asked once and comes back: asked %d times, %v", fetches, ferr)
	}
}

// TestASpotCheckFetchTheDrivesCheckStoppedIsAskedOnce: the spot-check
// branch's fetch, stopped by the drive's check, comes back as that stop
// without listing origin, so the check is asked once. Watched failing on the
// merge of main: the listing asked a second time.
func TestASpotCheckFetchTheDrivesCheckStoppedIsAskedOnce(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	refused := errors.New("held setting changed: core.sshcommand")
	var asked []string
	unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
		asked = append(asked, gitutil.Subcommand(args))
		return refused
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	found, ferr := FetchBranch(context.Background(), p.dir, spotcheckBranchRef, spotcheckTrackingRef)
	var nr *gitutil.NotRunError
	if found || !errors.As(ferr, &nr) || !errors.Is(ferr, refused) || len(asked) != 1 {
		t.Fatalf("the stopped fetch is asked once and comes back as the stop: asked %q, %v, %v", asked, found, ferr)
	}
}

// TestARedactedTextKeepsOnlyAGitNotRun: redacted keeps an error it quotes
// only when that is a git the check did not run, so an ordinary failure's
// cause (a deadline, git's exit) stays out of reach as it was.
func TestARedactedTextKeepsOnlyAGitNotRun(t *testing.T) {
	plain := errors.New("exit status 128")
	if r := refuse("%s; %s", redacted(plain), nothingPushed); errors.Unwrap(r) != nil || errors.Is(r, plain) {
		t.Fatalf("an ordinary failure is quoted as text alone: %v", errors.Unwrap(r))
	}
	stop := &gitutil.NotRunError{Verb: "push", Err: errors.New("changed")}
	r := refuse("%s; %s", redacted(stop), nothingPushed)
	var nr *gitutil.NotRunError
	if !errors.As(r, &nr) || nr != stop || r.Error() != "git push was not run: changed; "+nothingPushed {
		t.Fatalf("a git not run is quoted as text and kept: %v", r)
	}
	if e := redacted(stop).err(); !errors.As(e, &nr) || e.Error() != stop.Error() {
		t.Fatalf("as an error, too: %v", e)
	}
}
