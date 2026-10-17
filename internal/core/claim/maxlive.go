package claim

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// How many live reservations one person may hold (iss-2610170717515674) is a
// setting of the person's own machine, never of a checkout: a repository that
// could raise it could raise every contributor's ceiling. It is read from
// reservation.max_live in ~/.abcd.noindex/config.json, the machine layer
// loaded alone (layered.Roots carries no Repo), and is MaxLive when absent.
// It is still one ceiling per person, counted across all of their sessions
// and copies; a person who runs many sessions on one account raises it.
//
// A value that is not a whole number from 1 to MaxLiveCeiling, or a key under
// reservation that nothing reads, refuses an act that adds a reservation
// (reviving a lapsed one among them) naming the file, never replaced by the
// default; the cap is read by no other act, so a bad value never stops a
// release or the renewal of a live reservation. (An unknown key still stops
// every act of a person whose username is read from that file: the holder's
// reader claims the same namespace, holder.go.) A config.json that cannot be
// read at all (writable by others, not valid JSON) is read past instead, as is
// a HOME inside a git working tree that is not the account's own
// (layered.MachineRoots): neither is a file the person can be shown to have
// written, so MaxLive applies, and the refusal at the cap says why.

// MaxLiveKey is the machine configuration key.
const MaxLiveKey = holderNamespace + ".max_live"

// MaxLiveCeiling bounds the setting.
const MaxLiveCeiling = 100

// reservationKeys is every key the reservation namespace takes, claimed by
// each of its readers so neither refuses the other's key.
var reservationKeys = []string{"github_username", "max_live"}

// maxLive resolves the cap for the account whose home is home, with a note
// when the machine config.json was read past.
func maxLive(home string) (int, string, error) {
	if home == "" {
		return MaxLive, "", nil
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		return MaxLive, fmt.Sprintf("HOME lies inside a git working tree and is not this account's home, so %s is not read for %s and the default of %d applies",
			layered.Config.MachineOrigin(), MaxLiveKey, MaxLive), nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		return MaxLive, fmt.Sprintf("the machine configuration could not be read for %s (%s), so the default of %d applies",
			MaxLiveKey, redacted(err), MaxLive), nil
	}
	if err := stack.Claim(holderNamespace, reservationKeys...); err != nil {
		return 0, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	v, err := layered.Get(stack, MaxLiveKey, MaxLive, func(n int) error {
		if n < 1 || n > MaxLiveCeiling {
			return fmt.Errorf("want a whole number from 1 to %d", MaxLiveCeiling)
		}
		return nil
	})
	if err != nil {
		return 0, "", refuse("%s; %s", redacted(err), nothingPushed)
	}
	return v.V, "", nil
}
