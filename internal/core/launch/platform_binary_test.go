package launch

import (
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestBundleNeverShipsAPlatformBinary pins the structural half of the no-Go
// promise (itd-154): the abcd binary reaches a user's plugin root by
// checksum-verified DOWNLOAD and by nothing else, so a released platform
// artefact must never be able to enter the payload. Today it is out by omission
// — `bin/` is gitignored and no include names it — but omission is a config an
// edit can undo, and an included binary would ship a stale, unverified abcd to
// every install. The deny is by artefact SHAPE, above the include list, and it
// REJECTS rather than quietly dropping: a maintainer who names one has made a
// mistake worth a loud failure.
func TestBundleNeverShipsAPlatformBinary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "bin/abcd-darwin-arm64", "MZ-ish")
	writeFile(t, root, "bin/abcd-linux-amd64", "MZ-ish")
	writeFile(t, root, "bin/abcd-windows-amd64.exe", "MZ-ish")
	// The bare name `go build ./cmd/abcd` produces, and the name the binary runs
	// under in the plugin root — the likeliest of the three to be committed by
	// accident, so the deny would have had its hole at its most dangerous name.
	writeFile(t, root, "bin/abcd", "MZ-ish")
	writeFile(t, root, "bin/README.md", "what lives here")
	writeFile(t, root, "docs/abcd-darwin-arm64.md", "prose about the artefact")

	b, err := ResolveBundle(root, []string{"bin", "docs"})
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"bin/abcd-darwin-arm64",
		"bin/abcd-linux-amd64",
		"bin/abcd-windows-amd64.exe",
		"bin/abcd",
	} {
		if included(b, rel) {
			t.Errorf("%s entered the payload: the binary must arrive only by checksum-verified download", rel)
		}
		if !rejectedFor(b, rel, RejectedPlatformBinary) {
			t.Errorf("%s must be rejected as a platform binary, got %+v", rel, b.Rejected)
		}
	}
	if !b.HasViolation() {
		t.Error("a payload naming a platform binary must fail the ship, not pass with a quiet exclusion")
	}
	// The deny is on the artefact shape, not on the word "abcd": ordinary files
	// that merely mention one are untouched.
	if !included(b, "bin/README.md") || !included(b, "docs/abcd-darwin-arm64.md") {
		t.Errorf("the deny over-reached beyond the released artefact names: %+v", b.Rejected)
	}
}

// TestCommittedPayloadNamesNoBinaryDirectory is the companion at the config
// layer: the committed include list must not reach the build output directory at
// all, so the structural deny above is a second line rather than the only one.
func TestCommittedPayloadNamesNoBinaryDirectory(t *testing.T) {
	root := repoRootForTest(t)
	includes, err := LoadIncludes(root)
	if err != nil {
		t.Fatalf("load committed launch-payload includes: %v", err)
	}
	for _, inc := range includes {
		if inc == "bin" || inc == "bin/**" || inc == "*" || inc == "**" {
			t.Errorf("the committed payload include %q reaches the cross-compiled binaries; they ship as release assets only", inc)
		}
	}
	// And nothing TRACKED carries a built binary's name anywhere in the tree —
	// asserted against git rather than the working directory, so a developer's
	// untracked `bin/` build output is not mistaken for a committed one.
	cmd := exec.Command("git", "-C", root, "ls-files", "-z")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable in %s: %v", root, err)
	}
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if rel == "" {
			continue
		}
		if IsBinaryName(path.Base(rel)) {
			t.Errorf("%s is committed and carries a built binary's name, so it would fail every ship", rel)
		}
	}
}

// rejectedFor reports whether the bundle rejected logical for exactly reason.
func rejectedFor(b Bundle, logical string, reason RejectedReason) bool {
	for _, f := range b.Rejected {
		if f.LogicalPath == logical {
			return f.Reason == reason
		}
	}
	return false
}
