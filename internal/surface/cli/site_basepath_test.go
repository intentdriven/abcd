package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestSiteVerbsTakeABasePath: `site build` and `lint site` both take the path
// the site is served below (itd-2610091010348892), and both refuse one that is
// not plain path segments before reading anything else.
func TestSiteVerbsTakeABasePath(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	t.Chdir(repo.Root())
	for _, verb := range [][]string{{"site", "build"}, {"lint", "site"}} {
		out := filepath.Join(t.TempDir(), "site")
		args := append(append([]string{}, verb...), "--out", out, "--base-path", "/../elsewhere/")
		got, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "base path") {
			t.Fatalf("`abcd %s --base-path /../elsewhere/` = exit %d, %v:\n%s", strings.Join(verb, " "), code, err, got)
		}
		args = append(append([]string{}, verb...), "--out", out, "--base-path", "/widgets/")
		_, err = runCLIErr(t, args...)
		if err == nil || strings.Contains(err.Error(), "base path") || strings.Contains(err.Error(), "unknown flag") {
			t.Fatalf("`abcd %s --base-path /widgets/` was refused for its base path: %v", strings.Join(verb, " "), err)
		}
	}
}

// TestSiteVerbsTakeProject: `site build` and `lint site` both name the project
// route with --project (itd-2610091010348892), and both refuse a project
// site's settings without it: the route is declared, never inferred.
func TestSiteVerbsTakeProject(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/site.json", `{"schema_version": 1, "mode": "project", "purpose": "x", `+
		`"hosting": {"provider": "github-pages", "name": "widgets", "publish": true}}`+"\n")
	repo.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-01-02\n\n- The first cut.\n")
	repo.Commit("a project site's settings")
	// A GitHub origin names the site; with no identity block either, the
	// gate refuses a site with no title.
	repo.Git("remote", "add", "origin", "https://github.com/widgets-owner/widgets.git")
	t.Chdir(repo.Root())
	for _, verb := range [][]string{{"site", "build"}, {"lint", "site"}} {
		out := filepath.Join(t.TempDir(), "site")
		args := append(append([]string{}, verb...), "--out", out)
		got, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "pass --project") {
			t.Fatalf("`abcd %s` over project settings = exit %d, %v:\n%s", strings.Join(verb, " "), code, err, got)
		}
		got, err = runCLIErr(t, append(args, "--project")...)
		if err != nil {
			t.Fatalf("`abcd %s --project` = %v:\n%s", strings.Join(verb, " "), err, got)
		}
	}
}
