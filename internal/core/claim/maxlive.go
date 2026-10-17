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
// reservation that nothing reads, refuses the reservation naming the file,
// never replaced by the default. A config.json that cannot be read at all
// (writable by others, not valid JSON) is read past instead: such a file
// could only ever have raised the cap, so MaxLive applies, and the refusal at
// the cap says why the file was read past. The cap is read only when an act
// adds a reservation, so a bad setting never stops a renewal or a release.

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
	stack, err := layered.Load(layered.Config, layered.Roots{Home: home})
	if err != nil {
		return MaxLive, fmt.Sprintf("%s cannot be read (%s), so the default of %d applies",
			layered.Config.MachineOrigin(), redacted(err), MaxLive), nil
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
