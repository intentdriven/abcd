package site

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The copy-paste one-liners, run. The product thinker's ruling of 2026-10-12
// (iss-2610120617521209) holds every install path to one rule: verify the
// download with `gh attestation verify` when gh is present, and refuse when
// it fails; without gh, install on the checksum with a warning that the
// download was not checked against the signed build record. The served
// installer is proven by installsh_run_test.go; these run the README's
// universal one-liner and the install guide's per-OS ones in the same sandbox,
// so the forms a reader pastes are held to what they do, not what they say.

// oneLinerForm is one copy-paste form and the command block it is.
type oneLinerForm struct {
	name, block string
}

func oneLinerForms(t *testing.T) []oneLinerForm {
	t.Helper()
	guide := readFile(t, filepath.Join(repoRoot(), "docs", "how-to", "install.md"))
	return []oneLinerForm{
		{"readme", onlyInstallBlock(t, "README.md", fencedBlocks(readFile(t, filepath.Join(repoRoot(), "README.md"))))},
		{"how-to-macos", onlyInstallBlock(t, "docs/how-to/install.md (### macOS)", blocksUnder(fencedBlocks(guide), "macOS"))},
		{"how-to-linux", onlyInstallBlock(t, "docs/how-to/install.md (### Linux)", blocksUnder(fencedBlocks(guide), "Linux"))},
	}
}

func runOneLiner(t *testing.T, f oneLinerForm, gh ghBehaviour) installerRun {
	t.Helper()
	return runInstallForm(t, []string{"sh", "-c", strings.TrimSpace(f.block)}, gh, nil)
}

func TestOneLinersInstallADownloadGhVerifies(t *testing.T) {
	for _, f := range oneLinerForms(t) {
		t.Run(f.name, func(t *testing.T) {
			r := runOneLiner(t, f, ghVerifies)
			if r.err != nil {
				t.Fatalf("a verified download installs: %v\n%s", r.err, r.out)
			}
			if !r.installed(t) {
				t.Fatalf("the verified binary must be installed:\n%s", r.out)
			}
			calls := r.ghCalls(t)
			if len(calls) != 1 {
				t.Fatalf("gh attestation verify must run once, got %q\n%s", calls, r.out)
			}
			sum := sha256.Sum256(r.release)
			if !strings.HasPrefix(calls[0], hex.EncodeToString(sum[:])+"\t") {
				t.Errorf("gh must verify the downloaded binary: %q", calls[0])
			}
			if !strings.Contains(calls[0], ghPolicyFlags) {
				t.Errorf("gh must be held to the release workflow's policy %q: %q", ghPolicyFlags, calls[0])
			}
			if !strings.Contains(calls[0], "--source-digest "+releaseCommit) {
				t.Errorf("gh must be bound to the latest release's commit %s: %q", releaseCommit, calls[0])
			}
			if strings.Contains(r.out, "WARNING") {
				t.Errorf("a verified install carries no warning:\n%s", r.out)
			}
		})
	}
}

func TestOneLinersRefuseADownloadGhCannotVerify(t *testing.T) {
	for _, f := range oneLinerForms(t) {
		t.Run(f.name, func(t *testing.T) {
			r := runOneLiner(t, f, ghRefuses)
			if r.err == nil {
				t.Fatalf("a failed verification must refuse:\n%s", r.out)
			}
			if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
				t.Errorf("a failed verification must install nothing: %v", err)
			}
			if _, err := os.Lstat(abcdhome.Path(r.home, "path-entry")); !os.IsNotExist(err) {
				t.Errorf("a failed verification must record nothing: %v", err)
			}
			for _, want := range []string{"build provenance", "Nothing is installed"} {
				if !strings.Contains(r.out, want) {
					t.Errorf("the refusal must say %q:\n%s", want, r.out)
				}
			}
		})
	}
}

// TestOneLinersRefuseWhenTheReleaseCommitCannotBeResolved: with a gh that
// can verify, a latest release whose commit the API does not answer cannot
// have its provenance bound to it, so nothing is installed.
func TestOneLinersRefuseWhenTheReleaseCommitCannotBeResolved(t *testing.T) {
	for _, f := range oneLinerForms(t) {
		t.Run(f.name, func(t *testing.T) {
			r := runOneLiner(t, f, ghVerifiesCommitUnresolved)
			if r.err == nil {
				t.Fatalf("an unresolvable release commit must refuse:\n%s", r.out)
			}
			if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
				t.Errorf("an unbound check must install nothing: %v", err)
			}
			if calls := r.ghCalls(t); len(calls) != 0 {
				t.Errorf("gh must not verify without the release's commit: %q", calls)
			}
			for _, want := range []string{"commit", "Nothing is installed"} {
				if !strings.Contains(r.out, want) {
					t.Errorf("the refusal must say %q:\n%s", want, r.out)
				}
			}
		})
	}
}

func TestOneLinersWarnWithoutGh(t *testing.T) {
	for gh, why := range map[ghBehaviour]string{ghAbsent: "gh-absent", ghSignedOut: "gh-signed-out", ghNoSourceDigest: "gh-without-source-digest", ghNoSourceRef: "gh-without-source-ref", ghNoCertIdentity: "gh-without-cert-identity", ghNoDenySelfHosted: "gh-without-deny-self-hosted-runners"} {
		for _, f := range oneLinerForms(t) {
			t.Run(f.name+"/"+why, func(t *testing.T) {
				r := runOneLiner(t, f, gh)
				if r.err != nil {
					t.Fatalf("abcd does not require a signed-in gh: %v\n%s", r.err, r.out)
				}
				if !r.installed(t) {
					t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
				}
				assertInstallerWarning(t, r.out, "gh")
			})
		}
	}
}
