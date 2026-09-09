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

	"github.com/intentdriven/abcd/internal/core/memory"
)

// The store `memory` addresses is the CHECKOUT's memory substrate, wherever in
// the checkout the caller happens to stand — and outside a checkout there is no
// substrate to address at all. These are iss-2609091729516940's detectors for
// the memory family. The read half is the quiet one: from a subdirectory bare
// `memory` reported "store not present" against a checkout whose store holds
// pages, which reads as a true fact about the repository and invites no second
// look. The write half is the loud one done silently: `memory ingest` from the
// same subdirectory exited 0 and laid a COMPLETE second substrate — pages,
// index, registry, log — under it, and outside every repository it laid one in
// whatever plain directory the caller stood in. `memory lint` did the same one
// tier down, writing its run-log tree beside a store that was never there and
// reporting a clean bill of health for pages it never read.

// memoryStoreFixture builds a git working tree carrying a one-page memory store,
// with a package-shaped subdirectory two levels down, and chdirs into the root.
// HOME is redirected so nothing consults the developer's own home.
func memoryStoreFixture(t *testing.T) (repo, sub string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMemoryPage(t, repo, "topic_auth_tokens.md")
	t.Chdir(repo)
	return repo, sub
}

// writeMemoryPage plants one page in the memory store under root.
func writeMemoryPage(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(memory.RelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nclass: topic\n---\n\n# Rotation\n\nRotate tokens every 24 hours.\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// memoryBoard runs bare `memory --json` and returns the rendered status.
func memoryBoard(t *testing.T) memory.BareStatus {
	t.Helper()
	var st memory.BareStatus
	out := runCLI(t, "memory", "--json")
	if err := json.Unmarshal(out, &st); err != nil {
		t.Fatalf("memory --json: not JSON: %v\n%s", err, out)
	}
	return st
}

// ingestOperands writes a source file and the distiller payload `memory ingest`
// reads through --pages-json, both outside any store.
func ingestOperands(t *testing.T, dir string) (src, pages string) {
	t.Helper()
	src = filepath.Join(dir, "article.txt")
	if err := os.WriteFile(src, []byte("Rotate tokens every 24 hours."), 0o644); err != nil {
		t.Fatal(err)
	}
	pages = filepath.Join(dir, "pages.json")
	payload := `[{"type":"topic","domain":"auth","slug":"tokens","body":"# Rotation\nRotate tokens every 24 hours."}]`
	if err := os.WriteFile(pages, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	return src, pages
}

// noStrayMemoryStore fails when a record store exists anywhere under dir.
func noStrayMemoryStore(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil {
		t.Errorf("`memory` laid a second store at %s/.abcd — pages filed there are read by no ask, no lint and no reader of the checkout's store", dir)
	}
}

// TestMemoryFromSubdirectoryReportsTheCheckoutStore is the read half of the
// reproduction: "store not present" against a checkout whose store is populated
// is a plausible fact about a repository, and nothing catches it.
func TestMemoryFromSubdirectoryReportsTheCheckoutStore(t *testing.T) {
	_, sub := memoryStoreFixture(t)

	t.Chdir(sub)
	st := memoryBoard(t)
	if !st.StorePresent {
		t.Errorf("`abcd memory` from a subdirectory reports the store absent; the checkout's store is right there")
	}
	if st.Pages != 1 {
		t.Errorf("`abcd memory` from a subdirectory reports %d page(s), want 1 — it addressed a store that is not the checkout's", st.Pages)
	}
}

// TestMemoryIngestFromSubdirectoryWritesIntoTheCheckoutStore is the write half,
// and the one that costs most: the unresolved door exited 0 and laid a complete
// second substrate under the subdirectory, reporting a page name that reads
// exactly like the checkout store's.
func TestMemoryIngestFromSubdirectoryWritesIntoTheCheckoutStore(t *testing.T) {
	repo, sub := memoryStoreFixture(t)
	src, pages := ingestOperands(t, repo)

	t.Chdir(sub)
	out, err := runCLIErr(t, "memory", "ingest", src, "--pages-json", pages)
	if err != nil {
		t.Fatalf("`abcd memory ingest` from a subdirectory: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(memory.RelDir), "topic_auth_tokens.md")); statErr != nil {
		t.Errorf("the ingested page is not in the checkout's memory store: %v", statErr)
	}
	noStrayMemoryStore(t, sub)
}

// TestMemoryAskFileBackFromSubdirectoryWritesIntoTheCheckoutStore covers the
// second write path: `ask` reads the store to answer, and with --file-back it
// writes the answer back as a page. Both halves have to be the checkout's — an
// answer synthesised from a store that is not there, filed into a store that is
// not there either, is two losses in one command.
func TestMemoryAskFileBackFromSubdirectoryWritesIntoTheCheckoutStore(t *testing.T) {
	repo, sub := memoryStoreFixture(t)
	// A filed-back answer cites the pages it was synthesised from, so the store
	// is seeded through a real ingest at the root rather than by hand.
	src, pages := ingestOperands(t, repo)
	if out, err := runCLIErr(t, "memory", "ingest", src, "--pages-json", pages); err != nil {
		t.Fatalf("seeding ingest: %v\n%s", err, out)
	}
	page := filepath.Join(repo, "answer.json")
	payload := `{"type":"topic","domain":"auth","slug":"answer","body":"# Answer\nTokens are rotated every 24 hours."}`
	if err := os.WriteFile(page, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	out, err := runCLIErr(t, "memory", "ask", "what does the auth rotation page say", "--file-back", "--page-json", page)
	if err != nil {
		t.Fatalf("`abcd memory ask --file-back` from a subdirectory: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(memory.RelDir), "topic_auth_answer.md")); statErr != nil {
		t.Errorf("the filed-back page is not in the checkout's memory store: %v", statErr)
	}
	noStrayMemoryStore(t, sub)
}

// TestMemoryLintFromSubdirectoryReadsAndReportsInsideTheCheckout covers the
// third write path: lint reads the whole store and writes its run-log tree. Run
// from a subdirectory it read a store that was not there, reported a clean bill
// of health for pages it never opened, and wrote the run log under the caller.
func TestMemoryLintFromSubdirectoryReadsAndReportsInsideTheCheckout(t *testing.T) {
	repo, sub := memoryStoreFixture(t)

	t.Chdir(sub)
	var res struct {
		StorePath string `json:"store_path"`
		ReportDir string `json:"report_dir"`
	}
	out, err := runCLIErr(t, "memory", "lint", "--json")
	if err != nil {
		t.Fatalf("`abcd memory lint` from a subdirectory: %v\n%s", err, out)
	}
	if jerr := json.Unmarshal(out, &res); jerr != nil {
		t.Fatalf("memory lint --json: not JSON: %v\n%s", jerr, out)
	}
	if res.StorePath != filepath.Join(repo, filepath.FromSlash(memory.RelDir)) {
		t.Errorf("lint from a subdirectory read %q, want the checkout's store", res.StorePath)
	}
	if !strings.HasPrefix(res.ReportDir, repo+string(filepath.Separator)) {
		t.Errorf("lint wrote its run log to %q, outside the checkout", res.ReportDir)
	}
	noStrayMemoryStore(t, sub)
}

// TestMemoryOutsideAnyRepositoryRefuses pins the no-repository ruling: with no
// checkout anywhere above, every memory verb REFUSES with exit 2 and writes
// nothing. Laying a substrate in whatever directory the caller stood in is the
// failure this record is about, one directory further out — the pages would sit
// outside any checkout, committed by nothing and read by nothing.
func TestMemoryOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	t.Chdir(plain)
	src, pages := ingestOperands(t, plain)

	for _, args := range [][]string{
		{"memory"},
		{"memory", "ingest", src, "--pages-json", pages},
		{"memory", "lint"},
	} {
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
		t.Errorf("a refused memory verb still laid a store at %s/.abcd — the refusal must write nothing", plain)
	}
}

// TestMemoryRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state a
// repo root can be in. A directory carrying a .git that git will not answer for
// is repo-SHAPED, and a marker walk would hand it back as a root without
// checking either its shape or its owner (iss-2609090947359464). Resolving this
// front door must not make such a walk live: the verb refuses here, and says
// which of the two refusals it is.
func TestMemoryRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(planted)

	out, err := runCLIErr(t, "memory")
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(planted, filepath.FromSlash(memory.RelDir))); serr == nil {
		t.Errorf("the refusal still laid a memory store under the planted marker")
	}
}

// TestMemoryFromTheCheckoutRootIsUnchanged is the anti-vacuity control, and for
// a read it has to assert a real count: a resolution that refused everywhere, or
// that resolved to some other tree, would pass every detector above and report
// "store not present" here — the very answer this record is about.
func TestMemoryFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _ := memoryStoreFixture(t)
	writeMemoryPage(t, repo, "topic_auth_rotation.md")

	st := memoryBoard(t)
	if !st.StorePresent {
		t.Fatalf("`abcd memory` at the checkout root reports the store absent")
	}
	if st.Pages != 2 {
		t.Fatalf("`abcd memory` at the checkout root reports %d page(s), want 2", st.Pages)
	}
}

// TestMemoryReportsAStrayMemoryStoreBelowTheCheckoutRoot: a substrate an
// unresolved door already laid under a subdirectory holds pages nobody will ever
// see again, because the verb now correctly addresses the checkout's. The
// resolution therefore SAYS the stray store is there, on stderr, rather than
// stepping over it in silence. It reports; it moves nothing.
func TestMemoryReportsAStrayMemoryStoreBelowTheCheckoutRoot(t *testing.T) {
	repo, sub := memoryStoreFixture(t)
	writeMemoryPage(t, sub, "topic_auth_orphan.md")
	orphan := filepath.Join(sub, filepath.FromSlash(memory.RelDir), "topic_auth_orphan.md")

	t.Chdir(sub)
	out, err := runCLIErr(t, "memory")
	if err != nil {
		t.Fatalf("memory: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the render did not report the stray memory store under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, statErr := os.Stat(orphan); statErr != nil {
		t.Errorf("the report moved or removed the orphaned page: %v", statErr)
	}
}

// TestMemoryNeverAddressesTheRawWorkingDirectory is the static half, and the one
// that covers a call site this package has not grown yet: EVERY memory request
// in this package must take its repo root from the resolved checkout root, never
// from the caller's working directory. Two shapes carry one, so both are read
// out of the source — the bare positional root and the RepoRoot field of a
// request literal — and a fifth verb written either way fails here rather than
// shipping.
func TestMemoryNeverAddressesTheRawWorkingDirectory(t *testing.T) {
	const resolved = "repoRoot" // the one spelling: memoryStoreRoot's answer.

	// Which memory functions take the root positionally is DERIVED from the core
	// package, so a verb wired tomorrow to one of them is covered without editing
	// this test — and so memory.Ask(req), whose first argument is a request and
	// not a root, is not mistaken for one.
	rootFirst := coreRootFirstFuncs(t, filepath.Join("..", "..", "core", "memory"))
	if len(rootFirst) == 0 {
		t.Fatal("no memory function takes a repo root first: this detector is reading the wrong package")
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
	note := func(n ast.Node, what string, arg ast.Expr) {
		if ident, ok := arg.(*ast.Ident); ok && ident.Name == resolved {
			return
		}
		pos := fset.Position(n.Pos())
		offenders = append(offenders, fmt.Sprintf("%s:%d: %s = %s",
			filepath.Base(pos.Filename), pos.Line, what, exprText(arg)))
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.CallExpr:
					// memory.Bare(root) and any sibling that takes the root
					// positionally.
					sel, ok := node.Fun.(*ast.SelectorExpr)
					if !ok || len(node.Args) == 0 {
						return true
					}
					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "memory" || !rootFirst[sel.Sel.Name] {
						return true
					}
					seen++
					note(node, "memory."+sel.Sel.Name+"(…)", node.Args[0])
				case *ast.CompositeLit:
					// memory.IngestRequest{RepoRoot: root}, AskRequest, LintRequest.
					sel, ok := node.Type.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "memory" {
						return true
					}
					for _, elt := range node.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok || key.Name != "RepoRoot" {
							continue
						}
						seen++
						note(kv, "memory."+sel.Sel.Name+"{RepoRoot: …}", kv.Value)
					}
				}
				return true
			})
		}
	}
	if seen == 0 {
		t.Fatal("no memory request found in the surface package: this detector is inspecting the wrong tree")
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d memory request(s) take a repo root that is not the resolved checkout root (%q):\n  %s\n"+
			"every front door onto a repository-scoped record store resolves the checkout root first — see memoryStoreRoot",
			len(offenders), resolved, strings.Join(offenders, "\n  "))
	}
}
