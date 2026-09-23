package gitutil

import (
	"strings"
	"testing"
)

// TestIsFullSHA admits the two full object-name forms and nothing that could
// carry a path into a store key.
func TestIsFullSHA(t *testing.T) {
	for _, ok := range []string{strings.Repeat("a", 40), strings.Repeat("0", 64), "488a0aa96ac5de805348635b27036addf15cddc2"} {
		if !IsFullSHA(ok) {
			t.Errorf("IsFullSHA(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "488a0aa9", strings.Repeat("A", 40), strings.Repeat("a", 41), "../" + strings.Repeat("a", 37), strings.Repeat("a", 40) + "\n"} {
		if IsFullSHA(bad) {
			t.Errorf("IsFullSHA(%q) = true", bad)
		}
	}
}
