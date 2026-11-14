package sweep

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// notesDir is the repository's lane in the notes archive under home.
func notesDir(t *testing.T, repo *gittest.Repo, home string) string {
	t.Helper()
	return abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root()))
}

// salvageEntries is the names of the entries in the repository's notes
// archive; none when the archive is not made.
func salvageEntries(t *testing.T, repo *gittest.Repo, home string) []string {
	t.Helper()
	es, err := os.ReadDir(notesDir(t, repo, home))
	if errors.Is(err, fs.ErrNotExist) {
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

// dirtyWip is a lane worktree "wip" with an unstaged change to a tracked
// file, a staged new file, an untracked file naming the home directory, and a
// file in its ignored local tier.
func dirtyWip(t *testing.T, repo *gittest.Repo, home string) string {
	t.Helper()
	p := laneAdd(t, repo, "wip")
	write(t, filepath.Join(p, "a.txt"), "a\nchanged\n")
	write(t, filepath.Join(p, "staged.txt"), "staged\n")
	repo.Git("-C", p, "add", "staged.txt")
	if err := os.MkdirAll(filepath.Join(p, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(p, "notes", "u.txt"), "see "+filepath.Join(home, "plan.md")+"\n")
	if err := os.MkdirAll(filepath.Join(p, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(p, ".abcd", ".work.local", "NEXT.md"), "ignored\n")
	return p
}

// readTar is every member of the tar data: its name to its content, or to
// "-> target" for a link.
func readTar(t *testing.T, data []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeSymlink {
			out[h.Name] = "-> " + h.Linkname
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[h.Name] = string(b)
	}
}

// TestApplySavesADirtyWorktreesChangesAndKeepsIt is criterion 4: a dirty
// worktree nothing holds is named save-patch by a dry run that writes
// nothing, and the apply saves its unstaged and staged changes as patches and
// its untracked, non-ignored files as a tar, redacted, beside a manifest
// naming the worktree, its branch, HEAD, its home-redacted path and the
// files; the worktree is kept, as dirty as it was, and the row names where
// the salvage went.
func TestApplySavesADirtyWorktreesChangesAndKeepsIt(t *testing.T) {
	repo, home := sweepRepo(t)
	p := dirtyWip(t, repo, home)
	status := repo.Git("-C", p, "status", "--porcelain")
	head := repo.Git("-C", p, "rev-parse", "HEAD")
	when := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	o := sweepOpts()
	o.Probe.Now = at(when)

	dry := dryRun(t, repo, o)
	if r := rowFor(t, dry, "wip"); r.Action != ActionSavePatch || r.Reason != dirtyToSave {
		t.Fatalf("the dry run's dirty row: %+v; want save-patch, saying the worktree is kept", r)
	}
	if dry.ExitCode() != ExitActed {
		t.Fatalf("a dry run with a salvage to make exits %d; want %d", dry.ExitCode(), ExitActed)
	}
	if got := salvageEntries(t, repo, home); got != nil {
		t.Fatalf("the dry run writes no salvage: %q", got)
	}

	res := apply(t, repo, o)
	entries := salvageEntries(t, repo, home)
	if !slices.Equal(entries, []string{"20300102T030405Z-wip-salvage"}) {
		t.Fatalf("one salvage entry, stamped with the run's UTC time: %q", entries)
	}
	entry := filepath.Join(notesDir(t, repo, home), entries[0])
	r := rowFor(t, res, "wip")
	if r.Action != ActionSavePatch || r.Outcome != OutcomeSaved || !strings.Contains(r.Reason, fsutil.RedactHome(entry)) ||
		!strings.Contains(r.Reason, "kept") || strings.Contains(r.Reason, home) {
		t.Fatalf("the applied dirty row: %+v; want it saved, naming the home-redacted entry and the worktree kept", r)
	}
	if res.ExitCode() != ExitActed {
		t.Fatalf("a run that saved a salvage exits %d; want %d", res.ExitCode(), ExitActed)
	}
	if got := repo.Git("-C", p, "status", "--porcelain"); got != status {
		t.Fatalf("the worktree is kept as it was: status %q, was %q", got, status)
	}

	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(entry, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	var m salvageManifestFile
	if err := json.Unmarshal(read(salvageManifest), &m); err != nil {
		t.Fatal(err)
	}
	if m.Kind != salvageKind || m.Worktree != "wip" || m.Branch != "wip" || m.Head != head ||
		m.Source != fsutil.RedactHome(p) || !strings.HasPrefix(m.Source, "~") ||
		m.RootSHA != gittest.RootCommit(t, repo.Root()) || m.SavedAt != "2030-01-02T03:04:05Z" ||
		!slices.Equal(m.Untracked, []string{"notes/u.txt"}) || m.SHA256 == "" {
		t.Fatalf("the manifest: %+v", m)
	}
	var names []string
	for _, f := range m.Files {
		names = append(names, f.Path)
		sum := sha256.Sum256(read(f.Path))
		if f.SHA256 != hex.EncodeToString(sum[:]) || f.Size != int64(len(read(f.Path))) {
			t.Errorf("%s: the manifest's size and sha256 are the file's: %+v", f.Path, f)
		}
	}
	if !slices.Equal(names, []string{salvageUnstaged, salvageStaged, salvageUntracked}) {
		t.Fatalf("the manifest lists the salvage's files: %q", names)
	}

	members := readTar(t, read(salvageUntracked))
	if len(members) != 1 || members["notes/u.txt"] != "see ~/plan.md\n" {
		t.Fatalf("the tar holds the untracked file alone, its home path redacted, and nothing the worktree ignores: %q", members)
	}
	// The patches are the changes: applied in order to a fresh checkout of
	// the same HEAD, they give back the dirty worktree's tracked files.
	fresh := filepath.Join(t.TempDir(), "fresh")
	repo.Git("worktree", "add", "-q", "--detach", fresh, head)
	repo.Git("-C", fresh, "apply", "--index", filepath.Join(entry, salvageStaged))
	repo.Git("-C", fresh, "apply", filepath.Join(entry, salvageUnstaged))
	for _, rel := range []string{"a.txt", "staged.txt"} {
		want, _ := os.ReadFile(filepath.Join(p, rel))
		got, err := os.ReadFile(filepath.Join(fresh, rel))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s from the patches = %q, %v; want %q", rel, got, err, want)
		}
	}
	if got := repo.Git("-C", fresh, "diff", "--cached", "--name-only"); got != "staged.txt" {
		t.Errorf("the staged patch holds the staged change alone: %q", got)
	}
}

// TestASecondSalvageOfTheSameChangesWritesNothing: a run that finds the
// changes an earlier salvage saved, unchanged, writes nothing new: its dry
// run keeps the worktree pointing at the earlier entry, and so does its
// apply, which acted on nothing. Changed again, the worktree is saved anew.
func TestASecondSalvageOfTheSameChangesWritesNothing(t *testing.T) {
	repo, home := sweepRepo(t)
	p := dirtyWip(t, repo, home)
	t0 := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	o := sweepOpts()
	o.Probe.Now = at(t0)
	apply(t, repo, o)
	first := salvageEntries(t, repo, home)
	if len(first) != 1 {
		t.Fatalf("the first run saves one salvage: %q", first)
	}
	earlier := fsutil.RedactHome(filepath.Join(notesDir(t, repo, home), first[0]))

	o.Probe.Now = at(t0.Add(time.Hour))
	dry := rowFor(t, dryRun(t, repo, o), "wip")
	if dry.Action != ActionKeep || !strings.Contains(dry.Reason, "already saved, unchanged, at "+earlier) {
		t.Fatalf("the second dry run: %+v; want it kept, pointing at %s", dry, earlier)
	}
	res := apply(t, repo, o)
	r := rowFor(t, res, "wip")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, earlier) {
		t.Fatalf("the second apply: %+v; want it kept, pointing at %s", r, earlier)
	}
	if got := salvageEntries(t, repo, home); !slices.Equal(got, first) {
		t.Fatalf("the second run writes nothing new: %q", got)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a run that saved nothing new exits %d; want %d", res.ExitCode(), ExitNothing)
	}

	write(t, filepath.Join(p, "notes", "u.txt"), "rewritten\n")
	o.Probe.Now = at(t0.Add(2 * time.Hour))
	if r := rowFor(t, apply(t, repo, o), "wip"); r.Outcome != OutcomeSaved {
		t.Fatalf("changed changes are saved anew: %+v", r)
	}
	if got := salvageEntries(t, repo, home); len(got) != 2 {
		t.Fatalf("a second entry for the changed worktree: %q", got)
	}
}

// TestAFailedSalvageKeepsTheWorktree: changes the salvage cannot vouch for
// keep the worktree, named, in the dry run and the apply alike, with nothing
// written; and a salvage that cannot be written keeps the worktree, unsaved
// and as it was, and is the run's fault.
func TestAFailedSalvageKeepsTheWorktree(t *testing.T) {
	t.Run("changes the redaction pass cannot vouch for", func(t *testing.T) {
		repo, home := sweepRepo(t)
		p := laneAdd(t, repo, "bin")
		if err := os.WriteFile(filepath.Join(p, "blob.bin"), []byte{0, 1, 2, 0xff}, 0o644); err != nil {
			t.Fatal(err)
		}
		dry := rowFor(t, dryRun(t, repo, sweepOpts()), "bin")
		want := "its changes cannot be saved: blob.bin: not text"
		if dry.Action != ActionKeep || !strings.Contains(dry.Reason, want) {
			t.Fatalf("the dry run: %+v; want it kept, naming %q", dry, want)
		}
		res := apply(t, repo, sweepOpts())
		if r := rowFor(t, res, "bin"); r.Outcome != OutcomeKept || r.Reason != dry.Reason {
			t.Fatalf("the apply: %+v; want it kept with the dry run's reason", r)
		}
		if !exists(filepath.Join(p, "blob.bin")) {
			t.Fatal("the worktree stands with its file")
		}
		if got := salvageEntries(t, repo, home); got != nil {
			t.Fatalf("nothing is written: %q", got)
		}
		if res.ExitCode() != ExitNothing {
			t.Fatalf("a run that saved nothing exits %d; want %d", res.ExitCode(), ExitNothing)
		}
	})

	t.Run("an archive that cannot be written", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root writes into a directory it cannot write")
		}
		repo, home := sweepRepo(t)
		p := dirtyWip(t, repo, home)
		status := repo.Git("-C", p, "status", "--porcelain")
		dir := notesDir(t, repo, home)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

		res, err := runApply(context.Background(), repo.Root(), sweepOpts())
		if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "the salvage of wip failed") ||
			strings.Contains(res.Refusal, home) {
			t.Fatalf("runApply over an unwritable archive = %+v, %v; want a home-redacted fault, exiting %d", res, err, ExitRefused)
		}
		r := rowFor(t, res, "wip")
		if r.Action != ActionSavePatch || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "kept with its changes unsaved") {
			t.Fatalf("the worktree whose salvage failed: %+v; want it kept, saying its changes are unsaved", r)
		}
		if got := repo.Git("-C", p, "status", "--porcelain"); got != status {
			t.Fatalf("the worktree is kept as it was: status %q, was %q", got, status)
		}
		if got := salvageEntries(t, repo, home); got != nil {
			t.Fatalf("nothing is written: %q", got)
		}
	})
}

// TestTheEmbargoStoreAndAnOutsideWorktreeAreNeverSalvaged: dirty as each is,
// the embargo store and a worktree outside the store are kept by the dry run
// before any salvage is considered, so nothing is copied out of either; and
// collectSalvage refuses either again, reading nothing.
func TestTheEmbargoStoreAndAnOutsideWorktreeAreNeverSalvaged(t *testing.T) {
	repo, home := sweepRepo(t)
	l, err := store.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	emb := l.Path(storepath.Name)
	repo.Git("worktree", "add", "-q", "--detach", emb)
	write(t, filepath.Join(emb, "advisory.md"), "private\n")
	outside := filepath.Join(t.TempDir(), "beside")
	repo.Git("worktree", "add", "-q", "-b", "beside", outside)
	write(t, filepath.Join(outside, "wip.txt"), "wip\n")

	res := apply(t, repo, sweepOpts())
	for _, name := range []string{storepath.Name, "beside"} {
		if r := rowFor(t, res, name); r.Action != ActionKeep || r.Outcome != OutcomeKept {
			t.Errorf("%s: %+v; want it kept, never salvaged", name, r)
		}
	}
	if got := salvageEntries(t, repo, home); got != nil {
		t.Fatalf("nothing is salvaged: %q", got)
	}
	for _, r := range []store.Row{
		{Name: storepath.Name, Abs: emb, Where: store.WhereStore},
		{Name: "beside", Branch: "beside", Abs: outside, Where: store.WhereOutside},
	} {
		if _, err := collectSalvage(repo.Root(), home, l, r); err == nil {
			t.Errorf("collectSalvage(%s) = nil; want a refusal", r.Name)
		}
	}
}

// TestASalvageEntryIsKnownByItsNameAndKind: an entry's name parses back to
// its worktree, and nothing else does; and a prune's notes entry for a
// worktree whose name ends in -salvage is never taken for a salvage, since
// its manifest is not one.
func TestASalvageEntryIsKnownByItsNameAndKind(t *testing.T) {
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.FixedZone("east", 3600))
	if got := salvageEntryName("wip", at); got != "20300102T020405Z-wip-salvage" {
		t.Fatalf("salvageEntryName = %q; want the UTC stamp", got)
	}
	for entry, want := range map[string]string{
		"20300102T020405Z-wip-salvage":   "wip",
		"20300102T020405Z-a-b-salvage":   "a-b",
		"20300102T020405Z-wip":           "",
		"20300102T020405Z--salvage":      "",
		"2030-01-02T02:04-wip-salvage":   "",
		"notatime00000000-wip-salvage":   "",
		"20300102T020405Zxwip-salvage":   "",
		"20300102T020405Z-wip-salvage/x": "",
	} {
		if got, ok := parseSalvageEntry(entry); got != want || ok != (want != "") {
			t.Errorf("parseSalvageEntry(%q) = %q, %v; want %q", entry, got, ok, want)
		}
	}

	repo, home := sweepRepo(t)
	p := dirtyWip(t, repo, home)
	l, err := store.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	src := store.Row{Name: "wip", Branch: "wip", Abs: p, Where: store.WhereStore}
	s, err := collectSalvage(repo.Root(), home, l, src)
	if err != nil {
		t.Fatal(err)
	}
	dir, _, err := l.NotesDir(true)
	if err != nil {
		t.Fatal(err)
	}
	notes := filepath.Join(dir, salvageEntryName("wip", at))
	if err := os.Mkdir(notes, 0o700); err != nil {
		t.Fatal(err)
	}
	m := s.manifest
	m.Kind = ""
	b, _ := json.Marshal(m)
	write(t, filepath.Join(notes, salvageManifest), string(b))
	if got, err := earlierSalvage(l, s); err != nil || got != "" {
		t.Fatalf("earlierSalvage over an entry that is not a salvage = %q, %v; want none", got, err)
	}
}
