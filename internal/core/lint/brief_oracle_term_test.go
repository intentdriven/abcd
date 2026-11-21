package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPressReleaseNamesTheHostDelegatedOracle holds the press release's
// plumbing line to the glossary's term (iss-2610262031246229). The oracle entry
// lists "LLM" among its forbidden synonyms, because the generic word elides the
// review-and-decide role, yet the brief's lead document called the oracle
// seam's default a "host-delegated LLM". It now says "host-delegated oracle"
// and cites adr-25.
func TestPressReleaseNamesTheHostDelegatedOracle(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".abcd", "development", "brief", "01-product", "01-press-release.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the press release: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "host-delegated LLM") {
		t.Errorf("the press release still calls the oracle seam's default a %q; the oracle glossary entry forbids LLM as its synonym", "host-delegated LLM")
	}
	if want := "a host-delegated oracle with opt-in oracle adapters (adr-25)"; !strings.Contains(text, want) {
		t.Errorf("the press release's plumbing line does not say %q", want)
	}
}
