package launch

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestCommittedTreeSatisfiesDevPolarity is the adr-19 detector for THIS
// repository, not a synthetic fixture: the committed manifests must carry no
// version key, and the committed version-location contract must be readable
// enough to say where that key would be.
//
// It is pinned here rather than left to the synthetic lockstep tests because the
// two halves fail in opposite directions and only the real tree proves both at
// once: an unreadable contract (no version-location.json) makes the whole gate
// inert, and a version key added to a working-tree manifest silently breaks the
// premise that the version is an output of the release cut.
//
// Its third half is the amendment the 2026-09-23 ruling E2 made to adr-19/20
// (adr-2609231048308186): the committed catalog is not version-free by pointing
// into the unversioned tree any more. It sources the plugin from the LATEST
// release's pinned archive — that release's download URL and digest — so what an
// install at the tip of main receives is a cut release, fingerprinted. The one
// exception is the bootstrap: every release up to and including
// archiveBootstrapRelease was published before archives existed, so while the
// newest dated CHANGELOG heading is one of those, the relative-path source is
// still what the tree carries. The first ship past it writes the pin, and from
// then on a catalog that does not name the newest release fails here.
func TestCommittedTreeSatisfiesDevPolarity(t *testing.T) {
	root := repoRootForTest(t)
	res := CheckLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath))
	if res.Unreadable {
		t.Fatalf("the committed version-location contract must be readable, got %s", res.Detail)
	}
	if !res.OK || res.ExitCode != 0 {
		t.Fatalf("adr-19: the committed tree must be version-ABSENT, got drifts %v", res.Drifts)
	}

	latest := newestDatedRelease(t, root)
	pin, pinned, err := ReadArchivePin(root)
	if err != nil {
		t.Fatalf("read the committed catalog's plugin source: %v", err)
	}
	if !pinned {
		if !releaseAtOrBefore(t, latest, archiveBootstrapRelease) {
			t.Fatalf("the newest release %s is past the archive bootstrap (%s), so %s must source the plugin from its pinned archive",
				latest, archiveBootstrapRelease, marketplaceFile)
		}
		return
	}
	want, err := ArchiveReleaseURL(root, latest)
	if err != nil {
		t.Fatalf("derive the newest release's archive address: %v", err)
	}
	if pin.URL != want {
		t.Errorf("the catalog pins %s, but the newest release is published at %s", pin.URL, want)
	}
	if err := validatePin(pin); err != nil {
		t.Errorf("the committed pin is malformed: %v", err)
	}
}

// archiveBootstrapRelease is the last release published without a plugin
// archive. It is a fixed historical fact, never bumped.
const archiveBootstrapRelease = "0.9.0"

// newestDatedRelease reads the first dated CHANGELOG heading, the release
// auto-release.yml tags. (The changelog package owns the canonical reader but
// imports this one, so the test reads the heading itself.)
func newestDatedRelease(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	m := regexp.MustCompile(`(?m)^## \[v?([0-9]+\.[0-9]+\.[0-9]+)\] - `).FindSubmatch(data)
	if m == nil {
		t.Fatal("CHANGELOG.md carries no dated release heading")
	}
	return string(m[1])
}

// releaseAtOrBefore reports whether version a is no later than b.
func releaseAtOrBefore(t *testing.T, a, b string) bool {
	t.Helper()
	va, err := ParseSemver(a)
	if err != nil {
		t.Fatal(err)
	}
	vb, err := ParseSemver(b)
	if err != nil {
		t.Fatal(err)
	}
	if va.Major != vb.Major {
		return va.Major < vb.Major
	}
	if va.Minor != vb.Minor {
		return va.Minor < vb.Minor
	}
	return va.Patch <= vb.Patch
}

// TestCommittedContractSelectsPluginManifest pins the version location the
// render writes to. A silent relocation would leave the render stamping a
// version somewhere the harness never reads, and the lockstep check would still
// pass because it reads the same moved pointer.
func TestCommittedContractSelectsPluginManifest(t *testing.T) {
	root := repoRootForTest(t)
	decision, err := loadJSON(filepath.Join(root, versionLocationRelPath))
	if err != nil {
		t.Fatalf("read the committed version-location contract: %v", err)
	}
	path, ptr, verr := validateVersionLocation(decision)
	if verr != "" {
		t.Fatalf("the committed contract must validate, got %s", verr)
	}
	if path != ".claude-plugin/plugin.json" || ptr != "/version" {
		t.Errorf("expected adr-19's ACCEPT outcome (.claude-plugin/plugin.json /version), got %q %q", path, ptr)
	}
}
