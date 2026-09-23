package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	gate := indexOf(t, verify, `go run ./cmd/abcd launch archive --out "$out" --tag "${TAG}" --verify`, "verify")
	stepStart := strings.LastIndex(verify[:gate], "- name:")
	if !strings.Contains(verify[stepStart:gate], "if: github.event_name != 'workflow_dispatch'") {
		t.Error("the verify pin gate must be skipped on the rehearsal path and only there")
	}

	// release: the archive is rendered and re-verified from the same commit as
	// the binaries, AFTER the VCS-stamp check and BEFORE it is checksummed,
	// attested and uploaded.
	rel := jobSection(t, wf, "release")
	stamp := indexOf(t, rel, "./scripts/check-vcs-stamp.sh", "release")
	render := indexOf(t, rel, `go run ./cmd/abcd launch archive --out bin --tag "${TAG}" --verify`, "release")
	sums := indexOf(t, rel, "sha256sum abcd-* > checksums.txt", "release")
	attest := indexOf(t, rel, "actions/attest-build-provenance@", "release")
	create := indexOf(t, rel, `gh release create "${TAG}" bin/abcd-* bin/checksums.txt`, "release")
	if !(stamp < render && render < sums && sums < attest && attest < create) {
		t.Errorf("release job order: vcs-stamp %d < archive %d < checksums %d < attest %d < create %d must hold",
			stamp, render, sums, attest, create)
	}

	// The pinned address is this repository's release, asserted before the
	// Release is created: the URL derives from plugin.json's repository, which a
	// rename, transfer or fork leaves pointing elsewhere, and a pin at another
	// repository's release passes --verify yet 404s for every install.
	prefix := indexOf(t, rel, pinPrefixCheck, "release")
	if !(render < prefix && prefix < create) {
		t.Errorf("release job order: archive %d < pinned-address check %d < create %d must hold", render, prefix, create)
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

// pinPrefixCheck is the line that binds the pinned download address to the
// repository the workflow runs in.
const pinPrefixCheck = `prefix="https://github.com/${GITHUB_REPOSITORY}/releases/download/${TAG}/"`

// TestAutoReleaseProvesThePinBeforeTheTag is the pre-tag half of the pin gate.
// release.yml's verify job runs after the tag exists, so a pin the tagged
// commit cannot reproduce — a merge-queue batch that carried the ship with a
// payload-touching change re-renders to another digest — refused there only
// after the immutable tag had consumed the version. auto-release's detect job
// makes the same proof on the pushed commit before the tag job may run.
func TestAutoReleaseProvesThePinBeforeTheTag(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(AutoReleaseYMLPath)))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(data)

	detect := jobSection(t, wf, "detect")
	decide := indexOf(t, detect, "id: detect", "detect")
	setup := indexOf(t, detect, "actions/setup-go@", "detect")
	gate := indexOf(t, detect, `go run ./cmd/abcd launch archive --out "$out" --tag "${TAG}" --verify`, "detect")
	prefix := indexOf(t, detect, pinPrefixCheck, "detect")
	if !(decide < setup && setup < gate && gate < prefix) {
		t.Errorf("detect job order: decision %d < setup-go %d < archive verify %d < pinned-address check %d must hold",
			decide, setup, gate, prefix)
	}
	// Only a version about to be tagged is proved: between releases main pins
	// the last release's archive, which its moved-on tree no longer reproduces.
	for _, at := range []int{setup, gate} {
		stepStart := strings.LastIndex(detect[:at], "- name:")
		if !strings.Contains(detect[stepStart:at], "if: steps.detect.outputs.need_tag == 'true'") {
			t.Errorf("the pre-tag pin step at %d must run only when a tag is about to be made", at)
		}
	}
	if !strings.Contains(detect, "TAG: v${{ steps.detect.outputs.version }}") {
		t.Error("the pre-tag pin gate must be bound to the version the tag job will tag")
	}

	// The tag job waits on detect, so a refusal there leaves no tag behind.
	if tag := jobSection(t, wf, "tag"); !strings.Contains(tag, "needs: detect\n") {
		t.Error("the tag job must need detect, so a pre-tag refusal blocks the tag")
	}
}

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
