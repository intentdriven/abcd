package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestSelectsWhatLostPreflights: each of these failed a preflight 42 to 91
// minutes in on 6 and 7 November after precheck had passed
// (iss-2611061656036069), and each reaches the root another way: through a
// package helper around fsutil.ModuleRoot, a local walk up to go.mod, and a
// filepath.Join of parent directories. It reads the live tree, so renaming one
// of them means changing this list too.
func TestSelectsWhatLostPreflights(t *testing.T) {
	got, err := selectAudits(filepath.Join("..", ".."), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"TestCommandSurfaceResolvesBinaryFromPluginRoot",
		"TestTestGitCallsAreHermetic",
		"TestNoPrivateDelimiterCompare",
	} {
		if !slices.Contains(got.names, name) {
			t.Errorf("%s is not selected", name)
		}
	}
	for _, pkg := range []string{"./internal/core/launch", "./internal/gittest", "./internal/core/frontmatter"} {
		if !slices.Contains(got.packages, pkg) {
			t.Errorf("%s is not selected", pkg)
		}
	}
}

// TestSelectionFollowsWhatATestDoes holds the detector to both sides of the
// line: what reaches the committed tree is chosen, and a test that only runs a
// script, or writes a go.mod of its own, is not.
func TestSelectionFollowsWhatATestDoes(t *testing.T) {
	const head = "package x\nimport (\n\t\"os\"\n\t\"os/exec\"\n\t\"path/filepath\"\n\t\"testing\"\n\t\"github.com/intentdriven/abcd/internal/fsutil\"\n)\nvar _, _, _ = os.Getwd, exec.Command, filepath.Join\n"
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(head+body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/a/a_test.go", `
func TestJoinClimb(t *testing.T) { _ = filepath.Join("..", "..", "commands") }
func TestLiteralClimb(t *testing.T) { _, _ = os.ReadFile("../../AGENTS.md") }
func TestModuleRoot(t *testing.T) { _, _ = fsutil.ModuleRoot(".") }
func TestThroughHelper(t *testing.T) { _ = rootHelper(t) }
func TestThroughTwoHelpers(t *testing.T) { _ = outerHelper(t) }
func TestWalkUp(t *testing.T) { _ = walkUp() }
func TestInAClosure(t *testing.T) { t.Run("x", func(t *testing.T) { _ = rootHelper(t) }) }
func TestThroughAPackageValue(t *testing.T) { _ = treeRoot }
func TestSpawnsItselfToListFiles(t *testing.T) {
	_ = exec.Command("git", "-C", rootHelper(t), "ls-files").Run()
}
func testlowercaseIsNotATest(t *testing.T) { _ = rootHelper(t) }
var treeRoot = filepath.Join("..", "..", "..")
func rootHelper(t *testing.T) string { r, _ := fsutil.ModuleRoot("."); return r }
func outerHelper(t *testing.T) string { return rootHelper(t) }
func walkUp() string {
	d, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		d = filepath.Dir(d)
	}
}
`)
	write("internal/b/b_test.go", `
func TestRunsAHook(t *testing.T) { runHook(t) }
func TestWritesItsOwnGoMod(t *testing.T) {
	_ = os.WriteFile(filepath.Join(t.TempDir(), "go.mod"), nil, 0o644)
}
func TestMain(m *testing.M) { _ = filepath.Join("..", "..") }
func runHook(t *testing.T) {
	_ = exec.Command("bash", filepath.Join(rootHelper(t), "hooks", "x")).Run()
}
func rootHelper(t *testing.T) string { r, _ := fsutil.ModuleRoot("."); return r }
`)
	write("cmd/c/c_test.go", `func TestInACommand(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/skipped/s_test.go", `func TestSkippedPackage(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/a/testdata/fixture_test.go", `func TestInTestdata(t *testing.T) { _ = filepath.Join("..", "..") }`)
	write("internal/a/a.go", `func TestNotInATestFile() { _ = filepath.Join("..", "..") }`)

	got, err := selectAudits(root, []string{"./internal/skipped"})
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"TestInAClosure", "TestInACommand", "TestJoinClimb", "TestLiteralClimb", "TestModuleRoot", "TestSpawnsItselfToListFiles",
		"TestThroughAPackageValue", "TestThroughHelper", "TestThroughTwoHelpers", "TestWalkUp",
	}
	if !slices.Equal(got.names, wantNames) {
		t.Errorf("names = %v\nwant    %v", got.names, wantNames)
	}
	if want := []string{"./cmd/c", "./internal/a"}; !slices.Equal(got.packages, want) {
		t.Errorf("packages = %v, want %v", got.packages, want)
	}
}

// TestAFileThatDoesNotParseIsAnError: a test file the detector cannot read is
// one whose audits it would silently stop choosing.
func TestAFileThatDoesNotParseIsAnError(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "a")
	for _, d := range []string{dir, filepath.Join(root, "cmd")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package x\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := selectAudits(root, nil); err == nil {
		t.Fatal("a test file that does not parse was read past")
	}
}
