package launch_test

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/launch"
)

// TestMarkerBlockSpellingMatchesAhoy pins the marker-block gate to the pair ahoy
// actually writes. ahoy imports launch, so launch cannot read ahoy's spelling;
// the gate declares its own, and this proves ahoy recognises exactly that
// spelling as a block. A drift in either would leave the gate checking a
// marker nothing writes.
func TestMarkerBlockSpellingMatchesAhoy(t *testing.T) {
	body := []byte("# Title\n\n" + launch.MarkerBlockBegin + "\nloader\n" + launch.MarkerBlockEnd + "\n\nBody.\n")
	stripped, ok := ahoy.StripMarkerBlock(body)
	if !ok {
		t.Fatalf("ahoy does not recognise the gate's marker pair %q … %q as a block", launch.MarkerBlockBegin, launch.MarkerBlockEnd)
	}
	if string(stripped) == string(body) {
		t.Fatal("ahoy recognised the block but removed nothing")
	}
}
