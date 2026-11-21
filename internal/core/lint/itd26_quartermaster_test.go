package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLootDraftCountsQuartermasterAgainstTheCatalog holds itd-26 to the agent
// catalog (iss-2610262031347358). The draft called loot-quartermaster the
// 16th agent and noted the brief's count of 15 becoming 16, while the scope
// chapter and the catalog in 05-internals/01-agents.md count eighteen; it also
// compared the agent with `lifeboat-oracle`, which agents/ does not ship.
func TestLootDraftCountsQuartermasterAgainstTheCatalog(t *testing.T) {
	path := filepath.Join(repoRootFromPackage, ".abcd", "development", "intents", "drafts", "itd-26-loot-oss-vendor.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("itd-26 has left drafts/; the remedy is moot once the draft moves on")
	}
	if err != nil {
		t.Fatalf("read itd-26: %v", err)
	}
	text := string(raw)
	for _, banned := range []string{"16th agent", "count of 15", "count to 16", "lifeboat-oracle"} {
		if strings.Contains(text, banned) {
			t.Errorf("itd-26 still says %q; the catalog counts eighteen agents and ships no lifeboat-oracle", banned)
		}
	}
	for _, want := range []string{"eighteen", "05-internals/01-agents.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("itd-26 does not name %q: loot-quartermaster is an addition to the catalog's eighteen", want)
		}
	}
}
