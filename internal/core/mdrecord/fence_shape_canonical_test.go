package mdrecord

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fenceShapeReaders names every non-test Go file outside this package that
// reads a fence character in one of the two shapes TestNoSecondFenceRule's
// delimiter scan cannot see (iss-2609251600029607), with the number of such
// reads it holds and the reason none of them is a second fence rule. The count
// is pinned for the reason fenceWriters pins its own: a toggle written into a
// file already on the list changes the count and fails until a reviewer reads
// the new reason. The default for a file this test names is to route it
// through Read.
var fenceShapeReaders = map[string]fenceWriter{
	"internal/core/positioning/check.go": {1, "emphasisRe strips inline emphasis and code markers from one tagline candidate, so a bolded tagline still reads as the tagline; it judges one string and tracks no lines"},
	"internal/termsafe/prose.go":         {2, "BlockText escapes a line-leading value's first byte when it would open a block (a backtick run, a tilde, a heading or list marker), so the rendered record keeps its text as prose; it judges one string and tracks no lines"},
	"internal/core/lifeboat/mdrender.go": {2, "escapeLeadingMarker escapes a value's first byte when it opens a block, a backtick run termsafe judges unbalanced or a tilde among the other markers, so a rendered field stays prose; it judges one string and tracks no lines"},
}

// fenceRunProbes are the runs a fence opens with: a pattern that matches one
// of them, and names a fence character itself, reads fences.
var fenceRunProbes = []string{"```", "~~~"}

// TestNoFenceRunReaderOutsideMdrecord is TestNoSecondFenceRule's second half
// (iss-2609251600029607). That scan is delimiter-only, so two spellings of a
// private toggle escaped it: a regexp literal whose pattern matches a fence
// RUN without spelling a three-character delimiter (`^ {0,3}(` + "`" + `+|~+)`,
// with the length checked afterwards), and a comparison of a line's FIRST
// byte against a backtick or a tilde (`ln[0] == '~'`, or a `switch ln[0]` with
// such a case). Both are read from the parsed source, so a comment or an
// unrelated string holding the characters is not a read. A pattern spelling a
// delimiter is the delimiter scan's and is not claimed twice, and a pattern
// that takes a run of letters whole as readily as a fence run is an alphabet
// that admits the fence characters, not a reader of them.
//
// Its reach ends where the source stops being a literal and a comparison stops
// being at the first byte: a pattern assembled at run time from non-constant
// parts, a comparison at an index other than a constant 0 (a byte read after an
// indent walk), and a one-character prefix test (`strings.HasPrefix(ln, "~")`,
// which the tree spells for code spans and home paths far more often than for
// fences) are outside it and are left to review.
func TestNoFenceRunReaderOutsideMdrecord(t *testing.T) {
	root := filepath.Join("..", "..", "..") // internal/core/mdrecord -> repository root
	var offenders []string
	seen := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == filepath.Join(root, "internal", "core", "mdrecord") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			reads, err := fenceShapeReads(path)
			if err != nil {
				return err
			}
			if len(reads) == 0 {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			seen[rel] = true
			w, ok := fenceShapeReaders[rel]
			switch {
			case !ok:
				offenders = append(offenders, fmt.Sprintf("%s (reads a fence character at %s and is not allowlisted)",
					rel, strings.Join(reads, ", ")))
			case w.count != len(reads):
				offenders = append(offenders, fmt.Sprintf("%s (reads a fence character %d time(s), at %s; the allowlist names %d)",
					rel, len(reads), strings.Join(reads, ", "), w.count))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("fence runs read outside mdrecord (route through mdrecord.Read, or allowlist with a reason and the count):\n  %s",
			strings.Join(offenders, "\n  "))
	}
	for rel := range fenceShapeReaders {
		if !seen[rel] {
			t.Errorf("%s is allowlisted but no longer reads a fence character; remove the entry", rel)
		}
	}
}

// TestFenceShapeReadsAreSeen pins the detector against the spellings the
// delimiter scan missed, and against the neighbours it must leave alone.
func TestFenceShapeReadsAreSeen(t *testing.T) {
	for name, tc := range map[string]struct {
		src  string
		want int
	}{
		"run regexp":           {"var r = regexp.MustCompile(`^ {0,3}(` + \"`\" + `+|~+)`)", 1},
		"tilde run regexp":     {"var r = regexp.MustCompile(`^\\s*~+`)", 1},
		"class run regexp":     {"var r = regexp.MustCompile(\"^[`~]+\")", 1},
		"escaped run regexp":   {"var r = regexp.MustCompile(`^\\x60+`)", 1},
		"first byte ==":        {"func f(s string) bool { return s[0] == '~' }", 1},
		"first byte !=":        {"func f(s string) bool { return '`' != s[0] }", 1},
		"switch on first byte": {"func f(s string) { switch s[0] { case '#', '`': } }", 1},
		"code span regexp":     {"var r = regexp.MustCompile(\"`[^`]+`\")", 0},
		"negated class":        {"var r = regexp.MustCompile(`[^~]`)", 0},
		"no fence char":        {"var r = regexp.MustCompile(`^.*$`)", 0},
		"identifier alphabet":  {"var r = regexp.MustCompile(`^[A-Za-z0-9~]+$`)", 0},
		"delimiter spelled":    {"var r = regexp.MustCompile(\"^ {0,3}(`{3,}|~{3,})\")", 0},
		"inner byte":           {"func f(s string, i int) bool { return s[i] == '~' }", 0},
		"other first byte":     {"func f(s string) bool { return s[0] == '#' }", 0},
		"comment only":         {"// ln[0] == '~' and regexp.MustCompile(`~+`)\nvar x = 1", 0},
		"run-time pattern":     {"func f(p string) { regexp.MustCompile(p + `+`) }", 0},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "x.go")
			if err := os.WriteFile(path, []byte("package x\n\n"+tc.src+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			reads, err := fenceShapeReads(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(reads) != tc.want {
				t.Errorf("reads = %v, want %d", reads, tc.want)
			}
		})
	}
}

// fenceShapeReads parses one Go file and returns the position of every fence
// read in the two shapes: a regexp literal matching a fence run, and a
// first-byte comparison against a fence character.
func fenceShapeReads(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var reads []string
	at := func(n ast.Node) {
		p := fset.Position(n.Pos())
		reads = append(reads, "line "+strconv.Itoa(p.Line))
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			switch callName(n) {
			case "regexp.MustCompile", "regexp.Compile", "regexp.MustCompilePOSIX", "regexp.CompilePOSIX":
				if len(n.Args) == 1 {
					if pat, ok := constString(n.Args[0]); ok && patternReadsFenceRun(pat) {
						at(n)
					}
				}
			}
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) &&
				(firstByte(n.X) && fenceChar(n.Y) || firstByte(n.Y) && fenceChar(n.X)) {
				at(n)
			}
		case *ast.SwitchStmt:
			if n.Tag == nil || !firstByte(n.Tag) {
				return true
			}
			for _, s := range n.Body.List {
				cc, ok := s.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, e := range cc.List {
					if fenceChar(e) {
						at(e)
					}
				}
			}
		}
		return true
	})
	return reads, nil
}

// callName is a call's `pkg.Func` spelling, or "" for any other callee.
func callName(c *ast.CallExpr) string {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name + "." + sel.Sel.Name
}

// constString folds a string literal, or a `+` concatenation of them, to its
// value; ok is false when any part is not a literal.
func constString(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.ParenExpr:
		return constString(e.X)
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, ok := constString(e.X)
		if !ok {
			return "", false
		}
		r, ok := constString(e.Y)
		return l + r, ok
	}
	return "", false
}

// patternReadsFenceRun reports whether a regexp pattern names a fence
// character and matches a whole fence run. A pattern spelling a delimiter is
// the delimiter scan's, and is not claimed twice.
func patternReadsFenceRun(pat string) bool {
	if fenceDelimiterRe.MatchString(pat) {
		return false
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return false
	}
	names := strings.ContainsAny(pat, "`~") || strings.Contains(strings.ToLower(pat), `\x60`) ||
		strings.Contains(strings.ToLower(pat), `\x7e`)
	if !names {
		return false
	}
	whole := func(run string) bool {
		loc := re.FindStringIndex(run)
		return loc != nil && loc[1]-loc[0] == len(run)
	}
	// A pattern that takes a run of plain letters whole as readily is a class
	// admitting the fence characters among others (an identifier's alphabet),
	// not a reader of fence runs.
	if whole("aaa") {
		return false
	}
	for _, run := range fenceRunProbes {
		if whole(run) {
			return true
		}
	}
	return false
}

// firstByte reports whether e indexes a value at the constant 0.
func firstByte(e ast.Expr) bool {
	ix, ok := e.(*ast.IndexExpr)
	if !ok {
		return false
	}
	lit, ok := ix.Index.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

// fenceChar reports whether e is a backtick or tilde character literal.
func fenceChar(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.CHAR {
		return false
	}
	r, _, _, err := strconv.UnquoteChar(strings.Trim(lit.Value, "'"), '\'')
	return err == nil && (r == '`' || r == '~')
}
