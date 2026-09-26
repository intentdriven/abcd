package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// error_termsafe_surface_test.go — the error surface cli.Run prints for every
// verb masks terminal-display attack runes (iss-2609012037438844). A refusal
// that echoes an operand (`embark probe <dir>` names the lifeboat it could not
// open) carried ESC, C1 and bidi runes to stderr raw, so an operand could
// recolour or rewrite the refusal the reader sees. The mask is applied once, at
// the print site, so every verb's refusal is covered without each verb
// remembering to.

const hostileOperand = "x\x1b[31mRED\u009b\u202e"

// assertNoAttackRunes (intent_render_sanitize_test.go) checks ESC, the C1
// CSI and the RLO override, the three runes hostileOperand carries.

func TestErrorSurfaceMasksAttackRunesInAnEchoedOperand(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())

	code, _, stderr := runMain(t, "embark", "probe", hostileOperand)
	if code == 0 {
		t.Fatalf("embark probe on a missing lifeboat must refuse")
	}
	if !strings.Contains(stderr, "RED") {
		t.Fatalf("the refusal no longer echoes the operand, so this test proves nothing:\n%q", stderr)
	}
	assertNoAttackRunes(t, "stderr", stderr)

	code, stdout, _ := runMain(t, "--json", "embark", "probe", hostileOperand)
	if code == 0 {
		t.Fatalf("embark probe --json on a missing lifeboat must refuse")
	}
	var env errorEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("the --json refusal is not an envelope: %v\n%s", err, stdout)
	}
	assertNoAttackRunes(t, "the --json envelope's error", env.Error)
}

// The mask keeps a multi-line refusal's own line structure: the stale-usage
// note and joined errors are lines by construction, not by injection.
func TestErrorSurfaceKeepsTheRefusalsOwnLines(t *testing.T) {
	root := stalePluginRoot(t)
	writeCommandPage(t, root, "frobnicate", "```bash\nabcd frobnicate\n```\n")
	_, _, stderr := runMain(t, "frobnicate")
	if strings.Count(stderr, "\nabcd: ") != 1 {
		t.Fatalf("the two-line refusal lost its line break:\n%q", stderr)
	}
}
