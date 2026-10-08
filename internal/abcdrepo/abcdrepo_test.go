package abcdrepo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLooksLikeSourceTreeKeysOnTheEntryPoint: a tree carrying cmd/abcd/main.go
// as a regular file looks like abcd's source; one without it, or with a
// directory in its place, does not. This checkout is one.
func TestLooksLikeSourceTreeKeysOnTheEntryPoint(t *testing.T) {
	if !LooksLikeSourceTree(filepath.Join("..", "..")) {
		t.Error("abcd's own checkout does not look like abcd's source tree")
	}
	bare := t.TempDir()
	if LooksLikeSourceTree(bare) {
		t.Error("an empty tree looks like abcd's source tree")
	}
	dirInPlace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dirInPlace, "cmd", "abcd", "main.go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if LooksLikeSourceTree(dirInPlace) {
		t.Error("a directory named main.go reads as the entry point")
	}
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "cmd", "abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "cmd", "abcd", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !LooksLikeSourceTree(src) {
		t.Error("a tree carrying cmd/abcd/main.go does not look like abcd's source tree")
	}
}
