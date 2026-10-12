package gittest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// helperImportPath is the import a test file must carry once it spawns git, so
// that its git commands run under gittest.Env's hermetic environment.
const helperImportPath = "github.com/intentdriven/abcd/internal/gittest"

// allowlist maps a repo-relative directory prefix to the reason its *_test.go
// files are exempt from the hermetic-git rule. Keep this SMALL and every entry
// justified — an exemption is a hole in the guarantee, not a convenience.
var allowlist = map[string]string{
	// The lifeboat tests build real repositories and drive `git fast-import`
	// over substantial history. They disable git's async background operations
	// (gc.auto/maintenance.auto/core.fsmonitor) via a TestMain that injects the
	// config with GIT_CONFIG_COUNT — the exact family gitutil.IsolatedEnv()
	// deliberately scrubs. Routing them through gittest.Env would strip that
	// injection and reintroduce the documented Linux ".git: directory not empty"
	// cleanup flake (see internal/core/lifeboat/testmain_test.go). They already
	// isolate global/system config and identity explicitly. Left as follow-up.
	"internal/core/lifeboat/": "async-disable via GIT_CONFIG_COUNT, scrubbed by IsolatedEnv; see testmain_test.go",
	// The check-ignore exec-pin regression (GHSA-h2gm) deliberately plants a
	// HOSTILE repo-local core.fsmonitor and drives CheckIgnored to prove the
	// isolatedGit pins neutralise it. Routing through gittest.Env would inject
	// core.fsmonitor=false and strip the exact config under test, defeating the
	// test's purpose. File-precise entry (matches only this one file), not the
	// whole gitutil package.
	"internal/gitutil/fsmonitor_exec_test.go": "plants a hostile repo-local core.fsmonitor to verify the isolatedGit exec-pins (GHSA-h2gm); gittest.Env would strip it",
}

// TestTestGitCallsAreHermetic is the enforcement half of iss-28. It walks every
// *_test.go in the module and fails if a test spawns git as a subprocess
// (exec.Command("git", …) / exec.CommandContext(…, "git", …)) without importing
// the shared gittest helper — the only sanctioned way to give a test git command
// the same GIT_DIR/GIT_WORK_TREE/GIT_CONFIG scrub production git runs under.
//
// A file that has been converted imports gittest and passes; a NEW test file that
// shells out to git without the helper is not in the allowlist and fails here,
// naming itself. This is what turns the convention into a guarantee.
func TestTestGitCallsAreHermetic(t *testing.T) {
	root := moduleRoot(t)

	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip VCS, the curated development record, and build output — never
			// Go source we test.
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "bin" || name == "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		// The helper package itself never spawns git; skip it (and this detector).
		if strings.HasPrefix(rel, "internal/gittest/") {
			return nil
		}
		if reason, ok := allowlistedReason(rel); ok {
			_ = reason // exempt by design; the reason documents why in the map.
			return nil
		}

		spawns, imports, parseErr := analyseTestFile(path)
		if parseErr != nil {
			return parseErr
		}
		if spawns && !imports {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("iss-28: %d test file(s) spawn git without the shared gittest helper.\n"+
			"Route every git command through cmd.Env = gittest.Env(t) "+
			"(import %q), or add a justified allowlist entry:\n  %s",
			len(offenders), helperImportPath, strings.Join(offenders, "\n  "))
	}
}

// allowlistedReason reports whether rel is under an allowlisted directory prefix.
func allowlistedReason(rel string) (string, bool) {
	for prefix, reason := range allowlist {
		if strings.HasPrefix(rel, prefix) {
			return reason, true
		}
	}
	return "", false
}

// analyseTestFile parses a Go test file and reports whether it (a) spawns git via
// exec.Command/exec.CommandContext with a literal "git" argument, and (b) imports
// the shared gittest helper.
func analyseTestFile(path string) (spawnsGit, importsHelper bool, err error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, false, err
	}

	for _, imp := range file.Imports {
		p, perr := strconv.Unquote(imp.Path.Value)
		if perr == nil && p == helperImportPath {
			importsHelper = true
			break
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "exec" {
			return true
		}
		if sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext" {
			return true
		}
		// Any string-literal "git" argument marks this as a git subprocess. This
		// catches both exec.Command("git", …) and exec.CommandContext(ctx, "git", …).
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, uerr := strconv.Unquote(lit.Value); uerr == nil && v == "git" {
					spawnsGit = true
					return false
				}
			}
		}
		return true
	})
	return spawnsGit, importsHelper, nil
}

// moduleRoot walks up from the working directory to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod above %s", dir)
		}
		dir = parent
	}
}

// TestNewRepoDisablesBackgroundMaintenance pins the iss-252 guarantee: no git
// command run in a fixture repo — by the fixture or by the code under test —
// may spawn a background maintenance process that outlives the test, because
// t.TempDir cleanup then races it on .git/objects and fails the run.
func TestNewRepoDisablesBackgroundMaintenance(t *testing.T) {
	r := gittest.NewRepo(t)
	for key, want := range maintenanceOff {
		if got := r.Git("config", "--local", "--get", key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
