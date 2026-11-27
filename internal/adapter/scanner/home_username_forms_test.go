package scanner

import "testing"

// The account name is matched in both Unicode normal forms: a non-ASCII login
// typed composed reaches a transcript decomposed as often as not, outside any
// home path, and is still the caller's name.
func TestLocalUsernameMatchesBothNormalForms(t *testing.T) {
	const nfc, nfd = "café", "café"
	for _, c := range []struct{ user, text string }{{nfc, nfd}, {nfd, nfc}} {
		id := Identity{HomePath: "/srv/people/" + c.user, HomeUser: c.user}
		var found bool
		for _, f := range ScanText("signed by "+c.text+" today", id, nil, DefaultIdentitySeverities(), "f") {
			found = found || f.Kind == kindLocalUser
		}
		if !found {
			t.Errorf("account name %+q not found in the text spelling it %+q", c.user, c.text)
		}
	}
}
