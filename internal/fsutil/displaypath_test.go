package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DisplayPath is the one rendering of a path that travels into machine output,
// which never carries an absolute developer-identity path (iss-81): a path
// inside the repository is named relative to it, one outside it has the home
// directory redacted to "~", and a relative path is reported as it was given.
func TestDisplayPathNamesAPathWithoutTheHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "src", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()

	for name, tc := range map[string]struct{ in, want string }{
		"inside the repository":         {filepath.Join(repo, "public", "a.zip"), "public/a.zip"},
		"the repository root":           {repo, "."},
		"outside it, under the home":    {filepath.Join(home, "staging"), "~/staging"},
		"a sibling sharing a prefix":    {repo + "-other", "~/src/repo-other"},
		"outside it and the home":       {filepath.Join(elsewhere, "out"), filepath.Join(elsewhere, "out")},
		"relative, reported as given":   {"site", "site"},
		"empty, reported as given":      {"", ""},
		"inside, through the real home": {filepath.Join(realPath(t, home), "src", "repo", "bin"), "bin"},
	} {
		if got := fsutil.DisplayPath(repo, tc.in); got != tc.want {
			t.Errorf("%s: DisplayPath(%q) = %q, want %q", name, tc.in, got, tc.want)
		}
	}
}

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
