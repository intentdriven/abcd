package loop

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The start and the lookup of a run with members (spc-2611201455051897 step
// 4): the lane a drain opens on a bundle starts with each member and the hold
// the drain reserved it on, as the lead starts on its own hold, and a run is
// found for any issue it still carries, so a build of a member in the copy
// that holds the bundle resumes the bundle's run and names its lead.

// reserveHeld reserves id on origin from the fixture's checkout at at, with its
// page as footprint, as the drain reserves an issue before its judgement, and
// returns the hold as a lane's start is handed it.
func reserveHeld(t *testing.T, repo *gittest.Repo, id string, at time.Time) *heldReservation {
	t.Helper()
	res, err := claim.Reserve(repo.Root(), claim.Request{Record: id, Footprint: []string{bundleFiles[id]}, Drain: true}, clockAt(at))
	if err != nil {
		t.Fatalf("reserving %s: %v", id, err)
	}
	return &heldReservation{at: res.ClaimedAt, note: "reserved " + id + " before its judgement, footprint: " + bundleFiles[id]}
}

// bundleStart starts the lead's lane at at with the two members, each on the
// hold reserveHeld took for it, as the drain's start of a bundle does.
func bundleStart(t *testing.T, repo *gittest.Repo, at time.Time) StartResult {
	t.Helper()
	o := Options{Now: func() time.Time { return at }, held: reserveHeld(t, repo, bundleLead, at)}
	for _, id := range []string{bundleTwo, bundleThree} {
		o.members = append(o.members, laneMember{issue: id, footprint: []string{bundleFiles[id]}, held: reserveHeld(t, repo, id, at)})
	}
	res, err := Start(repo.Root(), bundleLead, o)
	if err != nil {
		t.Fatalf("starting the bundle: %v", err)
	}
	return res
}

// TestAStartWithMembersWritesEachWithItsHold: the run a bundle's start creates
// carries each member with its footprint and the time its hold was claimed,
// each reservation line names its issue, and nothing more is pushed to origin.
func TestAStartWithMembersWritesEachWithItsHold(t *testing.T) {
	repo, bare := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	res := bundleStart(t, repo, at)
	if res.Resumed {
		t.Fatal("a bundle's first start creates its run")
	}
	before := tipsOf(t, bare)
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != bundleLead || len(st.Bundle) != 2 {
		t.Fatalf("the run is keyed by its lead and carries two members: key %s, bundle %+v", st.Key, st.Bundle)
	}
	for i, id := range []string{bundleTwo, bundleThree} {
		m := st.Bundle[i]
		if m.Issue != id || !slices.Equal(m.Footprint, []string{bundleFiles[id]}) || m.ReservedAt == nil || !m.ReservedAt.Equal(at) || m.HandBack != nil || m.Split {
			t.Fatalf("member %d is %s on its hold of %s: %+v", i+1, id, at, m)
		}
	}
	if st.ReservedAt == nil || !st.ReservedAt.Equal(at) {
		t.Fatalf("the lead starts on its own hold: %v", st.ReservedAt)
	}
	if !slices.Equal(st.Issues(), bundleIssues) {
		t.Fatalf("the run's issues are the lead and its members: %v", st.Issues())
	}
	lines := linesAt(st, StageReservation)
	if !slices.Equal(issuesOf(lines), bundleIssues) {
		t.Fatalf("each reservation line names its issue, the lead's first: %+v", lines)
	}
	for i, e := range lines {
		if !strings.Contains(e.Note, bundleIssues[i]) {
			t.Fatalf("the line for %s is its hold's: %q", bundleIssues[i], e.Note)
		}
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == "start" && strings.Contains(e.Note, bundleTwo) && strings.Contains(e.Note, bundleThree)
	}) {
		t.Fatalf("the start's record names the members: %+v", st.Record)
	}
	for _, id := range bundleIssues {
		if tip := tipOf(t, bare, id); tip.released || !tip.claimed.Equal(at) {
			t.Fatalf("%s stands on the hold it was started on: %+v", id, tip)
		}
	}
	if after := tipsOf(t, bare); !mapsEqualCommits(before, after) {
		t.Fatalf("the start pushed nothing: before %+v, after %+v", before, after)
	}
}

// mapsEqualCommits reports whether each issue's tip is the same commit.
func mapsEqualCommits(a, b map[string]reservation) bool {
	for id, r := range a {
		if b[id].commit != r.commit {
			return false
		}
	}
	return len(a) == len(b)
}

// TestABuildOfAMemberResumesTheBundlesRunAndNamesItsLead: in the copy that
// holds the bundle, a build of a member finds the run that carries it rather
// than opening a second one, and names the run's lead.
func TestABuildOfAMemberResumesTheBundlesRunAndNamesItsLead(t *testing.T) {
	repo, _ := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	started := bundleStart(t, repo, at)
	res, err := Start(repo.Root(), bundleThree, Options{Now: func() time.Time { return at.Add(time.Minute) }})
	if err != nil {
		t.Fatalf("a build of a member resumes its bundle: %v", err)
	}
	if !res.Resumed || res.RunID != started.RunID {
		t.Fatalf("the bundle's run %s is resumed, got %+v", started.RunID, res)
	}
	if res.Lead != bundleLead {
		t.Fatalf("the result names the run's lead %s: %q", bundleLead, res.Lead)
	}
	if !slices.ContainsFunc(res.Diagnostics, func(d string) bool {
		return strings.Contains(d, bundleThree) && strings.Contains(d, bundleLead) && strings.Contains(d, started.RunID)
	}) {
		t.Fatalf("a line says the member is the bundle's and names its lead: %q", res.Diagnostics)
	}
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("no second run is opened on the member: %d runs", len(runs))
	}
	// A build of the lead resumes it as today, naming no lead of its own.
	lead, err := Start(repo.Root(), bundleLead, Options{Now: func() time.Time { return at.Add(2 * time.Minute) }})
	if err != nil || !lead.Resumed || lead.RunID != started.RunID || lead.Lead != "" {
		t.Fatalf("a build of the lead resumes the run as its own: %+v, %v", lead, err)
	}
}

// TestAMemberHandedBackAloneOpensItsOwnRun: a member handed back alone while
// its bundle is parked is no longer carried by it, so a build of it opens the
// member's own run rather than resuming the bundle.
func TestAMemberHandedBackAloneOpensItsOwnRun(t *testing.T) {
	repo, _ := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	started := bundleStart(t, repo, at)
	if err := mutate(repo.Root(), started.RunID, func(_ *os.Root, st *State) (bool, error) {
		st.Bundle[0].HandBack = &HandBack{At: at, Kind: "design", Reason: "the page's sentence needs a decision", Home: "the person"}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res, err := Start(repo.Root(), bundleTwo, Options{Now: func() time.Time { return at.Add(time.Minute) }})
	if err != nil {
		t.Fatalf("a build of a member handed back alone opens its own run: %v", err)
	}
	if res.Resumed || res.RunID == started.RunID || res.Lead != "" {
		t.Fatalf("a new run is opened for the member: %+v", res)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != bundleTwo || st.hasMembers() {
		t.Fatalf("the new run is the member's own, with no members: key %s, bundle %+v", st.Key, st.Bundle)
	}
	// The member still carried resumes the bundle.
	three, err := Start(repo.Root(), bundleThree, Options{Now: func() time.Time { return at.Add(2 * time.Minute) }})
	if err != nil || !three.Resumed || three.RunID != started.RunID {
		t.Fatalf("the member still carried resumes the bundle: %+v, %v", three, err)
	}
}

// TestLiveRunLooksUpByWhatARunCarries: the lookup finds a run for its key and
// for every member it still carries, never for a member handed back alone,
// nor for any issue of a run the drain split, and a run without members is
// found by its key alone, as before.
func TestLiveRunLooksUpByWhatARunCarries(t *testing.T) {
	single := State{RunID: "run-a", Key: "iss-2611201500000009", Lanes: []Lane{{ID: "lane-1"}}}
	bundle := State{RunID: "run-b", Key: bundleLead, Lanes: []Lane{{ID: "lane-1"}}, Bundle: []Member{
		{Issue: bundleTwo},
		{Issue: bundleThree, HandBack: &HandBack{Kind: "design"}},
	}}
	split := State{RunID: "run-c", Key: "iss-2611201500000007", Lanes: []Lane{{ID: "lane-1"}}, Bundle: []Member{
		{Issue: "iss-2611201500000008", Split: true},
	}}
	runs := []State{single, bundle, split}
	for _, c := range []struct {
		key  string
		want string
	}{
		{"iss-2611201500000009", "run-a"},
		{bundleLead, "run-b"},
		{bundleTwo, "run-b"},
		{bundleThree, ""},
		{"iss-2611201500000007", ""},
		{"iss-2611201500000008", ""},
		{"iss-2611201500000006", ""},
	} {
		st, ok := liveRun(runs, c.key, false, false)
		if got := map[bool]string{true: st.RunID}[ok]; got != c.want {
			t.Fatalf("liveRun(%s) is %q, want %q", c.key, got, c.want)
		}
	}
}

// TestAStartRefusesMembersItCannotCarry: members ride only an issue run the
// drain reserved each of, and never an issue another live run carries; each
// refusal creates no run.
func TestAStartRefusesMembersItCannotCarry(t *testing.T) {
	repo, _ := bundleRepo(t)
	at := time.Now().UTC().Truncate(time.Second)
	two := laneMember{issue: bundleTwo, footprint: []string{bundleFiles[bundleTwo]}, held: &heldReservation{at: at, note: "reserved"}}
	for name, o := range map[string]Options{
		"a member with no hold":   {members: []laneMember{{issue: bundleTwo, footprint: []string{bundleFiles[bundleTwo]}}}},
		"a member twice":          {members: []laneMember{two, two}},
		"the lead as its member":  {members: []laneMember{{issue: bundleLead, footprint: []string{"docs/a.md"}, held: two.held}}},
		"a member that is no id":  {members: []laneMember{{issue: "not-an-issue", footprint: []string{"docs/b.md"}, held: two.held}}},
		"a member with no path":   {members: []laneMember{{issue: bundleTwo, held: two.held}}},
		"members on a trial copy": {Trial: true, members: []laneMember{two}},
	} {
		o.Now = func() time.Time { return at }
		if _, err := Start(repo.Root(), bundleLead, o); err == nil {
			t.Fatalf("%s is refused", name)
		} else if _, ok := AsRefusal(err); !ok {
			t.Fatalf("%s is the loop's refusal: %v", name, err)
		}
	}
	if runs, err := Runs(repo.Root()); err != nil || len(runs) != 0 {
		t.Fatalf("no refused start creates a run: %d, %v", len(runs), err)
	}
	// An issue a live run already carries is no member of another.
	if _, err := Start(repo.Root(), bundleThree, Options{Now: func() time.Time { return at }}); err != nil {
		t.Fatal(err)
	}
	three := laneMember{issue: bundleThree, footprint: []string{bundleFiles[bundleThree]}, held: &heldReservation{at: at, note: "reserved"}}
	_, err := Start(repo.Root(), bundleLead, Options{Now: func() time.Time { return at }, held: reserveHeld(t, repo, bundleLead, at), members: []laneMember{three}})
	if r, ok := AsRefusal(err); !ok || !strings.Contains(r.Reason, bundleThree) {
		t.Fatalf("a member another run carries is refused naming it: %v", err)
	}
	if runs, err := Runs(repo.Root()); err != nil || len(runs) != 1 {
		t.Fatalf("the refused bundle creates no run: %d, %v", len(runs), err)
	}
}

// handedBackLanes is a run's one lane, handed back to the person.
func handedBackLanes() []Lane {
	return []Lane{{ID: "lane-1", Stage: StageHandedBack, HandBack: &HandBack{Kind: "design", Reason: "a decision", Home: "the person"}}}
}

// TestALaterRunThatCarriesAnIssueSupersedesTheHandedBackOne: a run handed
// back is superseded by a later run that carries its lead or any of its
// members, so a build finds the later run and never the dead one: a single
// run handed back before a bundle took its issue as a member, and a bundle
// handed back whole before its member was taken up alone.
func TestALaterRunThatCarriesAnIssueSupersedesTheHandedBackOne(t *testing.T) {
	// A single run of the member, handed back, then a bundle carrying it.
	single := State{RunID: "run-a", Key: bundleTwo, Lanes: handedBackLanes()}
	bundle := State{RunID: "run-b", Key: bundleLead, Lanes: []Lane{{ID: "lane-1"}}, Bundle: []Member{{Issue: bundleTwo, Footprint: []string{"docs/b.md"}}}}
	if !single.issueHandedBack() {
		t.Fatal("premise: the single run is handed back")
	}
	runs := []State{single, bundle}
	if !superseded(runs, 0) {
		t.Fatal("the bundle that carries the member supersedes the member's run handed back")
	}
	if st, ok := liveRun(runs, bundleTwo, false, false); !ok || st.RunID != "run-b" {
		t.Fatalf("a build of the member finds the bundle, not the run handed back: %s, %v", st.RunID, ok)
	}

	// A bundle handed back whole, then the member's own run.
	dead := State{RunID: "run-c", Key: bundleLead, Lanes: handedBackLanes(), Bundle: []Member{{Issue: bundleTwo, Footprint: []string{"docs/b.md"}}}}
	alone := State{RunID: "run-d", Key: bundleTwo, Lanes: []Lane{{ID: "lane-1"}}}
	if !dead.issueHandedBack() {
		t.Fatal("premise: the bundle is handed back")
	}
	runs = []State{dead, alone}
	if !superseded(runs, 0) {
		t.Fatal("the member's later run supersedes the bundle handed back")
	}
	if st, ok := liveRun(runs, bundleTwo, false, false); !ok || st.RunID != "run-d" {
		t.Fatalf("a build of the member finds its own later run, not the bundle: %s, %v", st.RunID, ok)
	}
	if st, ok := liveRun(runs, bundleLead, false, false); ok {
		t.Fatalf("the dead bundle is resumed for no issue it carried: %s", st.RunID)
	}

	// A run handed back that nothing later carries stays as it was, and a
	// run in progress is never superseded.
	if superseded([]State{single}, 0) || superseded([]State{bundle, {RunID: "run-e", Key: bundleTwo, Lanes: []Lane{{ID: "lane-1"}}}}, 0) {
		t.Fatal("only a run handed back is superseded, and only by a later run carrying one of its issues")
	}
}

// TestAMembersResumeThatClosesAMergedLaneSaysItOnce: a build of a member whose
// resume closes a lane whose pull request merged names the run's lead, and
// the line saying the member is the bundle's appears once.
func TestAMembersResumeThatClosesAMergedLaneSaysItOnce(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	// The fixture's run, made an issue run that carries a member.
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Key = f.issue
		st.Bundle = []Member{{Issue: bundleTwo, Footprint: []string{"docs/b.md"}}}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res, err := Start(f.repo.Root(), bundleTwo, Options{noReserve: true})
	if err != nil {
		t.Fatalf("the build of the member resumes its run: %v", err)
	}
	if !res.Resumed || res.RunID != f.runID || len(res.Closed) != 1 {
		t.Fatalf("premise: the resume closes the merged lane: %+v", res)
	}
	if res.Lead != f.issue {
		t.Fatalf("the result names the lead %s: %q", f.issue, res.Lead)
	}
	n := 0
	for _, d := range res.Diagnostics {
		if strings.Contains(d, "is a member of") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the member's line appears once, not %d times: %q", n, res.Diagnostics)
	}
}
