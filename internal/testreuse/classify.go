package testreuse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Why a test reads what Go's test cache cannot see. Each is what the report
// says of a test the classifier sends to the always-rerun line.
const (
	whyProgram = "runs a program through go, make or a shell, whose reads the test cache never sees"
	whyTree    = "reaches the repository tree in a package that starts subprocesses, whose reads the test cache never sees"
	whyNetwork = "reads the network"
	whyClock   = "compares the clock with a fixed date"
)

// shellPrograms are the programs through which a test's subprocess can read
// any file.
var shellPrograms = []string{"go", "make", "bash", "sh", "zsh", "dash", "python", "python3", "perl", "ruby", "node"}

var (
	dateLiteral = regexp.MustCompile(`\b20\d\d-\d\d-\d\d\b`)
	// needsCheckout is a skip's message saying git could not answer for the tree.
	needsCheckout = regexp.MustCompile(`(?i)\bcheckout\b|\bgit (ls-files|grep|rev-parse)\b|\bnot a git\b`)
	// relativePath is a package pattern or path relative to the working
	// directory, handed to a program: ./cmd/abcd, ./..., ../x.
	relativePath = regexp.MustCompile(`^\.\.?/`)
)

// evidence is what one declaration does directly. Now and date are one
// declaration's alone: a helper that reads the clock and another that holds a
// date, in one test's reach, compare nothing.
type evidence struct {
	program, tree, network, loopback, now, date bool
}

// reached is what a test reaches through every declaration it names.
type reached struct {
	program, tree, network, loopback, clock bool
}

func (r *reached) merge(o reached) {
	r.program = r.program || o.program
	r.tree = r.tree || o.tree
	r.network = r.network || o.network
	r.loopback = r.loopback || o.loopback
	r.clock = r.clock || o.clock
}

func (r *reached) add(e evidence) {
	r.program = r.program || e.program
	r.tree = r.tree || e.tree
	r.network = r.network || e.network
	r.loopback = r.loopback || e.loopback
	r.clock = r.clock || (e.now && e.date)
}

// why names what sends a test to the always-rerun line, or "" for a test the
// classifier found nothing against.
func (r reached) why(spawns bool) string {
	switch {
	case r.program:
		return whyProgram
	case r.tree && spawns:
		return whyTree
	case r.network && !r.loopback:
		return whyNetwork
	case r.clock:
		return whyClock
	}
	return ""
}

// node is one package-level declaration of a test file: a function, a method
// (keyed with a leading dot), or a variable with what initialises or assigns
// it.
type node struct {
	direct   evidence
	borrowed reached
	refs     map[string]bool
}

// Verdict is the classifier's reading of one package's test files: a reason
// the whole package always reruns, or the tests that do, each with its reason.
type Verdict struct {
	Whole string
	Tests map[string]string
}

// ClassifyFiles reads the named test files of one package. spawns says that
// the package's tests can start a subprocess through the code they link: its
// own test files or anything in its import closure imports os/exec. helpers
// holds what the module's test-helper packages reach, by import path and
// symbol. A file that does not parse sends the whole package to the
// always-rerun line: what cannot be read cannot be shown safe.
func ClassifyFiles(paths []string, spawns bool, helpers map[string]map[string]reached) Verdict {
	files, err := parseFiles(paths)
	if err != nil {
		return Verdict{Whole: "a test file could not be read: " + err.Error()}
	}
	return classify(files, spawns, helpers)
}

func parseFiles(paths []string) ([]*ast.File, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		files = append(files, f)
	}
	return files, nil
}

// graph is a package's declarations and how they name each other.
type graph struct {
	nodes     map[string]*node
	tests     []string
	setUp     []string
	methodsOf map[string][]string
	topLevel  []string
}

func buildGraph(files []*ast.File, helpers map[string]map[string]reached) *graph {
	top, methods := declaredNames(files)
	nodes := map[string]*node{}
	get := func(key string) *node {
		n := nodes[key]
		if n == nil {
			n = &node{refs: map[string]bool{}}
			nodes[key] = n
		}
		return n
	}
	var tests, setUp, topLevel []string
	methodsOf := map[string][]string{}
	for _, f := range files {
		r := reader{file: f, imports: importNames(f), top: top, methods: methods, helpers: helpers}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				key := d.Name.Name
				if d.Recv != nil {
					key = "." + key
					if t := receiverType(d); t != "" {
						methodsOf[t] = append(methodsOf[t], key)
					}
				} else {
					topLevel = append(topLevel, key)
					switch {
					case key == "TestMain" || key == "init":
						setUp = append(setUp, key)
					case isTestName(key):
						tests = append(tests, key)
					}
				}
				n := get(key)
				if d.Body == nil {
					continue
				}
				r.read(d.Body, n)
				if d.Recv == nil && isTestName(key) {
					continue
				}
				// A package variable a set-up or helper function assigns carries
				// what that function did, so a test reading the variable reaches
				// it. A test's own assignment is left out: it would tie every
				// reader of a shared variable to that one test.
				ast.Inspect(d.Body, func(x ast.Node) bool {
					as, ok := x.(*ast.AssignStmt)
					if !ok {
						return true
					}
					for _, l := range as.Lhs {
						if id, ok := l.(*ast.Ident); ok && r.global(id) && id.Name != key {
							get(id.Name).refs[key] = true
						}
					}
					return true
				})
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					vs := s.(*ast.ValueSpec)
					for _, id := range vs.Names {
						n := get(id.Name)
						if vs.Type != nil {
							r.read(vs.Type, n)
						}
						for _, v := range vs.Values {
							r.read(v, n)
						}
					}
				}
			}
		}
	}
	return &graph{nodes: nodes, tests: tests, setUp: setUp, methodsOf: methodsOf, topLevel: topLevel}
}

// reach is everything a declaration comes to through what it names.
func (g *graph) reach(start string) (reached, map[string]bool) {
	var e reached
	seen := map[string]bool{}
	stack := []string{start}
	for len(stack) > 0 {
		key := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[key] {
			continue
		}
		seen[key] = true
		n := g.nodes[key]
		if n == nil {
			continue
		}
		e.add(n.direct)
		e.merge(n.borrowed)
		for ref := range n.refs {
			stack = append(stack, ref)
		}
	}
	return e, seen
}

// symbols is what each exported function and type of a helper package reaches:
// a type, all of its methods, since a test holds a value and calls any of them.
func (g *graph) symbols() map[string]reached {
	out := map[string]reached{}
	for _, name := range g.topLevel {
		out[name], _ = g.reach(name)
	}
	for t, methods := range g.methodsOf {
		var all reached
		for _, m := range methods {
			r, _ := g.reach(m)
			all.merge(r)
		}
		out[t] = all
	}
	return out
}

// receiverType is the name of a method's receiver type.
func receiverType(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	t := d.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func classify(files []*ast.File, spawns bool, helpers map[string]map[string]reached) Verdict {
	g := buildGraph(files, helpers)
	// source names one declaration that carries the evidence, for the report.
	source := func(seen map[string]bool, why string) string {
		var keys []string
		for key := range seen {
			if n := g.nodes[key]; n != nil {
				own := n.borrowed
				own.add(n.direct)
				if own.why(spawns) == why {
					keys = append(keys, strings.TrimPrefix(key, "."))
				}
			}
		}
		if len(keys) == 0 {
			return why
		}
		return why + " (" + slices.Min(keys) + ")"
	}

	v := Verdict{Tests: map[string]string{}}
	slices.Sort(g.setUp)
	for _, name := range slices.Compact(g.setUp) {
		// TestMain and init run before every test, so what their own bodies do
		// sends the whole package over. What they only name is left to the
		// tests that reach it: a TestMain names every stand-in the package has,
		// and its re-execution branches are not what a test run does. A walk to
		// the repository alone is how a TestMain remembers where it started,
		// and a test reaches that through the variable it assigns.
		own := g.nodes[name].borrowed
		own.add(g.nodes[name].direct)
		own.tree = false
		if why := own.why(spawns); why != "" {
			v.Whole = name + " " + why
			return v
		}
	}
	for _, name := range g.tests {
		e, seen := g.reach(name)
		if why := e.why(spawns); why != "" {
			v.Tests[name] = source(seen, why)
		}
	}
	return v
}

// declaredNames is every package-level function and variable name, and every
// method name, so a reference can tell a declaration from any other word.
func declaredNames(files []*ast.File) (top, methods map[string]bool) {
	top, methods = map[string]bool{}, map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil {
					methods[d.Name.Name] = true
				} else {
					top[d.Name.Name] = true
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					for _, id := range s.(*ast.ValueSpec).Names {
						top[id.Name] = true
					}
				}
			}
		}
	}
	return top, methods
}

// isTestName reports whether a top-level function is one go test runs.
func isTestName(name string) bool {
	for _, p := range []string{"Test", "Fuzz", "Example"} {
		if rest, ok := strings.CutPrefix(name, p); ok && (rest == "" || rest[0] < 'a' || rest[0] > 'z') {
			return true
		}
	}
	return false
}

// importNames maps a file's local names for its imports to their paths.
func importNames(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, im := range f.Imports {
		path, err := strconv.Unquote(im.Path.Value)
		if err != nil {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if im.Name != nil {
			name = im.Name.Name
		}
		out[name] = path
	}
	return out
}

// reader reads declarations of one file.
type reader struct {
	file         *ast.File
	imports      map[string]string
	top, methods map[string]bool
	helpers      map[string]map[string]reached
}

// global reports whether id names a package-level declaration: unresolved in
// its file, so declared in another of the package's, or resolved to this file's
// own scope. A local variable that shares a helper's name is neither.
func (r reader) global(id *ast.Ident) bool {
	if !r.top[id.Name] {
		return false
	}
	return id.Obj == nil || r.file.Scope.Lookup(id.Name) == id.Obj
}

// read adds what body does to n: its evidence and the declarations it names.
func (r reader) read(body ast.Node, n *node) {
	skip := map[*ast.Ident]bool{}
	ast.Inspect(body, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.KeyValueExpr:
			// A composite literal's key is a field name far more often than
			// a variable, and a field shares no reach with a helper of its name.
			if id, ok := x.Key.(*ast.Ident); ok {
				skip[id] = true
			}
		case *ast.SelectorExpr:
			skip[x.Sel] = true
			if pkg, name, ok := stdRef(x, r.imports); ok {
				readSelector(pkg, name, &n.direct)
				if sym, found := r.helpers[pkg][name]; found {
					n.borrowed.merge(sym)
				}
			} else if r.methods[x.Sel.Name] {
				n.refs["."+x.Sel.Name] = true
			}
		case *ast.Ident:
			if !skip[x] && r.global(x) {
				n.refs[x.Name] = true
			}
		case *ast.BasicLit:
			if s, ok := stringArg(x); ok {
				readLiteral(s, &n.direct)
			}
		case *ast.CallExpr:
			if pkg, fn, ok := stdRef(x.Fun, r.imports); ok {
				readCall(pkg, fn, x, &n.direct)
			}
			readSkip(x, &n.direct)
		}
		return true
	})
}

// stringArg is an expression's value when it is a string literal.
func stringArg(a ast.Expr) (string, bool) {
	lit, ok := a.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

// upward reports whether a path climbs out of the working directory.
func upward(s string) bool {
	return s == ".." || strings.HasPrefix(s, "../") || strings.Contains(s, "/../")
}

// stdRef resolves pkg.Name through the file's imports.
func stdRef(e ast.Expr, imports map[string]string) (pkg, name string, ok bool) {
	sel, isSel := e.(*ast.SelectorExpr)
	if !isSel {
		return "", "", false
	}
	id, isID := sel.X.(*ast.Ident)
	if !isID {
		return "", "", false
	}
	path, found := imports[id.Name]
	if !found {
		return "", "", false
	}
	return path, sel.Sel.Name, true
}

func readLiteral(s string, e *evidence) {
	if s == "--show-toplevel" || s == "--git-common-dir" {
		e.tree = true
	}
	if dateLiteral.MatchString(s) {
		e.date = true
	}
}

func readSelector(pkg, name string, e *evidence) {
	switch pkg {
	case "net/http":
		if name == "DefaultClient" || name == "DefaultTransport" {
			e.network = true
		}
	case "net/http/httptest":
		e.loopback = true
	}
}

func readCall(pkg, fn string, call *ast.CallExpr, e *evidence) {
	switch pkg {
	case "os/exec":
		readSpawn(fn, call, e)
	case "path/filepath":
		switch fn {
		case "Join":
			for _, a := range call.Args {
				if s, ok := stringArg(a); ok && upward(s) {
					e.tree = true
				}
			}
		case "Abs":
			if len(call.Args) == 1 {
				if s, ok := stringArg(call.Args[0]); ok && (s == "." || upward(s)) {
					e.tree = true
				}
			}
		}
	case "os":
		switch fn {
		case "Getwd":
			e.tree = true
		case "StartProcess":
			e.program = true
		case "Open", "ReadFile", "ReadDir", "Stat", "Lstat", "DirFS":
			if len(call.Args) > 0 {
				if s, ok := stringArg(call.Args[0]); ok && upward(s) {
					e.tree = true
				}
			}
		}
	case "syscall":
		if fn == "Exec" {
			e.program = true
		}
	case "runtime":
		if fn == "Caller" {
			e.tree = true
		}
	case "net":
		if strings.HasPrefix(fn, "Dial") || strings.HasPrefix(fn, "Lookup") {
			e.network = true
		}
		if fn == "Listen" {
			e.loopback = true
		}
	case "net/http":
		switch fn {
		case "Get", "Post", "Head", "PostForm":
			e.network = true
		}
	case "time":
		switch fn {
		case "Now":
			e.now = true
		case "Date":
			if len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.INT {
					e.date = true
				}
			}
		}
	}
}

// readSkip reads a test that skips outside a checkout: in the check directory,
// which holds no .git, it would pass by skipping, every time, and nothing would
// say so. It goes to the worktree, where git answers.
func readSkip(call *ast.CallExpr, e *evidence) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Skip" && sel.Sel.Name != "Skipf") || len(call.Args) == 0 {
		return
	}
	if s, ok := stringArg(call.Args[0]); ok && needsCheckout.MatchString(s) {
		e.tree = true
	}
}

// readSpawn reads exec.Command and exec.CommandContext: the program named, and
// a relative path handed to it.
func readSpawn(fn string, call *ast.CallExpr, e *evidence) {
	first := 0
	switch fn {
	case "Command":
	case "CommandContext":
		first = 1
	default:
		return
	}
	if first >= len(call.Args) {
		return
	}
	name, named := stringArg(call.Args[first])
	sub := ""
	if first+1 < len(call.Args) {
		sub, _ = stringArg(call.Args[first+1])
	}
	// `go env`, `go version` and `go telemetry` read the toolchain, not the tree.
	if named && slices.Contains(shellPrograms, name) && !(name == "go" && (sub == "env" || sub == "version" || sub == "telemetry")) {
		e.program = true
	}
	for _, a := range call.Args[first+1:] {
		if s, ok := stringArg(a); ok && relativePath.MatchString(s) {
			e.tree = true
		}
	}
}
