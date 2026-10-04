package gitutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestParseStatusKeepsEveryPathVerbatim pins the one parser of git's
// NUL-separated status listing: each entry's two columns and its path as
// written (a leading space, a newline and a quote survive), and a rename's or
// copy's source, in either column, carried with its destination rather than
// read as an entry of its own.
func TestParseStatusKeepsEveryPathVerbatim(t *testing.T) {
	out := " M  lead.txt\x00?? new\nline.txt\x00R  to.md\x00from.md\x00 C copy.md\x00orig.md\x00!! build/\x00?? q\"uote.txt\x00"
	got := gitutil.ParseStatus([]byte(out))
	want := []gitutil.StatusEntry{
		{XY: " M", Path: " lead.txt"},
		{XY: "??", Path: "new\nline.txt"},
		{XY: "R ", Path: "to.md", Orig: "from.md"},
		{XY: " C", Path: "copy.md", Orig: "orig.md"},
		{XY: "!!", Path: "build/"},
		{XY: "??", Path: "q\"uote.txt"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ParseStatus =\n%q\nwant\n%q", got, want)
	}
	if len(gitutil.ParseStatus(nil)) != 0 {
		t.Fatal("an empty listing parsed to entries")
	}
}

// TestStatusListsUntrackedFilesOneByOneAndIgnoredOnlyWhenAsked: the listing
// names every untracked file inside a new directory (never the directory
// alone), names ignored paths only under the Ignored option, an ignored
// directory as the directory alone, and narrows to the pathspecs given.
func TestStatusListsUntrackedFilesOneByOneAndIgnoredOnlyWhenAsked(t *testing.T) {
	repo := newRepo(t, "out/\n*.log\n")
	commitAll(t, repo)
	for rel, body := range map[string]string{
		"fresh/deep/a.txt": "a", "debug.log": "l", "out/x/y.bin": "y", "keep/b.txt": "b",
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths := func(es []gitutil.StatusEntry) []string {
		var out []string
		for _, e := range es {
			out = append(out, e.XY+" "+e.Path)
		}
		slices.Sort(out)
		return out
	}

	plain, err := gitutil.Status(repo, 1<<20, gitutil.StatusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(plain), []string{"?? fresh/deep/a.txt", "?? keep/b.txt"}; !slices.Equal(got, want) {
		t.Fatalf("plain status = %q, want %q", got, want)
	}

	all, err := gitutil.Status(repo, 1<<20, gitutil.StatusOptions{Ignored: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(all), []string{"!! debug.log", "!! out/", "?? fresh/deep/a.txt", "?? keep/b.txt"}; !slices.Equal(got, want) {
		t.Fatalf("status with ignored = %q, want %q", got, want)
	}

	narrow, err := gitutil.Status(repo, 1<<20, gitutil.StatusOptions{Pathspecs: []string{"keep"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(narrow), []string{"?? keep/b.txt"}; !slices.Equal(got, want) {
		t.Fatalf("narrowed status = %q, want %q", got, want)
	}

	if _, err := gitutil.Status(repo, 8, gitutil.StatusOptions{}); err == nil {
		t.Fatal("a listing past the cap was returned, not refused")
	}
}
