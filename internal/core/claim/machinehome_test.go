package claim

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAHomeInsideAWorkingTreeNamesNoHolder: reservation.github_username in a
// HOME that lies inside a git working tree is not read as the machine's
// layer, by the rule layered.RootsFor applies, so the token names the holder;
// a malformed file there refuses nothing either.
func TestAHomeInsideAWorkingTreeNamesNoHolder(t *testing.T) {
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	for _, body := range []string{`{"reservation": {"github_username": "mallory"}}`, `{"reservation": {"github_username": 5}}`} {
		tree := gittest.NewRepo(t)
		home := filepath.Join(tree.Root(), "home")
		writeConfig(t, home, body)
		r := newCheckout(t, "pat@corp.com")
		r.Git("config", "user.name", "Pat")
		fake := &forgeFake{login: "octo-person", id: 4242}
		h, err := ResolveHolder(r.Root(), Options{Home: home, ForgeUser: fake.user})
		if err != nil || h != want {
			t.Fatalf("%s in a HOME inside a working tree: holder = %+v, %v; want %+v from the token", body, h, err, want)
		}
	}
}
