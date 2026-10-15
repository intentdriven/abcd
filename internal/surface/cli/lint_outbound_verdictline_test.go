package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// TestScaffoldedCommitMsgGateKnowsTheVerdictLine pins the line the managed
// repository's commit-msg gate (internal/core/ahoy/defaults/commit-msg) reads
// an exit 1 by. The gate counts an exit 1 as a refusal only when this line is
// in the output, so a change to the renderer's wording that the template does
// not follow would turn every refusal into a fail-open pass.
func TestScaffoldedCommitMsgGateKnowsTheVerdictLine(t *testing.T) {
	var b bytes.Buffer
	renderOutboundHuman(&b, "commit message", []scanner.Finding{{Line: 1, Column: 1, Kind: "harness:attribution_footer"}})
	first, _, _ := strings.Cut(b.String(), "\n")

	tmpl, err := os.ReadFile(filepath.Join("..", "..", "core", "ahoy", "defaults", "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `verdict="` + first + `"`; !strings.Contains(string(tmpl), want) {
		t.Fatalf("the scaffolded gate does not read the renderer's verdict line\nrenderer: %q\nwant the template to carry: %s", first, want)
	}
}
