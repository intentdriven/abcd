package tools

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestWalkReadsARouteAsTheSystemDoes holds Walk to every link shape the four
// rounds of iss-2611202034242491 met, and the shapes around them. Each case
// builds its layout under t.TempDir(), which on macOS lies under /var, itself
// a link, so every route also passes a link outside the layout: the expected
// links and directories are named at their real locations and asked for as
// members, and every link and directory Walk reports is checked to be one.
func TestWalkReadsARouteAsTheSystemDoes(t *testing.T) {
	type want struct {
		real    string   // the route's end; "" when err is set
		links   []string // links the route must pass, at their real locations
		dirs    []string // real directories the route must pass through
		clear   string   // a tree no link on the route may lie in
		missing bool
		err     error // a sentinel, or syscall.ENOTDIR
	}
	type layout struct {
		root, real string // the case's folder, as made and resolved
	}
	mk := func(t *testing.T, l layout, rel ...string) string {
		t.Helper()
		p := filepath.Join(append([]string{l.root}, rel...)...)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	file := func(t *testing.T, p string) string {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("not run\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ln := func(t *testing.T, target, at string) string {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, at); err != nil {
			t.Fatal(err)
		}
		return at
	}
	cases := []struct {
		name  string
		build func(t *testing.T, l layout) (string, want)
	}{
		{"a plain file", func(t *testing.T, l layout) (string, want) {
			p := file(t, filepath.Join(l.root, "bin", "go"))
			return p, want{real: filepath.Join(l.real, "bin", "go"), dirs: []string{filepath.Join(l.real, "bin")}}
		}},
		{"a chain of links read as text (round 1)", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "sdk", "go"))
			ln(t, "../sdk/go", filepath.Join(l.root, "middle", "go"))
			p := ln(t, "../middle/go", filepath.Join(l.root, "bin", "go"))
			return p, want{real: filepath.Join(l.real, "sdk", "go"), links: []string{
				filepath.Join(l.real, "bin", "go"), filepath.Join(l.real, "middle", "go")}}
		}},
		{"a repository link reached through an outside directory link (round 2)", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "safe", "go"))
			ln(t, filepath.Join(l.root, "safe", "go"), filepath.Join(l.root, "repo", "bin", "go"))
			alias := ln(t, filepath.Join(l.root, "repo", "bin"), filepath.Join(l.root, "home", "alias"))
			return filepath.Join(alias, "go"), want{real: filepath.Join(l.real, "safe", "go"), links: []string{
				filepath.Join(l.real, "home", "alias"), filepath.Join(l.real, "repo", "bin", "go")}}
		}},
		{"d/../x with d a link elsewhere (round 3)", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "repo", "sub")
			file(t, filepath.Join(l.root, "usr", "real"))
			ln(t, filepath.Join(l.root, "usr", "real"), filepath.Join(l.root, "repo", "x"))
			ln(t, filepath.Join(l.root, "repo", "sub"), filepath.Join(l.root, "out", "d"))
			return ln(t, "d/../x", filepath.Join(l.root, "out", "tool")), want{err: ErrReadOtherwise}
		}},
		{"d/../x in the path itself", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "other", "sub")
			file(t, filepath.Join(l.root, "other", "x"))
			ln(t, filepath.Join(l.root, "other", "sub"), filepath.Join(l.root, "bin", "d"))
			return filepath.Join(l.root, "bin", "d") + "/../x", want{err: ErrReadOtherwise}
		}},
		{"d/../x with d a real directory", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "out", "d")
			file(t, filepath.Join(l.root, "usr", "real"))
			ln(t, filepath.Join(l.root, "usr", "real"), filepath.Join(l.root, "out", "x"))
			p := ln(t, "d/../x", filepath.Join(l.root, "out", "tool"))
			return p, want{real: filepath.Join(l.real, "usr", "real"), links: []string{
				filepath.Join(l.real, "out", "tool"), filepath.Join(l.real, "out", "x")},
				dirs: []string{filepath.Join(l.real, "out", "d")}}
		}},
		{"d/../x with d a link to a folder beside it", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "out", "sub")
			file(t, filepath.Join(l.root, "out", "x"))
			ln(t, "sub", filepath.Join(l.root, "out", "d"))
			p := ln(t, "d/../x", filepath.Join(l.root, "out", "tool"))
			return p, want{real: filepath.Join(l.real, "out", "x"), links: []string{filepath.Join(l.real, "out", "d")}}
		}},
		{"a committed directory link in the middle of the folder path (round 4)", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "opt", "tools", "bin", "gitleaks"))
			ln(t, filepath.Join(l.root, "opt", "tools", "bin"), filepath.Join(l.root, "repo", "scripts"))
			alias := ln(t, filepath.Join(l.root, "repo", "scripts"), filepath.Join(l.root, "home", "alias"))
			return filepath.Join(alias, "gitleaks"), want{real: filepath.Join(l.real, "opt", "tools", "bin", "gitleaks"),
				links: []string{filepath.Join(l.real, "home", "alias"), filepath.Join(l.real, "repo", "scripts")},
				dirs:  []string{filepath.Join(l.real, "repo"), filepath.Join(l.real, "opt", "tools", "bin")}}
		}},
		{"a directory link deep in a written path", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "store", "v1", "bin", "go"))
			ln(t, filepath.Join(l.root, "store", "v1"), filepath.Join(l.root, "a", "mid"))
			return filepath.Join(l.root, "a", "mid", "bin", "go"), want{real: filepath.Join(l.real, "store", "v1", "bin", "go"),
				links: []string{filepath.Join(l.real, "a", "mid")}, dirs: []string{filepath.Join(l.real, "store", "v1", "bin")}}
		}},
		{"nested links: a link whose target passes another directory link", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "deep", "inner", "go"))
			ln(t, "deep", filepath.Join(l.root, "x"))
			ln(t, "x/inner", filepath.Join(l.root, "top"))
			p := ln(t, "../top/go", filepath.Join(l.root, "bin", "go"))
			return p, want{real: filepath.Join(l.real, "deep", "inner", "go"), links: []string{
				filepath.Join(l.real, "bin", "go"), filepath.Join(l.real, "top"), filepath.Join(l.real, "x")}}
		}},
		{"a relative .. across a directory link another text wrote (merged /usr)", func(t *testing.T, l layout) (string, want) {
			file(t, filepath.Join(l.root, "usr", "lib", "foo", "foo"))
			ln(t, "../lib/foo/foo", filepath.Join(l.root, "usr", "bin", "foo"))
			ln(t, "usr/bin", filepath.Join(l.root, "bin"))
			return filepath.Join(l.root, "bin", "foo"), want{real: filepath.Join(l.real, "usr", "lib", "foo", "foo"),
				links: []string{filepath.Join(l.real, "bin"), filepath.Join(l.real, "usr", "bin", "foo")},
				dirs:  []string{filepath.Join(l.real, "usr", "bin"), filepath.Join(l.real, "usr", "lib", "foo")}}
		}},
		{"a loop", func(t *testing.T, l layout) (string, want) {
			a := filepath.Join(l.root, "bin", "go")
			ln(t, filepath.Join(l.root, "go"), a)
			ln(t, a, filepath.Join(l.root, "go"))
			return a, want{err: ErrLinksDoNotEnd}
		}},
		{"a dangling final link", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "gone")
			p := ln(t, filepath.Join(l.root, "gone", "go"), filepath.Join(l.root, "bin", "go"))
			return p, want{real: filepath.Join(l.real, "gone", "go"), missing: true, links: []string{filepath.Join(l.real, "bin", "go")}}
		}},
		{"a folder not made yet behind a directory link", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "data")
			ln(t, filepath.Join(l.root, "data"), filepath.Join(l.root, "home", "local"))
			return filepath.Join(l.root, "home", "local", "bin", "go"), want{real: filepath.Join(l.real, "data", "bin", "go"), missing: true,
				links: []string{filepath.Join(l.real, "home", "local")}}
		}},
		{"a part under a file", func(t *testing.T, l layout) (string, want) {
			f := file(t, filepath.Join(l.root, "file"))
			return filepath.Join(f, "bin", "go"), want{err: syscall.ENOTDIR}
		}},
		{"a .. after a file", func(t *testing.T, l layout) (string, want) {
			f := file(t, filepath.Join(l.root, "file"))
			file(t, filepath.Join(l.root, "go"))
			return f + "/../go", want{err: syscall.ENOTDIR}
		}},
		{"an outside directory link to nothing guarded", func(t *testing.T, l layout) (string, want) {
			mk(t, l, "repo")
			file(t, filepath.Join(l.root, "opt", "bin", "go"))
			alias := ln(t, filepath.Join(l.root, "opt", "bin"), filepath.Join(l.root, "home", "alias"))
			return filepath.Join(alias, "go"), want{real: filepath.Join(l.real, "opt", "bin", "go"),
				links: []string{filepath.Join(l.real, "home", "alias")}, clear: filepath.Join(l.real, "repo")}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			real, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			p, w := tc.build(t, layout{root: root, real: real})
			route, err := Walk(p)
			if w.err != nil {
				if !errors.Is(err, w.err) {
					t.Fatalf("Walk(%q) = %+v, %v; want %v", p, route, err, w.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Walk(%q): %v", p, err)
			}
			if route.Real != w.real || route.Missing != w.missing {
				t.Fatalf("Walk(%q) ends at %q (missing %v); want %q (missing %v)", p, route.Real, route.Missing, w.real, w.missing)
			}
			for _, l := range w.links {
				if !slices.Contains(route.Links, l) {
					t.Errorf("links = %q; want %q among them", route.Links, l)
				}
			}
			for _, d := range w.dirs {
				if !slices.Contains(route.Dirs, d) {
					t.Errorf("dirs = %q; want %q among them", route.Dirs, d)
				}
			}
			for _, l := range route.Links {
				if fi, err := os.Lstat(l); err != nil || fi.Mode()&os.ModeSymlink == 0 {
					t.Errorf("%q is reported as a link on the route, but is not one", l)
				}
				if dir, err := filepath.EvalSymlinks(filepath.Dir(l)); err != nil || dir != filepath.Dir(l) {
					t.Errorf("%q is not named at its real location", l)
				}
				if w.clear != "" && fsutil.PathWithin(l, w.clear, false) {
					t.Errorf("%q lies in %q, which the route does not pass", l, w.clear)
				}
			}
			for _, d := range route.Dirs {
				if r, err := filepath.EvalSymlinks(d); err != nil || r != d {
					t.Errorf("%q is reported as a real directory, but resolves to %q (%v)", d, r, err)
				}
			}
		})
	}
	if _, err := Walk("bin/go"); err == nil {
		t.Error("Walk accepted a relative path")
	}
}
