package lifeboat

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestWalkFilesBoundsEntriesReadPerDirectory is the iss-112 property: the walk
// must not materialise a whole directory listing before its file cap applies. A
// single directory holding far more entries than the per-directory bound is read
// only up to that bound — the same guard ListDir already enforces with
// ReadDir(maxDirEntries) — and reaching it is reported as truncation, never
// silent. Without the guard fs.WalkDir calls ReadDir(-1) and one directory of
// millions of entries balloons memory before any file cap fires.
func TestWalkFilesBoundsEntriesReadPerDirectory(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}
	for i := 0; i < 20; i++ {
		files[fmt.Sprintf("f%02d.txt", i)] = "x\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	// A per-directory bound of 5 over a 20-entry directory: at most 5 entries are
	// read from it, and the walk reports it stopped short.
	paths, truncated := ctx.walkFilesBounded(".", 1000, 5)
	if truncated.oversizedCount != 1 || truncated.oversized[0] != "." || truncated.stopped {
		t.Errorf("walk of a 20-entry directory under a 5-entry per-directory bound reported no truncation")
	}
	if len(paths) > 5 {
		t.Errorf("walk returned %d files from a directory read under a 5-entry bound, want <= 5 (full listing materialised)", len(paths))
	}

	// A directory under the per-directory bound is read whole and not truncated.
	small := t.TempDir()
	writeTree(t, small, map[string]string{"a.txt": "x\n", "b.txt": "x\n", "c.txt": "x\n"})
	ctx2, err := newSourceContext(small)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx2.Close()
	paths2, truncated2 := ctx2.walkFilesBounded(".", 1000, 5)
	if truncated2.Any() || len(paths2) != 3 {
		t.Errorf("walk of 3 files under a 5-entry bound = %d files, truncated=%v; want 3 untruncated", len(paths2), truncated2)
	}
}

// TestWalkFilesSkipsBroadDependencyTrees is the iss-116 property: the skip set
// covers the common ecosystems' dependency, cache, and build-output trees — not
// just Node and Go — so none of them is walked as if it were the team's own
// source. Each planted marker sits in a tree that must be skipped; only the real
// src/ file survives the walk.
func TestWalkFilesSkipsBroadDependencyTrees(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"src/keep.go":              "package src\n\n// TODO: keep me\n",
		".venv/lib/dep.py":         "# TODO: python virtualenv\n",
		"venv/lib/dep.py":          "# TODO: bare python virtualenv\n",
		".tox/py311/dep.py":        "# TODO: tox environment\n",
		"pkg/__pycache__/m.pyc":    "# TODO: python bytecode cache\n",
		"target/debug/build.rs":    "// TODO: rust build output\n",
		"dist/bundle.js":           "// TODO: distribution output\n",
		"build/out.o":              "// TODO: build output\n",
		"Pods/Alamofire/net.swift": "// TODO: cocoapods dependency\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	paths, _ := ctx.WalkFiles(".")
	if got := strings.Join(paths, ","); got != "src/keep.go" {
		t.Errorf("WalkFiles = %v, want only [src/keep.go]; skip set is %v", paths, walkSkipDirs)
	}
}

// TestOpenQuestionsAdapterIgnoresVendoredMarkers is the iss-116 first
// consequence: the open-questions adapter must not cite a vendored dependency's
// TODO as this project's own open question. A record-less repo carrying markers
// in dependency trees and one real marker in src/ grounds the section on src/
// alone.
func TestOpenQuestionsAdapterIgnoresVendoredMarkers(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"src/app.go":         "package app\n\n// TODO: wire the retry path\n",
		".venv/lib/dep.py":   "# TODO: not our question\n",
		"__pycache__/m.pyc":  "# FIXME: not our question\n",
		"target/debug/b.rs":  "// XXX: not our question\n",
		"dist/bundle.js":     "// HACK: not our question\n",
		"build/out.o":        "// BUG: not our question\n",
		"Pods/Lib/net.swift": "// TODO: not our question\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convOpenQuestionsSource{}.Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("status = %q, want partial (one real marker in src/)", ev.Status)
	}
	joined := strings.Join(ev.Sources, "\n")
	if !strings.Contains(joined, "src/app.go:") {
		t.Errorf("evidence does not cite the real src/ marker:\n%s", joined)
	}
	for _, vendored := range []string{".venv", "__pycache__", "target", "dist", "build", "Pods"} {
		if strings.Contains(joined, vendored+"/") {
			t.Errorf("evidence cites a vendored marker under %q:\n%s", vendored, joined)
		}
	}
}

// TestWalkFilesDotPrefixedTreeDoesNotEatTheCapBeforeSrc is the iss-116 second
// consequence: because fs.WalkDir reads entries in lexical order, a large
// dot-prefixed dependency tree (.venv sorts before src) could consume the whole
// file cap before the project's own src/ was reached. Skipping the dependency
// tree means the cap is spent on real source. The pre-fix behaviour is
// reproduced through the baseline walk, and killed by the shipped skip set.
func TestWalkFilesDotPrefixedTreeDoesNotEatTheCapBeforeSrc(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"src/real.go": "package src\n\n// TODO: the real question\n"}
	for i := 0; i < 20; i++ {
		files[fmt.Sprintf(".venv/lib/dep%02d.py", i)] = "# TODO: dependency noise\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	// Pre-fix reproduction: the old walk over the old (Node+Go only) skip set
	// walks .venv, and a 5-file cap is exhausted by the lexically-first entries
	// before src/ is reached.
	oldSkip := []string{".git", "generated", "node_modules", "vendor"}
	prePaths, preTrunc := walkFilesRootFSBaseline(ctx, ".", 5, oldSkip)
	if !preTrunc {
		t.Fatalf("baseline pre-fix walk did not truncate under a 5-file cap; fixture too small to reproduce iss-116")
	}
	if containsPath(prePaths, "src/real.go") {
		t.Fatalf("baseline pre-fix walk reached src/real.go; fixture does not reproduce the cap-eating consequence")
	}

	// Post-fix: the shipped skip set excludes .venv, so the cap is never spent on
	// the dependency tree and src/ is reached.
	postPaths, _ := ctx.walkFilesBounded(".", 5, maxDirEntries)
	if !containsPath(postPaths, "src/real.go") {
		t.Errorf("post-fix walk did not reach src/real.go under the cap: %v", postPaths)
	}
	for _, p := range postPaths {
		if strings.HasPrefix(p, ".venv/") {
			t.Errorf("post-fix walk cited a .venv path %q; the dependency tree was walked", p)
		}
	}
}

// TestWalkFilesSubRootContainmentAtDepth strengthens the containment property
// for the sub-root rewrite (iss-114): a symlink that escapes the root must be
// refused even when it sits several directories deep, where the walk holds a
// sub-root per directory rather than resolving every path from the top. The
// os.Root containment guarantee must survive the cost change.
func TestWalkFilesSubRootContainmentAtDepth(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	writeTree(t, repo, map[string]string{
		"a/b/c/inside.go": "package c\n\n// TODO: deep inside\n",
	})
	writeTree(t, outside, map[string]string{"secret.txt": "// TODO: escaped\n"})
	// A symlink three directories deep pointing outside the containment root.
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(repo, "a", "b", "c", "escape.txt")); err != nil {
		t.Fatal(err)
	}
	// A symlinked directory three levels deep pointing outside the root.
	if err := os.Symlink(outside, filepath.Join(repo, "a", "b", "c", "escape-dir")); err != nil {
		t.Fatal(err)
	}

	ctx, err := newSourceContext(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	paths, _ := ctx.WalkFiles(".")
	if got := strings.Join(paths, ","); got != "a/b/c/inside.go" {
		t.Fatalf("WalkFiles = %v, want only [a/b/c/inside.go]; a deep symlink escaped the sub-root containment", paths)
	}
}

// containsPath reports whether paths contains p.
func containsPath(paths []string, p string) bool {
	for _, x := range paths {
		if x == p {
			return true
		}
	}
	return false
}

// pathDepth counts the segments in a cleaned slash path, with "." the zero
// depth — how the baseline walk measured how far below its start it had
// descended. It is test-only: the shipped walk now tracks depth as a passed-down
// counter, so this lives beside the baseline that still needs it.
func pathDepth(p string) int {
	if p == "." {
		return 0
	}
	return strings.Count(p, "/") + 1
}

// walkFilesRootFSBaseline is the pre-fix walk preserved for measurement and for
// reproducing the iss-116 cap-eating consequence: it drives fs.WalkDir over
// c.root.FS(), which re-resolves every path from the containment root one
// component at a time (the O(entries x depth) cost iss-114 removes) and reads
// every directory with ReadDir(-1) (the unbounded per-directory read iss-112
// removes). It is test-only and never shipped.
func walkFilesRootFSBaseline(c *SourceContext, rel string, limit int, skip []string) (paths []string, truncated bool) {
	if c.root == nil {
		return nil, false
	}
	start := path.Clean(filepath.ToSlash(rel))
	if !fs.ValidPath(start) {
		return nil, false
	}
	startDepth := pathDepth(start)
	dirs := 0
	_ = fs.WalkDir(c.root.FS(), start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			for _, s := range skip {
				if path.Base(p) == s {
					return fs.SkipDir
				}
			}
			if dirs >= limit {
				truncated = true
				return fs.SkipAll
			}
			dirs++
			if pathDepth(p)-startDepth >= maxWalkDepth {
				truncated = true
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if len(paths) >= limit {
			truncated = true
			return fs.SkipAll
		}
		paths = append(paths, p)
		return nil
	})
	sort.Strings(paths)
	return paths, truncated
}

// buildDeepTree writes chains directories each depth levels deep under dir, with
// one file at the bottom of each chain, for the iss-114 cost benchmark. It
// returns the total directory count. depth stays under maxWalkDepth so the depth
// cap never prunes the measured tree.
func buildDeepTree(tb testing.TB, dir string, chains, depth int) int {
	tb.Helper()
	total := 0
	for i := 0; i < chains; i++ {
		segs := make([]string, 0, depth+1)
		segs = append(segs, fmt.Sprintf("c%05d", i))
		for j := 0; j < depth-1; j++ {
			segs = append(segs, "d")
		}
		leafDir := filepath.Join(append([]string{dir}, segs...)...)
		if err := os.MkdirAll(leafDir, 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(leafDir, "leaf.go"), []byte("// x\n"), 0o644); err != nil {
			tb.Fatal(err)
		}
		total += depth
	}
	return total
}

// BenchmarkWalkFilesDeepTreeNew measures the shipped sub-root walk over a
// 60000-directory tree at depth 30 — the iss-114 measurement shape.
func BenchmarkWalkFilesDeepTreeNew(b *testing.B) {
	dir := b.TempDir()
	total := buildDeepTree(b, dir, 1600, 30)
	ctx, err := newSourceContext(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer ctx.Close()
	b.Logf("tree: %d directories, depth 30", total)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paths, _ := ctx.WalkFiles(".")
		if len(paths) != 1600 {
			b.Fatalf("walk returned %d leaf files, want 1600", len(paths))
		}
	}
}

// BenchmarkWalkFilesDeepTreeBaseline measures the pre-fix root.FS() walk over the
// same tree, so the iss-114 before/after is a like-for-like number.
func BenchmarkWalkFilesDeepTreeBaseline(b *testing.B) {
	dir := b.TempDir()
	total := buildDeepTree(b, dir, 1600, 30)
	ctx, err := newSourceContext(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer ctx.Close()
	b.Logf("tree: %d directories, depth 30", total)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paths, _ := walkFilesRootFSBaseline(ctx, ".", maxWalkFiles, walkSkipDirs)
		if len(paths) != 1600 {
			b.Fatalf("baseline walk returned %d leaf files, want 1600", len(paths))
		}
	}
}

// TestWalkFilesStartBoundaryMatchesTheWholeWalk is iss-135: a walk that starts
// below the root must agree with the whole-tree walk about the start itself. From
// "." a skip-set directory is never entered, a regular file is yielded, and a
// symlink is never followed; the same three must hold when the start IS such an
// entry, or a future caller passing a non-dot start gets a different tree.
func TestWalkFilesStartBoundaryMatchesTheWholeWalk(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"src/keep.go":               "package src\n",
		"src/nested/deep.go":        "package nested\n",
		"vendor/dep.go":             "package dep\n",
		"node_modules/lib/index.js": "x\n",
		"pkg/build":                 "a regular file named like a skip directory\n",
	})
	if err := os.Symlink("src", filepath.Join(dir, "linkdir")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := os.Symlink("src/keep.go", filepath.Join(dir, "linkfile.go")); err != nil {
		t.Fatal(err)
	}
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	whole, _ := ctx.WalkFiles(".")
	inWhole := map[string]bool{}
	for _, p := range whole {
		inWhole[p] = true
	}

	for _, tc := range []struct {
		start string
		want  []string
	}{
		// ok: an ordinary directory start walks its subtree.
		{"src", []string{"src/keep.go", "src/nested/deep.go"}},
		// A start named in the skip set is not entered, exactly as from ".".
		{"vendor", nil},
		// A start BELOW a skip-set directory is unreachable from "." too.
		{"node_modules/lib", nil},
		// A regular-file start yields the file, exactly as the whole walk does.
		{"src/keep.go", []string{"src/keep.go"}},
		// A regular file whose name is in the skip set is still a file.
		{"pkg/build", []string{"pkg/build"}},
		// A symlink start, file or directory, is never followed.
		{"linkdir", nil},
		{"linkfile.go", nil},
		{"linkdir/keep.go", nil},
	} {
		got, _ := ctx.WalkFiles(tc.start)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("WalkFiles(%q) = %v, want %v", tc.start, got, tc.want)
		}
		for _, p := range got {
			if !inWhole[p] {
				t.Errorf("WalkFiles(%q) yielded %q, which the whole-tree walk does not", tc.start, p)
			}
		}
	}
}
