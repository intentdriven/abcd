package site

// The link-targets gate (iss-2610252224215664): every /docs/ link on a
// composed page and every redirect target names a file in the built output.
//
// The composer writes a /docs/ link from the docs generator's URL shape
// (siteHref), and the redirect map names /docs/ routes by hand, but `/docs/`
// is the documentation generator's own tree: nothing in the site build can see
// it. The gate reads the output after the generator has written into it, so a
// generator that moved its URLs, or a page that went, fails here instead of on
// a visitor.

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The prose chapter, with a link into another docs page, so the composed
// landing page carries a /docs/ link deeper than the docs index.
const linkingProcessDoc = `# Process

The loop runs from the brief to the verdict and back.

![The loop](../assets/img/loop.svg)

## Capturing intents

Write an intent down the moment it exists, as [the roles](roles.md) say.

` + "```bash\nabcd capture \"a thought\"\n```" + `

Nothing else is required.
`

// writeBuiltDocs stands in for the documentation generator: it writes the docs
// tree into the output the way it serves it, every markdown page as a
// directory URL (README.md as its directory's index) and every other file as
// itself. directoryURLs false writes the generator's other shape, page.html,
// which serves none of the directory URLs the composer links.
func writeBuiltDocs(t *testing.T, r *checkRepo, directoryURLs bool) {
	t.Helper()
	src := filepath.Join(r.root, "docs")
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		dst := rel
		if strings.HasSuffix(rel, ".md") {
			route := strings.TrimSuffix(rel, ".md")
			if strings.EqualFold(filepath.Base(route), "README") {
				route = strings.TrimSuffix(strings.TrimSuffix(route, filepath.Base(route)), "/")
			}
			switch {
			case route == "":
				dst = "index.html"
			case directoryURLs:
				dst = route + "/index.html"
			default:
				dst = route + ".html"
			}
		}
		full := filepath.Join(r.out, "docs", filepath.FromSlash(dst))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, []byte("<!doctype html><title>docs</title>\n"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func wantNote(t *testing.T, res CheckResult, check, want string) {
	t.Helper()
	for _, n := range res.Notes {
		if n.Check == check && strings.Contains(n.Detail, want) {
			return
		}
	}
	t.Fatalf("no %s note names %q; notes %+v", check, want, res.Notes)
}

func TestLinkTargetsPassWhenTheDocsTreeServesEveryLink(t *testing.T) {
	r := newCheckRepo(t)
	r.write("docs/explanation/process.md", linkingProcessDoc)
	r.write("site-src/redirects", "# a comment\n\n/old/ /docs/explanation/roles/ 301\n/gone/ https://example.com/ 301\n/x/* /docs/:splat 301\n")
	r.build()
	writeBuiltDocs(t, r, true)
	res := r.check()
	if !res.OK() {
		t.Fatalf("a docs tree serving every link failed the gates: %+v", res.Findings)
	}
	if !containsString(res.Checks, CheckLinkTargets) {
		t.Fatalf("checks run = %v, want %s among them", res.Checks, CheckLinkTargets)
	}
	for _, n := range res.Notes {
		if n.Check == CheckLinkTargets {
			t.Errorf("a docs tree that is present left a link unchecked: %+v", n)
		}
	}
}

// TestLinkTargetsRefuseAMovedDocsURLShape is the issue's own scenario: the
// docs generator moves its URLs from page/ to page.html, the composer still
// links page/, and every other gate passes.
func TestLinkTargetsRefuseAMovedDocsURLShape(t *testing.T) {
	r := newCheckRepo(t)
	r.write("docs/explanation/process.md", linkingProcessDoc)
	r.build()
	writeBuiltDocs(t, r, false)
	res := r.check()
	wantFinding(t, res, CheckLinkTargets, "/docs/explanation/roles/")
	wantFinding(t, res, CheckLinkTargets, "docs/explanation/roles/index.html")
	for _, f := range findingsFor(res, CheckLinkTargets) {
		if f.Where != "index.html" {
			t.Errorf("the finding is not on the page that carries the link: %+v", f)
		}
		if f.Source == "" {
			t.Errorf("the finding names no source span for the link: %+v", f)
		}
	}
	// The docs index is served as index.html in both shapes, so /docs/ itself
	// is not among the findings.
	for _, f := range findingsFor(res, CheckLinkTargets) {
		if strings.Contains(f.Detail, "=\"/docs/\"") {
			t.Errorf("the docs index was refused though the tree serves it: %+v", f)
		}
	}
}

func TestLinkTargetsRefuseARedirectToNothing(t *testing.T) {
	r := newCheckRepo(t)
	r.write("site-src/redirects", "/old/ /docs/explanation/gone/ 301\n/here/ /docs/ 301\n/also/ /record/ 301\n/nowhere/ /not-built/ 301\n")
	r.build()
	writeBuiltDocs(t, r, true)
	res := r.check()
	wantFinding(t, res, CheckLinkTargets, "/docs/explanation/gone/")
	wantFinding(t, res, CheckLinkTargets, "/not-built/")
	for _, f := range findingsFor(res, CheckLinkTargets) {
		if f.Where != "site-src/redirects" {
			t.Errorf("a redirect finding does not name the redirect map's source: %+v", f)
		}
		if strings.Contains(f.Detail, "/here/") || strings.Contains(f.Detail, "/also/") {
			t.Errorf("a redirect to a built page was refused: %+v", f)
		}
	}
	if n := len(findingsFor(res, CheckLinkTargets)); n != 2 {
		t.Errorf("findings = %d, want 2: %+v", n, findingsFor(res, CheckLinkTargets))
	}
}

// The site-render gate builds without the docs generator, so the output holds
// no docs tree at all: the /docs/ targets cannot be judged, which is a note
// naming how many, never a pass dressed as a check or a failure of every link.
// A redirect target outside /docs/ is still judged.
func TestLinkTargetsNoteADocsTreeThatIsAbsent(t *testing.T) {
	r := newCheckRepo(t)
	r.write("docs/explanation/process.md", linkingProcessDoc)
	r.write("site-src/redirects", "/old/ /docs/explanation/roles/ 301\n/nowhere/ /not-built/ 301\n")
	r.build()
	res := r.check()
	wantFinding(t, res, CheckLinkTargets, "/not-built/")
	if n := len(findingsFor(res, CheckLinkTargets)); n != 1 {
		t.Errorf("findings = %d, want only the non-docs redirect: %+v", n, findingsFor(res, CheckLinkTargets))
	}
	wantNote(t, res, CheckLinkTargets, "docs/")
}
