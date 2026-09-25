package scanner

import (
	"strings"
	"testing"
)

// persona_host_suggestion_test.go — iss-2609190338409222: net_lan_hostname
// flagged a possessive persona host ("alices-mac.local") while its suggestion
// offered "a persona-derived fixture host", so the author saw a persona host
// refused and could not tell why. The skip stays as it is — the possessive,
// capitalised, is how macOS names a real person's machine — and the suggestion
// names the exact shape it accepts.
func TestHostnameSuggestionNamesTheAcceptedShape(t *testing.T) {
	id := Identity{}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	if f := ScanText("ping alice-mac.local", id, pats, sev, "f"); hasKind(f, kindNetLANHost) {
		t.Fatalf("the accepted persona shape was flagged: %+v", f)
	}
	for _, line := range []string{"ping alices-mac.local", "ssh alices-macbook"} {
		f := ScanText(line, id, pats, sev, "f")
		if len(f) == 0 {
			t.Fatalf("the possessive host in %q was not flagged", line)
		}
		for _, x := range f {
			if !strings.Contains(x.Suggested, "<persona>-") || !strings.Contains(x.Suggested, "no possessive") {
				t.Errorf("%s suggestion does not name the accepted shape: %q", x.Kind, x.Suggested)
			}
		}
	}
}
