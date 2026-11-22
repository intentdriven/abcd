package site

// The screenshots audit (site-src/audit/overflow-audit.js, run by
// .github/workflows/site-screenshots.yml) measures two builds once abcd's
// documentation site is narrowed (spc-2610142146250176 step 4): the composed
// build abcdev.app serves, which holds the documentation's own pages and none
// of the record, and the project route's build, which holds the record. Each
// build's routes are named in site-src/audit/routes.json, which the audit
// reads, so these tests hold every route it names to a page that build
// really writes: a route the build does not write would be audited as a 404.

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// auditRoutesRel is the audit's route list, beside the audit.
const auditRoutesRel = "site-src/audit/routes.json"

// screenshotsWorkflowRel is the workflow that runs the audit.
const screenshotsWorkflowRel = ".github/workflows/site-screenshots.yml"

// auditTarget is one build the audit measures: the routes it loads, and, for
// the record, the record types it samples pages of from the build's export.
type auditTarget struct {
	Routes []string `json:"routes"`
	Sample *struct {
		Export string   `json:"export"`
		Types  []string `json:"types"`
	} `json:"sample"`
}

// auditRoutes is routes.json as the audit reads it.
type auditRoutes struct {
	Purpose string                 `json:"purpose"`
	Targets map[string]auditTarget `json:"targets"`
}

func readAuditRoutes(t *testing.T) auditRoutes {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(auditRoutesRel)))
	if err != nil {
		t.Fatalf("the audit's route list: %v", err)
	}
	var r auditRoutes
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		t.Fatalf("%s: %v", auditRoutesRel, err)
	}
	return r
}

// routeFile is the file a build writes for an audited route.
func routeFile(route string) string {
	return strings.TrimPrefix(path.Join(route, "index.html"), "/")
}

// The documentation target names only pages the narrowed composed build
// writes, samples nothing from a record export it does not write, and keeps
// the contributors and references pages, which stay on abcdev.app.
func TestTheScreenshotsAuditMeasuresOnlyPagesTheNarrowedDocsBuildWrites(t *testing.T) {
	docs, ok := readAuditRoutes(t).Targets["docs"]
	if !ok {
		t.Fatalf("%s names no docs target", auditRoutesRel)
	}
	f := newFixture(t)
	declareRecordElsewhere(t, f, false, true)
	out := t.TempDir()
	buildFixture(t, f, out)
	files := siteFiles(t, out)

	for _, route := range docs.Routes {
		if _, ok := files[routeFile(route)]; !ok {
			t.Errorf("the audit loads %s from the documentation site, which the narrowed build does not write", route)
		}
	}
	if docs.Sample != nil {
		t.Errorf("the docs target samples the record from %s, which the narrowed build does not write", docs.Sample.Export)
	}
	for _, want := range []string{"/", "/contributors/", "/references/"} {
		if !containsString(docs.Routes, want) {
			t.Errorf("the docs target no longer audits %s, which abcdev.app still serves", want)
		}
	}
}

// The record target names the record's pages as the project route builds
// them, and samples its record pages from the export that build writes.
func TestTheScreenshotsAuditMeasuresTheRecordOnTheProjectRoute(t *testing.T) {
	rec, ok := readAuditRoutes(t).Targets["record"]
	if !ok {
		t.Fatalf("%s names no record target", auditRoutesRel)
	}
	// Opted in to the issue ledger and the decision log, as abcd's own
	// project settings are.
	f := newProjectFixture(t)
	f.write(ManifestRelPath, strings.Replace(projectSettingsJSON, `"hosting"`, `"record": {"issue_ledger": true, "decisions": true}, "hosting"`, 1))
	f.addDecisions(false)
	_, files := buildProject(t, f)

	for _, route := range rec.Routes {
		if _, ok := files[routeFile(route)]; !ok {
			t.Errorf("the audit loads %s from the record's site, which the project route does not write", route)
		}
	}
	for _, want := range []string{"/record/", "/record/graph/"} {
		if !containsString(rec.Routes, want) {
			t.Errorf("the record target no longer audits %s", want)
		}
	}
	if rec.Sample == nil {
		t.Fatal("the record target samples no record pages")
	}
	raw, ok := files[rec.Sample.Export]
	if !ok {
		t.Fatalf("the record target samples from %s, which the project route does not write", rec.Sample.Export)
	}
	var export struct {
		Nodes []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &export); err != nil {
		t.Fatal(err)
	}
	for _, typ := range rec.Sample.Types {
		found := false
		for _, n := range export.Nodes {
			if n.Type != typ {
				continue
			}
			found = true
			if _, ok := files[routeFile("/record/"+typ+"/"+n.ID+"/")]; !ok {
				t.Errorf("the audit would sample /record/%s/%s/, which the project route does not write", typ, n.ID)
			}
		}
		if !found {
			t.Errorf("the project route's export holds no %s for the audit to sample", typ)
		}
	}
}

// auditRunRe is one run of the audit in the workflow: its target and the
// variable holding the address it measures.
var auditRunRe = regexp.MustCompile(`node site-src/audit/overflow-audit\.js (\w+) "\$(\w+)"`)

// The workflow builds the project route beside the composed site, below the
// base path GitHub Pages serves it at, and runs the audit once per target
// the route list names.
func TestTheScreenshotsWorkflowAuditsBothBuilds(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(screenshotsWorkflowRel)))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(raw)
	for _, want := range []string{
		`site build --out "$RUNNER_TEMP/site"`,
		`site build --project --base-path /abcd/ --out "$RUNNER_TEMP/pages/abcd"`,
		`- '.abcd/project-site.json'`,
		`- '` + auditRoutesRel + `'`,
	} {
		if !strings.Contains(wf, want) {
			t.Errorf("%s does not carry %s", screenshotsWorkflowRel, want)
		}
	}
	ran := map[string]bool{}
	for _, m := range auditRunRe.FindAllStringSubmatch(wf, -1) {
		ran[m[1]] = true
	}
	for name := range readAuditRoutes(t).Targets {
		if !ran[name] {
			t.Errorf("%s never runs the audit on its %s target", screenshotsWorkflowRel, name)
		}
	}
	if len(ran) != len(readAuditRoutes(t).Targets) {
		t.Errorf("%s runs the audit on %v, and %s names %d targets", screenshotsWorkflowRel, ran, auditRoutesRel, len(readAuditRoutes(t).Targets))
	}
}

// The audit reads its routes from the list rather than carrying its own, so
// the list these tests check is the one it loads.
func TestTheOverflowAuditReadsItsRoutesFromTheList(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "audit", "overflow-audit.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(raw)
	if !strings.Contains(js, "'routes.json'") {
		t.Errorf("overflow-audit.js does not read %s", auditRoutesRel)
	}
	if regexp.MustCompile(`'/record/[a-z-]*/?'`).MatchString(js) {
		t.Error("overflow-audit.js still lists record routes of its own")
	}
}
