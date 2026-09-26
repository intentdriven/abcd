package launch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// itd-2609150819432059 AC8 / decision 8: a non-plugin kind with no payload
// include config previews the tree the release tag would archive — git
// archive's view of HEAD, export-ignore honoured — with the record namespace
// denied by the same DenyNamespaces a plugin payload is held to.
func TestResolveArchiveBundleIsTheArchivedTreeMinusTheRecordNamespace(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("main.go", "package main\n")
	r.Write("cmd/tool/run.sh", "#!/bin/sh\n")
	r.Write(".abcd/work/CONTEXT.md", "the record\n")
	r.Write("docs/.abcd/nested.md", "a nested record namespace\n")
	r.Write("testdata/big.bin", "fixture\n")
	r.Write("vendor-notes/a.txt", "export-ignored directory\n")
	r.Write(".gitattributes", "testdata/big.bin export-ignore\nvendor-notes export-ignore\n")
	if err := os.Chmod(filepath.Join(r.Root(), "cmd/tool/run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("main.go", filepath.Join(r.Root(), "link.go")); err != nil {
		t.Fatal(err)
	}
	r.Commit("the tree a tag would archive")
	// Present in the working tree, absent from HEAD: never archived.
	r.Write("untracked.txt", "not committed\n")

	b, err := ResolveArchiveBundle(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	included := map[string]string{}
	for _, f := range b.Included {
		included[f.LogicalPath] = f.GitMode
	}
	want := map[string]string{".gitattributes": "100644", "main.go": "100644", "cmd/tool/run.sh": "100755"}
	if len(included) != len(want) {
		t.Fatalf("included %v, want %v", included, want)
	}
	for p, mode := range want {
		if included[p] != mode {
			t.Errorf("included[%s] = %q, want %q (all: %v)", p, included[p], mode, included)
		}
	}
	excluded := map[string]ExcludedReason{}
	for _, e := range b.Excluded {
		excluded[e.LogicalPath] = e.Reason
	}
	for p, reason := range map[string]ExcludedReason{
		".abcd/work/CONTEXT.md": ExcludedDeniedNamespace,
		"docs/.abcd/nested.md":  ExcludedDeniedNamespace,
		"link.go":               ExcludedSymlink,
	} {
		if excluded[p] != reason {
			t.Errorf("excluded[%s] = %q, want %q (all: %v)", p, excluded[p], reason, excluded)
		}
	}
	for _, p := range []string{"testdata/big.bin", "vendor-notes/a.txt", "untracked.txt"} {
		if _, ok := included[p]; ok {
			t.Errorf("%s is not in the archived tree but was included", p)
		}
	}
	if len(b.Rejected) != 0 {
		t.Errorf("rejected %v, want none", b.Rejected)
	}
}

func TestResolveArchiveBundleWithNoCommitIsAPreflightFault(t *testing.T) {
	r := gittest.NewRepo(t)
	if _, err := ResolveArchiveBundle(r.Root()); err == nil {
		t.Fatal("a repository with no HEAD has no tree to archive, and the preview must say so")
	}
}
