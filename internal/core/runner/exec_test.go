package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// putOnPath writes an executable named name running script into dir and makes
// dir the only PATH entry beside the system folders a shell script needs.
func putOnPath(t *testing.T, dir, name, script string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
}

// TestExecKeepsTheStreamsApart: a command another package runs through Exec
// is admitted and run as a harness is, its two streams returned apart, in the
// directory it names.
func TestExecKeepsTheStreamsApart(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	putOnPath(t, bin, "vendortool", `echo "1.2.3"; echo "noise" >&2; pwd`)
	work := t.TempDir()
	out, err := Exec(context.Background(), Command{Name: "vendortool", Args: []string{"--version"}, Dir: work, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	resolved, _ := filepath.EvalSymlinks(work)
	if got := string(out.Stdout); !strings.HasPrefix(got, "1.2.3\n") || !strings.Contains(got, resolved) {
		t.Errorf("stdout = %q, want the version then the directory %s", got, resolved)
	}
	if string(out.Stderr) != "noise\n" {
		t.Errorf("stderr = %q", out.Stderr)
	}
}

// TestExecNamesTheStageThatFailed: each launch failure Exec meets is a
// Failure whose errors.Is names its stage, and the admission is the
// harness's own, the refusal of a folder others can write included.
func TestExecNamesTheStageThatFailed(t *testing.T) {
	t.Run("not on PATH", func(t *testing.T) {
		putOnPath(t, filepath.Join(t.TempDir(), "bin"), "othertool", "true")
		_, err := Exec(context.Background(), Command{Name: "vendortool", Timeout: time.Second})
		if !errors.Is(err, ErrNotOnPath) {
			t.Errorf("err = %v, want ErrNotOnPath", err)
		}
	})
	t.Run("inside a guarded folder", func(t *testing.T) {
		repo := t.TempDir()
		putOnPath(t, filepath.Join(repo, "tools"), "vendortool", "true")
		_, err := Exec(context.Background(), Command{Name: "vendortool", Guards: []string{repo}, Timeout: time.Second})
		if !errors.Is(err, ErrRefused) {
			t.Errorf("err = %v, want ErrRefused", err)
		}
	})
	t.Run("a folder others can write", func(t *testing.T) {
		bin := filepath.Join(t.TempDir(), "bin")
		putOnPath(t, bin, "vendortool", "true")
		if err := os.Chmod(bin, 0o777); err != nil {
			t.Fatal(err)
		}
		_, err := Exec(context.Background(), Command{Name: "vendortool", Timeout: time.Second})
		if !errors.Is(err, ErrRefused) {
			t.Errorf("err = %v, want ErrRefused", err)
		}
	})
	t.Run("past its time", func(t *testing.T) {
		putOnPath(t, filepath.Join(t.TempDir(), "bin"), "vendortool", "sleep 5")
		start := time.Now()
		_, err := Exec(context.Background(), Command{Name: "vendortool", Timeout: 200 * time.Millisecond})
		var fl *Failure
		if !errors.As(err, &fl) || fl.Reason != ReasonFailed {
			t.Errorf("err = %v, want a failed run", err)
		}
		for _, stage := range []error{ErrNotOnPath, ErrRefused, ErrNotStarted} {
			if errors.Is(err, stage) {
				t.Errorf("a killed run is named as the stage %v", stage)
			}
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("Exec waited %s on a command past its time", elapsed)
		}
	})
	t.Run("no time bound", func(t *testing.T) {
		putOnPath(t, filepath.Join(t.TempDir(), "bin"), "vendortool", "true")
		if _, err := Exec(context.Background(), Command{Name: "vendortool"}); err == nil {
			t.Error("a command without a time bound was run")
		}
	})
}

// TestExecTakesOnlyAFixedName: Name is a program's fixed name looked up on
// PATH, so a name carrying a separator is refused before anything resolves
// it, even when it names an admissible file directly.
func TestExecTakesOnlyAFixedName(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	putOnPath(t, bin, "vendortool", "true")
	for _, name := range []string{filepath.Join(bin, "vendortool"), "bin/vendortool"} {
		_, err := Exec(context.Background(), Command{Name: name, Dir: "/", Timeout: time.Second})
		if !errors.Is(err, ErrRefused) {
			t.Errorf("Exec(%q) err = %v, want ErrRefused", name, err)
		}
	}
}

// TestExecRunsAtTheRootWithoutADir: a command naming no directory runs at the
// file-system root, never in the caller's working directory, which for abcd
// is usually the project a vendor binary must not read settings from.
func TestExecRunsAtTheRootWithoutADir(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	putOnPath(t, bin, "vendortool", "pwd")
	t.Chdir(t.TempDir())
	out, err := Exec(context.Background(), Command{Name: "vendortool", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if got := string(out.Stdout); got != "/\n" {
		t.Errorf("ran in %q, want the root", got)
	}
}
