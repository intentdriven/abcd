package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// outboundInstallBase is a fully answered, non-interactive install.
var outboundInstallBase = []string{"ahoy", "install", "--yes", "--adopt",
	"--visibility", "private", "--docs-target", "agents_md",
	"--oracle-backend", "host-delegated", "--scan-deep", "false"}

// TestAhoyInstallOutboundFlagIsWired — iss-2610152024327989 at the surface. The
// outbound checks reach a managed repository only through the one opt-in flag:
// a plain install turns neither on, and `--outbound` turns on both, the
// committed commit-msg hook and the shell guard's forge-text check.
func TestAhoyInstallOutboundFlagIsWired(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	hook := filepath.Join(repo, filepath.FromSlash(ahoy.OutboundHookRelPath))

	if _, err := runCLIErr(t, outboundInstallBase...); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(hook); err == nil {
		t.Fatal("a plain install wrote the outbound commit-msg hook nobody opted into")
	}
	if guard.ForgeTextOptedIn(repo) {
		t.Fatal("a plain install opted the repository into the forge-text check")
	}
	if _, err := runCLIErr(t, append(outboundInstallBase, "--outbound")...); err != nil {
		t.Fatalf("install --outbound: %v", err)
	}
	if _, err := os.Stat(hook); err != nil {
		t.Errorf("install --outbound wrote no commit-msg hook: %v", err)
	}
	if !guard.ForgeTextOptedIn(repo) {
		t.Error("install --outbound did not record outbound.forge: true")
	}
}

// TestAhoyInstallRetiredOutboundFlagsNameTheReplacement — iss-2610152024327989.
// The two flags the one switch replaced are refused, before anything is
// written, with a message naming `--outbound`: a script passing an old flag
// gets neither a silent success nor a bare unknown-flag error.
func TestAhoyInstallRetiredOutboundFlagsNameTheReplacement(t *testing.T) {
	for _, flag := range []string{"--outbound-hook", "--outbound-forge", "--outbound-hook=false", "--outbound-forge=true"} {
		t.Run(flag, func(t *testing.T) {
			hermeticEnv(t)
			repo := gittest.NewRepo(t).Root()
			t.Chdir(repo)
			_, err := runCLIErr(t, append(outboundInstallBase, flag)...)
			if err == nil {
				t.Fatalf("install %s succeeded; a retired flag must be refused", flag)
			}
			var coded interface{ ExitCode() int }
			if !errors.As(err, &coded) || coded.ExitCode() != 2 {
				t.Errorf("install %s: want a usage refusal at exit 2, got %v", flag, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, "`--outbound`") || !strings.Contains(msg, "retired") {
				t.Errorf("the refusal must say the flag was retired and name --outbound: %q", msg)
			}
			if strings.Contains(msg, "unknown flag") {
				t.Errorf("the refusal is the bare unknown-flag error: %q", msg)
			}
			if _, serr := os.Stat(filepath.Join(repo, ".abcd")); serr == nil {
				t.Errorf("install %s wrote .abcd before refusing", flag)
			}
		})
	}
}
