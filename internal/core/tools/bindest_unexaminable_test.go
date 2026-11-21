package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestABinFolderThatCannotBeExaminedSaysWhy: XDG_BIN_HOME names a folder
// under a regular file, so its steps cannot be examined (not a directory).
// The install still stops before anything is downloaded, but it says what
// stopped it, not that the folder lies inside the repository.
func TestABinFolderThatCannotBeExaminedSaysWhy(t *testing.T) {
	r := newRig(t)
	file := filepath.Join(r.home, "file")
	if err := os.WriteFile(file, []byte("not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.env["XDG_BIN_HOME"] = filepath.Join(file, "bin")
	res := r.installer().Install("gitleaks", TranscriptScan, yes)
	if res.Ran || res.Installed || r.hits != 0 {
		t.Fatalf("an unexaminable bin folder ran the install: %+v", res)
	}
	if strings.Contains(res.Why, "inside the repository") {
		t.Errorf("the refusal says the folder lies inside the repository, which it does not: %q", res.Why)
	}
	if !strings.Contains(res.Why, "not a directory") {
		t.Errorf("the refusal does not name the cause: %q", res.Why)
	}
}

// TestAdmitSaysWhenAProgramCannotBeExamined: a program on PATH whose link
// target d/../x is read through a directory link to a folder outside the
// repository cannot have its steps examined, so it is refused, but nothing on
// its way lies inside the repository and the refusal does not say so.
func TestAdmitSaysWhenAProgramCannotBeExamined(t *testing.T) {
	r := newRig(t)
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(elsewhere, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(t.TempDir(), "gitleaks")
	if err := os.WriteFile(real, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(elsewhere, "x")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(elsewhere, "sub"), filepath.Join(r.pathDir, "d")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d/../x", filepath.Join(r.pathDir, "gitleaks")); err != nil {
		t.Fatal(err)
	}
	_, err := r.installer().admit("gitleaks")
	if err == nil {
		t.Fatal("admit accepted a program whose steps cannot be examined")
	}
	if strings.Contains(err.Error(), "inside the repository") {
		t.Errorf("the refusal says the program lies inside the repository, which it does not: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot be judged") {
		t.Errorf("the refusal does not say the program cannot be judged: %v", err)
	}
}
