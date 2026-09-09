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

	"github.com/intentdriven/abcd/internal/core/decide"
)

// The store a `decide` mint writes into is the CHECKOUT's decision store,
// wherever in the checkout the caller happens to stand — and outside a checkout
// there is no store to write into at all. These are iss-2609091707224329's
// detectors: the front door handed its working directory to the core as the
// repo root, verbatim, so a mint from a subdirectory laid a second store under
// that subdirectory and a mint outside every repository laid one in whatever
// plain directory the caller stood in, both reporting a repo-relative path that
// reads exactly like the checkout store's. A decision filed that way reaches no
// gate, no release cut, and nobody looking for it — and the durable record is
// the family where a lost entry is least recoverable, because the decision it
// holds was never written anywhere else.

// decideStoreFixture builds a git working tree with a package-shaped
// subdirectory two levels down, and chdirs into the root. HOME is redirected so
// nothing consults the developer's own home.
func decideStoreFixture(t *testing.T) (repo, sub string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo, sub
}

// mintDecision runs the verb and returns the minted record's id and reported
// path.
func mintDecision(t *testing.T, title string) (id, path string) {
	t.Helper()
	var res struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	out := runCLI(t, "decide", title, "--json")
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decide --json: not JSON: %v\n%s", err, out)
	}
	return res.ID, res.Path
}

// noStrayDecisionStore fails when a decision store exists anywhere under dir.
func noStrayDecisionStore(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil {
		t.Errorf("`decide` minted a second decision store at %s/.abcd — a record filed there is invisible to every gate, every release cut, and every reader of the checkout's store", dir)
	}
}

// TestDecideFromSubdirectoryMintsIntoTheCheckoutStore is the write half of the
// reproduction: the record must land in the CHECKOUT's store, and no second
// store may appear under the subdirectory the caller happened to stand in.
func TestDecideFromSubdirectoryMintsIntoTheCheckoutStore(t *testing.T) {
	repo, sub := decideStoreFixture(t)

	t.Chdir(sub)
	id, rel := mintDecision(t, "a decision minted from a package directory")
	if !strings.HasPrefix(rel, decide.ADRsRelDir) {
		t.Errorf("the record path %q is not relative to the decision store", rel)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); err != nil {
		t.Errorf("the record %s reported at %q is not in the checkout's decision store: %v", id, rel, err)
	}
	noStrayDecisionStore(t, sub)
}

// TestDecideOutsideAnyRepositoryRefuses pins the no-repository ruling: with no
// checkout anywhere above, the mint REFUSES with exit 2 and writes nothing.
// Laying a decision store in whatever directory the caller stood in is the
// failure this record is about, one directory further out — the record would be
// committed by nothing and read by nothing.
func TestDecideOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)

	out, err := runCLIErr(t, "decide", "a decision minted outside any repository")
	if err == nil {
		t.Fatalf("`abcd decide` outside a repository succeeded; it must refuse:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("`abcd decide` outside a repository exited %d, want 2 (the operand-refusal code the verb reserves)", code)
	}
	if !strings.Contains(err.Error(), "repository") {
		t.Errorf("the refusal does not name the repository as the reason: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(plain, ".abcd")); serr == nil {
		t.Errorf("a refused mint still laid a decision store at %s/.abcd — the refusal must write nothing", plain)
	}
}

// TestDecideRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state a
// repo root can be in. A directory carrying a .git that git will not answer for
// is repo-SHAPED, and a marker walk would hand it back as a root without
// checking either its shape or its owner (iss-2609090947359464). Resolving this
// front door must not make such a walk live: the verb refuses here, and says
// which of the two refusals it is.
func TestDecideRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(planted)

	out, err := runCLIErr(t, "decide", "a decision minted beneath a planted marker")
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(planted, filepath.FromSlash(decide.ADRsRelDir))); serr == nil {
		t.Errorf("the refusal still laid a decision store under the planted marker")
	}
}

// TestDecideFromTheCheckoutRootIsUnchanged is the anti-vacuity control: the
// resolution must be invisible where the caller already stood at the root. A
// fix that refused everywhere, or that resolved to some other tree, would pass
// the detectors above and fail here.
func TestDecideFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _ := decideStoreFixture(t)

	id, rel := mintDecision(t, "a decision minted from the checkout root")
	if !strings.HasPrefix(rel, decide.ADRsRelDir) {
		t.Errorf("the record path %q is not relative to the decision store", rel)
	}
	body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("the record reported at %q is not where it says: %v", rel, err)
	}
	if !strings.Contains(string(body), "id: "+id) {
		t.Errorf("the record at %q does not carry the minted id %s:\n%s", rel, id, body)
	}
	if !strings.Contains(string(body), "status: proposed") {
		t.Errorf("the record at %q does not land `proposed`:\n%s", rel, body)
	}
}

// TestDecideReportsAStrayDecisionStoreBelowTheCheckoutRoot: a store this defect
// already laid under a subdirectory holds decisions nobody will ever see again,
// because the verb now correctly addresses the checkout's. The resolution
// therefore SAYS the stray store is there, on stderr, rather than stepping over
// it in silence. It reports; it moves nothing.
func TestDecideReportsAStrayDecisionStoreBelowTheCheckoutRoot(t *testing.T) {
	repo, sub := decideStoreFixture(t)
	stray := filepath.Join(sub, filepath.FromSlash(decide.ADRsRelDir))
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(stray, "2609090000000001-an-orphaned-decision.md")
	if err := os.WriteFile(orphan, []byte("---\nid: adr-2609090000000001\n---\n\n# ADR\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	out, err := runCLIErr(t, "decide", "a decision minted beside a stray store")
	if err != nil {
		t.Fatalf("decide: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the mint did not report the stray decision store under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, serr := os.Stat(orphan); serr != nil {
		t.Errorf("the report moved or removed the orphaned record: %v", serr)
	}
}

// TestDecideNeverMintsFromTheRawWorkingDirectory is the static half, and the one
// that covers a call site this package has not grown yet: EVERY decide.Create
// call in this package must take its repo root from the resolved checkout root,
// never from the caller's working directory. It reads the call sites out of the
// source, so a second front door written the old way fails here rather than
// shipping.
func TestDecideNeverMintsFromTheRawWorkingDirectory(t *testing.T) {
	const resolved = "repoRoot" // the one spelling: decideStoreRoot's answer.

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse the surface package: %v", err)
	}
	var offenders []string
	seen := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Create" {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "decide" || len(call.Args) == 0 {
					return true
				}
				seen++
				arg, ok := call.Args[0].(*ast.Ident)
				if ok && arg.Name == resolved {
					return true
				}
				pos := fset.Position(call.Pos())
				offenders = append(offenders, fmt.Sprintf("%s:%d: decide.Create(%s, …)",
					filepath.Base(pos.Filename), pos.Line, exprText(call.Args[0])))
				return true
			})
		}
	}
	if seen == 0 {
		t.Fatal("no decide.Create call found in the surface package: this detector is inspecting the wrong tree")
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d decide mint(s) take a repo root that is not the resolved checkout root (%q):\n  %s\n"+
			"every front door onto a repository-scoped record store resolves the checkout root first — see decideStoreRoot",
			len(offenders), resolved, strings.Join(offenders, "\n  "))
	}
}
