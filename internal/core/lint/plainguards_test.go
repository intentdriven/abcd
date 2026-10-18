package lint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Some tests assert only without the race detector (iss-2610171803053186).
// A cost guard that counts work, or bounds CPU time, measures the detector's
// instrumentation under -race, so it skips there (`if raceEnabled { t.Skip }`,
// directly or through a helper such as guard's assertWorkGrowthOn) or drops
// its bound there (`!raceEnabled && elapsed > bound`). The race lane runs those
// tests and asserts nothing, so on each leg a plain pass must still run
// them: they are what catches a quadratic blow-up in CI rather than only in a
// local preflight.
//
// The ubuntu leg's plain pass no longer reruns ./internal/... (the race lane
// runs every internal test, as on the macOS leg since iss-2610110938293273),
// so these tests run by name in their own step. The list is derived, not kept:
// this test reads every test file under internal/ and requires the step to name
// exactly the tests that reach a plain-only assertion and exactly the packages
// that hold them, so a guard added, renamed or removed fails here until the
// step says so.

// plainGuardSteps name the steps that run the plain-only guards, one per leg:
// the ubuntu leg's runs ahead of its race lane, so a guard fails fast, and
// the macOS leg's after its one race pass (iss-2610180542030806), so that
// pass still starts the leg's testing. Each must name the derived list.
var plainGuardSteps = []string{"Test (plain-only guards)", "Test (plain-only guards, macOS)"}

// raceFlag is the build constant the guards read, declared per package in a
// race_on_test.go and a race_off_test.go pair.
const raceFlag = "raceEnabled"

// raceFlagNonGuards are the uses of raceFlag that assert nothing and so need no
// plain pass, keyed by package directory (slash-separated, from the repository
// root) and function. Each is a reading the scanner cannot classify by shape.
var raceFlagNonGuards = map[string]string{
	// It picks an uninstrumented stub binary under -race to make every stubbed
	// turn cheaper; the tests that use it assert the same things either way.
	"internal/core/interview stubBinary": "chooses a cheaper stub binary",
}

// plainOnlyGuards scans the Go files in dir (one package directory) and
// returns the top-level tests that reach a plain-only assertion, sorted, and
// every reading of raceFlag it could classify neither as a guard nor as a
// declared non-guard, as "<what>: line <n>" strings. rel is dir's slash path
// from the repository root, the key raceFlagNonGuards uses.
//
// It fails closed: only a plain function in a test file is classified, so the
// flag read anywhere else (a method, a package-level declaration, a file that
// is not a test) is reported, as is a file behind a race build constraint that
// declares anything but the flag, and a function name a package declares
// twice. Functions are keyed by package name, so a directory's internal and
// external test packages never stand in for each other.
func plainOnlyGuards(t *testing.T, dir, rel string) (tests, unclassified []string) {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	type fn struct {
		pkg   string
		decl  *ast.FuncDecl
		guard bool
	}
	funcs := map[string]*fn{} // keyed by package name and function name
	var order []string
	odd := func(what string, pos token.Pos) {
		unclassified = append(unclassified, what+": line "+strconv.Itoa(fset.Position(pos).Line))
	}
	flagReads := func(n ast.Node, what string, skip map[*ast.Ident]bool) {
		ast.Inspect(n, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == raceFlag && !skip[id] {
				odd(what, id.Pos())
			}
			return true
		})
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raceTagged := regexp.MustCompile(`(?m)^//go:build .*\brace\b`).Match(src)
		isTest := strings.HasSuffix(name, "_test.go")
		if !isTest && !raceTagged && !strings.Contains(string(src), raceFlag) {
			continue
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatalf("%s: %v", rel+"/"+name, err)
		}
		if raceTagged {
			// The flag's own declaration, and nothing else, may sit behind
			// a race build constraint.
			for _, d := range f.Decls {
				g, ok := d.(*ast.GenDecl)
				if ok && g.Tok == token.CONST && len(g.Specs) == 1 {
					if v := g.Specs[0].(*ast.ValueSpec); len(v.Names) == 1 && v.Names[0].Name == raceFlag {
						continue
					}
				}
				odd(name+" declares more than "+raceFlag+" behind a race build constraint", d.Pos())
			}
			continue
		}
		if !isTest {
			flagReads(f, name+" (not a test file)", nil)
			continue
		}
		pkg := f.Name.Name
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || fd.Recv != nil {
				flagReads(d, name+" outside a plain function", nil)
				continue
			}
			key := pkg + "." + fd.Name.Name
			if _, dup := funcs[key]; dup {
				odd("package "+pkg+" declares "+fd.Name.Name+" twice", fd.Pos())
				continue
			}
			order = append(order, key)
			funcs[key] = &fn{pkg: pkg, decl: fd}
		}
	}

	// A function is a guard when its own body holds a plain-only assertion.
	for _, key := range order {
		f := funcs[key]
		name := f.decl.Name.Name
		seen := map[*ast.Ident]bool{}
		ast.Inspect(f.decl.Body, func(n ast.Node) bool {
			is, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			if id, ok := is.Cond.(*ast.Ident); ok && id.Name == raceFlag && skips(is.Body) {
				f.guard, seen[id] = true, true
			}
			for _, id := range negatedInConjunction(is.Cond) {
				f.guard, seen[id] = true, true
			}
			return true
		})
		if _, declared := raceFlagNonGuards[rel+" "+name]; !declared {
			flagReads(f.decl.Body, name, seen)
		}
	}

	// A function reaches a guard when it names one, called or passed as a
	// value; repeat until nothing changes.
	for changed := true; changed; {
		changed = false
		for _, key := range order {
			f := funcs[key]
			if f.guard {
				continue
			}
			ast.Inspect(f.decl.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name != f.decl.Name.Name {
					if g, ok := funcs[f.pkg+"."+id.Name]; ok && g.guard {
						f.guard, changed = true, true
						return false
					}
				}
				return !f.guard
			})
		}
	}
	for _, key := range order {
		if f := funcs[key]; f.guard && isTopLevelTest(f.decl) {
			tests = append(tests, f.decl.Name.Name)
		}
	}
	sort.Strings(tests)
	return tests, unclassified
}

// skips reports whether body calls a Skip, Skipf or SkipNow method.
func skips(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if s, ok := c.Fun.(*ast.SelectorExpr); ok && slices.Contains([]string{"Skip", "Skipf", "SkipNow"}, s.Sel.Name) {
				found = true
			}
		}
		return !found
	})
	return found
}

// negatedInConjunction returns each `!raceEnabled` that is an operand of an
// && chain in cond: the shape of a bound asserted only without the detector.
func negatedInConjunction(cond ast.Expr) []*ast.Ident {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.LAND {
		return nil
	}
	var out []*ast.Ident
	for _, side := range []ast.Expr{b.X, b.Y} {
		if u, ok := side.(*ast.UnaryExpr); ok && u.Op == token.NOT {
			if id, ok := u.X.(*ast.Ident); ok && id.Name == raceFlag {
				out = append(out, id)
			}
		}
		out = append(out, negatedInConjunction(side)...)
	}
	return out
}

// isTopLevelTest reports whether fd is a test go test runs by name.
func isTopLevelTest(fd *ast.FuncDecl) bool {
	name := fd.Name.Name
	if !strings.HasPrefix(name, "Test") || name == "TestMain" {
		return false
	}
	if rest := strings.TrimPrefix(name, "Test"); rest != "" && rest[0] >= 'a' && rest[0] <= 'z' {
		return false
	}
	return fd.Type.Params != nil && len(fd.Type.Params.List) == 1
}

// TestThePlainOnlyGuardScanReadsEachShape holds the scanner to the shapes it
// classifies, on a package written for it: a skip under the flag, a skip
// reached through a helper (called, and passed as a value), a bound dropped
// under the flag, a declared non-guard, and a reading of the flag it cannot
// classify.
func TestThePlainOnlyGuardScanReadsEachShape(t *testing.T) {
	dir := t.TempDir()
	src := `package p

import "testing"

func TestSkipsDirectly(t *testing.T) {
	if raceEnabled {
		t.Skip("plain only")
	}
}

func assertGrowth(t *testing.T) {
	if raceEnabled {
		t.Skipf("plain %s", "only")
	}
}

func TestSkipsThroughAHelper(t *testing.T) { assertGrowth(t) }

func shared(t *testing.T) { assertGrowth(t) }

func TestReachesItTwoCallsDown(t *testing.T) { t.Run("x", shared) }

func TestDropsABound(t *testing.T) {
	if elapsed := 1; !raceEnabled && elapsed > 0 {
		t.Error("slow")
	}
}

func TestRunsTheSameEitherWay(t *testing.T) {
	if raceEnabled {
		t.Log("instrumented")
	}
}

func stubBinary() string {
	if !raceEnabled {
		return "self"
	}
	return "plain"
}

func TestUsesTheStub(t *testing.T) { _ = stubBinary() }
`
	if err := os.WriteFile(filepath.Join(dir, "p_test.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	tests, unclassified := plainOnlyGuards(t, dir, "internal/core/interview")
	want := []string{"TestDropsABound", "TestReachesItTwoCallsDown", "TestSkipsDirectly", "TestSkipsThroughAHelper"}
	if !slices.Equal(tests, want) {
		t.Errorf("guards = %q; want %q", tests, want)
	}
	// TestRunsTheSameEitherWay reads the flag in a shape that is no guard;
	// stubBinary is declared for internal/core/interview, the rel given here.
	if want := []string{"TestRunsTheSameEitherWay: line 30"}; !slices.Equal(unclassified, want) {
		t.Errorf("unclassified = %q; want %q", unclassified, want)
	}
}

// TestThePlainOnlyGuardScanFailsClosed: a reading of the flag the scan does not
// classify is reported, never passed over. A method, a package-level
// declaration and a file that is not a test each read it; a file behind a race
// build constraint declares more than the flag; and the external test package
// declares a helper of the same name as the internal one's guard, which stands
// in for it in neither direction.
func TestThePlainOnlyGuardScanFailsClosed(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"race_on_test.go": "//go:build race\n\npackage p\n\nconst raceEnabled = true\n\nfunc extra() {}\n",
		"p_test.go": `package p

import "testing"

type harness struct{}

func (harness) assertLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("plain only")
	}
}

var skipUnderRace = func(t *testing.T) {
	if raceEnabled {
		t.Skip("plain only")
	}
}

func assertGrowth(t *testing.T) {
	if raceEnabled {
		t.Skip("plain only")
	}
}

func TestInternal(t *testing.T) { assertGrowth(t) }
`,
		"x_test.go": `package p_test

import "testing"

func assertGrowth(t *testing.T) {}

func TestExternal(t *testing.T) { assertGrowth(t) }
`,
		"p.go": "package p\n\nvar slow = !raceEnabled\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests, unclassified := plainOnlyGuards(t, dir, "internal/p")
	if want := []string{"TestInternal"}; !slices.Equal(tests, want) {
		t.Errorf("guards = %q; want %q: the external package's assertGrowth is no guard", tests, want)
	}
	sort.Strings(unclassified)
	want := []string{
		"p.go (not a test file): line 3",
		"p_test.go outside a plain function: line 8",
		"p_test.go outside a plain function: line 14",
		"race_on_test.go declares more than raceEnabled behind a race build constraint: line 7",
	}
	sort.Strings(want)
	if !slices.Equal(unclassified, want) {
		t.Errorf("unclassified = %q; want %q", unclassified, want)
	}
}

// TestEveryLegRunsEveryPlainOnlyGuard: each leg's plain-only guard step
// names exactly the tests under internal/ that assert something only without
// -race, and exactly the packages that hold them, so none of them leaves CI
// where a leg's only pass over internal/ is race-enabled.
func TestEveryLegRunsEveryPlainOnlyGuard(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	wantTests := map[string]bool{}
	wantPkgs := map[string]bool{}
	var unclassified []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if name := d.Name(); name == "testdata" || (strings.HasPrefix(name, ".") && path != root) {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		tests, odd := plainOnlyGuards(t, path, rel)
		for _, o := range odd {
			unclassified = append(unclassified, rel+" "+o)
		}
		if len(tests) > 0 {
			wantPkgs["./"+rel] = true
		}
		for _, n := range tests {
			wantTests[n] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range unclassified {
		t.Errorf("%s reads %s in a shape this scan cannot classify: write it as `if %s { t.Skip(...) }` "+
			"or `!%s && <bound>`, or declare it in raceFlagNonGuards with why it asserts nothing", u, raceFlag, raceFlag, raceFlag)
	}
	if len(wantTests) == 0 {
		t.Fatalf("no plain-only guard found under internal/; the scan or the tree changed shape")
	}

	const where = ".github/workflows/ci.yml job check"
	check, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}
	for _, name := range plainGuardSteps {
		step, ok := workflowStepBlock(check, name)
		if !ok {
			t.Errorf("%s: no %q step; the tests that assert only without -race (%d of them) run on "+
				"that leg in no pass that asserts them", where, name, len(wantTests))
			continue
		}
		cmd := oneGoTest(t, where+" step "+name, step)
		// A guard run under the detector asserts nothing, which is the gap
		// the step exists to close.
		if slices.ContainsFunc(strings.Fields(cmd), func(w string) bool { return w == "-race" || strings.HasPrefix(w, "-race=") }) {
			t.Errorf("%s step %q runs `%s` under -race; its tests assert only without the detector, "+
				"so there they assert nothing", where, name, cmd)
		}
		m := regexp.MustCompile(`\s-run '\^\(([A-Za-z0-9_|]+)\)\$'`).FindStringSubmatch(cmd)
		if m == nil {
			t.Errorf("%s step %q: `%s` carries no `-run '^(TestA|TestB)$'`", where, name, cmd)
			continue
		}
		gotTests := strings.Split(m[1], "|")
		var gotPkgs []string
		for _, w := range strings.Fields(cmd)[2:] {
			if strings.HasPrefix(w, "./") {
				gotPkgs = append(gotPkgs, w)
			}
		}
		missing, extra := setDiff(gotTests, wantTests)
		if len(missing) > 0 || len(extra) > 0 {
			t.Errorf("%s step %q names the wrong tests:\n  missing (assert only without -race, run nowhere): %q\n"+
				"  extra (no plain-only assertion): %q", where, name, missing, extra)
		}
		missing, extra = setDiff(gotPkgs, wantPkgs)
		if len(missing) > 0 || len(extra) > 0 {
			t.Errorf("%s step %q runs the wrong packages:\n  missing: %q\n  extra: %q", where, name, missing, extra)
		}
	}
}

// oneGoTest returns the one `go test` command in a step, failing the test when
// there is not exactly one.
func oneGoTest(t *testing.T, where, step string) string {
	t.Helper()
	var cmds []string
	for _, l := range strings.Split(step, "\n") {
		l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "run:"))
		if strings.HasPrefix(l, "go test ") {
			cmds = append(cmds, l)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("%s: want one `go test` command, found %d", where, len(cmds))
	}
	return cmds[0]
}

// setDiff returns the members of want that got lacks and the members of got
// that want lacks, each sorted.
func setDiff(got []string, want map[string]bool) (missing, extra []string) {
	have := map[string]bool{}
	for _, g := range got {
		have[g] = true
		if !want[g] {
			extra = append(extra, g)
		}
	}
	for w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
