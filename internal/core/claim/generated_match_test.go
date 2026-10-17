package claim

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The generated-file matcher (spc-2610161044461448 step 11;
// iss-2610152125101468): the subset of git's gitattributes rules abcd reads
// from the root .gitattributes on origin's default branch, judging
// linguist-generated alone. Every row the subset supports agrees with git's
// own `git check-attr`; a line the reader does not understand answers "not
// generated", whatever git makes of it, so a misreading never shares a path.

// attrCase is one .gitattributes and the answer abcd gives for each path.
type attrCase struct {
	name  string
	attrs string
	// want is abcd's answer for each path: generated or not.
	want map[string]bool
	// skipped is true for a declaration the reader does not understand (a
	// macro, a C-quoted pattern, a POSIX class, an unclosed class): abcd
	// answers false for every path, and git is not asked to agree.
	skipped bool
}

var attrCorpus = []attrCase{
	{name: "a basename pattern matches at any depth", attrs: "*.pb.go linguist-generated\n",
		want: map[string]bool{"a.pb.go": true, "x/y/a.pb.go": true, "a.go": false, "a.pb.go.txt": false}},
	{name: "a literal basename", attrs: "commands.md linguist-generated\n",
		want: map[string]bool{"commands.md": true, "docs/commands.md": true, "docs/commands.mdx": false}},
	{name: "a pattern with a slash is anchored to the root", attrs: "docs/ref/*.md linguist-generated\n",
		want: map[string]bool{"docs/ref/a.md": true, "docs/ref/x/a.md": false, "x/docs/ref/a.md": false}},
	{name: "a leading slash anchors and is stripped", attrs: "/gen.md linguist-generated\n",
		want: map[string]bool{"gen.md": true, "x/gen.md": false}},
	{name: "a leading **/", attrs: "**/gen/*.go linguist-generated\n",
		want: map[string]bool{"gen/a.go": true, "x/y/gen/a.go": true, "gen/x/a.go": false, "xgen/a.go": false}},
	{name: "a trailing /**", attrs: "vendor/** linguist-generated\n",
		want: map[string]bool{"vendor/a": true, "vendor/x/y.go": true, "vendor": false, "x/vendor/a": false}},
	{name: "a middle /**/", attrs: "a/**/b.go linguist-generated\n",
		want: map[string]bool{"a/b.go": true, "a/x/b.go": true, "a/x/y/b.go": true, "b.go": false, "ab/b.go": false, "a/xb.go": false}},
	{name: "** away from a slash is a plain *", attrs: "a/x**y linguist-generated\n",
		want: map[string]bool{"a/xy": true, "a/xzzy": true, "a/xz/y": false}},
	{name: "a single * never crosses a slash", attrs: "docs/* linguist-generated\n",
		want: map[string]bool{"docs/a": true, "docs/a/b": false}},
	{name: "* followed by a slash names one directory", attrs: "src/*/gen.go linguist-generated\n",
		want: map[string]bool{"src/a/gen.go": true, "src/gen.go": false, "src/a/b/gen.go": false}},
	{name: "? matches one character but a slash", attrs: "docs/?.md linguist-generated\n",
		want: map[string]bool{"docs/a.md": true, "docs/ab.md": false, "docs/.md": false}},
	{name: "a character class and a range", attrs: "gen[0-9].md linguist-generated\nfile[!a].txt linguist-generated\nn[^x]t linguist-generated\n",
		want: map[string]bool{"gen1.md": true, "genx.md": false, "fileb.txt": true, "filea.txt": false, "nyt": true, "nxt": false}},
	{name: "a class opening with ]", attrs: "a[]]b linguist-generated\n",
		want: map[string]bool{"a]b": true, "axb": false}},
	{name: "a trailing slash matches no file", attrs: "gen/ linguist-generated\n",
		want: map[string]bool{"gen": false, "gen/a.md": false, "x/gen": false}},
	{name: "a later line unsets an earlier one", attrs: "*.md linguist-generated\nREADME.md -linguist-generated\n",
		want: map[string]bool{"a.md": true, "README.md": false, "x/README.md": false}},
	{name: "a later line sets what an earlier one unset", attrs: "*.md -linguist-generated\ndocs/*.md linguist-generated\n",
		want: map[string]bool{"docs/a.md": true, "a.md": false}},
	{name: "! leaves it unspecified, over an earlier line", attrs: "*.md linguist-generated\nb.md !linguist-generated\n",
		want: map[string]bool{"a.md": true, "b.md": false}},
	{name: "=true, =false and any other value", attrs: "t.md linguist-generated=true\nf.md linguist-generated=false\ny.md linguist-generated=yes\n",
		want: map[string]bool{"t.md": true, "f.md": false, "y.md": false}},
	{name: "-attr=value is unset", attrs: "*.md linguist-generated\nb.md -linguist-generated=true\n",
		want: map[string]bool{"b.md": false, "a.md": true}},
	{name: "the last token on a line wins", attrs: "a.md linguist-generated -linguist-generated\nb.md -linguist-generated linguist-generated\n",
		want: map[string]bool{"a.md": false, "b.md": true}},
	{name: "a negated pattern is ignored, as git ignores it", attrs: "*.md linguist-generated\n!b.md -linguist-generated\n",
		want: map[string]bool{"b.md": true, "!b.md": true}},
	{name: "comments, blanks, leading space, CRLF and other attributes", attrs: "# gen.md linguist-generated\n\n   gen.md text eol=lf linguist-generated diff\r\nother.md text\n",
		want: map[string]bool{"gen.md": true, "other.md": false, "#": false}},
	{name: "an escaped #", attrs: "\\#gen.md linguist-generated\n",
		want: map[string]bool{"#gen.md": true, "gen.md": false}},
	{name: "a byte-order mark", attrs: "\ufeffgen.md linguist-generated\n",
		want: map[string]bool{"gen.md": true}},
	{name: "matching is case-sensitive", attrs: "GEN.md linguist-generated\n",
		want: map[string]bool{"GEN.md": true, "gen.md": false}},
	{name: "** alone matches every file", attrs: "** linguist-generated\n",
		want: map[string]bool{"a": true, "x/y.go": true}},
	{name: "a macro unsetting it is honoured", attrs: "*.md linguist-generated\n[attr]nogen -linguist-generated\nb.md nogen\n",
		want: map[string]bool{"a.md": true, "b.md": false}},
	{name: "an invalid or reserved attribute name voids the line", attrs: "*.md linguist-generated\nb.md -linguist-generated bad^name\nc.txt linguist-generated bad^name\nd.md builtin_x -linguist-generated\n",
		want: map[string]bool{"a.md": true, "b.md": true, "c.txt": false, "d.md": true}},

	{name: "a macro setting it is not followed", attrs: "[attr]gen linguist-generated\ngen.md gen\n", skipped: true,
		want: map[string]bool{"gen.md": false}},
	{name: "a C-quoted pattern is skipped", attrs: "\"gen file.md\" linguist-generated\n", skipped: true,
		want: map[string]bool{"gen file.md": false}},
	{name: "a C-quoted pattern that unsets voids every earlier line", attrs: "*.md linguist-generated\n\"b c.md\" -linguist-generated\n", skipped: true,
		want: map[string]bool{"a.md": false, "b c.md": false}},
	{name: "a bare [attr] is a pattern git reads, and its unset voids every earlier line", attrs: "* linguist-generated\n[attr] -linguist-generated\n", skipped: true,
		want: map[string]bool{"a": false, "x": false}},
	{name: "a POSIX class is skipped", attrs: "gen[[:digit:]].md linguist-generated\n", skipped: true,
		want: map[string]bool{"gen1.md": false}},
	{name: "an unclosed class is skipped", attrs: "gen[.md linguist-generated\n", skipped: true,
		want: map[string]bool{"gen[.md": false}},
	{name: "an overlong line that unsets voids every earlier line", attrs: "*.md linguist-generated\n" + strings.Repeat("x", 2100) + ".md -linguist-generated\n", skipped: true,
		want: map[string]bool{"a.md": false}},
}

// TestTheGeneratedMatcherAnswersTheCorpus is the matcher's own table.
func TestTheGeneratedMatcherAnswersTheCorpus(t *testing.T) {
	for _, c := range attrCorpus {
		rules := parseAttributes(c.attrs)
		for p, want := range c.want {
			if got := rules.generated(p); got != want {
				t.Errorf("%s: generated(%q) = %v; want %v\n%s", c.name, p, got, want, c.attrs)
			}
		}
	}
}

// TestTheGeneratedMatcherAgreesWithGit runs every supported row through git's
// own check-attr in a scratch repository, case-sensitive, and requires the same
// answer: set or true is generated, and unset, unspecified or any other value
// is not.
func TestTheGeneratedMatcherAgreesWithGit(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Git("config", "core.ignorecase", "false")
	for _, c := range attrCorpus {
		if c.skipped {
			continue
		}
		if err := os.WriteFile(filepath.Join(r.Root(), ".gitattributes"), []byte(c.attrs), 0o644); err != nil {
			t.Fatal(err)
		}
		paths := make([]string, 0, len(c.want))
		for p := range c.want {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		got := checkAttr(t, r, paths)
		for _, p := range paths {
			if got[p] != c.want[p] {
				t.Errorf("%s: git check-attr says %q is generated=%v; the row says %v\n%s", c.name, p, got[p], c.want[p], c.attrs)
			}
		}
	}
}

// checkAttr asks git whether each path is linguist-generated (set or true).
func checkAttr(t *testing.T, r *gittest.Repo, paths []string) map[string]bool {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.Root(), "check-attr", "-z", "linguist-generated", "--"}, paths...)...)
	cmd.Env = r.Env()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git check-attr: %v\n%s", err, stderr.String())
	}
	f := strings.Split(strings.TrimSuffix(stdout.String(), "\x00"), "\x00")
	if len(f)%3 != 0 {
		t.Fatalf("git check-attr -z wrote %d fields: %q", len(f), stdout.String())
	}
	out := map[string]bool{}
	for i := 0; i < len(f); i += 3 {
		out[f[i]] = f[i+2] == "set" || f[i+2] == "true"
	}
	return out
}
