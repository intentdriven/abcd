package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// iss-2610120617521209, the product thinker's ruling of 2026-10-12: the
// bootstrap checked its download only against checksums.txt from the same
// release page, which whoever can change that page can change too. It now
// verifies the download's build provenance with `gh attestation verify` when
// gh is present, refusing on a failed verification, and otherwise installs on
// the checksum check alone with a warning that says the download was not
// checked against the signed build record and how to check it by hand. abcd
// does not require gh: a gh that is installed but not signed in counts as not
// present, and the warning says why.
//
// gh is a stub on PATH in every bootstrap test (ghStubDir), so no run here asks
// the real gh, which on a developer's machine is signed in and would verify
// fixture bytes against GitHub.

// ghVerifyArgs is the policy every gh call must carry: this repository's
// release workflow on main, built from main on a GitHub-hosted runner, on
// github.com.
const ghVerifyArgs = "--hostname github.com --repo intentdriven/abcd --cert-identity https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main --source-ref refs/heads/main --deny-self-hosted-runners"

func TestBootstrapInstallsADownloadGhVerifies(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghVerified)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code != 0 {
		t.Fatalf("a download gh verifies installs: exit %d, output %q", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(body) {
		t.Fatalf("the verified download must be installed; got %q (%v)", got, err)
	}
	calls := gh.verifyCalls(t)
	if len(calls) != 1 {
		t.Fatalf("gh attestation verify must run once, got %d calls: %q", len(calls), calls)
	}
	if !strings.Contains(calls[0].args, ghVerifyArgs) {
		t.Errorf("gh must be held to the release workflow's policy %q; ran %q", ghVerifyArgs, calls[0].args)
	}
	// The workflow identity says who built the bytes, not which release: an
	// older release's genuine build passes it under any tag. So gh is held
	// to the commit the release tag names, as the API resolved it.
	if !strings.Contains(calls[0].args, "--source-digest "+bootstrapRelease) {
		t.Errorf("gh must be bound to the release's commit %s; ran %q", bootstrapRelease, calls[0].args)
	}
	if calls[0].sha != sha256Hex(body) {
		t.Errorf("gh must verify the downloaded binary; it was handed bytes hashing to %s", calls[0].sha)
	}
	if !strings.Contains(out, "build provenance was verified") {
		t.Errorf("the notice must say the provenance was verified; output %q", out)
	}
	if strings.Contains(out, "WARNING") {
		t.Errorf("a verified install carries no warning; output %q", out)
	}
}

func TestBootstrapRefusesADownloadGhCannotVerify(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghFails)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code == 0 {
		t.Fatalf("a failed verification must refuse; output %q", out)
	}
	assertNothingInstalled(t, root, out)
	for _, want := range []string{"build provenance", "gh attestation verify", "no attestations found for subject"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal must say %q; output %q", want, out)
		}
	}
}

// TestBootstrapCachesNothingGhCannotVerify: in cache mode a refused download
// reaches neither the cache nor the home-scoped attestation, so `ahoy install`
// has nothing to promote.
func TestBootstrapCachesNothingGhCannotVerify(t *testing.T) {
	root := bootstrapRoot(t)
	data := t.TempDir()
	home := t.TempDir()
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghFails)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root,
		append(fx.env(), "CLAUDE_PLUGIN_DATA="+data, "HOME="+home), "", gh.dir)
	if code == 0 {
		t.Fatalf("a failed verification must refuse; output %q", out)
	}
	if _, err := os.Stat(filepath.Join(data, "cache", bootstrapAsset())); !os.IsNotExist(err) {
		t.Errorf("an unverified download must not be cached: %v", err)
	}
	if _, err := os.Stat(abcdhome.Path(home, "cache-attestation")); !os.IsNotExist(err) {
		t.Errorf("an unverified download must not be attested: %v", err)
	}
}

func TestBootstrapWarnsWithoutGh(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", "")
	if code != 0 {
		t.Fatalf("abcd does not require gh: without it the checksum-verified download installs; exit %d, output %q", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(body) {
		t.Fatalf("the checksum-verified download must be installed; got %q (%v)", got, err)
	}
	assertProvenanceWarning(t, out, "gh is not installed")
}

func TestBootstrapWarnsWhenGhIsNotSignedIn(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghUnauthenticated)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code != 0 {
		t.Fatalf("a gh that is not signed in counts as no gh, not as a failed verification; exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); err != nil {
		t.Fatalf("the checksum-verified download must be installed: %v", err)
	}
	assertProvenanceWarning(t, out, "not signed in")
}

// assertProvenanceWarning: the success line says the download was checked
// against the checksum alone, why, and the command that checks it by hand.
func assertProvenanceWarning(t *testing.T, out, why string) {
	t.Helper()
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"WARNING", "not checked against the signed build record", why, "gh attestation verify", "--repo intentdriven/abcd --cert-identity https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main --source-ref refs/heads/main --deny-self-hosted-runners"} {
		if !strings.Contains(flat, want) {
			t.Errorf("the warning must say %q; output %q", want, out)
		}
	}
	if lines := bootstrapTerminalLines(out); len(lines) != 1 || !strings.Contains(lines[0], "WARNING") {
		t.Errorf("the warning must ride on the one terminal line a hook's reader sees; lines %q", lines)
	}
}

// TestBootstrapRefusesWhenTheReleaseCommitCannotBeResolved: with a gh that
// can verify, a release whose commit the API does not answer cannot have its
// provenance bound to it, so nothing is installed and gh is not asked a
// question that would pass for any release's build.
func TestBootstrapRefusesWhenTheReleaseCommitCannotBeResolved(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	*fx.failCommits = 1
	gh := newGhStub(t, ghVerified)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code == 0 {
		t.Fatalf("an unresolvable release commit must refuse when gh can verify; output %q", out)
	}
	assertNothingInstalled(t, root, out)
	if calls := gh.verifyCalls(t); len(calls) != 0 {
		t.Errorf("gh must not verify without the release's commit: %q", calls)
	}
	if !strings.Contains(out, "commit") || !strings.Contains(out, bootstrapTag) {
		t.Errorf("the refusal must say the release's commit could not be resolved; output %q", out)
	}
}

// TestBootstrapWarnsWhenGhCannotBindTheCommit: a gh whose attestation verify
// has no --source-digest cannot hold the check to the release's commit, so it
// counts as a gh that cannot verify: the checksum stands alone, with the
// warning.
func TestBootstrapWarnsWhenGhCannotBindTheCommit(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghNoSourceDigest)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code != 0 {
		t.Fatalf("a gh too old to bind the commit is no gh, not a failed verification; exit %d, output %q", code, out)
	}
	if calls := gh.verifyCalls(t); len(calls) != 0 {
		t.Errorf("a gh that cannot bind the commit must not be asked to verify: %q", calls)
	}
	assertProvenanceWarning(t, out, "--source-digest")
}

// TestBootstrapWarnsWhenGhCannotPinTheSourceRef: a gh whose attestation
// verify has --source-digest but no --source-ref cannot hold the check to the
// ref the release was built from, so it counts as a gh that cannot verify.
func TestBootstrapWarnsWhenGhCannotPinTheSourceRef(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	gh := newGhStub(t, ghNoSourceRef)

	out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
	if code != 0 {
		t.Fatalf("a gh that cannot pin the source ref is no gh, not a failed verification; exit %d, output %q", code, out)
	}
	if calls := gh.verifyCalls(t); len(calls) != 0 {
		t.Errorf("a gh that cannot pin the source ref must not be asked to verify: %q", calls)
	}
	assertProvenanceWarning(t, out, "--source-ref")
}

// TestBootstrapWarnsWhenGhLacksAPolicyFlag: a gh whose attestation verify has
// no --cert-identity or no --deny-self-hosted-runners cannot apply the
// release's verification policy, so it counts as a gh that cannot verify
// rather than failing part-way through the check.
func TestBootstrapWarnsWhenGhLacksAPolicyFlag(t *testing.T) {
	for mode, flag := range map[ghMode]string{ghNoCertIdentity: "--cert-identity", ghNoDenySelfHosted: "--deny-self-hosted-runners"} {
		t.Run(flag, func(t *testing.T) {
			root := bootstrapRoot(t)
			body := []byte("#!/bin/sh\nexit 0\n")
			fx := bootstrapServer(t, body, bootstrapManifest(body))
			gh := newGhStub(t, mode)

			out, code := runScriptGh(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), "", gh.dir)
			if code != 0 {
				t.Fatalf("a gh without %s is no gh, not a failed verification; exit %d, output %q", flag, code, out)
			}
			if calls := gh.verifyCalls(t); len(calls) != 0 {
				t.Errorf("a gh without %s must not be asked to verify: %q", flag, calls)
			}
			assertProvenanceWarning(t, out, flag)
		})
	}
}
