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
