package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyInstallOutboundHookFlagIsWired — iss-2609250834251447 at the surface. The
// outbound commit-msg gate reaches a managed repository only through the opt-in
// flag: a plain install writes no commit-msg hook, and `--outbound-hook` writes
// the one abcd marks as its own.
func TestAhoyInstallOutboundHookFlagIsWired(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	base := []string{"ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}
	hook := filepath.Join(repo, filepath.FromSlash(ahoy.OutboundHookRelPath))

	if _, err := runCLIErr(t, base...); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(hook); err == nil {
		t.Fatal("a plain install wrote the outbound commit-msg hook nobody opted into")
	}
	if _, err := runCLIErr(t, append(base, "--outbound-hook")...); err != nil {
		t.Fatalf("install --outbound-hook: %v", err)
	}
	if _, err := os.Stat(hook); err != nil {
		t.Errorf("install --outbound-hook wrote no commit-msg hook: %v", err)
	}
}
