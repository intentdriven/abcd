package gitutil_test

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestArchiveTreeAgreesWithGitArchive holds ArchiveTree to the tree `git
// archive HEAD` actually writes, over every export-ignore form a repository can
// declare: a file, a directory named without a slash, the directory-only form
// `dir/` (at the root and in a nested .gitattributes), a negated glob, a
// directory-only pattern whose name is a file, which must not match it, and a
// directory ignored by name but re-included by the directory-only form, which
// archive keeps because it asks a directory's attributes as `dir/` alone.
func TestArchiveTreeAgreesWithGitArchive(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("keep.txt", "kept\n")
	r.Write("ignored.txt", "file pattern\n")
	r.Write("plaindir/a.txt", "directory named without a slash\n")
	r.Write("slashdir/c.txt", "directory-only pattern\n")
	r.Write("slashdir/deeper/e.txt", "beneath a directory-only pattern\n")
	r.Write("nested/keep.txt", "kept\n")
	r.Write("nested/sub/d.txt", "nested directory-only pattern\n")
	r.Write("nested/.gitattributes", "sub/ export-ignore\n")
	r.Write("neg/x.txt", "negated glob drops this\n")
	r.Write("neg/keep.txt", "negation keeps this\n")
	r.Write("notadir", "a file a directory-only pattern names\n")
	r.Write("reincluded/f.txt", "ignored by name, re-included by the directory-only form\n")
	r.Write(".gitattributes", strings.Join([]string{
		"ignored.txt export-ignore",
		"plaindir export-ignore",
		"slashdir/ export-ignore",
		"neg/* export-ignore",
		"neg/keep.txt -export-ignore",
		"notadir/ export-ignore",
		"reincluded export-ignore",
		"reincluded/ -export-ignore",
	}, "\n")+"\n")
	if err := os.Symlink("keep.txt", filepath.Join(r.Root(), "link.txt")); err != nil {
		t.Fatal(err)
	}
	r.Commit("fixture")

	entries, err := gitutil.ArchiveTree(r.Root(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Path)
	}
	sort.Strings(got)

	want := archivedFiles(t, r)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ArchiveTree disagrees with git archive HEAD:\n got: %v\nwant: %v", got, want)
	}
	// The fixture must exercise what it claims: the directory-only forms drop
	// their trees, and a directory-only pattern leaves a file of that name.
	for _, p := range []string{"slashdir/c.txt", "slashdir/deeper/e.txt", "nested/sub/d.txt"} {
		for _, g := range got {
			if g == p {
				t.Errorf("%s is export-ignored by a directory-only pattern but was listed", p)
			}
		}
	}
	if !contains(got, "reincluded/f.txt") {
		t.Errorf("reincluded/ -export-ignore re-includes the directory in the archive; it must be listed (got %v)", got)
	}
	if !contains(got, "notadir") {
		t.Errorf("notadir is a file; the directory-only pattern notadir/ must not drop it (got %v)", got)
	}
}

// TestArchiveTreeReadsAnOptionShapedRevisionAsARevision holds the positional
// revision behind --end-of-options: a rev that looks like an option is looked
// up as an object name and refused as one, never parsed as a flag.
func TestArchiveTreeReadsAnOptionShapedRevisionAsARevision(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("fixture")
	_, err := gitutil.ArchiveTree(r.Root(), "--name-only")
	if err == nil {
		t.Fatal("an option-shaped revision names no commit and must be refused")
	}
	if strings.Contains(err.Error(), "usage:") || !strings.Contains(err.Error(), "--name-only") {
		t.Errorf("the revision was parsed as an option, not looked up as an object name: %v", err)
	}
}

// TestArchiveTreeReadsAttributesFromTheRevision holds the listing to the
// revision's own .gitattributes, as git archive reads them, when the index
// holds a different one: a staged but uncommitted export-ignore neither drops
// a file the archive carries nor keeps one it drops (iss-2609260933592838).
func TestArchiveTreeReadsAttributesFromTheRevision(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("keep.txt", "kept\n")
	r.Write("secret.txt", "ignored in the index only\n")
	r.Write("dropped.txt", "ignored in the revision only\n")
	r.Write("nested/n.txt", "ignored by a nested file added in the index only\n")
	r.Write(".gitattributes", "dropped.txt export-ignore\n")
	r.Commit("fixture")
	r.Write(".gitattributes", "secret.txt export-ignore\n")
	r.Write("nested/.gitattributes", "n.txt export-ignore\n")
	r.Git("add", ".gitattributes", "nested/.gitattributes")

	entries, err := gitutil.ArchiveTree(r.Root(), "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "git 2.40") {
			t.Skipf("this git reads no attributes from a revision: %v", err)
		}
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Path)
	}
	sort.Strings(got)
	if want := archivedFiles(t, r); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ArchiveTree read the index's attributes, not HEAD's:\n got: %v\nwant: %v", got, want)
	}
	if !contains(got, "secret.txt") || !contains(got, "nested/n.txt") || contains(got, "dropped.txt") {
		t.Fatalf("the fixture does not exercise both directions: %v", got)
	}
}

// TestArchiveTreeRefusesWhenItCannotReadTheRevisionsAttributes: where the index's
// .gitattributes differ from the revision's and git cannot read attributes from
// a revision (check-attr --source needs git 2.40), the listing is refused with
// the remedy, never answered from the index (iss-2609260933592838).
func TestArchiveTreeRefusesWhenItCannotReadTheRevisionsAttributes(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("keep.txt", "kept\n")
	r.Write(".gitattributes", "\n")
	r.Commit("fixture")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in --source=*) echo \"error: unknown option $a\" >&2; exit 129;; esac; done\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	// The same attributes in the index and the revision: answered on any git.
	if _, err := gitutil.ArchiveTree(r.Root(), "HEAD"); err != nil {
		t.Fatalf("identical attributes need no --source: %v", err)
	}
	r.Write(".gitattributes", "keep.txt export-ignore\n")
	r.Git("add", ".gitattributes")
	_, err = gitutil.ArchiveTree(r.Root(), "HEAD")
	if err == nil || !strings.Contains(err.Error(), ".gitattributes") || !strings.Contains(err.Error(), "git 2.40") {
		t.Fatalf("a staged attributes change on a git without --source: err %v", err)
	}
}

// archivedFiles lists the non-directory members of `git archive HEAD`.
func archivedFiles(t *testing.T, r *gittest.Repo) []string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "head.tar")
	r.Git("archive", "--format=tar", "-o", out, "HEAD")
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var files []string
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeDir || h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		files = append(files, h.Name)
	}
	sort.Strings(files)
	return files
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
