package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A lane's reservation narrows to the files its branch changed
// (spc-2610161044461448 step 12; iss-2610152125101468; dcn-2610161040385152):
// a drain lane that reserved a directory has its footprint narrowed, at the
// first renewal once its validators have passed and it is at its landing, to
// the files its branch changed within it, so other accounts may take the rest
// of the directory, and a narrowing never names a path outside the footprint
// it narrows.

// originFootprint is the footprint the tip of origin's reservation of id names.
func originFootprint(t *testing.T, bare, id string) []string {
	t.Helper()
	sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id)
	if sha == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(gitIn(t, bare, "log", "-1", "--format=%B", sha), "\n") {
		if v, ok := strings.CutPrefix(line, "Abcd-Footprint: "); ok {
			out = append(out, v)
		}
	}
	return out
}

// commitIn commits each of names, written fresh, in the lane's worktree.
func commitIn(t *testing.T, repo *gittest.Repo, l Lane, names ...string) string {
	t.Helper()
	for _, name := range names {
		p := filepath.Join(l.Worktree, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(name+" changed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", l.Worktree, "add", "--", name)
	}
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "lane: "+strings.Join(names, ", "))
	return strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
}

// narrowFixture is a drain lane over the documentation issue, whose found_at
// names the directory docs/, opened in a checkout whose origin is a bare
// repository, its working window long enough for every renewal a test makes.
type narrowFixture struct {
	repo   *gittest.Repo
	bare   string
	runID  string
	stages Stages
}

func newNarrowFixture(t *testing.T) *narrowFixture {
	t.Helper()
	return newNarrowFixtureAt(t, "docs/", "docs")
}

// newNarrowFixtureAt is newNarrowFixture with the documentation issue's
// found_at, and the footprint its reservation names, given.
func newNarrowFixtureAt(t *testing.T, foundAt, reserved string) *narrowFixture {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	repo.Write("docs/guide.md", "the guide\n")
	repo.Write("docs/page.md", "the page\n")
	repo.Commit("the docs")
	issueRepoRule(t, repo)
	fileIssueAt(t, repo, eligibleIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy, "AGENTS.md")
	fileIssueAt(t, repo, secondIssue, capture.SeverityMinor, capture.Category("documentation"), "Name the flag in the page.", foundAt)
	repo.Commit("the ledger")
	asPat(t, repo)
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")

	pace := "480/10"
	first := drainToLane(t, repo, Options{Pace: &pace}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f := &narrowFixture{repo: repo, bare: bare, runID: first.Start.RunID, stages: DefaultStages()}
	if fp := f.footprint(t); !slices.Equal(fp, []string{reserved}) {
		t.Fatalf("the lane reserves the path its found_at names: %q", fp)
	}
	return f
}

// reserveAs pushes, as another account would, holder's live reservation of
// id on origin with footprint, claimed at at and held five hours on.
func (f *narrowFixture) reserveAs(t *testing.T, id, holder, footprint string, at time.Time) {
	t.Helper()
	msg := "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: " + holder +
		"\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: " + footprint +
		"\nAbcd-Claimed-At: " + stamp(at) + "\nAbcd-Expires-At: " + stamp(at.Add(5*time.Hour)) + "\n"
	d := filepath.Join(t.TempDir(), "other")
	gitIn(t, "", "clone", "-q", f.bare, d)
	mp := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(mp, []byte(msg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := gitIn(t, d, "hash-object", "-w", "-t", "tree", "/dev/null")
	sha := gitIn(t, d, "-c", "user.name="+holder, "-c", "user.email="+holder+"@example.invalid", "-c", "commit.gpgsign=false", "commit-tree", tree, "-F", mp)
	gitIn(t, d, "push", "-q", "origin", sha+":"+claim.RefPrefix+id)
}

// moveDefault lands another account's change of name on origin's default
// branch.
func (f *narrowFixture) moveDefault(t *testing.T, name string) {
	t.Helper()
	f.repo.Write(name, "another account's "+name+"\n")
	f.repo.Commit("another account's change of " + name)
	f.repo.Git("push", "-q", "origin", "main")
}

// footprint is the footprint origin's reservation of the issue names.
func (f *narrowFixture) footprint(t *testing.T) []string {
	t.Helper()
	return originFootprint(t, f.bare, secondIssue)
}

// tip is origin's reservation of the issue.
func (f *narrowFixture) tip(t *testing.T) reservation {
	t.Helper()
	return originReservation(t, f.bare, claim.RefPrefix+secondIssue)
}

// implement drives the lane to its implementer, commits names there, and
// hands the receipt back at o's clock, moving the lane to its validators.
func (f *narrowFixture) implement(t *testing.T, o Options, names ...string) {
	t.Helper()
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	sha := commitIn(t, f.repo, l, names...)
	dir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Resolves = []Resolution{{Issue: secondIssue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(f.repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, o); err != nil {
		t.Fatal(err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageValidate {
		t.Fatalf("the receipt moves the lane past the implement stage: %s", l.Stage)
	}
}

// awaitImplementer drives the lane to its implement stage's await, once.
func (f *narrowFixture) awaitImplementer(t *testing.T) {
	t.Helper()
	if l := currentLane(t, f.repo, f.runID); l.Stage == StageImplement && l.awaiting() != nil {
		return
	}
	stepTo(t, f.repo, f.runID, f.stages, StageImplement)
	if res, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
}

// stepAt takes one step at at, a refusal allowed: the renewal before the
// stage is what the tests read.
func (f *narrowFixture) stepAt(t *testing.T, at time.Time) {
	t.Helper()
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return at }}); err != nil {
		if _, ok := AsRefusal(err); !ok {
			t.Fatalf("a step at %s: %v", stamp(at), err)
		}
	}
}

// renewedAt reports whether origin's reservation was renewed by this holder
// at at, over the tip before.
func (f *narrowFixture) renewedAt(t *testing.T, before reservation, at time.Time) reservation {
	t.Helper()
	tip := f.tip(t)
	if tip.parent != before.commit || tip.released || tip.holder != "octo" || !tip.claimed.Equal(at) {
		t.Fatalf("the renewal is this holder's, over the last one, at %s: before %+v, after %+v", stamp(at), before, tip)
	}
	return tip
}

// toLand passes the lane's round and steps it to its landing, at the real
// clock, before any renewal falls due.
func (f *narrowFixture) toLand(t *testing.T) {
	t.Helper()
	pass := map[string]string{RoleRuthless: reviewerReturn("SHIP"), RoleSecurity: reviewerReturn("APPROVE")}
	for range 4 * len(Sequence) {
		l := currentLane(t, f.repo, f.runID)
		if l.Stage == StageLand && l.awaiting() == nil {
			return
		}
		if a := l.awaiting(); a != nil && pass[a.Role] != "" {
			path := filepath.Join(f.repo.Root(), filepath.FromSlash(a.Receipt))
			if err := os.WriteFile(path, []byte(pass[a.Role]), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
				t.Fatalf("the %s's return: %v", a.Role, err)
			}
			continue
		}
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			t.Fatalf("advancing to the landing: %v", err)
		}
	}
	t.Fatal("the lane never reached its landing")
}

// TestALanesDirectoryReservationNarrowsToItsChangedFiles: the drain reserves
// the documentation issue with the directory its found_at names; a renewal
// while the implementer works keeps the directory, and so does one while the
// validators judge the lane, since a fix round may yet change another file
// in it; the first renewal at the landing, its validators passed, narrows it
// to the files the branch changed inside it, leaving out a file the branch
// changed outside it. A resumed `abcd build` of the issue renews the narrowed
// footprint rather than reserving the directory again, so another person's
// reservation of a file the narrowing freed does not refuse it.
func TestALanesDirectoryReservationNarrowsToItsChangedFiles(t *testing.T) {
	f := newNarrowFixture(t)

	// While the implementer works, a renewal keeps the directory.
	at := f.tip(t).claimed.Add(61 * time.Minute)
	f.awaitImplementer(t)
	before := f.tip(t)
	f.stepAt(t, at)
	renewed := f.renewedAt(t, before, at)
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("a renewal during the implement stage keeps the directory: %q", fp)
	}

	f.implement(t, Options{Now: func() time.Time { return at }}, "docs/page.md", "README.txt")

	// While the validators judge the lane, a renewal keeps the directory.
	validating := at.Add(61 * time.Minute)
	f.stepAt(t, validating)
	renewed = f.renewedAt(t, renewed, validating)
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("a renewal while the validators judge the lane keeps the directory: %q", fp)
	}

	// The first renewal at the landing narrows the footprint.
	f.toLand(t)
	later := validating.Add(61 * time.Minute)
	f.stepAt(t, later)
	f.renewedAt(t, renewed, later)
	fp := f.footprint(t)
	if !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the footprint narrows to the branch's changed files within the directory: %q", fp)
	}
	if !covers([]string{"docs"}, fp) {
		t.Fatalf("a narrowed footprint never names a path outside the first: %q", fp)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservationRenewed && strings.Contains(e.Note, "narrowed") && strings.Contains(e.Note, "docs/page.md")
	}) {
		t.Fatalf("the run record says the footprint narrowed: %+v", st.Record)
	}

	// Another person reserves a file in the freed part of the directory, and
	// the issue's build is resumed.
	quinn := "abcd reservation: iss-99\n\nAbcd-Reservation: iss-99\nAbcd-Holder: quinn\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: docs/guide.md\n" +
		"Abcd-Claimed-At: " + stamp(later) + "\nAbcd-Expires-At: " + stamp(later.Add(5*time.Hour)) + "\n"
	d := filepath.Join(t.TempDir(), "raced")
	gitIn(t, "", "clone", "-q", f.bare, d)
	mp := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(mp, []byte(quinn), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := gitIn(t, d, "hash-object", "-w", "-t", "tree", "/dev/null")
	sha := gitIn(t, d, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false", "commit-tree", tree, "-F", mp)
	gitIn(t, d, "push", "-q", "origin", sha+":"+claim.RefPrefix+"iss-99")

	narrowed := f.tip(t)
	again, err := Start(f.repo.Root(), secondIssue, Options{Now: func() time.Time { return later.Add(time.Minute) }})
	if err != nil {
		t.Fatalf("the resume renews the narrowed reservation, which overlaps nothing: %v", err)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the resume keeps the narrowed footprint, never reserving the directory again: %q (%s)", fp, again.Reservation)
	}
	if tip := f.tip(t); tip.parent != narrowed.commit || tip.released || tip.holder != "octo" {
		t.Fatalf("the resume is a renewal of the narrowed reservation: before %+v, after %+v", narrowed, tip)
	}
}

// TestANarrowingLeavesOutWhatASyncMerged: origin's default branch moved, in a
// file of the reserved directory the lane never touched, and the lane's sync
// merged it; the narrowing at the landing names the lane's own changes, the
// diff of its head against its merge base with origin's default branch, not
// the file the merge brought.
func TestANarrowingLeavesOutWhatASyncMerged(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")

	// Another account lands a change of docs/guide.md on origin's default
	// branch.
	f.repo.Write("docs/guide.md", "another account's guide\n")
	f.repo.Commit("another account's change of the guide")
	f.repo.Git("push", "-q", "origin", "main")

	f.toLand(t)
	f.stepAt(t, time.Now())
	if l := currentLane(t, f.repo, f.runID); len(l.Syncs) != 1 || l.Syncs[0].Conflicted || l.Stage != StageValidate {
		t.Fatalf("the landing's first step merges the moved default branch: %+v", l)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("no renewal fell due, so the directory is held: %q", fp)
	}
	f.toLand(t)
	before := f.tip(t)
	due := before.claimed.Add(61 * time.Minute)
	f.stepAt(t, due)
	f.renewedAt(t, before, due)
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the narrowing names the lane's own change, not the file its sync merged: %q", fp)
	}
}

// TestANarrowingWithAnUnwritablePathIsNotMade: the lane changed, within the
// reserved directory, a file whose name no reservation may carry (a
// zero-width rune, or a control character); a narrowing that named it would
// fail the renewal, and one that left it out would free a file the lane
// still changes, so the footprint is not narrowed and the renewal is made.
func TestANarrowingWithAnUnwritablePathIsNotMade(t *testing.T) {
	for _, odd := range []string{"docs/zero\u200bwidth.md", "docs/tab\there.md"} {
		t.Run(odd, func(t *testing.T) {
			f := newNarrowFixture(t)
			f.implement(t, Options{}, "docs/page.md", odd)
			f.toLand(t)
			before := f.tip(t)
			due := before.claimed.Add(61 * time.Minute)
			f.stepAt(t, due)
			f.renewedAt(t, before, due)
			if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
				t.Fatalf("the footprint is not narrowed: %q", fp)
			}
			// The renewal says why it kept the whole footprint, once, in
			// terminal-safe words.
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			i := slices.IndexFunc(st.Record, func(e Entry) bool {
				return e.Stage == StageReservationRenewed && e.At.Equal(due)
			})
			if i < 0 {
				t.Fatalf("the renewal is recorded: %+v", st.Record)
			}
			note := st.Record[i].Note
			if strings.Count(note, "reservation kept at its full footprint: ") != 1 || !strings.Contains(note, "no reservation may carry") {
				t.Fatalf("the renewal says once why the footprint was not narrowed: %q", note)
			}
			if strings.ContainsAny(note, "\t\u200b") {
				t.Fatalf("the note is terminal-safe: %q", note)
			}
		})
	}
}

// TestALaneSentBackFromItsLandingReservesItsFootprintAgain: the first renewal
// at the landing narrows the reservation to the lane's own change, and the
// landing's sync in the same step merges the moved default branch and sends
// the lane back to a fresh round, whose fix round may change any file of the
// found_at. Before the lane goes on, its reservation names the whole found_at
// again (dcn-2610170919040486, refined).
func TestALaneSentBackFromItsLandingReservesItsFootprintAgain(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	f.moveDefault(t, "docs/guide.md")
	f.toLand(t)
	before := f.tip(t)
	due := before.claimed.Add(61 * time.Minute)
	f.stepAt(t, due)
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageValidate || len(l.Syncs) != 1 {
		t.Fatalf("the landing's sync sends the lane back to a fresh round: stage %s, syncs %d", l.Stage, len(l.Syncs))
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("a lane sent back from its landing reserves its whole found_at again: %q", fp)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservationRenewed && strings.Contains(e.Note, "narrowed")
	}) || !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservation && strings.Contains(e.Note, "footprint: docs")
	}) {
		t.Fatalf("the run record says the reservation narrowed and was widened again: %+v", st.Record)
	}
}

// TestALaneSentBackFromItsLandingIsRefusedWhenItsFootprintIsTaken: the
// narrowing freed docs/guide.md and another account reserved it; the
// landing's sync then sends the lane back to a fresh round, whose reservation
// cannot name the whole found_at again over the other account's. The step is
// refused for a person, naming the overlap, the reservation stays as it was
// and nothing of the lane is pushed.
func TestALaneSentBackFromItsLandingIsRefusedWhenItsFootprintIsTaken(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	f.moveDefault(t, "docs/guide.md")
	f.toLand(t)
	before := f.tip(t)
	due := before.claimed.Add(61 * time.Minute)
	f.reserveAs(t, "iss-99", "quinn", "docs/guide.md", due.Add(-time.Minute))
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return due }})
	r, ok := AsRefusal(err)
	if !ok || r.Contention {
		t.Fatalf("the step is refused for a person: %v", err)
	}
	if !strings.Contains(r.Reason, "iss-99") || !strings.Contains(r.Reason, "docs/guide.md") {
		t.Fatalf("the refusal names the overlap: %s", r.Reason)
	}
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the refused widening leaves the narrowed reservation: %q", fp)
	}
	l := currentLane(t, f.repo, f.runID)
	if l.Stage != StageLand || len(l.Syncs) != 0 {
		t.Fatalf("the lane stays at its landing, its sync unrecorded: stage %s, syncs %d", l.Stage, len(l.Syncs))
	}
	if out := gitIn(t, f.bare, "for-each-ref", "--format=%(refname)", "refs/heads/"+BranchPrefix); out != "" {
		t.Fatalf("nothing of the lane is pushed: %s", out)
	}
}

// TestAResumeDuringImplementPublishesAWidenedFoundAt: the issue's found_at
// named docs/page.md when its lane reserved it, and was widened to docs while
// the implementer worked. A resumed `abcd build` reserves docs, judged for
// overlap, since a tip short of the landing was never narrowed.
func TestAResumeDuringImplementPublishesAWidenedFoundAt(t *testing.T) {
	for _, taken := range []bool{false, true} {
		t.Run(map[bool]string{false: "free", true: "taken"}[taken], func(t *testing.T) {
			f := newNarrowFixtureAt(t, "docs/page.md", "docs/page.md")
			f.awaitImplementer(t)
			widenFoundAt(t, f.repo, secondIssue, "docs/")
			now := f.tip(t).claimed.Add(time.Minute)
			if taken {
				f.reserveAs(t, "iss-99", "quinn", "docs/guide.md", now)
			}
			_, err := Start(f.repo.Root(), secondIssue, Options{Now: func() time.Time { return now.Add(time.Minute) }})
			if taken {
				if r, ok := AsRefusal(err); !ok || !r.Contention {
					t.Fatalf("the widened reservation is judged for overlap and backs off: %v", err)
				}
				if fp := f.footprint(t); !slices.Equal(fp, []string{"docs/page.md"}) {
					t.Fatalf("a refused widening leaves the reservation: %q", fp)
				}
				return
			}
			if err != nil {
				t.Fatalf("the resume reserves the widened found_at: %v", err)
			}
			if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
				t.Fatalf("the resume publishes the widened found_at: %q", fp)
			}
		})
	}
}

// widenFoundAt rewrites id's found_at in the checkout's ledger to foundAt and
// commits it.
func widenFoundAt(t *testing.T, repo *gittest.Repo, id, foundAt string) {
	t.Helper()
	dir := filepath.Join(repo.Root(), ".abcd", "work", "issues", "open")
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), id+"-") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(b), "\n")
		for i, l := range lines {
			if strings.HasPrefix(l, "found_at:") {
				lines[i] = "found_at: " + strconv.Quote(foundAt)
			}
		}
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
			t.Fatal(err)
		}
		repo.Commit("widen " + id + "'s found_at")
		return
	}
	t.Fatalf("%s is not in %s", id, dir)
}
