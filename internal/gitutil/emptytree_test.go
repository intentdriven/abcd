package gitutil_test

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestEmptyTreeIsNamedForTheObjectFormatAndWritten names the empty tree under
// both object formats git ships, and holds that the object then exists in the
// store, so a commit made on it can be pushed.
func TestEmptyTreeIsNamedForTheObjectFormatAndWritten(t *testing.T) {
	sha1 := gittest.NewRepo(t)
	got, err := gitutil.EmptyTree(sha1.Root())
	if err != nil || got != "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
		t.Fatalf("EmptyTree under SHA-1 = %q, %v", got, err)
	}
	if typ := sha1.Git("cat-file", "-t", got); typ != "tree" {
		t.Fatalf("the empty tree's object type = %q; want it written as a tree", typ)
	}

	dir := filepath.Join(t.TempDir(), "sha256")
	sha1.Git("init", "-q", "--object-format=sha256", dir)
	got, err = gitutil.EmptyTree(dir)
	if err != nil || got != "6ef19b41225c5369f1c104d45d8d85efa9b057b53b14b4b9b939dd74decc5321" {
		t.Fatalf("EmptyTree under SHA-256 = %q, %v", got, err)
	}
	if typ := sha1.Git("-C", dir, "cat-file", "-t", got); typ != "tree" {
		t.Fatalf("the SHA-256 empty tree's object type = %q; want it written as a tree", typ)
	}
}
