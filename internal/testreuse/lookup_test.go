package testreuse

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// TestTheCheckDoesNotRunAProgramFromTheWorkingTree: the go the check lists
// packages and runs test lines with, and the bash it reads the tree's state
// with, each found by name on PATH, are held to the in-checkout rule
// (tools.LookPath): a program inside the working tree is refused and never run.
func TestTheCheckDoesNotRunAProgramFromTheWorkingTree(t *testing.T) {
	checkout := t.TempDir()
	bin, sub := filepath.Join(checkout, "bin"), filepath.Join(checkout, "pkg")
	marker := filepath.Join(t.TempDir(), "ran")
	for _, d := range []string{filepath.Join(checkout, ".git"), bin, sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go", "bash"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := context.Background()

	if _, err := listPackages(ctx, sub, nil, false, "./..."); !errors.Is(err, tools.ErrInsideWorkingTree) {
		t.Errorf("listPackages = %v, want tools.ErrInsideWorkingTree", err)
	}
	if failed := (lineRunner{ctx: ctx}).exec(goLine{dir: sub, args: []string{"test", "./..."}}, io.Discard, io.Discard); !failed {
		t.Error("a test line whose go lies inside the working tree did not fail")
	}
	if why := treeNotCommit(ctx, checkout); why == "" {
		t.Error("the tree's state read as clean through a bash inside the working tree")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("a program inside the working tree ran (%v)", err)
	}
}
