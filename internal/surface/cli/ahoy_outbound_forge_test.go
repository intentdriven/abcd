package cli

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyInstallOutboundForgeFlagIsWired — iss-2610142210511831 at the
// surface. The forge-text check reaches a managed repository only through the
// opt-in flag: a plain install records nothing, and `--outbound-forge` records
// the key the shell guard reads.
func TestAhoyInstallOutboundForgeFlagIsWired(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	base := []string{"ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}

	if _, err := runCLIErr(t, base...); err != nil {
		t.Fatalf("install: %v", err)
	}
	if guard.ForgeTextOptedIn(repo) {
		t.Fatal("a plain install opted the repository into the forge-text check")
	}
	if _, err := runCLIErr(t, append(base, "--outbound-forge")...); err != nil {
		t.Fatalf("install --outbound-forge: %v", err)
	}
	if !guard.ForgeTextOptedIn(repo) {
		t.Error("install --outbound-forge did not record outbound.forge: true")
	}
}
