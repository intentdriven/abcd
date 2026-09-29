// Package reachaudit is the exported-reach caller audit iss-33 asked for, as
// one matcher every audit shares. It is test support: only tests import it, and
// no production code depends on it.
//
// An exported function exists to be reached from outside its package, so one
// that no production code outside the package names is silent scaffolding — it
// compiles, it can rot, and nothing would notice (loud-staging.md). The audit
// finds them by parsing the module, never by matching text: only a selector on
// an imported package counts as a caller, so a comment or a string literal
// naming pkg.X does not, and neither does a selector on a local variable that
// shadows the import's name. Only Go the release build compiles counts: test
// files, testdata/ trees, nested modules and worktrees, and any file a build
// constraint excludes from every release target (`//go:build ignore`, an
// eval-only tag) name no caller.
package reachaudit

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Targets is the set of GOOS/GOARCH pairs the release build compiles, the
// Makefile's TARGETS. A file counts when any target compiles it, so a caller in
// a linux-only file reaches its callee on a darwin machine too, and the verdict
// does not depend on the machine the audit runs on. TestTargetsMatchTheMakefile
// holds the two lists together.
var Targets = []string{"darwin/arm64", "darwin/amd64", "linux/arm64", "linux/amd64"}

// Reach maps every exported top-level function in scope, keyed
// "<package dir relative to the module root>.<Name>" (for example
// "internal/core/launch.Ship"), to whether production code outside its package
// selects it. Functions named ...ForTest are left out: the suffix declares a
// cross-package test seam, exempt by its name.
type Reach map[string]bool

// Unreached lists, sorted, the functions in r that nothing reaches.
func (r Reach) Unreached() []string {
	var out []string
	for name, reached := range r {
		if !reached {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Scan audits the module rooted at root (its go.mod names the module path).
// scope is a package directory relative to root, slash-separated; the exported
// functions of that package and of every package beneath it are the ones
// audited, while the callers are read from the whole module. A Go file that
// does not parse fails the scan, so a file is never silently counted as naming
// nothing.
func Scan(root, scope string) (Reach, error) {
	modPath, err := modulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	scope = strings.Trim(path.Clean(filepath.ToSlash(scope)), "/")
	type file struct {
		dir string
		f   *ast.File
	}
	var files []file
	names := map[string]string{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDir(p, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if ok, err := Compiled(name, src); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		} else if !ok {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, src, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(rel)
		names[dir] = f.Name.Name
		files = append(files, file{dir, f})
		return nil
	})
	if err != nil {
		return nil, err
	}
	reach := Reach{}
	reached := map[string]bool{}
	for _, fl := range files {
		for _, key := range Selectors(fl.f, modPath, names) {
			reached[key] = true
		}
		if fl.dir == scope || scope == "." || strings.HasPrefix(fl.dir, scope+"/") {
			for _, fn := range exportedFuncs(fl.f) {
				reach[fl.dir+"."+fn] = false
			}
		}
	}
	for key := range reach {
		reach[key] = reached[key]
	}
	return reach, nil
}

// skipDir reports whether a directory below the module root holds no
// production Go of this module: version-control and hidden trees, directories
// the go tool ignores (a leading underscore, testdata/), vendored or packaged
// JavaScript, and a nested module or worktree (its own go.mod or .git), which
// is another build altogether.
func skipDir(p, name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "node_modules" || name == "vendor" {
		return true
	}
	for _, marker := range []string{"go.mod", ".git"} {
		if _, err := os.Lstat(filepath.Join(p, marker)); err == nil {
			return true
		}
	}
	return false
}

// Compiled reports whether the release build compiles a non-test Go file named
// name with contents src for at least one of Targets, honouring both its
// filename suffixes (_linux, _arm64) and its //go:build line. No build tag is
// set, so a file under `//go:build ignore` or behind an opt-in tag (an eval
// lane) is not compiled, and cgo is off, as a cross-compiled release has it.
func Compiled(name string, src []byte) (bool, error) {
	for _, target := range Targets {
		goos, goarch, _ := strings.Cut(target, "/")
		ctx := build.Default
		ctx.GOOS, ctx.GOARCH = goos, goarch
		ctx.CgoEnabled = false
		ctx.BuildTags = nil
		ctx.ToolTags = nil
		ctx.OpenFile = func(string) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(src)), nil
		}
		ok, err := ctx.MatchFile(".", name)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// ParseSource parses one Go file for Selectors, with the object resolution
// Selectors relies on to tell an import from a local that shadows its name.
func ParseSource(src []byte) (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "", src, 0)
}

// Selectors names every function, type, variable or constant f selects from a
// package of the module modPath, as "<package dir relative to the module
// root>.<Name>". names maps a package directory to its package name, the name
// an import binds when it declares none; a directory it lacks binds the last
// element of its import path. It counts a selector only when its left side is the name the
// file imports that package under and that name is not a local declaration: f
// must be parsed with object resolution (ParseSource, or parser flags without
// SkipObjectResolution), under which an identifier that resolves to a package
// import has no object, while a variable, parameter or constant shadowing the
// import's name resolves to its declaration. A blank or dot import yields no
// selector, so a function reached only through one reads as unreached: the
// audit fails loud rather than quiet.
func Selectors(f *ast.File, modPath string, names map[string]string) []string {
	local := map[string]string{}
	for _, imp := range f.Imports {
		ip := strings.Trim(imp.Path.Value, "`\"")
		if !strings.HasPrefix(ip, modPath+"/") {
			continue
		}
		dir := strings.TrimPrefix(ip, modPath+"/")
		name, ok := names[dir]
		if !ok {
			name = path.Base(ip)
		}
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "_" || name == "." {
			continue
		}
		local[name] = dir
	}
	if len(local) == 0 {
		return nil
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Obj != nil {
			return true
		}
		if dir, ok := local[id.Name]; ok {
			out = append(out, dir+"."+sel.Sel.Name)
		}
		return true
	})
	return out
}

// exportedFuncs lists f's exported top-level functions, methods excluded, and
// leaves out a name ending ForTest: a declared cross-package test seam.
func exportedFuncs(f *ast.File) []string {
	var out []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !fn.Name.IsExported() || strings.HasSuffix(fn.Name.Name, "ForTest") {
			continue
		}
		out = append(out, fn.Name.Name)
	}
	return out
}

// modulePath reads the module path from a go.mod file.
func modulePath(gomod string) (string, error) {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), "\""), nil
		}
	}
	return "", fmt.Errorf("%s names no module path", gomod)
}
