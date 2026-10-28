package site

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
)

// TestTheWrittenWorkflowDeploysOnlyAReleaseTheDefaultBranchHolds: the render
// refuses a release whose commit the default branch does not hold, read from a
// freshly fetched copy of that branch before anything renders, and the deploy
// takes its host configuration from the default branch, never from the
// release's tag.
func TestTheWrittenWorkflowDeploysOnlyAReleaseTheDefaultBranchHolds(t *testing.T) {
	wf, err := renderSiteWorkflow("trunk", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(wf)

	render := codeOf(topJob(t, text, "render"))
	checkout := stepOf(t, render, "Check out the released commit")
	if !strings.Contains(checkout, "fetch-depth: 0") {
		t.Errorf("the render's checkout fetches no history, so the default branch is not there to read:\n%s", checkout)
	}
	check := stepOf(t, render, "Refuse a release the default branch does not hold")
	for _, want := range []string{
		"DEFAULT_BRANCH: 'trunk'",
		`git merge-base --is-ancestor HEAD "refs/remotes/origin/$DEFAULT_BRANCH"`,
		"exit 1",
	} {
		if !strings.Contains(check, want) {
			t.Errorf("the ancestry check lacks %q:\n%s", want, check)
		}
	}
	at := func(s string) int { return strings.Index(render, s) }
	if !(at(checkout) < at(check) && at(check) < at("- name: Download abcd")) {
		t.Errorf("the ancestry check does not run straight after the checkout, before abcd is fetched:\n%s", render)
	}

	deploy := codeOf(topJob(t, text, "deploy"))
	if !strings.Contains(deploy, "\n    needs: render\n") {
		t.Errorf("the deploy job does not wait on the render:\n%s", deploy)
	}
	host := stepOf(t, deploy, "Check out the default branch")
	if !strings.Contains(host, "ref: refs/heads/trunk") {
		t.Errorf("the deploy's checkout is not the default branch:\n%s", host)
	}
	if strings.Contains(deploy, "refs/tags/") {
		t.Errorf("the deploy job still reads the release's tag:\n%s", deploy)
	}

	relay := codeOf(topJob(t, text, "relay"))
	if !strings.Contains(relay, "DEFAULT_BRANCH: 'trunk'") {
		t.Errorf("the relay's branch is not a quoted string, so a branch named like a number is read as one:\n%s", relay)
	}
}

// TestTheSiteDeploysOnlyAReleaseMainHolds: abcd's own site workflow refuses a
// release whose commit main does not hold before it hands the commit to any
// job, and its deploy takes wrangler.jsonc from main, never from the release's
// commit.
func TestTheSiteDeploysOnlyAReleaseMainHolds(t *testing.T) {
	text := readWorkflow(t, siteWorkflowRel)

	resolve := codeOf(topJob(t, text, "resolve"))
	// main is read by its full ref, since a bare name can also name a tag.
	head := strings.Index(resolve, `gh api "repos/${GITHUB_REPOSITORY}/git/ref/heads/main" --jq '.object.sha'`)
	compare := strings.Index(resolve, `gh api "repos/${GITHUB_REPOSITORY}/compare/${sha}...${main_sha}" --jq '.status'`)
	if head < 0 || head > compare || strings.Contains(resolve, `...main"`) {
		t.Errorf("resolve does not read main's commit by its full ref before comparing (head %d, compare %d)", head, compare)
	}
	handed := strings.Index(resolve, `echo "sha=$sha" >> "$GITHUB_OUTPUT"`)
	if compare < 0 || handed < 0 || compare > handed {
		t.Errorf("resolve does not compare the release's commit with main before handing it on (compare %d, output %d)", compare, handed)
	}
	if !strings.Contains(resolve, "ahead | identical)") {
		t.Errorf("resolve does not admit only a commit main holds (status ahead or identical)")
	}

	deploy := codeOf(topJob(t, text, "deploy"))
	host := stepOf(t, deploy, "Check out the default branch")
	if !strings.Contains(host, "ref: refs/heads/main") {
		t.Errorf("the deploy's checkout is not main:\n%s", host)
	}
	if strings.Contains(deploy, "needs.resolve.outputs.sha") {
		t.Errorf("the deploy job still checks out the release's commit:\n%s", deploy)
	}
}
