package testreuse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Package is what the check reads of one package from go list.
type Package struct {
	ImportPath   string
	Dir          string
	GoFiles      []string
	Imports      []string
	TestGoFiles  []string
	XTestGoFiles []string
	// IgnoredGoFiles holds a test file built for another stage alone, whose tests
	// a list line can name.
	IgnoredGoFiles []string
	TestImports    []string
	XTestImports   []string
}

const goListFields = "ImportPath,Dir,GoFiles,IgnoredGoFiles,Imports,TestGoFiles,XTestGoFiles,TestImports,XTestImports"

// listPackages reads the packages patterns name, under the race build's file
// set when race is set, since a test file can be built for one stage alone.
func listPackages(ctx context.Context, dir string, env []string, race bool, patterns ...string) ([]Package, error) {
	args := []string{"list", "-json=" + goListFields}
	if race {
		args = append(args, "-race")
	}
	args = append(args, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []Package
	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				return pkgs, nil
			}
			return nil, fmt.Errorf("go list: %w", err)
		}
		pkgs = append(pkgs, p)
	}
}

// Survey classifies every package in all that has tests. All must hold the
// whole module, which gives each package its import closure: a package's tests
// can start a subprocess when its own test files, or anything they link in the
// module, import os/exec.
func Survey(all []Package) map[string]Verdict {
	byPath := make(map[string]Package, len(all))
	for _, p := range all {
		byPath[p.ImportPath] = p
	}
	memo := map[string]bool{}
	var spawns func(path string) bool
	spawns = func(path string) bool {
		if v, ok := memo[path]; ok {
			return v
		}
		memo[path] = false // an import cycle cannot compile, but must not hang
		p, ok := byPath[path]
		if !ok {
			return false
		}
		found := slices.Contains(p.Imports, "os/exec")
		for _, im := range p.Imports {
			if found {
				break
			}
			found = spawns(im)
		}
		memo[path] = found
		return found
	}
	// A package whose ordinary files import testing is a test helper, or holds
	// some: what a test calls in it counts as what the test reaches.
	helpers := map[string]map[string]reached{}
	unreadable := map[string]bool{}
	for _, p := range all {
		if !slices.Contains(p.Imports, "testing") {
			continue
		}
		var paths []string
		for _, f := range p.GoFiles {
			paths = append(paths, filepath.Join(p.Dir, f))
		}
		files, err := parseFiles(paths)
		if err != nil {
			unreadable[p.ImportPath] = true
			continue
		}
		helpers[p.ImportPath] = buildGraph(files, nil).symbols()
	}
	out := map[string]Verdict{}
	for _, p := range all {
		var files []string
		for _, f := range slices.Concat(p.TestGoFiles, p.XTestGoFiles) {
			files = append(files, filepath.Join(p.Dir, f))
		}
		if len(files) == 0 {
			continue
		}
		can := false
		for _, im := range slices.Concat(p.Imports, p.TestImports, p.XTestImports) {
			if im == "os/exec" || spawns(im) {
				can = true
				break
			}
		}
		if i := slices.IndexFunc(slices.Concat(p.TestImports, p.XTestImports), func(im string) bool { return unreadable[im] }); i >= 0 {
			out[p.ImportPath] = Verdict{Whole: "a test helper could not be read: " + slices.Concat(p.TestImports, p.XTestImports)[i]}
			continue
		}
		out[p.ImportPath] = ClassifyFiles(files, can, helpers)
	}
	return out
}

// testNames is every test a package's test files declare, in any build's files, so
// a list line naming a test built for one stage alone still resolves in the other.
func testNames(p Package) []string {
	paths := slices.Concat(p.TestGoFiles, p.XTestGoFiles)
	for _, f := range p.IgnoredGoFiles {
		if strings.HasSuffix(f, "_test.go") {
			paths = append(paths, f)
		}
	}
	var names []string
	fset := token.NewFileSet()
	for _, f := range paths {
		parsed, err := parser.ParseFile(fset, filepath.Join(p.Dir, f), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, d := range parsed.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && isTestName(fn.Name.Name) {
				names = append(names, fn.Name.Name)
			}
		}
	}
	return names
}
