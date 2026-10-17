package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
)

// verifyCalls returns every `gh attestation verify` invocation in a workflow,
// its backslash continuations joined onto one line.
func verifyCalls(workflow string) []string {
	lines := strings.Split(workflow, "\n")
	var calls []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "gh attestation verify ") {
			continue
		}
		call := line
		for strings.HasSuffix(call, `\`) && i+1 < len(lines) {
			i++
			call = strings.TrimSuffix(call, `\`) + " " + strings.TrimSpace(lines[i])
		}
		calls = append(calls, call)
	}
	return calls
}

// assertPinnedCalls holds every call to the exact certificate identity of
// abcd's release.yml on main, main as the source ref, a source digest, and a
// GitHub-hosted runner.
func assertPinnedCalls(t *testing.T, name string, calls []string, identity, digest string) {
	t.Helper()
	if len(calls) == 0 {
		t.Fatalf("%s verifies no attestation", name)
	}
	for _, call := range calls {
		for _, want := range []string{
			"--cert-identity " + identity,
			"--source-ref refs/heads/main",
			"--source-digest " + digest,
			"--deny-self-hosted-runners",
		} {
			if !strings.Contains(call, want) {
				t.Errorf("%s: the provenance check must carry %s:\n%s", name, want, call)
			}
		}
		if strings.Contains(call, "--signer-workflow") {
			t.Errorf("%s: the provenance check names its signer with --cert-identity, never --signer-workflow:\n%s", name, call)
		}
	}
}

// TestSiteWorkflowVerifiesTheBinaryAgainstReleaseYMLOnMain: the binary the
// production render executes is held to release.yml on main, built from the
// tagged commit `resolve` read.
func TestSiteWorkflowVerifiesTheBinaryAgainstReleaseYMLOnMain(t *testing.T) {
	own, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(siteWorkflowRel)))
	if err != nil {
		t.Fatal(err)
	}
	assertPinnedCalls(t, siteWorkflowRel, verifyCalls(string(own)),
		`"https://github.com/${GITHUB_REPOSITORY}/.github/workflows/release.yml@refs/heads/main"`, `"${RELEASE_COMMIT}"`)
	if !strings.Contains(string(own), "RELEASE_COMMIT: ${{ needs.resolve.outputs.sha }}") {
		t.Error("the render's provenance check must take its digest from the commit resolve read for the tag")
	}
}

// TestTheSetupWorkflowVerifiesAbcdAgainstReleaseYMLOnMain: a managed
// repository's site renders with abcd's released binary, held to abcd's
// release.yml on main and to the commit of the release it downloaded.
func TestTheSetupWorkflowVerifiesAbcdAgainstReleaseYMLOnMain(t *testing.T) {
	wf, err := renderSiteWorkflow("trunk", cloudflare.Adapter{})
	if err != nil {
		t.Fatal(err)
	}
	assertPinnedCalls(t, "the setup workflow", verifyCalls(string(wf)),
		"https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main", `"$commit"`)
}
