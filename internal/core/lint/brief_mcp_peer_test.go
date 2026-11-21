package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUniversalPatternsMarkTheMCPPeerContractAsADesignTarget holds the
// universal patterns' peer-interop section to the tree (iss-2610262038376261).
// Its MCP bullet said that, with both present, abcd and the companion harness
// interoperate as two independent MCP servers, while the build sequence marks
// abcd's MCP front door a design target and internal/surface/ holds no mcp
// door. Until that door exists, the bullet marks abcd's MCP server a design
// target and says today abcd and a peer interoperate over shared on-disk
// conventions only.
func TestUniversalPatternsMarkTheMCPPeerContractAsADesignTarget(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "internal", "surface", "mcp")); err == nil {
		t.Skip("internal/surface/mcp exists: abcd has an MCP front door, so the bullet may describe it as built")
	}
	const rel = ".abcd/development/brief/05-internals/04-universal-patterns.md"
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	text := string(raw)
	const sectionOpener = "## 3. "
	start := strings.Index(text, sectionOpener)
	if start < 0 {
		t.Fatalf("%s has no section 3", rel)
	}
	section := text[start:]
	if end := strings.Index(section[len(sectionOpener):], "\n## "); end >= 0 {
		section = section[:len(sectionOpener)+end]
	}
	const bulletOpener = "- **MCP**"
	b := strings.Index(section, bulletOpener)
	if b < 0 {
		t.Fatalf("%s section 3 has no bullet opening %q", rel, bulletOpener)
	}
	bullet := section[b:]
	if end := strings.Index(bullet[len(bulletOpener):], "\n- "); end >= 0 {
		bullet = bullet[:len(bulletOpener)+end]
	}
	flat := strings.Join(strings.Fields(bullet), " ")

	if !strings.Contains(flat, "*(design target)*") {
		t.Errorf("%s section 3's MCP bullet does not mark abcd's MCP server a design target:\n%s", rel, flat)
	}
	if want := "today abcd and a peer interoperate over shared on-disk conventions only"; !strings.Contains(strings.ToLower(flat), want) {
		t.Errorf("%s section 3's MCP bullet does not say %q:\n%s", rel, want, flat)
	}
	if strings.Contains(flat, "abcd and the companion harness interoperate as two independent MCP servers") {
		t.Errorf("%s section 3's MCP bullet still states as fact that abcd interoperates as an MCP server, which it does not have:\n%s", rel, flat)
	}
}
