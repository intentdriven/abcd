package site

import (
	"os"
	"os/exec"
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

// publishFakeGH answers the four gh calls site.yml's publish step makes: the
// checksums.txt download, the asset listing (applying the step's own --jq
// filter to the canned assets), the publish edit, and the draft read-back.
const publishFakeGH = `#!/usr/bin/env bash
set -euo pipefail
sub="$1 $2"; shift 3
expr=""; dir=""
while [ $# -gt 0 ]; do
  case "$1" in
    --jq) expr="$2"; shift 2 ;;
    --dir) dir="$2"; shift 2 ;;
    --repo|--pattern|--json) shift 2 ;;
    --draft=false) : > "$FAKE_DIR/published"; shift ;;
    *) echo "fake gh: unexpected $1" >&2; exit 97 ;;
  esac
done
case "$sub" in
  "release download") printf '%s  abcd-linux-amd64\n%s  abcd-plugin-v1.2.3.zip\n' aaa bbb > "$dir/checksums.txt" ;;
  "release view")
    if [ "$expr" = .isDraft ]; then
      if [ -f "$FAKE_DIR/published" ]; then echo false; else echo true; fi
    else
      jq -r "$expr" "$FAKE_DIR/release.json"
    fi ;;
  "release edit") ;;
  *) echo "fake gh: $sub is not faked" >&2; exit 99 ;;
esac
`

// runBlockOf returns the script of the first `run: |` block in text.
func runBlockOf(t *testing.T, text string) string {
	t.Helper()
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "run: |" {
			continue
		}
		key := len(l) - len(strings.TrimLeft(l, " "))
		var body []string
		indent := -1
		for _, b := range lines[i+1:] {
			if strings.TrimSpace(b) == "" {
				body = append(body, "")
				continue
			}
			n := len(b) - len(strings.TrimLeft(b, " "))
			if n <= key {
				break
			}
			if indent < 0 {
				indent = n
			}
			body = append(body, b[indent:])
		}
		return strings.Join(body, "\n") + "\n"
	}
	t.Fatal("no run: | block")
	return ""
}

// TestPublishRequiresEveryAssetUploaded executes site.yml's publish step
// against a fake gh: a Release is published only when every asset
// checksums.txt names, checksums.txt and site.tar.gz are attached in the
// uploaded state. An asset whose upload has not completed counts as missing.
func TestPublishRequiresEveryAssetUploaded(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	script := runBlockOf(t, topJob(t, readWorkflow(t, siteWorkflowRel), "publish"))
	names := []string{"abcd-linux-amd64", "abcd-plugin-v1.2.3.zip", "checksums.txt", "site.tar.gz"}
	for name, c := range map[string]struct {
		pending string // an asset whose upload has not completed
		publish bool
	}{
		"every asset uploaded":               {publish: true},
		"a binary still uploading":           {pending: "abcd-linux-amd64"},
		"the site archive still uploading":   {pending: "site.tar.gz"},
		"the checksums file still uploading": {pending: "checksums.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := t.TempDir()
			bin := t.TempDir()
			var assets []string
			for _, n := range names {
				state := "uploaded"
				if n == c.pending {
					state = "open"
				}
				assets = append(assets, `{"name":"`+n+`","state":"`+state+`"}`)
			}
			if err := os.WriteFile(filepath.Join(fake, "release.json"), []byte(`{"assets":[`+strings.Join(assets, ",")+`]}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(publishFakeGH), 0o755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-eo", "pipefail", "-c", script)
			cmd.Env = []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
				"FAKE_DIR=" + fake, "TAG=v1.2.3", "GITHUB_REPOSITORY=example/fixture", "RUNNER_TEMP=" + t.TempDir()}
			out, err := cmd.CombinedOutput()
			_, statErr := os.Stat(filepath.Join(fake, "published"))
			published := statErr == nil
			if c.publish {
				if err != nil || !published {
					t.Fatalf("every asset is uploaded, so the Release must publish (err=%v, published=%v):\n%s", err, published, out)
				}
				return
			}
			if err == nil || published {
				t.Fatalf("%s is not uploaded, so the Release must not publish (err=%v, published=%v):\n%s", c.pending, err, published, out)
			}
			if !strings.Contains(string(out), c.pending) {
				t.Errorf("the refusal must name %s:\n%s", c.pending, out)
			}
		})
	}
}
