package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// scrubPathString takes the home out of a finding through the shared rule
// (fsutil.RedactHome): a home is replaced only where it stands as a path —
// nothing path-like before it, no letter or digit after it — and in every
// spelling a path carries it in. It used to replace the home unanchored, which
// turned a longer name sharing the home's prefix into "~bert/…" and a path
// that merely ends in the home's bytes into "/srv~/…", and it read HOME in the
// one spelling the environment gave, so the resolved or cleaned spelling went
// out whole.
func TestScrubPathStringRedactsTheHomeOnlyWhereItStandsAsAPath(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "alex")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv(abcdhome.StageHomeEnv, "")
	for _, tc := range []struct{ in, want string }{
		// The home under it, bare, and before punctuation.
		{home + "/notes/a.md: bad front matter", "~/notes/a.md: bad front matter"},
		{"cannot read " + home, "cannot read ~"},
		{"cannot read " + home + ".", "cannot read ~."},
		{"PATH=/bin:" + home + "/bin", "PATH=/bin:~/bin"},
		// A longer name sharing the home's prefix is a different directory.
		{home + "bert/a.md", home + "bert/a.md"},
		{home + "2/a.md", home + "2/a.md"},
		// The home's bytes as the tail of a longer path name another place.
		{"/srv" + home + "/a.md", "/srv" + home + "/a.md"},
		// The spelling the kernel reports where the home sits behind a link.
		{resolved + "/a.md", "~/a.md"},
		// Twice in one finding, each redacted once.
		{home + "/a.md and " + home + "/b.md", "~/a.md and ~/b.md"},
	} {
		if got := scrubPathString(tc.in, ""); got != tc.want {
			t.Errorf("scrubPathString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A HOME written with a doubled slash names the same directory as its cleaned
// spelling, which is the one a path filepath.Join built carries; both are
// scrubbed.
func TestScrubPathStringRedactsTheCleanedSpellingOfAnUncleanHome(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "alex"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := base + "//alex"
	t.Setenv("HOME", raw)
	t.Setenv(abcdhome.StageHomeEnv, "")
	for in, want := range map[string]string{
		filepath.Join(raw, "notes", "a.md"): "~/notes/a.md",
		raw + "/notes/a.md":                 "~/notes/a.md",
	} {
		if got := scrubPathString(in, ""); got != want {
			t.Errorf("scrubPathString(%q) = %q, want %q", in, got, want)
		}
	}
}

// The repository root is scrubbed before the home, so a root under the home
// reads as repo-relative, and a path under the home but outside the root still
// loses the home.
func TestScrubPathStringScrubsTheRootBeforeTheHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(abcdhome.StageHomeEnv, "")
	root := filepath.Join(home, "repo")
	for in, want := range map[string]string{
		root + "/a.md":       "a.md",
		"at " + root:         "at .",
		home + "/other/b.md": "~/other/b.md",
	} {
		if got := scrubPathString(in, root); got != want {
			t.Errorf("scrubPathString(%q, root) = %q, want %q", in, got, want)
		}
	}
}
