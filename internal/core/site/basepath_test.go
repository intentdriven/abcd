package site

// The base path (itd-2610091010348892): a site served below the root of its
// host — a project's GitHub Pages address, owner.github.io/<repo>/ — links
// every page, stylesheet, script, picture and record route under that path,
// and the gate resolves what it reads against the same prefixed root.

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNormaliseBasePath(t *testing.T) {
	for in, want := range map[string]string{
		"":              "/",
		"/":             "/",
		"widgets":       "/widgets/",
		"/widgets":      "/widgets/",
		"widgets/":      "/widgets/",
		"/widgets/":     "/widgets/",
		"/a/b.c_d~e-f/": "/a/b.c_d~e-f/",
	} {
		got, err := NormaliseBasePath(in)
		if err != nil || got != want {
			t.Errorf("NormaliseBasePath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{
		"..", "/../x/", "/a/../b/", "/./", "//", "/a//b/", "https://example.com/", "//example.com/",
		"/a b/", "/a?b/", "/a#b/", "/%2e%2e/", `/a"b/`, "/a<b/", "/a\\b/", "/é/",
	} {
		if got, err := NormaliseBasePath(in); err == nil {
			t.Errorf("NormaliseBasePath(%q) = %q, want a refusal", in, got)
		}
	}
}

// rootAbsoluteRe finds every root-absolute URL an attribute carries: an href,
// a src, and the graph's data address.
var rootAbsoluteRe = regexp.MustCompile(`(?:href|src|data-graph)="(/[^"]*)"`)

// siteFiles reads every file a build wrote, keyed output-relative, the marker
// aside.
func siteFiles(t *testing.T, out string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(out, p)
		data, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func buildFixtureAt(t *testing.T, f *fixture, base string) (string, map[string][]byte) {
	t.Helper()
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp, BasePath: base}); err != nil {
		t.Fatalf("build at %q: %v", base, err)
	}
	return out, siteFiles(t, out)
}

func TestABasePathPrefixesEveryRootAbsoluteURL(t *testing.T) {
	f := newFixture(t)
	_, files := buildFixtureAt(t, f, "/widgets/")
	seen := 0
	for rel, data := range files {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		for _, m := range rootAbsoluteRe.FindAllSubmatch(data, -1) {
			u := string(m[1])
			if strings.HasPrefix(u, "//") {
				continue
			}
			seen++
			if !strings.HasPrefix(u, "/widgets/") {
				t.Errorf("%s links %q, outside the base path /widgets/", rel, u)
			}
		}
	}
	if seen == 0 {
		t.Fatal("the build carries no root-absolute URL at all, so the test proves nothing")
	}
	for _, want := range []string{`href="/widgets/site.css"`, `src="/widgets/site.js"`, `href="/widgets/record/"`, `data-graph="/widgets/record.json"`} {
		found := false
		for _, data := range files {
			if bytes.Contains(data, []byte(want)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no page carries %s", want)
		}
	}
}

// The default output does not move: a build that names no base path and one
// that names "/" write the same files byte for byte (the golden files pin
// those bytes against the render before the base path existed), and a build
// below a base path differs from the default only by that prefix.
func TestTheDefaultBasePathChangesNoByte(t *testing.T) {
	f := newFixture(t)
	_, def := buildFixtureAt(t, f, "")
	_, root := buildFixtureAt(t, f, "/")
	_, widgets := buildFixtureAt(t, f, "/widgets/")
	if len(def) != len(root) || len(def) != len(widgets) {
		t.Fatalf("file counts differ: default %d, root %d, widgets %d", len(def), len(root), len(widgets))
	}
	for rel, want := range def {
		if !bytes.Equal(root[rel], want) {
			t.Errorf("%s: a build at / differs from the default build", rel)
		}
		got := widgets[rel]
		for _, attr := range []string{"href", "src", "data-graph"} {
			got = bytes.ReplaceAll(got, []byte(attr+`="/widgets/`), []byte(attr+`="/`))
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: a build at /widgets/ differs from the default by more than the prefix", rel)
		}
	}
}

func TestTheGateResolvesAgainstTheBasePath(t *testing.T) {
	f := newFixture(t)
	out, _ := buildFixtureAt(t, f, "/widgets/")

	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, BasePath: "/widgets/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, fd := range res.Findings {
		if fd.Check == CheckBasePath || (fd.Check == CheckMobile && strings.Contains(fd.Detail, "links the stylesheet")) {
			t.Errorf("a build at its own base path was refused: %+v", fd)
		}
	}

	// The same output judged as if served at the root links stylesheets the
	// output does not carry at those paths.
	res, err = Check(CheckRequest{RepoRoot: f.Root(), OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(res, CheckMobile, "widgets/site.css") {
		t.Errorf("a prefixed build checked at the root resolved its stylesheet anyway: %+v", res.Findings)
	}

	// A root-absolute link that lacks the prefix leaves the site.
	page := filepath.Join(out, "index.html")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`href="/widgets/record/"`), []byte(`href="/record/"`), 1)
	if err := os.WriteFile(page, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, BasePath: "/widgets/"})
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(res, CheckBasePath, "/record/") {
		t.Errorf("an unprefixed link was not refused: %+v", res.Findings)
	}
}

func TestTheGateRefusesAnUnusableBasePath(t *testing.T) {
	f := newFixture(t)
	if _, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: t.TempDir(), BasePath: "/../x/"}); err == nil {
		t.Fatal("the gate accepted a base path that climbs out of the site")
	}
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), BasePath: "/a b/"}); err == nil {
		t.Fatal("the build accepted a base path that is not plain path segments")
	}
}

func hasFinding(res CheckResult, check, detail string) bool {
	for _, fd := range res.Findings {
		if fd.Check == check && strings.Contains(fd.Detail, detail) {
			return true
		}
	}
	return false
}

// TestTheBasePathGateIsDeniedAPassOnAnUnparsedPage: a page that does not parse
// is never walked, so the base-path gate cannot report ok for it; a link that
// leaves the site on that page would otherwise disappear behind the parse
// fault.
func TestTheBasePathGateIsDeniedAPassOnAnUnparsedPage(t *testing.T) {
	f := newFixture(t)
	out, _ := buildFixtureAt(t, f, "/widgets/")
	page := filepath.Join(out, "index.html")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	broken := bytes.Replace(raw, []byte(`<footer class="site-foot">`), []byte(`<footer class="site-foot"><!-- smuggled -->`), 1)
	if bytes.Equal(broken, raw) {
		t.Fatal("the fixture's index page has no footer to break")
	}
	if err := os.WriteFile(page, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, BasePath: "/widgets/"})
	if err != nil {
		t.Fatal(err)
	}
	denied := false
	for _, fd := range res.Findings {
		if fd.Check == CheckBasePath && fd.Where == "index.html" && strings.Contains(fd.Detail, "did not parse") {
			denied = true
		}
	}
	if !denied {
		t.Fatalf("the base-path gate passed a page it never read: %+v", res.Findings)
	}
}

// Every attribute a page can carry a site URL in is held to the base path: a
// responsive image's every candidate, a video's poster and an SVG reference
// as well as href and src. A URL that starts with the prefix but climbs out
// of it with a `..` segment leaves the site as surely as one without it.
func TestTheGateReadsEveryURLAttributeAndRefusesAClimb(t *testing.T) {
	f := newFixture(t)
	out, _ := buildFixtureAt(t, f, "/widgets/")
	page := filepath.Join(out, "index.html")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	inside := `<img src="/widgets/a.png" srcset="/widgets/a.png 1x, /widgets/a2.png 2x" alt="">` +
		`<video poster="/widgets/p.png"></video><svg><use xlink:href="/widgets/s.svg#i"></use></svg>`
	outside := `<img src="/widgets/b.png" srcset="/widgets/b.png 1x, /srcset-out.png 2x" alt="">` +
		`<video poster="/poster-out.png"></video><svg><use xlink:href="/xlink-out.svg#i"></use></svg>` +
		`<a href="/widgets/../climb-out/">x</a><img src="/widgets/./../dots-out.png" alt="">` +
		`<a href="/widgets/%2e%2e/encoded-out/">x</a>`
	marker := []byte(`<main id="app">`)
	if !bytes.Contains(raw, marker) {
		t.Fatal("the fixture's index page has no main element to add to")
	}
	if err := os.WriteFile(page, bytes.Replace(raw, marker, append(marker, inside+outside...), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, BasePath: "/widgets/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/srcset-out.png", "/poster-out.png", "/xlink-out.svg", "/widgets/../climb-out/",
		"/widgets/./../dots-out.png", "/widgets/%2e%2e/encoded-out/"} {
		if !hasFinding(res, CheckBasePath, want) {
			t.Errorf("the gate let %s through", want)
		}
	}
	for _, fd := range res.Findings {
		if fd.Check != CheckBasePath {
			continue
		}
		for _, ok := range []string{"/widgets/a.png", "/widgets/a2.png", "/widgets/p.png", "/widgets/s.svg"} {
			if strings.Contains(fd.Detail, ok) {
				t.Errorf("a URL inside the base path was refused: %+v", fd)
			}
		}
	}
}
