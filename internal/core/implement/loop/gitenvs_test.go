package loop

// gitenvs_test.go holds the table of the git commands a drive can run
// (driveGitCommands, gitenvs.go) to the code (iss-2611131308266006, fix
// round 2, ruling 2). The module's packages a drive's loop imports are
// type-checked from source with the standard library alone (go list names
// the files; go/parser and go/types do the rest), as the hooks' audit in
// internal/surface/cli/hook_git_audit_test.go does, and the functions a
// drive's entry (RunTerminal) reaches are walked: any use of a function,
// called or taken as a value, is an edge to it, a function literal belongs to
// the function that holds it, a use of a package variable is an edge to what
// its initialiser names, and a call of an interface method is an edge to that
// method on every module type that implements the interface. A walk that
// reaches the git helpers' package (internal/gitutil) from outside it stops
// there and names the helper used when it reaches a git command; a function
// outside it that builds a git command itself is named too. Every name found
// must be a row of the table, and every row a name found.
//
// NOT FOLLOWED (a static walk cannot follow every route a value takes): a
// function reached only through a function-typed struct field set outside
// the walked code (the drive's surface sets several, DriveHooksPath's caller
// among them), reflection, and a call the standard library makes back into
// the module through an interface. A program named by anything but the
// constant "git" is named in nonGitPrograms below with what it is, so a new
// one fails here until it is looked at.
//
// NOT CHECKED: the environment and the hooks flag of each row. The walk
// finds which constructors a drive reaches, not what environment each gives
// git or whether its command runs hooks; those are read by people, from each
// constructor's own code, when a row is added. The walk type-checks the
// module from source, so `go test -short` skips it; preflight and CI run the
// package without -short, and run it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// nonGitPrograms are the functions a drive reaches that start a program
// named by a value, not the constant "git", each with what that program is.
var nonGitPrograms = map[string]string{
	"credential.runKeychain":    "the system keychain tool the credential store names",
	"github.runContext":         "the GitHub CLI (gh)",
	"gitleaks.(execRunner).Run": "the gitleaks binary",
	"landing.gh":                "the GitHub CLI (gh)",
	"runner.(launcher).run":     "an agent harness the runner admits",
}

// TestEveryGitCommandADriveRunsIsInTheTable: the constructors of the git
// commands a drive's entry reaches are exactly the rows of driveGitCommands,
// so the hold iterates every environment a drive's git runs in. Watched
// failing: with a row taken out, and with a row naming no constructor added.
func TestEveryGitCommandADriveRunsIsInTheTable(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the module from source")
	}
	found, programs, err := driveGitConstructors("../../../..", "./internal/core/implement/loop", "internal/core/implement/loop.RunTerminal")
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, c := range driveGitCommands {
		rows = append(rows, c.name)
	}
	if missing, stale := tableDiff(found, rows); len(missing)+len(stale) > 0 {
		t.Fatalf("driveGitCommands differs from the git commands a drive reaches:\n  reached, not in the table: %v\n  in the table, not reached: %v", missing, stale)
	}
	for _, p := range programs {
		if _, ok := nonGitPrograms[p]; !ok {
			t.Errorf("%s starts a program named by a value: say what it is in nonGitPrograms, or add it to driveGitCommands if it is git", p)
		}
	}
}

// TestTheTableCheckSeesARowTakenOut: the comparison names a constructor the
// table lacks and a row no constructor answers.
func TestTheTableCheckSeesARowTakenOut(t *testing.T) {
	missing, stale := tableDiff([]string{"gitutil.Net", "gitutil.LocalGit"}, []string{"gitutil.Net", "gitutil.Gone"})
	if !slices.Equal(missing, []string{"gitutil.LocalGit"}) || !slices.Equal(stale, []string{"gitutil.Gone"}) {
		t.Fatalf("the comparison names both sides: %v %v", missing, stale)
	}
}

// TestEveryDistinctEnvironmentIsHeldOnce: the environments the hold
// iterates are the distinct ones the table names, each once, and one runs
// hooks when any of its rows does.
func TestEveryDistinctEnvironmentIsHeldOnce(t *testing.T) {
	envs := driveEnvs()
	seen := map[*gitEnv]bool{}
	for _, e := range envs {
		if seen[e] {
			t.Fatalf("%s is held twice", e.name)
		}
		seen[e] = true
	}
	for _, c := range driveGitCommands {
		if !seen[c.env] {
			t.Fatalf("%s's environment is not held", c.name)
		}
		if c.hooks && !c.env.hooks() {
			t.Fatalf("%s runs hooks, so its environment's hooks directory is held", c.name)
		}
	}
}

// TestTheWalkNamesAHelperThatReachesGitThroughACycle: two git helpers that
// call each other, one of which reaches a git command past the other, are
// both named when the drive uses both; the answer of the one asked while
// the cycle through it was open is not kept as "reaches none". Watched
// failing: the second helper was left out.
func TestTheWalkNamesAHelperThatReachesGitThroughACycle(t *testing.T) {
	const mod = "example.com/m"
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	loopPkg := types.NewPackage(mod+"/internal/core/implement/loop", "loop")
	gitPkg := types.NewPackage(mod+"/internal/gitutil", "gitutil")
	entry := types.NewFunc(token.NoPos, loopPkg, "Entry", sig)
	a := types.NewFunc(token.NoPos, gitPkg, "A", sig)
	b := types.NewFunc(token.NoPos, gitPkg, "B", sig)
	c := types.NewFunc(token.NoPos, gitPkg, "C", sig)
	w := &gitWalk{mod: mod, names: map[any]string{},
		edges:   map[any][]any{entry: {a, b}, a: {b, c}, b: {a}},
		gitExec: map[any]bool{c: true}, programs: map[any]bool{}}
	found, _, err := w.reach(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(found, []string{"gitutil.A", "gitutil.B"}) {
		t.Fatalf("both helpers reach a git command: %v", found)
	}
}

func tableDiff(found, rows []string) (missing, stale []string) {
	for _, f := range found {
		if !slices.Contains(rows, f) {
			missing = append(missing, f)
		}
	}
	for _, r := range rows {
		if !slices.Contains(found, r) {
			stale = append(stale, r)
		}
	}
	return missing, stale
}

// listed is the part of go list's record the walk reads.
type listed struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Module     *struct{ Main bool }
	ImportMap  map[string]string
}

type importerFn func(string) (*types.Package, error)

func (f importerFn) Import(path string) (*types.Package, error) { return f(path) }

// testGo is the go command of the toolchain that built this test.
func testGo(ctx context.Context) (string, error) {
	if goroot := runtime.GOROOT(); goroot != "" {
		return filepath.Join(goroot, "bin", "go"), nil
	}
	path, err := exec.LookPath("go")
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, path, "env", "GOVERSION").Output()
	if err != nil || strings.TrimSpace(string(out)) != runtime.Version() {
		return "", fmt.Errorf("the go on PATH is %q, not this test's %s (%v)", strings.TrimSpace(string(out)), runtime.Version(), err)
	}
	return path, nil
}

// driveGitConstructors type-checks pkg's module packages from source under
// root and walks from entry (module-relative pkg.Func): it returns the names
// of the git command constructors reached (sorted), and the functions that
// start a program named by a value.
func driveGitConstructors(root, pkg, entry string) ([]string, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	goBin, err := testGo(ctx)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, goBin, "list", "-deps", "-json=ImportPath,Dir,GoFiles,Module,ImportMap", pkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOPROXY=off", "GOFLAGS=")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	var pkgs []listed
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var p listed
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, p)
	}
	w := &gitWalk{edges: map[any][]any{}, gitExec: map[any]bool{}, programs: map[any]bool{}, names: map[any]string{}, funcs: map[string]*types.Func{}}
	fset := token.NewFileSet()
	checked := map[string]*types.Package{"unsafe": types.Unsafe}
	type modPkg struct {
		pkg   *types.Package
		files []*ast.File
		info  *types.Info
	}
	var mods []modPkg
	for _, lp := range pkgs {
		if lp.ImportPath == "unsafe" {
			continue
		}
		mine := lp.Module != nil && lp.Module.Main
		if mine && w.mod == "" {
			w.mod = lp.ImportPath[:strings.Index(lp.ImportPath+"/internal", "/internal")]
		}
		var files []*ast.File
		for _, name := range lp.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return nil, nil, err
			}
			files = append(files, f)
		}
		var errs []error
		conf := types.Config{
			Importer: importerFn(func(path string) (*types.Package, error) {
				if m, ok := lp.ImportMap[path]; ok {
					path = m
				}
				if p := checked[path]; p != nil {
					return p, nil
				}
				return nil, fmt.Errorf("%s imports %s, not yet checked", lp.ImportPath, path)
			}),
			IgnoreFuncBodies: !mine,
			Sizes:            types.SizesFor("gc", runtime.GOARCH),
			Error: func(err error) {
				if len(errs) < 5 {
					errs = append(errs, err)
				}
			},
		}
		var info *types.Info
		if mine {
			info = &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
		}
		p, _ := conf.Check(lp.ImportPath, fset, files, info)
		if len(errs) > 0 {
			return nil, nil, fmt.Errorf("type-checking %s: %w", lp.ImportPath, errors.Join(errs...))
		}
		checked[lp.ImportPath] = p
		if mine {
			mods = append(mods, modPkg{p, files, info})
		}
	}
	for _, m := range mods {
		w.add(m.pkg, m.files, m.info)
	}
	start := w.funcs[entry]
	if start == nil {
		return nil, nil, fmt.Errorf("no function %s", entry)
	}
	return w.reach(start)
}

// gitWalk is the call graph: nodes are *types.Func (an origin) and
// *types.Var (a package variable).
type gitWalk struct {
	mod      string
	edges    map[any][]any
	gitExec  map[any]bool // builds a git command itself
	programs map[any]bool // starts a program named by a value
	names    map[any]string
	funcs    map[string]*types.Func
	named    []*types.Named
	methods  []*types.Func
}

func (w *gitWalk) rel(path string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, w.mod), "/")
}

// name is a node's short name: the package's last element, then the
// function, a method as pkg.(T).M.
func (w *gitWalk) name(n any) string {
	if s, ok := w.names[n]; ok {
		return s
	}
	var s string
	switch x := n.(type) {
	case *types.Func:
		pkg := x.Pkg().Name()
		if recv := x.Type().(*types.Signature).Recv(); recv != nil {
			t := recv.Type()
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
			}
			if nt, ok := t.(*types.Named); ok {
				s = pkg + ".(" + nt.Obj().Name() + ")." + x.Name()
				break
			}
		}
		s = pkg + "." + x.Name()
	case *types.Var:
		s = x.Pkg().Name() + "." + x.Name()
	}
	w.names[n] = s
	return s
}

func (w *gitWalk) add(pkg *types.Package, files []*ast.File, info *types.Info) {
	for _, name := range pkg.Scope().Names() {
		if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok {
			if nt, ok := tn.Type().(*types.Named); ok {
				if _, iface := nt.Underlying().(*types.Interface); !iface {
					w.named = append(w.named, nt)
				}
			}
		}
	}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				fn, ok := info.Defs[d.Name].(*types.Func)
				if !ok || d.Body == nil {
					continue
				}
				if d.Recv == nil {
					w.funcs[w.rel(pkg.Path())+"."+fn.Name()] = fn
				} else {
					w.methods = append(w.methods, fn)
				}
				w.body(fn, d.Body, pkg, info)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, id := range vs.Names {
						if v, ok := info.Defs[id].(*types.Var); ok {
							for _, val := range vs.Values {
								w.body(v, val, pkg, info)
							}
						}
					}
				}
			}
		}
	}
}

// body adds node's edges from what n uses, and marks a git command or a
// program named by a value built there.
func (w *gitWalk) body(node any, n ast.Node, pkg *types.Package, info *types.Info) {
	ast.Inspect(n, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.Ident:
			switch obj := info.Uses[x].(type) {
			case *types.Func:
				w.edges[node] = append(w.edges[node], obj.Origin())
			case *types.Var:
				if obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope() {
					w.edges[node] = append(w.edges[node], obj)
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			fn, ok := info.Uses[sel.Sel].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "os/exec" || (fn.Name() != "Command" && fn.Name() != "CommandContext") {
				return true
			}
			i := 0
			if fn.Name() == "CommandContext" {
				i = 1
			}
			if len(x.Args) <= i {
				return true
			}
			tv := info.Types[x.Args[i]]
			switch {
			case tv.Value != nil && tv.Value.Kind() == constant.String && constant.StringVal(tv.Value) == "git":
				w.gitExec[node] = true
			case tv.Value == nil:
				w.programs[node] = true
			}
		}
		return true
	})
}

// successors are n's edges, an interface method's being every module
// method of that name on a type that implements the interface.
func (w *gitWalk) successors(n any) []any {
	out := slices.Clone(w.edges[n])
	for _, e := range w.edges[n] {
		fn, ok := e.(*types.Func)
		if !ok {
			continue
		}
		recv := fn.Type().(*types.Signature).Recv()
		if recv == nil {
			continue
		}
		iface, ok := recv.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		for _, m := range w.methods {
			if m.Name() != fn.Name() {
				continue
			}
			t := m.Type().(*types.Signature).Recv().Type()
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
			}
			if types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface) {
				out = append(out, m)
			}
		}
	}
	return out
}

func (w *gitWalk) inGitutil(n any) bool {
	fn, ok := n.(*types.Func)
	return ok && fn.Pkg() != nil && w.rel(fn.Pkg().Path()) == "internal/gitutil"
}

// reach walks from start: a gitutil function used from outside gitutil
// that reaches a git command is named and not walked into; a function
// outside it that builds one is named. Whether a function reaches a git
// command is a walk of its own over everything it reaches, so its answer is
// whole when it is kept: no answer is kept while a cycle through it is open.
func (w *gitWalk) reach(start any) ([]string, []string, error) {
	reaches := map[any]bool{}
	reachGit := func(n any) bool {
		if v, ok := reaches[n]; ok {
			return v
		}
		seen := map[any]bool{n: true}
		stack := []any{n}
		r := false
		for len(stack) > 0 && !r {
			m := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if w.gitExec[m] {
				r = true
				break
			}
			for _, s := range w.successors(m) {
				if !seen[s] {
					seen[s] = true
					stack = append(stack, s)
				}
			}
		}
		reaches[n] = r
		return r
	}
	found, programs := map[string]bool{}, map[string]bool{}
	seen := map[any]bool{start: true}
	queue := []any{start}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if w.gitExec[n] {
			found[w.name(n)] = true
		}
		if w.programs[n] {
			programs[w.name(n)] = true
		}
		for _, s := range w.successors(n) {
			if seen[s] {
				continue
			}
			seen[s] = true
			if w.inGitutil(s) && !w.inGitutil(n) {
				if reachGit(s) {
					found[w.name(s)] = true
				}
				continue
			}
			queue = append(queue, s)
		}
	}
	return sortedKeys(found), sortedKeys(programs), nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
