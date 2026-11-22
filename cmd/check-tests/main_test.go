package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/testreuse"
)

// TestARecipeThatIsNotAGoTestIsRefused: the recipe writes the go test command out
// whole after the stage, so anything else is a recipe mistake, which fails the
// stage there rather than passing over no tests.
func TestARecipeThatIsNotAGoTestIsRefused(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"unit"},
		{"unit", "go"},
		{"unit", "go", "vet", "./..."},
		{"-unknown", "unit", "go", "test", "./..."},
		{"unit", "go", "test", "-count=2", "./..."},
	} {
		var out, errb bytes.Buffer
		code := run(context.Background(), args, t.TempDir(), t.TempDir(), &out, &errb)
		if code != testreuse.ExitUsage {
			t.Errorf("%q exited %d, want %d: %s", args, code, testreuse.ExitUsage, errb.String())
		}
		if !strings.Contains(errb.String(), "check-tests") && !strings.Contains(errb.String(), "flag") {
			t.Errorf("%q says nothing useful: %s", args, errb.String())
		}
	}
}

// TestTheRaceGateDoesNotRunAGoFromTheCheckout: the race gate finds go as the
// stage's go lines do, so a go first on PATH inside the checkout is not run
// and the gate says why.
func TestTheRaceGateDoesNotRunAGoFromTheCheckout(t *testing.T) {
	wd := t.TempDir()
	bin := filepath.Join(wd, "bin")
	if err := os.MkdirAll(filepath.Join(wd, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "ran")
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errb bytes.Buffer
	err := passRaceGate(context.Background(), wd, &out, &errb)
	if !errors.Is(err, tools.ErrInsideWorkingTree) {
		t.Errorf("the race gate = %v, want the in-checkout refusal", err)
	}
	if _, serr := os.Stat(marker); !os.IsNotExist(serr) {
		t.Fatalf("the go inside the checkout ran (%v)", serr)
	}
}

// TestTheCheckDirectoryIsThePersonsUnderAStagesHome: a drive's preflight keeps the
// check directory its results are reused from, which a stage's home would lose.
func TestTheCheckDirectoryIsThePersonsUnderAStagesHome(t *testing.T) {
	stage, person := t.TempDir(), t.TempDir()
	t.Setenv("HOME", stage)
	t.Setenv(abcdhome.StageHomeEnv, stage)
	t.Setenv(abcdhome.PersonHomeEnv, person)
	if got := checkHome(); got != person {
		t.Fatalf("the check directory is kept under %q, want %q", got, person)
	}
}
