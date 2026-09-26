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
