package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/gittest"
)

// auto-release's detect step decides whether a tagged version whose Release is
// missing is rebuilt. These tests execute that step's script, read out of the
// committed workflow and out of both template profiles, against a scratch
// repository and a fake gh. The fake answers the three reads the step makes —
// the Release, the tag's release.yml runs, one run's jobs — from canned JSON,
// applying the step's own --jq filters, and refuses a run listing that is not
// scoped to release.yml's push runs for the tag being judged.

const fakeReleaseGH = `set -euo pipefail
sub="$1 $2"; shift 2
workflow=""; event=""; branch=""; expr=""; id=""
while [ $# -gt 0 ]; do
  case "$1" in
    --workflow) workflow="$2"; shift 2 ;;
    --event) event="$2"; shift 2 ;;
    --branch) branch="$2"; shift 2 ;;
    --jq) expr="$2"; shift 2 ;;
    --limit|--json) shift 2 ;;
    -*) echo "fake gh: unexpected flag $1" >&2; exit 96 ;;
    *) id="$1"; shift ;;
  esac
done
case "$sub" in
  "release view")
    [ -f "$FAKE_DIR/released" ] && exit 0
    echo "release not found" >&2; exit 1 ;;
  "run list")
    if [ "$workflow" != release.yml ] || [ "$event" != push ] || [ "$branch" != "$FAKE_TAG" ]; then
      echo "fake gh: run list not scoped to release.yml's push runs for $FAKE_TAG (workflow=$workflow event=$event branch=$branch)" >&2; exit 97
    fi
    if [ -f "$FAKE_DIR/list-fails" ]; then echo "gh: Resource not accessible by integration (HTTP 403)" >&2; exit 1; fi
    jq -r "${expr:-.}" "$FAKE_DIR/runs.json" ;;
  "run view")
    [ -f "$FAKE_DIR/run-$id.json" ] || { echo "fake gh: no run $id" >&2; exit 95; }
    jq -r "${expr:-.}" "$FAKE_DIR/run-$id.json" ;;
  *) echo "fake gh: $sub is not faked" >&2; exit 99 ;;
esac
`

// detectCase is one state of the world the detect step reads.
type detectCase struct {
	tagged   bool              // the newest CHANGELOG version is tagged
	released bool              // its GitHub Release exists
	runs     []string          // release.yml push-run ids for the tag, newest first
	verify   map[string]string // run id -> the conclusion of that run's verify job
	listFail bool              // the run listing errors
}

// runDetect runs one workflow's detect script in a scratch repository shaped
// by c, returning its combined output, its exit code and what it wrote to
// GITHUB_OUTPUT.
func runDetect(t *testing.T, workflow string, c detectCase) (string, int, string) {
	t.Helper()
	const version, tag = "1.4.0", "v1.4.0"
	repo := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(gittest.Env(t),
			"GIT_AUTHOR_NAME=Test Person", "GIT_AUTHOR_EMAIL=person@example.com",
			"GIT_COMMITTER_NAME=Test Person", "GIT_COMMITTER_EMAIL=person@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	changelog := "# Changelog\n\n## [Unreleased]\n\n## [" + version + "] - 2026-09-25\n\n- A change.\n"
	if err := os.WriteFile(filepath.Join(repo, "CHANGELOG.md"), []byte(changelog), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "CHANGELOG.md")
	git("commit", "-q", "-m", "release "+version)
	if c.tagged {
		git("tag", "-a", tag, "-m", tag)
	}

	fake := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(fake, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if c.released {
		write("released", "")
	}
	if c.listFail {
		write("list-fails", "")
	}
	var items []string
	for _, id := range c.runs {
		items = append(items, `{"databaseId":`+id+`}`)
		write("run-"+id+".json", `{"jobs":[{"name":"verify","conclusion":"`+c.verify[id]+
			`"},{"name":"tag","conclusion":"skipped"},{"name":"release","conclusion":"failure"}]}`)
	}
	write("runs.json", "["+strings.Join(items, ",")+"]")

	bin := t.TempDir()
	fakeTool(t, bin, "gh", fakeReleaseGH)
	output := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(output, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	script := stepScript(t, findStep(t, workflow, "auto-release.yml", "need_release="))
	out, code := runScript(t, script, repo, bin, map[string]string{
		"FAKE_DIR": fake, "FAKE_TAG": tag, "GH_TOKEN": "fake", "GITHUB_OUTPUT": output,
	})
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return out, code, string(got)
}

// autoReleaseSources is the committed workflow and both template profiles'
// renderings of it, keyed by where each came from.
func autoReleaseSources(t *testing.T) map[string]string {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	sources := map[string]string{
		".github/workflows/auto-release.yml": readRepoFile(t, root, ".github/workflows/auto-release.yml"),
	}
	for name, subs := range map[string]scaffold.Substitutions{
		"auto-release.yml.tmpl (abcd)": scaffold.AbcdSubstitutions(),
		"auto-release.yml.tmpl (bare)": scaffold.BareSubstitutions("main"),
	} {
		rendered, err := scaffold.Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = string(rendered.AutoReleaseYML)
	}
	return sources
}

// TestAutoReleaseRefusesToRebuildATagItsVerifyRefused is iss-2609251125599536.
// A tag whose Release is missing is healed by rebuilding the tagged commit,
// which is right after a transient publish failure and wrong after a refused
// gate: a hand-pushed tag runs release.yml on its own push, and when that run's
// verify refused it, the tagged commit fails the same way on every later push
// to main. detect tells the two apart by the verify job of the tag's newest
// release.yml push run, and refuses loudly on a failure, naming the re-cut.
func TestAutoReleaseRefusesToRebuildATagItsVerifyRefused(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	for where, wf := range autoReleaseSources(t) {
		// The run listing reads the Actions API, which a job granted contents
		// alone cannot.
		detect := wf[strings.Index(wf, "\n  detect:\n"):strings.Index(wf, "\n  release:\n")]
		if !strings.Contains(detect, "      actions: read\n") {
			t.Errorf("%s: detect must be granted actions: read to list release.yml's runs", where)
		}

		t.Run(where+": verify refused the hand-pushed tag", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{
				tagged: true, runs: []string{"802", "801"},
				verify: map[string]string{"802": "failure", "801": "success"},
			})
			if code == 0 {
				t.Errorf("detect exited 0 for a tag whose verify refused it; it must refuse:\n%s", out)
			}
			if strings.Contains(got, "need_release=true") {
				t.Errorf("detect asked for a rebuild of a commit its gate refused:\n%s", got)
			}
			for _, want := range []string{"802", "new dated CHANGELOG version"} {
				if !strings.Contains(out, want) {
					t.Errorf("the refusal must name %q (the refused run and the re-cut):\n%s", want, out)
				}
			}
		})
		t.Run(where+": a transient publish failure after a green verify heals", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{
				tagged: true, runs: []string{"802", "801"},
				verify: map[string]string{"802": "success", "801": "failure"},
			})
			if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
				t.Errorf("detect did not heal a green-verify tag with no Release (rc=%d):\n%s\n%s", code, out, got)
			}
		})
		t.Run(where+": a tag with no push run of its own heals", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{tagged: true})
			if code != 0 || !strings.Contains(got, "need_release=true") {
				t.Errorf("detect did not heal a tag release.yml made after verify (rc=%d):\n%s\n%s", code, out, got)
			}
		})
		t.Run(where+": a failed run listing refuses loudly", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{tagged: true, listFail: true})
			if code == 0 || strings.Contains(got, "need_release=true") {
				t.Errorf("detect guessed past a failed run listing (rc=%d):\n%s\n%s", code, out, got)
			}
		})
		t.Run(where+": a released tag does nothing", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{
				tagged: true, released: true, runs: []string{"802"},
				verify: map[string]string{"802": "failure"},
			})
			if code != 0 || !strings.Contains(got, "need_release=false") {
				t.Errorf("detect acted on a tag whose Release exists (rc=%d):\n%s\n%s", code, out, got)
			}
		})
		t.Run(where+": an untagged version is tagged after verify", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{})
			if code != 0 || !strings.Contains(got, "need_tag=true") || !strings.Contains(got, "need_release=true") {
				t.Errorf("detect did not ask for a tag and a release (rc=%d):\n%s\n%s", code, out, got)
			}
		})
	}
}
