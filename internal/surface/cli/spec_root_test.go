package cli

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// The store `spec` addresses is the CHECKOUT's spec store, wherever in the
// checkout the caller happens to stand — and outside a checkout there is no
// store to address at all. These are iss-2609091729516940's detectors for the
// spec family: the front door handed its working directory to the core as the
// repo root, verbatim, so from a subdirectory bare `spec` reported
// `open 0 · closed 0` against a checkout holding eighty-one records and
// `spec close` refused with "spec spc-N not found" for a spec that is right
// there. Both read as ordinary statements about the repository, and both are
// false — which is what makes the READ half the dangerous one: a zero count
// invites no second look, where a refusal at least stops the caller.

// specStoreFixture builds a git working tree carrying one open spec, with a
// package-shaped subdirectory two levels down, and chdirs into the root. HOME is
// redirected so nothing consults the developer's own home.
func specStoreFixture(t *testing.T) (repo, sub string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSpecRecord(t, repo, "open", "spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	t.Chdir(repo)
	return repo, sub
}

// writeSpecRecord plants one spec record in a bucket of the store under root.
func writeSpecRecord(t *testing.T, root, bucket, name, body string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(spec.SpecsRelDir), bucket)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// specBoard runs bare `spec --json` and returns the rendered counts.
func specBoard(t *testing.T) specStatusView {
	t.Helper()
	var view specStatusView
	out := runCLI(t, "spec", "--json")
	if err := json.Unmarshal(out, &view); err != nil {
		t.Fatalf("spec --json: not JSON: %v\n%s", err, out)
	}
	return view
}

// TestSpecFromSubdirectoryReportsTheCheckoutStore is the read half of the
// reproduction: the counts must be the CHECKOUT's, not those of a store that is
// not under the subdirectory the caller happened to stand in. `open 0 · closed
// 0` against a populated checkout is a plausible fact about a repository, which
// is exactly why nothing catches it.
func TestSpecFromSubdirectoryReportsTheCheckoutStore(t *testing.T) {
	_, sub := specStoreFixture(t)

	t.Chdir(sub)
	view := specBoard(t)
	if view.Open != 1 || view.Closed != 0 {
		t.Errorf("`abcd spec` from a subdirectory reports open %d · closed %d, want open 1 · closed 0 — it addressed a store that is not the checkout's",
			view.Open, view.Closed)
	}
	if len(view.Specs) != 1 || view.Specs[0].ID != "spc-1" {
		t.Errorf("the render does not carry the checkout's spec: %+v", view.Specs)
	}
}

// TestSpecCloseFromSubdirectoryClosesInTheCheckoutStore is the write half. The
// unresolved door made this a REFUSAL rather than a misplaced write — the store
// it addressed was empty, so the spec "did not exist" — but the refusal names
// the record rather than the resolution, so the caller is told the wrong thing
// about their own checkout.
func TestSpecCloseFromSubdirectoryClosesInTheCheckoutStore(t *testing.T) {
	repo, sub := specStoreFixture(t)
	plantPlannedIntent(t, repo, "itd-10", "alpha", "spc-1")

	t.Chdir(sub)
	out, err := runCLIErr(t, "spec", "close", "spc-1")
	if err != nil {
		t.Fatalf("`abcd spec close` from a subdirectory: %v\n%s", err, out)
	}
	closed := filepath.Join(repo, filepath.FromSlash(spec.SpecsRelDir), spec.StatusClosed, "spc-1-alpha.md")
	if _, statErr := os.Stat(closed); statErr != nil {
		t.Errorf("the spec did not close in the checkout's store: %v", statErr)
	}
	shipped := filepath.Join(repo, ".abcd", "development", "intents", "shipped", "itd-10-alpha.md")
	if _, statErr := os.Stat(shipped); statErr != nil {
		t.Errorf("the linked intent did not ship in the checkout's store: %v", statErr)
	}
	noStraySpecStore(t, sub)
}

// plantPlannedIntent writes a planned intent already carrying both link sides
// and an impact — the shape `spec close` ships (see intent.Reconcile).
func plantPlannedIntent(t *testing.T, root, id, slug, specID string) {
	t.Helper()
	dir := filepath.Join(root, ".abcd", "development", "intents", "planned")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\nslug: " + slug + "\nspec_id: " + specID +
		"\nkind: standalone\nimpact: fix\n---\n# " + slug +
		"\n\n## Scope Conditions\n\nNONE\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n"
	if err := os.WriteFile(filepath.Join(dir, id+"-"+slug+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// noStraySpecStore fails when a record store exists anywhere under dir.
func noStraySpecStore(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil {
		t.Errorf("`spec` laid a second store at %s/.abcd — records filed there reach no gate, no release cut and no reader", dir)
	}
}

// TestSpecOutsideAnyRepositoryRefuses pins the no-repository ruling: with no
// checkout anywhere above, both halves REFUSE with exit 2 and write nothing.
// Reporting `open 0 · closed 0` in a plain directory is the same false fact one
// directory further out.
func TestSpecOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)

	for _, args := range [][]string{{"spec"}, {"spec", "close", "spc-1"}} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Fatalf("`abcd %s` outside a repository succeeded; it must refuse:\n%s", strings.Join(args, " "), out)
		}
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("`abcd %s` outside a repository exited %d, want 2 (the operand-refusal code the verb reserves)",
				strings.Join(args, " "), code)
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("the refusal does not name the repository as the reason: %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(plain, ".abcd")); err == nil {
		t.Errorf("a refused `spec` still laid a store at %s/.abcd — the refusal must write nothing", plain)
	}
}

// TestSpecRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state a repo
// root can be in. A directory carrying a .git that git will not answer for is
// repo-SHAPED, and a marker walk would hand it back as a root without checking
// either its shape or its owner (iss-2609090947359464). Resolving this front
// door must not make such a walk live: the verb refuses here, and says which of
// the two refusals it is.
func TestSpecRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(planted)

	out, err := runCLIErr(t, "spec")
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
}

// TestSpecFromTheCheckoutRootIsUnchanged is the anti-vacuity control, and for a
// read it has to assert a real count: a resolution that refused everywhere, or
// that resolved to some other tree, would pass every detector above and report
// `open 0 · closed 0` here — the very answer this record is about.
func TestSpecFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _ := specStoreFixture(t)
	writeSpecRecord(t, repo, "closed", "spc-2-beta.md",
		"---\nid: spc-2\nslug: beta\nintent: itd-11\n---\n# beta\n")

	view := specBoard(t)
	if view.Open != 1 || view.Closed != 1 {
		t.Fatalf("`abcd spec` at the checkout root reports open %d · closed %d, want open 1 · closed 1", view.Open, view.Closed)
	}
	ids := []string{}
	for _, sp := range view.Specs {
		ids = append(ids, sp.ID)
	}
	sort.Strings(ids)
	if got := strings.Join(ids, ","); got != "spc-1,spc-2" {
		t.Fatalf("the render does not carry both records: %v", got)
	}
}

// TestSpecReportsAStraySpecStoreBelowTheCheckoutRoot: a store an unresolved door
// already laid under a subdirectory holds records nobody will ever see again,
// because the verb now correctly addresses the checkout's. The resolution
// therefore SAYS the stray store is there, on stderr, rather than stepping over
// it in silence. It reports; it moves nothing.
func TestSpecReportsAStraySpecStoreBelowTheCheckoutRoot(t *testing.T) {
	repo, sub := specStoreFixture(t)
	writeSpecRecord(t, sub, "open", "spc-9-orphan.md",
		"---\nid: spc-9\nslug: orphan\nintent: itd-99\n---\n# orphan\n")
	orphan := filepath.Join(sub, filepath.FromSlash(spec.SpecsRelDir), "open", "spc-9-orphan.md")

	t.Chdir(sub)
	out, err := runCLIErr(t, "spec")
	if err != nil {
		t.Fatalf("spec: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the render did not report the stray spec store under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, statErr := os.Stat(orphan); statErr != nil {
		t.Errorf("the report moved or removed the orphaned record: %v", statErr)
	}
}

// TestSpecNeverAddressesTheRawWorkingDirectory is the static half, and the one
// that covers a call site this package has not grown yet: EVERY spec-store
// request in this package — the load bare `spec` renders and the reconcile
// `spec close` performs — must take its repo root from the resolved checkout
// root, never from the caller's working directory. It reads the call sites out
// of the source, so a second front door written the old way fails here rather
// than shipping.
func TestSpecNeverAddressesTheRawWorkingDirectory(t *testing.T) {
	const resolved = "repoRoot" // the one spelling: specStoreRoot's answer.

	// The set is DERIVED, not listed: every spec function whose first parameter
	// is a repo root is one this surface must resolve before it calls, so a verb
	// wired to spec.Close or spec.Create tomorrow is covered without editing this
	// test. intent.Reconcile is named explicitly because it lives in the intent
	// package but is reached ONLY through `spec close`, which makes the spec
	// store what it addresses; the rest of the intent surface is a different
	// front door with its own record.
	wanted := map[string]map[string]bool{
		"spec":   coreRootFirstFuncs(t, filepath.Join("..", "..", "core", "spec")),
		"intent": {"Reconcile": true},
	}
	if len(wanted["spec"]) == 0 {
		t.Fatal("no spec function takes a repo root first: this detector is reading the wrong package")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse the surface package: %v", err)
	}
	var offenders []string
	seen := map[string]int{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || !wanted[pkgIdent.Name][sel.Sel.Name] || len(call.Args) == 0 {
					return true
				}
				seen[pkgIdent.Name]++
				if arg, ok := call.Args[0].(*ast.Ident); ok && arg.Name == resolved {
					return true
				}
				pos := fset.Position(call.Pos())
				offenders = append(offenders, fmt.Sprintf("%s:%d: %s.%s(%s, …)",
					filepath.Base(pos.Filename), pos.Line, pkgIdent.Name, sel.Sel.Name, exprText(call.Args[0])))
				return true
			})
		}
	}
	for pkgName := range wanted {
		if seen[pkgName] == 0 {
			t.Fatalf("no %s call found in the surface package: this detector is inspecting the wrong tree", pkgName)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d spec-store request(s) take a repo root that is not the resolved checkout root (%q):\n  %s\n"+
			"every front door onto a repository-scoped record store resolves the checkout root first — see specStoreRoot",
			len(offenders), resolved, strings.Join(offenders, "\n  "))
	}
}

// coreRootFirstFuncs reads a core package and returns the exported functions
// whose FIRST parameter is the repo root — the calls a front door has to resolve
// before it makes. Deriving the set from the core rather than listing it here is
// what makes the static detectors outlive this change: a verb wired tomorrow to
// a function that already takes a root is covered the day it is written, with no
// edit to the test that would have to be remembered.
func coreRootFirstFuncs(t *testing.T, pkgDir string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", pkgDir, err)
	}
	found := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !fn.Name.IsExported() || fn.Type.Params == nil {
					continue
				}
				params := fn.Type.Params.List
				if len(params) == 0 || len(params[0].Names) == 0 {
					continue
				}
				if ident, ok := params[0].Type.(*ast.Ident); !ok || ident.Name != "string" {
					continue
				}
				switch params[0].Names[0].Name {
				case "repoRoot", "root":
					found[fn.Name.Name] = true
				}
			}
		}
	}
	return found
}
