package gitutil_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheStoredStreamWritesGitsWholeOutputAsStored: a stored stream hands
// every byte of git's output to the writer, however much of it there is, with
// no cap, and runs git reading history as stored.
func TestTheStoredStreamWritesGitsWholeOutputAsStored(t *testing.T) {
	stubGit(t, `printf 'replace=%s\n' "$GIT_NO_REPLACE_OBJECTS"; i=0; while [ $i -lt 30000 ]; do echo "line $i of a long bundle"; i=$((i+1)); done`)
	var out bytes.Buffer
	if err := gitutil.StreamAsStored(t.TempDir(), &out, "bundle", "create", "-q", "-", "refs/heads/x"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "replace=1\n") {
		t.Fatalf("the stream did not run git as stored: %q", out.String()[:min(out.Len(), 40)])
	}
	if n := strings.Count(out.String(), "\n"); n != 30001 {
		t.Fatalf("the stream wrote %d lines; want 30001", n)
	}
}

// TestAFailedStoredStreamCarriesGitsStderr: a git that fails is an error
// naming what git said.
func TestAFailedStoredStreamCarriesGitsStderr(t *testing.T) {
	stubGit(t, `echo "fatal: Refusing to create empty bundle." >&2; exit 128`)
	err := gitutil.StreamAsStored(t.TempDir(), &bytes.Buffer{}, "bundle", "create", "-q", "-", "refs/heads/x")
	if err == nil || !strings.Contains(err.Error(), "Refusing to create empty bundle") {
		t.Fatalf("err = %v; want git's stderr", err)
	}
}
