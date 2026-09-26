package frontmatter

import "testing"

// A value that is only zero-width runes carries nothing (iss-2608301808197261).
func TestEmptinessOfReadsZeroWidthRunesAsBlank(t *testing.T) {
	for raw, want := range map[string]Emptiness{
		"​":      Blank,
		" ‌‍":    Blank,
		`"​"`:    EmptyString, // the YAML escape of the same rune
		"\"​ \"": EmptyString,
		"a​":     Populated,
	} {
		if got := EmptinessOf(raw); got != want {
			t.Errorf("EmptinessOf(%q) = %v, want %v", raw, got, want)
		}
	}
}
