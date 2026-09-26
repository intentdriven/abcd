package gitutil_test

import (
	"archive/tar"
	"errors"
	"io"
	"os"
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
