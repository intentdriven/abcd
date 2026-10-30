package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestATakeOverBackOffSaysWhetherADeclaredIDsPushIsItsPairedAccounts: a live
// reservation under an id octo declares backs off octo's reservation without
// --take-over, and the back-off says whether the id is paired and whether
// its paired account pushed it, naming no login; when nothing ties it to an
// account of octo's, it ends by saying what a take-over would do and why it
// is allowed, rather than inviting one.
func TestATakeOverBackOffSaysWhetherADeclaredIDsPushIsItsPairedAccounts(t *testing.T) {
	for _, c := range []struct {
		name, declared, pusher, want string
		doubtful                     bool
	}{
		{"paired, its account", mayaB + " maya\n", "maya", "the forge's record of pushes says the account it is paired with pushed this reservation", false},
		{"paired, another account", mayaB + " maya\n", "mal", "the forge's record of pushes says an account other than the one it is paired with pushed this reservation, so it is not your other account's", true},
		{"unpaired", mayaB + "\n", "maya", "with no login beside it, so which account pushed this reservation is not checked", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			f := armedForge(t)
			m := message{Record: "iss-5", Holder: mayaB, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", Footprint: []string{"a.go"},
				ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour)}
			f.pusherOf[w.pushTipAt("iss-5", m.text(), w.clock())] = c.pusher
			octo := w.clone("1234+octo@users.noreply.github.com", "")
			declareHolderIDs(t, octo.home, c.declared)
			_, err := octo.reserve("iss-5", "a.go")
			if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "you are already building iss-5 on another computer") ||
				!strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "maya") || strings.Contains(err.Error(), "mal") {
				t.Fatalf("err = %v; want the other-copy back-off saying %q and naming no login", err, c.want)
			}
			doubtful := strings.Contains(err.Error(), "--take-over is still allowed here, only because "+DeclaredHolderIDsFile+" lists "+mayaB+" as yours: it replaces that reservation")
			if doubtful != c.doubtful || doubtful == strings.Contains(err.Error(), "pass --take-over to continue here") {
				t.Fatalf("err = %v; want the tail saying what a take-over does %v, and the plain invitation otherwise", err, c.doubtful)
			}
		})
	}
}
