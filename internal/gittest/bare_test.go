package gittest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// maintenanceOff is the repository config every fixture repository carries,
// written out here rather than read from the helper so the test pins the
// settings themselves, not whatever list the helper happens to hold.
var maintenanceOff = map[string]string{
	"gc.auto":          "0",
	"gc.autodetach":    "false",
	"maintenance.auto": "false",
	"core.fsmonitor":   "false",
}

// TestBareRepositoriesDisableBackgroundMaintenance holds a bare fixture to the
// iss-252 guarantee NewRepo already keeps: a push into a bare origin spawns
// its auto-gc there, and a detached gc that outlives the test races t.TempDir
// cleanup ("origin.git: directory not empty", iss-2610112202081803).
func TestBareRepositoriesDisableBackgroundMaintenance(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Commit("seed")
	inited := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, inited)
	cloned := filepath.Join(t.TempDir(), "mirror.git")
	gittest.CloneBare(t, src.Root(), cloned)
	for _, dir := range []string{inited, cloned} {
		if got := gitIn(t, dir, "rev-parse", "--is-bare-repository"); got != "true" {
			t.Errorf("%s: --is-bare-repository = %q, want true", filepath.Base(dir), got)
		}
		for key, want := range maintenanceOff {
			if got := gitIn(t, dir, "config", "--local", "--get", key); got != want {
				t.Errorf("%s: %s = %q, want %q", filepath.Base(dir), key, got, want)
			}
		}
	}
	if got := gitIn(t, inited, "symbolic-ref", "HEAD"); got != "refs/heads/main" {
		t.Errorf("an initialised bare repository's HEAD = %q, want refs/heads/main", got)
	}
	if got, want := gitIn(t, cloned, "rev-parse", "HEAD"), src.Git("rev-parse", "HEAD"); got != want {
		t.Errorf("a bare clone's HEAD = %q, want the source's %q", got, want)
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// bareInOneString is a git init or clone written as one string carrying --bare.
var bareInOneString = regexp.MustCompile(`\b(init|clone)\b.*--bare\b`)

// TestNoTestMakesABareRepositoryByHand keeps every bare fixture going through
// InitBare or CloneBare. A test that runs git init --bare or git clone --bare
// itself makes a repository without the maintenance config, and the race that
// config exists to stop comes back with it (iss-2610112202081803).
func TestNoTestMakesABareRepositoryByHand(t *testing.T) {
	root := moduleRoot(t)
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "bin" || name == "dist" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/gittest/") {
			return nil
		}
		lines, err := bareGitCalls(path)
		if err != nil {
			return err
		}
		for _, line := range lines {
			offenders = append(offenders, rel+":"+strconv.Itoa(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("%d call(s) make a bare repository by hand; use gittest.InitBare or gittest.CloneBare, "+
			"which write the maintenance config a bare fixture needs:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}

// bareGitCalls returns the line of every call in the file whose string
// arguments run git init or git clone with --bare, as separate arguments or
// as one string.
func bareGitCalls(path string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var lines []int
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var verb, bare bool
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			v, err := strconv.Unquote(lit.Value)
			if err != nil {
				continue
			}
			switch {
			case v == "init" || v == "clone":
				verb = true
			case v == "--bare":
				bare = true
			case bareInOneString.MatchString(v):
				verb, bare = true, true
			}
		}
		if verb && bare {
			lines = append(lines, fset.Position(call.Pos()).Line)
		}
		return true
	})
	return lines, nil
}
