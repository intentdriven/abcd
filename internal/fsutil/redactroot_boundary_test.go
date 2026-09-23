package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// A root is redacted only where it starts a path: at the string's start or
// after a character that cannot continue one. The same root appearing as the
// tail of a LONGER path (HOME /var/… inside the resolved /private/var/…) names
// a different directory, and redacting it strands the longer path's head in
// front of the replacement — "/private~/wt/a", neither redacted nor intact
// (iss-2609230641546141). The right-hand boundary already held; this is its
// left-hand twin.
func TestRedactRootRequiresABoundaryBeforeTheRoot(t *testing.T) {
	root := "/var/folders/x/T/home"
	for _, tc := range []struct{ in, want string }{
		// Inside a longer path: left intact, under the root and bare.
		{"/private/var/folders/x/T/home/wt/a", "/private/var/folders/x/T/home/wt/a"},
		{"cannot access /private/var/folders/x/T/home", "cannot access /private/var/folders/x/T/home"},
		{"/srv.d/var/folders/x/T/home/a", "/srv.d/var/folders/x/T/home/a"},
		// A real prefix still redacts, at the start and after a delimiter.
		{"/var/folders/x/T/home/wt/a", "~/wt/a"},
		{"/var/folders/x/T/home", "~"},
		{"open /var/folders/x/T/home/wt/a: denied", "open ~/wt/a: denied"},
		{"PATH=/bin:/var/folders/x/T/home/bin", "PATH=/bin:~/bin"},
		{`"/var/folders/x/T/home"`, `"~"`},
		{"file:///var/folders/x/T/home/a", "file://~/a"},
		// Both shapes in one string: only the one that starts a path.
		{"/private/var/folders/x/T/home/a and /var/folders/x/T/home/b", "/private/var/folders/x/T/home/a and ~/b"},
		// The root repeated inside a path under it is part of that path.
		{"/var/folders/x/T/home/var/folders/x/T/home", "~/var/folders/x/T/home"},
	} {
		if got := fsutil.RedactRoot(tc.in, root, "~"); got != tc.want {
			t.Errorf("RedactRoot(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// RedactHome redacts the home directory in both spellings a message can carry
// it in: the one the environment names, and the symlink-resolved one git and
// the kernel report back. Where the environment's spelling is a suffix of the
// resolved one, the boundary rule is what keeps the two from mangling each
// other, so neither order of redaction strands a prefix.
func TestRedactHomeRedactsBothSpellingsOfASymlinkedHome(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real", "home")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(filepath.Join(base, "real"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("HOME", filepath.Join(link, "home"))
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}

	for in, want := range map[string]string{
		filepath.Join(link, "home", "wt", "a"): filepath.Join("~", "wt", "a"),
		filepath.Join(resolved, "wt", "a"):     filepath.Join("~", "wt", "a"),
		"at " + resolved:                       "at ~",
	} {
		if got := fsutil.RedactHome(in); got != want {
			t.Errorf("RedactHome(%q) = %q, want %q", in, got, want)
		}
	}
}
