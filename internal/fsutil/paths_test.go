package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

func TestValidRelPath(t *testing.T) {
	for _, p := range []string{"README.md", "a/b/c.json", ".abcd/development/brief/01-product/README.md"} {
		if !fsutil.ValidRelPath(p) {
			t.Errorf("ValidRelPath(%q) = false, want true", p)
		}
	}
	for _, p := range []string{
		"", "/etc/passwd", "../outside", "a/../../b", "./a", "a//b", "a/", "a/./b",
		"a\x00b", "a\nb",
		// A backslash is never legitimate in a committed repo-relative path, and on
		// a Windows target it acts as a separator: path.Clean leaves it untouched,
		// so a backslash-bearing traversal would slip a slash-only guard and, after
		// filepath.FromSlash on GOOS=windows, walk out of the intended root. Refuse
		// any backslash outright, target-independently (iss-2608280807166510).
		`..\..\x`, `a\b`, `\x`, `a/..\..\b`,
	} {
		if fsutil.ValidRelPath(p) {
			t.Errorf("ValidRelPath(%q) = true, want false", p)
		}
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "present.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"regular file", file, true},
		{"directory", dir, true},
		{"absent", filepath.Join(dir, "nope.txt"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fsutil.Exists(tt.path)
			if err != nil {
				t.Fatalf("Exists(%q) returned error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// Exists follows symlinks: a link to a real file exists, a dangling link does not.
func TestExistsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good.link")
	if err := os.Symlink(target, good); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	dangling := filepath.Join(dir, "dangling.link")
	if err := os.Symlink(filepath.Join(dir, "absent"), dangling); err != nil {
		t.Fatal(err)
	}

	if got, err := fsutil.Exists(good); err != nil || !got {
		t.Errorf("Exists(link to file) = %v, %v; want true, nil", got, err)
	}
	if got, err := fsutil.Exists(dangling); err != nil || got {
		t.Errorf("Exists(dangling link) = %v, %v; want false, nil", got, err)
	}
}

// ExistsNoFollow lstats: a regular file exists, an absent path does not, and —
// unlike Exists — a DANGLING symlink exists, because the name occupies the path
// and that is what a placement check must see.
func TestExistsNoFollow(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "present.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, err := fsutil.ExistsNoFollow(file); err != nil || !got {
		t.Errorf("ExistsNoFollow(regular file) = %v, %v; want true, nil", got, err)
	}
	if got, err := fsutil.ExistsNoFollow(filepath.Join(dir, "nope.txt")); err != nil || got {
		t.Errorf("ExistsNoFollow(absent) = %v, %v; want false, nil", got, err)
	}

	dangling := filepath.Join(dir, "dangling.link")
	if err := os.Symlink(filepath.Join(dir, "absent"), dangling); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if got, err := fsutil.ExistsNoFollow(dangling); err != nil || !got {
		t.Errorf("ExistsNoFollow(dangling link) = %v, %v; want true, nil", got, err)
	}
}

// A path whose parent component is a regular file cannot exist; stat returns
// ENOTDIR, and Exists/ExistsNoFollow/IsDir/DirHasEntries must read that as "not present"
// (false, nil), not propagate it as a hard error — otherwise a caller that
// fails closed on errors aborts on an obviously-absent path.
func TestPathUnderAFileIsNotPresent(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	under := filepath.Join(file, "child") // afile is not a directory

	if got, err := fsutil.Exists(under); err != nil || got {
		t.Errorf("Exists(under-a-file) = %v, %v; want false, nil", got, err)
	}
	if got, err := fsutil.ExistsNoFollow(under); err != nil || got {
		t.Errorf("ExistsNoFollow(under-a-file) = %v, %v; want false, nil", got, err)
	}
	if got, err := fsutil.IsDir(under); err != nil || got {
		t.Errorf("IsDir(under-a-file) = %v, %v; want false, nil", got, err)
	}
	if got, err := fsutil.DirHasEntries(under); err != nil || got {
		t.Errorf("DirHasEntries(under-a-file) = %v, %v; want false, nil", got, err)
	}
}

func TestIsDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, err := fsutil.IsDir(dir); err != nil || !got {
		t.Errorf("IsDir(dir) = %v, %v; want true, nil", got, err)
	}
	if got, err := fsutil.IsDir(file); err != nil || got {
		t.Errorf("IsDir(file) = %v, %v; want false, nil", got, err)
	}
	if got, err := fsutil.IsDir(filepath.Join(dir, "absent")); err != nil || got {
		t.Errorf("IsDir(absent) = %v, %v; want false, nil", got, err)
	}
}

func TestDirHasEntries(t *testing.T) {
	empty := t.TempDir()
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, err := fsutil.DirHasEntries(empty); err != nil || got {
		t.Errorf("DirHasEntries(empty) = %v, %v; want false, nil", got, err)
	}
	if got, err := fsutil.DirHasEntries(full); err != nil || !got {
		t.Errorf("DirHasEntries(full) = %v, %v; want true, nil", got, err)
	}
}

// An absent directory is not an error — it simply holds no entries. The caller
// distinguishes "missing" from "empty" with Exists, so a presence rule and a
// non-empty rule stay independent.
func TestDirHasEntriesAbsent(t *testing.T) {
	got, err := fsutil.DirHasEntries(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("DirHasEntries(absent) returned error: %v", err)
	}
	if got {
		t.Errorf("DirHasEntries(absent) = true, want false")
	}
}

// A dotfile is an entry. A directory holding only .gitkeep is not empty.
func TestDirHasEntriesDotfileCounts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fsutil.DirHasEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Errorf("DirHasEntries(dir with only .gitkeep) = false, want true")
	}
}

// TestPathWithinFoldsCaseWhenAsked is the canonical containment comparison's
// case detector, stated over literal paths so it is deterministic on a
// case-folding host and a case-sensitive one alike: with fold set, a re-cased
// child is INSIDE its parent (the fail-closed direction a destructive-write
// gate needs); without it, the two are unrelated paths.
func TestPathWithinFoldsCaseWhenAsked(t *testing.T) {
	sep := string(filepath.Separator)
	root := sep + filepath.Join("Users", "dev", "repo")
	variant := sep + filepath.Join("Users", "dev", "REPO", "dist")

	for _, tc := range []struct {
		name             string
		child, parent    string
		fold             bool
		within, overlaps bool
	}{
		{"exact child folded", variant, root, true, true, true},
		{"exact child unfolded", variant, root, false, false, false},
		{"same directory re-cased", sep + filepath.Join("Users", "dev", "REPO"), root, true, true, true},
		{"identical", root, root, false, true, true},
		{"true sibling stays out", sep + filepath.Join("Users", "dev", "other"), root, true, false, false},
		{"parent inside child folded", root, sep + filepath.Join("Users", "DEV"), true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fsutil.PathWithin(tc.child, tc.parent, tc.fold); got != tc.within {
				t.Errorf("PathWithin(%q, %q, %v) = %v, want %v", tc.child, tc.parent, tc.fold, got, tc.within)
			}
			if got := fsutil.PathsOverlap(tc.child, tc.parent, tc.fold); got != tc.overlaps {
				t.Errorf("PathsOverlap(%q, %q, %v) = %v, want %v", tc.child, tc.parent, tc.fold, got, tc.overlaps)
			}
			if got := fsutil.PathsOverlap(tc.parent, tc.child, tc.fold); got != tc.overlaps {
				t.Errorf("PathsOverlap is not symmetric for (%q, %q, %v)", tc.parent, tc.child, tc.fold)
			}
		})
	}
}

// TestFoldPathMintsOneComparisonKey: the fold is the single key-minting step, so
// two case-variant spellings collapse to one key when folding and stay distinct
// when not.
func TestFoldPathMintsOneComparisonKey(t *testing.T) {
	a, b := ".abcd/development/decisions/adr-1.md", ".abcd/development/decisions/ADR-1.md"
	if fsutil.FoldPath(a, true) != fsutil.FoldPath(b, true) {
		t.Errorf("FoldPath(%q, true) != FoldPath(%q, true); case variants must share one key", a, b)
	}
	if fsutil.FoldPath(a, false) == fsutil.FoldPath(b, false) {
		t.Errorf("FoldPath(_, false) collapsed %q and %q; an unfolded key must stay verbatim", a, b)
	}
	if got := fsutil.FoldPath(a, false); got != a {
		t.Errorf("FoldPath(%q, false) = %q, want it unchanged", a, got)
	}
}

// TestRealExistingPath pins the existing-prefix resolver: a symlinked ancestor
// that exists is resolved, the absent remainder is rejoined lexically, and an
// empty path stays empty rather than resolving to the working directory.
func TestRealExistingPath(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	realBase, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}

	got := fsutil.RealExistingPath(filepath.Join(link, "absent", "leaf"))
	if want := filepath.Join(realBase, "absent", "leaf"); got != want {
		t.Errorf("RealExistingPath(link/absent/leaf) = %q, want %q", got, want)
	}
	if got := fsutil.RealExistingPath(""); got != "" {
		t.Errorf("RealExistingPath(\"\") = %q, want empty", got)
	}
}

// TestOwnerUIDReportsTheCallersOwnUID pins the canonical ownership lookup on the
// one case a single-uid test process can stage for real: a file it just created
// is its own. The foreign-uid half cannot be staged here — a test cannot create
// a second uid — so the callers that must refuse a foreign owner substitute this
// lookup instead (internal/core/rules.ownerUID), the same predicate-substitution
// idiom caseFoldingFS keeps in this package.
func TestOwnerUIDReportsTheCallersOwnUID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mine")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{dir, path} {
		got, err := fsutil.OwnerUID(target)
		if err != nil {
			t.Fatalf("OwnerUID(%q): %v", target, err)
		}
		if want := uint32(os.Getuid()); got != want {
			t.Errorf("OwnerUID(%q) = %d, want this process's uid %d", target, got, want)
		}
	}
	if _, err := fsutil.OwnerUID(filepath.Join(dir, "absent")); err == nil {
		t.Error("OwnerUID must report an error for an absent path, never a uid the caller could read as its own")
	}
}
