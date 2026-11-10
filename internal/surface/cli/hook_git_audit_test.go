package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The hooks' git calls and their deadlines (iss-2611100548341325,
// dcn-2611100854404911). A hook runs on the host's clock: a git that hangs in
// one holds the hook until the host's own timeout, so every git call a hook
// reaches is to run under that hook's context. Hundreds of call sites in the
// module still use the context-free forms, so this audit is a ratchet, not a
// strict rule: it walks the calls from each hook verb's entry function, lists
// every context-free call it reaches that has a Context form, and compares
// that list with testdata/hook-git-baseline.txt. A reached call missing from
// the baseline fails (new unbound git in a hook), and so does a baseline entry
// no longer reached (the list only shrinks: take the entry out). The docs may
// say every hook git call is bound only once the baseline is empty.
//
// THE WALK. The module's packages are type-checked from source with the
// standard library alone (go list names the files; go/parser and go/types do
// the rest; golang.org/x/tools is not a dependency). The graph's nodes are
// the module's declared functions and methods, its function literals and its
// package-level variables. It over-counts, never under-counts, in four ways:
// any use of a function, called or taken as a value, is an edge to it; a
// function literal is an edge from the function that holds it; a call of an
// interface method is an edge to that method on every module type that
// implements the interface; and a use of a package variable is an edge to
// whatever its initialiser refers to, which is how a seam (var x = pkg.F) is
// followed. Every package's init function is a root besides the hook verbs.
// What it does not follow: calls the standard library or a dependency makes
// back into the module through an interface (a String or MarshalJSON method,
// a sort.Interface), and reflection. The cobra tree's constructor,
// NewRootCommand, is not followed from the dispatcher, since following it
// reaches every verb's closure; the hook verbs' closures are rooted directly.
//
// A FLAGGED CALL is a use of exec.Command, or of a package-level function of
// gitutil or localtier that has a sibling named <Name>Context. Each is keyed
// on the caller (module-relative package and function, a literal as its
// enclosing function with $lit) and the callee, never a line, so an edit that
// moves a call does not churn the baseline.

// hookGitBaseline is the ratchet's file, beside this test.
const hookGitBaseline = "testdata/hook-git-baseline.txt"

// hookGitBaselineHeader heads the baseline when the update rewrites it.
const hookGitBaselineHeader = `# Context-free git calls the hook verbs reach (iss-2611100548341325,
# dcn-2611100854404911). One entry per line, sorted: <caller> -> <callee>.
# TestHookGitCallsMatchBaseline (hook_git_audit_test.go) fails on a reached
# call not listed here and on a listed call no longer reached.
# Entries may only be removed, never added: bind the call to the hook's
# context instead. After binding calls, drop the stale entries with
#   ABCD_UPDATE_BASELINE=1 go test -run TestHookGitCallsMatchBaseline ./internal/surface/cli
# which removes stale entries only and still fails on a new one.
`

// hookEntry names one hook verb's entry: the cobra command literal whose Use
// begins with use, inside the top-level function fn of package cli. Its
// Run-family fields are the roots.
type hookEntry struct{ fn, use string }

// hookEntries are the verbs hooks/*.sh run.
var hookEntries = []hookEntry{
	{"newHookCommand", "prompt-router"},
	{"newHookCommand", "prompt-router-reset"},
	{"newHookCommand", "question-answered"},
	{"newHookCommand", "session-end"},
	{"newHookCommand", "session-start"},
	{"newGuardHookCommand", "hook"},
	{"newSubagentStopCommand", "subagent-stop"},
	{"newPermissionRequestCommand", "permission-request"},
	{"newHookStopCommand", "stop"},
	{"newHookStopFailureCommand", "stop-failure"},
	{"newDogfoodBinaryCommand", "dogfood-binary"},
}

// auditSpec says what one walk starts from and skips.
type auditSpec struct {
	entryPkg string      // module-relative package holding the entries
	entries  []hookEntry // command literals to root
	funcs    []string    // plain functions of entryPkg to root
	skip     []string    // functions of entryPkg never followed
}

var hookAuditSpec = auditSpec{
	entryPkg: "internal/surface/cli",
	entries:  hookEntries,
	funcs:    []string{"run"},
	skip:     []string{"NewRootCommand"},
}

func TestHookGitCallsMatchBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the module from source")
	}
	root := moduleRoot(t)
	reached, err := hookGitAudit(root, "./cmd/abcd", hookAuditSpec)
	if err != nil {
		t.Fatal(err)
	}
	base, err := readHookGitBaseline(hookGitBaseline)
	if err != nil {
		t.Fatal(err)
	}
	added, stale := diffHookGitBaseline(reached, base)
	if len(stale) > 0 && os.Getenv("ABCD_UPDATE_BASELINE") == "1" {
		if err := writeHookGitBaseline(hookGitBaseline, keepReached(base, reached)); err != nil {
			t.Fatal(err)
		}
		t.Logf("removed %d stale entries from %s", len(stale), hookGitBaseline)
		stale = nil
	}
	var msg strings.Builder
	if len(added) > 0 {
		fmt.Fprintf(&msg, "%d context-free git call(s) a hook reaches are not in %s; bind each to the hook's context (never add it to the baseline):\n", len(added), hookGitBaseline)
		for _, k := range added {
			fmt.Fprintf(&msg, "  %s\n", k)
		}
	}
	if len(stale) > 0 {
		fmt.Fprintf(&msg, "%d entr(ies) of %s are no longer reached; remove them (ABCD_UPDATE_BASELINE=1 does):\n", len(stale), hookGitBaseline)
		for _, k := range stale {
			fmt.Fprintf(&msg, "  %s\n", k)
		}
	}
	if msg.Len() > 0 {
		t.Fatal(msg.String())
	}
}

// TestHookGitAuditFollowsAndFlags plants calls in a module of its own and
// proves the walk reaches them through a helper, an interface, a seam and a
// literal, flags the context-free forms that have a Context sibling and a
// call from outside gitutil of one that reaches such a form (Wrapped), and
// leaves an unreached caller and a git-free helper (Plain) out.
func TestHookGitAuditFollowsAndFlags(t *testing.T) {
	got, err := hookGitAudit(writeAuditFixture(t), "./cmd/fx", auditSpec{
		entryPkg: "cmd/fx",
		entries:  []hookEntry{{"newHook", "hook"}},
		skip:     []string{"tree"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"cmd/fx.(*disk).Read -> exec.Command",
		"cmd/fx.helper -> gitutil.Run",
		"cmd/fx.loadPlain -> localtier.Untracked",
		"cmd/fx.newHook$lit -> gitutil.Run",
		"cmd/fx.newHook$lit -> gitutil.Wrapped",
		"cmd/fx.viaLit$lit -> exec.Command",
		"internal/gitutil.Wrapped -> gitutil.Run",
		"internal/gitutil.init -> exec.Command",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("reached:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// TestHookGitBaselineRefusesNewAndStale: a reached call the baseline lacks is
// new, and a baseline entry nothing reaches is stale; both fail the audit.
func TestHookGitBaselineRefusesNewAndStale(t *testing.T) {
	reached := []string{"a.f -> gitutil.Run", "b.g -> exec.Command"}
	base := []string{"a.f -> gitutil.Run", "c.h -> localtier.Untracked"}
	added, stale := diffHookGitBaseline(reached, base)
	if strings.Join(added, ",") != "b.g -> exec.Command" {
		t.Errorf("added = %q, want the planted call", added)
	}
	if strings.Join(stale, ",") != "c.h -> localtier.Untracked" {
		t.Errorf("stale = %q, want the entry nothing reaches", stale)
	}
	if a, s := diffHookGitBaseline(reached, reached); len(a)+len(s) != 0 {
		t.Errorf("an exact baseline reads as %q new and %q stale", a, s)
	}
}

// TestHookGitBaselineFileIsSortedAndUnique: the reader refuses a baseline out
// of order or with a repeat, so the file's diffs stay one line per call.
func TestHookGitBaselineFileIsSortedAndUnique(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"unsorted":  "# h\nb.g -> exec.Command\na.f -> gitutil.Run\n",
		"repeated":  "a.f -> gitutil.Run\na.f -> gitutil.Run\n",
		"shapeless": "a.f gitutil.Run\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readHookGitBaseline(p); err == nil {
			t.Errorf("%s baseline read without error", name)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			t.Fatalf("no go.mod above %s", dir)
		}
	}
}

// auditFixture is a module of its own: a fake gitutil and localtier, and a
// command package whose hook verb reaches context-free calls four ways.
var auditFixture = map[string]string{
	"go.mod": "module example.com/fx\n\ngo 1.22\n",
	"internal/gitutil/gitutil.go": `package gitutil

import (
	"context"
	"os/exec"
)

func init() { _ = exec.Command("true") }

func Run(args ...string) string                                 { return RunContext(context.Background(), args...) }
func RunContext(ctx context.Context, args ...string) string      { _ = exec.CommandContext(ctx, "git"); return "" }
func Plain() string                                              { return "" }
func Wrapped() string                                            { return Run("x") }
`,
	"internal/localtier/localtier.go": `package localtier

import "context"

func Untracked(p string) bool                             { return UntrackedContext(context.Background(), p) }
func UntrackedContext(ctx context.Context, p string) bool { return ctx.Err() == nil && p != "" }
`,
	"cmd/fx/main.go": `package main

import (
	"context"
	"os/exec"

	"example.com/fx/internal/gitutil"
	"example.com/fx/internal/localtier"
)

type Command struct {
	Use  string
	RunE func() error
}

type reader interface{ Read() string }

type disk struct{}

func (*disk) Read() string { _ = exec.Command("git"); return "" }

type quiet struct{}

func (quiet) Read() string { return gitutil.Plain() }

var load = loadPlain

func loadPlain() bool { return localtier.Untracked("x") }

func helper() string { return gitutil.Run("status") }

func viaLit() func() { return func() { _ = exec.Command("git") } }

func unreached() string { return gitutil.Run("log") }

func tree() *Command { return &Command{Use: "other", RunE: func() error { unreached(); return nil }} }

func newHook(r reader) *Command {
	return &Command{Use: "hook [flags]", RunE: func() error {
		_ = gitutil.Run("rev-parse")
		_ = gitutil.RunContext(context.Background())
		_ = gitutil.Plain()
		_ = gitutil.Wrapped()
		helper()
		r.Read()
		load()
		viaLit()()
		return nil
	}}
}

func main() { _ = newHook(&disk{}); _ = tree() }
`,
}

func writeAuditFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range auditFixture {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// diffHookGitBaseline returns the reached calls the baseline lacks and the
// baseline entries no longer reached, each sorted.
func diffHookGitBaseline(reached, base []string) (added, stale []string) {
	r, b := stringSet(reached), stringSet(base)
	for k := range r {
		if !b[k] {
			added = append(added, k)
		}
	}
	for k := range b {
		if !r[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(added)
	sort.Strings(stale)
	return added, stale
}

func stringSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, k := range list {
		m[k] = true
	}
	return m
}

// keepReached is the baseline with its stale entries taken out, in order:
// the update removes, and never adds.
func keepReached(base, reached []string) []string {
	r := stringSet(reached)
	var out []string
	for _, k := range base {
		if r[k] {
			out = append(out, k)
		}
	}
	return out
}

// readHookGitBaseline reads the entries, skipping blank and # lines, and
// refuses an entry not shaped <caller> -> <callee>, out of order or repeated.
func readHookGitBaseline(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		caller, callee, ok := strings.Cut(line, " -> ")
		if !ok || caller == "" || callee == "" || strings.ContainsAny(caller+callee, " \t") {
			return nil, fmt.Errorf("%s:%d: %q is not <caller> -> <callee>", path, n, line)
		}
		if len(out) > 0 && line <= out[len(out)-1] {
			return nil, fmt.Errorf("%s:%d: %q is out of order or repeated; keep the file sorted, one entry per call", path, n, line)
		}
		out = append(out, line)
	}
	return out, sc.Err()
}

func writeHookGitBaseline(path string, entries []string) error {
	var b strings.Builder
	b.WriteString(hookGitBaselineHeader)
	for _, k := range entries {
		b.WriteString(k + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// listedPkg is the part of go list's record the audit reads.
type listedPkg struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Module     *struct {
		Path string
		Main bool
	}
	ImportMap map[string]string
}

// goListDeps names mainPkg's packages, each after its dependencies, without
// cgo (the walk needs types, not a build) and without the network.
func goListDeps(root, mainPkg string) ([]listedPkg, error) {
	goBin := "go"
	if g := os.Getenv("GOROOT"); g != "" {
		if p := filepath.Join(g, "bin", "go"); fileIsThere(p) {
			goBin = p
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, goBin, "list", "-deps", "-json=ImportPath,Dir,GoFiles,Module,ImportMap", mainPkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOPROXY=off", "GOTOOLCHAIN=local")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	var pkgs []listedPkg
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var p listedPkg
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			return pkgs, nil
		} else if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
}

func fileIsThere(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// hookGitAudit type-checks mainPkg's packages from source (bodies only for
// the main module's), builds the call graph and returns the flagged calls the
// spec's roots reach, as sorted "<caller> -> <callee>" keys.
func hookGitAudit(root, mainPkg string, spec auditSpec) ([]string, error) {
	listed, err := goListDeps(root, mainPkg)
	if err != nil {
		return nil, err
	}
	g := &callGraph{
		edges: map[any][]any{}, flags: map[any][]string{}, ifaceUses: map[any][]*types.Func{},
		names: map[any]string{}, pkgs: map[string]*pkgDecls{}, methodSets: map[types.Type]*types.MethodSet{},
		resolved: map[*types.Func][]any{},
	}
	for _, lp := range listed {
		if lp.Module != nil && lp.Module.Main {
			g.mod = lp.Module.Path
			break
		}
	}
	if g.mod == "" {
		return nil, errors.New("go list named no package of the main module")
	}
	fset := token.NewFileSet()
	checked := map[string]*types.Package{"unsafe": types.Unsafe}
	sizes := types.SizesFor("gc", runtime.GOARCH)
	for _, lp := range listed {
		if lp.ImportPath == "unsafe" {
			continue
		}
		mine := lp.Module != nil && lp.Module.Main
		files := make([]*ast.File, 0, len(lp.GoFiles))
		for _, name := range lp.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		var errs []error
		conf := types.Config{
			Importer: importerFunc(func(path string) (*types.Package, error) {
				if m, ok := lp.ImportMap[path]; ok {
					path = m
				}
				if p := checked[path]; p != nil {
					return p, nil
				}
				return nil, fmt.Errorf("%s imports %s, which go list did not name before it", lp.ImportPath, path)
			}),
			IgnoreFuncBodies: !mine,
			Sizes:            sizes,
			Error: func(err error) {
				if len(errs) < 5 {
					errs = append(errs, err)
				}
			},
		}
		var info *types.Info
		if mine {
			info = &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
		}
		pkg, _ := conf.Check(lp.ImportPath, fset, files, info)
		if len(errs) > 0 {
			return nil, fmt.Errorf("type-checking %s: %w", lp.ImportPath, errors.Join(errs...))
		}
		checked[lp.ImportPath] = pkg
		if mine {
			g.addPackage(pkg, files, info)
		}
	}
	return g.reach(spec)
}

// pkgDecls keeps what the entry finder needs of one module package.
type pkgDecls struct {
	funcs map[string]*ast.FuncDecl // top-level functions, not methods
	info  *types.Info
}

// callGraph's nodes are *types.Func (an origin), *ast.FuncLit and
// *types.Var (a package-level variable).
type callGraph struct {
	mod        string
	edges      map[any][]any
	flags      map[any][]string
	ifaceUses  map[any][]*types.Func
	names      map[any]string
	inits      []any
	named      []*types.Named
	pkgs       map[string]*pkgDecls
	methodSets map[types.Type]*types.MethodSet
	resolved   map[*types.Func][]any
}

func (g *callGraph) rel(path string) string {
	if path == g.mod {
		return "."
	}
	return strings.TrimPrefix(path, g.mod+"/")
}

func (g *callGraph) inModule(path string) bool {
	return path == g.mod || strings.HasPrefix(path, g.mod+"/")
}

func (g *callGraph) funcName(fn *types.Func) string {
	sig := fn.Type().(*types.Signature)
	name := g.rel(fn.Pkg().Path()) + "."
	if r := sig.Recv(); r != nil {
		t, star := r.Type(), ""
		if p, ok := t.(*types.Pointer); ok {
			t, star = p.Elem(), "*"
		}
		tn := t.String()
		if n, ok := t.(*types.Named); ok {
			tn = n.Obj().Name()
		}
		name += "(" + star + tn + ")."
	}
	return strings.ReplaceAll(name+fn.Name(), " ", "")
}

func (g *callGraph) addPackage(pkg *types.Package, files []*ast.File, info *types.Info) {
	pd := &pkgDecls{funcs: map[string]*ast.FuncDecl{}, info: info}
	g.pkgs[g.rel(pkg.Path())] = pd
	for _, name := range pkg.Scope().Names() {
		if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && !tn.IsAlias() {
			if n, ok := tn.Type().(*types.Named); ok && !types.IsInterface(n) {
				g.named = append(g.named, n)
			}
		}
	}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				fn, ok := info.Defs[d.Name].(*types.Func)
				if !ok {
					continue
				}
				name := g.funcName(fn)
				g.names[fn] = name
				if d.Recv == nil {
					pd.funcs[d.Name.Name] = d
					if d.Name.Name == "init" {
						g.inits = append(g.inits, fn)
					}
				}
				if d.Body != nil {
					g.walk(fn, d.Body, info, name)
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					vs, ok := s.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, id := range vs.Names {
						v, ok := info.Defs[id].(*types.Var)
						if !ok {
							continue
						}
						name := g.rel(pkg.Path()) + "." + id.Name
						g.names[v] = name
						for _, e := range vs.Values {
							g.walk(v, e, info, name)
						}
					}
				}
			}
		}
	}
}

// walk records n's uses under node; a literal is a node of its own, named
// for the declaration that holds it.
func (g *callGraph) walk(node any, n ast.Node, info *types.Info, name string) {
	ast.Inspect(n, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.FuncLit:
			lit := strings.TrimSuffix(name, "$lit") + "$lit"
			g.names[x] = lit
			g.edges[node] = append(g.edges[node], x)
			g.walk(x, x.Body, info, lit)
			return false
		case *ast.Ident:
			g.use(node, info.Uses[x])
		}
		return true
	})
}

func (g *callGraph) use(node any, obj types.Object) {
	switch o := obj.(type) {
	case *types.Func:
		o = o.Origin()
		if k := g.flagKey(o); k != "" {
			g.flags[node] = append(g.flags[node], k)
		}
		if r := o.Type().(*types.Signature).Recv(); r != nil && types.IsInterface(r.Type()) {
			g.ifaceUses[node] = append(g.ifaceUses[node], o)
			return
		}
		if o.Pkg() != nil && g.inModule(o.Pkg().Path()) {
			g.edges[node] = append(g.edges[node], o)
		}
	case *types.Var:
		if o.Pkg() != nil && g.inModule(o.Pkg().Path()) && o.Parent() == o.Pkg().Scope() {
			g.edges[node] = append(g.edges[node], o)
		}
	}
}

// flagKey names a context-free call the audit lists, or "" for any other.
func (g *callGraph) flagKey(fn *types.Func) string {
	p := fn.Pkg()
	if p == nil || fn.Type().(*types.Signature).Recv() != nil {
		return ""
	}
	switch p.Path() {
	case "os/exec":
		if fn.Name() == "Command" {
			return "exec.Command"
		}
	case g.mod + "/internal/gitutil", g.mod + "/internal/localtier":
		if _, ok := p.Scope().Lookup(fn.Name() + "Context").(*types.Func); ok {
			return p.Name() + "." + fn.Name()
		}
	}
	return ""
}

// implementations is m on every module type, or its pointer, that implements
// m's interface; a generic type is matched on the method's name alone.
func (g *callGraph) implementations(m *types.Func) []any {
	if out, ok := g.resolved[m]; ok {
		return out
	}
	iface, _ := m.Type().(*types.Signature).Recv().Type().Underlying().(*types.Interface)
	var out []any
	for _, n := range g.named {
		for _, t := range []types.Type{n, types.NewPointer(n)} {
			ms := g.methodSets[t]
			if ms == nil {
				ms = types.NewMethodSet(t)
				g.methodSets[t] = ms
			}
			sel := ms.Lookup(m.Pkg(), m.Name())
			if sel == nil {
				continue
			}
			if n.TypeParams().Len() == 0 && iface != nil && !types.Implements(t, iface) {
				continue
			}
			out = append(out, sel.Obj().(*types.Func).Origin())
			break
		}
	}
	g.resolved[m] = out
	return out
}

// reach walks from the spec's roots and returns the flagged calls reached.
func (g *callGraph) reach(spec auditSpec) ([]string, error) {
	pd := g.pkgs[spec.entryPkg]
	if pd == nil {
		return nil, fmt.Errorf("the entry package %s was not loaded", spec.entryPkg)
	}
	skip := map[any]bool{}
	for _, name := range spec.skip {
		d := pd.funcs[name]
		if d == nil {
			return nil, fmt.Errorf("the skipped function %s.%s is not declared", spec.entryPkg, name)
		}
		skip[pd.info.Defs[d.Name]] = true
	}
	roots := append([]any(nil), g.inits...)
	for _, name := range spec.funcs {
		d := pd.funcs[name]
		if d == nil {
			return nil, fmt.Errorf("the root %s.%s is not declared", spec.entryPkg, name)
		}
		roots = append(roots, pd.info.Defs[d.Name])
	}
	for _, e := range spec.entries {
		got, err := entryRoots(pd, e)
		if err != nil {
			return nil, err
		}
		roots = append(roots, got...)
	}
	seen := map[any]bool{}
	keys := map[string]bool{}
	for queue := roots; len(queue) > 0; {
		n := queue[0]
		queue = queue[1:]
		if seen[n] || skip[n] {
			continue
		}
		seen[n] = true
		for _, k := range g.flags[n] {
			keys[g.names[n]+" -> "+k] = true
		}
		queue = append(queue, g.edges[n]...)
		for _, m := range g.ifaceUses[n] {
			queue = append(queue, g.implementations(m)...)
		}
	}
	// A call from outside gitutil and localtier of one of theirs that has no
	// Context sibling but reaches a flagged call is listed at the caller too,
	// so a new hook caller of such a wrapper is caught where it is made.
	tainted := g.taintedGitNodes()
	for n := range seen {
		if g.gitName(g.names[n]) {
			continue
		}
		for _, e := range g.edges[n] {
			if fn, ok := e.(*types.Func); ok && tainted[fn] && g.flagKey(fn) == "" && fn.Type().(*types.Signature).Recv() == nil {
				keys[g.names[n]+" -> "+fn.Pkg().Name()+"."+fn.Name()] = true
			}
		}
	}
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// gitName says whether a node's name is in gitutil or localtier.
func (g *callGraph) gitName(name string) bool {
	return strings.HasPrefix(name, "internal/gitutil.") || strings.HasPrefix(name, "internal/localtier.")
}

// taintedGitNodes are the nodes of gitutil and localtier that make a flagged
// call or reach, within those packages, one that does.
func (g *callGraph) taintedGitNodes() map[any]bool {
	taint := map[any]bool{}
	for changed := true; changed; {
		changed = false
		for n, name := range g.names {
			if taint[n] || !g.gitName(name) {
				continue
			}
			hit := len(g.flags[n]) > 0
			for _, e := range g.edges[n] {
				if taint[e] {
					hit = true
					break
				}
			}
			if hit {
				taint[n] = true
				changed = true
			}
		}
	}
	return taint
}

// entryRoots finds the command literal whose Use begins with e.use inside the
// function e.fn and returns its Run-family handlers. Finding none is an
// error, so a renamed verb or constructor cannot drop a root unnoticed.
func entryRoots(pd *pkgDecls, e hookEntry) ([]any, error) {
	d := pd.funcs[e.fn]
	if d == nil {
		return nil, fmt.Errorf("hook entry %s: the function is not declared", e.fn)
	}
	var roots []any
	found := false
	ast.Inspect(d, func(x ast.Node) bool {
		cl, ok := x.(*ast.CompositeLit)
		if !ok || found || !useIs(cl, e.use) {
			return true
		}
		found = true
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, _ := kv.Key.(*ast.Ident)
			if key == nil || !strings.HasSuffix(strings.TrimSuffix(key.Name, "E"), "Run") {
				continue
			}
			switch v := kv.Value.(type) {
			case *ast.FuncLit:
				roots = append(roots, v)
			case *ast.Ident:
				roots = append(roots, pd.info.Uses[v])
			case *ast.SelectorExpr:
				roots = append(roots, pd.info.Uses[v.Sel])
			}
		}
		return false
	})
	if len(roots) == 0 {
		return nil, fmt.Errorf("hook entry %s %q: no command literal with a Run handler", e.fn, e.use)
	}
	for i, r := range roots {
		if fn, ok := r.(*types.Func); ok {
			roots[i] = fn.Origin()
		}
	}
	return roots, nil
}

func useIs(cl *ast.CompositeLit, use string) bool {
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, _ := kv.Key.(*ast.Ident); key == nil || key.Name != "Use" {
			continue
		}
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return false
		}
		f := strings.Fields(s)
		return len(f) > 0 && f[0] == use
	}
	return false
}
