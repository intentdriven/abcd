package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingRowStatesTheStarOrderMatchUses pins iss-2611040554036731: the
// naming table's `*<DOMAIN>` row said several star commands activate their
// domains in left-to-right order, while Match injects every named domain in
// name-sorted order whatever order the prompt wrote them in. The test runs
// Match on a prompt whose written order is the reverse of the sorted one, then
// holds the row to what Match did.
func TestNamingRowStatesTheStarOrderMatchUses(t *testing.T) {
	var got []string
	for _, d := range Defaults().Match("*SCRATCH *COMMITTING") {
		got = append(got, d.Name)
	}
	if len(got) != 2 || got[0] != "COMMITTING" || got[1] != "SCRATCH" {
		t.Fatalf("Match(%q) = %v, want [COMMITTING SCRATCH]: every named domain, name-sorted", "*SCRATCH *COMMITTING", got)
	}

	path := filepath.Join("..", "..", "..", ".abcd", "development", "brief", "02-constraints", "04-naming.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the naming table: %v", err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| `*<DOMAIN>` |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("%s has no `*<DOMAIN>` row", path)
	}
	if strings.Contains(row, "left-to-right") {
		t.Errorf("the `*<DOMAIN>` row still says left-to-right order, which Match does not keep:\n%s", row)
	}
	for _, want := range []string{"every named domain", "name-sorted order"} {
		if !strings.Contains(row, want) {
			t.Errorf("the `*<DOMAIN>` row does not say %q:\n%s", want, row)
		}
	}
}
