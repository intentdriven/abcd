package glossary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory to the directory holding
// go.mod.
func repoRoot(t *testing.T) string {
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
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}

// between returns the text the named markers enclose in doc.
func between(t *testing.T, doc, begin, end string) string {
	t.Helper()
	b := strings.Index(doc, begin)
	e := strings.Index(doc, end)
	if b < 0 || e < 0 {
		t.Fatalf("%s does not carry the markers %q / %q; the glossary index is derived from the term files, so the README must render it",
			READMERelPath, begin, end)
	}
	if e < b {
		t.Fatalf("%s: end marker %q precedes begin marker %q", READMERelPath, end, begin)
	}
	return strings.TrimSpace(doc[b+len(begin) : e])
}

// readREADME returns the committed glossary index page.
func readREADME(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(READMERelPath)))
	if err != nil {
		t.Fatalf("read %s: %v", READMERelPath, err)
	}
	return string(raw)
}

// scan reads the committed glossary directory.
func scan(t *testing.T) Glossary {
	t.Helper()
	g, err := Scan(repoRoot(t))
	if err != nil {
		t.Fatalf("scan glossary: %v", err)
	}
	return g
}

// TestGlossaryREADMECarriesTheRenderedIndex is the anti-drift detector for the
// term index: the term files are the source of truth and the README is a render
// of them. A term file added, removed, renamed, or re-defined without
// regenerating the index fails here.
func TestGlossaryREADMECarriesTheRenderedIndex(t *testing.T) {
	got := between(t, readREADME(t), IndexMarkerBegin, IndexMarkerEnd)
	want := strings.TrimSpace(RenderIndex(scan(t)))
	if got != want {
		t.Errorf("%s term index has drifted from the term files.\n\n--- README has ---\n%s\n\n--- RenderIndex wants ---\n%s",
			READMERelPath, got, want)
	}
}

// TestGlossaryREADMECarriesTheRenderedLayout holds the same contract for the
// directory layout, which is the other half of the hand-kept index that used to
// omit a whole bounded context.
func TestGlossaryREADMECarriesTheRenderedLayout(t *testing.T) {
	got := between(t, readREADME(t), LayoutMarkerBegin, LayoutMarkerEnd)
	want := strings.TrimSpace(RenderLayout(scan(t)))
	if got != want {
		t.Errorf("%s directory layout has drifted from the glossary directory.\n\n--- README has ---\n%s\n\n--- RenderLayout wants ---\n%s",
			READMERelPath, got, want)
	}
}

// TestScanFindsEveryBoundedContext states the failure that motivated the
// generator: the README listed two contexts while a third stood on disk. The
// scan reports what the directory holds, so a context cannot be invisible.
func TestScanFindsEveryBoundedContext(t *testing.T) {
	g := scan(t)
	if len(g.Contexts) < 3 {
		t.Fatalf("scan found %d bounded contexts; the glossary directory holds core, distribution, and interview", len(g.Contexts))
	}
	seen := map[string]int{}
	for _, ctx := range g.Contexts {
		seen[ctx.Name] = len(ctx.Terms)
	}
	for _, want := range []string{"core", "distribution", "interview"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("bounded context %q is absent from the scan", want)
			continue
		}
		if seen[want] == 0 {
			t.Errorf("bounded context %q scanned with no terms", want)
		}
	}
}

// TestScanSkipsTheTemplateAndContextREADMEs holds the term-file boundary: the
// template's placeholder term and the per-context READMEs are not glossary
// entries and must never reach the index.
func TestScanSkipsTheTemplateAndContextREADMEs(t *testing.T) {
	for _, ctx := range scan(t).Contexts {
		for _, term := range ctx.Terms {
			if term.File == "README.md" || strings.HasPrefix(term.File, "_") {
				t.Errorf("%s/%s is indexed as a term; only term files register terms", ctx.Name, term.File)
			}
			if term.Name == "example-term" {
				t.Errorf("the template's placeholder term reached the index via %s/%s", ctx.Name, term.File)
			}
		}
	}
}

// TestRenderIndexEscapesTableCells holds the rendering invariant a definition
// containing a pipe would otherwise break: the row must keep three columns.
func TestRenderIndexEscapesTableCells(t *testing.T) {
	g := Glossary{Contexts: []Context{{
		Name:  "core",
		Files: []string{"widget.md"},
		Terms: []Term{{Name: "widget", File: "widget.md", Status: "draft", Definition: "a | b"}},
	}}}
	row := RenderIndex(g)
	if !strings.Contains(row, `a \| b`) {
		t.Errorf("RenderIndex did not escape the pipe in a definition:\n%s", row)
	}
}

// TestScanReadsTermAliases pins the other spellings a term answers to. They are
// what lets a site link "roadmap phase" to the phase entry, and an alias field
// the scan dropped would be a spelling nothing reaches. An absent field and a
// YAML null are both "no aliases": a term called `null` is the shape
// iss-2608270908339164 already ruled out for the required fields.
func TestScanReadsTermAliases(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(DirRelPath), "core")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, aliases string) {
		body := "<!-- attribution -->\n---\nterm: " + name +
			"\nbounded_context: core\ndefinition: a definition of some length\n" +
			aliases + "status: stable\n---\n\n# " + name + "\n"
		if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("phase", "aliases: [\"roadmap phase\", 'sequencing phase']\n")
	write("spec", "aliases: null\n")
	write("brief", "")

	g, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, ctx := range g.Contexts {
		for _, term := range ctx.Terms {
			got[term.Name] = term.Aliases
		}
	}
	want := map[string][]string{
		"phase": {"roadmap phase", "sequencing phase"},
		"spec":  nil,
		"brief": nil,
	}
	for name, aliases := range want {
		if len(got[name]) != len(aliases) {
			t.Errorf("%s: aliases = %v, want %v", name, got[name], aliases)
			continue
		}
		for i, a := range aliases {
			if got[name][i] != a {
				t.Errorf("%s: alias %d = %q, want %q", name, i, got[name][i], a)
			}
		}
	}
}
