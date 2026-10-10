package ahoy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestDetectNamesAnIgnoredFilterRootsFile is iss-2610091920437492: a
// ~/.abcd.noindex/filter-roots file that fails its ownership, mode or symlink
// checks is ignored, which switches the content filters of every checkout it
// lists off, and nothing said so. ahoy reports it from any folder, naming the
// file and the check it failed; an absent or honoured file reports nothing.
func TestDetectNamesAnIgnoredFilterRootsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes and symlinks")
	}
	find := func(gaps []Gap) *Gap {
		for i := range gaps {
			if gaps[i].ID == FilterRootsIgnoredGapID {
				return &gaps[i]
			}
		}
		return nil
	}
	write := func(t *testing.T, home string, mode os.FileMode) string {
		t.Helper()
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		p := abcdhome.Path(home, "filter-roots")
		if err := os.WriteFile(p, []byte("/some/checkout\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("group-writable", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		write(t, home, 0o620)
		res, err := Detect(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		g := find(res.Gaps)
		if g == nil {
			t.Fatalf("a group-writable filter-roots file must be reported, gaps: %+v", res.Gaps)
		}
		if g.Scope != "machine" || g.Required || g.Resolvable {
			t.Fatalf("the report is a machine-scope note, never a repair: %+v", g)
		}
		if !strings.Contains(g.Detail, abcdhome.Display("filter-roots")) || !strings.Contains(g.Detail, "writable by others") {
			t.Fatalf("the report must name the file and the check it failed: %q", g.Detail)
		}
	})

	t.Run("symlinked", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		real := filepath.Join(t.TempDir(), "roots")
		if err := os.WriteFile(real, []byte("/some/checkout\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, abcdhome.Path(home, "filter-roots")); err != nil {
			t.Fatal(err)
		}
		g := find(detectFilterRoots())
		if g == nil {
			t.Fatal("a symlinked filter-roots file must be reported")
		}
		if !strings.Contains(g.Detail, abcdhome.Display("filter-roots")) || !strings.Contains(g.Detail, "not a regular file") {
			t.Fatalf("the report must name the file and the check it failed: %q", g.Detail)
		}
	})

	t.Run("honoured or absent", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if g := find(detectFilterRoots()); g != nil {
			t.Fatalf("an absent file reports nothing: %+v", g)
		}
		write(t, home, 0o600)
		if g := find(detectFilterRoots()); g != nil {
			t.Fatalf("an honoured file reports nothing: %+v", g)
		}
	})
}
