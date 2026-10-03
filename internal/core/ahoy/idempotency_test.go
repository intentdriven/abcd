package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// treeHash walks root and returns a stable map of relpath -> content hash. It
// is the evidence for the "apply twice => identical tree" invariant.
func treeHash(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("treeHash: %v", err)
	}
	return out
}

func sameTree(a, b map[string]string) (string, bool) {
	if len(a) != len(b) {
		return "file count differs", false
	}
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if a[k] != b[k] {
			return "content differs: " + k, false
		}
	}
	return "", true
}

// installOpts is the fully-specified, non-interactive install used by the
// idempotency test: adopt the repo, approve every category, pin the config.
func installOpts() InstallOptions {
	adopt := true
	return InstallOptions{
		Adopt: &adopt,
		Yes:   true,
		ValueOverrides: map[string]string{
			"visibility":     "private",
			"docs_target":    "both",
			"oracle_backend": "host-delegated",
			"scan_deep":      "false",
		},
	}
}

func TestInstallThenReinstallIsExactNoOp(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := t.TempDir()
	// A bare .git dir makes this an adoptable unmanaged repo.
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// First install resolves every actionable gap.
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("first install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}

	// Evidence artefacts landed.
	for _, rel := range []string{".abcd/config.json", ".abcd/rules.json", "CLAUDE.md", "AGENTS.md", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(repo, rel)); err != nil {
			t.Errorf("expected %s after install: %v", rel, err)
		}
	}
	if _, err := os.Stat(abcdhome.Path(home, "history", "index.json")); err != nil {
		t.Errorf("history store not bootstrapped: %v", err)
	}

	repoBefore := treeHash(t, repo)
	homeBefore := treeHash(t, home)

	// Second install: detection reports zero actionable gaps => exact no-op.
	res2, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != "already_up_to_date" {
		t.Errorf("re-install status = %q, want already_up_to_date", res2.Status)
	}
	if len(res2.Writes) != 0 {
		t.Errorf("re-install wrote files: %v", res2.Writes)
	}

	if msg, ok := sameTree(repoBefore, treeHash(t, repo)); !ok {
		t.Errorf("repo tree changed on re-install: %s", msg)
	}
	if msg, ok := sameTree(homeBefore, treeHash(t, home)); !ok {
		t.Errorf("home tree changed on re-install: %s", msg)
	}
}

func TestInstallRestoresHandDeletedMarker(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// Hand-delete the marker file while setup_version stays current.
	if err := os.Remove(filepath.Join(repo, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "marker.missing") {
		t.Fatalf("state-keyed detection failed to report marker.missing: %+v", det.Gaps)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Errorf("repair status = %q, want clean", res.Status)
	}
	if classifyMarker(filepath.Join(repo, "CLAUDE.md")) != markerCurrent {
		t.Errorf("marker not restored")
	}
}

func TestUnmanagedFolderInstallAborts(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir() // no .git, no markers
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "aborted" {
		t.Errorf("status = %q, want aborted", res.Status)
	}
}

func TestUninstallInstallRoundTrip(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(repo, ""); err != nil {
		t.Fatal(err)
	}
	// After uninstall the marker block is gone but .abcd/ survives.
	if classifyMarker(filepath.Join(repo, "CLAUDE.md")) != markerMissing {
		t.Errorf("uninstall left a marker block")
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "config.json")); err != nil {
		t.Errorf("uninstall removed .abcd/ namespace: %v", err)
	}
	// Re-install converges back to a clean tree.
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Errorf("re-install status = %q, want clean", res.Status)
	}
}
