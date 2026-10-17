package lint_test

import (
	"fmt"
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
    [ -f "$FAKE_DIR/release.json" ] && { jq -r "${expr:-.}" "$FAKE_DIR/release.json"; exit 0; }
    echo "release not found" >&2; exit 1 ;;
  "run list")
    if [ "$workflow" != release.yml ] || [ "$event" != push ] || [ "$branch" != "$FAKE_TAG" ]; then
      echo "fake gh: run list not scoped to release.yml's push runs for $FAKE_TAG (workflow=$workflow event=$event branch=$branch)" >&2; exit 97
    fi
    if [ -f "$FAKE_DIR/list-fails" ]; then echo "gh: Resource not accessible by integration (HTTP 403)" >&2; exit 1; fi
    jq -r "${expr:-.}" "$FAKE_DIR/runs.json" ;;
  "run view")
    if [ -f "$FAKE_DIR/view-fails" ]; then echo "gh: HTTP 502: Server Error" >&2; exit 1; fi
    [ -f "$FAKE_DIR/run-$id.json" ] || { echo "fake gh: no run $id" >&2; exit 95; }
    jq -r "${expr:-.}" "$FAKE_DIR/run-$id.json" ;;
  *) echo "fake gh: $sub is not faked" >&2; exit 99 ;;
esac
`

// detectCase is one state of the world the detect step reads.
type detectCase struct {
	tagged   bool              // the newest CHANGELOG version is tagged
	released bool              // its GitHub Release exists and is published
	draft    bool              // its GitHub Release exists as an unpublished draft
	runs     []string          // release.yml push-run ids for the tag, newest first
	verify   map[string]string // run id -> the conclusion of that run's verify job ("in_progress": still running, no conclusion)
	listFail bool              // the run listing errors
	viewFail bool              // reading a run's jobs errors
	moved    bool              // the default branch has a commit past the tag, and the push is that commit
	private  bool              // the repository is private, so its releases carry no attestation
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
	if c.moved {
		if err := os.WriteFile(filepath.Join(repo, "later.txt"), []byte("later\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", "later.txt")
		git("commit", "-q", "-m", "a later change")
	}
	head := exec.Command("git", "rev-parse", "HEAD")
	head.Dir = repo
	head.Env = gittest.Env(t)
	pushed, err := head.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}

	fake := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(fake, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if c.released {
		write("release.json", `{"isDraft":false}`)
	}
	if c.draft {
		write("release.json", `{"isDraft":true}`)
	}
	if c.listFail {
		write("list-fails", "")
	}
	if c.viewFail {
		write("view-fails", "")
	}
	var items []string
	for _, id := range c.runs {
		items = append(items, `{"databaseId":`+id+`}`)
		// The API reports a job that has not finished with a null conclusion.
		verify := `"status":"completed","conclusion":"` + c.verify[id] + `"`
		if c.verify[id] == "in_progress" {
			verify = `"status":"in_progress","conclusion":null`
		}
		write("run-"+id+".json", `{"jobs":[{"name":"verify",`+verify+
			`},{"name":"tag","conclusion":"skipped"},{"name":"release","conclusion":"failure"}]}`)
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
		"GITHUB_SHA": strings.TrimSpace(string(pushed)), "REPO_PRIVATE": fmt.Sprint(c.private),
	})
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return out, code, string(got)
}

// autoReleaseSource is one rendering of auto-release.yml. attested reports
// that the release workflow it calls verifies build provenance on a public
// repository (abcd's own profile, or one with a semantic gate configured).
// pushVerifies reports that a run of that workflow started by a pushed
// version tag runs verify: it has a tag-push trigger and does not refuse such
// a run (the bare profile; not the gate workflow, which has no such trigger).
type autoReleaseSource struct {
	wf           string
	attested     bool
	pushVerifies bool
}

// autoReleaseSources is the committed workflow and the template profiles'
// renderings of it, keyed by where each came from.
func autoReleaseSources(t *testing.T) map[string]autoReleaseSource {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	sources := map[string]autoReleaseSource{
		".github/workflows/auto-release.yml": {wf: readRepoFile(t, root, ".github/workflows/auto-release.yml"), attested: true},
	}
	semantic := scaffold.BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	for name, subs := range map[string]scaffold.Substitutions{
		"auto-release.yml.tmpl (abcd)":          scaffold.AbcdSubstitutions(),
		"auto-release.yml.tmpl (bare)":          scaffold.BareSubstitutions("main"),
		"auto-release.yml.tmpl (bare+semantic)": semantic,
		"auto-release.yml.tmpl (gate)":          scaffold.GateSubstitutions("main", ""),
	} {
		rendered, err := scaffold.Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		attested := subs.Abcd || len(subs.SemanticGates) > 0
		sources[name] = autoReleaseSource{wf: string(rendered.AutoReleaseYML), attested: attested,
			pushVerifies: !attested && !subs.Gate}
	}
	return sources
}

// TestAutoReleaseRefusesToRebuildATagItsVerifyRefused is iss-2609251125599536.
// A tag whose Release is missing is healed by rebuilding the tagged commit,
// which is right after a transient publish failure and wrong after a refused
// gate: in a profile whose release workflow verifies a pushed tag, a
// hand-pushed tag runs release.yml on its own push, and when that run's verify
// refused it, the tagged commit fails the same way on every later push to
// main. detect tells the two apart by the verify job of the tag's newest
// release.yml push run, and refuses loudly on a failure, naming the re-cut.
func TestAutoReleaseRefusesToRebuildATagItsVerifyRefused(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	for where, src := range autoReleaseSources(t) {
		if !src.pushVerifies {
			continue // no pushed tag's run carries a verify there; see the test below
		}
		wf := src.wf
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
		// iss-2609251616310535: the two remaining reads of the step. A run
		// whose jobs cannot be read is refused like a failed listing, and a
		// verify still running has not refused the tag, so it heals.
		t.Run(where+": a failed read of the run's jobs refuses loudly", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{
				tagged: true, runs: []string{"802"},
				verify: map[string]string{"802": "success"}, viewFail: true,
			})
			if code == 0 || strings.Contains(got, "need_release=true") {
				t.Errorf("detect guessed past a failed read of run 802's jobs (rc=%d):\n%s\n%s", code, out, got)
			}
			if !strings.Contains(out, "802") {
				t.Errorf("the refusal must name the run whose jobs it could not read:\n%s", out)
			}
		})
		t.Run(where+": a verify still in progress heals", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{
				tagged: true, runs: []string{"802"},
				verify: map[string]string{"802": "in_progress"},
			})
			if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
				t.Errorf("detect refused a tag whose verify has not concluded (rc=%d):\n%s\n%s", code, out, got)
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
		// A release is created as a draft and published once every asset is
		// attached, so a run that stopped in between leaves a draft. Only a
		// published release counts as released; a draft is re-released.
		t.Run(where+": a tag whose Release is still a draft is re-released", func(t *testing.T) {
			out, code, got := runDetect(t, wf, detectCase{tagged: true, draft: true})
			if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
				t.Errorf("detect counted a draft as released (rc=%d):\n%s\n%s", code, out, got)
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

// TestAutoReleaseReadsNoPushRunsWhereAPushedTagRunsNoVerify holds detect in a
// profile whose release workflow runs no verify for a pushed version tag: it
// refuses such a run before verify (abcd's own profile and one with semantic
// gates), or has no tag-push trigger at all (the gate workflow). No push run
// of a tag carries a verify conclusion there, so detect lists no runs, and its
// job is granted no actions scope: a tag with no published Release takes the
// re-release path whatever the Actions API would answer.
func TestAutoReleaseReadsNoPushRunsWhereAPushedTagRunsNoVerify(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	for where, src := range autoReleaseSources(t) {
		if src.pushVerifies {
			continue
		}
		wf := src.wf
		detect := wf[strings.Index(wf, "\n  detect:\n"):strings.Index(wf, "\n  release:\n")]
		for _, absent := range []string{"actions: read", "gh run list", "gh run view"} {
			if strings.Contains(detect, absent) {
				t.Errorf("%s: detect must not carry %q", where, absent)
			}
		}
		for name, c := range map[string]detectCase{
			"a failed run listing":           {tagged: true, listFail: true},
			"a failed read of a run's jobs":  {tagged: true, runs: []string{"802"}, verify: map[string]string{"802": "failure"}, viewFail: true},
			"a push run whose verify failed": {tagged: true, runs: []string{"802"}, verify: map[string]string{"802": "failure"}},
		} {
			t.Run(where+": "+name, func(t *testing.T) {
				out, code, got := runDetect(t, wf, c)
				if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
					t.Errorf("detect did not re-release a tag with no Release (rc=%d):\n%s\n%s", code, out, got)
				}
			})
		}
	}
}

// TestAutoReleaseFinishesAnUnpublishedReleaseOnlyFromItsOwnCommit holds the
// re-release path to the commit a run is started for. A release workflow that
// verifies build provenance holds it to the release workflow on the default
// branch and the commit being released, and a run started by a push signs
// with that push's commit. So a missing or draft Release at a tag behind the
// pushed commit is not re-released from the push: detect fails, names the
// remedy (re-run the failed jobs of the release run of the tagged commit) and
// asks for no release. At the pushed commit itself, on a private repository
// (no attestation) and in a profile that verifies no provenance, the
// re-release path stands.
func TestAutoReleaseFinishesAnUnpublishedReleaseOnlyFromItsOwnCommit(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply the step's --jq filters")
	}
	for where, src := range autoReleaseSources(t) {
		for name, c := range map[string]detectCase{
			"a draft Release":   {tagged: true, draft: true, moved: true},
			"a missing Release": {tagged: true, moved: true},
		} {
			t.Run(where+": "+name+" behind the pushed commit", func(t *testing.T) {
				out, code, got := runDetect(t, src.wf, c)
				if !src.attested {
					if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
						t.Errorf("a profile that verifies no provenance must still re-release from the tag (rc=%d):\n%s\n%s", code, out, got)
					}
					return
				}
				if code == 0 {
					t.Errorf("detect exited 0; it must fail and name the remedy:\n%s", out)
				}
				if strings.Contains(got, "need_release=true") {
					t.Errorf("detect asked for a release of a tag behind the pushed commit:\n%s", got)
				}
				for _, want := range []string{"::error::", "re-run the failed jobs",
					"If no such run exists, or if the re-run fails again, roll a new dated CHANGELOG version"} {
					if !strings.Contains(out, want) {
						t.Errorf("the failure must carry %q:\n%s", want, out)
					}
				}
			})
			t.Run(where+": "+name+" behind the pushed commit, private repository", func(t *testing.T) {
				c := c
				c.private = true
				out, code, got := runDetect(t, src.wf, c)
				if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
					t.Errorf("a private repository's releases carry no attestation, so the tag re-releases (rc=%d):\n%s\n%s", code, out, got)
				}
			})
			t.Run(where+": "+name+" at the pushed commit", func(t *testing.T) {
				c := c
				c.moved = false
				out, code, got := runDetect(t, src.wf, c)
				if code != 0 || !strings.Contains(got, "need_release=true") || !strings.Contains(got, "release_ref=") {
					t.Errorf("the tag is the pushed commit, so it re-releases (rc=%d):\n%s\n%s", code, out, got)
				}
			})
		}
	}
}
