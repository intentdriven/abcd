package site

// A project site's content policy rides in its pages (iss-2610142036270003).
// GitHub Pages serves no response header a site declares and reads no
// `_headers`, so a project build writes the policy the binary's headers file
// gives a route into each page's head, as a <meta http-equiv> element ahead of
// everything the page loads, and the gate holds every page to it.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var cspMetaRe = regexp.MustCompile(`<meta http-equiv="Content-Security-Policy" content="([^"]*)">`)

// binaryPolicy is the content policy the binary's headers file gives a route,
// less frame-ancestors, which a policy delivered in a page is not allowed to
// carry. It reads the file through the test's own parser, not the build's.
func binaryPolicy(t *testing.T, route string) string {
	t.Helper()
	data, err := setupSources.ReadFile("setupsrc/headers")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range parseHeaders(string(data)) {
		if b.pattern != route {
			continue
		}
		csp := b.headers["Content-Security-Policy"]
		if csp == "" {
			t.Fatalf("the binary's headers give %s no content policy", route)
		}
		if !strings.Contains(csp, "frame-ancestors 'none'; ") {
			t.Fatalf("the binary's policy for %s no longer refuses framing the way this test strips it: %s", route, csp)
		}
		return strings.Replace(csp, "frame-ancestors 'none'; ", "", 1)
	}
	t.Fatalf("the binary's headers have no block for %s", route)
	return ""
}

// decodeAttr undoes the five entities escapeAttr writes.
func decodeAttr(s string) string {
	return strings.NewReplacer("&#39;", "'", "&quot;", `"`, "&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(s)
}

func TestEveryProjectPageCarriesItsContentPolicyInItsHead(t *testing.T) {
	f := newProjectFixture(t)
	_, files := buildProject(t, f)
	about, record := binaryPolicy(t, "/"), binaryPolicy(t, "/record/*")
	pages := 0
	for rel, data := range files {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		pages++
		page := string(data)
		ms := cspMetaRe.FindAllStringSubmatchIndex(page, -1)
		if len(ms) != 1 {
			t.Errorf("%s carries %d content-policy elements, want one", rel, len(ms))
			continue
		}
		got := decodeAttr(page[ms[0][2]:ms[0][3]])
		want := record
		if rel == "index.html" {
			want = about
		}
		if got != want {
			t.Errorf("%s carries the policy\n  %s\nwant the binary's\n  %s", rel, got, want)
		}
		if strings.Contains(got, "frame-ancestors") {
			t.Errorf("%s carries frame-ancestors, which a policy in a page cannot enforce", rel)
		}
		// The policy governs only what is fetched after it is read, so it
		// stands ahead of every stylesheet and script the head loads.
		head, _, _ := strings.Cut(page, "</head>")
		for _, load := range []string{"<link ", "<script "} {
			if i := strings.Index(head, load); i >= 0 && i < ms[0][0] {
				t.Errorf("%s loads %s before its content policy", rel, strings.TrimSpace(load))
			}
		}
	}
	if pages < 2 {
		t.Fatalf("the project build wrote %d pages; the test needs the about page and the record", pages)
	}
	if _, ok := files["record/index.html"]; !ok {
		t.Fatal("the project build wrote no record dashboard")
	}
}

// The composed route's policy stays on its host, in `_headers`: its pages
// carry no element of their own, so the two never disagree.
func TestAComposedPageCarriesNoContentPolicyElement(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatal(err)
	}
	for rel, data := range siteFiles(t, out) {
		if strings.HasSuffix(rel, ".html") && cspMetaRe.Match(data) {
			t.Errorf("composed page %s carries a content-policy element", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "_headers")); err != nil {
		t.Errorf("the composed build no longer writes _headers: %v", err)
	}
}

// The gate's content-policy half: a project page with no policy, a weaker
// one, or one read only after the page has loaded its stylesheet fails.
func TestTheProjectGateHoldsEveryPageToItsContentPolicy(t *testing.T) {
	f := newProjectFixture(t)
	out, files := buildProject(t, f)
	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true})
	if err != nil {
		t.Fatal(err)
	}
	wantClean(t, res, CheckContentPolicy)
	if !containsString(res.Checks, CheckContentPolicy) {
		t.Errorf("a project run reports the checks %v, without %s", res.Checks, CheckContentPolicy)
	}

	var recordPage string
	for rel := range files {
		if strings.HasPrefix(rel, "record/adr/") && strings.HasSuffix(rel, "index.html") {
			recordPage = rel
			break
		}
	}
	if recordPage == "" {
		t.Fatal("the build wrote no ADR page")
	}
	for name, c := range map[string]struct {
		rel  string
		edit func(page string) string
		want string
	}{
		"no policy": {"index.html", func(p string) string { return cspMetaRe.ReplaceAllString(p, "") }, "carries no content policy"},
		"a weaker policy": {recordPage, func(p string) string {
			return strings.Replace(p, "script-src &#39;self&#39;", "script-src &#39;self&#39; &#39;unsafe-inline&#39;", 1)
		}, "not the policy"},
		"the record's policy on the about page": {"index.html", func(p string) string {
			return strings.Replace(p, "connect-src &#39;none&#39;", "connect-src &#39;self&#39;", 1)
		}, "not the policy"},
		"read after the stylesheet": {recordPage, func(p string) string {
			m := cspMetaRe.FindString(p)
			p = strings.Replace(p, m+"\n", "", 1)
			return strings.Replace(p, "</head>", m+"\n</head>", 1)
		}, "after"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for rel, data := range files {
				dst := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					t.Fatal(err)
				}
				if rel == c.rel {
					data = []byte(c.edit(string(data)))
					if string(data) == string(files[rel]) {
						t.Fatalf("the edit changed nothing on %s", rel)
					}
				}
				if err := os.WriteFile(dst, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: dir, Project: true})
			if err != nil {
				t.Fatal(err)
			}
			wantFinding(t, res, CheckContentPolicy, c.want)
			for _, fd := range findingsFor(res, CheckContentPolicy) {
				if fd.Where != c.rel {
					t.Errorf("a content-policy finding on %s, which was not edited: %+v", fd.Where, fd)
				}
			}
		})
	}
}
