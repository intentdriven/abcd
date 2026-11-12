package testreuse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// classifyText classifies one test file of the given source.
func classifyText(t *testing.T, spawns bool, src string) Verdict {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x_test.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return ClassifyFiles([]string{path}, spawns, nil)
}

// TestTheClassifierSendsWhatTheCacheCannotSeeToTheRerunLine: each kind of read
// Go's test cache cannot see, spelled the way this repository's tests spell it,
// sends the test that makes it to the always-rerun line, and a test that does
// none of them stays reusable.
func TestTheClassifierSendsWhatTheCacheCannotSeeToTheRerunLine(t *testing.T) {
	const head = "package x\nimport (\n\t\"net/http\"\n\t\"net/http/httptest\"\n\t\"os\"\n\t\"os/exec\"\n\t\"path/filepath\"\n\t\"testing\"\n\t\"time\"\n)\n" +
		"var _ = http.Get\nvar _ = httptest.NewServer\nvar _ = os.Getwd\nvar _ = exec.Command\nvar _ = filepath.Join\nvar _ testing.T\nvar _ = time.Now\n"
	cases := []struct {
		name   string
		spawns bool
		body   string
		want   string
	}{
		{"go build", true, `func TestA(t *testing.T) { exec.Command("go", "build", "./cmd/abcd").Run() }`, whyProgram},
		{"go build with no subprocess elsewhere", false, `func TestA(t *testing.T) { exec.Command("go", "build", "./cmd/abcd").Run() }`, whyProgram},
		{"bash", true, `func TestA(t *testing.T) { exec.Command("bash", "scripts/check.sh").Run() }`, whyProgram},
		{"make", true, `func TestA(t *testing.T) { exec.Command("make", "-n", "preflight").Run() }`, whyProgram},
		{"through a helper", true, "func TestA(t *testing.T) { helper() }\nfunc helper() { deeper() }\nfunc deeper() { exec.Command(\"sh\", \"-c\", \"x\").Run() }", whyProgram},
		{"through a method", true, "type c struct{}\nfunc (c) run() { exec.Command(\"go\", \"test\").Run() }\nfunc TestA(t *testing.T) { c{}.run() }", whyProgram},
		{"through a variable a helper assigns", true, "var bin string\nfunc build() { bin = \"x\"; exec.Command(\"go\", \"build\").Run() }\nfunc TestA(t *testing.T) { _ = bin }", whyProgram},
		{"through a variable's initialiser", true, "var root = find()\nfunc find() string { d, _ := os.Getwd(); return d }\nfunc TestA(t *testing.T) { _ = root }", whyTree},
		{"git asked for the repository", true, `func TestA(t *testing.T) { exec.Command("git", "rev-parse", "--show-toplevel").Run() }`, whyTree},
		{"a walk up in a package that spawns", true, `func TestA(t *testing.T) { d, _ := os.Getwd(); _ = d }`, whyTree},
		{"a climb to the repository in a package that spawns", true, `func TestA(t *testing.T) { _ = filepath.Join("..", "..", "Makefile") }`, whyTree},
		{"a relative path handed to a program", true, `func TestA(t *testing.T) { exec.Command("gofmt", "./cmd/x").Run() }`, whyTree},
		{"a skip outside a checkout", true, `func TestA(t *testing.T) { t.Skip("not in a git checkout") }`, whyTree},
		{"a skip where git cannot answer", true, `func TestA(t *testing.T) { t.Skipf("git ls-files unavailable in %s", "x") }`, whyTree},
		{"an absolute path to a shell", true, `func TestA(t *testing.T) { exec.Command("/bin/bash", "x.sh").Run() }`, whyProgram},
		{"a shell found on PATH", true, `func TestA(t *testing.T) { p, _ := exec.LookPath("bash"); exec.Command(p, "x.sh").Run() }`, whyProgram},
		{"a command built by hand", true, `func TestA(t *testing.T) { c := exec.Cmd{Path: "/usr/local/bin/x"}; c.Run() }`, whyProgram},
		{"a binary a helper builds and TestMain's assignment holds", true, "var helperBin string\nfunc build(p string) { exec.Command(\"gofmt\", p).Run(); exec.Command(\"go\", \"build\", \"-o\", p).Run() }\nfunc TestMain(m *testing.M) { helperBin = \"x\"; build(helperBin); m.Run() }\nfunc TestA(t *testing.T) { exec.Command(helperBin).Run() }", whyProgram},
		{"a command declared by hand", true, `func TestA(t *testing.T) { var c exec.Cmd; c.Run() }`, whyProgram},
		{"a command made with new", true, `func TestA(t *testing.T) { c := new(exec.Cmd); c.Run() }`, whyProgram},
		{"a climb held in a constant", true, "const up = \"../..\"\nfunc TestA(t *testing.T) { _ = filepath.Join(up, \"Makefile\") }", whyTree},
		{"a climb held in a local constant", true, `func TestA(t *testing.T) { const up = "../.."; _ = filepath.Join(up, "Makefile") }`, whyTree},
		{"a field TestMain assigns from a builder", true, "type s struct{ bin string }\nvar stub s\nfunc build() string { exec.Command(\"go\", \"build\").Run(); return \"x\" }\nfunc TestMain(m *testing.M) { stub.bin = build(); m.Run() }\nfunc TestA(t *testing.T) { _ = stub.bin }", whyProgram},
		{"a variable a builder fills through a pointer", true, "var bin string\nfunc build(p *string) { exec.Command(\"go\", \"build\").Run(); *p = \"x\" }\nfunc TestMain(m *testing.M) { build(&bin); m.Run() }\nfunc TestA(t *testing.T) { _ = bin }", whyProgram},
		{"a program name held in a local variable", true, `func TestA(t *testing.T) { name := "bash"; exec.Command(name, "x.sh").Run() }`, whyUnnamed},
		{"a program name held in a local variable, in a package with no subprocess elsewhere", false, `func TestA(t *testing.T) { name := "go"; exec.Command(name, "build").Run() }`, whyUnnamed},
		{"a program name held in a local variable, behind a context", true, `func TestA(t *testing.T) { name := "make"; exec.CommandContext(t.Context(), name, "preflight").Run() }`, whyUnnamed},
		{"a program name a helper's parameter holds", true, "func run(name string) { exec.Command(name, \"-c\", \"x\").Run() }\nfunc TestA(t *testing.T) { run(\"sh\") }", whyUnnamed},
		{"a program name joined from parts", true, `func TestA(t *testing.T) { exec.Command("ba"+"sh", "x.sh").Run() }`, whyUnnamed},
		{"a program name held in a constant", true, "const shell = \"bash\"\nfunc TestA(t *testing.T) { exec.Command(shell, \"x.sh\").Run() }", whyUnnamed},
		{"the test binary run again", true, `func TestA(t *testing.T) { exec.Command(os.Args[0], "-test.run=TestA").Run() }`, whyUnnamed},
		{"a parenthesised literal names its program", true, `func TestA(t *testing.T) { d := t.TempDir(); exec.Command(("git"), "-C", d, "init").Run() }`, ""},
		{"the network", true, `func TestA(t *testing.T) { http.Get("https://example.invalid") }`, whyNetwork},
		{"the clock against a fixed date", true, `func TestA(t *testing.T) { _ = time.Now().After(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)) }`, whyClock},
		{"the clock against a date string", true, `func TestA(t *testing.T) { _ = time.Now(); _ = "2030-01-01" }`, whyClock},

		{"a tracked file opened in a package that cannot spawn", false, `func TestA(t *testing.T) { _ = filepath.Join("..", "..", "Makefile") }`, ""},
		{"a testdata file", true, `func TestA(t *testing.T) { os.ReadFile("testdata/x") }`, ""},
		{"git in a fixture", true, `func TestA(t *testing.T) { d := t.TempDir(); exec.Command("git", "-C", d, "init").Run() }`, ""},
		{"go env reads the toolchain", true, `func TestA(t *testing.T) { exec.Command("go", "env", "GOCACHE").Run() }`, ""},
		{"a local server", true, `func TestA(t *testing.T) { s := httptest.NewServer(nil); http.Get(s.URL) }`, ""},
		{"a skip for another reason", true, `func TestA(t *testing.T) { t.Skip("no pty on this machine") }`, ""},
		{"elapsed time", true, `func TestA(t *testing.T) { _ = time.Since(time.Now()) }`, ""},
		{"a fixed date with no clock", true, `func TestA(t *testing.T) { _ = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) }`, ""},
		{"a command held but not made here", true, `func TestA(t *testing.T) { var c *exec.Cmd; _ = c }`, ""},
		{"a constant that climbs nowhere", true, "const name = \"Makefile\"\nfunc TestA(t *testing.T) { _ = filepath.Join(t.TempDir(), name) }", ""},
		{"a local variable sharing a helper's name", true, "func helper() { exec.Command(\"go\", \"build\").Run() }\nfunc TestA(t *testing.T) { helper := 1; _ = helper }", ""},
		{"a blank assignment beside a blank variable", true, "var _ = shell()\nfunc shell() string { exec.Command(\"sh\", \"-c\", \"x\").Run(); return \"\" }\nfunc TestA(t *testing.T) { _, _ = os.ReadFile(\"testdata/x\") }", ""},
		{"a field sharing a helper's name", true, "func env() { exec.Command(\"go\", \"build\").Run() }\ntype s struct{ env int }\nfunc TestA(t *testing.T) { v := s{env: 1}; _ = v.env }", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := classifyText(t, c.spawns, head+c.body+"\n")
			if v.Whole != "" {
				t.Fatalf("the whole package was sent over: %s", v.Whole)
			}
			got := v.Tests["TestA"]
			if c.want == "" {
				if got != "" {
					t.Fatalf("a reusable test was sent to the rerun line: %s", got)
				}
				return
			}
			if !strings.HasPrefix(got, c.want) {
				t.Fatalf("got %q; want %q", got, c.want)
			}
		})
	}
}

// TestTheModuleRootHelperCountsAsReachingTheTree: the repository's own helper for
// finding the root is a walk to the repository like any other, in a package that
// starts subprocesses.
func TestTheModuleRootHelperCountsAsReachingTheTree(t *testing.T) {
	const src = "package x\nimport (\n\t\"testing\"\n\t\"github.com/intentdriven/abcd/internal/fsutil\"\n)\nfunc TestA(t *testing.T) { fsutil.ModuleRoot(\"x\") }\n"
	if v := classifyText(t, true, src); !strings.HasPrefix(v.Tests["TestA"], whyTree) {
		t.Errorf("fsutil.ModuleRoot stayed reusable: %+v", v)
	}
}

// TestSetUpThatRunsAProgramSendsTheWholePackage: TestMain and init run before
// every test, so what their own bodies do counts against every test, while what
// they only name (a stand-in a re-execution branch runs) counts for no test.
func TestSetUpThatRunsAProgramSendsTheWholePackage(t *testing.T) {
	const head = "package x\nimport (\n\t\"os\"\n\t\"os/exec\"\n\t\"testing\"\n)\nvar _ = os.Getwd\nfunc TestA(t *testing.T) {}\n"
	for _, name := range []string{"TestMain(m *testing.M)", "init()"} {
		v := classifyText(t, true, head+"func "+name+" { exec.Command(\"go\", \"build\").Run() }\n")
		if !strings.Contains(v.Whole, whyProgram) {
			t.Errorf("%s runs go, and the verdict is %+v", name, v)
		}
	}
	v := classifyText(t, true, head+"func standIn() { exec.Command(\"sh\", \"-c\", \"x\").Run() }\nfunc TestMain(m *testing.M) { if os.Getenv(\"X\") != \"\" { standIn() }; m.Run() }\n")
	if v.Whole != "" || len(v.Tests) != 0 {
		t.Errorf("a TestMain that only names a stand-in moved the package: %+v", v)
	}
}

// TestAFileThatDoesNotParseSendsTheWholePackage: what cannot be read cannot be
// shown safe.
func TestAFileThatDoesNotParseSendsTheWholePackage(t *testing.T) {
	v := classifyText(t, false, "package x\nfunc TestA(t *testing.T) {")
	if v.Whole == "" {
		t.Fatalf("a test file that does not parse left the package reusable: %+v", v)
	}
}

// TestAHelperPackageCountsAsWhatTheTestReaches: gittest.Abcd builds the binary
// from the tree in a package no test file sits in, and a package holds one as a
// variable, so its methods count against every test that reads the variable.
func TestAHelperPackageCountsAsWhatTheTestReaches(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper.go")
	const src = "package helper\nimport (\n\t\"os/exec\"\n\t\"testing\"\n)\nvar _ testing.T\ntype Bin struct{}\nfunc (b *Bin) Path() string { b.build(); return \"\" }\nfunc (b *Bin) build() { exec.Command(\"go\", \"build\").Run() }\nfunc Plain() {}\nfunc Build() { exec.Command(\"go\", \"build\").Run() }\n"
	if err := os.WriteFile(helper, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	test := filepath.Join(dir, "x_test.go")
	const body = "package x\nimport (\n\t\"testing\"\n\t\"example.com/m/helper\"\n)\nvar built helper.Bin\nfunc TestHolds(t *testing.T) { built.Path() }\nfunc TestCalls(t *testing.T) { helper.Build() }\nfunc TestPlain(t *testing.T) { helper.Plain() }\n"
	if err := os.WriteFile(test, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgs := []Package{
		{ImportPath: "example.com/m/helper", Dir: dir, GoFiles: []string{"helper.go"}, Imports: []string{"os/exec", "testing"}},
		{ImportPath: "example.com/m/x", Dir: dir, XTestGoFiles: []string{"x_test.go"}, XTestImports: []string{"example.com/m/helper", "testing"}},
	}
	v := Survey(pkgs)["example.com/m/x"]
	if v.Tests["TestHolds"] == "" || v.Tests["TestCalls"] == "" {
		t.Errorf("a test reaching the helper's build stayed reusable: %+v", v)
	}
	if v.Tests["TestPlain"] != "" {
		t.Errorf("a test calling a harmless helper went to the rerun line: %+v", v)
	}
}

// TestAHelperThatCannotBeReadSendsItsImportersOver: a helper package that does not
// parse says nothing about what a test reaches through it.
func TestAHelperThatCannotBeReadSendsItsImportersOver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.go"), []byte("package helper\nfunc Broken( {"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x_test.go"), []byte("package x\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgs := []Package{
		{ImportPath: "m/helper", Dir: dir, GoFiles: []string{"helper.go"}, Imports: []string{"testing"}},
		{ImportPath: "m/x", Dir: dir, TestGoFiles: []string{"x_test.go"}, TestImports: []string{"m/helper", "testing"}},
		{ImportPath: "m/other", Dir: dir, TestGoFiles: []string{"x_test.go"}, TestImports: []string{"testing"}},
	}
	got := Survey(pkgs)
	if got["m/x"].Whole == "" {
		t.Errorf("an importer of an unreadable helper stayed reusable: %+v", got["m/x"])
	}
	if got["m/other"].Whole != "" {
		t.Errorf("a package that does not import it was sent over: %+v", got["m/other"])
	}
}

// TestSpawningIsReadThroughTheImportClosure: a test that reaches the tree is
// sent over when anything it links can start a subprocess, however deep.
func TestSpawningIsReadThroughTheImportClosure(t *testing.T) {
	dir := t.TempDir()
	test := filepath.Join(dir, "x_test.go")
	if err := os.WriteFile(test, []byte("package x\nimport (\n\t\"path/filepath\"\n\t\"testing\"\n)\nfunc TestA(t *testing.T) { _ = filepath.Join(\"..\", \"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mk := func(deep string) []Package {
		return []Package{
			{ImportPath: "m/x", Dir: dir, TestGoFiles: []string{"x_test.go"}, Imports: []string{"m/a"}},
			{ImportPath: "m/a", Imports: []string{"m/b"}},
			{ImportPath: "m/b", Imports: []string{deep}},
		}
	}
	if v := Survey(mk("os/exec"))["m/x"]; v.Tests["TestA"] == "" {
		t.Errorf("os/exec two imports away left the test reusable: %+v", v)
	}
	if v := Survey(mk("strings"))["m/x"]; v.Tests["TestA"] != "" {
		t.Errorf("a package that cannot spawn lost the reuse of a read the cache sees: %+v", v)
	}
}
