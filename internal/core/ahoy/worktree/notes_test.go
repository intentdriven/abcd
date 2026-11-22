package worktree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes: the binary scripts/abcd.sh
// builds into the tier's bin/ is a build product, past the per-file cap and
// not text, so the archive leaves it out instead of keeping the worktree
// (iss-2611022247131866). The tier's notes beside it are still archived, and a
// bin/ deeper in the tier is still judged as notes.
func TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "built")
	tier := filepath.Join(p, ".abcd", ".work.local")
	if err := os.MkdirAll(filepath.Join(tier, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 5<<20) // over the 4 MiB per-file cap, and NUL bytes: not text
	if err := os.WriteFile(filepath.Join(tier, "bin", "abcd"), big, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "NEXT.md"), []byte("next: land it\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, "built")
	if r.Outcome != "reclaimed" {
		t.Fatalf("a merged, clean worktree whose tier holds the built binary is reclaimed: %+v", r)
	}
	if exists(p) || listed(t, repo, p) {
		t.Fatal("the reclaimed worktree is gone")
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	if _, err := os.Stat(filepath.Join(entry, "NEXT.md")); err != nil {
		t.Fatalf("the notes beside the binary are archived: %v", err)
	}
	if exists(filepath.Join(entry, "bin")) {
		t.Fatal("the built binary is left out of the archive")
	}
	b, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "bin/abcd") {
		t.Fatalf("the manifest does not list the binary: %s", b)
	}
}

// TestPruneArchivesTextAtTheBuildsNames: the build is never text, so text at the
// binary's path or at a name of its abcd.<pid> shape is what a person wrote, and
// removing the worktree would delete its only copy (iss-2611041759314115).
func TestPruneArchivesTextAtTheBuildsNames(t *testing.T) {
	for _, name := range []string{"abcd", "abcd.20261107"} {
		t.Run(name, func(t *testing.T) {
			repo, home := pruneFixture(t)
			p := mergedWorktree(t, repo, "text-at-name")
			bin := filepath.Join(p, ".abcd", ".work.local", "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, name), []byte("kept by hand\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "text-at-name")
			if r.Outcome != "reclaimed" {
				t.Fatalf("a merged, clean worktree whose bin/ holds only text is reclaimed: %+v", r)
			}
			entries := notesEntries(t, repo, home)
			if len(entries) != 1 {
				t.Fatalf("one archive entry: %v", entries)
			}
			entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
			b, err := os.ReadFile(filepath.Join(entry, "bin", name))
			if err != nil {
				t.Fatalf("the text at bin/%s is archived: %v", name, err)
			}
			if !strings.Contains(string(b), "kept by hand") {
				t.Fatalf("the archived file holds what the person wrote: %q", b)
			}
		})
	}
}

// TestPruneLeavesAShortBuildLeftoverOut: a killed build can leave a file under
// the cap, and a file that is not text at the build's names is still the build.
func TestPruneLeavesAShortBuildLeftoverOut(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "short-leftover")
	tier := filepath.Join(p, ".abcd", ".work.local")
	if err := os.MkdirAll(filepath.Join(tier, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "bin", "abcd.12345"), []byte{0xcf, 0xfa, 0xed, 0xfe, 0}, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tier, "NEXT.md"), []byte("next: land it\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "short-leftover")
	if r.Outcome != "reclaimed" {
		t.Fatalf("a short build leftover does not keep the worktree: %+v", r)
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	if exists(filepath.Join(entry, "bin", "abcd.12345")) {
		t.Fatal("the leftover is left out of the archive")
	}
}

// TestPruneKeepsAWorktreeWhenTheFileAtABuildNameCannotBeRead: a file the walk
// cannot read could be anyone's, so it is no build product, and the worktree stays
// with the file named rather than removing the only copy.
func TestPruneKeepsAWorktreeWhenTheFileAtABuildNameCannotBeRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a file whatever its mode")
	}
	repo, _ := pruneFixture(t)
	p := mergedWorktree(t, repo, "unreadable")
	bin := filepath.Join(p, ".abcd", ".work.local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "abcd.123"), []byte("kept by hand\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "unreadable")
	if r.Outcome != "kept" || !strings.Contains(r.Reason, "bin/abcd.123") {
		t.Fatalf("an unreadable file at a build name keeps the worktree, naming it: %+v", r)
	}
	if !exists(p) {
		t.Fatal("the kept worktree is still there")
	}
}

// TestPruneStillJudgesABinDeeperInTheTier: only the tier's top-level bin/ holds
// the build; a bin/ under scratch/ is notes like any other, and a binary there
// still keeps the worktree.
func TestPruneStillJudgesABinDeeperInTheTier(t *testing.T) {
	repo, _ := pruneFixture(t)
	p := mergedWorktree(t, repo, "deep")
	deep := filepath.Join(p, ".abcd", ".work.local", "scratch", "bin")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "tool"), []byte{0, 1, 2}, 0o755); err != nil {
		t.Fatal(err)
	}
	got := prune(t, repo, worktree.PruneOptions{})
	r := pruneRow(t, got, "deep")
	if r.Outcome != "kept" || !strings.Contains(r.Reason, "scratch/bin/tool") {
		t.Fatalf("a binary under scratch/bin keeps the worktree: %+v", r)
	}
}

// TestPruneArchivesWhatElseTheTiersBinHolds: only the binary scripts/abcd.sh
// builds, and the abcd.<pid> file it builds that binary through, are left out
// of the archive. A file a person keeps beside them is a note, and removing the
// worktree would delete its only copy (iss-2611041759314115).
func TestPruneArchivesWhatElseTheTiersBinHolds(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "kept-script")
	bin := filepath.Join(p, ".abcd", ".work.local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 5<<20) // over the 4 MiB per-file cap, and NUL bytes: not text
	for _, name := range []string{"abcd", "abcd.12345"} {
		if err := os.WriteFile(filepath.Join(bin, name), big, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "run.sh"), []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("run.sh", filepath.Join(bin, "abcd.link")); err != nil {
		t.Fatal(err)
	}
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "kept-script")
	if r.Outcome != "reclaimed" {
		t.Fatalf("a merged, clean worktree whose tier's bin/ holds the build and a script is reclaimed: %+v", r)
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	b, err := os.ReadFile(filepath.Join(entry, "bin", "run.sh"))
	if err != nil {
		t.Fatalf("the script beside the binary is archived: %v", err)
	}
	if !strings.Contains(string(b), "echo mine") {
		t.Fatalf("the archived script holds what the person wrote: %q", b)
	}
	manifest, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"bin/abcd.link"`) {
		t.Fatalf("a link whose name only starts like the build is in the manifest: %s", manifest)
	}
	for _, name := range []string{"abcd", "abcd.12345"} {
		if exists(filepath.Join(entry, "bin", name)) {
			t.Fatalf("the build's %s is left out of the archive", name)
		}
	}
}

// TestPruneArchivesALinkAtTheBinarysPath: the build is a regular file, so a link
// a person put at the binary's own path is archived as a link, like any other.
func TestPruneArchivesALinkAtTheBinarysPath(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "linked")
	bin := filepath.Join(p, ".abcd", ".work.local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(bin, "abcd")); err != nil {
		t.Fatal(err)
	}
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "linked")
	if r.Outcome != "reclaimed" {
		t.Fatalf("a merged, clean worktree is reclaimed: %+v", r)
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	manifest, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"bin/abcd"`) {
		t.Fatalf("the link is in the manifest: %s", manifest)
	}
}

// TestPruneStillJudgesWhatIsNotTheBuildInTheTiersBin: a file in the tier's bin/
// that is neither the binary nor its abcd.<pid> file is judged like any note, so
// one the archive cannot vouch for keeps the worktree, and so does one inside a
// directory at the binary's own path.
func TestPruneStillJudgesWhatIsNotTheBuildInTheTiersBin(t *testing.T) {
	for _, name := range []string{"tool", "abcd.old", "abcd.", "abcd.12a"} {
		t.Run(name, func(t *testing.T) {
			repo, _ := pruneFixture(t)
			p := mergedWorktree(t, repo, "other-binary")
			bin := filepath.Join(p, ".abcd", ".work.local", "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, name), []byte{0, 1, 2}, 0o755); err != nil {
				t.Fatal(err)
			}
			r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "other-binary")
			if r.Outcome != "kept" || !strings.Contains(r.Reason, "bin/"+name) {
				t.Fatalf("a binary at bin/%s keeps the worktree: %+v", name, r)
			}
			if !exists(p) {
				t.Fatal("the kept worktree is still there")
			}
		})
	}
	t.Run("a directory at the binary's path", func(t *testing.T) {
		repo, _ := pruneFixture(t)
		p := mergedWorktree(t, repo, "other-binary")
		dir := filepath.Join(p, ".abcd", ".work.local", "bin", "abcd")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tool"), []byte{0, 1, 2}, 0o755); err != nil {
			t.Fatal(err)
		}
		r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "other-binary")
		if r.Outcome != "kept" || !strings.Contains(r.Reason, "bin/abcd/tool") {
			t.Fatalf("a binary inside a directory at the binary's path keeps the worktree: %+v", r)
		}
	})
}

// generatedSet is a generated-set cache as the reservation views keep it
// (internal/core/claim/generatedcache.go): its magic, format, key and outcome,
// then each path, every field ended by a NUL. It is written out here rather than
// through claim, so the archive is held to the bytes on disk.
var generatedSet = []byte("abcd generated set\x001\x00" + strings.Repeat("ab", 32) + "\x00set\x00go.sum\x00")

// writeTierFile writes data at the tier-relative rel of the worktree at p.
func writeTierFile(t *testing.T, p, rel string, data []byte) {
	t.Helper()
	f := filepath.Join(p, ".abcd", ".work.local", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestPruneLeavesTheGeneratedSetCacheOutOfTheNotes: the generated-set cache is
// abcd's own, re-derived on the next reservation view, and its fields are
// NUL-separated, so the redaction pass cannot vouch for it. The archive leaves
// it out rather than keeping every merged worktree a view ever ran in
// (iss-2611181412255215, iss-2611221058212814); the reservation files beside it
// and the person's notes are still archived, and a dry run reports the reclaim
// a real run makes.
func TestPruneLeavesTheGeneratedSetCacheOutOfTheNotes(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "cached")
	writeTierFile(t, p, "reservation/generated", generatedSet)
	writeTierFile(t, p, "reservation/copy-key", []byte(strings.Repeat("c", 32)+"\n"))
	writeTierFile(t, p, "NEXT.md", []byte("next: land it\n"))

	dry := pruneRow(t, prune(t, repo, worktree.PruneOptions{DryRun: true}), "cached")
	if dry.Outcome != "would reclaim" || dry.Notes != "2 files" {
		t.Fatalf("a dry run would reclaim the worktree, archiving its two notes: %+v", dry)
	}
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "cached")
	if r.Outcome != "reclaimed" || r.Notes != "2 files" {
		t.Fatalf("a merged, clean worktree whose tier holds the generated-set cache is reclaimed: %+v", r)
	}
	if exists(p) {
		t.Fatal("the reclaimed worktree is gone")
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	for _, rel := range []string{"NEXT.md", "reservation/copy-key"} {
		if _, err := os.Stat(filepath.Join(entry, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s beside the cache is archived: %v", rel, err)
		}
	}
	if exists(filepath.Join(entry, "reservation", "generated")) {
		t.Fatal("the generated-set cache is left out of the archive")
	}
	manifest, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), "reservation/generated") {
		t.Fatalf("the manifest does not list the cache: %s", manifest)
	}
}

// TestPruneReclaimsATierHoldingOnlyTheGeneratedSetCache: with the cache the
// tier's only file, there is nothing to archive and no entry is written.
func TestPruneReclaimsATierHoldingOnlyTheGeneratedSetCache(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "cache-only")
	writeTierFile(t, p, "reservation/generated", generatedSet)
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "cache-only")
	if r.Outcome != "reclaimed" || r.Notes != "none" {
		t.Fatalf("a worktree whose tier holds only the cache is reclaimed with no notes: %+v", r)
	}
	if entries := notesEntries(t, repo, home); len(entries) != 0 {
		t.Fatalf("no archive entry for a tier with no notes: %v", entries)
	}
}

// TestPruneArchivesTextAtTheGeneratedSetsName: the cache is never text, so text
// at its path is what a person wrote, even text that opens like the cache, and
// removing the worktree would delete its only copy.
func TestPruneArchivesTextAtTheGeneratedSetsName(t *testing.T) {
	repo, home := pruneFixture(t)
	p := mergedWorktree(t, repo, "text-at-cache")
	writeTierFile(t, p, "reservation/generated", []byte("abcd generated set\nkept by hand\n"))
	r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "text-at-cache")
	if r.Outcome != "reclaimed" || r.Notes != "1 file" {
		t.Fatalf("a merged, clean worktree whose tier holds text at the cache's path is reclaimed: %+v", r)
	}
	entries := notesEntries(t, repo, home)
	if len(entries) != 1 {
		t.Fatalf("one archive entry: %v", entries)
	}
	entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
	b, err := os.ReadFile(filepath.Join(entry, "reservation", "generated"))
	if err != nil || !strings.Contains(string(b), "kept by hand") {
		t.Fatalf("the text at the cache's path is archived as written: %q, %v", b, err)
	}
}

// TestPruneStillJudgesWhatIsNotTheGeneratedSetCache: only a file at the
// cache's own path that opens with its magic and the NUL ending it is left
// out. Bytes the pass cannot vouch for anywhere else, or at that path without
// the magic, keep the worktree and are named, so nothing a person keeps there is
// dropped.
func TestPruneStillJudgesWhatIsNotTheGeneratedSetCache(t *testing.T) {
	for _, tc := range []struct {
		name, rel string
		data      []byte
	}{
		{"binary at the cache's path", "reservation/generated", []byte{0, 1, 2}},
		{"the magic without its NUL", "reservation/generated", []byte("abcd generated set\xff\x00")},
		{"the cache's bytes elsewhere", "scratch/generated", generatedSet},
		{"the cache's bytes at a sibling name", "reservation/generated.old", generatedSet},
		{"the cache's bytes a level deeper", "scratch/reservation/generated", generatedSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := pruneFixture(t)
			p := mergedWorktree(t, repo, "not-the-cache")
			writeTierFile(t, p, tc.rel, tc.data)
			r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "not-the-cache")
			if r.Outcome != "kept" || !strings.Contains(r.Reason, tc.rel+": not text") {
				t.Fatalf("%s keeps the worktree, naming it: %+v", tc.rel, r)
			}
			if !exists(p) {
				t.Fatal("the kept worktree is still there")
			}
		})
	}
	t.Run("a link at the cache's path", func(t *testing.T) {
		repo, home := pruneFixture(t)
		p := mergedWorktree(t, repo, "linked-cache")
		dir := filepath.Join(p, ".abcd", ".work.local", "reservation")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("elsewhere", filepath.Join(dir, "generated")); err != nil {
			t.Fatal(err)
		}
		r := pruneRow(t, prune(t, repo, worktree.PruneOptions{}), "linked-cache")
		if r.Outcome != "reclaimed" || r.Notes != "1 file" {
			t.Fatalf("a link at the cache's path is archived as a link: %+v", r)
		}
		entries := notesEntries(t, repo, home)
		if len(entries) != 1 {
			t.Fatalf("one archive entry: %v", entries)
		}
		entry := filepath.Join(abcdhome.Path(home, "notes", gittest.RootCommit(t, repo.Root())), entries[0])
		manifest, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
		if err != nil || !strings.Contains(string(manifest), `"reservation/generated"`) {
			t.Fatalf("the link is in the manifest: %s, %v", manifest, err)
		}
	})
}

// TestNotesDirReadsOrMakesTheLanesArchive: without create, an archive not yet
// made is absent and nothing is made; with create, each level is made the
// caller's alone and the lane's directory answered; a symlink standing in for
// the lane's level refuses either way, and a Lane that is not LaneFor's answer
// is refused before anything is read.
func TestNotesDirReadsOrMakesTheLanesArchive(t *testing.T) {
	repo, home := repoWithCommit(t)
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	want := abcdhome.Path(home, "notes", l.RootSHA)
	if dir, present, err := l.NotesDir(false); err != nil || present || dir != want {
		t.Fatalf("NotesDir(false) before the archive exists = %q, %v, %v; want %q, absent", dir, present, err, want)
	}
	if exists(abcdhome.Path(home)) {
		t.Fatal("a read of the archive makes nothing")
	}
	dir, present, err := l.NotesDir(true)
	if err != nil || !present || dir != want {
		t.Fatalf("NotesDir(true) = %q, %v, %v; want %q, present", dir, present, err, want)
	}
	for _, level := range []string{abcdhome.Path(home), abcdhome.Path(home, "notes"), want} {
		if fi, err := os.Lstat(level); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s is a 0700 directory: %v %v", level, fi, err)
		}
	}
	if _, present, err := l.NotesDir(false); err != nil || !present {
		t.Fatalf("NotesDir(false) once made = %v, %v; want present", present, err)
	}

	if err := os.Remove(want); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), want); err != nil {
		t.Fatal(err)
	}
	for _, create := range []bool{false, true} {
		if _, _, err := l.NotesDir(create); err == nil || strings.Contains(err.Error(), home) {
			t.Fatalf("NotesDir(%v) over a symlinked level = %v; want a home-redacted refusal", create, err)
		}
	}
	bad := worktree.Lane{Home: home, RootSHA: l.RootSHA, Rel: l.Rel, Dir: filepath.Join(home, "elsewhere")}
	mustRefusal(t, func() error { _, _, err := bad.NotesDir(true); return err }())
}
