package fsutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteRebuildableAtLeavesTheFileWhole: the content, mode and time all arrive
// with the name, over an old file and through a directory that does not exist
// yet, with no temporary file left, a symlink at the path replaced rather than
// written through, and a link out of the root refused on the way to the path.
func TestWriteRebuildableAtLeavesTheFileWhole(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	path := filepath.Join(dir, "a", "b", "file")
	at := time.Date(2003, 4, 5, 6, 7, 8, 0, time.UTC)
	for _, body := range []string{"first", "second"} {
		if err := WriteRebuildableAtInRoot(root, "a/b/file", []byte(body), 0o755, at); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != body {
			t.Errorf("the file holds %q, want %q", raw, body)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("mode %o, want 755", info.Mode().Perm())
		}
		if !info.ModTime().Equal(at) {
			t.Errorf("time %s, want %s", info.ModTime(), at)
		}
	}
	items, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("the directory holds %d entries, want the one file: %v", len(items), items)
	}

	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteRebuildableAtInRoot(root, "link", []byte("new"), 0o644, at); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(target); string(raw) != "kept" {
		t.Errorf("the write went through the link: %q", raw)
	}
	if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink != 0 {
		t.Error("the link was not replaced")
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "out")); err != nil {
		t.Fatal(err)
	}
	if err := WriteRebuildableAtInRoot(root, "out/escaped", []byte("x"), 0o644, at); err == nil {
		t.Error("a write through a link out of the root was taken")
	}
	if items, _ := os.ReadDir(outside); len(items) != 0 {
		t.Errorf("the write escaped the root: %v", items)
	}
}
