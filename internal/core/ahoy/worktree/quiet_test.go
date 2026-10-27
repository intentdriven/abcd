package worktree_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Quiet candidates (spc-2609301811532881, step 5; criterion 11): an unmerged
// worktree whose last activity is older than the window, with no open pull
// request for its branch, is listed with a dossier and left in place unless
// named with --yes.

// fifteenDaysOn is a clock fifteen days past every commit and file the test
// made, so each worktree it has not touched since reads fifteen days quiet.
func fifteenDaysOn() time.Time { return time.Now().Add(15 * 24 * time.Hour) }

// quietPrune is prune under the bundled window, fifteen days on.
func quietPrune(t *testing.T, repo *gittest.Repo, opts worktree.PruneOptions) worktree.Pruned {
	t.Helper()
	if opts.QuietDays == 0 {
		opts.QuietDays = worktree.DefaultQuietDays
	}
	if opts.Now == nil {
		opts.Now = fifteenDaysOn
	}
	return prune(t, repo, opts)
}

// touchAt writes rel in dir with its modification time at when.
func touchAt(t *testing.T, dir, rel string, when time.Time) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.WriteFile(p, []byte(rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, when, when); err != nil {
		t.Fatal(err)
	}
}

// forgeWithOpen is a forge whose one listing holds an open pull request from
// branch.
func forgeWithOpen(branch string) worktree.GH {
	return func(_ string, args ...string) (string, error) {
		if len(args) < 2 || args[0] != "pr" || args[1] != "list" {
			return "", fmt.Errorf("unexpected forge call %v", args)
		}
		return fmt.Sprintf(`[{"headRefName":%q,"headRefOid":"0000000000000000000000000000000000000000","baseRefName":"main","isCrossRepository":false,"state":"OPEN"}]`, branch), nil
	}
}

// TestPruneListsAQuietWorktreeWithItsDossier is criterion 11's first half: a
// fifteen-day-quiet unmerged worktree is a quiet row carrying its dossier,
// dirty state first, and stands untouched; one active within the window,
// and one with an open pull request, are not quiet.
func TestPruneListsAQuietWorktreeWithItsDossier(t *testing.T) {
	repo, home := pruneFixture(t)
	idle := addToLane(t, repo, "idle")
	commitIn(t, repo, idle, "notes/itd-2609091014076309-plan.md", "plan\n")
	repo.Git("-C", idle, "commit", "-q", "--allow-empty", "-m", "fix: the gate (iss-2610231011465325)")
	if err := os.WriteFile(filepath.Join(idle, "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(idle, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	busy := addToLane(t, repo, "busy")
	touchAt(t, busy, "today.txt", fifteenDaysOn().Add(-time.Hour))
	reviewed := addToLane(t, repo, "reviewed")
	commitIn(t, repo, reviewed, "r.txt", "r\n")

	got := quietPrune(t, repo, worktree.PruneOptions{Forge: forgeWithOpen("reviewed")})
	r := pruneRow(t, got, "idle")
	if r.Outcome != worktree.OutcomeQuiet || r.Reason != "" || r.Dossier == nil {
		t.Fatalf("the idle worktree is a quiet candidate with a dossier: %+v", r)
	}
	d := r.Dossier
	if d.Dirty.Modified != 1 || d.Dirty.Untracked != 1 || !slices.Contains(d.Dirty.Paths, "a.txt") || !slices.Contains(d.Dirty.Paths, "new.txt") {
		t.Errorf("the dirty state: %+v", d.Dirty)
	}
	if d.Commits != 2 || len(d.Subjects) != 2 || d.Subjects[0] != "fix: the gate (iss-2610231011465325)" {
		t.Errorf("the branch's subjects, newest first: %d %q", d.Commits, d.Subjects)
	}
	if !strings.Contains(d.Diff, "1 file changed") {
		t.Errorf("the diff summary from the merge base: %q", d.Diff)
	}
	if !slices.Equal(d.Records, []string{"iss-2610231011465325", "itd-2609091014076309"}) {
		t.Errorf("the records the subjects and paths cite: %q", d.Records)
	}
	if at, err := time.Parse(time.RFC3339, d.LastActivity); err != nil || fifteenDaysOn().Sub(at) < 14*24*time.Hour {
		t.Errorf("the last activity is fifteen days back (%v): %q", err, d.LastActivity)
	}
	if !strings.Contains(d.Recommendation, "commit or discard the changes, then name it") {
		t.Errorf("a dirty candidate's recommendation: %q", d.Recommendation)
	}
	js, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if s := string(js); strings.Index(s, `"dirty"`) > strings.Index(s, `"subjects"`) {
		t.Errorf("the dossier carries its dirty state first: %s", s)
	}
	if !exists(idle) || !listed(t, repo, idle) {
		t.Fatal("a quiet candidate no one named stands")
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing is archived: %v", n)
	}
	if b := pruneRow(t, got, "busy"); b.Outcome != "kept" || b.Dossier != nil || !strings.HasPrefix(b.Reason, "dirty") {
		t.Errorf("a worktree active within the window is not quiet: %+v", b)
	}
	if v := pruneRow(t, got, "reviewed"); v.Outcome != "kept" || v.Reason != "unmerged" || v.Dossier != nil {
		t.Errorf("a worktree with an open pull request is not quiet: %+v", v)
	}
	if got.Reclaimed() != 0 {
		t.Fatalf("nothing is reclaimed: %d", got.Reclaimed())
	}

	wide := quietPrune(t, repo, worktree.PruneOptions{QuietDays: 20})
	if r := pruneRow(t, wide, "idle"); r.Outcome != "kept" || r.Dossier != nil {
		t.Errorf("under a twenty-day window the idle worktree is not quiet: %+v", r)
	}
	if r := pruneRow(t, prune(t, repo, worktree.PruneOptions{Now: fifteenDaysOn}), "idle"); r.Outcome != "kept" {
		t.Errorf("a run given no window judges nothing quiet: %+v", r)
	}
}

// TestPruneYesRemovesTheNamedQuietCandidateOnly: the named candidate is
// archived and removed through prune's own path (PreRemove asked about it
// alone), its branch stands, and the other candidate is untouched; a dry run
// reports the same rows and removes nothing.
func TestPruneYesRemovesTheNamedQuietCandidateOnly(t *testing.T) {
	repo, home := pruneFixture(t)
	gone := addToLane(t, repo, "gone")
	commitIn(t, repo, gone, "g.txt", "g\n")
	if err := os.MkdirAll(filepath.Join(gone, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gone, ".abcd", ".work.local", "NEXT.md"), []byte("next\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stay := addToLane(t, repo, "stay")

	dry := quietPrune(t, repo, worktree.PruneOptions{Yes: []string{"gone"}, DryRun: true})
	if r := pruneRow(t, dry, "gone"); r.Outcome != worktree.OutcomeWouldReclaim || r.Dossier != nil || dry.Reclaimed() != 1 {
		t.Fatalf("a dry run would reclaim the named candidate: %+v", r)
	}
	if !exists(gone) || len(notesEntries(t, repo, home)) != 0 {
		t.Fatal("a dry run archives and removes nothing")
	}

	var asked []string
	got := quietPrune(t, repo, worktree.PruneOptions{Yes: []string{"gone"}, PreRemove: func(abs string) (bool, string) {
		asked = append(asked, abs)
		return false, ""
	}})
	if r := pruneRow(t, got, "gone"); r.Outcome != worktree.OutcomeReclaimed || r.Notes != "1 file" || r.Archive == "" || r.Dossier != nil {
		t.Fatalf("the named candidate is archived and reclaimed, its dossier spent: %+v", r)
	}
	if exists(gone) || listed(t, repo, gone) {
		t.Fatal("the named candidate is removed")
	}
	repo.Git("rev-parse", "--verify", "refs/heads/gone")
	if r := pruneRow(t, got, "stay"); r.Outcome != worktree.OutcomeQuiet || !exists(stay) || !listed(t, repo, stay) {
		t.Fatalf("the candidate no one named stands: %+v", r)
	}
	if len(asked) != 1 || filepath.Base(asked[0]) != "gone" {
		t.Errorf("the caller's check is asked about the named candidate alone: %v", asked)
	}
	if n := notesEntries(t, repo, home); len(n) != 1 || !strings.HasSuffix(n[0], "-gone") {
		t.Fatalf("one archive entry, for gone: %v", n)
	}
	if got.Reclaimed() != 1 {
		t.Fatalf("one reclaimed: %d", got.Reclaimed())
	}
}

// TestPruneKeepsADirtyNamedQuietCandidate: naming a candidate never forces
// it, so one holding uncommitted changes is kept with that reason, its
// dossier showing them, and nothing is archived.
func TestPruneKeepsADirtyNamedQuietCandidate(t *testing.T) {
	repo, home := pruneFixture(t)
	wip := addToLane(t, repo, "wip")
	if err := os.WriteFile(filepath.Join(wip, "half.txt"), []byte("half\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(wip, "half.txt"), time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	removed := false
	defer worktree.SwapRemoveForTest(func(string, string) error { removed = true; return nil })()

	got := quietPrune(t, repo, worktree.PruneOptions{Yes: []string{"wip"}})
	r := pruneRow(t, got, "wip")
	if r.Outcome != "kept" || r.Reason != "dirty: uncommitted changes" || r.Dossier == nil || r.Dossier.Dirty.Untracked != 1 {
		t.Fatalf("the dirty named candidate is kept with why, its dossier showing it: %+v", r)
	}
	if removed || !exists(wip) {
		t.Fatal("no removal is asked for")
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing is archived: %v", n)
	}
}

// TestPruneKeepsANamedWorktreeThatIsNotQuiet: a name in the lane that is not
// a quiet candidate this run is kept and says why: a merged one needs no
// name, an active one is not quiet.
func TestPruneKeepsANamedWorktreeThatIsNotQuiet(t *testing.T) {
	repo, _ := pruneFixture(t)
	done := mergedWorktree(t, repo, "done")
	busy := addToLane(t, repo, "busy")
	commitIn(t, repo, busy, "b.txt", "b\n")

	// An hour on, the commit just made is inside the window.
	got := quietPrune(t, repo, worktree.PruneOptions{Yes: []string{"done", "busy"}, Now: func() time.Time { return time.Now().Add(time.Hour) }})
	if r := pruneRow(t, got, "done"); r.Outcome != "kept" || !strings.Contains(r.Reason, "merged") || !strings.Contains(r.Reason, "needs no name") {
		t.Errorf("a named merged worktree: %+v", r)
	}
	if r := pruneRow(t, got, "busy"); r.Outcome != "kept" || !strings.Contains(r.Reason, "not quiet") {
		t.Errorf("a named active worktree: %+v", r)
	}
	if !exists(done) || !exists(busy) || got.Reclaimed() != 0 {
		t.Fatalf("both stand: %+v", got.Rows)
	}
}

// TestPruneNeverOffersADetachedWorktreeAsQuiet: a worktree on a detached
// HEAD holds commits no branch does, and removing it drops the HEAD reflog
// that is their last reference. So however long it has been quiet it is no
// candidate, and naming it keeps it, saying its commits are on no branch.
func TestPruneNeverOffersADetachedWorktreeAsQuiet(t *testing.T) {
	repo, home := pruneFixture(t)
	det := addToLane(t, repo, "det")
	repo.Git("-C", det, "checkout", "-q", "--detach")
	commitIn(t, repo, det, "d.txt", "d\n")
	tip := strings.TrimSpace(repo.Git("-C", det, "rev-parse", "HEAD"))

	if r := pruneRow(t, quietPrune(t, repo, worktree.PruneOptions{}), "det"); r.Outcome == worktree.OutcomeQuiet || r.Dossier != nil {
		t.Errorf("a detached worktree is offered as a quiet candidate: %+v", r)
	}
	got := quietPrune(t, repo, worktree.PruneOptions{Yes: []string{"det"}})
	if r := pruneRow(t, got, "det"); r.Outcome != "kept" || !strings.Contains(r.Reason, "no branch") {
		t.Errorf("a named detached worktree: %+v", r)
	}
	if !exists(det) || got.Reclaimed() != 0 || len(notesEntries(t, repo, home)) != 0 {
		t.Fatalf("the detached worktree stands, nothing archived: %+v", got.Rows)
	}
	repo.Git("-C", det, "cat-file", "-e", tip)
}

// TestPruneRefusesAYesNameNotInTheLane: every --yes name is checked before
// anything is judged for removal, so a name the lane does not hold refuses
// the whole run and nothing is archived or removed, the named candidate
// beside it included.
func TestPruneRefusesAYesNameNotInTheLane(t *testing.T) {
	repo, home := pruneFixture(t)
	idle := addToLane(t, repo, "idle")
	done := mergedWorktree(t, repo, "done")
	for _, yes := range [][]string{{"idle", "nope"}, {"../idle"}, {""}} {
		_, err := worktree.Prune(repo.Root(), worktree.PruneOptions{
			Held: noneHeld, Forge: unreachable, QuietDays: worktree.DefaultQuietDays, Now: fifteenDaysOn, Yes: yes,
		})
		if r := mustRefusal(t, err); !strings.Contains(r.Error(), "--yes") {
			t.Errorf("the refusal names the flag: %v", r)
		}
	}
	for _, p := range []string{idle, done} {
		if !exists(p) || !listed(t, repo, p) {
			t.Fatalf("%s stands", p)
		}
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing is archived: %v", n)
	}
}

// writeMachineConfig writes body as the machine layer's config.json under
// home, private to the caller as the resolver requires.
func writeMachineConfig(t *testing.T, home, body string) {
	t.Helper()
	p := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestQuietDaysReadsTheLayeredConfiguration: worktrees.quiet_days comes from
// the repository's file, then the machine's, then the bundled 14, through the
// layered resolver; a value below 1, a misspelt key in the claimed namespace
// and a value that is not a whole number are refused naming the file, never
// replaced by the default.
func TestQuietDaysReadsTheLayeredConfiguration(t *testing.T) {
	repoDir, home := t.TempDir(), t.TempDir()
	roots := layered.Roots{Repo: repoDir, Home: home}
	writeRepo := func(body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(repoDir, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := worktree.QuietDays(roots); err != nil || n != 14 || worktree.DefaultQuietDays != 14 {
		t.Fatalf("the bundled window: %d, %v", n, err)
	}
	writeMachineConfig(t, home, `{"worktrees":{"quiet_days":30}}`)
	if n, err := worktree.QuietDays(roots); err != nil || n != 30 {
		t.Fatalf("the machine's window: %d, %v", n, err)
	}
	writeRepo(`{"repo":{"name":"x"},"worktrees":{"quiet_days":20}}`)
	if n, err := worktree.QuietDays(roots); err != nil || n != 20 {
		t.Fatalf("the repository's window wins: %d, %v", n, err)
	}
	for _, bad := range []string{
		`{"worktrees":{"quiet_days":0}}`,
		`{"worktrees":{"quiet_days":-3}}`,
		`{"worktrees":{"quiet_days":1.5}}`,
		`{"worktrees":{"quiet_days":"14"}}`,
		`{"worktrees":{"quiet_day":3}}`,
	} {
		writeRepo(bad)
		_, err := worktree.QuietDays(roots)
		if err == nil || !strings.Contains(err.Error(), ".abcd/config.json") {
			t.Errorf("%s is refused naming its file: %v", bad, err)
		}
	}
	writeRepo(`{}`)
	writeMachineConfig(t, home, `{"worktrees":{"quiet_days":0}}`)
	if _, err := worktree.QuietDays(roots); err == nil || !strings.Contains(err.Error(), "config.json") {
		t.Errorf("a bad machine value is refused, never replaced by the default: %v", err)
	}
}
