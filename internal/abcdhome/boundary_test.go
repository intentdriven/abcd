package abcdhome

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The home folder's name is spelled once, in this package (spc-2610031309233367,
// D3). Two scanners hold the tree to it, in the shape of invariant 15's
// TestOnlyTheHistoryPackageNamesTheStorePath: each parses every non-test Go
// file under internal/ and cmd/ (testdata skipped) and judges string LITERALS,
// never comments, so prose that names the home to explain a boundary is
// invisible by construction rather than by an exception.
//
// A spelling of the home is one of three shapes:
//
//  1. a literal containing ".abcd.noindex";
//  2. a literal containing "~/.abcd";
//  3. a call that takes a home value and a ".abcd"-led string (".abcd", or
//     ".abcd/" and more): a literal, a constant or variable of the same package
//     holding one, or such a value as the left end of a concatenation or as the
//     one argument of a wrapping call (filepath.FromSlash). A home value is an
//     identifier or selector whose name contains "home" in any case (home,
//     s.home, homeDir, c.Home) or a call to a function whose name does
//     (userHome(), os.UserHomeDir).
//
// The third shape is what tells the home's ".abcd" from a repository's
// ".abcd/": the two tiers share a name and the repository tier keeps it, so
// filepath.Join(repoRoot, ".abcd", "config.json") is left alone (open design
// question 1, decided (a)). A home value under a name the rule does not know
// passes unseen; TestHomeNameScannerIsArmed pins the shapes it does know.

// homeNameNeedles are the literal shapes 1 and 2.
var homeNameNeedles = []string{".abcd.noindex", "~/.abcd"}

// homeLedName reports whether s is ".abcd" or a path led by it.
func homeLedName(s string) bool {
	return s == ".abcd" || strings.HasPrefix(s, ".abcd/")
}

// homeSpelling is one finding: where, and the string that spells the home.
type homeSpelling struct {
	line int
	text string
}

func (h homeSpelling) String() string { return fmt.Sprintf("line %d: %q", h.line, h.text) }

// packageStrings folds the package-level string constants and variables of
// the given files into their values, as far as literals, same-package names
// and concatenation determine them.
func packageStrings(files []*ast.File) map[string]string {
	exprs := map[string]ast.Expr{}
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Values) != len(vs.Names) {
					continue
				}
				for i, n := range vs.Names {
					exprs[n.Name] = vs.Values[i]
				}
			}
		}
	}
	values := map[string]string{}
	// A constant may name one declared after it; fold until nothing new resolves.
	for changed := true; changed; {
		changed = false
		for n, e := range exprs {
			if _, done := values[n]; done {
				continue
			}
			if v, ok := foldString(e, values); ok {
				values[n] = v
				changed = true
			}
		}
	}
	return values
}

// foldString is e's string value when literals, the known names and
// concatenation determine all of it.
func foldString(e ast.Expr, known map[string]string) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return "", false
		}
		v, err := strconv.Unquote(x.Value)
		return v, err == nil
	case *ast.Ident:
		v, ok := known[x.Name]
		return v, ok
	case *ast.ParenExpr:
		return foldString(x.X, known)
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return "", false
		}
		l, ok := foldString(x.X, known)
		if !ok {
			return "", false
		}
		r, ok := foldString(x.Y, known)
		return l + r, ok
	}
	return "", false
}

// leadString is the known leading part of e: its whole value when it folds,
// else the lead of a concatenation's left end, else the lead of the one
// argument of a wrapping call.
func leadString(e ast.Expr, known map[string]string) (string, bool) {
	if v, ok := foldString(e, known); ok {
		return v, true
	}
	switch x := e.(type) {
	case *ast.ParenExpr:
		return leadString(x.X, known)
	case *ast.BinaryExpr:
		if x.Op == token.ADD {
			return leadString(x.X, known)
		}
	case *ast.CallExpr:
		if len(x.Args) == 1 {
			return leadString(x.Args[0], known)
		}
	}
	return "", false
}

// calleeName is the name a call is made by: f for f(), Sel for x.Sel().
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// isHomeValue reports whether e names the person's home directory by the
// naming rule above.
func isHomeValue(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return strings.Contains(strings.ToLower(x.Name), "home")
	case *ast.SelectorExpr:
		return strings.Contains(strings.ToLower(x.Sel.Name), "home")
	case *ast.CallExpr:
		return strings.Contains(strings.ToLower(calleeName(x.Fun)), "home")
	case *ast.ParenExpr:
		return isHomeValue(x.X)
	}
	return false
}

// homeSpellings returns every spelling of the home in f, judged with the
// package's folded strings in known.
func homeSpellings(fset *token.FileSet, f *ast.File, known map[string]string) []homeSpelling {
	var out []homeSpelling
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(x.Value)
			if err != nil {
				return true
			}
			for _, needle := range homeNameNeedles {
				if strings.Contains(v, needle) {
					out = append(out, homeSpelling{fset.Position(x.Pos()).Line, v})
					break
				}
			}
		case *ast.CallExpr:
			home := false
			for _, a := range x.Args {
				if isHomeValue(a) {
					home = true
					break
				}
			}
			if !home {
				return true
			}
			for _, a := range x.Args {
				if lead, ok := leadString(a, known); ok && homeLedName(lead) {
					out = append(out, homeSpelling{fset.Position(a.Pos()).Line, lead})
				}
			}
		}
		return true
	})
	return out
}

// parsedPackage is one directory's non-test Go files, parsed.
type parsedPackage struct {
	fset  *token.FileSet
	files map[string]*ast.File // repo-relative path -> file
}

// sourceTree parses every non-test Go file under internal/ and cmd/ (testdata
// skipped), grouped by directory, and returns the number of files read.
func sourceTree(t *testing.T) (map[string]*parsedPackage, int) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	pkgs := map[string]*parsedPackage{}
	walked := 0
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// testdata holds fixtures, not shipped code.
				if d.Name() == "testdata" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			dir := filepath.Dir(rel)
			pp := pkgs[dir]
			if pp == nil {
				pp = &parsedPackage{fset: token.NewFileSet(), files: map[string]*ast.File{}}
				pkgs[dir] = pp
			}
			f, err := parser.ParseFile(pp.fset, rel, src, 0)
			if err != nil {
				return fmt.Errorf("parse %s: %w", rel, err)
			}
			pp.files[rel] = f
			walked++
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	// A walk that found nothing would pass while holding nothing, which is how a
	// boundary test rots: a renamed tree, a changed root.
	if walked < 100 {
		t.Fatalf("the walk read only %d non-test Go files under internal/ and cmd/; "+
			"it is not looking at the repository it is meant to hold", walked)
	}
	return pkgs, walked
}

// TestOnlyTheHomeResolverNamesTheHome is D3's boundary test: outside this
// package, no Go code spells the home folder's name.
func TestOnlyTheHomeResolverNamesTheHome(t *testing.T) {
	pkgs, _ := sourceTree(t)
	thisPkg := filepath.Join("internal", "abcdhome")
	var lines []string
	for dir, pp := range pkgs {
		if dir == thisPkg {
			continue
		}
		var files []*ast.File
		for _, f := range pp.files {
			files = append(files, f)
		}
		known := packageStrings(files)
		for rel, f := range pp.files {
			for _, h := range homeSpellings(pp.fset, f, known) {
				lines = append(lines, fmt.Sprintf("%s:%d: %q", rel, h.line, h.text))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Errorf("%s spells the abcd home folder's name; reach it through abcdhome.Rel, "+
			"abcdhome.Path or abcdhome.Display, the one place the name is written", l)
	}
}

// parseHostile parses one source held in a test and returns its spellings.
func parseHostile(t *testing.T, src string) []homeSpelling {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hostile.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	return homeSpellings(fset, f, packageStrings([]*ast.File{f}))
}

// TestHomeNameScannerIsArmed proves the scanner reports each shape it claims
// to and leaves a comment and the repository tier alone, so a pass above is
// the property holding rather than the scanner missing what it walked.
func TestHomeNameScannerIsArmed(t *testing.T) {
	hostile := map[string]string{
		"the new name":                   `var x = "` + ".abcd" + `.noindex/trusted-roots"`,
		"the tilde form":                 `var x = "~/` + ".abcd" + `/rules.json"`,
		"a join with home":               `func f(home string) string { return filepath.Join(home, ".abcd") }`,
		"a join with a selector home":    `func (s *S) f() string { return filepath.Join(s.home, ".abcd", "x") }`,
		"a join with homeDir":            `func f(homeDir string) string { return filepath.Join(homeDir, ".abcd/x") }`,
		"a join with userHome()":         `func f() string { return filepath.Join(userHome(), ".abcd") }`,
		"a same-package constant":        "const UserRelPath = \".abcd/rules.json\"\nfunc f(home string) { fsutil.ReadHomeDeclaration(home, UserRelPath, 1) }",
		"a constant through FromSlash":   "const rel = \".abcd/transcripts\"\nfunc f(home string) string { return filepath.Join(home, filepath.FromSlash(rel)) }",
		"a constant led concatenation":   "const store = \".abcd/worktrees\"\nfunc f(home, sha string) { fsutil.EnsureHomeScope(home, store+\"/\"+sha, 0) }",
		"a constant folded from another": "const a = \".abcd\"\nconst b = a + \"/x\"\nfunc f(home string) { g(home, b) }",
		"a working-tree probe":           `func f(home string) string { return workingTreeAbove(home, ".abcd") }`,
	}
	for label, body := range hostile {
		src := "package p\n\n" + body + "\n"
		if got := parseHostile(t, src); len(got) != 1 {
			t.Errorf("%s: the scanner reports %v, want exactly one spelling; it is not armed for that shape\n%s", label, got, src)
		}
	}
	benign := map[string]string{
		"a comment naming the home": "// the home is ~/" + ".abcd" + "/trusted-roots, read by rules.\nvar x = \"unrelated\"",
		"the repository tier":       `func f(repoRoot string) string { return filepath.Join(repoRoot, ".abcd", "config.json") }`,
		"a repo-tier constant":      "const rel = \".abcd/rules.json\"\nfunc f(root string) string { return filepath.Join(root, rel) }",
		"a home with another leaf":  `func f(home string) string { return filepath.Join(home, ".config") }`,
	}
	for label, body := range benign {
		src := "package p\n\n" + body + "\n"
		if got := parseHostile(t, src); len(got) != 0 {
			t.Errorf("%s: the scanner reports %v from a source that does not spell the home\n%s", label, got, src)
		}
	}
}

// searchSettingsNeedles name the computer's search settings: the indexing
// switch, the volume's index store and the privacy list inside it. abcd
// changes only its own folder and never these (adr-2610030720195401, D5).
var searchSettingsNeedles = []string{"mdutil", ".Spotlight-V100", "VolumeConfiguration.plist"}

// searchSettingsLiterals returns every literal in f naming a search setting.
func searchSettingsLiterals(fset *token.FileSet, f *ast.File) []homeSpelling {
	var out []homeSpelling
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		for _, needle := range searchSettingsNeedles {
			if strings.Contains(v, needle) {
				out = append(out, homeSpelling{fset.Position(lit.Pos()).Line, v})
				break
			}
		}
		return true
	})
	return out
}

// TestNoCodeNamesTheSearchSettings is D5's test: no Go code, this package's
// included, names the computer's search settings in a literal.
func TestNoCodeNamesTheSearchSettings(t *testing.T) {
	pkgs, _ := sourceTree(t)
	var lines []string
	for _, pp := range pkgs {
		for rel, f := range pp.files {
			for _, h := range searchSettingsLiterals(pp.fset, f) {
				lines = append(lines, fmt.Sprintf("%s:%d: %q", rel, h.line, h.text))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Errorf("%s names the computer's search settings; abcd keeps its folders out of "+
			"indexing only by changing its own folder (adr-2610030720195401)", l)
	}
}

// TestSearchSettingsScannerIsArmed proves the search-settings scanner reports
// each setting in a literal and ignores it in a comment. The hostile literals
// are written out here rather than drawn from searchSettingsNeedles, so a
// needle dropped from the scanner fails this test instead of shrinking it.
func TestSearchSettingsScannerIsArmed(t *testing.T) {
	hostile := []string{
		"mdutil -i off /",
		"/.Spotlight-V100/Store-V2",
		"/System/Volumes/Data/.Spotlight-V100/VolumeConfiguration.plist",
		"VolumeConfiguration.plist",
	}
	for _, needle := range hostile {
		src := "package p\n\nvar x = " + strconv.Quote(needle) + "\n"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "hostile.go", src, 0)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got := searchSettingsLiterals(fset, f); len(got) != 1 {
			t.Errorf("the scanner admits the literal %q; it is not armed for that needle", needle)
		}
	}
	src := "package p\n\n// abcd never runs mdutil, nor edits .Spotlight-V100 or VolumeConfiguration.plist.\nvar x = \"unrelated\"\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "commented.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := searchSettingsLiterals(fset, f); len(got) != 0 {
		t.Errorf("the scanner reports %v from a comment; it judges literals, not prose", got)
	}
}
