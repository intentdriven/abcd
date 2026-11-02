package ideate

import (
	"strings"
	"testing"
)

func TestExamplePassesCheck(t *testing.T) {
	if err := Check(t.TempDir(), []byte(Example)); err != nil {
		t.Fatalf("Example: %v", err)
	}
	// A grill hit naming a record the checkout lacks is refused, as Record
	// refuses it: the check resolves citations.
	cited := strings.Replace(Example, `"hits": []`, `"hits": [{"record": "itd-2609010000000001", "relation": "covered"}]`, 1)
	if err := Check(t.TempDir(), []byte(cited)); err == nil {
		t.Error("Check accepted a grill hit citing a record that does not exist")
	}
}
