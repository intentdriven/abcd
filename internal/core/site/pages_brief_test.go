package site

// The brief's page set and its written list name every page a build writes
// (iss-2611040554479009). The brief said every repository gets seven pages and
// that a build writes nothing beyond its list, while the explorer also writes
// the contributors, references, decision-log, development and foundations
// pages; a page the code writes and the specification does not name is a page
// nobody reviewed the switch of.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// explorerRoutes is every top-level route Pages writes (one record's own page
// sits under the explorer's root and is named by it).
var explorerRoutes = []string{
	routeDashboard, routeGraph, routeFoundations, routeDevelopment, routeDecisionLog,
	routeHealth, routeGlossary, routeContributors, routeReferences,
}

const siteBriefRel = ".abcd/development/brief/04-surfaces/22-site.md"

// briefSpan returns the text from the first line starting with from up to the
// first later line starting with to (to included when inclusive).
func briefSpan(t *testing.T, doc, from, to string) string {
	t.Helper()
	i := strings.Index(doc, "\n"+from)
	if i < 0 {
		t.Fatalf("%s has no line starting %q", siteBriefRel, from)
	}
	rest := doc[i+1:]
	j := strings.Index(rest[len(from):], to)
	if j < 0 {
		t.Fatalf("%s has no %q after %q", siteBriefRel, to, from)
	}
	return rest[:len(from)+j+len(to)]
}

func TestBriefPageSetNamesEveryPageTheExplorerWrites(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(siteBriefRel)))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	set := briefSpan(t, doc, "## The page set", "\n## ")
	for _, r := range explorerRoutes {
		if !strings.Contains(set, "`/"+r+"`") {
			t.Errorf("the page set in %s does not name `/%s`", siteBriefRel, r)
		}
	}
	written := briefSpan(t, doc, "The build reads the repository and nothing else", "Nothing else, nowhere else.")
	for _, r := range explorerRoutes {
		if !strings.Contains(written, "`/"+r+"`") {
			t.Errorf("the written list in %s does not name `/%s`", siteBriefRel, r)
		}
	}
}

// The brief's project-site section names where a project site kept beside a
// composition lives (iss-2611040554507299). It said setup writes the site's
// settings and workflow, and that the build and the gate refuse settings of
// the other shape, without naming ProjectSettingsRelPath or
// ProjectWorkflowRelPath, which setup writes beside a composition and the
// project route reads first.
func TestBriefProjectSiteNamesTheSideBySideFiles(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(siteBriefRel)))
	if err != nil {
		t.Fatal(err)
	}
	section := briefSpan(t, string(raw), "## A project's own site", "\n## ")
	for _, rel := range []string{ManifestRelPath, SiteWorkflowRelPath, ProjectSettingsRelPath, ProjectWorkflowRelPath} {
		if !strings.Contains(section, "`"+rel+"`") {
			t.Errorf("the project-site section of %s does not name `%s`", siteBriefRel, rel)
		}
	}
}
