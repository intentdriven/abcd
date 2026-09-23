package fsutil

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestAppendLineInAppendsOneTerminatedLine is the ok case: two appends leave two
// newline-terminated lines, in order, and the file is created at perm.
func TestAppendLineInAppendsOneTerminatedLine(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, l := range []string{`{"n":1}`, `{"n":2}`} {
		if err := AppendLineIn(root, "d/log.jsonl", []byte(l), 0o600); err != nil {
			t.Fatalf("append %s: %v", l, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "d", "log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"n\":1}\n{\"n\":2}\n" {
		t.Fatalf("content = %q", got)
	}
	fi, err := os.Stat(filepath.Join(dir, "d", "log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o, want 600", fi.Mode().Perm())
	}
}

// TestAppendLineInRefusesAnythingButOneLine is the guard: a payload carrying a
// newline would let one call write two records (or split one), and an empty one
// writes a blank line no reader can attribute. Both are refused, and nothing is
// written.
func TestAppendLineInRefusesAnythingButOneLine(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, bad := range []string{"a\nb", "trailing\n", "", "cr\rline"} {
		err := AppendLineIn(root, "log.jsonl", []byte(bad), 0o600)
		if !errors.Is(err, ErrNotOneLine) {
			t.Errorf("AppendLineIn(%q) = %v, want ErrNotOneLine", bad, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "log.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused append created the file: %v", err)
	}
}

// TestAppendLineInRefusesASymlinkedAncestor proves the containment half: a
// directory swapped for a symlink pointing out of the root is not followed.
func TestAppendLineInRefusesASymlinkedAncestor(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "d")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := AppendLineIn(root, "d/log.jsonl", []byte("x"), 0o600); err == nil {
		t.Fatal("append through a symlinked ancestor succeeded; want a refusal")
	}
	if _, err := os.Stat(filepath.Join(outside, "log.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the append escaped the root: %v", err)
	}
}

// TestAppendLineInTwoWritersNeverInterleave is the property the run log rests
// on: many concurrent writers appending to one file leave every line whole.
func TestAppendLineInTwoWritersNeverInterleave(t *testing.T) {
	dir := t.TempDir()
	const writers, lines = 8, 200
	pad := strings.Repeat("x", 512)
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			// Each writer opens its own Root, as two processes would.
			root, err := os.OpenRoot(dir)
			if err != nil {
				errs <- err
				return
			}
			defer root.Close()
			for i := 0; i < lines; i++ {
				b, _ := json.Marshal(map[string]any{"w": w, "i": i, "pad": pad})
				if err := AppendLineIn(root, "log.jsonl", b, 0o600); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 64<<10)
	seen := map[string]bool{}
	for sc.Scan() {
		var v struct {
			W, I int
			Pad  string
		}
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil || v.Pad != pad {
			t.Fatalf("an interleaved or torn line: %q (%v)", sc.Text(), err)
		}
		seen[fmt.Sprint(v.W, "/", v.I)] = true
	}
	if len(seen) != writers*lines {
		t.Fatalf("lines = %d, want %d", len(seen), writers*lines)
	}
}
