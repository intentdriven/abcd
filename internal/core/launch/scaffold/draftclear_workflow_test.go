package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The release job refuses a tag whose Release is already published and clears
// an unfinished draft of the tag before it creates the Release afresh. These
// tests hold that step to two rules, in the committed workflow and every
// template profile that publishes:
//
//   - it runs straight after the tag shape-check, before anything is signed,
//     built or created, so a refused tag mints no attestation;
//   - it selects each draft by the id the release listing reports for it, and
//     deletes only that id, never a Release looked up by tag name.

const draftClearStep = "- name: Refuse a tag already released; clear an unfinished draft of it"

// releaseJobSources is the committed release workflow and each publishing
// template profile's rendering of it, keyed by where each came from.
func releaseJobSources(t *testing.T) map[string]string {
	t.Helper()
	committed, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(ReleaseYMLPath)))
	if err != nil {
		t.Fatalf("read committed %s: %v", ReleaseYMLPath, err)
	}
	sources := map[string]string{ReleaseYMLPath: jobSection(t, string(committed), "release")}
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	for name, subs := range map[string]Substitutions{
		"release.yml.tmpl (abcd)":          AbcdSubstitutions(),
		"release.yml.tmpl (bare)":          BareSubstitutions("main"),
		"release.yml.tmpl (bare+semantic)": semantic,
		"release.yml.tmpl (gate)":          GateSubstitutions("main", ""),
	} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = jobSection(t, string(rendered.ReleaseYML), "release")
	}
	return sources
}

func TestTheDraftClearRunsBeforeAnythingIsSigned(t *testing.T) {
	for where, rel := range releaseJobSources(t) {
		shape := indexOf(t, rel, "- name: Refuse a tag that is not a vX.Y.Z release tag", where)
		clear := indexOf(t, rel, draftClearStep, where)
		if clear < shape {
			t.Errorf("%s: the draft clear runs before the tag shape-check", where)
		}
		// Every step between the two is the shape-check itself.
		if between := rel[shape+1 : clear]; strings.Contains(between, "\n      - name:") {
			t.Errorf("%s: a step runs between the tag shape-check and the draft clear", where)
		}
		for _, later := range []string{"actions/attest@", "Semantic-gate", "make build", "launch archive", "gh release create", "actions/download-artifact@"} {
			if at := strings.Index(rel, later); at >= 0 && at < clear {
				t.Errorf("%s: %q precedes the draft clear", where, later)
			}
		}
	}
}

// fakeReleasesGH answers the release listing from releases.json, applying the
// step's own --jq filter, and records each delete by id. It refuses a delete
// of a published Release and any call it does not model.
const fakeReleasesGH = `#!/usr/bin/env bash
set -euo pipefail
[ "${1:-}" = api ] || { echo "fake gh: $*" >&2; exit 97; }
shift
method=GET path="" expr="" paginate=0
while [ $# -gt 0 ]; do
  case "$1" in
    -X|--method) method="$2"; shift 2 ;;
    --paginate) paginate=1; shift ;;
    --jq) expr="$2"; shift 2 ;;
    -*) echo "fake gh: unexpected flag $1" >&2; exit 96 ;;
    *) path="$1"; shift ;;
  esac
done
case "$method $path" in
  "GET repos/example/fixture/releases?per_page=100")
    [ "$paginate" = 1 ] || { echo "fake gh: release listing without --paginate" >&2; exit 95; }
    [ ! -f "$FAKE_DIR/list-fails" ] || { echo "gh: HTTP 502" >&2; exit 1; }
    jq -r "${expr:-.}" "$FAKE_DIR/releases.json" ;;
  "DELETE repos/example/fixture/releases/"*)
    id="${path##*/}"
    draft="$(jq -r --argjson id "$id" '[.[] | select(.id == $id) | .draft] | first // "absent"' "$FAKE_DIR/releases.json")"
    [ "$draft" = true ] || { echo "fake gh: delete of release $id (draft=$draft)" >&2; exit 98; }
    jq --argjson id "$id" '[.[] | select(.id != $id)]' "$FAKE_DIR/releases.json" > "$FAKE_DIR/releases.next"
    mv "$FAKE_DIR/releases.next" "$FAKE_DIR/releases.json"
    echo "$id" >> "$FAKE_DIR/deleted" ;;
  *) echo "fake gh: $method $path" >&2; exit 99 ;;
esac
`

func TestTheDraftClearDeletesOnlyItsOwnDraftsById(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filter")
	}
	cases := []struct {
		name     string
		releases string
		listFail bool
		ok       bool
		deleted  string
	}{
		{name: "no Release", releases: `[]`, ok: true},
		{
			name:     "a draft of the tag beside other tags' Releases",
			releases: `[{"id":11,"draft":true,"tag_name":"v1.2.3"},{"id":12,"draft":true,"tag_name":"v1.2.4"},{"id":13,"draft":false,"tag_name":"v1.2.2"}]`,
			ok:       true, deleted: "11\n",
		},
		{
			name:     "two drafts of the tag",
			releases: `[{"id":21,"draft":true,"tag_name":"v1.2.3"},{"id":22,"draft":true,"tag_name":"v1.2.3"}]`,
			ok:       true, deleted: "21\n22\n",
		},
		{
			name:     "a published Release of the tag beside a draft of it",
			releases: `[{"id":31,"draft":true,"tag_name":"v1.2.3"},{"id":32,"draft":false,"tag_name":"v1.2.3"}]`,
			ok:       false,
		},
		{name: "a release listing that fails", releases: `[]`, listFail: true, ok: false},
	}
	for where, rel := range releaseJobSources(t) {
		script := runBlock(t, rel[indexOf(t, rel, draftClearStep, where):])
		if strings.Contains(script, "gh release delete") {
			t.Errorf("%s: the draft clear deletes a Release by tag name", where)
		}
		for _, c := range cases {
			t.Run(where+": "+c.name, func(t *testing.T) {
				fake, bin := t.TempDir(), t.TempDir()
				mustWrite(t, filepath.Join(fake, "releases.json"), c.releases)
				if c.listFail {
					mustWrite(t, filepath.Join(fake, "list-fails"), "")
				}
				mustWrite(t, filepath.Join(bin, "gh"), fakeReleasesGH)
				if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command("bash", "-c", script)
				cmd.Env = []string{
					"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
					"TAG=v1.2.3", "GITHUB_REPOSITORY=example/fixture", "GH_TOKEN=fake", "FAKE_DIR=" + fake,
				}
				out, err := cmd.CombinedOutput()
				if got := err == nil; got != c.ok {
					t.Errorf("exit ok=%v, want %v\n%s", got, c.ok, out)
				}
				deleted, _ := os.ReadFile(filepath.Join(fake, "deleted"))
				if string(deleted) != c.deleted {
					t.Errorf("deleted ids %q, want %q\n%s", deleted, c.deleted, out)
				}
			})
		}
	}
}
