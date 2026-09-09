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

	"github.com/intentdriven/abcd/internal/core/intent"
)

// The store an `intent` verb addresses is the CHECKOUT's intent store, wherever
// in the checkout the caller happens to stand — and outside a checkout there is
// no store to address at all. These are iss-2609091729516940's detectors for the
// intent family: the front door handed its working directory to the core as the
// repo root, verbatim, so `abcd intent` from a package directory reported
// drafts 0 against a checkout holding one, `abcd intent "<text>"` from there
// minted a SECOND store beneath it, and outside every repository the verb exited
// 0 and laid the full intent skeleton in whatever plain directory the caller
// stood in — every one of them reporting a repo-relative path that reads exactly
// like the checkout store's. A draft filed that way reaches no gate, no release
// cut and no reader, and its spec can never be closed against it.

// intentStoreFixture builds a git working tree holding one draft, with a
// package-shaped subdirectory two levels down, and chdirs into the root. HOME is
// redirected so nothing consults the developer's own home.
func intentStoreFixture(t *testing.T) (repo, sub string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo, sub
}

// intentDrafts runs the bare (read-only) status render and returns the drafts
// count it reports.
func intentDrafts(t *testing.T) int {
	t.Helper()
	var got struct {
		Buckets map[string]int `json:"buckets"`
	}
	out := runCLI(t, "intent", "--json")
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("intent --json: not JSON: %v\n%s", err, out)
	}
	return got.Buckets["drafts"]
}

// noStrayIntentStore fails when an intent store exists anywhere under dir.
func noStrayIntentStore(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil {
		t.Errorf("`intent` laid a second store at %s/.abcd — a draft filed there is invisible to every gate, every release cut, and every reader of the checkout's store", dir)
	}
}

// TestIntentFromSubdirectoryReadsTheCheckoutStore is the read half of the
// reproduction: the status board must count the CHECKOUT's drafts, not the
// zero the caller's own directory holds. Reporting drafts 0 against a populated
// checkout is the quiet half of this defect — it looks like an empty store
// rather than a missed one.
func TestIntentFromSubdirectoryReadsTheCheckoutStore(t *testing.T) {
	_, sub := intentStoreFixture(t)

	t.Chdir(sub)
	if n := intentDrafts(t); n != 1 {
		t.Errorf("`abcd intent` from a package directory reports drafts %d; the checkout holds 1 — the verb read a store that is not there", n)
	}
}

// TestIntentFromSubdirectoryFilesIntoTheCheckoutStore is the write half: the
// draft must land in the CHECKOUT's store, and no second store may appear under
// the subdirectory the caller happened to stand in.
func TestIntentFromSubdirectoryFilesIntoTheCheckoutStore(t *testing.T) {
	repo, sub := intentStoreFixture(t)

	t.Chdir(sub)
	var it struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	out := runCLI(t, "intent", "a draft filed from a package directory", "--json")
	if err := json.Unmarshal(out, &it); err != nil {
		t.Fatalf("intent --json: not JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(it.Path, intent.IntentsRelDir) {
		t.Errorf("the record path %q is not relative to the intent store", it.Path)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(it.Path))); err != nil {
		t.Errorf("the record %s reported at %q is not in the checkout's intent store: %v", it.ID, it.Path, err)
	}
	noStrayIntentStore(t, sub)
}

// TestIntentOutsideAnyRepositoryRefuses pins the no-repository ruling for both
// halves: with no checkout anywhere above, the read and the write REFUSE with
// exit 2 and nothing is laid down. Laying an intent skeleton in whatever
// directory the caller stood in is the failure this record is about, one
// directory further out — the draft would be committed by nothing, read by
// nothing, and its spec could never be closed against it.
func TestIntentOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)

	for _, args := range [][]string{
		{"intent"},
		{"intent", "an intent filed outside any repository"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Fatalf("`abcd %s` outside a repository succeeded; it must refuse:\n%s", strings.Join(args, " "), out)
		}
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("`abcd %s` outside a repository exited %d, want 2 (the operand-refusal code the verb reserves)", strings.Join(args, " "), code)
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("the refusal does not name the repository as the reason: %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(plain, ".abcd")); err == nil {
		t.Errorf("a refused run still laid an intent store at %s/.abcd — the refusal must write nothing", plain)
	}
}

// TestIntentRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state a
// repo root can be in. A directory carrying a .git that git will not answer for
// is repo-SHAPED, and a marker walk would hand it back as a root without
// checking either its shape or its owner (iss-2609090947359464). Resolving this
// front door must not make such a walk live: the verb refuses here, and says
// which of the two refusals it is.
func TestIntentRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(planted)

	out, err := runCLIErr(t, "intent", "a draft filed beneath a planted marker")
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(planted, filepath.FromSlash(intent.IntentsRelDir))); serr == nil {
		t.Errorf("the refusal still laid an intent store under the planted marker")
	}
}

// TestIntentFromTheCheckoutRootIsUnchanged is the anti-vacuity control: the
// resolution must be invisible where the caller already stood at the root. A fix
// that refused everywhere, or that resolved to some other tree, would pass the
// detectors above and fail here.
func TestIntentFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _ := intentStoreFixture(t)

	if n := intentDrafts(t); n != 1 {
		t.Errorf("`abcd intent` at the checkout root reports drafts %d, want 1", n)
	}
	var it struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	out := runCLI(t, "intent", "a draft filed from the checkout root", "--json")
	if err := json.Unmarshal(out, &it); err != nil {
		t.Fatalf("intent --json: not JSON: %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(it.Path)))
	if err != nil {
		t.Fatalf("the record reported at %q is not where it says: %v", it.Path, err)
	}
	if !strings.Contains(string(body), "id: "+it.ID) {
		t.Errorf("the record at %q does not carry the minted id %s:\n%s", it.Path, it.ID, body)
	}
}

// TestIntentBareStaysReadOnly is the contract the bare form carries and the one
// the resolution must not quietly break: resolving a root is a question, not a
// write. In a checkout with no intent store at all the status board renders and
// creates nothing — not the store, not a bucket, not a lock.
func TestIntentBareStaysReadOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	sub := filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, from := range []string{repo, sub} {
		t.Chdir(from)
		if n := intentDrafts(t); n != 0 {
			t.Errorf("bare `abcd intent` from %s reports drafts %d, want 0", from, n)
		}
		if _, err := os.Stat(filepath.Join(repo, ".abcd")); err == nil {
			t.Fatalf("bare `abcd intent` run from %s created %s/.abcd — the read-only render must write nothing", from, repo)
		}
		noStrayIntentStore(t, sub)
	}
}

// TestIntentReportsAStrayIntentStoreBelowTheCheckoutRoot: a store this defect
// already laid under a subdirectory holds drafts nobody will ever see again,
// because the verb now correctly addresses the checkout's. The resolution
// therefore SAYS the stray store is there, on stderr, rather than stepping over
// it in silence. It reports; it moves nothing.
func TestIntentReportsAStrayIntentStoreBelowTheCheckoutRoot(t *testing.T) {
	repo, sub := intentStoreFixture(t)
	stray := filepath.Join(sub, filepath.FromSlash(intent.IntentsRelDir), "drafts")
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(stray, "itd-2609090000000001-an-orphaned-draft.md")
	if err := os.WriteFile(orphan, []byte("---\nid: itd-2609090000000001\n---\n\n# draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	out, err := runCLIErr(t, "intent", "a draft filed beside a stray store")
	if err != nil {
		t.Fatalf("intent: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the run did not report the stray intent store under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, serr := os.Stat(orphan); serr != nil {
		t.Errorf("the report moved or removed the orphaned record: %v", serr)
	}
}

// TestIntentCallsNeverTakeTheRawWorkingDirectory is the static half, and the one
// that covers a front door this package has not grown yet: EVERY call into the
// intent core that takes a repo root must take the RESOLVED checkout root, never
// the caller's working directory. It reads both halves out of the source — which
// core functions take a repo root, and what each call site passes them — so a
// ninth intent front door written the old way fails here rather than shipping.
func TestIntentCallsNeverTakeTheRawWorkingDirectory(t *testing.T) {
	assertCoreRootsAreResolved(t, coreRootCheck{
		pkg:     "intent",
		coreDir: filepath.Join("..", "..", "core", "intent"),
		door:    "intentStoreRoot",
		// `spec close` reconciles the intent whose spec it closes, so it — and
		// only it — calls intent.Reconcile. That call site belongs to the SPEC
		// family's front door, which iss-2609091729516940 lists as its own half
		// and which resolves its own root under its own detector. The exemption
		// is keyed on the enclosing function, not on the core function's name:
		// an `abcd intent` verb that reached for Reconcile with a raw working
		// directory is still an offence here.
		exempt: map[string]string{"Reconcile": "newSpecCommand"},
	})
}

// coreRootCheck describes one family's static pass.
type coreRootCheck struct {
	// pkg is the core package's identifier as the surface imports it.
	pkg string
	// coreDir is that package's directory, relative to this one.
	coreDir string
	// door names the front-door resolver in the diagnostic.
	door string
	// exempt maps a core function name to the ONE enclosing surface function
	// allowed to call it with an unresolved root, because that call site belongs
	// to a different record family's front door.
	exempt map[string]string
}

// assertCoreRootsAreResolved is the shared static pass behind the intent and
// ideate detectors. It derives the rule from the code on both sides rather than
// from a hand-kept list: the core package says which of its exported functions
// take a repo root (the ones whose FIRST parameter is named repoRoot), and the
// surface package says what each call site hands them. Only `repoRoot` — the
// spelling every front-door resolver in this package answers in — is accepted.
func assertCoreRootsAreResolved(t *testing.T, c coreRootCheck) {
	t.Helper()
	const resolved = "repoRoot" // the one spelling: a store-root resolver's answer.

	takers := coreRootTakers(t, c.coreDir)
	if len(takers) == 0 {
		t.Fatalf("no exported %s function takes a repoRoot: this detector is inspecting the wrong tree (%s)", c.pkg, c.coreDir)
	}

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
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				enclosing := fn.Name.Name
				ast.Inspect(fn, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != c.pkg || !takers[sel.Sel.Name] || len(call.Args) == 0 {
						return true
					}
					if c.exempt[sel.Sel.Name] == enclosing {
						return true
					}
					seen++
					if arg, ok := call.Args[0].(*ast.Ident); ok && arg.Name == resolved {
						return true
					}
					pos := fset.Position(call.Pos())
					offenders = append(offenders, fmt.Sprintf("%s:%d: %s.%s(%s, …) in %s",
						filepath.Base(pos.Filename), pos.Line, c.pkg, sel.Sel.Name, exprText(call.Args[0]), enclosing))
					return true
				})
			}
		}
	}
	if seen == 0 {
		t.Fatalf("no %s core call taking a repo root was found in the surface package: this detector is inspecting the wrong tree", c.pkg)
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d %s call(s) take a repo root that is not the resolved checkout root (%q):\n  %s\n"+
			"every front door onto a repository-scoped record store resolves the checkout root first — see %s",
			len(offenders), c.pkg, resolved, strings.Join(offenders, "\n  "), c.door)
	}
}

// coreRootTakers reads a core package and returns the exported functions whose
// FIRST parameter is named repoRoot — the ones a caller can hand the wrong tree
// to. Deriving the set from the core rather than listing it here is what makes
// the pass survive a new core entry point: it is covered the moment it exists.
func coreRootTakers(t *testing.T, dir string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}
	takers := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !fn.Name.IsExported() || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
					continue
				}
				for _, name := range fn.Type.Params.List[0].Names {
					if name.Name == "repoRoot" {
						takers[fn.Name.Name] = true
					}
					break
				}
			}
		}
	}
	return takers
}
