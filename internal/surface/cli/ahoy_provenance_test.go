package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/update"
)

// iss-2610120617515834: `ahoy install` promotes the plugin's cached release
// binary to the PATH copy only once its build provenance verifies, through
// abcd update's own in-process check. These tests hold the CLI's wiring to
// that one primitive and run it against the recorded real bundles the update
// tests use (internal/core/update/testdata/provenance).

// provenanceFixture reads one of the update package's recorded fixtures.
func provenanceFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "core", "update", "testdata", "provenance", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// recordedProvenance swaps the CLI's check for the shipping one pointed at a
// served attestations API that answers every digest with body (404 when nil)
// and at the recorded Sigstore trusted root.
func recordedProvenance(t *testing.T, body []byte) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	check, err := update.ReleaseProvenanceForTest(srv.URL+"/attestations/", provenanceFixture(t, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}
	saved := ahoyReleaseProvenance
	t.Cleanup(func() { ahoyReleaseProvenance = saved })
	ahoyReleaseProvenance = check
}

func TestAhoyInstallIsWiredToTheUpdatersProvenanceCheck(t *testing.T) {
	if shippedAhoyReleaseProvenance == nil ||
		reflect.ValueOf(shippedAhoyReleaseProvenance).Pointer() != reflect.ValueOf(update.VerifyReleaseProvenance).Pointer() {
		t.Fatal("ahoy install must hand core abcd update's own provenance check, update.VerifyReleaseProvenance")
	}
}

// TestAhoyInstallRefusesACacheTheReleaseWorkflowDidNotBuild: a cache the
// bootstrap bound (checksum and attestation agree) whose bytes no provenance
// signed by release.yml names is not promoted, whatever the API answers: the
// release's real bundle for other bytes, another workflow's, another
// repository's, or none at all.
func TestAhoyInstallRefusesACacheTheReleaseWorkflowDidNotBuild(t *testing.T) {
	for _, tc := range []struct {
		name string
		body []byte
		want string // what the refusal must name as the failure
	}{
		{"the release's bundle for other bytes", provenanceFixture(t, "abcd-v0.13.4-linux-amd64.attestations.json"), "verifies as signed by intentdriven/abcd/.github/workflows/release.yml"},
		{"another workflow", provenanceFixture(t, "abcd-v0.13.4-site.attestations.json"), "verifies as signed by intentdriven/abcd/.github/workflows/release.yml"},
		{"another repository", provenanceFixture(t, "cli-v2.102.0.attestations.json"), "verifies as signed by intentdriven/abcd/.github/workflows/release.yml"},
		{"no attestation", nil, "no build-provenance attestation is published"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binDir, repo := provenanceInstallEnv(t)
			recordedProvenance(t, tc.body) // reads its fixture before the chdir
			t.Chdir(repo)

			out, _ := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
				"--visibility", "private", "--docs-target", "agents_md",
				"--oracle-backend", "host-delegated", "--scan-deep", "false")
			if _, err := os.Lstat(filepath.Join(binDir, "abcd")); !os.IsNotExist(err) {
				t.Fatalf("a cache without release.yml's provenance must leave no PATH entry: %v\n%s", err, out)
			}
			flat := strings.Join(strings.Fields(string(out)), " ")
			for _, want := range []string{"build provenance could not be verified", tc.want} {
				if !strings.Contains(flat, want) {
					t.Errorf("the refusal must say %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestAhoyInstallPromotesACacheWhoseProvenanceVerifies: the check's verdict
// reaches core: one that verifies lets the bound cache become the PATH copy.
// The recorded bundles name real release binaries this test cannot hold, so
// the verifying check is the suite's stand-in.
func TestAhoyInstallPromotesACacheWhoseProvenanceVerifies(t *testing.T) {
	binDir, repo := provenanceInstallEnv(t)
	t.Chdir(repo)
	out := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false")
	if fi, err := os.Lstat(filepath.Join(binDir, "abcd")); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("a cache whose provenance verifies must become the owned PATH copy: %v\n%s", err, out)
	}
}

// provenanceInstallEnv is a sandboxed home whose plugin data dir holds a cache
// the bootstrap bound, and an adoptable repository.
func provenanceInstallEnv(t *testing.T) (binDir, repo string) {
	t.Helper()
	home, _ := userScopeEnv(t)
	binDir = filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	repo = t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, repo
}
