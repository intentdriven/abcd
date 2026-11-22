package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFilesKeepsASeedAndRefusesDriftedMachinery: a seed file the
// repository already has is its own and is kept whatever it holds, while a
// machinery file that drifted refuses the whole run, and the refusal writes
// nothing — not even the absent seed beside it.
func TestWriteFilesKeepsASeedAndRefusesDriftedMachinery(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "a"), 0o755))
	must(os.WriteFile(filepath.Join(root, "a", "seed.json"), []byte("the repository's own\n"), 0o644))

	out, wrote, refused, err := WriteFiles(root, []PlannedFile{
		{Path: "a/seed.json", Data: []byte("abcd's default\n"), Seed: true},
		{Path: "a/new-seed.json", Data: []byte("new\n"), Seed: true},
		{Path: "b/machine.yml", Data: []byte("machinery\n")},
	}, false)
	if err != nil || wrote != 2 || refused != 0 {
		t.Fatalf("first run: wrote %d refused %d err %v", wrote, refused, err)
	}
	if out[0].Status != StatusKept {
		t.Fatalf("a present seed is %q, want kept", out[0].Status)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "a", "seed.json")); string(got) != "the repository's own\n" {
		t.Fatalf("the present seed was rewritten: %q", got)
	}

	must(os.WriteFile(filepath.Join(root, "b", "machine.yml"), []byte("hand-edited\n"), 0o644))
	must(os.Remove(filepath.Join(root, "a", "new-seed.json")))
	out, wrote, refused, err = WriteFiles(root, []PlannedFile{
		{Path: "a/new-seed.json", Data: []byte("new\n"), Seed: true},
		{Path: "b/machine.yml", Data: []byte("machinery\n")},
	}, false)
	if !errors.Is(err, ErrScaffoldBlocked) || wrote != 0 || refused != 1 {
		t.Fatalf("drifted machinery: wrote %d refused %d err %v", wrote, refused, err)
	}
	if _, serr := os.Stat(filepath.Join(root, "a", "new-seed.json")); !os.IsNotExist(serr) {
		t.Fatal("a refused run wrote the absent seed")
	}
	if out[1].Status != StatusRefused {
		t.Fatalf("drifted machinery is %q, want refused", out[1].Status)
	}
}

// TestWriteFilesKeepsEveryWriteInsideTheRepository: a planned file whose
// parent directory is a symlink to a directory outside the repository is not
// created there, with or without confirm, and the run reports it as not
// written. A planned file under a directory that does not exist yet is still
// created inside the repository.
func TestWriteFilesKeepsEveryWriteInsideTheRepository(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "clone")
	outside := filepath.Join(parent, "outside")
	mustMkdir(t, root)
	mustMkdir(t, outside)
	if err := os.Symlink("../outside", filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, confirm := range []bool{false, true} {
		out, wrote, _, err := WriteFiles(root, []PlannedFile{
			{Path: "linked/workflows/machine.yml", Data: []byte("machinery\n")},
			{Path: "fresh/machine.yml", Data: []byte("machinery\n")},
		}, confirm)
		if err == nil {
			t.Fatalf("confirm=%v: a planned file behind a linked directory was accepted", confirm)
		}
		if out[0].Status == StatusWritten || wrote != 0 {
			t.Fatalf("confirm=%v: report %+v wrote %d, want nothing written", confirm, out, wrote)
		}
		if entries, _ := os.ReadDir(outside); len(entries) != 0 {
			t.Fatalf("confirm=%v: the run created %d entries outside the repository", confirm, len(entries))
		}
	}

	out, wrote, _, err := WriteFiles(root, []PlannedFile{{Path: "fresh/machine.yml", Data: []byte("machinery\n")}}, false)
	if err != nil || wrote != 1 || out[0].Status != StatusWritten {
		t.Fatalf("a new directory inside the repository: %+v wrote %d err %v", out, wrote, err)
	}
	if got := readFile(t, filepath.Join(root, "fresh", "machine.yml")); got != "machinery\n" {
		t.Fatalf("the file inside the repository holds %q", got)
	}
}

// TestAScaffoldWritesNothingThroughALinkedGithubDirectory: a checkout whose
// .github is a symlink to an existing directory outside it receives no
// workflow in that directory, with or without --confirm, and the run returns
// an error.
func TestAScaffoldWritesNothingThroughALinkedGithubDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "clone")
	outside := filepath.Join(parent, "outside")
	mustMkdir(t, outside)
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
	declarePlugin(t, dir)
	gitInit(t, dir, "main")
	if err := os.Symlink("../outside", filepath.Join(dir, ".github")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, confirm := range []bool{false, true} {
		rep, err := Scaffold(Request{RepoRoot: dir, Confirm: confirm})
		if err == nil {
			t.Fatalf("confirm=%v: the scaffold succeeded through a linked .github: %+v", confirm, rep.Files)
		}
		if entries, _ := os.ReadDir(outside); len(entries) != 0 {
			t.Fatalf("confirm=%v: the scaffold created %d entries outside the checkout", confirm, len(entries))
		}
	}
}

// TestTheCheckNamesAreNotReadThroughALinkedGithubDirectory: the workflows of a
// directory outside the checkout, reached through a linked .github, supply no
// check name; the workflows directory is reported as one that cannot be listed.
func TestTheCheckNamesAreNotReadThroughALinkedGithubDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "clone")
	outside := filepath.Join(parent, "outside")
	mustMkdir(t, dir)
	mustWrite(t, filepath.Join(outside, "workflows", "ci.yml"), "on: pull_request\njobs:\n  elsewhere:\n    runs-on: x\n")
	if err := os.Symlink("../outside", filepath.Join(dir, ".github")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	checks, unread := DeriveCIChecks(dir)
	if len(checks) != 0 {
		t.Fatalf("check names %q were read from outside the checkout", checks)
	}
	if len(unread) != 1 || unread[0] != workflowsDir {
		t.Fatalf("unread = %q, want the workflows directory named", unread)
	}
}
