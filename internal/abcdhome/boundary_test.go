package abcdhome

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
// A spelling of the home is one of five shapes:
//
//  1. a literal containing ".abcd.noindex";
//  2. a literal containing "~/.abcd";
//  3. a literal holding a format verb followed by "/.abcd" as a whole path
//     element ("%s/.abcd/lab", "%v/.abcd"): the shape fmt.Sprintf(.., home)
//     builds the path in. It is judged anywhere, not only beside a home
//     value, since no repository-tier code formats its ".abcd" that way;
//  4. a call that takes a home value and a ".abcd"-led string (".abcd", or
//     ".abcd/" and more): a literal, a constant or variable of the same package
//     holding one, or such a value as the left end of a concatenation or as the
//     one argument of a wrapping call (filepath.FromSlash);
//  5. a composite literal holding the same pair as elements
//     ([]string{home, ".abcd", "lab"}), a keyed element judged by its value.
//
// A home value is an identifier or selector whose name contains "home" in any
// case (home, s.home, homeDir, c.Home); a call to a function whose name does
// (userHome(), os.UserHomeDir) or that carries the string "HOME" as an
// argument (os.Getenv("HOME")); or an identifier assigned from a home value
// anywhere in the same top-level declaration (h, _ := os.UserHomeDir(), and
// d := h after it), closures inside it included.
//
// Shapes 4 and 5 are what tell the home's ".abcd" from a repository's
// ".abcd/": the two tiers share a name and the repository tier keeps it, so
// filepath.Join(repoRoot, ".abcd", "config.json") is left alone (open design
// question 1, decided (a)). A home value under a name the rule does not know
// passes unseen, and so does a concatenation that is not a call's argument,
// home + "/.abcd/lab": its literal is not ".abcd"-led and the scanner does not
// follow "+" from a home value (the spec states this limit). Assignment
// tracking is by name, not by scope: a shadowing name inside the declaration
// is judged as the outer one. TestHomeNameScannerIsArmed pins the shapes the
// scanner does know.

// homeNameNeedles are the literal shapes 1 and 2.
var homeNameNeedles = []string{".abcd.noindex", "~/.abcd"}

// homeFormatVerb is the literal shape 3: a format verb (flags, width and
// precision allowed) followed by "/.abcd" ending a path element.
var homeFormatVerb = regexp.MustCompile(`%[-+# 0-9.*\[\]]*[a-zA-Z]/\.abcd(?:[^\w.-]|$)`)

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
// naming rule above; locals holds the identifiers assigned from a home value
// in the declaration e sits in.
func isHomeValue(e ast.Expr, locals map[string]bool) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return locals[x.Name] || strings.Contains(strings.ToLower(x.Name), "home")
	case *ast.SelectorExpr:
		return strings.Contains(strings.ToLower(x.Sel.Name), "home")
	case *ast.CallExpr:
		if strings.Contains(strings.ToLower(calleeName(x.Fun)), "home") {
			return true
		}
		for _, a := range x.Args {
			if b, ok := a.(*ast.BasicLit); ok && b.Kind == token.STRING {
				if v, err := strconv.Unquote(b.Value); err == nil && v == "HOME" {
					return true
				}
			}
		}
	case *ast.ParenExpr:
		return isHomeValue(x.X, locals)
	}
	return false
}

// homeLocals returns the identifiers decl assigns from a home value, through
// := and = assignments and var declarations, followed until nothing new is
// learned so a copy of a home local is one too. A single call assigned to
// several names makes one of them a home: the one in the place of the result
// the callee declares with a home name, when results names the callee and one
// of its results is so named (repoRoot, _ := virginHome(t) against
// virginHome's (repoRoot, home string)); otherwise the first
// (h, err := os.UserHomeDir()), and none when another of the names is a home
// by its own name (repoRoot, home := virginHome()), the call's home being the
// one so named. An error named for the home (h, homeErr := ...) names no
// home.
func homeLocals(decl ast.Node, results map[string][]string) map[string]bool {
	type pair struct {
		lhs []ast.Expr
		rhs []ast.Expr
	}
	var pairs []pair
	ast.Inspect(decl, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			pairs = append(pairs, pair{x.Lhs, x.Rhs})
		case *ast.ValueSpec:
			lhs := make([]ast.Expr, len(x.Names))
			for i, id := range x.Names {
				lhs[i] = id
			}
			pairs = append(pairs, pair{lhs, x.Values})
		}
		return true
	})
	locals := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, p := range pairs {
			for i, l := range p.lhs {
				var r ast.Expr
				switch {
				case len(p.rhs) == len(p.lhs):
					r = p.rhs[i]
				case len(p.rhs) == 1 && i == declaredHome(p.rhs[0], results, len(p.lhs)):
					r = p.rhs[0]
				case len(p.rhs) == 1 && i == 0 && declaredHome(p.rhs[0], results, len(p.lhs)) < 0 && !homeNamedAmong(p.lhs[1:]):
					r = p.rhs[0]
				default:
					continue
				}
				id, ok := l.(*ast.Ident)
				if !ok || id.Name == "_" || locals[id.Name] || !isHomeValue(r, locals) {
					continue
				}
				locals[id.Name] = true
				changed = true
			}
		}
	}
	return locals
}

// homeNamedAmong reports whether one of names is an identifier whose own name
// says home.
func homeNamedAmong(names []ast.Expr) bool {
	for _, n := range names {
		if id, ok := n.(*ast.Ident); ok && namesHome(id.Name) {
			return true
		}
	}
	return false
}

// namesHome reports whether an identifier's own name says it holds a home: it
// says home and is not an error (homeErr, errHome).
func namesHome(name string) bool {
	l := strings.ToLower(name)
	return strings.Contains(l, "home") && !strings.Contains(l, "err")
}

// declaredHome returns the place of the home-named result among the n that
// the function call e calls declares in results, or -1 when e calls no
// function results names with n results, one of them home-named.
func declaredHome(e ast.Expr, results map[string][]string, n int) int {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return -1
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok {
		return -1
	}
	names := results[fn.Name]
	if len(names) != n {
		return -1
	}
	for i, name := range names {
		if namesHome(name) {
			return i
		}
	}
	return -1
}

// resultNames returns, for each function f declares (methods aside) with
// named results, the names of its results in order.
func resultNames(f *ast.File) map[string][]string {
	out := map[string][]string{}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Type.Results == nil {
			continue
		}
		var names []string
		for _, field := range fd.Type.Results.List {
			for _, id := range field.Names {
				names = append(names, id.Name)
			}
		}
		if len(names) > 0 {
			out[fd.Name.Name] = names
		}
	}
	return out
}

// besideHome returns the ".abcd"-led elements of elems when one of elems is a
// home value (shapes 4 and 5), each with its line.
func besideHome(fset *token.FileSet, elems []ast.Expr, known map[string]string, locals map[string]bool) []homeSpelling {
	home := false
	for _, e := range elems {
		if isHomeValue(e, locals) {
			home = true
			break
		}
	}
	if !home {
		return nil
	}
	var out []homeSpelling
	for _, e := range elems {
		if lead, ok := leadString(e, known); ok && homeLedName(lead) {
			out = append(out, homeSpelling{fset.Position(e.Pos()).Line, lead})
		}
	}
	return out
}

// homeSpellings returns every spelling of the home in f, judged with the
// package's folded strings in known. Each top-level declaration is judged
// with the home locals it assigns.
func homeSpellings(fset *token.FileSet, f *ast.File, known map[string]string) []homeSpelling {
	return spellingsSkipping(fset, f, known, nil)
}

// testHomeSpellings is homeSpellings for a test file: the same five shapes,
// save a literal that is a test's failure or log message (messageLiterals),
// which is prose about the test the way a comment is prose about the code.
func testHomeSpellings(fset *token.FileSet, f *ast.File, known map[string]string) []homeSpelling {
	return spellingsSkipping(fset, f, known, messageLiterals(f))
}

// messageMethods are the testing methods whose string arguments are a message
// a person reads when the test fails or logs, or the name a subtest is listed
// by, never a value the code under test is handed or judged against.
var messageMethods = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true,
	"Log": true, "Logf": true, "Skip": true, "Skipf": true, "Run": true,
}

// messageLiterals returns the string literals in f that are a message
// argument of a messageMethods call on a testing receiver (t.Errorf("..."),
// b.Fatal("a" + "b"), t.Run("name", ...)), directly or through parentheses
// and concatenation. The receiver must be a name f declares as a
// testing parameter (testingParams): a production method of the same name
// (gitutil.Run, a run's Log) is handed its arguments as values, and
// fmt.Errorf's builds an error value that can be a fixture. A literal inside
// a call nested in the arguments (t.Errorf("%s", filepath.Join(home,
// ".abcd"))) is not a message and stays judged.
func messageLiterals(f *ast.File) map[*ast.BasicLit]bool {
	testers := testingParams(f)
	out := map[*ast.BasicLit]bool{}
	var mark func(e ast.Expr)
	mark = func(e ast.Expr) {
		switch x := e.(type) {
		case *ast.BasicLit:
			out[x] = true
		case *ast.ParenExpr:
			mark(x.X)
		case *ast.BinaryExpr:
			mark(x.X)
			mark(x.Y)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !messageMethods[sel.Sel.Name] {
			return true
		}
		if recv, ok := sel.X.(*ast.Ident); !ok || !testers[recv.Name] {
			return true
		}
		for _, a := range call.Args {
			mark(a)
		}
		return true
	})
	return out
}

// testingTypes are the parameter types whose methods messageMethods names.
var testingTypes = map[string]bool{"*testing.T": true, "*testing.B": true, "*testing.F": true, "testing.TB": true}

// testingParams returns the names f declares as a parameter of a testingTypes
// type, in any function or function literal. Names are tracked as the
// scanner tracks home locals, by name rather than scope.
func testingParams(f *ast.File) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		ft, ok := n.(*ast.FuncType)
		if !ok || ft.Params == nil {
			return true
		}
		for _, field := range ft.Params.List {
			if !testingTypes[typeString(field.Type)] {
				continue
			}
			for _, id := range field.Names {
				out[id.Name] = true
			}
		}
		return true
	})
	return out
}

// typeString spells a parameter type of the shapes testingTypes holds
// (pkg.Name, *pkg.Name), and "" for any other.
func typeString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		if s := typeString(x.X); s != "" {
			return "*" + s
		}
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			return pkg.Name + "." + x.Sel.Name
		}
	}
	return ""
}

// spellingsSkipping is homeSpellings judging every string literal but those
// in skip.
func spellingsSkipping(fset *token.FileSet, f *ast.File, known map[string]string, skip map[*ast.BasicLit]bool) []homeSpelling {
	var out []homeSpelling
	results := resultNames(f)
	for _, decl := range f.Decls {
		locals := homeLocals(decl, results)
		ast.Inspect(decl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if x.Kind != token.STRING || skip[x] {
					return true
				}
				v, err := strconv.Unquote(x.Value)
				if err != nil {
					return true
				}
				spelled := homeFormatVerb.MatchString(v)
				for _, needle := range homeNameNeedles {
					spelled = spelled || strings.Contains(v, needle)
				}
				if spelled {
					out = append(out, homeSpelling{fset.Position(x.Pos()).Line, v})
				}
			case *ast.CallExpr:
				out = append(out, besideHome(fset, x.Args, known, locals)...)
			case *ast.CompositeLit:
				elems := make([]ast.Expr, len(x.Elts))
				for i, e := range x.Elts {
					if kv, ok := e.(*ast.KeyValueExpr); ok {
						e = kv.Value
					}
					elems[i] = e
				}
				out = append(out, besideHome(fset, elems, known, locals)...)
			}
			return true
		})
	}
	return out
}

// parsedPackage is one directory's Go files, parsed.
type parsedPackage struct {
	fset  *token.FileSet
	files map[string]*ast.File // repo-relative path -> file
}

// sourceTree parses every non-test Go file under internal/ and cmd/ (testdata
// skipped), grouped by directory, and returns the number of files read.
func sourceTree(t *testing.T) (map[string]*parsedPackage, int) {
	t.Helper()
	return goTree(t, false)
}

// testTree parses every Go file under internal/ and cmd/ (testdata skipped),
// test files and the shipped code beside them, grouped by directory, so a
// test's constants fold with its package's; the count is of test files.
func testTree(t *testing.T) (map[string]*parsedPackage, int) {
	t.Helper()
	return goTree(t, true)
}

// goTree walks internal/ and cmd/ for non-test Go files, and for test files
// too when tests is set, counting the files of the kind asked for.
func goTree(t *testing.T, tests bool) (map[string]*parsedPackage, int) {
	t.Helper()
	kind := "non-test"
	if tests {
		kind = "test"
	}
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
			isTest := strings.HasSuffix(d.Name(), "_test.go")
			if !strings.HasSuffix(d.Name(), ".go") || (isTest && !tests) {
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
			if isTest == tests {
				walked++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	// A walk that found nothing would pass while holding nothing, which is how a
	// boundary test rots: a renamed tree, a changed root.
	if walked < 100 {
		t.Fatalf("the walk read only %d %s Go files under internal/ and cmd/; "+
			"it is not looking at the repository it is meant to hold", walked, kind)
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

// testHomeSpellers are the test files that plant or name the home's old or new
// folder on purpose, each with the reason it must spell the name rather than
// reach it through abcdhome.Path or abcdhome.Display. Every other test file
// is held to the boundary above, so a fixture follows the name when it
// changes instead of seeding a folder the code no longer reads
// (iss-2610042333573913). This package's own tests are exempt as its code is.
var testHomeSpellers = map[string]string{
	"internal/surface/cli/homestop_test.go": "the stop test: it plants the old folder beside and " +
		"instead of the new one and expects the stop to name both",
	"internal/surface/cli/hooks_homestop_test.go": "the hooks' stop test: it plants the old folder " +
		"and expects every hook to stop on it",
	"internal/surface/cli/firstrun_home_test.go": "it expects a first run to create the new folder " +
		"and never the old one, each by its name",
	"internal/core/history/store_boundary_test.go": "invariant 15's boundary test: its needles and " +
		"hostile sources spell the transcript store's path under both names",
	"internal/core/lint/scribecontract_test.go": "it arms the scribe's transcript-store check with " +
		"every vintage of the store's path, the old folder's included",
	"internal/core/ahoy/noindex_block_test.go": "it expects the managed block to name the new folder " +
		"and no spelling of the old one left",
	"internal/core/ahoy/store_worktrees_test.go": "it plants worktrees under the old folder and " +
		"moves them, as the rename does",
	"internal/fsutil/home_test.go": "the home-scope primitives' own tests: fsutil takes any rel " +
		"below the home and they pass their own",
	"internal/fsutil/home_race_test.go":       "the home-scope primitives' race tests, on their own rels",
	"internal/fsutil/home_scope_mode_test.go": "the home-scope primitives' mode tests, on their own rels",
	"internal/fsutil/home_declared_test.go":   "the home-declaration primitive's tests, on their own rels",
	"internal/fsutil/replaced_test.go":        "the replaced-file primitive's tests, on their own rels",
}

// TestNoTestBuildsTheHomeByHand is the boundary above held over test files:
// outside this package and testHomeSpellers, no test spells the home folder's
// name, in a fixture it builds or an output it expects. A failure or log
// message is prose and is not judged (testHomeSpellings).
func TestNoTestBuildsTheHomeByHand(t *testing.T) {
	pkgs, _ := testTree(t)
	thisPkg := filepath.Join("internal", "abcdhome")
	var lines []string
	spelled := map[string]bool{}
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
			if !strings.HasSuffix(rel, "_test.go") {
				continue
			}
			slashed := filepath.ToSlash(rel)
			for _, h := range testHomeSpellings(pp.fset, f, known) {
				spelled[slashed] = true
				if _, ok := testHomeSpellers[slashed]; ok {
					continue
				}
				lines = append(lines, fmt.Sprintf("%s:%d: %q", rel, h.line, h.text))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Errorf("%s spells the abcd home folder's name in a test; build the fixture with "+
			"abcdhome.Path and expect the output through abcdhome.Display, so the test follows "+
			"the name when it changes, or declare the file in testHomeSpellers with the reason "+
			"it names the folder on purpose", l)
	}
	// An entry that no longer spells the home exempts nothing and would exempt
	// the next fixture written there unseen.
	var stale []string
	for rel := range testHomeSpellers {
		if !spelled[rel] {
			stale = append(stale, rel)
		}
	}
	sort.Strings(stale)
	for _, rel := range stale {
		t.Errorf("testHomeSpellers declares %s, which no longer spells the home; remove the entry", rel)
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
		"a join with HOME read inline":   `func f() string { return filepath.Join(os.Getenv("HOME"), ".abcd", "lab") }`,
		"a local from UserHomeDir":       `func f() string { h, _ := os.UserHomeDir(); return filepath.Join(h, ".abcd", "lab") }`,
		"a local from Getenv HOME":       `func f() string { d := os.Getenv("HOME"); return filepath.Join(d, ".abcd/lab") }`,
		"a local declared from HOME":     `func f() string { var d = os.Getenv("HOME"); return filepath.Join(d, ".abcd") }`,
		"a local assigned in a closure":  `func f() func() string { h, _ := os.UserHomeDir(); return func() string { return filepath.Join(h, ".abcd") } }`,
		"a local copied from a local":    `func f() string { h, _ := os.UserHomeDir(); d := h; return filepath.Join(d, ".abcd") }`,
		"a format verb before the name":  `func f(home string) string { return fmt.Sprintf("%s/` + ".abcd" + `/lab", home) }`,
		"a format verb, the name last":   `func f(home string) string { return fmt.Sprintf("%v/` + ".abcd" + `", home) }`,
		"a slice beside a home":          `func f(home string) string { return filepath.Join([]string{home, ".abcd", "lab"}...) }`,
		"a slice beside a HOME local":    `func f() []string { h, _ := os.UserHomeDir(); return []string{h, ".abcd/lab"} }`,
		"a local beside a home error":    `func f() string { h, homeErr := os.UserHomeDir(); _ = homeErr; return filepath.Join(h, ".abcd", "lab") }`,
	}
	for label, body := range hostile {
		src := "package p\n\n" + body + "\n"
		if got := parseHostile(t, src); len(got) != 1 {
			t.Errorf("%s: the scanner reports %v, want exactly one spelling; it is not armed for that shape\n%s", label, got, src)
		}
	}
	benign := map[string]string{
		"a comment naming the home":  "// the home is ~/" + ".abcd" + "/trusted-roots, read by rules.\nvar x = \"unrelated\"",
		"the repository tier":        `func f(repoRoot string) string { return filepath.Join(repoRoot, ".abcd", "config.json") }`,
		"a repo-tier constant":       "const rel = \".abcd/rules.json\"\nfunc f(root string) string { return filepath.Join(root, rel) }",
		"a home with another leaf":   `func f(home string) string { return filepath.Join(home, ".config") }`,
		"a local from the cwd":       `func f() string { d, _ := os.Getwd(); return filepath.Join(d, ".abcd") }`,
		"another variable read":      `func f() string { return filepath.Join(os.Getenv("PWD"), ".abcd") }`,
		"a HOME local elsewhere":     "func g() { h, _ := os.UserHomeDir(); _ = h }\nfunc f(h string) string { return filepath.Join(h, \".abcd\") }",
		"a repository-tier slice":    `func f(repoRoot string) []string { return []string{repoRoot, ".abcd", "config.json"} }`,
		"a format with another dot":  `func f(home string) string { return fmt.Sprintf("%s/.abcdef", home) }`,
		"a root beside a named home": `func f() string { repoRoot, home := virginHome(); _ = home; return filepath.Join(repoRoot, ".abcd") }`,
	}
	for label, body := range benign {
		src := "package p\n\n" + body + "\n"
		if got := parseHostile(t, src); len(got) != 0 {
			t.Errorf("%s: the scanner reports %v from a source that does not spell the home\n%s", label, got, src)
		}
	}
}

// TestTestHomeScannerIsArmed proves the test-file scanner reports a fixture
// or an expected output that spells the home, and leaves a failure or log
// message and a subtest's name alone, so a pass of
// TestNoTestBuildsTheHomeByHand is the property holding rather than the
// message exemption swallowing a fixture.
func TestTestHomeScannerIsArmed(t *testing.T) {
	scan := func(src string) []homeSpelling {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "hostile_test.go", src, 0)
		if err != nil {
			t.Fatalf("parse: %v\n%s", err, src)
		}
		return testHomeSpellings(fset, f, packageStrings([]*ast.File{f}))
	}
	hostile := map[string]string{
		"a fixture joined by hand":        `func TestX(t *testing.T) { home := t.TempDir(); _ = os.MkdirAll(filepath.Join(home, ".abcd", "lab"), 0o700) }`,
		"a fixture under HOME":            `func TestX(t *testing.T) { _ = os.WriteFile(filepath.Join(os.Getenv("HOME"), ".abcd/rules.json"), nil, 0o600) }`,
		"an expected output":              `func TestX(t *testing.T) { if !strings.Contains(out, "~/` + ".abcd" + `.noindex is a symlink") { t.Fatal("no") } }`,
		"an expected output in a table":   `var cases = []struct{ want string }{{"~/` + ".abcd" + `/rules.json"}}`,
		"a join inside a message":         `func TestX(t *testing.T, home string) { t.Errorf("%s", filepath.Join(home, ".abcd")) }`,
		"an error value a test builds":    `func f() error { return fmt.Errorf("~/` + ".abcd" + `.noindex is a symlink") }`,
		"a message helper's want":         `func TestX(t *testing.T) { wantAll(t, err, "~/` + ".abcd" + `.noindex/config.json") }`,
		"a fixture beside a named home":   `func TestX(t *testing.T) { root, home := virginHome(t); _ = root; _ = os.MkdirAll(filepath.Join(home, ".abcd"), 0o700) }`,
		"a result declared as the home":   "func virginHome(t *testing.T) (repoRoot, home string) { return \"\", \"\" }\nfunc TestX(t *testing.T) { repo, h := virginHome(t); _ = repo; _ = os.MkdirAll(filepath.Join(h, \".abcd\"), 0o700) }",
		"a fixture handed to gitutil.Run": `func TestX(t *testing.T) { gitutil.Run(".", "add", "~/` + ".abcd" + `.noindex/worktrees/x") }`,
		"a fixture handed to a run's Log": `func TestX(t *testing.T) { r := newRun(); r.Log("~/` + ".abcd" + `.noindex/runs/s1", 1) }`,
		"a message on an untyped t":       `func check(t *fakeT) { t.Fatalf("~/` + ".abcd" + `.noindex/x") }`,
		"a fixture from a test constant":  "const rel = \".abcd/trusted-roots\"\nfunc TestX(t *testing.T) { home := t.TempDir(); _ = os.WriteFile(filepath.Join(home, rel), nil, 0o600) }",
	}
	for label, body := range hostile {
		src := "package p\n\n" + body + "\n"
		if got := scan(src); len(got) != 1 {
			t.Errorf("%s: the test scanner reports %v, want exactly one spelling; it is not armed for that shape\n%s", label, got, src)
		}
	}
	benign := map[string]string{
		"a failure message":     `func TestX(t *testing.T) { t.Errorf("wrote under ~/` + ".abcd" + `.noindex: %v", 1) }`,
		"a concatenated fatal":  `func TestX(t *testing.T) { t.Fatal("the stop names ~/` + ".abcd" + `" + " and the new folder") }`,
		"a log line":            `func TestX(t *testing.T) { t.Logf("~/` + ".abcd" + `.noindex holds %d", 1) }`,
		"a skip":                `func BenchmarkX(b *testing.B) { b.Skip("no ~/` + ".abcd" + `.noindex here") }`,
		"a subtest's name":      `func TestX(t *testing.T) { t.Run("symlinked ~/` + ".abcd" + `.noindex", func(t *testing.T) {}) }`,
		"a helper's TB message": `func check(tb testing.TB) { tb.Fatalf("no ~/` + ".abcd" + `.noindex here") }`,
		"a fuzz skip":           `func FuzzX(f *testing.F) { f.Skip("no ~/` + ".abcd" + `.noindex here") }`,
		"a closure's message":   `var check = func(t *testing.T) { t.Error("~/` + ".abcd" + `.noindex") }`,
		"a root beside a home":  `func TestX(t *testing.T) { root, home := virginHome(t); _ = home; _ = os.MkdirAll(filepath.Join(root, ".abcd", "work"), 0o700) }`,
		"a root by its result":  "func virginHome(t *testing.T) (repoRoot, home string) { return \"\", \"\" }\nfunc TestX(t *testing.T) { repoRoot, _ := virginHome(t); _ = os.MkdirAll(filepath.Join(repoRoot, \".abcd\", \"work\"), 0o700) }",
		"the repository tier":   `func TestX(t *testing.T) { repo := t.TempDir(); _ = os.MkdirAll(filepath.Join(repo, ".abcd", "work"), 0o700) }`,
		"a fixture by resolver": `func TestX(t *testing.T) { home := t.TempDir(); _ = os.MkdirAll(abcdhome.Path(home, "lab"), 0o700) }`,
	}
	for label, body := range benign {
		src := "package p\n\n" + body + "\n"
		if got := scan(src); len(got) != 0 {
			t.Errorf("%s: the test scanner reports %v from a source that builds no home by hand\n%s", label, got, src)
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
