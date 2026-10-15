package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// topJob is the text of one job of a workflow: from its key under `jobs:` to
// the next job's key.
func topJob(t *testing.T, workflow, job string) string {
	t.Helper()
	start := strings.Index(workflow, "\n  "+job+":\n")
	if start < 0 {
		t.Fatalf("no job %q", job)
	}
	rest := workflow[start+1:]
	next := regexp.MustCompile(`\n  [A-Za-z0-9_-]+:\n`).FindStringIndex(rest[1:])
	if next == nil {
		return rest
	}
	return rest[:next[0]+1]
}

func readWorkflow(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestTheReleaseIsPublishedOnlyOnceEveryAssetIsAttached: release.yml creates
// the Release as a draft with the binaries, the archive and checksums.txt;
// site.yml's render attaches site.tar.gz to that draft, and only then does its
// publish job publish the Release, before the deploy moves the website. A
// published Release never has an asset added to it.
func TestTheReleaseIsPublishedOnlyOnceEveryAssetIsAttached(t *testing.T) {
	release := readWorkflow(t, releaseWorkflowRel)
	site := readWorkflow(t, siteWorkflowRel)

	rel := topJob(t, release, "release")
	if !strings.Contains(rel, `gh release create "${TAG}" bin/abcd-* bin/checksums.txt --draft `) {
		t.Error("release.yml must create the Release as a --draft with its built assets")
	}
	if strings.Contains(release, "--draft=false") {
		t.Error("release.yml must leave the Release a draft; site.yml publishes it once site.tar.gz is attached")
	}

	render := topJob(t, site, "render")
	upload := strings.Index(render, `gh release upload "${TAG}" site.tar.gz`)
	if upload < 0 {
		t.Fatal("the render must attach site.tar.gz to the Release")
	}
	uploadStep := render[strings.LastIndex(render[:upload], "- name:"):upload]
	if !strings.Contains(uploadStep, "if: steps.state.outputs.draft == 'true'") {
		t.Errorf("site.tar.gz is attached only while the Release is a draft:\n%s", uploadStep)
	}
	if !strings.Contains(render, "draft: ${{ steps.state.outputs.draft }}") {
		t.Error("the render must hand the Release's draft state to the publish job")
	}

	publish := topJob(t, site, "publish")
	for _, want := range []string{
		"needs: [resolve, render]",
		"if: ${{ needs.render.outputs.draft == 'true' }}",
		"contents: write",
	} {
		if !strings.Contains(publish, want) {
			t.Errorf("the publish job must carry %q:\n%s", want, publish)
		}
	}
	edit := strings.Index(publish, `gh release edit "${TAG}" --repo "${GITHUB_REPOSITORY}" --draft=false`)
	if edit < 0 {
		t.Fatal("the publish job must publish the draft Release")
	}
	for _, check := range []string{"checksums.txt", "site.tar.gz"} {
		if i := strings.Index(publish, check); i < 0 || i > edit {
			t.Errorf("the publish job must check %s is attached before it publishes", check)
		}
	}

	deploy := topJob(t, site, "deploy")
	if !strings.Contains(deploy, "needs: [resolve, render, publish]") ||
		!strings.Contains(deploy, "(needs.publish.result == 'success' || needs.publish.result == 'skipped')") {
		t.Errorf("the deploy must follow publication (or its skip, when the Release was already published):\n%s", deploy[:min(len(deploy), 600)])
	}
	order := []string{"\n  resolve:\n", "\n  render:\n", "\n  publish:\n", "\n  deploy:\n"}
	last := -1
	for _, key := range order {
		i := strings.Index(site, key)
		if i <= last {
			t.Errorf("site.yml's jobs must run resolve, render, publish, deploy in that order")
		}
		last = i
	}
}
