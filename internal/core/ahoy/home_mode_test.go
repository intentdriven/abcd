package ahoy

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestHomeWritersMakeTheHomePrivate is iss-2610032205304585 at the writers
// that made abcd's home readable by other accounts: whichever of them creates
// the home first, the home is the account's alone and the record it writes is
// read and written by the account alone.
func TestHomeWritersMakeTheHomePrivate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		write  func() error
		record []string
	}{
		{"the path-entry record", func() error { return writePathEntry("/usr/local/bin/abcd", "ab", "") }, []string{"path-entry"}},
		{"the history registry", func() error { _, err := bootstrapHistory(); return err }, []string{"history", "index.json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			if err := tc.write(); err != nil {
				t.Fatal(err)
			}
			fi, err := os.Lstat(abcdhome.Path(home))
			if err != nil {
				t.Fatal(err)
			}
			if got := fi.Mode().Perm(); got != abcdhome.DirMode {
				t.Errorf("%s created %s at %o, want %o", tc.name, abcdhome.Display(), got, abcdhome.DirMode)
			}
			fi, err = os.Lstat(abcdhome.Path(home, tc.record...))
			if err != nil {
				t.Fatal(err)
			}
			if got := fi.Mode().Perm(); got != abcdhome.FileMode {
				t.Errorf("%s wrote %s at %o, want %o", tc.name, abcdhome.Display(tc.record...), got, abcdhome.FileMode)
			}
		})
	}
}

// TestAnEarlierIndexIsNarrowedAtItsNextWrite: a registry index an earlier
// version wrote 0o644 is the account's alone once abcd writes it again; the
// home folder that holds it keeps its mode (iss-2610032205304585).
func TestAnEarlierIndexIsNarrowedAtItsNextWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(abcdhome.Path(home, "history"), 0o755); err != nil {
		t.Fatal(err)
	}
	index := abcdhome.Path(home, "history", "index.json")
	if err := os.WriteFile(index, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(index, 0o644); err != nil {
		t.Fatal(err)
	}
	dir, err := historyDir(true)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if err := writeHistoryIndexIn(dir, &historyIndex{}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(index)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != abcdhome.FileMode {
		t.Errorf("the rewritten index is %o, want %o", got, abcdhome.FileMode)
	}
	fi, err = os.Lstat(abcdhome.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o755 {
		t.Errorf("the existing home was re-moded to %o; a level that exists keeps its mode", got)
	}
}
