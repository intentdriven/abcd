package claim

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

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
	// minGit, when set, is the oldest git whose check-attr the row is
	// compared with ("2.42"); an older git skips the row, saying so.
	minGit string
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
	{name: "a leading **/ matches at the root too", attrs: "**/a linguist-generated\nx/**/b/c linguist-generated\n",
		want: map[string]bool{"a": true, "y/a": true, "ya": false, "x/b/c": true, "x/y/b/c": true, "xb/c": false}},
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
	// linguist-generated=true is deliberately read as generated, as GitHub's
	// linguist reads it, though git itself reports the value "true" rather
	// than "set"; checkAttr maps both to generated.
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

	{name: "an invalid attribute name voids the line", attrs: "*.md linguist-generated\nb.md -linguist-generated bad^name\nc.txt linguist-generated bad^name\n",
		want: map[string]bool{"a.md": true, "b.md": true, "c.txt": false}},
	// git refuses a builtin_ name as reserved from 2.42 on.
	{name: "a reserved attribute name voids the line", attrs: "*.md linguist-generated\nd.md builtin_x -linguist-generated\n", minGit: "2.42",
		want: map[string]bool{"a.md": true, "d.md": true}},

	{name: "a bare [attr] defines no macro", attrs: "*.md linguist-generated\n[attr] nogen linguist-generated\nx.md nogen\n",
		want: map[string]bool{"x.md": true, "b.md": true}},
	{name: "a macro definition that unsets voids every earlier line", attrs: "*.md linguist-generated\n[attr]nogen -linguist-generated\nb.md nogen\n", skipped: true,
		want: map[string]bool{"a.md": false, "b.md": false}},
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
	have := gitVersion(t, r)
	for _, c := range attrCorpus {
		if c.skipped {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			if c.minGit != "" && versionBelow(have, c.minGit) {
				t.Skipf("this row needs git %s or later to compare with; this machine has git %s", c.minGit, strings.Join(have, "."))
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
					t.Errorf("git check-attr says %q is generated=%v; the row says %v\n%s", p, got[p], c.want[p], c.attrs)
				}
			}
		})
	}
}

// gitVersion is the fixture's git version, its numbers as git prints them.
func gitVersion(t *testing.T, r *gittest.Repo) []string {
	t.Helper()
	f := strings.Fields(r.Git("version"))
	if len(f) < 3 {
		t.Fatalf("git version printed %q", strings.Join(f, " "))
	}
	return strings.Split(f[2], ".")
}

// versionBelow reports whether the version have is older than want, a
// major.minor such as "2.42".
func versionBelow(have []string, want string) bool {
	for i, w := range strings.Split(want, ".") {
		if i >= len(have) {
			return true
		}
		h, _ := strconv.Atoi(have[i])
		n, _ := strconv.Atoi(w)
		if h != n {
			return h < n
		}
	}
	return false
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

// TestAPatternPastTheStepBudgetIsNotGenerated: matching one path against one
// line is bounded (matchSteps), and a line that would take longer answers
// "not generated" for that path, quickly, whatever earlier lines say.
func TestAPatternPastTheStepBudgetIsNotGenerated(t *testing.T) {
	// The long line begins and ends as the deep path does, so only matching
	// it can tell, and matching it explores every way to split the path.
	attrs := "*y linguist-generated\n*x linguist-generated\n" + strings.Repeat("**/", 300) + "q/*/x -linguist-generated\n"
	deep := strings.Repeat("a/", 60) + "x"
	done := make(chan bool, 1)
	start := time.Now()
	go func() { done <- parseAttributes(attrs).generated(deep) }()
	select {
	case got := <-done:
		if got {
			t.Fatalf("a path past the step budget is generated")
		}
		if took := time.Since(start); took > 2*time.Second {
			t.Fatalf("matching one path took %s; want it bounded", took)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("matching one path did not finish in 10s")
	}
	if !parseAttributes(attrs).generated("b/y") {
		t.Fatal("a path the long line matches within budget is judged as git would: generated by *y")
	}
}

// hostileScale is many long literal lines against many long basenames that
// end as the first line does: its literal is compared byte by byte at every
// offset of every basename.
func hostileScale() (attrRules, []string) {
	var attrs strings.Builder
	for i := range 120 {
		fmt.Fprintf(&attrs, "*%sb%d linguist-generated\n", strings.Repeat("a", 1990), i)
	}
	name := strings.Repeat("a", 4000) + "b0"
	paths := make([]string, 8000)
	for i := range paths {
		paths[i] = fmt.Sprintf("d%d/%s", i, name)
	}
	return parseAttributes(attrs.String()), paths
}

// benignScale is a large ordinary tree: 150,000 files of about 70 bytes and
// 50 lines of the shapes repositories declare.
func benignScale() (attrRules, []string) {
	var attrs strings.Builder
	for i := range 50 {
		switch i % 5 {
		case 0:
			fmt.Fprintf(&attrs, "**/*.gen%d.go linguist-generated\n", i)
		case 1:
			fmt.Fprintf(&attrs, "*.pb%d.go linguist-generated\n", i)
		case 2:
			fmt.Fprintf(&attrs, "**/gen%d/*.go linguist-generated\n", i)
		case 3:
			fmt.Fprintf(&attrs, "**/vendor%d/** linguist-generated\n", i)
		default:
			fmt.Fprintf(&attrs, "*_generated%d.go -linguist-generated\n", i)
		}
	}
	paths := make([]string, 150_000)
	for i := range paths {
		paths[i] = fmt.Sprintf("internal/pkg%03d/subsystem%02d/file_%06d_with_a_longer_name.gen.go", i%200, i%37, i)
	}
	paths[0] = "internal/third_party/vendor3/module/a.go"
	return parseAttributes(attrs.String()), paths
}

// TestTheBudgetBoundsWallTimeOverLongLiterals: the budget counts every byte
// compared or skipped, so a declaration whose lines compare long literals at
// every offset of long names stops well inside a second, declaring nothing.
func TestTheBudgetBoundsWallTimeOverLongLiterals(t *testing.T) {
	rules, paths := hostileScale()
	done := make(chan bool, 1)
	start := time.Now()
	go func() {
		_, spent := rules.generatedAmong(paths)
		done <- spent
	}()
	select {
	case spent := <-done:
		took := time.Since(start)
		t.Logf("the long-literal declaration stopped after %s", took)
		if !spent {
			t.Fatal("the long-literal declaration was judged within the budget")
		}
		if took > 3*time.Second {
			t.Fatalf("it took %s; want it stopped well inside a second", took)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the long-literal declaration did not stop in 20s")
	}
}

// TestALargeOrdinaryTreeFitsTheBudget: an ordinary tree of 150,000 files
// against 50 lines is judged in full, the budget nowhere near spent.
func TestALargeOrdinaryTreeFitsTheBudget(t *testing.T) {
	rules, paths := benignScale()
	start := time.Now()
	files, spent := rules.generatedAmong(paths)
	t.Logf("150,000 files against 50 lines took %s", time.Since(start))
	if spent {
		t.Fatal("an ordinary tree of 150,000 files spent the budget")
	}
	if !files["internal/third_party/vendor3/module/a.go"] {
		t.Fatal("a file the declaration names was not generated")
	}
}

// TestAnUnsettingLineMatchesRegardlessOfCase: a clone on a case-insensitive
// file system reads patterns without regard to case, so a line unsetting the
// attribute matches regardless of case, and reaches at least as far as git's
// would on any clone; a line setting it matches case-sensitively, so it never
// sets more than git would on any clone.
func TestAnUnsettingLineMatchesRegardlessOfCase(t *testing.T) {
	for _, c := range []struct {
		attrs string
		want  map[string]bool
	}{
		{"*.go linguist-generated\n*.GO -linguist-generated\n", map[string]bool{"x.go": false, "X.GO": false}},
		{"*.GO linguist-generated\n", map[string]bool{"x.go": false, "X.GO": true}},
		{"*.md linguist-generated\nDOCS/*.MD !linguist-generated\n", map[string]bool{"docs/a.md": false, "a.md": true}},
		{"*.md linguist-generated\n[0-_].MD linguist-generated=false\n", map[string]bool{"a.md": false, "A.md": false, "ab.md": true}},
		// git ignoring case lowers the text byte but not a class's own, so
		// [A] matches only A there, as it does case-sensitively.
		{"*.md linguist-generated\n[A].md -linguist-generated\nB\\.MD -linguist-generated\n", map[string]bool{"A.md": false, "a.md": true, "b.md": false, "c.md": true}},
		{"* linguist-generated\n[^a-z]* -linguist-generated\n[!]a]x -linguist-generated\n", map[string]bool{"Z": false, "Ax": false, "z": true, "ax": true}},
		// A setting line must match as git matches on both kinds of clone:
		// ignoring case, git lowers the text byte, so a negated class or an
		// escaped capital matches less there.
		{"[!a-c]x linguist-generated\n\\Ab linguist-generated\n[A]c linguist-generated\n", map[string]bool{"Bx": false, "dx": true, "Ab": false, "Ac": false}},
	} {
		rules := parseAttributes(c.attrs)
		for p, want := range c.want {
			if got := rules.generated(p); got != want {
				t.Errorf("generated(%q) = %v; want %v\n%s", p, got, want, c.attrs)
			}
		}
	}
}

// TestNoGeneratedFileIsUnsetOnEitherKindOfClone: whatever abcd calls
// generated, git calls generated on a clone that ignores case and on one that
// does not, for every row the reader supports.
func TestNoGeneratedFileIsUnsetOnEitherKindOfClone(t *testing.T) {
	cases := append([]attrCase(nil), attrCorpus...)
	cases = append(cases, attrCase{name: "an unset in other capitals", attrs: "*.go linguist-generated\n*.GO -linguist-generated\nDocs/*.md linguist-generated\n[0-_].MD -linguist-generated\n",
		want: map[string]bool{"x.go": false, "docs/a.md": false, "Docs/b.md": false, "Docs/a.md": false, "Docs/x.md": false}},
		attrCase{name: "a negated class in an unset", attrs: "* linguist-generated\n[^a-z]* -linguist-generated\n[!]a]x -linguist-generated\n",
			want: map[string]bool{"Z": false, "Ax": false, "z": true, "ax": true, "x/Y": false}},
		attrCase{name: "a negated class or an escaped capital that sets", attrs: "[!a-c]x linguist-generated\n\\Ab linguist-generated\n[A]c linguist-generated\n[!0-_]y linguist-generated\n",
			want: map[string]bool{"Bx": false, "dx": true, "Ab": false, "Ac": false, "ay": false, "~y": true}})
	for _, icase := range []string{"true", "false"} {
		t.Run("core.ignorecase="+icase, func(t *testing.T) { neverMoreGenerated(t, icase, cases) })
	}
}

// neverMoreGenerated fails for each path of cases abcd calls generated and
// git, under core.ignorecase=icase, does not.
func neverMoreGenerated(t *testing.T, icase string, cases []attrCase) {
	r := gittest.NewRepo(t)
	r.Git("config", "core.ignorecase", icase)
	have := gitVersion(t, r)
	for _, c := range cases {
		if c.skipped || c.minGit != "" && versionBelow(have, c.minGit) {
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
		rules := parseAttributes(c.attrs)
		for _, p := range paths {
			if rules.generated(p) && !got[p] {
				t.Errorf("%s: abcd calls %q generated, and git ignoring case does not\n%s", c.name, p, c.attrs)
			}
		}
	}
}
