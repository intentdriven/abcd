package landing

// A person's own facilitator mode (itd-2611252049031692). Two places decide
// the mode, as the product owner ruled on 2026-11-26: what the project commits
// on its default branch, which enforces auto or prevents it; and otherwise the
// person's global setting, facilitator.mode in ~/.abcd.noindex/config.json
// (the machine layer). With neither, a person is the facilitator. There is no
// setting of a person's for one project alone.
//
// The global setting is read only under the home the account database names
// for this account (AccountHome), never $HOME where the two differ: under
// another HOME, a hosted session's among them, a person has no setting, and
// the mode is human where the project commits none.

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// AccountHome is the home the account database names for this account
// (gitutil.AccountHome), where a person's global setting is read for the
// mode. A test names its own HOME.
var AccountHome = gitutil.AccountHome

// foreignHomeLayer is the layer a reading under a HOME that is not this
// account's own decides from where the project commits no mode: no layer but
// the project's is read there, so the mode is human.
const foreignHomeLayer = "home"

// homeIsAccounts says whether home is the account database's home for this
// account, after every link is resolved on both sides. A home that cannot be
// told is not.
func homeIsAccounts(home string) bool {
	if home == "" {
		return false
	}
	acct, err := AccountHome()
	if err != nil || acct == "" {
		return false
	}
	a, errA := filepath.EvalSymlinks(acct)
	h, errH := filepath.EvalSymlinks(home)
	return errA == nil && errH == nil && a == h
}

// foreignHomeOrigin says why the person's global setting is not read for the
// mode.
func foreignHomeOrigin() string {
	return "HOME is not the home the account database names for this account, so your own setting is not read for " + KeyFacilitatorMode + " here, on purpose; run abcd with your own HOME to read it"
}

// modeInput is what the resolver takes beside the layers' values: why the
// person's global setting is not read where it is not (a refused home, or,
// foreign, a HOME that is not the account's).
type modeInput struct {
	homeOff string
	foreign bool
}

// inputFor reads what the resolver takes beside vals for the reading r names.
// The machine layer's values are dropped from vals where the home is not the
// account's.
func inputFor(r layered.Roots, vals []layerValue[string]) ([]layerValue[string], modeInput) {
	var in modeInput
	switch {
	case r.HomeRefused:
		in.homeOff = homeRefusedOrigin(ModeHuman)
	case !homeIsAccounts(r.Home):
		in.homeOff, in.foreign = foreignHomeOrigin(), true
		kept := vals[:0:0]
		for _, v := range vals {
			if v.layer != layered.Machine.String() {
				kept = append(kept, v)
			}
		}
		vals = kept
	}
	return vals, in
}
