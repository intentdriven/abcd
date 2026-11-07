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
