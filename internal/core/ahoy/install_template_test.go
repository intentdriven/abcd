package ahoy

// One install, kept and replayed. A fixture that needs a repository abcd has
// already set up runs Install for it, about forty git processes before the test
// has begun to test anything, and over a hundred tests start that way. The first
// install in a given starting state runs for real and what it changed is kept;
// every later one starting from the same state is written back from the record,
// so the test meets the same files, modes and links in its own directories.
//
// The starting state is the key: the fixture's name, the environment, and every
// file under the directories named in installRoots, each with the test's own
// paths taken out. A test that changes any of them first gets an install of its
// own. So does a test that has swapped a package seam, whose effect no fingerprint
// can see. TestAKeptInstallIsTheInstallItReplays holds the replay to a real
// install, and TestEverySwappedSeamIsListedOrExempt holds the list of seams.

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/question"
)

// installTemplates holds the kept installs by starting state; replays counts
// the times one was written back, for the test that needs to know it was.
var installTemplates struct {
	mu      sync.Mutex
	byKey   map[string]*keptInstall
	replays int
}

// keptInstall is what one install changed, in tokenised form.
type keptInstall struct {
	writes  []keptEntry
	removes []keptEntry
}

// keptEntry is one file, link or folder under a root: root is the root's token
// and rel its slash-separated place there. body and link carry tokens for the
// test's own paths.
type keptEntry struct {
	root, rel string
	mode      fs.FileMode
	link      string
	body      []byte
}

// installRoot is a directory an install reads or writes, with the token that
// stands for its path in anything kept.
type installRoot struct {
	token, dir string
}

// templatedInstall makes repo, an empty directory the test owns, what build
// would make it: by running build the first time this fixture is asked for from
// this starting state, and by replaying what that run changed after that.
func templatedInstall(t *testing.T, repo, fixture string, build func()) {
	t.Helper()
	roots, ok := installRoots(repo)
	if !ok || !seamsAtDefault() {
		build()
		return
	}
	before := captureRoots(t, roots)
	key := installKey(t, fixture, roots, before)

	installTemplates.mu.Lock()
	defer installTemplates.mu.Unlock()
	if kept, ok := installTemplates.byKey[key]; ok {
		kept.replay(t, roots)
		installTemplates.replays++
		return
	}
	build()
	kept := diffInstall(before, captureRoots(t, roots))
	if installTemplates.byKey == nil {
		installTemplates.byKey = map[string]*keptInstall{}
	}
	installTemplates.byKey[key] = kept
}

// installRoots names the directories in force, longest first so that a path
// inside two is tokenised by the nearer. It reports false when one of them is
// not a temporary directory of the test's, which is no state to key on.
func installRoots(repo string) ([]installRoot, bool) {
	candidates := []installRoot{
		{"@REPO@", repo},
		{"@HOME@", os.Getenv("HOME")},
		{"@PLUGIN@", os.Getenv("ABCD_PLUGIN_ROOT")},
		{"@DATA@", os.Getenv("CLAUDE_PLUGIN_DATA")},
		{"@CONFIG@", os.Getenv("CLAUDE_CONFIG_DIR")},
	}
	if target := os.Getenv("ABCD_BIN_TARGET"); target != "" {
		candidates = append(candidates, installRoot{"@BIN@", filepath.Dir(filepath.Dir(target))})
	}
	var roots []installRoot
	for _, r := range candidates {
		if r.dir == "" {
			continue
		}
		if !testOwnedDir(r.dir) {
			return nil, false
		}
		// One root inside another is captured once, as part of the outer.
		nested := false
		for _, o := range candidates {
			if o.dir != "" && o.dir != r.dir && strings.HasPrefix(r.dir, o.dir+string(filepath.Separator)) {
				nested = true
			}
		}
		if !nested {
			roots = append(roots, r)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return len(roots[i].dir) > len(roots[j].dir) })
	return roots, true
}

// testOwnedDir reports whether dir is under the temporary area tests make their
// directories in.
func testOwnedDir(dir string) bool {
	tmp := strings.TrimRight(os.TempDir(), string(filepath.Separator))
	return strings.HasPrefix(dir, tmp+string(filepath.Separator))
}

// physicalDir is dir with its links resolved, which is how a path the code
// under test resolved comes out where the temporary area sits behind a link.
func physicalDir(dir string) string {
	if p, err := filepath.EvalSymlinks(dir); err == nil {
		return p
	}
	return dir
}

// setupDateInConfig is the day an install stamps in the config it writes.
var setupDateInConfig = regexp.MustCompile(`"setup_date": "[0-9]{4}-[0-9]{2}-[0-9]{2}"`)

// repoNameInConfig is the repository's name as an install stamps it in the
// config, which is the base name of the test's temporary directory.
func repoNameInConfig(name string) string { return `"project_name": ` + strconv.Quote(name) }

// tokenise takes out of b what is the test's own or the day's: the paths, the
// name the repository's directory gives it, and the date.
func tokenise(b []byte, roots []installRoot) []byte {
	s := string(b)
	for _, r := range roots {
		s = strings.ReplaceAll(s, physicalDir(r.dir), r.token+"#physical")
		s = strings.ReplaceAll(s, r.dir, r.token)
		if r.token == "@REPO@" {
			s = strings.ReplaceAll(s, repoNameInConfig(filepath.Base(r.dir)), repoNameInConfig("@REPO_NAME@"))
		}
	}
	return []byte(setupDateInConfig.ReplaceAllString(s, `"setup_date": "@TODAY@"`))
}

// untokenise puts this test's own into what tokenise took it out of.
func untokenise(b []byte, roots []installRoot) []byte {
	s := string(b)
	for _, r := range roots {
		s = strings.ReplaceAll(s, r.token+"#physical", physicalDir(r.dir))
		s = strings.ReplaceAll(s, r.token, r.dir)
		if r.token == "@REPO@" {
			s = strings.ReplaceAll(s, repoNameInConfig("@REPO_NAME@"), repoNameInConfig(filepath.Base(r.dir)))
		}
	}
	return []byte(strings.ReplaceAll(s, "@TODAY@", time.Now().UTC().Format("2006-01-02")))
}

// captureRoots reads every entry under the roots, tokenised.
func captureRoots(t *testing.T, roots []installRoot) map[string]keptEntry {
	t.Helper()
	entries := map[string]keptEntry{}
	for _, r := range roots {
		err := filepath.WalkDir(r.dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if p == r.dir {
				return nil
			}
			rel, err := filepath.Rel(r.dir, p)
			if err != nil {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			e := keptEntry{root: r.token, rel: filepath.ToSlash(rel), mode: fi.Mode()}
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				link, err := os.Readlink(p)
				if err != nil {
					return err
				}
				e.link = string(tokenise([]byte(link), roots))
			case fi.Mode().IsRegular():
				body, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				e.body = tokenise(body, roots)
			case !fi.IsDir():
				return fmt.Errorf("%s: no replay for mode %v", p, fi.Mode())
			}
			entries[e.root+"/"+e.rel] = e
			return nil
		})
		if err != nil {
			t.Fatalf("reading %s: %v", r.dir, err)
		}
	}
	return entries
}

// environmentInKey is the environment with the test's own paths taken out: every
// variable is state an install may read. A path into a temporary directory that
// is no root (a fake tool's folder on PATH) stands for what it holds (heldBy).
func environmentInKey(roots []installRoot) []string {
	tmp := strings.TrimRight(os.TempDir(), string(filepath.Separator))
	tempPath := regexp.MustCompile(regexp.QuoteMeta(tmp) + `/[^/:]+/[0-9]+(/[^:]*)?`)
	var env []string
	for _, kv := range os.Environ() {
		kv = string(tokenise([]byte(kv), roots))
		kv = tempPath.ReplaceAllStringFunc(kv, func(path string) string {
			return "@TEMP" + tempPath.FindStringSubmatch(path)[1] + "[" + heldBy(path) + "]@"
		})
		env = append(env, kv)
	}
	sort.Strings(env)
	return env
}

// heldBy describes what is at path, so two tests' folders with the same contents
// read the same wherever they sit: every entry under it at any depth with its
// mode, a file by its digest, and a link by the digest of the file it reaches, or
// by its target when it reaches none.
//
// A fake gh's own files are left out of a folder that holds one: its script names
// its test's log, its routes are the test's answers, and no install runs gh
// (TestNoInstallRunsGh).
func heldBy(path string) string {
	_, statErr := os.Lstat(filepath.Join(path, "gh"))
	fakeGh := statErr == nil
	var held []string
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		if fakeGh && d.IsDir() && rel == "routes" {
			return fs.SkipDir
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		item := rel + " " + fi.Mode().String()
		switch {
		case fakeGh && (rel == "gh" || rel == "gh.log"):
		case fi.Mode().IsRegular():
			body, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			item += fmt.Sprintf(" %x", sha256.Sum256(body))
		case fi.Mode()&os.ModeSymlink != 0:
			body, err := os.ReadFile(p)
			if err == nil {
				target, _ := os.Stat(p)
				item += fmt.Sprintf(" -> %v %x", target.Mode(), sha256.Sum256(body))
			} else {
				link, _ := os.Readlink(p)
				item += " -> " + link
			}
		}
		held = append(held, item)
		return nil
	})
	if err != nil {
		return "?"
	}
	return strings.Join(held, ",")
}

// installKey is the fingerprint of the starting state.
func installKey(t *testing.T, fixture string, roots []installRoot, before map[string]keptEntry) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00", fixture, cwd)
	for _, kv := range environmentInKey(roots) {
		fmt.Fprintf(h, "%s\x00", kv)
	}
	names := make([]string, 0, len(before))
	for name := range before {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e := before[name]
		fmt.Fprintf(h, "%s\x00%v\x00%s\x00%x\x00", name, e.mode, e.link, sha256.Sum256(e.body))
	}
	return string(h.Sum(nil))
}

// diffInstall is what an install changed.
func diffInstall(before, after map[string]keptEntry) *keptInstall {
	kept := &keptInstall{}
	for name, e := range after {
		if b, ok := before[name]; !ok || b.mode != e.mode || b.link != e.link || string(b.body) != string(e.body) {
			kept.writes = append(kept.writes, e)
		}
	}
	for name, e := range before {
		if _, ok := after[name]; !ok {
			kept.removes = append(kept.removes, e)
		}
	}
	byPlace := func(a, b keptEntry) int {
		return strings.Compare(a.root+"/"+a.rel, b.root+"/"+b.rel)
	}
	slices.SortFunc(kept.writes, byPlace)
	slices.SortFunc(kept.removes, byPlace)
	return kept
}

// replay makes the kept changes under this test's roots: parents before what
// they hold, and each folder's own mode last, since a read-only one could
// otherwise not be filled.
func (k *keptInstall) replay(t *testing.T, roots []installRoot) {
	t.Helper()
	dirOf := map[string]string{}
	for _, r := range roots {
		dirOf[r.token] = r.dir
	}
	place := func(e keptEntry) string { return filepath.Join(dirOf[e.root], filepath.FromSlash(e.rel)) }
	for i := len(k.removes) - 1; i >= 0; i-- {
		if err := os.Remove(place(k.removes[i])); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range k.writes {
		p := place(e)
		var err error
		switch {
		case e.mode.IsDir():
			err = os.MkdirAll(p, 0o700)
		case e.mode&os.ModeSymlink != 0:
			_ = os.Remove(p)
			err = os.Symlink(string(untokenise([]byte(e.link), roots)), p)
		default:
			err = os.WriteFile(p, untokenise(e.body, roots), 0o600)
			if err == nil {
				err = os.Chmod(p, e.mode.Perm())
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := len(k.writes) - 1; i >= 0; i-- {
		if e := k.writes[i]; e.mode.IsDir() {
			if err := os.Chmod(place(e), e.mode.Perm()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// defaultInstallSeams is every listed seam as TestMain left it.
var defaultInstallSeams map[string]string

// seamsAtDefault reports whether every listed seam is as TestMain left it. A test
// that swapped one wants an install through its own.
func seamsAtDefault() bool {
	return maps.Equal(installSeams(), defaultInstallSeams)
}

// installSeams identifies each seam an install can reach, by name: a function by
// the code it points at, anything else by its value.
// TestEverySwappedSeamIsListedOrExempt fails on a seam a test or the package swaps
// that is neither here nor in seamsNotInTheKey.
func installSeams() map[string]string {
	funcs := map[string]any{
		"downloadTag": downloadTag, "newToolInstaller": newToolInstaller, "afterCacheBound": afterCacheBound,
		"readRootAgents": readRootAgents, "beforeToolFileRemove": beforeToolFileRemove,
		"copyRelease": copyRelease, "checkReleaseClaim": checkReleaseClaim, "goRunModule": goRunModule,
		"checkoutBuild": checkoutBuild, "readClaudeVersion": readClaudeVersion, "caseFoldingFS": caseFoldingFS,
		"shellSettingsAsks": shellSettingsAsks, "measureFits": measureFits, "entryTrust": entryTrust,
		"writeHarnessSettings": writeHarnessSettings, "shellEnv": shellEnv, "shellApply": shellApply,
		"removeBackupCopy": removeBackupCopy, "gitInstalled": gitInstalled, "harnessInstalled": harnessInstalled,
		"osExecutable": osExecutable, "afterHistoryReloadHook": afterHistoryReloadHook,
		"beforeHistoryIndexCreateHook": beforeHistoryIndexCreateHook, "currentVintage": currentVintage,
		"waitingRange": waitingRange, "waitingDefaultChanges": waitingDefaultChanges,
	}
	seams := map[string]string{
		"claudeCodeAgentsFloor": fmt.Sprint(claudeCodeAgentsFloor),
		"core.Version":          core.Version,
		"earlierShimSHA256":     fmt.Sprint(earlierShimSHA256),
		"hostVersionTimeout":    hostVersionTimeout.String(),
		"historyLockTimeout":    historyLockTimeout.String(),
		"http.DefaultTransport": fmt.Sprintf("%p", http.DefaultTransport),
		"oracleAdapterShipped":  fmt.Sprint(oracleAdapterShipped),
		"rewriteLockTimeout":    rewriteLockTimeout.String(),
	}
	for name, f := range funcs {
		seams[name] = fmt.Sprint(reflect.ValueOf(f).Pointer())
	}
	return seams
}

// seamsNotInTheKey is every swapped seam installSeams does not list, with why. A
// seam belongs here only when listing it is impossible or pointless: the test
// below holds an entry marked neverBesideAFixture to that claim.
var seamsNotInTheKey = map[string]string{
	// Each swaps an unexported variable of fsutil, which this package cannot read.
	"fsutil.SwapHomeScopeVettedForTest": neverBesideAFixture,
	"fsutil.SwapOwnerUIDForTest":        neverBesideAFixture,
	// These two read and change nothing: a test's way of looking at the record
	// and the floors, named *ForTest because only a test wants to.
	"identity.LoadRecordForTest": "reads a record, swaps nothing",
	"question.FloorsForTest":     "reads the floors, swaps nothing",
}

// neverBesideAFixture marks a seam no test reaching a swap of it reaches a kept
// install, so it cannot be kept under, or replayed to, a swapped state.
const neverBesideAFixture = "never beside a fixture"

// TestEverySwappedSeamIsListedOrExempt reads this package's source for every
// swap: an assignment to a package-level variable of its non-test files or to an
// exported variable of an imported package, or a call of an imported package's
// *ForTest function. A swap neither in installSeams nor in seamsNotInTheKey could
// be kept under the default key and replayed to later tests, whose result would
// then depend on run order, so it fails here by name.
//
// A variable declared in a test file is not a seam: production code cannot read
// it except through a seam, which is itself swapped. A swap made some other way,
// through a pointer or a foreign setter not named *ForTest, is not seen.
func TestEverySwappedSeamIsListedOrExempt(t *testing.T) {
	src := parsePackageSource(t)
	listed := installSeams()
	swaps := src.swaps()
	var names []string
	for name := range swaps {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("the walk found no swap at all, so it is reading nothing")
	}
	for _, name := range names {
		_, inList := listed[name]
		why, exempt := seamsNotInTheKey[name]
		switch {
		case inList && exempt:
			t.Errorf("%s is both listed and exempt", name)
		case !inList && !exempt:
			t.Errorf("%s is swapped (%s) and neither listed in installSeams nor exempt in seamsNotInTheKey: "+
				"a fixture could keep an install under the default key with it swapped", name, swaps[name])
		case exempt && why == neverBesideAFixture:
			if test := src.testReachingFixtureAndSwap(name); test != "" {
				t.Errorf("%s is exempt as %s, but %s reaches both a swap of it and a kept install", name, why, test)
			}
		case exempt && why == "":
			t.Errorf("%s is exempt with no reason", name)
		}
	}
	for name := range seamsNotInTheKey {
		if _, ok := swaps[name]; !ok {
			t.Errorf("%s is exempt but nothing swaps it any more", name)
		}
	}
}

// packageSource is this package's files, parsed.
type packageSource struct {
	files map[string]*ast.File
	// productionVars is every package-level variable of a non-test file.
	productionVars map[string]bool
}

func parsePackageSource(t *testing.T) packageSource {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil || len(paths) == 0 {
		t.Fatalf("listing this package's files: %v (%d)", err, len(paths))
	}
	src := packageSource{files: map[string]*ast.File{}, productionVars: map[string]bool{}}
	fset := token.NewFileSet()
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		src.files[p] = f
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok || d.Tok != token.VAR || strings.HasSuffix(p, "_test.go") {
				continue
			}
			for _, spec := range d.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					src.productionVars[name.Name] = true
				}
			}
		}
	}
	return src
}

// swappedBy names what one assignment target swaps, or "": x, x.f and x[k] of a
// production variable x, and pkg.Name of an imported package.
func (src packageSource) swappedBy(f *ast.File, target ast.Expr) string {
	for {
		switch e := target.(type) {
		case *ast.ParenExpr:
			target = e.X
			continue
		case *ast.IndexExpr:
			target = e.X
			continue
		case *ast.StarExpr:
			target = e.X
			continue
		case *ast.SelectorExpr:
			if pkg, ok := e.X.(*ast.Ident); ok && pkg.Obj == nil && importsName(f, pkg.Name) {
				return pkg.Name + "." + e.Sel.Name
			}
			target = e.X
			continue
		case *ast.Ident:
			if !src.productionVars[e.Name] {
				return ""
			}
			// Declared in a function, it shadows the variable. The parser
			// resolves names within a file, so one from another file has no
			// object, and one from this file's top level is a ValueSpec.
			if e.Obj != nil {
				if spec, ok := e.Obj.Decl.(*ast.ValueSpec); !ok || !topLevelSpec(f, spec) {
					return ""
				}
			}
			return e.Name
		default:
			return ""
		}
	}
}

// topLevelSpec reports whether spec is one of f's package-level declarations.
func topLevelSpec(f *ast.File, spec *ast.ValueSpec) bool {
	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			for _, s := range d.Specs {
				if s == spec {
					return true
				}
			}
		}
	}
	return false
}

// importsName reports whether f imports a package under name.
func importsName(f *ast.File, name string) bool {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		local := pathpkg.Base(path)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		if local == name {
			return true
		}
	}
	return false
}

// swapsIn collects the swaps under node, by name, with where each one sits.
func (src packageSource) swapsIn(file string, f *ast.File, node ast.Node, into map[string]string) {
	note := func(name string) {
		if name != "" {
			if _, ok := into[name]; !ok {
				into[name] = file
			}
		}
	}
	ast.Inspect(node, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if s.Tok == token.DEFINE {
				return true
			}
			for _, lhs := range s.Lhs {
				note(src.swappedBy(f, lhs))
			}
		case *ast.IncDecStmt:
			note(src.swappedBy(f, s.X))
		case *ast.CallExpr:
			if sel, ok := s.Fun.(*ast.SelectorExpr); ok && strings.HasSuffix(sel.Sel.Name, "ForTest") {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Obj == nil && importsName(f, pkg.Name) {
					note(pkg.Name + "." + sel.Sel.Name)
				}
			}
		}
		return true
	})
}

// swaps is every swap in the package's declarations, by name: function bodies, and
// closures in a package-level variable's initialiser, such as a table of cases. A
// declaration's own initial value is no assignment, so it is no swap.
func (src packageSource) swaps() map[string]string {
	found := map[string]string{}
	for file, f := range src.files {
		for _, decl := range f.Decls {
			src.swapsIn(file, f, decl, found)
		}
	}
	return found
}

// testReachingFixtureAndSwap names a Test function that reaches, through this
// package's functions and package-level variables, both a swap of name and a kept
// install, or "". A function is reached by being named at all, since tests pass
// fixtures around as values, and a variable by the same rule, which reaches all
// its initialiser names: a table of cases holds closures the test runs.
func (src packageSource) testReachingFixtureAndSwap(name string) string {
	type node struct {
		file string
		f    *ast.File
		name string
		body ast.Node
	}
	var nodes []node
	known := map[string]bool{}
	for file, f := range src.files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body != nil {
					nodes = append(nodes, node{file, f, d.Name.Name, d.Body})
					known[d.Name.Name] = true
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					for _, id := range spec.(*ast.ValueSpec).Names {
						nodes = append(nodes, node{file, f, id.Name, spec})
						known[id.Name] = true
					}
				}
			}
		}
	}
	mentions, swaps := map[string][]string{}, map[string]bool{}
	var tests []string
	for _, n := range nodes {
		ast.Inspect(n.body, func(c ast.Node) bool {
			if id, ok := c.(*ast.Ident); ok && known[id.Name] && id.Name != n.name {
				mentions[n.name] = append(mentions[n.name], id.Name)
			}
			return true
		})
		found := map[string]string{}
		src.swapsIn(n.file, n.f, n.body, found)
		if _, ok := found[name]; ok {
			swaps[n.name] = true
		}
		if strings.HasPrefix(n.name, "Test") {
			tests = append(tests, n.name)
		}
	}
	sort.Strings(tests)
	for _, test := range tests {
		seen := map[string]bool{}
		var walk func(fn string)
		walk = func(fn string) {
			if !seen[fn] {
				seen[fn] = true
				for _, next := range mentions[fn] {
					walk(next)
				}
			}
		}
		walk(test)
		swapped := false
		for fn := range swaps {
			swapped = swapped || seen[fn]
		}
		if swapped && seen["templatedInstall"] {
			return test
		}
	}
	return ""
}

// TestAKeptInstallIsTheInstallItReplays: a fixture that replayed an install
// leaves what a real install from the same starting state leaves under the same
// roots, byte for byte, mode for mode and link for link, with each test's own
// paths, name and date in place. It catches a replay that gets a kept file wrong,
// such as a path or a name left behind from the test that kept it.
//
// It compares inside installRoots, in one starting state, so it cannot see an
// install that writes outside those roots, which a replay would lose, or one that
// comes to depend on something the key leaves out. installRoots and installKey
// are held to those by reading them, and by TestEverySwappedSeamIsListedOrExempt
// for the seams.
func TestAKeptInstallIsTheInstallItReplays(t *testing.T) {
	const origin = "https://github.com/example-org/example-repo"
	for _, tc := range []struct {
		name string
		make func(t *testing.T) string
		real func(t *testing.T, repo string)
	}{
		{"installed", installedRepo, func(t *testing.T, repo string) {
			idMustMkdir(t, filepath.Join(repo, ".git"))
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
		}},
		{"managed", func(t *testing.T) string { return managedRepoWithOrigin(t, origin) }, func(t *testing.T, repo string) {
			idMustGit(t, repo, "init")
			idMustGit(t, repo, "config", "user.name", "Alice Example")
			idMustGit(t, repo, "config", "user.email", "alice@example.com")
			idMustGit(t, repo, "remote", "add", "origin", origin)
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The first may be the one that keeps it; the second replays it.
			t.Run("keep", func(t *testing.T) {
				setupHermetic(t)
				tc.make(t)
			})
			var replayed map[string]keptEntry
			t.Run("replay", func(t *testing.T) {
				setupHermetic(t)
				// Another numbered directory from the first, so anything an
				// install takes from the name of the directory shows below.
				t.TempDir()
				installTemplates.mu.Lock()
				was := installTemplates.replays
				installTemplates.mu.Unlock()
				repo := tc.make(t)
				installTemplates.mu.Lock()
				now := installTemplates.replays
				installTemplates.mu.Unlock()
				if now != was+1 {
					t.Fatalf("the second fixture replayed nothing (%d replays, then %d)", was, now)
				}
				if name := readConfigT(t, repo)["meta"].(map[string]any)["project_name"]; name != filepath.Base(repo) {
					t.Errorf("project_name = %v, want this repository's name %s", name, filepath.Base(repo))
				}
				if date := readConfigT(t, repo)["meta"].(map[string]any)["setup_date"]; date != time.Now().UTC().Format("2006-01-02") {
					t.Errorf("setup_date = %v, want today", date)
				}
				roots, _ := installRoots(repo)
				replayed = captureRoots(t, roots)
			})
			t.Run("real", func(t *testing.T) {
				setupHermetic(t)
				repo := t.TempDir()
				tc.real(t, repo)
				roots, _ := installRoots(repo)
				real := captureRoots(t, roots)
				for name, want := range real {
					got, ok := replayed[name]
					switch {
					case !ok:
						t.Errorf("%s: a real install writes it and the replay does not", name)
					case got.mode != want.mode:
						t.Errorf("%s: mode %v replayed, %v real", name, got.mode, want.mode)
					case got.link != want.link:
						t.Errorf("%s: link %q replayed, %q real", name, got.link, want.link)
					case string(got.body) != string(want.body):
						t.Errorf("%s differs:\nreplayed:\n%s\nreal:\n%s", name, got.body, want.body)
					}
				}
				for name := range replayed {
					if _, ok := real[name]; !ok {
						t.Errorf("%s: the replay writes it and a real install does not", name)
					}
				}
			})
		})
	}
}

// TestAStateTheKeyDoesNotKnowGetsItsOwnInstall: an environment or file changed
// before the fixture, or a swapped seam, is a real install and never a replay of
// one kept from another starting state.
func TestAStateTheKeyDoesNotKnowGetsItsOwnInstall(t *testing.T) {
	replays := func() int {
		installTemplates.mu.Lock()
		defer installTemplates.mu.Unlock()
		return installTemplates.replays
	}
	fresh := func(t *testing.T) {
		t.Helper()
		setupHermetic(t)
		installedRepo(t)
	}
	t.Run("keep", fresh)
	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	for name, change := range map[string]func(t *testing.T){
		// Each with a value no earlier run in this process kept (-count).
		"a variable": func(t *testing.T) { t.Setenv("ABCD_TEMPLATE_TEST_VARIABLE", unique) },
		"a file in the home": func(t *testing.T) {
			writeFixture(t, os.Getenv("HOME"), "a-file-of-the-tests", unique, 0o644)
		},
		"a seam": func(t *testing.T) {
			swap := measureFits
			t.Cleanup(func() { measureFits = swap })
			measureFits = func(h PromptHelp, l question.Limits, def string) bool { return swap(h, l, def) }
		},
		"another package's seam": func(t *testing.T) {
			was := core.Version
			t.Cleanup(func() { core.Version = was })
			core.Version = "v0.16.0"
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			change(t)
			was := replays()
			installedRepo(t)
			if replays() != was {
				t.Errorf("the fixture replayed an install kept from another starting state")
			}
		})
	}
	t.Run("unchanged replays", func(t *testing.T) {
		was := replays()
		fresh(t)
		if replays() != was+1 {
			t.Errorf("an unchanged state did not replay")
		}
	})
}

// TestNoInstallRunsGh is what lets the key leave a fake gh's own script out: it
// names its test's log, and a replay must lose no call from that log.
func TestNoInstallRunsGh(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	idMustGit(t, repo, "remote", "add", "origin", "https://github.com/example-org/example-repo")
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(logPath); err == nil && len(raw) != 0 {
		t.Errorf("an install ran gh:\n%s", raw)
	}
}

func idMustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestWhatATemporaryFolderOnPathHoldsReachesTheKey: Detect walks PATH, so a folder
// named there is part of the starting state however it is laid out: a tool in a
// nested folder, a tool behind a link and an executable bit all count, and the
// same contents again replay, wherever the test's folders happen to be.
func TestWhatATemporaryFolderOnPathHoldsReachesTheKey(t *testing.T) {
	replays := func() int {
		installTemplates.mu.Lock()
		defer installTemplates.mu.Unlock()
		return installTemplates.replays
	}
	write := func(t *testing.T, path, body string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	// A value no earlier run in this process kept (-count).
	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	for name, onPath := range map[string]func(t *testing.T, body string, mode os.FileMode) string{
		"a tool in a named nested folder": func(t *testing.T, body string, mode os.FileMode) string {
			dir := filepath.Join(t.TempDir(), "bin")
			write(t, filepath.Join(dir, "sometool"), body, mode)
			return dir
		},
		"a tool below the folder named": func(t *testing.T, body string, mode os.FileMode) string {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "deeper", "sometool"), body, mode)
			return dir
		},
		"a tool behind a link": func(t *testing.T, body string, mode os.FileMode) string {
			dir, target := t.TempDir(), filepath.Join(t.TempDir(), "elsewhere")
			write(t, target, body, mode)
			if err := os.Symlink(target, filepath.Join(dir, "sometool")); err != nil {
				t.Skipf("no symlinks here: %v", err)
			}
			return dir
		},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := func(body string, mode os.FileMode) func(t *testing.T) int {
				return func(t *testing.T) int {
					t.Helper()
					setupHermetic(t)
					t.Setenv("PATH", onPath(t, unique+body, mode)+string(os.PathListSeparator)+os.Getenv("PATH"))
					was := replays()
					installedRepo(t)
					return replays() - was
				}
			}
			for _, step := range []struct {
				name string
				run  func(t *testing.T) int
				want int
			}{
				{"kept", fixture("a", 0o755), 0},
				{"other contents", fixture("b", 0o755), 0},
				{"another mode", fixture("a", 0o644), 0},
				{"the same again", fixture("a", 0o755), 1},
			} {
				t.Run(step.name, func(t *testing.T) {
					if got := step.run(t); got != step.want {
						t.Errorf("%d replays, want %d", got, step.want)
					}
				})
			}
		})
	}
}
