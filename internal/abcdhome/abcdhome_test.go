package abcdhome

import (
	"path/filepath"
	"testing"
)

// TestResolverSpellsTheHome pins the three forms the resolver hands out, each
// byte-identical to the spelling it replaces, so the sweep onto it changes no
// path. Rel never cleans: a leaf that is not clean stays visible to the
// fsutil.ValidRelPath check every home-scope reader runs on it.
func TestResolverSpellsTheHome(t *testing.T) {
	rels := []struct {
		leaf []string
		want string
	}{
		{nil, ".abcd.noindex"},
		{[]string{"trusted-roots"}, ".abcd.noindex/trusted-roots"},
		{[]string{"worktrees", "0123abcd"}, ".abcd.noindex/worktrees/0123abcd"},
		{[]string{"lab/x"}, ".abcd.noindex/lab/x"},
		{[]string{""}, ".abcd.noindex/"},
		{[]string{"../x"}, ".abcd.noindex/../x"},
	}
	for _, c := range rels {
		if got := Rel(c.leaf...); got != c.want {
			t.Errorf("Rel(%q) = %q, want %q", c.leaf, got, c.want)
		}
	}

	home := filepath.Join(t.TempDir(), "personhome")
	if got, want := Path(home), filepath.Join(home, ".abcd.noindex"); got != want {
		t.Errorf("Path(home) = %q, want %q", got, want)
	}
	if got, want := Path(home, "transcripts", "abc"), filepath.Join(home, ".abcd.noindex", "transcripts", "abc"); got != want {
		t.Errorf("Path(home, transcripts, abc) = %q, want %q", got, want)
	}
	if got, want := Path(home, "a/b"), filepath.Join(home, ".abcd.noindex", "a", "b"); got != want {
		t.Errorf("Path(home, a/b) = %q, want %q", got, want)
	}

	displays := []struct {
		leaf []string
		want string
	}{
		{nil, "~/.abcd.noindex"},
		{[]string{"rules.json"}, "~/.abcd.noindex/rules.json"},
		{[]string{"worktrees/<root-sha>/<name>/"}, "~/.abcd.noindex/worktrees/<root-sha>/<name>/"},
		{[]string{"lab", "<name>"}, "~/.abcd.noindex/lab/<name>"},
		{[]string{""}, "~/.abcd.noindex/"},
	}
	for _, c := range displays {
		if got := Display(c.leaf...); got != c.want {
			t.Errorf("Display(%q) = %q, want %q", c.leaf, got, c.want)
		}
	}
}
