package lint

import (
	"strings"
	"testing"
)

// TestRecordTitleReadsPastABOM: a BOM-led issue's title is its first body line,
// not its opening delimiter (iss-2608221126066379).
func TestRecordTitleReadsPastABOM(t *testing.T) {
	t.Parallel()
	lines := strings.Split("\ufeff---\nid: iss-1\n---\nThe first body line.\n", "\n")
	if got := recordTitle(lines); got != "The first body line." {
		t.Fatalf("recordTitle = %q, want the first body line", got)
	}
}

// TestAgentCapabilityScopeReadsPastABOM: a BOM-led prompt's capability scope
// is read from its block (iss-2608221126066379).
func TestAgentCapabilityScopeReadsPastABOM(t *testing.T) {
	t.Parallel()
	lines := strings.Split("\ufeff---\nname: x\ncapability_scope:\n  designed_for: [review]\n---\nbody\n", "\n")
	if got := agentCapabilityScope(lines); got["designed_for"] != "[review]" {
		t.Fatalf("agentCapabilityScope = %v, want designed_for [review]", got)
	}
}
