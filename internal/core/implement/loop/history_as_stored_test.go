package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The loop's judgements of ancestry and of what a commit holds read history as
// stored: a refs/replace entry or a .git/info/grafts line that gives a commit
// other parents, or stands one object in for another, changes none of them.

// standParents gives commit the parents named as git presents it, through a
// refs/replace entry, or, with graftFile, a line of the common directory's
// info/grafts. The stored commit is not changed.
func standParents(t *testing.T, repo *gittest.Repo, graftFile bool, commit string, parents ...string) {
	t.Helper()
	if !graftFile {
		repo.Git(append([]string{"replace", "--graft", commit}, parents...)...)
		return
	}
	common := repo.Git("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err := os.MkdirAll(filepath.Join(common, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	line := strings.Join(append([]string{commit}, parents...), " ") + "\n"
	f, err := os.OpenFile(filepath.Join(common, "info", "grafts"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// presentedHolds reports whether git, as it presents history, walks from head
// to commit.
func presentedHolds(repo *gittest.Repo, commit, head string) bool {
	return slices.Contains(strings.Fields(repo.Git("rev-list", head)), commit)
}

// threeCommits is a repository with a base commit on main and two commits
// whose only parent is the base: the stored history holds neither in the
// other.
func threeCommits(t *testing.T) (repo *gittest.Repo, base, one, two string) {
	t.Helper()
	repo = gittest.NewRepo(t)
	repo.Commit("base")
	base = repo.Git("rev-parse", "HEAD")
	one = repo.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "one")
	two = repo.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "two")
	return repo, base, one, two
}

// TestALandedSiblingIsPlacedOnTheLaneAsStored: a sibling's landed head the
// lane's head does not store is a sibling the sync merges in, whatever parents
// a replace ref or a graft gives the lane's head.
func TestALandedSiblingIsPlacedOnTheLaneAsStored(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			repo, base, landed, head := threeCommits(t)
			standParents(t, repo, graftFile, head, base, landed)
			if !presentedHolds(repo, landed, head) {
				t.Fatal("git presents the landed head on the lane's head")
			}
			st := State{Lanes: []Lane{
				{ID: "lane-1", Stage: StageDone, Landing: &Landing{Pushed: landed}},
				{ID: "lane-2", HeadSHA: head},
			}}
			got, err := landedSiblings(Context{RepoRoot: repo.Root(), State: st}, st.Lanes[1])
			if err != nil || !slices.Equal(got, []string{"lane-1"}) {
				t.Fatalf("lane-1 landed a head lane-2 does not store: %v %v", got, err)
			}
		})
	}
}

// TestAResyncPlacesTheDefaultBranchOnTheHeadAsStored: a lane whose stored
// head does not hold the default branch's tip goes on to merge it, through
// the resync's check and the merge's own, whatever parents a replace ref gives
// the head. The lane's branch is missing here, so the merge stops at reading
// it: a check that read the replaced parents would stop earlier, the resync as
// a wait on the forge, or the merge as nothing to do.
func TestAResyncPlacesTheDefaultBranchOnTheHeadAsStored(t *testing.T) {
	repo, base, moved, head := threeCommits(t)
	repo.Git("update-ref", "refs/remotes/origin/main", moved)
	standParents(t, repo, false, head, base, moved)
	if !presentedHolds(repo, moved, head) {
		t.Fatal("git presents the default branch's tip on the lane's head")
	}
	lane := Lane{ID: "lane-1", Branch: "build/missing", PR: 7, HeadSHA: head, Landing: &Landing{Pushed: head}}
	_, err := resyncLane(Context{RepoRoot: repo.Root()}, &lane, "main", "refs/remotes/origin/main", mergeability{})
	r, ok := AsRefusal(err)
	if !ok || r.Contention || !strings.Contains(r.Reason, "build/missing cannot be read") {
		t.Fatalf("the resync goes on to merge a tip the head does not store: %v", err)
	}
}

// TestASyncFoundOnTheBranchIsReadAsStored: a branch tip that moved past the
// judged head is taken as the sync's merge only when its stored parents are
// the judged head and the merged tip; a replace ref giving another commit
// those parents is refused as a moved branch.
func TestASyncFoundOnTheBranchIsReadAsStored(t *testing.T) {
	repo, base, moved, head := threeCommits(t)
	tip := repo.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "not a sync")
	repo.Git("update-ref", "refs/remotes/origin/main", moved)
	repo.Git("update-ref", "refs/heads/build/lane", tip)
	standParents(t, repo, false, tip, head, moved)
	lane := Lane{ID: "lane-1", Branch: "build/lane", HeadSHA: head}
	_, _, err := mergeDefault(Context{RepoRoot: repo.Root()}, &lane, "main", "refs/remotes/origin/main", nil)
	r, ok := AsRefusal(err)
	if !ok || !strings.Contains(r.Reason, "which is not a sync of the judged head") {
		t.Fatalf("a tip whose stored parents are not the sync's is refused: %v", err)
	}
}

// TestASyncReceiptIsReadAsStored: a sync's receipt over a head that does not
// store the merged sha is refused, whatever parents a replace ref gives it.
func TestASyncReceiptIsReadAsStored(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4")})
	f.bothImplementersOut(t)
	f.implement(t, "lane-1", "same.txt")
	l2 := f.lane(t, "lane-2")
	if err := os.WriteFile(filepath.Join(l2.Worktree, "same.txt"), []byte("lane two's words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l2.Worktree, "add", "--", "same.txt")
	f.repo.Git("-C", l2.Worktree, "commit", "-q", "-m", "lane two")
	f.receipt(t, "lane-2", f.await(t, "lane-2", RoleImplementer), strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD")))
	f.roundPassed(t, "lane-1")
	f.roundPassed(t, "lane-2")
	f.landed(t, "lane-1")
	judged := f.lane(t, "lane-2").HeadSHA
	f.step(t)
	l2 = f.lane(t, "lane-2")
	if len(l2.Syncs) != 1 || !l2.Syncs[0].Conflicted {
		t.Fatalf("the sync conflicts: %+v", l2)
	}
	if res := f.step(t); res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("a fresh implementer is handed the sync brief: %+v", res)
	}
	merged := l2.Syncs[0].Merged
	plain := laneCommit(t, f.repo, l2, "other.txt")
	standParents(t, f.repo, false, plain, judged, merged)
	if !presentedHolds(f.repo, merged, plain) {
		t.Fatal("git presents the merged sha on the receipt's head")
	}
	a := f.await(t, "lane-2", RoleImplementer)
	laneDir := f.abs(RunRelDir + "/" + f.runID + "/lane-2")
	rel, _ := filepath.Rel(laneDir, filepath.Dir(f.abs(a.Receipt)))
	for name, body := range map[string]string{filepath.Join(rel, ReportFileName): "merged\n", filepath.Join(rel, DoDFileName): "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: "lane-2", Branch: l2.Branch, Commits: []string{plain},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: filepath.ToSlash(filepath.Join(rel, DoDFileName))},
		Report:           filepath.ToSlash(filepath.Join(rel, ReportFileName)), Model: "claude-mythos-5"}
	data, _ := json.Marshal(rc)
	if err := os.WriteFile(f.abs(a.Receipt), data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Receipt(f.repo.Root(), f.runID, f.abs(a.Receipt), f.stages, f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "does not contain") {
		t.Fatalf("a receipt whose stored head does not contain the merged sha is refused: %+v", r)
	}
}

// TestAReceiptPlacesItsCommitsAsStored: a receipt naming a commit the lane's
// branch does not store, or one the lane's base stores, is refused, whatever
// parents a replace ref gives the branch's commits or the base.
func TestAReceiptPlacesItsCommitsAsStored(t *testing.T) {
	cases := map[string]struct {
		arrange func(t *testing.T, repo *gittest.Repo, l Lane, own string) string
		says    string
	}{
		"a commit off the branch": {func(t *testing.T, repo *gittest.Repo, l Lane, own string) string {
			side := repo.Git("commit-tree", l.BaseSHA+"^{tree}", "-p", l.BaseSHA, "-m", "side")
			standParents(t, repo, false, own, repo.Git("rev-parse", own+"^"), side)
			if !presentedHolds(repo, side, "refs/heads/"+l.Branch) {
				t.Fatal("git presents the side commit on the branch")
			}
			return side
		}, "not on " + "build/"},
		"a commit the base holds": {func(t *testing.T, repo *gittest.Repo, l Lane, own string) string {
			old := repo.Git("rev-parse", "--verify", l.BaseSHA+"^")
			standParents(t, repo, false, l.BaseSHA)
			standParents(t, repo, false, own, repo.Git("rev-parse", own+"^"), old)
			if presentedHolds(repo, old, l.BaseSHA) || !presentedHolds(repo, old, "refs/heads/"+l.Branch) {
				t.Fatal("git presents the base's own commit on the branch past the base")
			}
			return old
		}, "already on the default branch at the lane's base"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo, runID, l, dir := awaitingLane(t)
			own := laneCommit(t, repo, l, "one.txt")
			named := c.arrange(t, repo, l, own)
			path := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, own, named))
			_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, shortSHA(named)+" ("+c.says) {
				t.Fatalf("the receipt is refused naming %s as %q: %+v", shortSHA(named), c.says, r)
			}
		})
	}
}

// TestAReceiptReadsThePickKeptAsStored: a branch whose first commit was
// amended away from the pick's record-only commit is refused, whatever parent
// a replace ref gives the amended commit.
func TestAReceiptReadsThePickKeptAsStored(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)}})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runID := res.Start.RunID
	advanceTo(t, repo, runID, StageImplement)
	if _, err := advance(repo.Root(), runID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	if !isFullHex(l.PickSHA) {
		t.Fatalf("the lane records its pick commit: %+v", l)
	}
	repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "-m", "chore: reword the first commit")
	amended := strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
	standParents(t, repo, false, amended, l.PickSHA)
	work := laneCommit(t, repo, l, "work.txt")
	if !presentedHolds(repo, l.PickSHA, work) {
		t.Fatal("git presents the pick's commit on the branch")
	}
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, "lane-1")
	rel := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, work))
	_, err = Receipt(repo.Root(), runID, rel, DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "no longer carries the pick's record-only commit "+shortSHA(l.PickSHA)) {
		t.Fatalf("a branch that does not store the pick commit is refused: %+v", r)
	}
}

// TestAPickCommitIsReadAsStored: the commit the worktree stage adopts as the
// pick's record-only commit is judged by what it stores. A crafted first
// commit is refused when a replace ref stands the genuine commit in for it,
// when a graft places the genuine commit under it, and when a replace ref
// stands a record that matches it in for the base's.
func TestAPickCommitIsReadAsStored(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", "")},
	})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	first := st.Lanes[0]
	if _, err := worktreeStage(c, &first); err != nil {
		t.Fatal(err)
	}
	ir10, _ := pickRel("10")
	rel := filepath.ToSlash(ir10)
	wt, base, genuine := first.Worktree, first.BaseSHA, first.PickSHA
	if !isFullHex(genuine) {
		t.Fatalf("the worktree stage made the pick commit: %+v", first)
	}
	record, err := os.ReadFile(filepath.Join(wt, ir10))
	if err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", wt, "reset", "-q", "--hard", base)
	baseRecord, err := os.ReadFile(filepath.Join(wt, ir10))
	if err != nil {
		t.Fatal(err)
	}
	// crafted commits the picked intent's record, gutted of its criteria,
	// on the base with the pick's subject, and returns its object name.
	crafted := func(t *testing.T) string {
		t.Helper()
		repo.Git("-C", wt, "reset", "-q", "--hard", base)
		if err := os.WriteFile(filepath.Join(wt, ir10), []byte(strings.Replace(string(record), gwt, "", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", wt, "commit", "-q", "-am", pickSubject(st))
		return strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD"))
	}
	for name, arrange := range map[string]func(t *testing.T, made string) (undo []string){
		"the genuine commit standing in for it": func(t *testing.T, made string) []string {
			repo.Git("replace", made, genuine)
			return []string{made}
		},
		"the genuine commit grafted under it": func(t *testing.T, made string) []string {
			standParents(t, repo, false, made, genuine)
			return []string{made}
		},
		"a matching record standing in for the base's": func(t *testing.T, made string) []string {
			was := filepath.Join(t.TempDir(), "was.md")
			if err := os.WriteFile(was, []byte(strings.Replace(string(baseRecord), gwt, "", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			stored := repo.Git("rev-parse", base+":"+rel)
			repo.Git("replace", stored, repo.Git("hash-object", "-w", was))
			return []string{stored}
		},
	} {
		t.Run(name, func(t *testing.T) {
			made := crafted(t)
			for _, ref := range arrange(t, made) {
				t.Cleanup(func() { repo.Git("replace", "-d", ref) })
			}
			lane := st.Lanes[0]
			_, err := worktreeStage(c, &lane)
			r := mustRefusal(t, err)
			if !strings.Contains(r.Reason, "is not the pick's record commit") || lane.PickSHA == made || lane.PickSHA == genuine {
				t.Fatalf("a crafted commit is refused, never adopted: %+v (pick %s)", r, lane.PickSHA)
			}
		})
	}
}

// TestADrainPlacesAPushedHeadOnTheDefaultBranchAsStored: a parked lane whose
// pushed head the default branch does not store is not finished, whatever
// parents a replace ref gives the default branch's tip.
func TestADrainPlacesAPushedHeadOnTheDefaultBranchAsStored(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	tip := f.repo.Git("rev-parse", "refs/remotes/origin/main")
	parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", tip))[1:]
	standParents(t, f.repo, false, tip, append(parents, l.Landing.Pushed)...)
	if !presentedHolds(f.repo, l.Landing.Pushed, tip) {
		t.Fatal("git presents the pushed head on the default branch")
	}
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 0 {
		t.Fatalf("a pushed head the default branch does not store is not finished: %+v", res.Finished)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Complete() || st.Lanes[0].Stage != StageLand {
		t.Fatalf("the lane stays parked at its landing: %+v", st.Lanes[0])
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("the lane keeps its worktree: %v", err)
	}
}
