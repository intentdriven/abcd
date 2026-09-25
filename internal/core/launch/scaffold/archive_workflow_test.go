package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// The release workflow's half of the pinned plugin archive
// (adr-2609231048308186; the 2026-09-23 rulings E1 and E3). The ship pins the
// release archive's address and digest in the committed catalog; release.yml
// must re-render the archive from the tagged commit, refuse unless the digest
// matches, and publish exactly that archive — checksummed, attested and
// uploaded with the binaries — with release notes that state the harness floor.
//
// It reads the COMMITTED workflow; TestSelfScaffoldParity already binds that to
// the template, so one assertion covers both.

// jobSection slices one job's body out of the workflow: from its key to the
// next top-level job key.
func jobSection(t *testing.T, wf, job string) string {
	t.Helper()
	start := strings.Index(wf, "\n  "+job+":\n")
	if start < 0 {
		t.Fatalf("the workflow has no %q job", job)
	}
	rest := wf[start+1:]
	lines := strings.Split(rest, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && strings.HasSuffix(line, ":") {
			break
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// indexOf fails the test when needle is absent, so an ordering assertion can
// never pass on a missing step.
func indexOf(t *testing.T, hay, needle, where string) int {
	t.Helper()
	i := strings.Index(hay, needle)
	if i < 0 {
		t.Fatalf("%s carries no %q", where, needle)
	}
	return i
}

func TestReleaseWorkflowPublishesThePinnedArchive(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(ReleaseYMLPath)))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(data)

	// verify: the pin is proved before anything is built, on a real release
	// only, bound to the tag being released.
	verify := jobSection(t, wf, "verify")
	gate := indexOf(t, verify, `go run ./cmd/abcd launch archive --out "$out" --tag "${TAG}" --verify`+repositoryBinding, "verify")
	stepStart := strings.LastIndex(verify[:gate], "- name:")
	if !strings.Contains(verify[stepStart:gate], "if: github.event_name != 'workflow_dispatch'") {
		t.Error("the verify pin gate must be skipped on the rehearsal path and only there")
	}

	// release: the archive is rendered and re-verified from the same commit as
	// the binaries, AFTER the VCS-stamp check and BEFORE it is checksummed,
	// attested and uploaded.
	rel := jobSection(t, wf, "release")
	stamp := indexOf(t, rel, "./scripts/check-vcs-stamp.sh", "release")
	// The same call binds the pinned address to this repository's release, so
	// it is asserted before the Release is created: the URL derives from
	// plugin.json's repository, which a rename, transfer or fork leaves pointing
	// elsewhere, and a pin at another repository's release passes --verify yet
	// 404s for every install.
	render := indexOf(t, rel, `go run ./cmd/abcd launch archive --out bin --tag "${TAG}" --verify`+repositoryBinding, "release")
	sums := indexOf(t, rel, "sha256sum abcd-* > checksums.txt", "release")
	attest := indexOf(t, rel, "- name: Attest build provenance for the binaries", "release")
	create := indexOf(t, rel, `gh release create "${TAG}" bin/abcd-* bin/checksums.txt`, "release")
	if !(stamp < render && render < sums && sums < attest && attest < create) {
		t.Errorf("release job order: vcs-stamp %d < archive %d < checksums %d < attest %d < create %d must hold",
			stamp, render, sums, attest, create)
	}

	// The release notes state the harness floor the archive source needs (E3).
	if !strings.Contains(rel, `--notes "${RELEASE_NOTES}"`) || !strings.Contains(rel, "v2.1.224") {
		t.Error("the release notes must state the v2.1.224 harness floor for the archive-sourced plugin")
	}

	// Post-release: the published archive is downloaded fresh, attested, and
	// byte-compared with the one verified against the pin.
	for _, needle := range []string{
		`--pattern "abcd-plugin-${TAG}.zip"`,
		`gh attestation verify "$DL/abcd-plugin-${TAG}.zip"`,
		`cmp "$DL/abcd-plugin-${TAG}.zip" "bin/abcd-plugin-${TAG}.zip"`,
	} {
		if !strings.Contains(rel, needle) {
			t.Errorf("the post-release check must carry %q", needle)
		}
	}
}

// repositoryBinding is the operand that binds the archive's download address
// to the repository the workflow runs in. The binary makes the check, so no
// workflow carries a copy of it in shell.
const repositoryBinding = ` --repository "${GITHUB_REPOSITORY}"`

// TestBareReleaseWorkflowHasNoArchive keeps the archive abcd-only: a managed
// repo's scaffolded workflow has no abcd binary to render with.
func TestBareReleaseWorkflowHasNoArchive(t *testing.T) {
	rendered, err := Render(BareSubstitutions("trunk"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered.ReleaseYML), "launch archive") {
		t.Error("the bare release.yml must not render the plugin archive")
	}
	if strings.Contains(string(rendered.AutoReleaseYML), "launch archive") {
		t.Error("the bare auto-release.yml must not prove a plugin archive pin")
	}
}

// TestNoWorkflowCarriesTheRepositoryCheckInShell keeps the repository binding
// in one place: `launch archive --repository` makes it, and a shell copy in a
// workflow is a second definition that drifts from the first.
func TestNoWorkflowCarriesTheRepositoryCheckInShell(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	for name, wf := range map[string][]byte{"release.yml": rendered.ReleaseYML, "auto-release.yml": rendered.AutoReleaseYML} {
		if strings.Contains(string(wf), "/releases/download/${TAG}/") {
			t.Errorf("%s carries the repository check in shell; call launch archive --repository instead", name)
		}
		if n, m := strings.Count(string(wf), "launch archive "), strings.Count(string(wf), repositoryBinding); n != m {
			t.Errorf("%s: %d launch archive call(s), %d bound to the repository", name, n, m)
		}
	}
}

// TestAbcdDeclaresTheArchiveItsWorkflowsPublish is the lockstep between the
// ship's pin and the workflows that make it resolve. The ship pins the catalog
// only on the version-location contract's "publishes_plugin_archive": true, and
// only the Abcd rendering's workflows upload the archive: abcd declares it, and
// the bare rendering — what a managed repository scaffolds — uploads none, so a
// managed repository that copied the declaration would pin an asset nothing
// publishes.
func TestAbcdDeclaresTheArchiveItsWorkflowsPublish(t *testing.T) {
	declared, err := launch.DeclaresPluginArchive(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	publishes := strings.Contains(string(rendered.ReleaseYML), "launch archive --out bin")
	if !declared || !publishes {
		t.Errorf("abcd's version-location contract declares publishes_plugin_archive=%v and its release.yml uploads the archive=%v; both must hold",
			declared, publishes)
	}
}
