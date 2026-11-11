package gitutil_test

import (
	"context"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestEmptyTreeIsNamedForTheObjectFormatAndWritten names the empty tree under
// both object formats git ships, and holds that the object then exists in the
// store, so a commit made on it can be pushed.
func TestEmptyTreeIsNamedForTheObjectFormatAndWritten(t *testing.T) {
	sha1 := gittest.NewRepo(t)
	got, err := gitutil.EmptyTree(sha1.Root())
	if err != nil || got != "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
		t.Fatalf("EmptyTree under SHA-1 = %q, %v", got, err)
	}
	if typ := sha1.Git("cat-file", "-t", got); typ != "tree" {
		t.Fatalf("the empty tree's object type = %q; want it written as a tree", typ)
	}

	dir := filepath.Join(t.TempDir(), "sha256")
	sha1.Git("init", "-q", "--object-format=sha256", dir)
	got, err = gitutil.EmptyTree(dir)
	if err != nil || got != "6ef19b41225c5369f1c104d45d8d85efa9b057b53b14b4b9b939dd74decc5321" {
		t.Fatalf("EmptyTree under SHA-256 = %q, %v", got, err)
	}
	if typ := sha1.Git("-C", dir, "cat-file", "-t", got); typ != "tree" {
		t.Fatalf("the SHA-256 empty tree's object type = %q; want it written as a tree", typ)
	}
}

// TestEmptyTreeNameNamesTheEmptyTreeAndWritesNothing holds that the read form
// gives EmptyTree's answer under both object formats and leaves the object
// store as it found it, in a repository whose store does not yet hold the
// empty tree: no new loose object and no file's time changed. EmptyTree run
// afterwards on the same store must change it, which shows the comparison
// would catch a write.
func TestEmptyTreeNameNamesTheEmptyTreeAndWritesNothing(t *testing.T) {
	for _, format := range []string{"sha1", "sha256"} {
		t.Run(format, func(t *testing.T) {
			r := gittest.NewRepoFormat(t, format)
			objects := filepath.Join(r.Root(), ".git", "objects")
			before := agedObjectStore(t, objects)
			got, err := gitutil.EmptyTreeName(r.Root())
			if err != nil {
				t.Fatalf("EmptyTreeName = %v", err)
			}
			if after := objectStore(t, objects); !maps.Equal(before, after) {
				t.Fatalf("EmptyTreeName changed the object store:\nbefore %v\nafter  %v", before, after)
			}
			want, err := gitutil.EmptyTree(r.Root())
			if err != nil || got != want {
				t.Fatalf("EmptyTreeName = %q; EmptyTree = %q, %v; want the same name", got, want, err)
			}
			if after := objectStore(t, objects); maps.Equal(before, after) {
				t.Fatal("EmptyTree left the object store unchanged, so the comparison above proves nothing")
			}
		})
	}
}

// agedObjectStore sets every file under the object store at dir an hour into
// the past and answers objectStore, so a write or a refresh of any of them
// afterwards shows as a changed time.
func agedObjectStore(t *testing.T, dir string) map[string]time.Time {
	t.Helper()
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	for path := range objectStore(t, dir) {
		if err := os.Chtimes(path, past, past); err != nil {
			t.Fatal(err)
		}
	}
	return objectStore(t, dir)
}

// objectStore is every file under the object store at dir with its
// modification time.
func objectStore(t *testing.T, dir string) map[string]time.Time {
	t.Helper()
	files := map[string]time.Time{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files[path] = info.ModTime()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestBothEmptyTreeNameFormsRefuseAnAnswerPastTheCap: a git that answers the
// empty tree's name with more than one object name could hold is refused
// whole, in the same words, by EmptyTreeName and EmptyTreeNameContext: the
// two forms share one cap and one body, and differ in the context alone.
func TestBothEmptyTreeNameFormsRefuseAnAnswerPastTheCap(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("c0")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	long := strings.Repeat("a", 300)
	shim := "#!/bin/sh\nfor a in \"$@\"; do\n  if [ \"$a\" = hash-object ]; then printf '%s\\n' '" + long + "'; exit 0; fi\ndone\n" +
		"exec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	free, ferr := gitutil.EmptyTreeName(r.Root())
	bound, berr := gitutil.EmptyTreeNameContext(context.Background(), r.Root())
	if ferr == nil || berr == nil || free != "" || bound != "" {
		t.Fatalf("an answer past the cap: EmptyTreeName = %q, %v; EmptyTreeNameContext = %q, %v; want both refused", free, ferr, bound, berr)
	}
	if ferr.Error() != berr.Error() || !strings.Contains(ferr.Error(), "output exceeded the") {
		t.Errorf("the two forms refused an answer past the cap differently:\n  EmptyTreeName: %v\n  EmptyTreeNameContext: %v", ferr, berr)
	}
}
