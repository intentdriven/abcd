package scaffold

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The release job creates its Release as a draft and then addresses it by
// the numeric id the forge assigned, never by its tag. A managed profile
// publishes the draft in the same step, by that id, once its assets are the
// files of dist/; abcd's own profile hands the id on to site.yml, which
// publishes it. These tests execute both steps against a fake forge.

// draftFakeGH keeps releases and assets in JSON files and applies each
// step's own --jq filter to them. `gh release create` adds the draft 4242
// with one uploaded asset per file; every other call must be an id-addressed
// REST call, and anything else is refused. Every call is logged, and every
// PATCH with its fields. Marker files in FAKE_DIR make the forge misbehave:
// second-draft (another draft of the tag appears beside the one created),
// misrecorded (the forge records a wrong digest for the named asset) and
// replaced-on-publish (the named asset is replaced as the Release publishes).
const draftFakeGH = `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$FAKE_DIR/calls"
R=example/fixture
rel="$FAKE_DIR/releases.json"
assets="$FAKE_DIR/assets.json"
case "${1:-} ${2:-}" in
"release create")
  tag="$3"; shift 3
  files=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --title|--notes) shift 2 ;;
      --draft|--verify-tag|--generate-notes) shift ;;
      -*) echo "fake gh: release create $1" >&2; exit 97 ;;
      *) files+=("$1"); shift ;;
    esac
  done
  jq --arg t "$tag" '. + [{"id": 4242, "tag_name": $t, "draft": true, "prerelease": false}]' "$rel" > "$rel.next"
  mv "$rel.next" "$rel"
  n=600
  for f in ${files[@]+"${files[@]}"}; do
    n=$((n + 1))
    d="sha256:$(sha256sum "$f" | cut -d' ' -f1)"
    if [ -f "$FAKE_DIR/misrecorded" ] && [ "$(cat "$FAKE_DIR/misrecorded")" = "${f##*/}" ]; then
      d="sha256:0000"
    fi
    jq --argjson id "$n" --arg n "${f##*/}" --arg d "$d" \
      '. + [{"id": $id, "release_id": 4242, "name": $n, "state": "uploaded", "digest": $d}]' "$assets" > "$assets.next"
    mv "$assets.next" "$assets"
  done
  if [ -f "$FAKE_DIR/second-draft" ]; then
    jq --arg t "$tag" '. + [{"id": 4243, "tag_name": $t, "draft": true, "prerelease": false}]' "$rel" > "$rel.next"
    mv "$rel.next" "$rel"
  fi
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
case "$method $path" in
"GET repos/$R/releases?per_page=100")
  [ "$paginate" = 1 ] || { echo "fake gh: a listing without --paginate" >&2; exit 95; }
  jq -r "$expr" "$rel" ;;
"GET repos/$R/releases/assets/"*)
  [ "$accept" = "Accept: application/octet-stream" ] || { echo "fake gh: an asset read without the octet-stream accept" >&2; exit 95; }
  aid="${path##*/}"
  name="$(jq -r --argjson id "$aid" '.[] | select(.id == $id) | .name' "$assets")"
  [ -n "$name" ] || { echo "gh: Not Found (HTTP 404)" >&2; exit 1; }
  cat "$FAKE_DIR/uploaded/$name" ;;
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

type draftRun struct {
	out     string
	err     error
	patches string
	calls   string
	output  string // what the step wrote to $GITHUB_OUTPUT
}

// runDraftStep executes script in a directory holding files (path -> bytes),
// against draftFakeGH seeded with releases and marker files.
func runDraftStep(t *testing.T, script, tag string, files map[string]string, releases []map[string]any, markers map[string]string) draftRun {
	t.Helper()
	dir, bin, work := t.TempDir(), t.TempDir(), t.TempDir()
	if releases == nil {
		releases = []map[string]any{}
	}
	rel, err := json.Marshal(releases)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "releases.json"), string(rel))
	mustWrite(t, filepath.Join(dir, "assets.json"), "[]")
	for name, content := range markers {
		mustWrite(t, filepath.Join(dir, name), content)
	}
	for path, content := range files {
		mustWrite(t, filepath.Join(work, filepath.FromSlash(path)), content)
		// What an upload stores on the forge, read back by an asset download.
		mustWrite(t, filepath.Join(dir, "uploaded", filepath.Base(path)), content)
	}
	mustWrite(t, filepath.Join(bin, "gh"), draftFakeGH)
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "output")
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = work
	cmd.Env = []string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TAG=" + tag, "GITHUB_REPOSITORY=example/fixture", "GH_TOKEN=fake", "FAKE_DIR=" + dir,
		"GITHUB_OUTPUT=" + output, "RELEASE_NOTES=notes",
	}
	b, err := cmd.CombinedOutput()
	p, _ := os.ReadFile(filepath.Join(dir, "patches"))
	c, _ := os.ReadFile(filepath.Join(dir, "calls"))
	o, _ := os.ReadFile(output)
	return draftRun{out: string(b), err: err, patches: string(p), calls: string(c), output: string(o)}
}

func requireJQ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
}

// stepScript is the run block of the release job's step named name.
func stepScript(t *testing.T, subs Substitutions, name string) string {
	t.Helper()
	rendered, err := Render(subs)
	if err != nil {
		t.Fatal(err)
	}
	rel := jobSection(t, string(rendered.ReleaseYML), "release")
	return runBlock(t, rel[indexOf(t, rel, "- name: "+name, "release job"):])
}

func published(id int, tag string) map[string]any {
	return map[string]any{"id": id, "tag_name": tag, "draft": false, "prerelease": false}
}

// TestAManagedReleaseIsPublishedByIdOnceItsAssetsAreDist: the managed
// profiles' release step publishes the draft it created by its id, with
// make_latest set only when no published release carries a higher version,
// and refuses a second draft of the tag, an asset the forge records with
// another digest, and an asset that changes as the Release publishes.
func TestAManagedReleaseIsPublishedByIdOnceItsAssetsAreDist(t *testing.T) {
	requireJQ(t)
	const step = "Create the GitHub Release as a draft, attach dist/, then publish it"
	dist := map[string]string{"dist/tool-linux": "linux bytes", "dist/tool-darwin": "darwin bytes"}
	cases := []struct {
		name     string
		tag      string
		files    map[string]string
		releases []map[string]any
		markers  map[string]string
		latest   string // the make_latest the PATCH carries; empty when nothing is published
		fails    bool
		mention  string
	}{
		{name: "the newest version", tag: "v1.2.3", files: dist, releases: []map[string]any{published(100, "v1.2.2")}, latest: "true"},
		{name: "no dist files", tag: "v1.2.3", latest: "true"},
		{
			name: "an older version re-released", tag: "v1.2.3", files: dist,
			releases: []map[string]any{published(100, "v1.2.2"), published(200, "v1.10.0")}, latest: "false",
		},
		{
			name: "a pre-release of a published version", tag: "v1.3.0-rc.1", files: dist,
			releases: []map[string]any{published(100, "v1.3.0")}, latest: "false",
		},
		{
			name: "a pre-release above every published version", tag: "v1.3.0-rc.1", files: dist,
			releases: []map[string]any{published(100, "v1.2.2")}, latest: "true",
		},
		{
			name: "a second draft of the tag", tag: "v1.2.3", files: dist,
			markers: map[string]string{"second-draft": ""}, fails: true, mention: "4243",
		},
		{
			name: "an asset recorded with another digest", tag: "v1.2.3", files: dist,
			markers: map[string]string{"misrecorded": "tool-linux"}, fails: true, mention: "tool-linux",
		},
		{
			name: "an asset replaced as the Release publishes", tag: "v1.2.3", files: dist,
			markers: map[string]string{"replaced-on-publish": "tool-darwin"}, latest: "true", fails: true, mention: "changed after they were checked",
		},
	}
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	for profile, subs := range map[string]Substitutions{
		"bare": BareSubstitutions("main"), "bare+semantic": semantic, "gate": GateSubstitutions("main", ""),
	} {
		script := stepScript(t, subs, step)
		for _, c := range cases {
			t.Run(profile+": "+c.name, func(t *testing.T) {
				r := runDraftStep(t, script, c.tag, c.files, c.releases, c.markers)
				if (r.err != nil) != c.fails {
					t.Fatalf("exit err=%v, want failure=%v:\n%s", r.err, c.fails, r.out)
				}
				if c.mention != "" && !strings.Contains(r.out, c.mention) {
					t.Errorf("the output must name %q:\n%s", c.mention, r.out)
				}
				want := ""
				if c.latest != "" {
					want = "4242 draft=false make_latest=" + c.latest + "\n"
				}
				if r.patches != want {
					t.Errorf("PATCH calls %q, want %q:\n%s", r.patches, want, r.out)
				}
				for _, line := range strings.Split(strings.TrimSpace(r.calls), "\n") {
					if strings.HasPrefix(line, "release ") && !strings.HasPrefix(line, "release create ") {
						t.Errorf("the step addressed the Release by tag: gh %s", line)
					}
				}
			})
		}
	}
}

// TestAbcdsDraftIsHandedOnByItsId: abcd's release step creates the draft and
// hands on its numeric id, refusing when the draft it created is not the one
// Release of the tag.
func TestAbcdsDraftIsHandedOnByItsId(t *testing.T) {
	requireJQ(t)
	script := stepScript(t, AbcdSubstitutions(), "Create the draft GitHub Release with the binaries")
	bin := map[string]string{"bin/abcd-linux-amd64": "linux", "bin/abcd-plugin-v1.2.3.zip": "zip", "bin/checksums.txt": "sums"}

	r := runDraftStep(t, script, "v1.2.3", bin, []map[string]any{published(100, "v1.2.2")}, nil)
	if r.err != nil {
		t.Fatalf("the draft must be created and its id handed on: %v\n%s", r.err, r.out)
	}
	if r.output != "id=4242\n" {
		t.Errorf("GITHUB_OUTPUT %q, want the draft's id", r.output)
	}
	if r.patches != "" {
		t.Errorf("abcd's release job must leave the Release a draft; it patched %q", r.patches)
	}

	r = runDraftStep(t, script, "v1.2.3", bin, nil, map[string]string{"second-draft": ""})
	if r.err == nil || r.output != "" {
		t.Fatalf("a second draft of the tag must refuse and hand on no id (output %q):\n%s", r.output, r.out)
	}
	if !strings.Contains(r.out, "4243") {
		t.Errorf("the refusal must list the Releases carrying the tag:\n%s", r.out)
	}
}
