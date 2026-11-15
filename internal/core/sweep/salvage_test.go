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
	"os/exec"
	"path/filepath"
	"runtime"
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
// the salvage's collect refuses either again, reading nothing.
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
		sv := &salvager{ctx: context.Background(), repoRoot: repo.Root(), home: home, lane: l}
		if _, err := sv.collect(r); err == nil {
			t.Errorf("collect(%s) = nil; want a refusal", r.Name)
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
	sv := &salvager{ctx: context.Background(), repoRoot: repo.Root(), home: home, lane: l}
	s, err := sv.collect(src)
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

// keptUnsaved asserts that the dry run and the apply over repo both keep the
// worktree name with a reason holding want, and that nothing is written and
// the apply exits as one that acted on nothing.
func keptUnsaved(t *testing.T, repo *gittest.Repo, home, name, want string) {
	t.Helper()
	dry := rowFor(t, dryRun(t, repo, sweepOpts()), name)
	if dry.Action != ActionKeep || !strings.Contains(dry.Reason, dirtyUnsaved+want) {
		t.Fatalf("the dry run: %+v; want it kept, its reason holding %q", dry, dirtyUnsaved+want)
	}
	res := apply(t, repo, sweepOpts())
	if r := rowFor(t, res, name); r.Outcome != OutcomeKept || r.Reason != dry.Reason {
		t.Fatalf("the apply: %+v; want it kept with the dry run's reason", r)
	}
	if got := salvageEntries(t, repo, home); got != nil {
		t.Fatalf("nothing is written: %q", got)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a run that saved nothing exits %d; want %d", res.ExitCode(), ExitNothing)
	}
}

// TestAChangeGitWritesAsABinaryPatchIsRefused: git writes a change to a path
// it treats as binary as a binary patch, whose encoded lines the redaction
// pass cannot read, so the worktree is kept with the reason an untracked
// binary file gets, naming the path, and nothing is saved: a tracked file
// changed to hold a NUL byte, and a staged text file its attributes mark
// -diff.
func TestAChangeGitWritesAsABinaryPatchIsRefused(t *testing.T) {
	t.Run("a tracked file holding a NUL byte", func(t *testing.T) {
		repo, home := sweepRepo(t)
		p := laneAdd(t, repo, "nul")
		if err := os.WriteFile(filepath.Join(p, "a.txt"), []byte("a\x00b\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		keptUnsaved(t, repo, home, "nul", "a.txt: "+notText)
	})

	t.Run("a text file marked -diff", func(t *testing.T) {
		repo, home := sweepRepo(t)
		repo.Write(".gitattributes", "doc.txt -diff\n")
		repo.Write("doc.txt", "one\n")
		repo.Commit("doc")
		p := laneAdd(t, repo, "nodiff")
		write(t, filepath.Join(p, "doc.txt"), "two\n")
		repo.Git("-C", p, "add", "doc.txt")
		keptUnsaved(t, repo, home, "nodiff", "doc.txt: "+notText)
	})
}

// TestBinaryChangeFindsTheOpeningLineAndNamesAPlainPath: the binary patch is
// known by a whole line, never by text inside a hunk, and its path is named
// only from a plain header naming one path on both sides.
func TestBinaryChangeFindsTheOpeningLineAndNamesAPlainPath(t *testing.T) {
	for _, c := range []struct {
		diff, path string
		binary     bool
	}{
		{"diff --git a/x.bin b/x.bin\nindex 1..2 100644\nGIT binary patch\nliteral 3\nKc${\n", "x.bin", true},
		{"diff --git a/a b/a\n+GIT binary patch\n GIT binary patch\n", "", false},
		{"diff --git a/old b/new\nGIT binary patch\n", "", true},
		{"diff --git \"a/q\\tx\" \"b/q\\tx\"\nGIT binary patch\n", "", true},
		{"diff --git a/a b b/a b\nGIT binary patch\n", "a b", true},
		{"GIT binary patch\n", "", true},
	} {
		p, ok := binaryChange(c.diff)
		if p != c.path || ok != c.binary {
			t.Errorf("binaryChange(%q) = %q, %v; want %q, %v", c.diff, p, ok, c.path, c.binary)
		}
	}
}

// markScript is a script under the test's temp dir that records each run in
// a file there and passes its input through: a content filter whose running
// the test can see.
func markScript(t *testing.T) (script, mark string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	dir := t.TempDir()
	mark = filepath.Join(dir, "filter-ran")
	script = filepath.Join(dir, "filter.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return script, mark
}

// staleAt moves the file's times back by age, so its stat no longer matches
// the index and git compares its content.
func staleAt(t *testing.T, p string, age time.Duration) {
	t.Helper()
	old := time.Now().Add(-age)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
}

// plainGit runs git in dir under the fixture's environment, outside every
// abcd runner, for a test proving its fixture is live.
func plainGit(t *testing.T, repo *gittest.Repo, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = repo.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestTheSalvageReadsChangesTheWayTheStatusReadDoes: the salvage reads a
// worktree's changes as gitutil.Status reads its status, so over a file whose
// stat no longer matches, neither a clean filter the repository names nor one
// a checked-out nested repository's own config names is started, by the dry
// run or the apply; and the changes are still saved.
func TestTheSalvageReadsChangesTheWayTheStatusReadDoes(t *testing.T) {
	check := func(t *testing.T, repo *gittest.Repo, p, stale, mark string) {
		t.Helper()
		// The fixture is live: a plain diff in the worktree starts the filter.
		staleAt(t, stale, 48*time.Hour)
		plainGit(t, repo, p, "diff")
		if !exists(mark) {
			t.Fatal("fixture: a plain git diff did not start the filter, so this test proves nothing")
		}
		if err := os.Remove(mark); err != nil {
			t.Fatal(err)
		}
		staleAt(t, stale, 24*time.Hour)

		if r := rowFor(t, dryRun(t, repo, sweepOpts()), "wip"); r.Action != ActionSavePatch {
			t.Fatalf("the dry run: %+v; want save-patch", r)
		}
		if exists(mark) {
			t.Fatal("the dry run started the filter")
		}
		if r := rowFor(t, apply(t, repo, sweepOpts()), "wip"); r.Outcome != OutcomeSaved {
			t.Fatalf("the apply: %+v; want the changes saved", r)
		}
		if exists(mark) {
			t.Fatal("the apply started the filter")
		}
	}

	t.Run("the repository's clean filter", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		p := laneAdd(t, repo, "wip")
		write(t, filepath.Join(p, "new.txt"), "new\n")
		script, mark := markScript(t)
		repo.Git("config", "filter.mark.clean", script)
		info := filepath.Join(repo.Root(), ".git", "info")
		if err := os.MkdirAll(info, 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(info, "attributes"), "a.txt filter=mark\n")
		check(t, repo, p, filepath.Join(p, "a.txt"), mark)
	})

	t.Run("a nested repository's own clean filter", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		p := laneAdd(t, repo, "wip")
		nested := filepath.Join(p, "nested")
		if err := os.Mkdir(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", nested, "init", "-q")
		write(t, filepath.Join(nested, "s.txt"), "s\n")
		repo.Git("-C", nested, "add", "s.txt")
		repo.Git("-C", nested, "commit", "-q", "-m", "nested seed")
		repo.Git("-C", p, "-c", "advice.addEmbeddedRepo=false", "add", "nested")
		script, mark := markScript(t)
		repo.Git("-C", nested, "config", "filter.subevil.clean", script)
		if err := os.MkdirAll(filepath.Join(nested, ".git", "info"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(nested, ".git", "info", "attributes"), "* filter=subevil\n")
		check(t, repo, p, filepath.Join(nested, "s.txt"), mark)
	})
}

// TestADryRunLeavesTheWorktreesIndexAsItWas: a tracked file touched without
// its bytes changing is compared without the index being refreshed, so a dry
// run leaves the index's bytes and time as they were, and the apply judges
// the worktree as the dry run did and saves its changes.
func TestADryRunLeavesTheWorktreesIndexAsItWas(t *testing.T) {
	repo, _ := sweepRepo(t)
	p := laneAdd(t, repo, "wip")
	write(t, filepath.Join(p, "new.txt"), "new\n")
	staleAt(t, filepath.Join(p, "a.txt"), 48*time.Hour)
	index := repo.Git("-C", p, "rev-parse", "--path-format=absolute", "--git-path", "index")
	read := func() ([]byte, time.Time) {
		b, err := os.ReadFile(index)
		if err != nil {
			t.Fatal(err)
		}
		fi, err := os.Stat(index)
		if err != nil {
			t.Fatal(err)
		}
		return b, fi.ModTime()
	}
	before, beforeAt := read()

	dry := rowFor(t, dryRun(t, repo, sweepOpts()), "wip")
	after, afterAt := read()
	if !bytes.Equal(after, before) || !afterAt.Equal(beforeAt) {
		t.Fatalf("the dry run rewrote the worktree's index: %v -> %v", beforeAt, afterAt)
	}
	if dry.Action != ActionSavePatch {
		t.Fatalf("the dry run: %+v; want save-patch", dry)
	}
	if r := rowFor(t, apply(t, repo, sweepOpts()), "wip"); r.Action != dry.Action || r.Outcome != OutcomeSaved {
		t.Fatalf("the apply: %+v; want it judged as the dry run judged it, and saved", r)
	}
}

// TestAWorktreeMidMergeIsKeptUnsaved: a worktree with unmerged paths holds
// combined hunks `git apply` cannot apply, so it is kept, named, and nothing
// is saved.
func TestAWorktreeMidMergeIsKeptUnsaved(t *testing.T) {
	repo, home := sweepRepo(t)
	p := laneAdd(t, repo, "mid")
	write(t, filepath.Join(p, "a.txt"), "wip side\n")
	repo.Git("-C", p, "commit", "-q", "-am", "wip side")
	repo.Write("a.txt", "main side\n")
	repo.Commit("main side")
	cmd := exec.Command("git", "-C", p, "-c", "user.name=F", "-c", "user.email=f@example.invalid",
		"-c", "commit.gpgsign=false", "merge", "-q", "main")
	cmd.Env = repo.Env()
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("fixture: the merge did not conflict:\n%s", out)
	}
	keptUnsaved(t, repo, home, "mid", mergeInProgress)
}
