package fsutil

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateExclusiveInFollowsAnInRootSymlinkedAncestor pins what os.Root
// does under CreateExclusiveIn, the containment IsRealDir's doc comment sends a
// caller to: an ancestor symlinked to somewhere INSIDE the root is followed and
// the file lands at the link's target, while an ancestor symlinked OUT of the
// root is refused and nothing is written there. Containment, not a refusal of
// every symlink, is the guarantee (iss-2610112206165535).
func TestCreateExclusiveInFollowsAnInRootSymlinkedAncestor(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "elsewhere"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(base, "inside")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(base, "outward")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := CreateExclusiveIn(root, "inside/f.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("an ancestor symlinked inside the root was refused: %v", err)
	}
	fi, err := os.Lstat(filepath.Join(base, "elsewhere", "f.txt"))
	if err != nil {
		t.Fatalf("the file did not land at the in-root link's target: %v", err)
	}
	if !fi.Mode().IsRegular() {
		t.Errorf("the file at the link's target is not regular: %v", fi.Mode())
	}

	if err := CreateExclusiveIn(root, "outward/f.txt", []byte("x"), 0o644); err == nil {
		t.Fatal("an ancestor symlinked out of the root was followed")
	}
	if _, err := os.Lstat(filepath.Join(outside, "f.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a file was written outside the root: %v", err)
	}
}

// TestIsRealDirDocSaysWhatOSRootRefuses holds IsRealDir's doc comment to the
// behaviour the test above pins: it must not claim os.Root refuses symlink
// traversal at every level, and must say an in-root link is followed and where
// a walk that refuses every symlinked level lives (iss-2610112206165535).
func TestIsRealDirDocSaysWhatOSRootRefuses(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "fsutil.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "IsRealDir" && fn.Doc != nil {
			doc = strings.Join(strings.Fields(fn.Doc.Text()), " ")
		}
	}
	if doc == "" {
		t.Fatal("IsRealDir has no doc comment")
	}
	if strings.Contains(doc, "symlink traversal at every level") {
		t.Errorf("IsRealDir's doc still claims os.Root refuses symlinks at every level: %q", doc)
	}
	for _, want := range []string{"escape", "followed", "EnsureRealDirAll"} {
		if !strings.Contains(doc, want) {
			t.Errorf("IsRealDir's doc does not say %q: %q", want, doc)
		}
	}
}
