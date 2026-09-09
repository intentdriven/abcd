package history

import (
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

// Brief invariant 15's last clause: "mechanically, only the history core package
// touches the store's path". Until this file existed the clause was held by
// convention and by nothing else — internal/core/lint/scribecontract_test.go
// scans the SCRIBE PROMPT for the same spellings, which is a different property
// (what one agent definition may name), and nothing read the source tree at all.
// adr-2609090717039680 rests its whole location argument on this test, twice, so
// the claim had to become a fact or stop being made.
//
// What it holds: no Go source outside this package spells a path into the
// session-transcript store as a string LITERAL. A package that needs the store
// calls Resolve, which is the one seam the store's creation, symlink discipline
// and legacy migration all sit behind; a package that spells the path itself has
// a second, unjudged door to the same bytes and goes stale in silence when the
// store moves — which it has already done once (adr-2609090717039680).
//
// Bounds, stated rather than left to be inferred:
//
//   - LITERALS, not comments. ahoy/apply.go and surface/cli/cli.go describe the
//     store in prose to say why they do NOT lay it out; that is the boundary
//     being observed, not breached. The AST walk sees only ast.BasicLit, so a
//     comment is invisible to it by construction rather than by an exception.
//   - Non-test files. The invariant governs what production code may reach; a
//     test builds a hermetic store under its own HOME by design, and holding it
//     to this rule would refuse the fixtures that prove the store's behaviour.
//   - `~/.abcd/history` is NOT a needle. That tree is ahoy's registry — index.json
//     and the per-repo meta.json — and ahoy owns it. Only the corpus moved out,
//     so the legacy spelling held here is the corpus leaf, `history/transcripts`.
var storePathNeedles = []string{
	// The user-level default and its opt-in per-repo pull-in, as location.go
	// spells them (userStoreRelPath, LocalStoreRelPath, LocalRootsRelPath).
	".abcd/transcripts",
	".work.local/transcripts",
	".abcd/local-transcript-roots",
	// The location the corpus was moved OUT of. A file still naming it declares
	// an access to the store just as much as one naming where it lives now.
	"history/transcripts",
}

// storeBoundaryExceptions are the literals outside this package that are allowed
// to name the store, each with the reason it is not a breach.
//
// internal/core/reading/include.go carries the store's local-tier path as the
// Detail of an EXCLUSION row: the cold-reading assembler names the path in order
// to deny it, and never resolves, reads or writes it. Encoding it as a shared
// constant would be worse, not better — it would give the assembler an import
// edge onto the transcript-store package, which is the structural separation
// invariant 15 asks the reading to keep. So the literal stays, declared here, and
// the declaration is exact: a second spelling, or a different one, fails.
var storeBoundaryExceptions = map[string][]string{
	filepath.Join("internal", "core", "reading", "include.go"): {".abcd/.work.local/transcripts"},
}

// storePathLiterals returns every string literal in src that names the store.
func storePathLiterals(t *testing.T, name, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		for _, needle := range storePathNeedles {
			if strings.Contains(v, needle) {
				out = append(out, v)
				return true
			}
		}
		return true
	})
	return out
}

// TestOnlyTheHistoryPackageNamesTheStorePath is invariant 15's boundary test.
func TestOnlyTheHistoryPackageNamesTheStorePath(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	thisPkg := filepath.Join("internal", "core", "history")

	seen := map[string][]string{}
	walked := 0
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// testdata holds fixtures, not shipped code, and a fixture that
				// names a path is data rather than a door into the store.
				if d.Name() == "testdata" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if strings.HasPrefix(rel, thisPkg+string(filepath.Separator)) {
				return nil
			}
			walked++
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if lits := storePathLiterals(t, rel, string(src)); len(lits) > 0 {
				seen[rel] = lits
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	// A walk that found nothing would pass this test while holding nothing, and
	// that is exactly how a boundary test rots: a renamed tree, a changed root.
	if walked < 100 {
		t.Fatalf("the walk read only %d non-test Go files under internal/ and cmd/; "+
			"it is not looking at the repository it is meant to hold", walked)
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		allowed, declared := storeBoundaryExceptions[f]
		if !declared {
			t.Errorf("%s names the session-transcript store's path as a string literal (%s); "+
				"invariant 15 reserves that path to internal/core/history — reach the store through "+
				"history.Resolve, or declare the literal in storeBoundaryExceptions with the reason it "+
				"is not a door into the store", f, strings.Join(seen[f], ", "))
			continue
		}
		got := append([]string(nil), seen[f]...)
		sort.Strings(got)
		want := append([]string(nil), allowed...)
		sort.Strings(want)
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s's declared exception is stale: it names %q and the file now carries %q; "+
				"an exception is exact, so a new or changed spelling is a new decision",
				f, want, got)
		}
	}
	for f := range storeBoundaryExceptions {
		if _, ok := seen[f]; !ok {
			t.Errorf("storeBoundaryExceptions still excuses %s, which no longer names the store; "+
				"remove the exception rather than leaving a licence nobody uses", f)
		}
	}
}

// TestStorePathBoundaryScannerIsArmed proves the scanner reports a literal and
// ignores a comment, so a pass above is the property holding rather than the
// scanner missing everything it walked.
func TestStorePathBoundaryScannerIsArmed(t *testing.T) {
	for _, needle := range storePathNeedles {
		src := "package p\n\nvar x = \"" + needle + "/aaaa/records\"\n"
		if got := storePathLiterals(t, "hostile.go", src); len(got) != 1 {
			t.Errorf("the scanner admits the literal %q; it is not armed for that spelling", needle)
		}
	}
	commented := "package p\n\n// the store lives at ~/.abcd/transcripts/<root-sha>/records/, laid out\n" +
		"// by internal/core/history and by nothing else.\nvar x = \"unrelated\"\n"
	if got := storePathLiterals(t, "commented.go", commented); len(got) != 0 {
		t.Errorf("the scanner reports %q from a comment; it judges literals, not prose", got)
	}
}
