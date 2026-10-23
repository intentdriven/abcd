package worktree_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// noneHeld is a build loop that holds nothing.
func noneHeld(string) (worktree.Held, error) { return worktree.Held{}, nil }

// pruneFixture is a repository whose local tier is ignored, as abcd's own
// checkouts ignore it, so the tier never makes a worktree dirty.
func pruneFixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo, home := repoWithCommit(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Commit("ignore the local tier")
	return repo, home
}

// mergedWorktree adds name to the lane with one commit of its own and
// fast-forwards main over it, so it is merged and clean.
func mergedWorktree(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	p := addToLane(t, repo, name)
	commitIn(t, repo, p, name+".txt", name+"\n")
	repo.Git("merge", "-q", "--ff-only", name)
	return p
}

func prune(t *testing.T, repo *gittest.Repo, opts worktree.PruneOptions) worktree.Pruned {
	t.Helper()
	if opts.Held == nil {
		opts.Held = noneHeld
	}
	if opts.Forge == nil {
		opts.Forge = unreachable
	}
	got, err := worktree.Prune(repo.Root(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func pruneRow(t *testing.T, got worktree.Pruned, name string) worktree.PruneRow {
	t.Helper()
	for _, r := range got.Rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no row named %q in %+v", name, got.Rows)
	return worktree.PruneRow{}
}

func listed(t *testing.T, repo *gittest.Repo, p string) bool {
	t.Helper()
	return strings.Contains(repo.Git("worktree", "list", "--porcelain"), "worktree "+fsutil.RealExistingPath(p)) ||
		strings.Contains(repo.Git("worktree", "list", "--porcelain"), "worktree "+p)
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

// notesEntries lists the notes archive's entries for the repository.
func notesEntries(t *testing.T, repo *gittest.Repo, home string) []string {
	t.Helper()
	dir := abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root()))
	es, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// TestPruneReclaimsAMergedCleanWorktree is criterion 8: the directory is gone,
// git no longer lists it, its branch stands, and the row names it; a fresh
// worktree beside it is kept as unmerged.
func TestPruneReclaimsAMergedCleanWorktree(t *testing.T) {
	repo, _ := pruneFixture(t)
	done := mergedWorktree(t, repo, "done")
	fresh := addToLane(t, repo, "fresh")

	got := prune(t, repo, worktree.PruneOptions{})
	if r := pruneRow(t, got, "done"); r.Outcome != "reclaimed" || r.Notes != "none" {
		t.Fatalf("the merged worktree: %+v", r)
	}
	if exists(done) || listed(t, repo, done) {
		t.Fatal("the directory is gone and git no longer lists it")
	}
	repo.Git("rev-parse", "--verify", "refs/heads/done")
	if r := pruneRow(t, got, "fresh"); r.Outcome != "kept" || r.Reason != "unmerged" || !exists(fresh) {
		t.Fatalf("the fresh worktree: %+v", r)
	}
	if got.Reclaimed() != 1 {
		t.Fatalf("one reclaimed: %d", got.Reclaimed())
	}
}

// TestPruneReclaimsASquashMergedWorktree is criterion 9: no ancestry, every
// change on the default branch as one commit, and no forge reachable.
func TestPruneReclaimsASquashMergedWorktree(t *testing.T) {
	repo, _ := pruneFixture(t)
	p := addToLane(t, repo, "sq")
	commitIn(t, repo, p, "s1.txt", "one\n")
	commitIn(t, repo, p, "s2.txt", "two\n")
	repo.Write("other.txt", "other\n")
	repo.Commit("other")
	repo.Git("merge", "-q", "--squash", "sq")
	repo.Git("commit", "-q", "-m", "squashed")

	got := prune(t, repo, worktree.PruneOptions{Forge: unreachable})
	if r := pruneRow(t, got, "sq"); r.Outcome != "reclaimed" || exists(p) {
		t.Fatalf("the squash-merged worktree: %+v", r)
	}
	if !strings.Contains(got.ForgeNote, "no forge answered") {
		t.Fatalf("the run says the forge did not answer: %q", got.ForgeNote)
	}
}

// TestPruneArchivesTheTierRedacted is criterion 10: the handover note and the
// log land redacted under ~/.abcd.noindex/notes/<root-sha>/<timestamp>-<name>/ beside a
// manifest naming the worktree, branch, time and files, and the row names
// the entry.
func TestPruneArchivesTheTierRedacted(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "noted")
	pat := "ghp_" + strings.Repeat("a", 40)
	tier := filepath.Join(p, ".abcd", ".work.local")
	if err := os.MkdirAll(filepath.Join(tier, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "NEXT.md"), []byte("next: rotate "+pat+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "logs", "run.log"), []byte("ran in "+home+"/project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "elsewhere"), filepath.Join(tier, "link")); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 23, 9, 8, 7, 0, time.UTC)

	got := prune(t, repo, worktree.PruneOptions{Now: func() time.Time { return at }})
	r := pruneRow(t, got, "noted")
	entry := abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root()), "20261023T090807Z-noted")
	if r.Outcome != "reclaimed" || r.Archive != fsutil.RedactHome(entry) {
		t.Fatalf("the row names the archive entry: %+v (want %s)", r, fsutil.RedactHome(entry))
	}
	for rel, banned := range map[string]string{"NEXT.md": pat, "logs/run.log": home} {
		b, err := os.ReadFile(filepath.Join(entry, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s is archived: %v", rel, err)
		}
		if strings.Contains(string(b), banned) {
			t.Fatalf("%s is archived redacted: %q", rel, b)
		}
		if fi, _ := os.Stat(filepath.Join(entry, filepath.FromSlash(rel))); fi.Mode().Perm() != 0o600 {
			t.Fatalf("%s is 0600: %v", rel, fi.Mode())
		}
	}
	if exists(filepath.Join(entry, "link")) {
		t.Fatal("a symlink is recorded, never followed or copied")
	}
	var m struct {
		SchemaVersion int    `json:"schema_version"`
		Worktree      string `json:"worktree"`
		Branch        string `json:"branch"`
		Source        string `json:"source"`
		ArchivedAt    string `json:"archived_at"`
		RootSHA       string `json:"root_sha"`
		Files         []struct {
			Path   string `json:"path"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
			Link   string `json:"link"`
		} `json:"files"`
	}
	b, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m.SchemaVersion != 1 || m.Worktree != "noted" || m.Branch != "noted" || m.Source != fsutil.RedactHome(p) ||
		m.ArchivedAt != "2026-10-23T09:08:07Z" || m.RootSHA != gittest.RootCommit(t, repo.Root()) {
		t.Fatalf("the manifest: %+v", m)
	}
	var paths []string
	for _, f := range m.Files {
		paths = append(paths, f.Path)
		if f.Path == "link" && f.Link != "~/elsewhere" {
			t.Fatalf("the link's target is recorded home-redacted: %+v", f)
		}
		if f.Path != "link" && (f.Size == 0 || len(f.SHA256) != 64) {
			t.Fatalf("a file's size and hash: %+v", f)
		}
	}
	if !slices.Equal(paths, []string{"NEXT.md", "link", "logs/run.log"}) {
		t.Fatalf("the manifest's files: %v", paths)
	}
	if strings.Contains(string(b), home) || strings.Contains(string(b), pat) {
		t.Fatalf("the manifest is redacted: %s", b)
	}
}

// TestPruneKeepsAWorktreeWhoseNotesCannotBeArchived: a file the redaction
// pass cannot vouch for fails the archive, the worktree is kept and named
// with the reason and the file, and nothing of the tier is lost or written.
func TestPruneKeepsAWorktreeWhoseNotesCannotBeArchived(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "binary")
	blob := filepath.Join(p, ".abcd", ".work.local", "blob.bin")
	if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blob, []byte{0, 1, 2, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, "binary")
	if r.Outcome != "kept" || !strings.Contains(r.Reason, "notes could not be archived") || !strings.Contains(r.Reason, "blob.bin") {
		t.Fatalf("a failed archive keeps the worktree: %+v", r)
	}
	if !exists(blob) || !listed(t, repo, p) {
		t.Fatal("the worktree and its tier stand")
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("a failed archive writes no entry: %v", n)
	}
	if got.Reclaimed() != 0 {
		t.Fatalf("nothing reclaimed: %d", got.Reclaimed())
	}
}

// TestPruneRefusesAStoreItCannotTrust is criterion 12: a store level another
// account could write refuses the run before anything is judged, and nothing
// is removed.
func TestPruneRefusesAStoreItCannotTrust(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func(t *testing.T, lane string)
	}{
		{"group-writable lane", func(t *testing.T, lane string) {
			if err := os.Chmod(lane, 0o770); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlinked store", func(t *testing.T, lane string) {
			store := filepath.Dir(lane)
			moved := filepath.Join(t.TempDir(), "worktrees")
			if err := os.Rename(store, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(moved, store); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := pruneFixture(t)
			p := mergedWorktree(t, repo, "done")
			l, _ := worktree.LaneFor(repo.Root())
			tc.spoil(t, l.Dir)
			_, err := worktree.Prune(repo.Root(), worktree.PruneOptions{Held: noneHeld, Forge: unreachable})
			mustRefusal(t, err)
			if !exists(filepath.Join(p, "done.txt")) || !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "/done\n") {
				t.Fatal("nothing is removed")
			}
		})
	}
}

// TestPruneKeepsWhatItCannotReclaim is criterion 13: dirty, unmerged and
// outside each stay, named with the reason, and nothing is reclaimed; a
// merged clean worktree beside them is.
func TestPruneKeepsWhatItCannotReclaim(t *testing.T) {
	repo, _ := pruneFixture(t)
	dirty := mergedWorktree(t, repo, "dirty")
	if err := os.WriteFile(filepath.Join(dirty, "wip.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := addToLane(t, repo, "work")
	commitIn(t, repo, work, "w.txt", "w\n")
	outside := filepath.Join(filepath.Dir(repo.Root()), "beside")
	repo.Git("worktree", "add", "-q", "--detach", outside)

	got := prune(t, repo, worktree.PruneOptions{})
	for name, want := range map[string]string{"dirty": "dirty: uncommitted changes", "work": "unmerged", "beside": "outside the store; never touched"} {
		if r := pruneRow(t, got, name); r.Outcome != "kept" || r.Reason != want {
			t.Errorf("%s: %+v, want kept, %q", name, r, want)
		}
	}
	for _, p := range []string{dirty, work, outside} {
		if !exists(p) || !listed(t, repo, p) {
			t.Fatalf("%s stands", p)
		}
	}
	if got.Reclaimed() != 0 {
		t.Fatalf("nothing reclaimed: %d", got.Reclaimed())
	}
	mergedWorktree(t, repo, "done")
	if got := prune(t, repo, worktree.PruneOptions{}); got.Reclaimed() != 1 || pruneRow(t, got, "done").Outcome != "reclaimed" {
		t.Fatalf("the merged clean one is reclaimed: %+v", got.Rows)
	}
}

// TestPruneLeavesWhatFailsTheProof is criterion 14: a directory in the lane
// git does not list, and one git lists whose common directory is another
// checkout's, are reported and untouched.
func TestPruneLeavesWhatFailsTheProof(t *testing.T) {
	repo, _ := pruneFixture(t)
	other := newRepo(t, "other")
	l, _ := worktree.LaneFor(repo.Root())
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	// Another repository's worktree in this lane: git here does not list it.
	foreign := l.Path("foreign")
	other.Git("worktree", "add", "-q", "--detach", foreign)
	// A worktree git here lists whose directory now holds another
	// checkout's worktree.
	swapped := mergedWorktree(t, repo, "swapped")
	if err := os.Rename(swapped, filepath.Join(t.TempDir(), "aside")); err != nil {
		t.Fatal(err)
	}
	other.Git("worktree", "add", "-q", "--detach", swapped)

	got := prune(t, repo, worktree.PruneOptions{})
	if r := pruneRow(t, got, "foreign"); r.Outcome != "kept" || r.Reason != "not a worktree of this repository" {
		t.Errorf("foreign: %+v", r)
	}
	if r := pruneRow(t, got, "swapped"); r.Outcome != "kept" || !strings.Contains(r.Reason, "another checkout's worktree") {
		t.Errorf("swapped: %+v", r)
	}
	for _, p := range []string{foreign, swapped} {
		if !exists(filepath.Join(p, "other.txt")) || !listed(t, other, p) {
			t.Fatalf("%s is untouched", p)
		}
	}
}

// TestPruneKeepsLockedOwnCwdAndHeld is criterion 15: a merged clean worktree
// git holds locked, the one prune runs from, and one the build loop holds
// each stay, named with the reason.
func TestPruneKeepsLockedOwnCwdAndHeld(t *testing.T) {
	repo, _ := pruneFixture(t)
	locked := mergedWorktree(t, repo, "locked")
	repo.Git("worktree", "lock", "--reason", "on a removable disk", locked)
	here := mergedWorktree(t, repo, "here")
	busy := mergedWorktree(t, repo, "run-0000000000000001-lane-1")
	unread := mergedWorktree(t, repo, "run-0000000000000002-lane-3")
	held := func(string) (worktree.Held, error) {
		return worktree.Held{Names: map[string]string{"run-0000000000000001-lane-1": "run-0000000000000001"},
			Runs: []string{"run-0000000000000002"}}, nil
	}

	got := prune(t, repo, worktree.PruneOptions{Held: held, Cwd: filepath.Join(here, "sub")})
	for name, want := range map[string]string{
		"locked":                      "locked by git: on a removable disk",
		"here":                        "prune's own working directory",
		"run-0000000000000001-lane-1": "held by the build loop's run run-0000000000000001",
		"run-0000000000000002-lane-3": "held by the build loop's run run-0000000000000002",
	} {
		if r := pruneRow(t, got, name); r.Outcome != "kept" || r.Reason != want {
			t.Errorf("%s: %+v, want %q", name, r, want)
		}
	}
	for _, p := range []string{locked, here, busy, unread} {
		if !exists(p) {
			t.Fatalf("%s stands", p)
		}
	}
	if _, err := worktree.Prune(repo.Root(), worktree.PruneOptions{Forge: unreachable}); err == nil {
		t.Fatal("prune with no reader of what the loop holds is an error, never a guess")
	}
}

// TestPruneNeverTouchesTheEmbargoStore (iss-2610230014564359): the embargo
// store's worktree in the lane is never judged, archived or removed, however
// reclaimable it looks.
func TestPruneNeverTouchesTheEmbargoStore(t *testing.T) {
	repo, home := pruneFixture(t)
	l, _ := worktree.LaneFor(repo.Root())
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	store := l.Path(storepath.Name)
	repo.Git("worktree", "add", "-q", "--detach", store)
	note := filepath.Join(store, ".abcd", ".work.local", "advisory.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("private\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, storepath.Name)
	if r.Outcome != "kept" || r.Where != "sibling" || !strings.Contains(r.Reason, "embargo store") || r.Archive != "" {
		t.Fatalf("the embargo store: %+v", r)
	}
	if !exists(note) || !listed(t, repo, store) {
		t.Fatal("the embargo store stands with its records")
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing of the embargo store is archived: %v", n)
	}
}

// TestPruneDryRunReportsTheRowsARealRunWould is criterion 16: a dry run
// removes and archives nothing and reports the rows a real run then reports,
// with "would reclaim" in place of "reclaimed".
func TestPruneDryRunReportsTheRowsARealRunWould(t *testing.T) {
	repo, home := pruneFixture(t)
	done := mergedWorktree(t, repo, "done")
	if err := os.MkdirAll(filepath.Join(done, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(done, ".abcd", ".work.local", "NEXT.md"), []byte("next\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	addToLane(t, repo, "fresh")

	dry := prune(t, repo, worktree.PruneOptions{DryRun: true})
	if !exists(done) || len(notesEntries(t, repo, home)) != 0 {
		t.Fatal("a dry run removes and archives nothing")
	}
	real := prune(t, repo, worktree.PruneOptions{})
	if dry.Reclaimed() != real.Reclaimed() || len(dry.Rows) != len(real.Rows) || !dry.DryRun || real.DryRun {
		t.Fatalf("dry %+v\nreal %+v", dry, real)
	}
	for i, d := range dry.Rows {
		r := real.Rows[i]
		if d.Outcome == "would reclaim" {
			d.Outcome = "reclaimed"
		}
		r.Archive = ""
		if d != r {
			t.Errorf("row %d: dry %+v, real %+v", i, d, r)
		}
	}
	if pruneRow(t, dry, "done").Outcome != "would reclaim" {
		t.Fatalf("the dry run's row: %+v", dry.Rows)
	}
}

// TestPruneFlushesTheArchiveBeforeTheRemoval: every archived file, the
// manifest, the directories they sit in and the entry itself are flushed to
// disk before git is asked to remove the worktree, so the only copy of the
// notes is durable when the tier goes. The flush itself is the filesystem's
// and is not observed; that it is asked for, and when, is.
func TestPruneFlushesTheArchiveBeforeTheRemoval(t *testing.T) {
	repo, _ := pruneFixture(t)
	p := mergedWorktree(t, repo, "noted")
	tier := filepath.Join(p, ".abcd", ".work.local")
	if err := os.MkdirAll(filepath.Join(tier, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, text := range map[string]string{"NEXT.md": "next\n", "logs/run.log": "ran\n"} {
		if err := os.WriteFile(filepath.Join(tier, filepath.FromSlash(rel)), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var events []string
	defer worktree.SwapFlushForTest(func(f *os.File) error {
		fi, err := f.Stat()
		if err != nil {
			t.Fatalf("flushing a closed file: %v", err)
		}
		kind := "file "
		if fi.IsDir() {
			kind = "dir "
		}
		events = append(events, kind+filepath.ToSlash(f.Name()))
		return nil
	})()
	defer worktree.SwapRemoveForTest(func(repoRoot, path string) error {
		events = append(events, "remove")
		repo.Git("-C", repoRoot, "worktree", "remove", "--", path)
		return nil
	})()

	got := prune(t, repo, worktree.PruneOptions{})
	if r := pruneRow(t, got, "noted"); r.Outcome != "reclaimed" {
		t.Fatalf("the row: %+v", r)
	}
	removeAt := slices.Index(events, "remove")
	if removeAt < 0 {
		t.Fatalf("the removal was never asked for: %v", events)
	}
	before := events[:removeAt]
	for _, want := range []string{"NEXT.md", "logs/run.log", "manifest.json"} {
		if !slices.ContainsFunc(before, func(e string) bool { return strings.HasPrefix(e, "file ") && strings.HasSuffix(e, want) }) {
			t.Errorf("%s is flushed before the removal: %v", want, before)
		}
	}
	dirs := 0
	for _, e := range before {
		if strings.HasPrefix(e, "dir ") {
			dirs++
		}
	}
	// logs/, the entry, and the lane's notes directory the entry was made in.
	if dirs < 3 {
		t.Errorf("the archive's directories are flushed before the removal: %v", before)
	}
	if len(events) != removeAt+1 {
		t.Errorf("nothing is flushed after the removal: %v", events)
	}
}
