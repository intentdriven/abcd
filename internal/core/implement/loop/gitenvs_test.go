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
// CHECKED for the hooks flag alone: every row whose command runs the
// repository's hooks names a runner that asks the drive's comparison before
// git runs (TestEveryHookedRowAsksTheComparisonFirst), or is named in
// hookedRowsThatAreTheComparisonsOwn with why it is not one; the functions
// a drive reaches that build that exempt constructor are exactly those named
// (TestTheExemptConstructorIsBuiltOnlyByTheComparison); and every call of an
// isolated runner whose subcommand is a constant names one gitutil judges for
// the programs it can start, or one on gitutil's starts-nothing list
// (TestEveryIsolatedCallADriveMakesIsJudgedForWhatItCanStart). A call whose
// command line is built elsewhere and passed whole (claim's own runners, the
// comparison's asks) is logged as not read: the runner judges it as it runs,
// and a subcommand it does not know asks.
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
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	_, found, programs, err := driveWalkOnce()
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

// hookedRowRunners run each hooked row's runner, as `git -C <dir> <args>`.
var hookedRowRunners = map[string]func(dir string, args ...string) error{
	"gitutil.Net":      func(dir string, args ...string) error { _, err := gitutil.Net(dir, args...); return err },
	"gitutil.NetEach":  func(dir string, args ...string) error { return gitutil.NetEach(dir, 64, func(string) {}, args...) },
	"gitutil.NetInput": func(dir string, args ...string) error { _, err := gitutil.NetInput(dir, "x\n", args...); return err },
	"gitutil.LocalGit": func(dir string, args ...string) error { _, err := gitutil.LocalGit(dir, args...); return err },
	"gitutil.RunCommitCommandBounded": func(dir string, args ...string) error {
		_, err := gitutil.RunCommitCommandBounded(dir, time.Minute, errors.New("stalled"), args...)
		return err
	},
	"gitutil.RunCommitCommandBoundedAsStored": func(dir string, args ...string) error {
		_, err := gitutil.RunCommitCommandBoundedAsStored(dir, time.Minute, errors.New("stalled"), args...)
		return err
	},
}

// hookedRowsThatAreTheComparisonsOwn are the hooked rows that are not
// runners that ask, each with why.
var hookedRowsThatAreTheComparisonsOwn = map[string]string{
	"gitutil.WorkingTreeCommitCommand": "a constructor, not a runner: a drive builds it itself only for the comparison's own reads " +
		"(workingTreeAsk, checkoutHooksPath), `git config` and `git rev-parse`, which start no hook; " +
		"its hooked commands run through RunCommitCommandBounded, which asks",
}

// TestEveryHookedRowAsksTheComparisonFirst: each row of driveGitCommands
// whose command runs the repository's hooks names a runner that, with a
// check set that refuses, gives git's subcommand to the check and does not
// run git; or it is named, with why, in hookedRowsThatAreTheComparisonsOwn.
// A hooked row added with neither fails here.
func TestEveryHookedRowAsksTheComparisonFirst(t *testing.T) {
	dir := gittest.NewRepo(t).Root()
	refused := errors.New("held setting changed")
	for _, c := range driveGitCommands {
		if !c.hooks {
			continue
		}
		if _, own := hookedRowsThatAreTheComparisonsOwn[c.name]; own {
			continue
		}
		run, ok := hookedRowRunners[c.name]
		if !ok {
			t.Errorf("%s runs the repository's hooks: name how it is run in hookedRowRunners, so it is seen to ask the comparison", c.name)
			continue
		}
		var saw []string
		unset, err := gitutil.SetBeforeHookedGit(func(args []string) error { saw = args; return refused })
		if err != nil {
			t.Fatal(err)
		}
		made := filepath.Join(dir, "made-"+strings.TrimPrefix(c.name, "gitutil."))
		err = run(dir, "init", "-q", made)
		unset()
		if !errors.Is(err, refused) || len(saw) == 0 || saw[0] != "init" {
			t.Errorf("%s does not ask the comparison before git runs: saw %q, %v", c.name, saw, err)
		}
		if _, serr := os.Stat(made); !errors.Is(serr, os.ErrNotExist) {
			t.Errorf("%s ran git under a refusing check: %v", c.name, serr)
		}
	}
}

// workingTreeCommitCommandUsers are the functions outside gitutil a drive
// reaches that build gitutil.WorkingTreeCommitCommand themselves, which
// hookedRowsThatAreTheComparisonsOwn exempts: the comparison's own reads of
// the stage's settings and of the checkout's hooks directory, and the
// stage's environment the hold reads.
var workingTreeCommitCommandUsers = []string{"loop.checkoutHooksPath", "loop.stageEnv", "loop.workingTreeAsk"}

// TestTheExemptConstructorIsBuiltOnlyByTheComparison: the functions a drive
// reaches that build WorkingTreeCommitCommand outside gitutil are exactly
// those the exemption names, so a hooked use added elsewhere fails here
// until it runs through a runner that asks. Watched failing with
// "loop.stageEnv" left out of the list.
func TestTheExemptConstructorIsBuiltOnlyByTheComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the module from source")
	}
	w, _, _, err := driveWalkOnce()
	if err != nil {
		t.Fatal(err)
	}
	var users []string
	for n := range w.reached {
		if slices.ContainsFunc(w.edges[n], func(e any) bool { return w.inGitutil(e) && w.name(e) == "gitutil.WorkingTreeCommitCommand" }) {
			users = append(users, w.name(n))
		}
	}
	sort.Strings(users)
	if !slices.Equal(slices.Compact(users), workingTreeCommitCommandUsers) {
		t.Fatalf("WorkingTreeCommitCommand is built outside gitutil by %v, not exactly %v: a hooked command runs through RunCommitCommandBounded, which asks", users, workingTreeCommitCommandUsers)
	}
}

// TestEveryIsolatedCallADriveMakesIsJudgedForWhatItCanStart: every call a
// drive reaches of an isolated runner whose command line the walk can read
// names a subcommand that gitutil.LeavesContentProgramsOn judges (so the
// runner asks when the command leaves a program on) or one on gitutil's
// starts-nothing list (gitutil.StartsNothing), which asks only in a form
// that can start a program; a subcommand in neither would ask at every run,
// so it fails here until it is looked at, unless its runner reaches the
// network and asks for that (isolatedRunnersThatAskByNetwork). A call through
// a runner that does not ask (isolatedRunnersThatDoNotAsk) fails here in a
// form that can start a program. Each call is logged with what it does: asks, blanks or reads
// nothing through a program. A call whose subcommand is not a constant is
// logged as not read: the runner still judges its command line when it runs.
// Watched failing with "diff-tree" taken out of the judged subcommands.
func TestEveryIsolatedCallADriveMakesIsJudgedForWhatItCanStart(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the module from source")
	}
	w, _, _, err := driveWalkOnce()
	if err != nil {
		t.Fatal(err)
	}
	isolated := map[string]bool{}
	for _, c := range driveGitCommands {
		if c.env == isolatedEnv {
			isolated[c.name] = true
		}
	}
	var lines []string
	for n := range w.reached {
		for _, c := range w.calls[n] {
			if !isolated[c.callee] {
				continue
			}
			at := w.name(n) + " (" + c.callee + ", " + filepath.Base(c.at) + ")"
			verb := gitutil.Subcommand(c.args)
			switch {
			case verb == "" || verb == "?":
				lines = append(lines, "not read: "+at+" "+strings.Join(c.args, " "))
			case isolatedRunnersThatAskByNetwork[c.callee]:
				// It asks before it is built, for the command as a whole.
				lines = append(lines, "asks before it reaches the network: "+at+" "+strings.Join(c.args, " "))
			case gitutil.LeavesContentProgramsOn(c.args) && (gitutil.JudgesContentPrograms(verb) || gitutil.StartsNothing(verb)):
				if _, own := isolatedRunnersThatDoNotAsk[c.callee]; own {
					t.Errorf("%s runs git %s, which can start a program the repository names, through a runner that does not ask: run it through one of gitutil's isolated runners", at, strings.Join(c.args, " "))
				}
				lines = append(lines, "asks: "+at+" "+strings.Join(c.args, " "))
			case gitutil.JudgesContentPrograms(verb):
				lines = append(lines, "blanks or reads no content through a program: "+at+" "+strings.Join(c.args, " "))
			default:
				if !gitutil.StartsNothing(verb) {
					t.Errorf("%s runs git %s in the isolated environment, which asks at every run: judge it in gitutil's contentJudges if it can start a filter, a textconv or external diff program or a merge driver, or say in gitutil's startsNothing which form is run and why it starts none", at, verb)
				}
				lines = append(lines, "starts nothing ("+verb+"): "+at)
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Log(l)
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

// walked is the drive's walk, made once for the package's tests: it
// type-checks the module from source, which takes a while.
var walked struct {
	once            sync.Once
	w               *gitWalk
	found, programs []string
	err             error
}

// driveWalkOnce is driveWalk of the drive's entry (RunTerminal), made once
// and walked: the walk, the constructors found and the programs named by a
// value.
func driveWalkOnce() (*gitWalk, []string, []string, error) {
	walked.once.Do(func() {
		var start any
		walked.w, start, walked.err = driveWalk("../../../..", "./internal/core/implement/loop", "internal/core/implement/loop.RunTerminal")
		if walked.err == nil {
			walked.found, walked.programs, walked.err = walked.w.reach(start)
		}
	})
	return walked.w, walked.found, walked.programs, walked.err
}

// driveWalk type-checks pkg's module packages from source under root and
// builds the walk, returning it and entry's node.
func driveWalk(root, pkg, entry string) (*gitWalk, any, error) {
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
	w := &gitWalk{edges: map[any][]any{}, gitExec: map[any]bool{}, programs: map[any]bool{}, names: map[any]string{}, funcs: map[string]*types.Func{},
		calls: map[any][]gitCall{}, reached: map[any]bool{}}
	fset := token.NewFileSet()
	w.fset = fset
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
	return w, start, nil
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
	calls    map[any][]gitCall // the calls of a gitutil function a node makes
	reached  map[any]bool      // the nodes outside gitutil reach walked
	fset     *token.FileSet
}

// gitCall is one call of a gitutil function with a variadic string
// parameter (a git command line): the function, the command line's
// arguments as written ("?" for one that is not a constant), and where.
type gitCall struct {
	callee string
	args   []string
	at     string
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
			w.gitutilCall(node, x, info)
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

// isolatedRunnersThatDoNotAsk are the isolated rows built outside gitutil,
// which run git without consulting gitutil's table of what a command can
// start, each with why. Their calls are read as gitutil's runners' are, and
// one the table says can start a program fails
// TestEveryIsolatedCallADriveMakesIsJudgedForWhatItCanStart.
var isolatedRunnersThatDoNotAsk = map[string]string{
	"spotcheck.storeGit": "the spot-check store's writes (hash-object --no-filters, read-tree with no -u, " +
		"update-index --index-info, write-tree, commit-tree, update-ref), none of which starts a program",
}

// isolatedRunnersThatAskByNetwork are the isolated rows that reach the
// network, and so ask the check gitutil.SetBeforeHookedGit set before the
// command is built, whatever its subcommand (ls-remote, which no list holds).
var isolatedRunnersThatAskByNetwork = map[string]bool{"gitutil.RunLimitedNetContext": true}

// gitutilCall records call when it calls a gitutil function whose last
// parameter is a variadic string (a git command line) from outside gitutil,
// or one of isolatedRunnersThatDoNotAsk.
func (w *gitWalk) gitutilCall(node any, call *ast.CallExpr, info *types.Info) {
	var fn *types.Func
	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		fn, _ = info.Uses[f.Sel].(*types.Func)
	case *ast.Ident:
		fn, _ = info.Uses[f].(*types.Func)
	}
	if fn == nil || fn.Pkg() == nil {
		return
	}
	if _, own := isolatedRunnersThatDoNotAsk[w.name(fn)]; !own && w.rel(fn.Pkg().Path()) != "internal/gitutil" {
		return
	}
	if p, ok := node.(*types.Func); ok && p.Pkg() != nil && w.rel(p.Pkg().Path()) == "internal/gitutil" {
		return
	}
	sig := fn.Type().(*types.Signature)
	if !sig.Variadic() {
		return
	}
	from := sig.Params().Len() - 1
	var args []string
	for i := from; i < len(call.Args); i++ {
		if call.Ellipsis.IsValid() && i == len(call.Args)-1 {
			args = append(args, constElems(call.Args[i], info)...)
			continue
		}
		args = append(args, constString(call.Args[i], info))
	}
	w.calls[node] = append(w.calls[node], gitCall{callee: w.name(fn), args: args, at: w.fset.Position(call.Pos()).String()})
}

// constString is e's value when it is a string constant, else "?".
func constString(e ast.Expr, info *types.Info) string {
	if tv := info.Types[e]; tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value)
	}
	return "?"
}

// constElems are the elements of a slice passed whole (args...): a
// composite literal's, or an append's onto one, each a constant or "?"; "?"
// for anything else.
func constElems(e ast.Expr, info *types.Info) []string {
	switch x := e.(type) {
	case *ast.CompositeLit:
		var out []string
		for _, el := range x.Elts {
			out = append(out, constString(el, info))
		}
		return out
	case *ast.CallExpr:
		if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "append" && len(x.Args) > 0 {
			out := constElems(x.Args[0], info)
			for i, a := range x.Args[1:] {
				if x.Ellipsis.IsValid() && i == len(x.Args)-2 {
					out = append(out, constElems(a, info)...)
					continue
				}
				out = append(out, constString(a, info))
			}
			return out
		}
	}
	return []string{"?"}
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
		if w.reached != nil {
			w.reached[n] = true
		}
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
