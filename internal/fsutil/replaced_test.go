//go:build unix

package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// swapIn renames staged over dst. rename(2) cannot replace a file with a
// directory, so a directory is moved in after the file is removed; every other
// kind replaces it atomically, as a real rename-swap would.
func swapIn(t *testing.T, staged, dst string) {
	t.Helper()
	if fi, err := os.Lstat(staged); err == nil && fi.IsDir() {
		if err := os.Remove(dst); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}
	if err := os.Rename(staged, dst); err != nil {
		t.Fatalf("swap: %v", err)
	}
}

// The benign replacement: another abcd process of the same user rewrites the
// declaration through WriteFileAtomic — a temp file renamed over it — inside
// the window between this read's vetting lstat and its open. The new file
// passes every guard the old one passed, so it is the caller's word as much as
// the old one was; refusing it made one of two concurrent abcd processes
// refuse its own configuration (iss-2609290518278152). It is re-vetted and
// read, and the bytes are the new file's.
func TestReadDeclarationReadsABenignReplacementAfterRevetting(t *testing.T) {
	dir := t.TempDir()
	path := writeDeclaration(t, dir, "decl", "before\n")
	prev := declarationVetted
	t.Cleanup(func() { declarationVetted = prev })
	calls := 0
	declarationVetted = func(p string) {
		calls++
		if calls == 1 {
			if err := WriteFileAtomic(p, []byte("after\n"), 0o600); err != nil {
				t.Fatalf("replace: %v", err)
			}
		}
	}
	raw, refusal, err := ReadDeclaration(path, 1024)
	if err != nil || refusal != DeclarationOK {
		t.Fatalf("a same-owner regular file renamed into place must be re-vetted and read: refusal %d, err %v", refusal, err)
	}
	if string(raw) != "after\n" {
		t.Fatalf("read %q, want the replacement's bytes", raw)
	}
	if calls != 2 {
		t.Fatalf("the replacement must be vetted before it is read: %d vetting(s), want 2", calls)
	}
}

// A replacement that keeps happening is not waited out forever: after
// declarationAttempts vettings that each lost the race the read refuses, named
// as the swap it is.
func TestReadDeclarationRefusesAnEndlessReplacement(t *testing.T) {
	dir := t.TempDir()
	path := writeDeclaration(t, dir, "decl", "before\n")
	prev := declarationVetted
	t.Cleanup(func() { declarationVetted = prev })
	calls := 0
	declarationVetted = func(p string) {
		calls++
		if err := WriteFileAtomic(p, []byte("again\n"), 0o600); err != nil {
			t.Fatalf("replace: %v", err)
		}
	}
	raw, refusal, err := ReadDeclaration(path, 1024)
	if raw != nil || refusal != DeclarationUnreadable || !errors.Is(err, ErrDeclarationSwapped) {
		t.Fatalf("an endless replacement must be refused as a swap: raw %q, refusal %d, err %v", raw, refusal, err)
	}
	if calls != declarationAttempts {
		t.Fatalf("%d vetting(s), want the bound %d", calls, declarationAttempts)
	}
}

// What the re-vetting still refuses: a replacement that is not a same-owner,
// owner-only-writable regular file. Each is renamed into place once, after the
// first vetting, so the only thing that can refuse it is the judgement of the
// replacement itself — the retry must never promote it into a read.
func TestReadDeclarationRefusesANonBenignReplacement(t *testing.T) {
	cases := []struct {
		name string
		// plant creates the replacement at dst (a sibling of the declaration).
		plant func(t *testing.T, dir, dst string)
		// foreign makes the owner lookup report another uid once swapped.
		foreign bool
		want    DeclarationRefusal
	}{
		{name: "symlink to an owned file", plant: func(t *testing.T, dir, dst string) {
			target := writeDeclaration(t, dir, "target", "linked\n")
			if err := os.Symlink(target, dst); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "fifo", plant: func(t *testing.T, _, dst string) {
			if err := syscall.Mkfifo(dst, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "directory", plant: func(t *testing.T, _, dst string) {
			if err := os.Mkdir(dst, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "group-writable file", want: DeclarationWritableByOthers, plant: func(t *testing.T, dir, dst string) {
			writeDeclaration(t, dir, filepath.Base(dst), "writable\n")
			if err := os.Chmod(dst, 0o664); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "foreign-owned file", foreign: true, want: DeclarationForeignOwner, plant: func(t *testing.T, dir, dst string) {
			writeDeclaration(t, dir, filepath.Base(dst), "foreign\n")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeDeclaration(t, dir, "decl", "vetted\n")
			staged := filepath.Join(dir, "staged")
			tc.plant(t, dir, staged)
			swapped := false
			restore := SwapOwnerUIDForTest(func(p string) (uint32, error) {
				if tc.foreign && swapped {
					return uint32(os.Getuid()) + 1, nil
				}
				return OwnerUID(p)
			})
			t.Cleanup(restore)
			prev := declarationVetted
			t.Cleanup(func() { declarationVetted = prev })
			declarationVetted = func(p string) {
				if swapped {
					return
				}
				swapped = true
				swapIn(t, staged, p)
			}
			raw, refusal, err := ReadDeclaration(path, 1024)
			if refusal == DeclarationOK || err == nil || raw != nil {
				t.Fatalf("a %s swapped in after vetting must be refused: raw %q, refusal %d, err %v", tc.name, raw, refusal, err)
			}
			if tc.want != 0 && refusal != tc.want {
				t.Fatalf("a %s must be refused by the guard that judges it: refusal %d, want %d (err %v)", tc.name, refusal, tc.want, err)
			}
		})
	}
}

// ReadGuardedInRoot closes the same lstat->open window, for files inside a
// checkout, and loses the same race to the same benign rewrite
// (WriteFileAtomicInRoot by a concurrent abcd). The replacement is re-vetted
// and read.
func TestReadGuardedInRootReadsABenignReplacementAfterRevetting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.json"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := openFixtureRoot(t, dir)
	prev := inRootVetted
	t.Cleanup(func() { inRootVetted = prev })
	calls := 0
	inRootVetted = func(root *os.Root, rel string) {
		calls++
		if calls == 1 {
			if err := WriteFileAtomicInRoot(root, rel, []byte("after"), 0o644); err != nil {
				t.Fatalf("replace: %v", err)
			}
		}
	}
	got, err := ReadGuardedInRoot(r, "f.json", 1<<20)
	if err != nil {
		t.Fatalf("a regular file renamed into place must be re-vetted and read: %v", err)
	}
	if string(got) != "after" || calls != 2 {
		t.Fatalf("read %q after %d vetting(s), want the replacement's bytes after 2", got, calls)
	}
}

// What ReadGuardedInRoot's re-vetting still refuses: a leaf that is not a
// regular file, swapped in after the first vetting, and a replacement that
// never stops.
func TestReadGuardedInRootRefusesANonBenignReplacement(t *testing.T) {
	cases := map[string]func(t *testing.T, dir, dst string){
		"symlink to a contained file": func(t *testing.T, dir, dst string) {
			if err := os.WriteFile(filepath.Join(dir, "target"), []byte("linked"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("target", dst); err != nil {
				t.Fatal(err)
			}
		},
		"fifo": func(t *testing.T, _, dst string) {
			if err := syscall.Mkfifo(dst, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"directory": func(t *testing.T, _, dst string) {
			if err := os.Mkdir(dst, 0o700); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, plant := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "f.json"), []byte("vetted"), 0o644); err != nil {
				t.Fatal(err)
			}
			staged := filepath.Join(dir, "staged")
			plant(t, dir, staged)
			r := openFixtureRoot(t, dir)
			prev := inRootVetted
			t.Cleanup(func() { inRootVetted = prev })
			swapped := false
			inRootVetted = func(_ *os.Root, _ string) {
				if swapped {
					return
				}
				swapped = true
				swapIn(t, staged, filepath.Join(dir, "f.json"))
			}
			got, err := ReadGuardedInRoot(r, "f.json", 1<<20)
			if !errors.Is(err, ErrNotRegular) || got != nil {
				t.Fatalf("a %s swapped in after vetting must be refused as not regular: read %q, err %v", name, got, err)
			}
		})
	}
	t.Run("endless replacement", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "f.json"), []byte("vetted"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := openFixtureRoot(t, dir)
		prev := inRootVetted
		t.Cleanup(func() { inRootVetted = prev })
		calls := 0
		inRootVetted = func(root *os.Root, rel string) {
			calls++
			if err := WriteFileAtomicInRoot(root, rel, []byte("again"), 0o644); err != nil {
				t.Fatalf("replace: %v", err)
			}
		}
		got, err := ReadGuardedInRoot(r, "f.json", 1<<20)
		if !errors.Is(err, ErrNotRegular) || got != nil {
			t.Fatalf("an endless replacement must be refused: read %q, err %v", got, err)
		}
		if calls != declarationAttempts {
			t.Fatalf("%d vetting(s), want the bound %d", calls, declarationAttempts)
		}
	})
}
