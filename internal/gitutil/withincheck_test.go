package gitutil_test

// withincheck_test.go holds the mark that keeps a read the drive's check
// makes itself from asking that check (gitutil.WithinCheck): what it does to
// an isolated read, and that the comparison's read is its one user.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheMarkKeepsAnIsolatedReadFromAskingTheCheck: an isolated read in a
// form that asks (a hash-object of a file, which reads it through its
// content filter) does not ask the check SetBeforeHookedGit set under
// WithinCheck's mark, and asks it once without the mark. Watched failing
// with the mark's test taken out of isolatedGitContextAsking: the marked
// read asked too.
func TestTheMarkKeepsAnIsolatedReadFromAskingTheCheck(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	args := []string{"hash-object", "--", "a.txt"}
	if !gitutil.LeavesContentProgramsOn(args) {
		t.Fatalf("%q must be a form that asks", args)
	}
	asked := countingCheck(t, nil)
	ctx := context.Background()
	if _, err := gitutil.RunLimitedContext(gitutil.WithinCheck(ctx), r.Root(), 1<<10, args...); err != nil || len(*asked) != 0 {
		t.Fatalf("a read under the mark asks nothing: %v, asked %q", err, *asked)
	}
	if _, err := gitutil.RunLimitedContext(ctx, r.Root(), 1<<10, args...); err != nil || len(*asked) != 1 {
		t.Fatalf("the same read without the mark asks once: %v, asked %q", err, *asked)
	}
}

// withinCheckUsers are the functions of the tree's non-test Go files that
// name gitutil.WithinCheck, as "<package dir>: <receiver>.<function>".
var withinCheckUsers = []string{"internal/core/implement/loop: gitEnv.readBy"}

// TestTheMarkIsSetOnlyByTheComparisonsRead: every use of gitutil.WithinCheck
// in the tree's non-test Go files, read from source, is in the drive's
// comparison's read (implement/loop's gitEnv.readBy), so no other caller
// marks a read of its own as one the check need not be asked about. Watched
// failing with a second use added at package level in gitutil.
func TestTheMarkIsSetOnlyByTheComparisonsRead(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	const self = "github.com/intentdriven/abcd/internal/gitutil"
	fset := token.NewFileSet()
	var users []string
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (strings.HasPrefix(n, ".") || n == "node_modules" || n == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		dir, _ := filepath.Rel(root, filepath.Dir(p))
		dir = filepath.ToSlash(dir)
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		inGitutil := dir == "internal/gitutil"
		imports := slices.ContainsFunc(f.Imports, func(s *ast.ImportSpec) bool {
			path, _ := strconv.Unquote(s.Path.Value)
			return path == self
		})
		if !inGitutil && !imports {
			return nil
		}
		for _, decl := range f.Decls {
			where := "package level"
			var declared *ast.Ident
			if fd, ok := decl.(*ast.FuncDecl); ok {
				declared = fd.Name
				where = fd.Name.Name
				if fd.Recv != nil && len(fd.Recv.List) > 0 {
					typ := fd.Recv.List[0].Type
					if star, ok := typ.(*ast.StarExpr); ok {
						typ = star.X
					}
					if id, ok := typ.(*ast.Ident); ok {
						where = id.Name + "." + where
					}
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == "WithinCheck" && !(inGitutil && id == declared) {
					users = append(users, dir+": "+where)
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(users)
	if !slices.Equal(slices.Compact(users), withinCheckUsers) {
		t.Fatalf("gitutil.WithinCheck is used by %q, not exactly %q: only the comparison's own read is marked as one the check is not asked about", users, withinCheckUsers)
	}
}
