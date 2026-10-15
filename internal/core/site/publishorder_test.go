package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// codeOf drops a workflow text's comment lines, so a check reads what runs.
func codeOf(text string) string {
	var keep []string
	for _, l := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "#") {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}

// stepOf is the text of the step whose name starts with name, up to the next
// step of the same job.
func stepOf(t *testing.T, job, name string) string {
	t.Helper()
	at := strings.Index(job, "- name: "+name)
	if at < 0 {
		t.Fatalf("no step %q", name)
	}
	rest := job[at:]
	if next := strings.Index(rest[1:], "\n      - name:"); next >= 0 {
		return rest[:next+1]
	}
	return rest
}

// TestTheReleaseIsPublishedOnlyOnceEveryAssetIsAttached: release.yml creates
// the Release as a draft with the binaries, the archive and checksums.txt and
// hands its numeric id to site.yml; site.yml's render attaches site.tar.gz to
// that draft, and only then does its publish job publish the Release, before
// the deploy moves the website. A published Release never has an asset added
// to it, and every read and write of the draft names it by its id.
func TestTheReleaseIsPublishedOnlyOnceEveryAssetIsAttached(t *testing.T) {
	release := readWorkflow(t, releaseWorkflowRel)
	site := readWorkflow(t, siteWorkflowRel)

	rel := topJob(t, release, "release")
	if !strings.Contains(rel, `gh release create "${TAG}" bin/abcd-* bin/checksums.txt --draft `) {
		t.Error("release.yml must create the Release as a --draft with its built assets")
	}
	if strings.Contains(codeOf(release), "draft=false") {
		t.Error("release.yml must leave the Release a draft; site.yml publishes it once site.tar.gz is attached")
	}
	if !strings.Contains(rel, "release_id: ${{ steps.draft.outputs.id }}") {
		t.Error("the release job must hand on the numeric id of the draft it created")
	}
	selfVerify := codeOf(stepOf(t, rel, "Verify the draft"))
	if strings.Contains(selfVerify, "gh release download") || !strings.Contains(selfVerify, `releases/${RELEASE_ID}/assets`) {
		t.Errorf("release.yml's own check of the draft must read it by its id:\n%s", selfVerify)
	}
	if !strings.Contains(topJob(t, release, "site"), "release_id: ${{ needs.release.outputs.release_id }}") {
		t.Error("release.yml's site job must pass the draft's id to site.yml")
	}

	render := codeOf(topJob(t, site, "render"))
	if strings.Contains(render, "gh release ") {
		t.Errorf("the render must read and write the Release by its id, never by tag:\n%s", render)
	}
	upload := stepOf(t, render, "Attach site.tar.gz")
	if !strings.Contains(upload, "if: steps.state.outputs.draft == 'true'") ||
		!strings.Contains(upload, `releases/${RELEASE_ID}`) {
		t.Errorf("site.tar.gz is attached to the draft by its id, only while it is a draft:\n%s", upload)
	}
	for _, want := range []string{"draft: ${{ steps.state.outputs.draft }}", "site_digest: ${{ steps.pack.outputs.digest }}"} {
		if !strings.Contains(render, want) {
			t.Errorf("the render must hand the publish job %q", want)
		}
	}

	publish := codeOf(topJob(t, site, "publish"))
	for _, want := range []string{
		"needs: [resolve, render]",
		"if: ${{ needs.render.outputs.draft == 'true' }}",
		"contents: write",
		"RELEASE_ID: ${{ needs.resolve.outputs.release_id }}",
		"RELEASE_COMMIT: ${{ needs.resolve.outputs.sha }}",
		"SITE_DIGEST: ${{ needs.render.outputs.site_digest }}",
	} {
		if !strings.Contains(publish, want) {
			t.Errorf("the publish job must carry %q:\n%s", want, publish)
		}
	}
	if strings.Contains(publish, "gh release ") {
		t.Errorf("the publish job must read and publish the Release by its id, never by tag:\n%s", publish)
	}
	if !strings.Contains(publish, `gh api -X PATCH "repos/${GITHUB_REPOSITORY}/releases/${RELEASE_ID}"`) {
		t.Fatal("the publish job must publish the draft by its id")
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

// publishFakeGH is the forge for site.yml's publish step. It keeps releases
// and assets in JSON files, applies the step's own --jq filters to them, and
// answers only the id-addressed REST calls and `gh attestation verify` held to
// the release policy; a call by tag (gh release view/download/edit/upload) is
// refused. It logs every call, and every PATCH with its fields. A fixture can
// have the forge replace one asset as the Release is published, or once every
// asset has been downloaded.
const publishFakeGH = `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$FAKE_DIR/calls"
R=example/fixture
case "${1:-} ${2:-}" in
"attestation verify")
  file="$3"; shift 3
  want="--repo $R --cert-identity https://github.com/$R/.github/workflows/release.yml@refs/heads/main --source-ref refs/heads/main --source-digest $FAKE_COMMIT --deny-self-hosted-runners"
  [ "$*" = "$want" ] || { echo "fake gh: attestation verify outside the policy: $*" >&2; exit 96; }
  d="$(sha256sum "$file" | cut -d' ' -f1)"
  [ -f "$FAKE_DIR/attested/$d" ] || { echo "no attestation for ${file##*/}" >&2; exit 1; }
  exit 0 ;;
"api "*) shift ;;
*) echo "fake gh: not faked: $*" >&2; exit 99 ;;
esac
method=GET path="" expr="" accept="" paginate=0 fields=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X|--method) method="$2"; shift 2 ;;
    -H) accept="$2"; shift 2 ;;
    -F|-f) fields="$fields $2"; shift 2 ;;
    --jq) expr="$2"; shift 2 ;;
    --paginate) paginate=1; shift ;;
    -*) echo "fake gh: api flag $1" >&2; exit 97 ;;
    *) path="$1"; shift ;;
  esac
done
rel="$FAKE_DIR/releases.json"
assets="$FAKE_DIR/assets.json"
case "$method $path" in
"GET repos/$R/releases?per_page=100")
  [ "$paginate" = 1 ] || { echo "fake gh: a listing without --paginate" >&2; exit 95; }
  jq -r "$expr" "$rel" ;;
"GET repos/$R/releases/assets/"*)
  [ "$accept" = "Accept: application/octet-stream" ] || { echo "fake gh: an asset read without the octet-stream accept" >&2; exit 95; }
  cat "$FAKE_DIR/files/${path##*/}"
  printf '%s\n' "${path##*/}" >> "$FAKE_DIR/downloads"
  if [ -f "$FAKE_DIR/replaced-after-downloads" ] &&
    [ "$(wc -l < "$FAKE_DIR/downloads")" -eq "$(jq length "$assets")" ]; then
    jq --arg n "$(cat "$FAKE_DIR/replaced-after-downloads")" \
      'map(if .name == $n then .id = 9998 | .digest = "sha256:1111" else . end)' "$assets" > "$assets.next"
    mv "$assets.next" "$assets"
  fi ;;
"GET repos/$R/releases/"*"/assets?per_page=100")
  [ "$paginate" = 1 ] || { echo "fake gh: an asset listing without --paginate" >&2; exit 95; }
  id="${path#repos/$R/releases/}"; id="${id%%/*}"
  jq -r --argjson id "$id" "[.[] | select(.release_id == \$id)] | $expr" "$assets" ;;
"GET repos/$R/releases/"*)
  id="${path##*/}"
  [ "$(jq --argjson id "$id" '[.[] | select(.id == $id)] | length' "$rel")" = 1 ] || { echo "gh: Not Found (HTTP 404)" >&2; exit 1; }
  jq -r --argjson id "$id" ".[] | select(.id == \$id) | $expr" "$rel" ;;
"PATCH repos/$R/releases/"*)
  id="${path##*/}"
  printf '%s%s\n' "$id" "$fields" >> "$FAKE_DIR/patches"
  jq --argjson id "$id" 'map(if .id == $id then .draft = false else . end)' "$rel" > "$rel.next"
  mv "$rel.next" "$rel"
  if [ -f "$FAKE_DIR/replaced-on-publish" ]; then
    jq --arg n "$(cat "$FAKE_DIR/replaced-on-publish")" \
      'map(if .name == $n then .id = 9999 | .digest = "sha256:0000" else . end)' "$assets" > "$assets.next"
    mv "$assets.next" "$assets"
  fi
  [ -z "$expr" ] || jq -r --argjson id "$id" ".[] | select(.id == \$id) | $expr" "$rel" ;;
*) echo "fake gh: api $method $path is not faked" >&2; exit 99 ;;
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

type publishRelease struct {
	ID         int    `json:"id"`
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type publishAsset struct {
	ID        int    `json:"id"`
	ReleaseID int    `json:"release_id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Digest    string `json:"digest"`
}

// publishFixture is the forge state one run of the publish step reads: the
// draft 4242 of v1.2.3 with its four assets, earlier Releases, the asset
// bytes, the digests the release workflow attested, and the step's env.
type publishFixture struct {
	releases []publishRelease
	assets   []publishAsset
	files    map[int][]byte
	attested map[string]bool
	env      map[string]string
	replaced string // the asset replaced on the forge as the Release is published
	// replacedAfterDownloads is the asset replaced on the forge once the step
	// has downloaded every asset, before it publishes.
	replacedAfterDownloads string
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

const publishCommit = "c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00"

func newPublishFixture() *publishFixture {
	bin, zip, site := []byte("linux-binary-bytes"), []byte("plugin-archive-bytes"), []byte("site-archive-bytes")
	manifest := []byte(sha256Hex(bin) + "  abcd-linux-amd64\n" + sha256Hex(zip) + "  abcd-plugin-v1.2.3.zip\n")
	f := &publishFixture{
		releases: []publishRelease{
			{ID: 4242, TagName: "v1.2.3", Draft: true},
			{ID: 100, TagName: "v1.2.2"},
			{ID: 90, TagName: "v1.9.0-rc.1", Prerelease: true},
			{ID: 80, TagName: "nightly"},
		},
		files:    map[int][]byte{},
		attested: map[string]bool{sha256Hex(bin): true, sha256Hex(zip): true, sha256Hex(manifest): true},
		env: map[string]string{
			"TAG": "v1.2.3", "RELEASE_ID": "4242", "RELEASE_COMMIT": publishCommit,
			"SITE_DIGEST": sha256Hex(site), "REPO_PRIVATE": "false", "GITHUB_REPOSITORY": "example/fixture",
		},
	}
	f.attach(501, "abcd-linux-amd64", bin)
	f.attach(502, "abcd-plugin-v1.2.3.zip", zip)
	f.attach(503, "checksums.txt", manifest)
	f.attach(504, "site.tar.gz", site)
	return f
}

// attach adds an uploaded asset to the draft, its digest that of its bytes.
func (f *publishFixture) attach(id int, name string, b []byte) {
	f.assets = append(f.assets, publishAsset{ID: id, ReleaseID: 4242, Name: name, State: "uploaded", Digest: "sha256:" + sha256Hex(b)})
	f.files[id] = b
}

// asset is the draft's asset of that name.
func (f *publishFixture) asset(name string) *publishAsset {
	for i := range f.assets {
		if f.assets[i].Name == name {
			return &f.assets[i]
		}
	}
	panic("no asset " + name)
}

// run executes script against the fixture and returns its output, its error,
// every PATCH it made and every gh call it made.
func (f *publishFixture) run(t *testing.T, script string) (out string, err error, patches, calls string) {
	t.Helper()
	dir, bin := t.TempDir(), t.TempDir()
	write := func(path string, v any) {
		b, ok := v.([]byte)
		if !ok {
			var mErr error
			if b, mErr = json.Marshal(v); mErr != nil {
				t.Fatal(mErr)
			}
		}
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
			t.Fatal(mkErr)
		}
		if wErr := os.WriteFile(path, b, 0o755); wErr != nil {
			t.Fatal(wErr)
		}
	}
	write(filepath.Join(dir, "releases.json"), f.releases)
	write(filepath.Join(dir, "assets.json"), f.assets)
	for id, b := range f.files {
		write(filepath.Join(dir, "files", strings.TrimSpace(jsonInt(id))), b)
	}
	for d := range f.attested {
		write(filepath.Join(dir, "attested", d), []byte{})
	}
	if f.replaced != "" {
		write(filepath.Join(dir, "replaced-on-publish"), []byte(f.replaced))
	}
	if f.replacedAfterDownloads != "" {
		write(filepath.Join(dir, "replaced-after-downloads"), []byte(f.replacedAfterDownloads))
	}
	write(filepath.Join(bin, "gh"), []byte(publishFakeGH))
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_DIR=" + dir, "FAKE_COMMIT=" + publishCommit, "RUNNER_TEMP=" + t.TempDir()}
	for k, v := range f.env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	b, err := cmd.CombinedOutput()
	p, _ := os.ReadFile(filepath.Join(dir, "patches"))
	c, _ := os.ReadFile(filepath.Join(dir, "calls"))
	return string(b), err, string(p), string(c)
}

func jsonInt(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// TestPublishVerifiesTheDraftByIdBeforePublishingIt executes site.yml's
// publish step against a fake forge. The step reads the draft only by the id
// release.yml handed on, requires it to be the one Release of its tag,
// downloads every asset checksums.txt names and checksums.txt itself by asset
// id, checks each against the manifest, the forge's recorded digest and the
// release provenance policy, checks site.tar.gz against the digest the render
// computed, publishes by id with make_latest set only for the highest
// version, and re-reads the assets immediately before publishing and again
// afterwards.
func TestPublishVerifiesTheDraftByIdBeforePublishingIt(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	script := runBlockOf(t, topJob(t, readWorkflow(t, siteWorkflowRel), "publish"))
	cases := []struct {
		name    string
		edit    func(*publishFixture)
		latest  string // the make_latest the PATCH carries; empty when nothing is published
		fails   bool   // the step exits non-zero
		mention string // what the output must name
	}{
		{name: "a draft whose every asset verifies", latest: "true"},
		{
			name:   "a higher version already published",
			edit:   func(f *publishFixture) { f.releases = append(f.releases, publishRelease{ID: 200, TagName: "v1.10.0"}) },
			latest: "false",
		},
		{
			name: "a higher version only as a draft",
			edit: func(f *publishFixture) {
				f.releases = append(f.releases, publishRelease{ID: 200, TagName: "v1.10.0", Draft: true})
			},
			latest: "true",
		},
		{
			name: "a private repository, which has no attestations",
			edit: func(f *publishFixture) {
				f.env["REPO_PRIVATE"] = "true"
				f.attested = map[string]bool{}
			},
			latest: "true",
		},
		{
			name: "two drafts of the tag",
			edit: func(f *publishFixture) {
				f.releases = append(f.releases, publishRelease{ID: 4243, TagName: "v1.2.3", Draft: true})
			},
			fails: true, mention: "4243",
		},
		{
			name:  "a published Release of the tag beside the draft",
			edit:  func(f *publishFixture) { f.releases = append(f.releases, publishRelease{ID: 4243, TagName: "v1.2.3"}) },
			fails: true, mention: "4243",
		},
		{
			name:  "an id that names another tag's Release",
			edit:  func(f *publishFixture) { f.env["RELEASE_ID"] = "100" },
			fails: true, mention: "4242",
		},
		{
			name:  "no id handed on",
			edit:  func(f *publishFixture) { f.env["RELEASE_ID"] = "" },
			fails: true, mention: "id",
		},
		{
			name:  "a binary still uploading",
			edit:  func(f *publishFixture) { f.asset("abcd-linux-amd64").State = "open" },
			fails: true, mention: "abcd-linux-amd64",
		},
		{
			name:  "the site archive still uploading",
			edit:  func(f *publishFixture) { f.asset("site.tar.gz").State = "open" },
			fails: true, mention: "site.tar.gz",
		},
		{
			name:  "the checksums file still uploading",
			edit:  func(f *publishFixture) { f.asset("checksums.txt").State = "open" },
			fails: true, mention: "checksums.txt",
		},
		{
			name: "a binary whose bytes differ from checksums.txt",
			edit: func(f *publishFixture) {
				a := f.asset("abcd-linux-amd64")
				f.files[a.ID] = []byte("other-binary-bytes")
				a.Digest = "sha256:" + sha256Hex(f.files[a.ID])
			},
			fails: true, mention: "abcd-linux-amd64",
		},
		{
			name:  "an asset whose recorded digest differs from its bytes",
			edit:  func(f *publishFixture) { f.asset("abcd-plugin-v1.2.3.zip").Digest = "sha256:" + sha256Hex([]byte("x")) },
			fails: true, mention: "abcd-plugin-v1.2.3.zip",
		},
		{
			name:  "a site archive other than the one rendered",
			edit:  func(f *publishFixture) { f.env["SITE_DIGEST"] = sha256Hex([]byte("another-site")) },
			fails: true, mention: "site.tar.gz",
		},
		{
			name:  "no site digest handed on",
			edit:  func(f *publishFixture) { f.env["SITE_DIGEST"] = "" },
			fails: true, mention: "site.tar.gz",
		},
		{
			name:  "an asset with no attestation",
			edit:  func(f *publishFixture) { delete(f.attested, sha256Hex(f.files[f.asset("abcd-plugin-v1.2.3.zip").ID])) },
			fails: true, mention: "abcd-plugin-v1.2.3.zip",
		},
		{
			name:  "an asset the manifest does not name",
			edit:  func(f *publishFixture) { f.attach(505, "notes.txt", []byte("notes")) },
			fails: true, mention: "notes.txt",
		},
		{
			name: "a manifest naming a path",
			edit: func(f *publishFixture) {
				a := f.asset("checksums.txt")
				f.files[a.ID] = append(f.files[a.ID], []byte(sha256Hex([]byte("x"))+"  ../escape\n")...)
				a.Digest = "sha256:" + sha256Hex(f.files[a.ID])
			},
			fails: true, mention: "../escape",
		},
		{
			name:  "an asset replaced between the last download and the publish",
			edit:  func(f *publishFixture) { f.replacedAfterDownloads = "abcd-linux-amd64" },
			fails: true, mention: "changed before publishing",
		},
		{
			name:   "an asset replaced as the Release is published",
			edit:   func(f *publishFixture) { f.replaced = "abcd-linux-amd64" },
			latest: "true", fails: true, mention: "changed after they were verified",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newPublishFixture()
			if c.edit != nil {
				c.edit(f)
			}
			out, err, patches, calls := f.run(t, script)
			if (err != nil) != c.fails {
				t.Fatalf("exit err=%v, want failure=%v:\n%s", err, c.fails, out)
			}
			if c.mention != "" && !strings.Contains(out, c.mention) {
				t.Errorf("the output must name %q:\n%s", c.mention, out)
			}
			want := ""
			if c.latest != "" {
				want = "4242 draft=false make_latest=" + c.latest + "\n"
			}
			if patches != want {
				t.Errorf("PATCH calls %q, want %q:\n%s", patches, want, out)
			}
			for _, line := range strings.Split(strings.TrimSpace(calls), "\n") {
				if strings.HasPrefix(line, "release ") {
					t.Errorf("the step addressed the Release by tag: gh %s", line)
				}
			}
			if c.latest != "" && f.env["REPO_PRIVATE"] == "false" {
				for _, name := range []string{"abcd-linux-amd64", "abcd-plugin-v1.2.3.zip", "checksums.txt"} {
					if !regexp.MustCompile(`(?m)^attestation verify \S+/` + regexp.QuoteMeta(name) + ` `).MatchString(calls) {
						t.Errorf("the step must verify %s's provenance before publishing:\n%s", name, calls)
					}
				}
			}
		})
	}
}
