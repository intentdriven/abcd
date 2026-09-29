package fsutil_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DisplayPath is the display rule for a directory a surface names: under HOME
// the home-relative form, outside HOME the base name, never the absolute path
// (iss-2609281329007423).
func TestDisplayPathNamesADirectoryOutsideHomeByItsBaseName(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	outside := filepath.Join(base, "elsewhere", "wt-a")
	for in, want := range map[string]string{
		filepath.Join(home, "wt", "a"):       filepath.Join("~", "wt", "a"),
		home:                                 "~",
		outside:                              "wt-a",
		outside + string(filepath.Separator): "wt-a",
		"relative/dir":                       "relative/dir",
		"":                                   "",
	} {
		if got := fsutil.DisplayPath(in); got != want {
			t.Errorf("DisplayPath(%q) = %q, want %q", in, got, want)
		}
	}
	if real, err := filepath.EvalSymlinks(home); err == nil && real != home {
		if got := fsutil.DisplayPath(filepath.Join(real, "x")); got != filepath.Join("~", "x") {
			t.Errorf("DisplayPath of the resolved home spelling = %q, want ~/x", got)
		}
	}
}

// DisplayPathsIn applies the rule inside a message: the named path outside HOME
// becomes its base name wherever it starts a path, in either spelling, a longer
// path under it keeps its tail, and the same bytes inside a longer unrelated
// path are left alone.
func TestDisplayPathsInReducesANamedPathOutsideHome(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	realDir := filepath.Join(base, "real", "wt")
	for _, d := range []string{home, realDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("HOME", home)
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}

	msg := "stat " + link + "/.git: denied; also " + resolved + " and /srv" + link + " and " + filepath.Join(home, "y")
	got := fsutil.DisplayPathsIn(msg, link, "relative")
	want := "stat link/.git: denied; also link and /srv" + link + " and " + filepath.Join("~", "y")
	if got != want {
		t.Fatalf("DisplayPathsIn =\n  %q\nwant\n  %q", got, want)
	}
}

// inlineBaseNameDisplayRe matches the display rule restated inline: an
// IsAbs test whose block's first statement takes filepath.Base — the shape
// ledgerIdentityOf and scrubPaths each carried before DisplayPath existed.
var inlineBaseNameDisplayRe = regexp.MustCompile(`filepath\.IsAbs\([^)]*\)\s*\{\s*[^\n]*filepath\.Base\(`)

// TestNoInlineBaseNameDisplayRule is the one-canonical-primitive detector for
// the display rule: no non-test file under internal/ outside fsutil restates
// "absolute, so print the base name" inline. A surface that names a directory
// routes through fsutil.DisplayPath or DisplayPathsIn, so the rule has one
// statement and a new surface cannot get it half right (iss-2609281329007423).
func TestNoInlineBaseNameDisplayRule(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir("..", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == "fsutil" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if inlineBaseNameDisplayRe.Match(data) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("the base-name display rule restated inline (route through fsutil.DisplayPath / DisplayPathsIn):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
