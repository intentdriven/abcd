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
	// spawn is any exec.Command, whatever it names: Audits stops at one, and the
	// always-rerun verdict, which asks what the program can read, does not use it.
	spawn bool
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
				if d.Recv == nil && key == "TestMain" {
					r.readMainAssignments(d.Body, get)
					r.readPointerArgs(d.Body, get)
					continue
				}
				if d.Recv == nil && isTestName(key) {
					continue
				}
				// A package variable a helper assigns carries what that helper did,
				// so a test reading the variable reaches it. A test's own assignment
				// is left out: it would tie every reader of a shared variable to
				// that one test.
				ast.Inspect(d.Body, func(x ast.Node) bool {
					as, ok := x.(*ast.AssignStmt)
					if !ok {
						return true
					}
					for _, l := range as.Lhs {
						if id := rootIdent(l); id != nil && r.global(id) && id.Name != key {
							get(id.Name).refs[key] = true
						}
					}
					return true
				})
				r.readPointerArgs(d.Body, get)
			case *ast.GenDecl:
				if d.Tok == token.CONST {
					for name := range climbingConsts(d) {
						get(name).direct.tree = true
					}
				}
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					vs := s.(*ast.ValueSpec)
					for _, id := range vs.Names {
						n := get(id.Name)
						if vs.Type != nil {
							r.read(vs.Type, n)
							if r.isCmd(vs.Type) {
								n.direct.program = true
							}
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
				if d.Tok == token.CONST {
					for name := range climbingConsts(d) {
						top[name] = true
					}
				}
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					for _, id := range s.(*ast.ValueSpec).Names {
						// The blank name is not a variable: `var _ = f()` and
						// `_, err := g()` have nothing to do with each other.
						if id.Name != "_" {
							top[id.Name] = true
						}
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

// rootIdent is the variable an assignment's target belongs to: stub.bin is stub's.
func rootIdent(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x
		case *ast.SelectorExpr:
			e = x.X
		default:
			return nil
		}
	}
}

// callee is the declaration a call names, by the key it has in the graph.
func (r reader) callee(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		if r.global(f) {
			return f.Name
		}
	case *ast.SelectorExpr:
		if r.methods[f.Sel.Name] {
			return "." + f.Sel.Name
		}
	}
	return ""
}

// readPointerArgs gives a package variable handed to a function by its address
// what that function does: build(&bin) fills bin the way bin = build() does.
func (r reader) readPointerArgs(body *ast.BlockStmt, get func(string) *node) {
	ast.Inspect(body, func(x ast.Node) bool {
		call, ok := x.(*ast.CallExpr)
		if !ok {
			return true
		}
		callee := r.callee(call)
		if callee == "" {
			return true
		}
		for _, a := range call.Args {
			if addr, ok := a.(*ast.UnaryExpr); ok && addr.Op == token.AND {
				if id := rootIdent(addr.X); id != nil && r.global(id) {
					get(id.Name).refs[callee] = true
				}
			}
		}
		return true
	})
}

// climbingConsts is the names of a constant declaration whose values climb out of
// the working directory: a root held in a const is a walk to the repository.
func climbingConsts(d *ast.GenDecl) map[string]bool {
	out := map[string]bool{}
	for _, s := range d.Specs {
		vs := s.(*ast.ValueSpec)
		for i, id := range vs.Names {
			if i < len(vs.Values) {
				if v, ok := stringArg(vs.Values[i]); ok && upward(v) {
					out[id.Name] = true
				}
			}
		}
	}
	return out
}

// isCmd reports whether e is the type exec.Cmd itself, a command made here rather
// than one held from exec.Command.
func (r reader) isCmd(e ast.Expr) bool {
	pkg, name, ok := stdRef(e, r.imports)
	return ok && pkg == "os/exec" && name == "Cmd"
}

// readMainAssignments gives a package variable TestMain assigns what its value
// came from: the expression assigned, and every function TestMain hands the
// variable to, which is how a helper binary is built before the tests that run it.
// What else TestMain names is its own: its re-execution branches and stand-ins are
// no part of what a test reading one of its variables reaches.
func (r reader) readMainAssignments(body *ast.BlockStmt, get func(string) *node) {
	assigned := map[string]bool{}
	ast.Inspect(body, func(x ast.Node) bool {
		as, ok := x.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, l := range as.Lhs {
			id := rootIdent(l)
			if id == nil || !r.global(id) {
				continue
			}
			assigned[id.Name] = true
			value := as.Rhs[0]
			if len(as.Rhs) == len(as.Lhs) {
				value = as.Rhs[i]
			}
			r.read(value, get(id.Name))
		}
		return true
	})
	ast.Inspect(body, func(x ast.Node) bool {
		call, ok := x.(*ast.CallExpr)
		if !ok {
			return true
		}
		callee := r.callee(call)
		for _, a := range call.Args {
			if id, ok := a.(*ast.Ident); ok && assigned[id.Name] && r.global(id) && callee != "" {
				get(id.Name).refs[callee] = true
			}
		}
		return true
	})
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
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "new" && len(x.Args) == 1 && r.isCmd(x.Args[0]) {
				n.direct.program = true
			}
		case *ast.CompositeLit:
			// A command built by hand names its program however it likes.
			if r.isCmd(x.Type) {
				n.direct.program = true
			}
		case *ast.ValueSpec:
			if x.Type != nil && r.isCmd(x.Type) {
				n.direct.program = true
			}
		case *ast.GenDecl:
			if x.Tok == token.CONST && len(climbingConsts(x)) > 0 {
				n.direct.tree = true
			}
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
	if name == "ModuleRoot" && strings.HasSuffix(pkg, "/internal/fsutil") {
		e.tree = true
	}
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
		e.spawn = true
	case "CommandContext":
		e.spawn = true
		first = 1
	case "LookPath":
		// A program found on PATH goes on to run under whatever name holds it.
		if len(call.Args) == 1 {
			if name, ok := stringArg(call.Args[0]); ok && slices.Contains(shellPrograms, filepath.Base(name)) {
				e.program = true
			}
		}
		return
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
	program := filepath.Base(name)
	if named && slices.Contains(shellPrograms, program) && !(program == "go" && (sub == "env" || sub == "version" || sub == "telemetry")) {
		e.program = true
	}
	for _, a := range call.Args[first+1:] {
		if s, ok := stringArg(a); ok && relativePath.MatchString(s) {
			e.tree = true
		}
	}
}

// Audits names the tests in one package's test files that read the committed
// tree, the choice `make precheck` makes (cmd/precheck-audits): a test that
// reaches the repository by what reached.tree counts, other than through a
// helper that starts a process. That helper is the line between an audit and a
// test that only finds a script at the root to run it, a line the always-rerun
// classifier above has no need of. A test running a program in its own body is
// still named, since an audit may list files through one, and naming one too
// many costs seconds. A file that does not parse is an error: what cannot be
// read cannot be shown to hold no audit.
func Audits(paths []string) ([]string, error) {
	files, err := parseFiles(paths)
	if err != nil {
		return nil, err
	}
	g := buildGraph(files, nil)
	var names []string
	for _, name := range g.tests {
		if g.auditsTree(name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}

// auditsTree walks what a test names as reach does, stopping at a helper that
// starts a process.
func (g *graph) auditsTree(test string) bool {
	seen := map[string]bool{}
	stack := []string{test}
	for len(stack) > 0 {
		key := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[key] {
			continue
		}
		seen[key] = true
		n := g.nodes[key]
		if n == nil || (key != test && (n.direct.program || n.direct.spawn)) {
			continue
		}
		if n.direct.tree {
			return true
		}
		for ref := range n.refs {
			stack = append(stack, ref)
		}
	}
	return false
}
