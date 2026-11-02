package site

// A project site kept beside a composed site (itd-2610142145236613): a
// repository whose `.abcd/site.json` is a composition keeps its project
// site's settings at `.abcd/project-site.json` and the workflow setup writes
// for it at `.github/workflows/project-site.yml`, and neither route reads the
// other's file. abcd's own repository is that repository: its documentation
// is composed for its own host, and its record is built on the project route
// onto its GitHub Pages, by the workflow setup writes for any project.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
)

// withComposedSite sets the harness's repository up with a composed site
// first, as a repository that already has one stands, and commits it.
func withComposedSite(t *testing.T, h *pagesHarness) {
	t.Helper()
	h.harness.run(t)
	if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
		t.Fatalf("the composed site was not set up: %v", err)
	}
	h.repo.Git("add", "-A")
	h.repo.Commit("the composed site")
}

func TestAProjectSiteIsSetUpBesideAComposition(t *testing.T) {
	h := newPagesHarness(t)
	withComposedSite(t, h)
	root := h.repo.Root()
	composition := mustRead(t, root, ManifestRelPath)
	composedWorkflow := mustRead(t, root, SiteWorkflowRelPath)

	res := h.run(t, asProject)
	if res.Status != StatusChanged {
		t.Fatalf("status = %q, notes %v", res.Status, res.Notes)
	}
	want := []string{ProjectSettingsRelPath, ProjectWorkflowRelPath}
	if got := changedPaths(t, h.harness); !reflect.DeepEqual(got, want) {
		t.Fatalf("the repository changed in\n  %v\nwant exactly\n  %v", got, want)
	}
	if !bytes.Equal(mustRead(t, root, ManifestRelPath), composition) || !bytes.Equal(mustRead(t, root, SiteWorkflowRelPath), composedWorkflow) {
		t.Fatal("setting the project site up beside the composed site changed the composed site's files")
	}
	q := strings.Join(h.ask.questions, "\n")
	if !strings.Contains(q, ProjectSettingsRelPath) || !strings.Contains(q, ProjectWorkflowRelPath) {
		t.Errorf("the switch question does not name the files it writes:\n%s", q)
	}

	m, err := loadRoute(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Project() || !m.Hosting.On() || m.Hosting.Provider != "github-pages" {
		t.Fatalf("the project route reads %+v, want a switched-on github-pages project site", m)
	}
	wf := string(mustRead(t, root, ProjectWorkflowRelPath))
	wantWF, err := renderSiteWorkflowAt("main", githubpages.Adapter{}, hosting.Site{Name: m.Hosting.Name, Repo: "example-owner/example-site"}, true, besidePaths)
	if err != nil {
		t.Fatal(err)
	}
	if wf != string(wantWF) {
		t.Fatal("the workflow written beside the composed site is not the one setup renders for those paths")
	}
	for _, s := range []string{"name: project-site\n", "group: project-site-production\n", "WORKFLOW: project-site.yml\n",
		"jq -e '.hosting.publish == true' .abcd/project-site.json", `git show "origin/main:.abcd/project-site.json"`,
		"site build --project --out site", "--base-path /example-site/"} {
		if !strings.Contains(wf, s) {
			t.Errorf("the workflow beside the composed site lacks %q", s)
		}
	}
	for _, s := range []string{".abcd/site.json both", "WORKFLOW: site.yml", "group: site-production"} {
		if strings.Contains(wf, s) {
			t.Errorf("the workflow beside the composed site still carries %q", s)
		}
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "`gh workflow run project-site.yml`") {
		t.Errorf("the notes do not name the workflow to dispatch: %v", res.Notes)
	}

	// Both routes build, each from its own file.
	h.repo.Git("add", "-A")
	h.repo.Commit("the project site")
	if _, err := Build(Request{RepoRoot: root, OutDir: t.TempDir(), Stamp: fixtureStamp, BasePath: "/example-site/", Project: true}); err != nil {
		t.Fatalf("the project site beside the composition does not build: %v", err)
	}
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: root, OutDir: out, Stamp: fixtureStamp}); err != nil {
		t.Fatalf("the composed site no longer builds beside the project site: %v", err)
	}

	// A second run rewrites nothing.
	again := h.run(t, asProject)
	for _, f := range again.Files {
		if f.Path != ProjectSettingsRelPath && f.Path != ProjectWorkflowRelPath {
			t.Errorf("the second run reported %s", f.Path)
		}
		if f.Status == "written" {
			t.Errorf("the second run rewrote %s", f.Path)
		}
	}
	if got := changedPaths(t, h.harness); len(got) != 0 {
		t.Fatalf("the second run changed %v", got)
	}
}

// asProject declares the project route, as --project does.
func asProject(r *SetupRequest) { r.Project = true }

// Without --project, a repository with a composition is set up as before: the
// route is never inferred from the provider, and nothing is written beside.
func TestWithoutProjectACompositionIsNotGivenAProjectSiteBeside(t *testing.T) {
	h := newPagesHarness(t)
	withComposedSite(t, h)
	_, err := Setup(pagesRequest(t, h))
	if err == nil || !strings.Contains(err.Error(), "already names the provider") {
		t.Fatalf("--provider github-pages without --project on a composed site: %v", err)
	}
	if got := changedPaths(t, h.harness); len(got) != 0 {
		t.Fatalf("a run without --project changed %v", got)
	}
}

// --project with a provider that publishes a composed site is refused before
// anything is written, whichever file it would have gone to.
func TestProjectWithAComposedProviderIsRefused(t *testing.T) {
	h := newPagesHarness(t)
	withComposedSite(t, h)
	_, err := Setup(SetupRequest{RepoRoot: h.repo.Root(), Project: true, Provider: "cloudflare", Asker: h.ask, Forge: h.forge})
	if err == nil || !strings.Contains(err.Error(), "publishes a composed site") {
		t.Fatalf("--project with cloudflare beside a composition: %v", err)
	}
	h2 := newPagesHarness(t)
	_, err = Setup(SetupRequest{RepoRoot: h2.repo.Root(), Project: true, Asker: h2.ask, Forge: h2.forge})
	if err == nil || !strings.Contains(err.Error(), "--project sets up a project site") {
		t.Fatalf("--project with the default provider in a repository with no site: %v", err)
	}
	for _, hh := range []*pagesHarness{h, h2} {
		if got := changedPaths(t, hh.harness); len(got) != 0 {
			t.Fatalf("a refused --project run changed %v", got)
		}
	}
}

// Switched off beside a composition, setup offers to remove the project
// site's workflow and never the composed site's.
func TestSwitchingOffBesideACompositionRemovesOnlyTheProjectWorkflow(t *testing.T) {
	h := newPagesHarness(t)
	withComposedSite(t, h)
	h.run(t, asProject)
	root := h.repo.Root()
	settings := strings.Replace(string(mustRead(t, root, ProjectSettingsRelPath)), `"publish": true`, `"publish": false`, 1)
	h.repo.Write(ProjectSettingsRelPath, settings)
	h.repo.Git("add", "-A")
	h.repo.Commit("the project site, switched off")

	res := h.run(t, asProject, func(r *SetupRequest) { r.Provider = "" })
	if _, err := os.Stat(filepath.Join(root, ProjectWorkflowRelPath)); !os.IsNotExist(err) {
		t.Fatalf("the project site's workflow was not removed: %v (files %+v)", err, res.Files)
	}
	if _, err := os.Stat(filepath.Join(root, SiteWorkflowRelPath)); err != nil {
		t.Fatalf("the composed site's workflow went too: %v", err)
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), ProjectSettingsRelPath) {
		t.Errorf("the notes do not name the file the switch is off in: %v", res.Notes)
	}
}

// The route is declared, never inferred: the file beside a composition is
// read only on the project route, holds a project site's settings only, and
// the composed route never reads it.
func TestTheFileBesideACompositionIsReadOnlyOnTheProjectRoute(t *testing.T) {
	f := newFixture(t)
	if _, err := loadRoute(f.Root(), true); err == nil || !strings.Contains(err.Error(), "is a composition") {
		t.Fatalf("the project route with no settings beside the composition: %v", err)
	}
	f.write(ProjectSettingsRelPath, projectSettingsJSON)
	m, err := loadRoute(f.Root(), true)
	if err != nil || !m.Project() {
		t.Fatalf("the project route does not read the settings beside the composition: %+v, %v", m, err)
	}
	if m, err := loadRoute(f.Root(), false); err != nil || m.Project() {
		t.Fatalf("the composed route reads the project settings: %+v, %v", m, err)
	}
	// The gate's notes about the settings name the file each route read.
	for project, want := range map[bool]string{true: ProjectSettingsRelPath, false: ManifestRelPath} {
		if _, from, err := loadRouteFrom(f.Root(), project); err != nil || from != want {
			t.Errorf("the route with project=%v read %q (%v), want %s", project, from, err, want)
		}
	}
	composition := mustRead(t, f.Root(), ManifestRelPath)
	f.write(ProjectSettingsRelPath, string(composition))
	if _, err := loadRoute(f.Root(), true); !errors.Is(err, ErrManifestInvalid) || !strings.Contains(err.Error(), ProjectSettingsRelPath) {
		t.Fatalf("a composition beside the composition is not refused, naming the file: %v", err)
	}
}

// abcd's own repository keeps its record on the project route (spc-2610142146250176
// step 2): its settings beside the documentation site's composition, switched
// on, and its workflow byte for byte the one `abcd site setup --provider
// github-pages` writes for any repository that keeps a composed site.
func TestAbcdsRecordIsOnTheProjectRouteSetupWrites(t *testing.T) {
	root := repoRoot()
	m, err := loadRoute(root, true)
	if err != nil {
		t.Fatalf("abcd's project route: %v", err)
	}
	if !m.Project() || m.Hosting == nil || m.Hosting.Provider != (githubpages.Adapter{}).Name() || !m.Hosting.On() {
		t.Fatalf("abcd's project settings = %+v, want github-pages, switched on", m.Hosting)
	}
	if !m.Record.IssueLedger || !m.Record.Decisions {
		t.Errorf("abcd's project site does not publish the issue ledger and decisions its record shows today: %+v", m.Record)
	}
	c, err := LoadManifest(root)
	if err != nil || c.Project() {
		t.Fatalf("abcd's documentation site is no longer composed at %s: %v", ManifestRelPath, err)
	}
	want, err := renderSiteWorkflowAt("main", githubpages.Adapter{}, hosting.Site{Name: m.Hosting.Name, Domain: m.Hosting.Domain, Repo: abcdReleaseRepo}, true, besidePaths)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ProjectWorkflowRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is not the workflow `abcd site setup` writes for abcd. "+
			"If a dependency bump changed its action pins, run `make scaffold-sync`, which carries them into "+
			"the site template, the github-pages adapter and the golden renders; never re-render over the "+
			"bump, which reverts it. Otherwise re-render it with `scripts/abcd.sh site setup --project --provider github-pages`", ProjectWorkflowRelPath)
	}
	if got := (githubpages.Adapter{}).Profile(hosting.Site{Name: m.Hosting.Name, Repo: abcdReleaseRepo}).Address; got != abcdRecordAddress {
		t.Fatalf("abcd's Pages address is %s, and abcdRecordAddress says %s", got, abcdRecordAddress)
	}
}

// abcdRecordAddress is where GitHub Pages serves abcd's record: the
// intentdriven owner's github.io host, below the repository's path.
const abcdRecordAddress = "https://intentdriven.github.io/abcd/"

// The README and the docs link abcd's record at its Pages address, never at
// the documentation host, which forwards the record's old addresses there
// (spc-2610142146250176).
func TestAbcdsDocsLinkTheRecordAtItsPagesAddress(t *testing.T) {
	root := repoRoot()
	files := []string{"README.md"}
	if err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			rel, _ := filepath.Rel(root, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	old := []string{"abcdev.app/record", "abcdev.app/contributors", "abcdev.app/references"}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range old {
			if bytes.Contains(data, []byte(o)) {
				t.Errorf("%s links abcd's record at %s; link it at %s", rel, o, abcdRecordAddress)
			}
		}
	}
	readme := string(mustRead(t, root, "README.md"))
	if !strings.Contains(readme, "]("+abcdRecordAddress+"record/)") {
		t.Errorf("README.md does not link abcd's record at %srecord/", abcdRecordAddress)
	}
}
