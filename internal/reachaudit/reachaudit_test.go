package reachaudit

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// writeTree lays files (slash paths relative to root) under root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// callerFile is a production file in package c calling p.<name> through an
// import of the audited package, behind the given header (a build line, or
// nothing).
func callerFile(header, name string) string {
	return header + "package c\n\nimport \"example.com/m/internal/core/p\"\n\nvar _ = p." + name + "\n"
}

// TestScanCountsOnlyProductionSelectors is the matcher's own detector. Each
// exported function of the audited package below is named by exactly one
// thing, and only a selector on the imported package in Go the release build
// compiles is a caller: a comment, a string literal, a file under testdata/, a
// `//go:build ignore` file, an eval-only tag, a local variable shadowing the
// import's name, a test file, a nested module and a platform no release target
// builds are not. A caller under another import name, a package name that is
// not its directory's, or a file only one release target compiles is.
func TestScanCountsOnlyProductionSelectors(t *testing.T) {
	root := t.TempDir()
	var decls strings.Builder
	decls.WriteString("package p\n\n")
	for _, fn := range []string{
		"Real", "Renamed", "LinuxOnly",
		"InComment", "InString", "InTestdata", "Ignored", "EvalOnly",
		"Shadowed", "InTest", "InNested", "WindowsOnly", "Uncalled",
	} {
		decls.WriteString("func " + fn + "() {}\n")
	}
	decls.WriteString("func SeamForTest() {}\n\ntype T struct{}\n\nfunc (T) Method() {}\n\nfunc unexported() {}\n")
	writeTree(t, root, map[string]string{
		"go.mod":                    "module example.com/m\n\ngo 1.26\n",
		"internal/core/p/p.go":      decls.String(),
		"internal/core/q/q.go":      "package named\n\nfunc Differ() {}\n",
		"cmd/c/real.go":             callerFile("", "Real"),
		"cmd/c/renamed.go":          "package c\n\nimport alias \"example.com/m/internal/core/p\"\n\nvar _ = alias.Renamed\n",
		"cmd/c/differ.go":           "package c\n\nimport \"example.com/m/internal/core/q\"\n\nvar _ = named.Differ\n",
		"cmd/c/only_linux.go":       callerFile("", "LinuxOnly"),
		"cmd/c/only_windows.go":     callerFile("", "WindowsOnly"),
		"cmd/c/text.go":             "package c\n\nimport \"example.com/m/internal/core/p\"\n\n// p.InComment is named only here.\nvar _ = \"p.InString\"\nvar _ = p.Real\n",
		"cmd/c/testdata/fixture.go": callerFile("", "InTestdata"),
		"cmd/c/ignored.go":          callerFile("//go:build ignore\n\n", "Ignored"),
		"cmd/c/eval.go":             callerFile("//go:build evals\n\n", "EvalOnly"),
		"cmd/c/shadow.go":           "package c\n\nimport \"example.com/m/internal/core/p\"\n\nvar _ = p.Real\n\nfunc f() {\n\tp := struct{ Shadowed int }{}\n\t_ = p.Shadowed\n}\n",
		"cmd/c/shadow_param.go":     "package c\n\nimport \"example.com/m/internal/core/p\"\n\nvar _ = p.Real\n\nfunc g(p struct{ Shadowed int }) int { return p.Shadowed }\n",
		"cmd/c/c_test.go":           "package c\n\nimport \"example.com/m/internal/core/p\"\n\nvar _ = p.InTest\n",
		"nested/go.mod":             "module example.com/nested\n\ngo 1.26\n",
		"nested/n.go":               "package n\n\nimport \"example.com/m/internal/core/p\"\n\nvar _ = p.InNested\n",
	})

	reach, err := Scan(root, "internal/core")
	if err != nil {
		t.Fatal(err)
	}
	want := Reach{
		"internal/core/p.Real":        true,
		"internal/core/p.Renamed":     true,
		"internal/core/p.LinuxOnly":   true,
		"internal/core/q.Differ":      true,
		"internal/core/p.InComment":   false,
		"internal/core/p.InString":    false,
		"internal/core/p.InTestdata":  false,
		"internal/core/p.Ignored":     false,
		"internal/core/p.EvalOnly":    false,
		"internal/core/p.Shadowed":    false,
		"internal/core/p.InTest":      false,
		"internal/core/p.InNested":    false,
		"internal/core/p.WindowsOnly": false,
		"internal/core/p.Uncalled":    false,
	}
	if !reflect.DeepEqual(reach, want) {
		for k, v := range want {
			if got, ok := reach[k]; !ok || got != v {
				t.Errorf("%s: reached=%v (present=%v), want reached=%v", k, got, ok, v)
			}
		}
		for k := range reach {
			if _, ok := want[k]; !ok {
				t.Errorf("%s audited, want it left out (a method, an unexported or a ForTest name)", k)
			}
		}
	}

	scoped, err := Scan(root, "internal/core/q")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scoped, Reach{"internal/core/q.Differ": true}) {
		t.Fatalf("Scan scoped to internal/core/q = %v; want only its own package", scoped)
	}
}

// TestScanFailsLoudOnAnUnparsableFile: a production file the audit cannot parse
// is an error, never a file that names no caller.
func TestScanFailsLoudOnAnUnparsableFile(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"go.mod":               "module example.com/m\n\ngo 1.26\n",
		"internal/core/p/p.go": "package p\n\nfunc Real() {}\n",
		"cmd/c/broken.go":      "package c\n\nfunc {\n",
	})
	if _, err := Scan(root, "internal/core"); err == nil {
		t.Fatal("Scan over an unparsable production file returned no error")
	}
}

// TestTargetsMatchTheMakefile holds Targets to the release build's own list: a
// target added there and not here would leave a caller in a file only that
// target compiles uncounted.
func TestTargetsMatchTheMakefile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^TARGETS\s*:?=\s*(.+)$`).FindSubmatch(data)
	if m == nil {
		t.Fatal("the Makefile declares no TARGETS line")
	}
	if got := strings.Fields(string(m[1])); !reflect.DeepEqual(got, Targets) {
		t.Fatalf("Makefile TARGETS = %v; reachaudit.Targets = %v", got, Targets)
	}
}
