package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shInsideTheTree puts a program named sh in a bin folder of root, first on
// PATH, that leaves a marker file when it runs, and returns the marker's path.
func shInsideTheTree(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "ran")
	if err := os.WriteFile(filepath.Join(bin, "sh"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return marker
}

// TestThePreviousStatusCommandRunsUnderTheSystemShell: the recorded previous
// status command runs through /bin/sh, so an sh first on PATH inside the
// working tree is never run and the command's own output is the line.
func TestThePreviousStatusCommandRunsUnderTheSystemShell(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	writeUserSettings(t, `{"schema_version":1,"previous_command":"printf theirs"}`)
	marker := shInsideTheTree(t, repo)

	stdout, stderr, err := runSplit(t, `{"model":{"display_name":"Opus"}}`, "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("the sh inside the working tree ran (%v)", err)
	}
	if stdout != "theirs" {
		t.Errorf("stdout = %q, want the previous command's output", stdout)
	}
}

// TestTheReportEditorRunsUnderTheSystemShell: the reporter's editor setting
// runs through /bin/sh, so an sh first on PATH inside the working tree is
// never run and the report the editor wrote is filed.
func TestTheReportEditorRunsUnderTheSystemShell(t *testing.T) {
	repo, home := gitRepoNoStore(t)
	t.Chdir(repo)
	skeleton := string(runCLI(t, "report", "--template"))
	filledPath := filepath.Join(t.TempDir(), "filled.md")
	if err := os.WriteFile(filledPath, []byte(fillTemplate(t, skeleton, "edited in place", "Written in the editor.")), 0o600); err != nil {
		t.Fatal(err)
	}
	editor := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\ncat '"+filledPath+"' > \"$1\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", editor)
	prev := reportInteractive
	reportInteractive = func() bool { return true }
	t.Cleanup(func() { reportInteractive = prev })
	marker := shInsideTheTree(t, repo)

	out, err := runCLIErr(t, "report")
	if _, serr := os.Stat(marker); !os.IsNotExist(serr) {
		t.Fatalf("the sh inside the working tree ran (%v)", serr)
	}
	if err != nil || !strings.Contains(string(out), "filed rpt-") {
		t.Fatalf("report = %q, %v; want it filed", out, err)
	}
	if files := inboxFiles(t, home); len(files) != 1 {
		t.Errorf("inbox holds %q, want one report", files)
	}
}
