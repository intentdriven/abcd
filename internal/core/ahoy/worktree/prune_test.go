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

func pruneOne(t *testing.T, repo *gittest.Repo, name string, opts worktree.PruneOptions) worktree.Pruned {
	t.Helper()
	if opts.Held == nil {
		opts.Held = noneHeld
	}
	if opts.Forge == nil {
		opts.Forge = unreachable
	}
	got, err := worktree.PruneOne(repo.Root(), name, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 1 || got.Rows[0].Name != name {
		t.Fatalf("one row, for %s: %+v", name, got.Rows)
	}
	return got
}

// TestPruneOneReclaimsTheNamedWorktreeAlone: the named merged worktree is
// reclaimed and archived, its merged neighbour stands, and the answer is the
// one row with Prune's count, so the door's exit code reads the same.
func TestPruneOneReclaimsTheNamedWorktreeAlone(t *testing.T) {
	repo, home := pruneFixture(t)
	done := mergedWorktree(t, repo, "done")
	if err := os.MkdirAll(filepath.Join(done, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(done, ".abcd", ".work.local", "NEXT.md"), []byte("next\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := mergedWorktree(t, repo, "other")
	work := addToLane(t, repo, "work")
	commitIn(t, repo, work, "w.txt", "w\n")

	got := pruneOne(t, repo, "done", worktree.PruneOptions{})
	if r := got.Rows[0]; r.Outcome != "reclaimed" || r.Notes != "1 file" || r.Archive == "" || got.Reclaimed() != 1 {
		t.Fatalf("the named worktree is reclaimed and archived: %+v", got)
	}
	if exists(done) || !exists(other) || !listed(t, repo, other) {
		t.Fatal("only the named worktree is removed")
	}
	if n := notesEntries(t, repo, home); len(n) != 1 || !strings.HasSuffix(n[0], "-done") {
		t.Fatalf("one archive entry, for done: %v", n)
	}
	if got := pruneOne(t, repo, "work", worktree.PruneOptions{}); got.Rows[0].Reason != "unmerged" || got.Reclaimed() != 0 {
		t.Fatalf("an unmerged worktree is kept: %+v", got)
	}
}

// TestPruneOneKeepsWhatPruneKeeps: every refusal Prune makes, PruneOne makes
// for the one worktree, with the same row, because it is the same path.
func TestPruneOneKeepsWhatPruneKeeps(t *testing.T) {
	repo, home := pruneFixture(t)
	locked := mergedWorktree(t, repo, "locked")
	repo.Git("worktree", "lock", "--reason", "on a removable disk", locked)
	here := mergedWorktree(t, repo, "here")
	mergedWorktree(t, repo, "run-0000000000000001-lane-1")
	dirty := mergedWorktree(t, repo, "dirty")
	if err := os.WriteFile(filepath.Join(dirty, "wip.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := addToLane(t, repo, "work")
	commitIn(t, repo, work, "w.txt", "w\n")
	binary := mergedWorktree(t, repo, "binary")
	if err := os.MkdirAll(filepath.Join(binary, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binary, ".abcd", ".work.local", "blob.bin"), []byte{0, 1, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	l, _ := worktree.LaneFor(repo.Root())
	repo.Git("worktree", "add", "-q", "--detach", l.Path(storepath.Name))
	if err := os.Mkdir(l.Path("stray"), 0o700); err != nil {
		t.Fatal(err)
	}
	held := func(string) (worktree.Held, error) {
		return worktree.Held{Names: map[string]string{"run-0000000000000001-lane-1": "run-0000000000000001"}}, nil
	}
	opts := worktree.PruneOptions{Held: held, Cwd: here}

	all := prune(t, repo, opts)
	for _, name := range []string{"locked", "here", "run-0000000000000001-lane-1", "dirty", "work", "binary", storepath.Name, "stray"} {
		want := pruneRow(t, all, name)
		if want.Outcome != "kept" {
			t.Fatalf("the fixture's %s is kept by Prune: %+v", name, want)
		}
		got := pruneOne(t, repo, name, opts)
		if got.Rows[0] != want || got.Reclaimed() != 0 {
			t.Errorf("%s: PruneOne %+v, Prune %+v", name, got.Rows[0], want)
		}
	}
	for _, p := range []string{locked, here, dirty, work, binary} {
		if !exists(p) || !listed(t, repo, p) {
			t.Fatalf("%s stands", p)
		}
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing kept is archived: %v", n)
	}
	if _, err := worktree.PruneOne(repo.Root(), "locked", worktree.PruneOptions{Forge: unreachable}); err == nil {
		t.Fatal("PruneOne with no reader of what the loop holds is an error, never a guess")
	}
	if err := os.Chmod(l.Dir, 0o770); err != nil {
		t.Fatal(err)
	}
	_, err := worktree.PruneOne(repo.Root(), "locked", worktree.PruneOptions{Held: noneHeld, Forge: unreachable})
	mustRefusal(t, err)
}

// TestPruneOneRefusesANameNotInTheLane: a name the lane does not hold, a path
// rather than a name, a worktree outside the store and a directory directly
// under the store that is not this lane are each refused, and nothing is
// archived or removed.
func TestPruneOneRefusesANameNotInTheLane(t *testing.T) {
	repo, home := pruneFixture(t)
	done := mergedWorktree(t, repo, "done")
	outside := filepath.Join(filepath.Dir(repo.Root()), "beside")
	repo.Git("worktree", "add", "-q", "-b", "beside", outside)
	repo.Git("-C", outside, "commit", "-q", "--allow-empty", "-m", "beside")
	repo.Git("merge", "-q", "--ff-only", "beside")
	l, _ := worktree.LaneFor(repo.Root())
	short := filepath.Join(filepath.Dir(l.Dir), l.RootSHA[:12])
	if err := os.Mkdir(short, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nope", "../" + filepath.Base(l.Dir) + "/done", "", "beside", l.RootSHA[:12], "DONE"} {
		_, err := worktree.PruneOne(repo.Root(), name, worktree.PruneOptions{Held: noneHeld, Forge: unreachable})
		mustRefusal(t, err)
	}
	for _, p := range []string{done, outside} {
		if !exists(p) || !listed(t, repo, p) {
			t.Fatalf("%s stands", p)
		}
	}
	if !exists(short) {
		t.Fatal("the abbreviated directory stands")
	}
	if n := notesEntries(t, repo, home); len(n) != 0 {
		t.Fatalf("nothing is archived: %v", n)
	}
}

// keepHook is a PreRemove that keeps every worktree with reason and records
// each path it is asked about.
func keepHook(reason string, asked *[]string) func(string) (bool, string) {
	return func(abs string) (bool, string) {
		*asked = append(*asked, abs)
		return true, reason
	}
}

// TestPreRemoveKeepsAWorktree: a caller's check that keeps a reclaimable
// worktree keeps it, through Prune and PruneOne alike, with its reason masked
// and home-redacted as a git reason is, and the archive entry written for it
// taken back, so a kept worktree leaves no orphan entry. A dry run asks it
// at the same point and reports the same row.
func TestPreRemoveKeepsAWorktree(t *testing.T) {
	for _, one := range []bool{false, true} {
		t.Run(map[bool]string{false: "Prune", true: "PruneOne"}[one], func(t *testing.T) {
			repo, home := pruneFixture(t)
			p := mergedWorktree(t, repo, "live")
			tier := filepath.Join(p, ".abcd", ".work.local")
			if err := os.MkdirAll(filepath.Join(tier, "logs"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tier, "logs", "run.log"), []byte("ran\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			var asked []string
			reason := "a session is live in " + home + "/x\x1b[31m\nforged"
			run := func(dry bool) worktree.PruneRow {
				opts := worktree.PruneOptions{DryRun: dry, PreRemove: keepHook(reason, &asked)}
				if one {
					return pruneOne(t, repo, "live", opts).Rows[0]
				}
				return pruneRow(t, prune(t, repo, opts), "live")
			}
			want := worktree.PruneRow{Name: "live", Branch: "live", Path: fsutil.RedactHome(p), Where: "store",
				Outcome: "kept", Reason: "a session is live in ~/x?[31m?forged"}
			for _, dry := range []bool{true, false} {
				if r := run(dry); r != want {
					t.Errorf("dry=%v: %+v, want %+v", dry, r, want)
				}
			}
			if len(asked) != 2 || fsutil.RealExistingPath(asked[0]) != fsutil.RealExistingPath(p) || !filepath.IsAbs(asked[0]) {
				t.Fatalf("the hook is asked about the worktree's absolute path, once a run: %v", asked)
			}
			if !exists(filepath.Join(tier, "logs", "run.log")) || !listed(t, repo, p) {
				t.Fatal("the kept worktree stands with its tier")
			}
			if n := notesEntries(t, repo, home); len(n) != 0 {
				t.Fatalf("no archive entry is left for a kept worktree: %v", n)
			}
		})
	}
}

// TestPreRemoveAllowsTheRemoval: a check that keeps nothing changes nothing.
func TestPreRemoveAllowsTheRemoval(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "done")
	if err := os.MkdirAll(filepath.Join(p, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, ".abcd", ".work.local", "NEXT.md"), []byte("next\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var asked []string
	got := prune(t, repo, worktree.PruneOptions{PreRemove: func(abs string) (bool, string) {
		asked = append(asked, abs)
		return false, "ignored when the check allows"
	}})
	r := pruneRow(t, got, "done")
	if r.Outcome != "reclaimed" || r.Reason != "" || r.Archive == "" || exists(p) || len(asked) != 1 {
		t.Fatalf("the check allowed the removal: %+v, asked %v", r, asked)
	}
	if n := notesEntries(t, repo, home); len(n) != 1 {
		t.Fatalf("the archive entry stays: %v", n)
	}
}

// TestPreRemoveIsNeverAskedAboutAWorktreeAlreadyKept: every earlier refusal,
// a failed archive included, keeps its worktree without asking the caller.
func TestPreRemoveIsNeverAskedAboutAWorktreeAlreadyKept(t *testing.T) {
	repo, _ := pruneFixture(t)
	locked := mergedWorktree(t, repo, "locked")
	repo.Git("worktree", "lock", locked)
	here := mergedWorktree(t, repo, "here")
	mergedWorktree(t, repo, "run-0000000000000001-lane-1")
	dirty := mergedWorktree(t, repo, "dirty")
	if err := os.WriteFile(filepath.Join(dirty, "wip.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := addToLane(t, repo, "work")
	commitIn(t, repo, work, "w.txt", "w\n")
	binary := mergedWorktree(t, repo, "binary")
	if err := os.MkdirAll(filepath.Join(binary, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binary, ".abcd", ".work.local", "blob.bin"), []byte{0, 1, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	l, _ := worktree.LaneFor(repo.Root())
	repo.Git("worktree", "add", "-q", "--detach", l.Path(storepath.Name))
	outside := filepath.Join(filepath.Dir(repo.Root()), "beside")
	repo.Git("worktree", "add", "-q", "--detach", outside)
	done := mergedWorktree(t, repo, "done")
	held := func(string) (worktree.Held, error) {
		return worktree.Held{Names: map[string]string{"run-0000000000000001-lane-1": "run-0000000000000001"}}, nil
	}
	var asked []string
	got := prune(t, repo, worktree.PruneOptions{Held: held, Cwd: here, PreRemove: keepHook("kept by the check", &asked)})
	if len(asked) != 1 || fsutil.RealExistingPath(asked[0]) != fsutil.RealExistingPath(done) {
		t.Fatalf("only the reclaimable worktree is asked about: %v", asked)
	}
	if r := pruneRow(t, got, "done"); r.Outcome != "kept" || r.Reason != "kept by the check" {
		t.Fatalf("the reclaimable one is kept by the check: %+v", r)
	}
	asked = nil
	for _, name := range []string{"locked", "here", "run-0000000000000001-lane-1", "dirty", "work", "binary", storepath.Name} {
		pruneOne(t, repo, name, worktree.PruneOptions{Held: held, Cwd: here, PreRemove: keepHook("kept by the check", &asked)})
	}
	if len(asked) != 0 {
		t.Fatalf("PruneOne asks about no worktree an earlier refusal kept: %v", asked)
	}
}

// TestPreRemoveIsAskedLastBeforeTheRemoval: the archive is written and
// flushed, then the caller is asked, and the removal follows with nothing
// between them.
func TestPreRemoveIsAskedLastBeforeTheRemoval(t *testing.T) {
	for _, one := range []bool{false, true} {
		t.Run(map[bool]string{false: "Prune", true: "PruneOne"}[one], func(t *testing.T) {
			repo, _ := pruneFixture(t)
			p := mergedWorktree(t, repo, "noted")
			if err := os.MkdirAll(filepath.Join(p, ".abcd", ".work.local"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p, ".abcd", ".work.local", "NEXT.md"), []byte("next\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			var events []string
			defer worktree.SwapFlushForTest(func(f *os.File) error {
				events = append(events, "flush "+filepath.Base(f.Name()))
				return nil
			})()
			defer worktree.SwapRemoveForTest(func(repoRoot, path string) error {
				events = append(events, "remove")
				repo.Git("-C", repoRoot, "worktree", "remove", "--", path)
				return nil
			})()
			opts := worktree.PruneOptions{PreRemove: func(abs string) (bool, string) {
				events = append(events, "pre-remove")
				return false, ""
			}}
			var r worktree.PruneRow
			if one {
				r = pruneOne(t, repo, "noted", opts).Rows[0]
			} else {
				r = pruneRow(t, prune(t, repo, opts), "noted")
			}
			if r.Outcome != "reclaimed" {
				t.Fatalf("the row: %+v", r)
			}
			at := slices.Index(events, "pre-remove")
			if at < 1 || len(events) != at+2 || events[at+1] != "remove" {
				t.Fatalf("the check comes after every flush and immediately before the removal: %v", events)
			}
			if !slices.Contains(events[:at], "flush NEXT.md") || !slices.Contains(events[:at], "flush manifest.json") {
				t.Fatalf("the archive is flushed before the check: %v", events)
			}
		})
	}
}
