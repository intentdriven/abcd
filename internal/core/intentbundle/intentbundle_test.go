package intentbundle

import "testing"

// The phrase is the contract between the writer and the gate, so its exact
// words are pinned here: a rewording is a deliberate change to both at once.
func TestOneMemberNamesTheBundle(t *testing.T) {
	if got, want := OneMember("alpha-beta"), "bundle alpha-beta now has one member"; got != want {
		t.Fatalf("OneMember = %q, want %q", got, want)
	}
}
