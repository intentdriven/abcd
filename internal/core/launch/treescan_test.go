package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ScanTree is the launch bundle's scan over a whole tree, for a model-matrix
// copy's clone before it is routed to a provider (spc-2610161949559050 step
// 3): the same scanner and the same refusals, so a clone the bundle's gate
// would refuse is refused before it leaves the machine.

// TestScanTreeRefusesASecretShapedFile: a token anywhere in the tree refuses
// it, naming the scan's hard fail.
func TestScanTreeRefusesASecretShapedFile(t *testing.T) {
	scanRoot, tree := t.TempDir(), t.TempDir()
	writeFile(t, tree, "README.md", "# a clone\n")
	writeFile(t, tree, "deep/in/config.txt", "token = "+fakeSecret+"\n")
	refusals, err := ScanTree(scanRoot, tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(refusals) == 0 || !strings.Contains(strings.Join(refusals, "\n"), "hard-fail") {
		t.Fatalf("a secret-shaped file refuses the tree: %v", refusals)
	}
}

// TestScanTreePassesACleanTreeAndSkipsTheGitDirectory: a clean tree has no
// refusal, and the clone's own git directory is not part of what is sent.
func TestScanTreePassesACleanTreeAndSkipsTheGitDirectory(t *testing.T) {
	scanRoot, tree := t.TempDir(), t.TempDir()
	writeFile(t, tree, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, tree, ".git/packed", "token = "+fakeSecret+"\n")
	refusals, err := ScanTree(scanRoot, tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(refusals) != 0 {
		t.Fatalf("a clean tree has no refusal: %v", refusals)
	}
}

// TestScanTreeRefusesWhatItCannotCover: a symbolic link is not followed and
// is left unscanned, which refuses the tree as the bundle's gate does, rather
// than reading whatever it points at or passing it unread.
func TestScanTreeRefusesWhatItCannotCover(t *testing.T) {
	scanRoot, tree := t.TempDir(), t.TempDir()
	writeFile(t, tree, "a.txt", "plain\n")
	if err := os.Symlink(filepath.Join(tree, "a.txt"), filepath.Join(tree, "link")); err != nil {
		t.Fatal(err)
	}
	refusals, err := ScanTree(scanRoot, tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(refusals) == 0 || !strings.Contains(strings.Join(refusals, "\n"), "link") {
		t.Fatalf("an uncoverable entry refuses the tree, naming it: %v", refusals)
	}
}
