package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderPayloadSecretRefuses is the gh-328 detector: the MATERIALISING path
// (PrecheckPayload / RenderPayload, the only production door that stages a
// release payload) must run the shipped secret/PII scan and FAIL CLOSED on a
// hard-fail — never write the secret to the payload dir. Before the fix the
// render never called the scanner, so a token in an included file was staged
// with err==nil (the scan lived only in DryRun, advisory, and Ship, which has no
// production caller).
func TestRenderPayloadSecretRefuses(t *testing.T) {
	root := renderFixture(t)
	// FAKE token shape only: ghp_ + 36 chars, matching \bghp_[A-Za-z0-9]{36,}.
	token := "ghp_" + strings.Repeat("a", 36)
	writeFile(t, root, "README.md", "readme\n<!-- "+token+" -->\n")
	dest := filepath.Join(t.TempDir(), "payload")

	// PrecheckPayload performs zero writes and must already refuse.
	if _, err := PrecheckPayload(root, dest, PrecheckOptions{Dirty: DirtySkip}); !errors.Is(err, ErrPayloadScanRefused) {
		t.Fatalf("PrecheckPayload must refuse on a secret in an included file, got %v", err)
	}

	// RenderPayload must refuse rather than materialise the secret.
	res, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: dest, Version: "0.4.0", Entry: sampleEntry(), Dirty: DirtySkip,
	})
	if !errors.Is(err, ErrPayloadScanRefused) {
		t.Fatalf("RenderPayload must refuse on a payload secret with ErrPayloadScanRefused, got err=%v res=%+v", err, res)
	}

	// The secret must NEVER reach the payload dir. Whether the dir was created or
	// not, the staged README must not exist / must not carry the token.
	if data, rerr := os.ReadFile(filepath.Join(dest, "README.md")); rerr == nil {
		if strings.Contains(string(data), token) {
			t.Fatalf("render staged the secret into the payload dir — the token was materialised unscanned")
		}
	}
}

// TestRenderPayloadCleanTreeStillRenders proves the scan gate does not block a
// clean payload: the fixture with no secret must still render and pass lockstep.
func TestRenderPayloadCleanTreeStillRenders(t *testing.T) {
	root := renderFixture(t)
	dest := filepath.Join(t.TempDir(), "payload")

	res, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: dest, Version: "0.4.0", Entry: sampleEntry(), Dirty: DirtySkip,
	})
	if err != nil {
		t.Fatalf("render must still succeed on a clean tree after the scan gate: %v", err)
	}
	if !res.Lockstep.OK {
		t.Fatalf("clean render must pass the public lockstep check, got %+v", res.Lockstep)
	}
	if _, serr := os.Stat(filepath.Join(dest, "README.md")); serr != nil {
		t.Fatalf("clean payload must carry the README include: %v", serr)
	}
}
