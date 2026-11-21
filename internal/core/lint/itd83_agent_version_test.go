package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewBarDraftNamesNoAgentVersion holds itd-83 to the prompt-version log
// (iss-2610262039184070). The draft said the reviewer agents "ship as-is at
// `0.1.0`", while 05-internals/05-prompt-quality.md records that they entered
// the changelog at `0.2.0`. The draft now says they ship as they are and names
// no version, so it cannot fall behind the log again.
func TestReviewBarDraftNamesNoAgentVersion(t *testing.T) {
	path := filepath.Join(repoRootFromPackage, ".abcd", "development", "intents", "drafts", "itd-83-review-bar-fires-itself.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("itd-83 has left drafts/; the remedy is moot once the draft moves on")
	}
	if err != nil {
		t.Fatalf("read itd-83: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "as-is at `0.1.0`") {
		t.Error("itd-83 still says the reviewer agents ship as-is at `0.1.0`; the prompt-version log records them at `0.2.0`, so the draft names no version")
	}
	if !strings.Contains(text, "They ship as they are.") {
		t.Error("itd-83 does not say the reviewer agents' prompts ship as they are")
	}
}
