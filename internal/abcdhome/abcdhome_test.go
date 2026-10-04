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
		{nil, ".abcd"},
		{[]string{"trusted-roots"}, ".abcd/trusted-roots"},
		{[]string{"worktrees", "0123abcd"}, ".abcd/worktrees/0123abcd"},
		{[]string{"lab/x"}, ".abcd/lab/x"},
		{[]string{""}, ".abcd/"},
		{[]string{"../x"}, ".abcd/../x"},
	}
	for _, c := range rels {
		if got := Rel(c.leaf...); got != c.want {
			t.Errorf("Rel(%q) = %q, want %q", c.leaf, got, c.want)
		}
	}

	home := filepath.Join(t.TempDir(), "personhome")
	if got, want := Path(home), filepath.Join(home, ".abcd"); got != want {
		t.Errorf("Path(home) = %q, want %q", got, want)
	}
	if got, want := Path(home, "transcripts", "abc"), filepath.Join(home, ".abcd", "transcripts", "abc"); got != want {
		t.Errorf("Path(home, transcripts, abc) = %q, want %q", got, want)
	}
	if got, want := Path(home, "a/b"), filepath.Join(home, ".abcd", "a", "b"); got != want {
		t.Errorf("Path(home, a/b) = %q, want %q", got, want)
	}

	displays := []struct {
		leaf []string
		want string
	}{
		{nil, "~/.abcd"},
		{[]string{"rules.json"}, "~/.abcd/rules.json"},
		{[]string{"worktrees/<root-sha>/<name>/"}, "~/.abcd/worktrees/<root-sha>/<name>/"},
		{[]string{"lab", "<name>"}, "~/.abcd/lab/<name>"},
		{[]string{""}, "~/.abcd/"},
	}
	for _, c := range displays {
		if got := Display(c.leaf...); got != c.want {
			t.Errorf("Display(%q) = %q, want %q", c.leaf, got, c.want)
		}
	}
}
