package site

// How the workflow setup writes obtains abcd's own binary before running it:
// the release is chosen from abcd's stable release list by version, never by
// the "latest" mark, and the binary's build provenance is held to release.yml
// as it stands on abcd's main branch, built from that release tag's commit on
// a GitHub-hosted runner. The download step is run with bash against a fake gh;
// nothing here reaches a network.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
)

// bothProviderWorkflows renders the workflow for each provider, since the two
// share the steps that fetch and verify abcd.
func bothProviderWorkflows(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for name, render := range map[string]func() ([]byte, error){
		"cloudflare": func() ([]byte, error) {
			return renderSiteWorkflow("main", cloudflare.Adapter{}, hosting.Site{Name: "x"}, false)
		},
		"github-pages": func() ([]byte, error) {
			return renderSiteWorkflow("main", githubpages.Adapter{}, hosting.Site{Name: "widgets", Repo: "acme/widgets"}, true)
		},
	} {
		wf, err := render()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out[name] = string(wf)
	}
	return out
}

// stepRunBlock is the run block of the step whose name is name, unindented.
func stepRunBlock(t *testing.T, wf, name string) string {
	t.Helper()
	lines := strings.Split(wf, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "- name: "+name {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("the workflow has no step named %q", name)
	}
	var script []string
	runIndent := -1
	for _, l := range lines[start+1:] {
		indent := len(l) - len(strings.TrimLeft(l, " "))
		if runIndent < 0 {
			if strings.HasPrefix(strings.TrimSpace(l), "- name:") {
				break
			}
			if strings.TrimSpace(l) == "run: |" {
				runIndent = indent
			}
			continue
		}
		if strings.TrimSpace(l) != "" && indent <= runIndent {
			break
		}
		script = append(script, strings.TrimPrefix(l, strings.Repeat(" ", runIndent+2)))
	}
	if len(script) == 0 {
		t.Fatalf("the step %q has no run block", name)
	}
	return strings.Join(script, "\n") + "\n"
}

const (
	downloadStepName = "Download abcd and its checksums"
	verifyStepName   = "Verify abcd against its release manifest and its provenance"
)

// TestTheWorkflowHoldsAbcdToReleaseYMLOnMain: the check names the whole
// certificate identity at refs/heads/main, the source ref, the release tag's
// commit and a GitHub-hosted runner, rather than the workflow path alone.
func TestTheWorkflowHoldsAbcdToReleaseYMLOnMain(t *testing.T) {
	for provider, wf := range bothProviderWorkflows(t) {
		t.Run(provider, func(t *testing.T) {
			if strings.Contains(wf, "--signer-workflow") {
				t.Error("the workflow verifies abcd with --signer-workflow, which never checks the ref the workflow ran from")
			}
			verify := stepRunBlock(t, wf, verifyStepName)
			for _, want := range []string{
				`commit="$(gh api "repos/` + abcdReleaseRepo + `/commits/${ABCD_TAG}" --jq .sha)"`,
				`[[ "$commit" =~ ^[0-9a-f]{40}$ ]]`,
				"gh attestation verify abcd-linux-amd64 --repo " + abcdReleaseRepo + " \\\n",
				"--cert-identity https://github.com/" + abcdReleaseRepo + "/.github/workflows/release.yml@refs/heads/main \\\n",
				"--source-ref refs/heads/main",
				`--source-digest "$commit"`,
				"--deny-self-hosted-runners",
			} {
				if !strings.Contains(verify, want) {
					t.Errorf("the verify step lacks %q:\n%s", want, verify)
				}
			}
			if !strings.Contains(verify, "grep ' abcd-linux-amd64$' checksums.txt | sha256sum -c -") {
				t.Errorf("the verify step dropped the checksum line:\n%s", verify)
			}
			if !strings.Contains(wf, "          ABCD_TAG: ${{ steps.abcd.outputs.tag }}\n") {
				t.Error("the verify step is not handed the downloaded release's tag through env")
			}
		})
	}
}

// TestTheWorkflowFetchesAbcdsHighestStableRelease: the download takes the
// highest vX.Y.Z of abcd's published, non-pre-release releases, by version, and
// downloads exactly that tag. The step is run against a fake gh.
func TestTheWorkflowFetchesAbcdsHighestStableRelease(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	for provider, wf := range bothProviderWorkflows(t) {
		script := stepRunBlock(t, wf, downloadStepName)
		for name, c := range map[string]struct {
			list    string
			want    string // the tag downloaded; empty means the step must refuse
			wantErr bool
		}{
			// Listed newest first, as gh lists them: a re-marked or newer
			// lower release, a pre-release-shaped tag and a malformed one are
			// all passed over, and v0.14.1 beats v0.9.0 by number.
			"the highest version, not the newest": {list: "v0.9.0\nv0.14.1\nv0.14.0\nv1.0.0-rc.1\nv2.0\nv3.0.0;id\n", want: "v0.14.1"},
			"a single release":                    {list: "v1.2.3\n", want: "v1.2.3"},
			"no stable release":                   {list: "v1.0.0-rc.1\nnightly\n", wantErr: true},
			"no release at all":                   {list: "", wantErr: true},
		} {
			t.Run(provider+"/"+name, func(t *testing.T) {
				dir := t.TempDir()
				bin := filepath.Join(dir, "bin")
				if err := os.Mkdir(bin, 0o755); err != nil {
					t.Fatal(err)
				}
				calls := filepath.Join(dir, "calls")
				fake := "#!/bin/sh\n" +
					"printf '%s\\n' \"$*\" >> \"$GH_CALLS\"\n" +
					"if [ \"$1 $2\" = 'release list' ]; then printf '%s' \"$GH_LIST\"; fi\n"
				if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(fake), 0o755); err != nil {
					t.Fatal(err)
				}
				output := filepath.Join(dir, "github_output")
				if err := os.WriteFile(output, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command("bash", "-c", script)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(),
					"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
					"GH_CALLS="+calls, "GH_LIST="+c.list,
					"GITHUB_OUTPUT="+output, "RUNNER_TEMP="+dir)
				out, err := cmd.CombinedOutput()
				raw, _ := os.ReadFile(calls)
				got := string(raw)
				if strings.Contains(got, "release view") {
					t.Errorf("the step asked gh for the latest-marked release:\n%s", got)
				}
				if !strings.Contains(got, "release list --repo "+abcdReleaseRepo+" --exclude-drafts --exclude-pre-releases") {
					t.Errorf("the step did not list abcd's stable releases:\n%s", got)
				}
				if c.wantErr {
					if err == nil {
						t.Fatalf("the step succeeded with no stable release to fetch:\n%s", out)
					}
					if strings.Contains(got, "release download") {
						t.Errorf("the step downloaded with no stable release chosen:\n%s", got)
					}
					return
				}
				if err != nil {
					t.Fatalf("the step failed: %v\n%s", err, out)
				}
				if want := "release download " + c.want + " --repo " + abcdReleaseRepo + " "; !strings.Contains(got, want) {
					t.Errorf("the step did not download exactly %s:\n%s", c.want, got)
				}
				if o, _ := os.ReadFile(output); strings.TrimSpace(string(o)) != "tag="+c.want {
					t.Errorf("the step wrote %q to GITHUB_OUTPUT, want tag=%s", o, c.want)
				}
			})
		}
	}
}
