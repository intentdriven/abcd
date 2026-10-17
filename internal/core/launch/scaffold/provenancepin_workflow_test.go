package scaffold

import (
	"sort"
	"strings"
	"testing"
)

// attestationVerifyCalls returns every `gh attestation verify` invocation in
// a workflow, its backslash continuations joined onto one line.
func attestationVerifyCalls(workflow string) []string {
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

// TestEveryProvenanceCheckIsPinnedToTheReleaseWorkflowOnTheDefaultBranch:
// each rendered profile verifies an attestation only against the exact
// certificate identity of its release workflow on the default branch, the
// default branch as the source ref, the commit being released as the source
// digest, and a GitHub-hosted runner.
func TestEveryProvenanceCheckIsPinnedToTheReleaseWorkflowOnTheDefaultBranch(t *testing.T) {
	profiles := AuditProfiles()
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	checked := 0
	for _, name := range names {
		subs := profiles[name]
		rendered, err := Render(subs)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, call := range attestationVerifyCalls(string(rendered.ReleaseYML)) {
			checked++
			for _, want := range []string{
				`--repo "${GITHUB_REPOSITORY}"`,
				`--cert-identity "https://github.com/${GITHUB_REPOSITORY}/.github/workflows/` + subs.ReleaseWorkflow + `@refs/heads/` + subs.DefaultBranch + `"`,
				`--source-ref refs/heads/` + subs.DefaultBranch,
				`--source-digest "${RELEASE_COMMIT}"`,
				`--deny-self-hosted-runners`,
			} {
				if !strings.Contains(call, want) {
					t.Errorf("%s: a provenance check must carry %s:\n%s", name, want, call)
				}
			}
			if strings.Contains(call, "--signer-workflow") {
				t.Errorf("%s: a provenance check names its signer with --cert-identity, never --signer-workflow:\n%s", name, call)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no rendered profile verifies an attestation; the abcd profile verifies three")
	}
}

// TestTheReleaseCommitIsTheCommitTheRunReleases: the digest every check is
// held to is the commit the release job checked out and built.
func TestTheReleaseCommitIsTheCommitTheRunReleases(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	wf := string(rendered.ReleaseYML)
	if n := strings.Count(wf, "RELEASE_COMMIT: ${{ inputs.ref || github.sha }}"); n != 2 {
		t.Errorf("both provenance-check steps must take RELEASE_COMMIT from the checked-out release commit; found %d", n)
	}
}
