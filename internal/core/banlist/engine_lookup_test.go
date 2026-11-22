package banlist

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// relookGrep makes the next lookGrep look the engine up again, and the one
// after the test as well.
func relookGrep(t *testing.T) {
	t.Helper()
	reset := func() { grepBinOnce, grepBin, grepErr = sync.Once{}, "", nil }
	reset()
	t.Cleanup(reset)
}

// TestAGrepInsideTheWorkingTreeIsNamedAsTheReasonNoEngineRuns: a grep PATH
// finds inside the working tree abcd runs from is never run, and the error the
// private layer's matcher and the text scan return says so, still as
// ErrNoEngine, rather than that no grep can be run.
func TestAGrepInsideTheWorkingTreeIsNamedAsTheReasonNoEngineRuns(t *testing.T) {
	checkout := t.TempDir()
	bin, sub := filepath.Join(checkout, "bin"), filepath.Join(checkout, "pkg")
	marker := filepath.Join(t.TempDir(), "ran")
	for _, d := range []string{filepath.Join(checkout, ".git"), bin, sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "grep"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\nk v\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(sub)
	relookGrep(t)

	_, matchErr := MatchPrivate(root, []string{"name"})
	_, scanErr := ScanText([]KeyedPattern{{Key: "k", Pattern: "v"}}, []byte("v\n"))
	for what, err := range map[string]error{"MatchPrivate": matchErr, "ScanText": scanErr} {
		if !errors.Is(err, ErrNoEngine) || !errors.Is(err, tools.ErrInsideWorkingTree) ||
			!strings.Contains(err.Error(), "never run") {
			t.Errorf("%s = %v, want ErrNoEngine naming the in-checkout refusal", what, err)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("the grep inside the working tree ran (%v)", err)
	}
}
