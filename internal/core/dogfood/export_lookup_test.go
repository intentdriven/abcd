package dogfood

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// TestTheExportDoesNotRunATarFromTheRepository: the tar the export pipes git's
// archive into, found by name on PATH, is held to the in-checkout rule
// (tools.LookPath) for the repository being built: a tar inside it is refused
// and never run.
func TestTheExportDoesNotRunATarFromTheRepository(t *testing.T) {
	repo := realDir(t)
	git(t, repo, "init", "-q")
	write(t, filepath.Join(repo, "README.md"), "fixture\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "first")
	marker := filepath.Join(realDir(t), "ran")
	bin := filepath.Join(repo, "bin")
	write(t, filepath.Join(bin, "tar"), "#!/bin/sh\ntouch '"+marker+"'\ncat >/dev/null\n")
	if err := os.Chmod(filepath.Join(bin, "tar"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := export(context.Background(), repo, "HEAD", realDir(t), io.Discard)
	if !errors.Is(err, tools.ErrInsideWorkingTree) {
		t.Fatalf("export = %v, want tools.ErrInsideWorkingTree", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("the tar inside the repository ran (%v)", err)
	}
}
